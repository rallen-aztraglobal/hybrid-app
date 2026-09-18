// Package service — 推送活动业务逻辑（ADR-0012）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// fcmWorkerCount worker pool 并发数（发送时逐 token 并发）。
const fcmWorkerCount = 20

// ---------- 类型定义（与 API 契约对齐）----------

// PushStatusResult GET /api/push/status 返回。
type PushStatusResult struct {
	Enabled bool            `json:"enabled"`
	Brands  map[string]bool `json:"brands"`
}

// PushCampaignInput 创建/编辑活动的请求体（对应 API 契约 PushCampaignInput）。
type PushCampaignInput struct {
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	ImageURL     string            `json:"imageUrl"`
	DeeplinkPath string            `json:"deeplinkPath"`
	ExtraData    map[string]string `json:"extraData"`
	TargetAppIDs []string          `json:"targetAppIds"`
}

// PushCampaignView 是对前端展示友好的活动响应（对应 API 契约 PushCampaign）。
type PushCampaignView struct {
	ID           uint64            `json:"id"`
	Kind         string            `json:"kind"` // channel / listing
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	ImageURL     string            `json:"imageUrl"`
	DeeplinkPath string            `json:"deeplinkPath"`
	ExtraData    map[string]string `json:"extraData,omitempty"`
	TargetAppIDs []string          `json:"targetAppIds"`
	ListingIDs   []uint64          `json:"listingIds,omitempty"` // kind=listing 时的目标上架包
	Status       string            `json:"status"`
	ScheduledAt  *time.Time        `json:"scheduledAt,omitempty"`
	SentAt       *time.Time        `json:"sentAt,omitempty"`
	TotalDevices int               `json:"totalDevices"`
	SuccessCount int               `json:"successCount"`
	FailureCount int               `json:"failureCount"`
	CreatedBy    string            `json:"createdBy"`
	CreatedAt    time.Time         `json:"createdAt"`

	// ---- 周期定时推送 ----
	RepeatEveryDays int        `json:"repeatEveryDays"`
	RepeatEndAt     *time.Time `json:"repeatEndAt,omitempty"`
	RepeatMaxRuns   int        `json:"repeatMaxRuns"`
	RunCount        int        `json:"runCount"`
	LastRunAt       *time.Time `json:"lastRunAt,omitempty"`
	ParentID        *uint64    `json:"parentId,omitempty"`
}

// ScheduleCampaignInput 是定时发送的请求参数（单次/周期，channel 与 listing 共用）。
type ScheduleCampaignInput struct {
	ScheduledAt     time.Time
	RepeatEveryDays int        // 0=单次；1=每天；N=每 N 天（1..365）
	RepeatEndAt     *time.Time // 仅周期任务有意义；单次任务传入会被忽略清零
	RepeatMaxRuns   int        // 仅周期任务有意义；单次任务传入会被忽略清零
}

// PushRecordView 对应 API 契约 PushRecord。
type PushRecordView struct {
	ApplicationID string     `json:"applicationId"`
	Sent          int        `json:"sent"`
	Failed        int        `json:"failed"`
	ErrorSample   string     `json:"errorSample,omitempty"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
}

// PushCampaignDetail 详情：活动 + records。
type PushCampaignDetail struct {
	PushCampaignView
	Records []PushRecordView `json:"records"`
}

// AudienceResult GET /api/push/audience 返回。
type AudienceResult struct {
	TotalDevices int64            `json:"totalDevices"`
	ByApp        map[string]int64 `json:"byApp"`
}

// PushSendResult 发送接口返回（真发时 DryRun=false，Preview 为 nil）。
type PushSendResult struct {
	Campaign PushCampaignView `json:"campaign"`
	DryRun   bool             `json:"dryRun"`
	// Preview 仅 dryRun=true 时填充：预览各 appId 的预计触达数，不持久化。
	Preview *DryRunPreview `json:"preview,omitempty"`
}

// DryRunPreview dry-run 预览数据（只存在于响应，绝不写回 campaign 持久字段）。
type DryRunPreview struct {
	TotalDevices int            `json:"totalDevices"`
	ByApp        map[string]int `json:"byApp"` // applicationId → 活跃 token 数
}

// ---------- Service 方法 ----------

// PushStatus 返回 FCM 配置状态（供前端展示提示条 / 发送按钮状态）。
func (s *Service) PushStatus() PushStatusResult {
	return PushStatusResult{
		Enabled: s.cfg.PushEnabled,
		Brands:  s.fcm.ConfiguredBrands(),
	}
}

// RegisterDeviceToken APK 上报 token（公开端点）。
// 校验 applicationId 对应渠道存在（轻量防滥用），然后 upsert。
func (s *Service) RegisterDeviceToken(ctx context.Context, appID, token, palCode, platform, modelInfo string) error {
	if appID == "" || token == "" {
		return errBadRequest("appId 与 token 不得为空")
	}
	// 校验 applicationId 对应渠道存在（ADR-0009 防滥用）。
	ch, err := s.repo.GetChannelByApplicationID(ctx, appID)
	if err != nil {
		return errBadRequest("applicationId 对应渠道不存在")
	}
	brandCode := ""
	if ch.Brand != nil {
		brandCode = ch.Brand.Code
	}

	dt := &model.PushDeviceToken{
		ApplicationID: appID,
		BrandCode:     brandCode,
		DeviceToken:   token,
		PalCode:       palCode,
		Platform:      platform,
		ModelInfo:     modelInfo,
	}
	if err := s.repo.UpsertDeviceToken(ctx, dt); err != nil {
		return fmt.Errorf("注册 token 失败: %w", err)
	}
	return nil
}

// CreateCampaign 创建推送活动草稿。scope 是调用者的数据范围（数据权限强制点：活动的 brand
// 在范围内才能建，有一个目标越界即整体拒绝，见 docs/admin/10-rbac.md）。
func (s *Service) CreateCampaign(ctx context.Context, scope auth.Scope, in PushCampaignInput, createdBy string) (*PushCampaignView, error) {
	if err := validateCampaignInput(in); err != nil {
		return nil, err
	}
	if err := s.assertAppIDsInScope(ctx, scope, in.TargetAppIDs); err != nil {
		return nil, err
	}
	c := &model.PushCampaign{
		Kind:         model.CampaignKindChannel, // 渠道推送；上架包推送走 CreateListingCampaign
		Name:         in.Name,
		Title:        in.Title,
		Body:         in.Body,
		ImageURL:     in.ImageURL,
		DeeplinkPath: in.DeeplinkPath,
		ExtraData:    marshalExtraData(in.ExtraData),
		Status:       model.CampaignDraft,
		CreatedBy:    createdBy,
	}
	if err := s.repo.CreateCampaign(ctx, c); err != nil {
		return nil, err
	}
	// 写入 targets。
	if err := s.repo.ReplaceCampaignTargets(ctx, c.ID, in.TargetAppIDs); err != nil {
		return nil, err
	}
	return s.campaignView(ctx, c, in.TargetAppIDs), nil
}

// ListCampaigns 查询推送活动列表，按调用者数据范围过滤（见 docs/admin/10-rbac.md）。
func (s *Service) ListCampaigns(ctx context.Context, brand string, scope auth.Scope) ([]PushCampaignView, error) {
	// 只列渠道推送；上架包推送走 ListListingCampaigns，避免两类混在一起。
	f := repo.CampaignFilter{Brand: brand, Kind: model.CampaignKindChannel, Limit: 100}
	applyCampaignScope(&f, scope)
	list, err := s.listWithActiveSchedules(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]PushCampaignView, 0, len(list))
	for i := range list {
		appIDs := extractTargetAppIDs(list[i].Targets)
		out = append(out, *s.campaignView(ctx, &list[i], appIDs))
	}
	return out, nil
}

// ListListingCampaigns 列出上架包推送活动（kind=listing），供 Console 历史展示。
func (s *Service) ListListingCampaigns(ctx context.Context) ([]PushCampaignView, error) {
	list, err := s.listWithActiveSchedules(ctx, repo.CampaignFilter{Kind: model.CampaignKindListing, Limit: 100})
	if err != nil {
		return nil, err
	}
	out := make([]PushCampaignView, 0, len(list))
	for i := range list {
		out = append(out, *s.campaignView(ctx, &list[i], nil))
	}
	return out, nil
}

// listWithActiveSchedules 取最近 f.Limit 条活动，并**额外并入所有仍在等待触发的定时任务**
// （scheduled/paused，同样受品牌/数据范围过滤）。周期任务每跑一次就派生一条子活动，只取最近 N 条
// 的话，跑久了父任务会被子活动挤出列表——Console「定时任务」区看不到它、也就无法暂停/停止，
// 而它仍在每天照发。结果按 id 倒序、去重。
func (s *Service) listWithActiveSchedules(ctx context.Context, f repo.CampaignFilter) ([]model.PushCampaign, error) {
	recent, err := s.repo.ListCampaigns(ctx, f)
	if err != nil {
		return nil, err
	}
	af := f
	af.Statuses = []string{model.CampaignScheduled, model.CampaignPaused}
	af.Limit = 200 // repo 上限；同时在跑的定时任务远小于此
	active, err := s.repo.ListCampaigns(ctx, af)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint64]bool, len(recent))
	for _, c := range recent {
		seen[c.ID] = true
	}
	merged := recent
	for _, c := range active {
		if !seen[c.ID] {
			merged = append(merged, c)
		}
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].ID > merged[j].ID })
	return merged, nil
}

// GetCampaign 取活动详情（含 records）。
func (s *Service) GetCampaign(ctx context.Context, id uint64) (*PushCampaignDetail, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	appIDs := extractTargetAppIDs(c.Targets)
	v := s.campaignView(ctx, c, appIDs)
	recs := make([]PushRecordView, 0, len(c.Records))
	for _, r := range c.Records {
		recs = append(recs, PushRecordView{
			ApplicationID: r.ApplicationID,
			Sent:          r.Sent,
			Failed:        r.Failed,
			ErrorSample:   r.ErrorSample,
			FinishedAt:    r.FinishedAt,
		})
	}
	return &PushCampaignDetail{PushCampaignView: *v, Records: recs}, nil
}

// UpdateCampaign 修改草稿活动（仅 draft 可改）。scope 见 CreateCampaign（"改" 同样要求
// brand 在范围内）。
func (s *Service) UpdateCampaign(ctx context.Context, scope auth.Scope, id uint64, in PushCampaignInput) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status != model.CampaignDraft {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 draft 可编辑", c.Status))
	}
	if err := validateCampaignInput(in); err != nil {
		return nil, err
	}
	if err := s.assertAppIDsInScope(ctx, scope, in.TargetAppIDs); err != nil {
		return nil, err
	}
	c.Name = in.Name
	c.Title = in.Title
	c.Body = in.Body
	c.ImageURL = in.ImageURL
	c.DeeplinkPath = in.DeeplinkPath
	c.ExtraData = marshalExtraData(in.ExtraData)

	if err := s.repo.UpdateCampaign(ctx, c, in.TargetAppIDs); err != nil {
		return nil, err
	}
	return s.campaignView(ctx, c, in.TargetAppIDs), nil
}

// ScheduleCampaign 设置渠道推送活动的定时发送（draft → scheduled），支持单次/周期。
// scope 是调用者的数据范围（数据权限强制点，同 CreateCampaign）。
func (s *Service) ScheduleCampaign(ctx context.Context, scope auth.Scope, id uint64, in ScheduleCampaignInput) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Kind != model.CampaignKindChannel {
		return nil, errBadRequest("该端点仅用于渠道推送活动，上架包推送请用 /api/push/listing-campaigns/:id/schedule")
	}
	if c.Status != model.CampaignDraft {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 draft 可设置定时", c.Status))
	}
	appIDs := extractTargetAppIDs(c.Targets)
	if err := s.assertAppIDsInScope(ctx, scope, appIDs); err != nil {
		return nil, err
	}
	norm, err := normalizeScheduleInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.casCampaignStatus(ctx, id, model.CampaignDraft, scheduleFields(norm)); err != nil {
		return nil, err
	}
	applyScheduleToCampaign(c, norm)
	return s.campaignView(ctx, c, appIDs), nil
}

// ScheduleListingCampaign 设置上架包推送活动的定时发送（draft → scheduled），支持单次/周期。
// 上架包推送不受渠道数据权限约束（与 ListListingCampaigns/SendListingCampaign 一致）。
func (s *Service) ScheduleListingCampaign(ctx context.Context, id uint64, in ScheduleCampaignInput) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Kind != model.CampaignKindListing {
		return nil, errBadRequest("该端点仅用于上架包推送活动")
	}
	if c.Status != model.CampaignDraft {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 draft 可设置定时", c.Status))
	}
	norm, err := normalizeScheduleInput(in)
	if err != nil {
		return nil, err
	}
	if err := s.casCampaignStatus(ctx, id, model.CampaignDraft, scheduleFields(norm)); err != nil {
		return nil, err
	}
	applyScheduleToCampaign(c, norm)
	return s.campaignView(ctx, c, nil), nil
}

// normalizeScheduleInput 校验并规范化定时参数：scheduledAt 必须晚于当前；repeatEveryDays∈[0,365]；
// repeatMaxRuns∈[0,1000]；repeatEveryDays=0（单次）时 end/maxRuns 一律清零（前端传了也忽略）；
// repeatEveryDays>0 时若给了 repeatEndAt 必须不早于 scheduledAt。
func normalizeScheduleInput(in ScheduleCampaignInput) (ScheduleCampaignInput, error) {
	if in.ScheduledAt.IsZero() {
		return in, errBadRequest("scheduledAt 不得为空")
	}
	if !in.ScheduledAt.After(time.Now()) {
		return in, errBadRequest("scheduledAt 不得早于/等于当前时间")
	}
	if in.RepeatEveryDays < 0 || in.RepeatEveryDays > 365 {
		return in, errBadRequest("repeatEveryDays 取值范围为 0~365")
	}
	if in.RepeatMaxRuns < 0 || in.RepeatMaxRuns > 1000 {
		return in, errBadRequest("repeatMaxRuns 取值范围为 0~1000")
	}
	if in.RepeatEveryDays == 0 {
		in.RepeatEndAt = nil
		in.RepeatMaxRuns = 0
		return in, nil
	}
	if in.RepeatEndAt != nil && in.RepeatEndAt.Before(in.ScheduledAt) {
		return in, errBadRequest("repeatEndAt 不得早于 scheduledAt")
	}
	return in, nil
}

// scheduleFields 组装 ScheduleCampaign 的局部更新字段。
func scheduleFields(in ScheduleCampaignInput) map[string]any {
	return map[string]any{
		"status":            model.CampaignScheduled,
		"scheduled_at":      in.ScheduledAt.UTC(), // 统一 UTC 存储，见 repo.ListScheduledCampaigns
		"repeat_every_days": in.RepeatEveryDays,
		"repeat_end_at":     in.RepeatEndAt,
		"repeat_max_runs":   in.RepeatMaxRuns,
		"run_count":         0,
	}
}

// applyScheduleToCampaign 把规范化后的定时参数同步写回内存中的 c（供本次响应直接用，不必重查库）。
func applyScheduleToCampaign(c *model.PushCampaign, in ScheduleCampaignInput) {
	scheduledAt := in.ScheduledAt
	c.Status = model.CampaignScheduled
	c.ScheduledAt = &scheduledAt
	c.RepeatEveryDays = in.RepeatEveryDays
	c.RepeatEndAt = in.RepeatEndAt
	c.RepeatMaxRuns = in.RepeatMaxRuns
	c.RunCount = 0
}

// checkCampaignScope 仅对 kind=channel 的活动做数据权限校验（listing 活动的数据权限口径与
// ListListingCampaigns/SendListingCampaign 一致——不受渠道 scope 约束）。
func (s *Service) checkCampaignScope(ctx context.Context, scope auth.Scope, c *model.PushCampaign) error {
	if c.Kind != model.CampaignKindChannel {
		return nil
	}
	return s.assertAppIDsInScope(ctx, scope, extractTargetAppIDs(c.Targets))
}

// PauseCampaign 暂停一个周期任务（scheduled → paused）。单次任务（RepeatEveryDays=0）不支持
// 暂停，只能取消——暂停语义是「先按下这条周期链，之后还能恢复」，对单次任务没有意义。
// 对 channel 与 listing 两种 kind 都生效。
func (s *Service) PauseCampaign(ctx context.Context, scope auth.Scope, id uint64) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkCampaignScope(ctx, scope, c); err != nil {
		return nil, err
	}
	if c.Status != model.CampaignScheduled {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 scheduled 可暂停", c.Status))
	}
	if c.RepeatEveryDays <= 0 {
		return nil, errBadRequest("单次任务不支持暂停，如需停止请直接取消")
	}
	if err := s.casCampaignStatus(ctx, id, model.CampaignScheduled, map[string]any{"status": model.CampaignPaused}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignPaused
	return s.campaignView(ctx, c, extractTargetAppIDs(c.Targets)), nil
}

// ResumeCampaign 恢复一个已暂停的周期任务（paused → scheduled）。若原定的下次运行时间已过去，
// 按 repeatEveryDays 步进到第一个晚于当前时间的时间点（宕机/暂停期间漏掉的不补发）；
// 若步进后已超出 repeatEndAt 或 runCount 已达 repeatMaxRuns，说明这条周期链已经跑完，拒绝恢复。
func (s *Service) ResumeCampaign(ctx context.Context, scope auth.Scope, id uint64) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkCampaignScope(ctx, scope, c); err != nil {
		return nil, err
	}
	if c.Status != model.CampaignPaused {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 paused 可恢复", c.Status))
	}
	now := time.Now()
	next := now.Add(time.Second) // 兜底：理论上 c.ScheduledAt 不该为 nil（paused 只能从 scheduled 而来）
	if c.ScheduledAt != nil {
		next = *c.ScheduledAt
		if !next.After(now) {
			next = nextRunAfter(next, now, c.RepeatEveryDays)
		}
	}
	if (c.RepeatMaxRuns > 0 && c.RunCount >= c.RepeatMaxRuns) || (c.RepeatEndAt != nil && next.After(*c.RepeatEndAt)) {
		return nil, errBadRequest("周期已结束，无剩余执行")
	}
	if err := s.casCampaignStatus(ctx, id, model.CampaignPaused, map[string]any{
		"status":       model.CampaignScheduled,
		"scheduled_at": next.UTC(),
	}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignScheduled
	c.ScheduledAt = &next
	return s.campaignView(ctx, c, extractTargetAppIDs(c.Targets)), nil
}

// CancelCampaign 取消一个定时活动（scheduled 或 paused → cancelled）。单次与周期任务都可取消。
func (s *Service) CancelCampaign(ctx context.Context, scope auth.Scope, id uint64) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkCampaignScope(ctx, scope, c); err != nil {
		return nil, err
	}
	if c.Status != model.CampaignScheduled && c.Status != model.CampaignPaused {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 scheduled/paused 可取消", c.Status))
	}
	if err := s.casCampaignStatus(ctx, id, c.Status, map[string]any{"status": model.CampaignCancelled}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignCancelled
	return s.campaignView(ctx, c, extractTargetAppIDs(c.Targets)), nil
}

// casCampaignStatus 以「当前状态仍为 from」为前提写入 fields；状态已被别处改掉（cron 刚触发发送、
// 周期刚跑完置 done、另一位操作者先动了手）时返回 409，让前端刷新后再决定，而不是静默覆盖。
func (s *Service) casCampaignStatus(ctx context.Context, id uint64, from string, fields map[string]any) error {
	ok, err := s.repo.UpdateCampaignFieldsIfStatus(ctx, id, from, fields)
	if err != nil {
		return err
	}
	if !ok {
		return errConflict("活动状态已变化（可能刚被定时任务触发或被他人操作），请刷新后重试")
	}
	return nil
}

// nextRunAfter 从 prev 起按 everyDays 步进，返回第一个晚于 now 的时间点。
// 纯函数，供 cron tick 与 ResumeCampaign 复用；宕机/暂停期间漏掉的运行不补发，只补最后一次。
// everyDays 必须 > 0（调用方保证，周期任务字段），否则会死循环。
func nextRunAfter(prev, now time.Time, everyDays int) time.Time {
	next := prev
	for !next.After(now) {
		next = next.AddDate(0, 0, everyDays)
	}
	return next
}

// SendCampaign 立即（或 dry-run）发送推送活动。
//
// dry-run=true：**无损预览**——只计算触达设备数/按 appId 分组，结果仅存在于响应体，
// 绝不改变 campaign 的 status、name、sentAt、success/failure 计数等持久化字段。
// 预览完活动仍保持原状（draft/scheduled），可随时真发。
//
// dry-run=false（真发）：PUSH_ENABLED 门控 + service account 校验；
// 置 sending → 异步 worker pool → 写 push_record → 终态 done/failed。
// 已处于 sending/done 的活动不可重复触发。
//
// scope 是调用者的数据范围（数据权限强制点：活动的 brand 在范围内才能发，dry-run 预览同样
// 受限——不能让一个只管 ap 的角色连预览都能看到 bp 的触达情况，见 docs/admin/10-rbac.md）。
func (s *Service) SendCampaign(ctx context.Context, scope auth.Scope, id uint64, dryRun bool) (*PushSendResult, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}

	// 取目标 appIds（dry-run 与真发共同需要）。
	appIDs := extractTargetAppIDs(c.Targets)
	if len(appIDs) == 0 {
		return nil, errBadRequest("活动没有目标渠道（targetAppIds 为空）")
	}
	if err := s.assertAppIDsInScope(ctx, scope, appIDs); err != nil {
		return nil, err
	}

	// 取目标 token（dry-run 只读，真发写 DB）。
	tokenMap, err := s.repo.ActiveTokensByAppIDs(ctx, appIDs)
	if err != nil {
		return nil, fmt.Errorf("查询目标 token 失败: %w", err)
	}

	// dry-run：纯内存计算，绝不写回任何持久化字段。
	if dryRun {
		preview := buildDryRunPreview(tokenMap)
		// campaign 原样返回（status/name/sentAt 均未改变）。
		return &PushSendResult{
			Campaign: *s.campaignView(ctx, c, appIDs),
			DryRun:   true,
			Preview:  preview,
		}, nil
	}

	// ---- 以下为真发路径 ----

	// 周期任务本体（父任务）不可被直接发送：真正发送的是每次触发克隆出的子活动
	// （RepeatEveryDays 恒为 0），父任务只用来记录「下次运行时间」。dry-run 预览不受此限。
	if c.RepeatEveryDays > 0 {
		return nil, errBadRequest("周期任务本体不可直接发送，请等待定时触发，或在历史里查看已发送的子活动")
	}
	// paused/cancelled 状态不允许发送（dry-run 不受此限制，仍可预览）。
	if c.Status == model.CampaignPaused || c.Status == model.CampaignCancelled {
		return nil, errBadRequest(fmt.Sprintf("活动已%s，不可发送", c.Status))
	}
	// 已发送/发送中的活动不允许重复发送（dry-run 不受此限制）。
	if c.Status == model.CampaignSending || c.Status == model.CampaignDone {
		return nil, errBadRequest(fmt.Sprintf("活动已处于 %s 状态，不可重复发送", c.Status))
	}

	// PUSH_ENABLED 门控。
	if !s.cfg.PushEnabled {
		return nil, NewError(http.StatusUnprocessableEntity, "FCM 未配置：PUSH_ENABLED=false，campaign 保持 draft")
	}

	// 统计总设备数（真发前记录到 campaign）。
	total := 0
	for _, ts := range tokenMap {
		total += len(ts)
	}

	// 置 sending：CAS 于读到的状态，与取消/暂停互斥——谁先写成功算谁的，另一方拿到 409。
	if err := s.casCampaignStatus(ctx, id, c.Status, map[string]any{
		"status":        model.CampaignSending,
		"total_devices": total,
	}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignSending
	c.TotalDevices = total

	// 真实发送：worker pool（异步）。
	go s.doSend(context.Background(), id, c, tokenMap, appIDs)

	return &PushSendResult{
		Campaign: *s.campaignView(ctx, c, appIDs),
		DryRun:   false,
	}, nil
}

// buildDryRunPreview 纯内存计算 dry-run 预览数据（不操作 DB）。
func buildDryRunPreview(tokenMap map[string][]model.PushDeviceToken) *DryRunPreview {
	byApp := make(map[string]int, len(tokenMap))
	total := 0
	for appID, tokens := range tokenMap {
		byApp[appID] = len(tokens)
		total += len(tokens)
	}
	return &DryRunPreview{
		TotalDevices: total,
		ByApp:        byApp,
	}
}

// PushAudience 预估目标活跃设备数（发送前展示）。scope 是调用者的数据范围（数据权限强制点：
// 按范围过滤——否则一个只管 ap 的角色能看到/给 bp 的用户发推送，见 docs/admin/10-rbac.md）。
// 越界的 appId 静默丢弃（预览类端点，不报错，与 filterAppIDsByScope 语义一致）。
func (s *Service) PushAudience(ctx context.Context, scope auth.Scope, appIDs []string) (*AudienceResult, error) {
	appIDs = s.filterAppIDsByScope(ctx, scope, appIDs)
	if len(appIDs) == 0 {
		return &AudienceResult{TotalDevices: 0, ByApp: map[string]int64{}}, nil
	}
	byApp, err := s.repo.CountActiveTokensByAppIDs(ctx, appIDs)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, n := range byApp {
		total += n
	}
	return &AudienceResult{TotalDevices: total, ByApp: byApp}, nil
}

// UploadPushImageRaw 上传推送图片到对象存储（复用 Storage 接口），接受 io.Reader。
func (s *Service) UploadPushImageRaw(ctx context.Context, r io.Reader, size int64, contentType, filename string) (string, error) {
	ext := ".jpg"
	if strings.HasSuffix(strings.ToLower(filename), ".png") || strings.Contains(contentType, "png") {
		ext = ".png"
	}
	key := fmt.Sprintf("push/images/%d%s", time.Now().UnixMilli(), ext)
	publicURL, err := s.storage.Put(ctx, key, r, size, contentType)
	if err != nil {
		return "", fmt.Errorf("上传推送图片失败: %w", err)
	}
	return publicURL, nil
}

// RunScheduledCampaigns cron 调用：扫描到期的定时活动并触发发送（PUSH_CRON_ENABLE 门控）。
// 到期活动分两类处理：
//   - 单次（RepeatEveryDays==0）：按 kind 分派直接真发，行为与周期化之前完全一致。
//   - 周期（RepeatEveryDays>0）：本体从不直接发送，走 tickRecurringCampaign 认领 + 克隆子活动。
func (s *Service) RunScheduledCampaigns(ctx context.Context) {
	campaigns, err := s.repo.ListScheduledCampaigns(ctx)
	if err != nil {
		fmt.Printf("[push-cron] 查询定时活动失败: %v\n", err)
		return
	}
	// 发送走 goroutine，而 cron 回调在本函数返回后立刻 cancel 掉 ctx（见 cmd/server/main.go 的
	// defer cancel）——异步发送必须脱离这个取消信号，否则 SendCampaign 里的 DB 查询会随机报
	// context canceled，导致到点的活动一直发不出去。
	sendCtx := context.WithoutCancel(ctx)
	for _, c := range campaigns {
		c := c
		if c.RepeatEveryDays <= 0 {
			go func() {
				// cron 是系统触发，不是人类请求，不受数据权限约束（活动的 brand 范围已在创建/设置
				// 定时时校验过一次，见 CreateCampaign/UpdateCampaign）。
				if err := s.dispatchScheduledSend(sendCtx, &c); err != nil {
					fmt.Printf("[push-cron] 活动 %d 触发发送失败: %v\n", c.ID, err)
				}
			}()
			continue
		}
		// 周期任务：PUSH_ENABLED 未开时不推进 scheduled_at，等启用后再发（不错过、不空转跑掉次数）。
		if !s.cfg.PushEnabled {
			fmt.Printf("[push-cron] 周期活动 %d 到期但 PUSH_ENABLED=false，本轮跳过（不推进）\n", c.ID)
			continue
		}
		s.tickRecurringCampaign(ctx, &c)
	}
}

// dispatchScheduledSend 按 kind 把「真发」分派到 SendCampaign / SendListingCampaign。
func (s *Service) dispatchScheduledSend(ctx context.Context, c *model.PushCampaign) error {
	if c.Kind == model.CampaignKindListing {
		_, err := s.SendListingCampaign(ctx, c.ID, false)
		return err
	}
	_, err := s.SendCampaign(ctx, auth.FullScope(), c.ID, false)
	return err
}

// tickRecurringCampaign 处理一个到期的周期任务（父任务）：
//  1. 算出下次运行时间 + 是否结束；
//  2. 用 CAS 原子认领这一轮触发（防止同一 tick / 多进程重复触发）；
//  3. 认领成功后克隆一条子活动并异步真发，子活动独立记录 status/统计/push_record。
func (s *Service) tickRecurringCampaign(ctx context.Context, c *model.PushCampaign) {
	if c.ScheduledAt == nil {
		fmt.Printf("[push-cron] 周期活动 %d 缺少 scheduled_at，跳过\n", c.ID)
		return
	}
	now := time.Now()
	prevScheduled := *c.ScheduledAt
	// 兜底：还没到点就不认领（防御查询侧时区/时钟偏差导致提前命中——否则 nextRunAfter
	// 会原样返回 prev，run_count 白涨一次、这一轮还会在到点后再发一遍）。
	if prevScheduled.After(now) {
		return
	}
	next := nextRunAfter(prevScheduled, now, c.RepeatEveryDays)
	runCount := c.RunCount + 1
	finished := (c.RepeatMaxRuns > 0 && runCount >= c.RepeatMaxRuns) ||
		(c.RepeatEndAt != nil && next.After(*c.RepeatEndAt))
	newStatus := model.CampaignScheduled
	if finished {
		newStatus = model.CampaignDone
	}

	child, appIDs, listingIDs := buildRunChild(c, runCount)
	claimed, err := s.repo.ClaimRecurringCampaignRun(ctx, c.ID, prevScheduled, repo.RecurringClaim{
		RunCount:        runCount,
		LastRunAt:       now,
		NextScheduledAt: next,
		Status:          newStatus,
	}, child, appIDs, listingIDs)
	if err != nil {
		// 事务已整体回滚：父任务未推进、未耗次数，下一轮 tick 会重试。
		fmt.Printf("[push-cron] 认领周期活动 %d 失败（已回滚，下轮重试）: %v\n", c.ID, err)
		return
	}
	if !claimed {
		// 已被别处（并发 tick/多进程）认领，本轮什么都不做，避免重复发送。
		return
	}

	go func() {
		if err := s.dispatchScheduledSend(context.WithoutCancel(ctx), child); err != nil {
			fmt.Printf("[push-cron] 周期活动 %d 的子活动 %d 发送失败: %v\n", c.ID, child.ID, err)
			_ = s.repo.UpdateCampaignFields(context.Background(), child.ID, map[string]any{"status": model.CampaignFailed})
		}
	}()
}

// buildRunChild 为周期任务的一次触发构造子活动（尚未落库）：同 kind、同内容，name 加「· 第N次」，
// status=draft（落库后随即被 dispatchScheduledSend 真发），parent_id 指向父任务；目标从父任务的
// Targets 复制（channel 取 applicationId，listing 取 listing_id）。落库与父任务推进在同一事务里做，
// 见 repo.ClaimRecurringCampaignRun。
func buildRunChild(parent *model.PushCampaign, runNo int) (*model.PushCampaign, []string, []uint64) {
	parentID := parent.ID
	child := &model.PushCampaign{
		Kind:         parent.Kind,
		Name:         fmt.Sprintf("%s · 第%d次", parent.Name, runNo),
		Title:        parent.Title,
		Body:         parent.Body,
		ImageURL:     parent.ImageURL,
		DeeplinkPath: parent.DeeplinkPath,
		ExtraData:    parent.ExtraData,
		Status:       model.CampaignDraft,
		CreatedBy:    parent.CreatedBy,
		ParentID:     &parentID,
	}
	var appIDs []string
	var listingIDs []uint64
	for _, t := range parent.Targets {
		switch {
		case parent.Kind == model.CampaignKindListing && t.ListingID != nil:
			listingIDs = append(listingIDs, *t.ListingID)
		case parent.Kind != model.CampaignKindListing && t.ApplicationID != "":
			appIDs = append(appIDs, t.ApplicationID)
		}
	}
	return child, appIDs, listingIDs
}

// ---------- 内部辅助 ----------

// fcmJob 一个待发送任务：路由键（决定发往哪个 Firebase 项目）+ 目标 token。
type fcmJob struct {
	routeKey string
	token    model.PushDeviceToken
}

// fcmJobResult 单条发送结果。
type fcmJobResult struct {
	token        model.PushDeviceToken
	err          error
	unregistered bool
	skipped      bool
}

// fcmSendStat 某分组（渠道按 appID / 上架包按 listingID）的发送计数与错误样本。
type fcmSendStat struct {
	sent    int
	failed  int
	skipped int
	errSamp string
}

// apply 累计一条结果；返回非空字符串表示该 token 已失效需下线。
func (st *fcmSendStat) apply(r fcmJobResult) (deadToken string) {
	switch {
	case r.skipped:
		st.skipped++
		if st.errSamp == "" {
			st.errSamp = "FCM 项目未配置，已跳过（如 gp2/listings 暂无私钥）"
		}
	case r.err == nil:
		st.sent++
	default:
		st.failed++
		if st.errSamp == "" {
			st.errSamp = r.err.Error()
		}
		if r.unregistered {
			return r.token.DeviceToken
		}
	}
	return ""
}

// buildPushData 组装 FCM data payload（deeplink + 活动自定义 extraData）。
func buildPushData(c *model.PushCampaign) map[string]string {
	data := map[string]string{"deeplink_path": c.DeeplinkPath}
	if c.ExtraData != "" {
		var extra map[string]string
		if err := json.Unmarshal([]byte(c.ExtraData), &extra); err == nil {
			for k, v := range extra {
				data[k] = v
			}
		}
	}
	return data
}

// dispatchFCMJobs 用 worker pool 并发发送一批任务，返回逐条结果（顺序不保证）。
// 渠道推送与上架包推送共用此发送内核，各自负责「构建 jobs」与「按自己的键汇总结果」。
// 每个 token 在公共 data 之上叠加自己的 palcode（透传给端上拼 URL）。
func (s *Service) dispatchFCMJobs(ctx context.Context, jobs []fcmJob, c *model.PushCampaign, data map[string]string) []fcmJobResult {
	if len(jobs) == 0 {
		return nil
	}
	jobCh := make(chan fcmJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	resCh := make(chan fcmJobResult, len(jobs))
	var wg sync.WaitGroup
	for i := 0; i < min(fcmWorkerCount, len(jobs)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				d := make(map[string]string, len(data)+1)
				for k, v := range data {
					d[k] = v
				}
				d["palcode"] = j.token.PalCode
				r := s.fcm.Send(ctx, j.routeKey, j.token.DeviceToken, c.Title, c.Body, c.ImageURL, d)
				resCh <- fcmJobResult{token: j.token, err: r.Err, unregistered: r.Unregistered, skipped: r.Skipped}
			}
		}()
	}
	wg.Wait()
	close(resCh)

	out := make([]fcmJobResult, 0, len(jobs))
	for r := range resCh {
		out = append(out, r)
	}
	return out
}

// doSend 在 goroutine 中执行真实 FCM 发送（worker pool），结束后更新 campaign 统计。
func (s *Service) doSend(ctx context.Context, campaignID uint64, c *model.PushCampaign, tokenMap map[string][]model.PushDeviceToken, appIDs []string) {
	data := buildPushData(c)

	// 构建 applicationId → 路由键 映射（gp 拆分：gp2 溢出包路由到 gp2 项目）。
	// 数据源是已上传的 fcm/gp2/google-services.json；未上传则索引为空、全部退回品牌路由。
	appIndex := s.buildFCMAppIndex(ctx)

	// 汇总所有 token 任务，逐 token 解析路由键（命中 gp2 索引→gp2，否则品牌 code）。
	var jobs []fcmJob
	for _, tokens := range tokenMap {
		for _, t := range tokens {
			key := resolveFCMKey(appIndex, t.ApplicationID, t.BrandCode)
			jobs = append(jobs, fcmJob{routeKey: key, token: t})
		}
	}

	results := s.dispatchFCMJobs(ctx, jobs, c, data)

	// 汇总结果（按 applicationId）。skipped（路由项目未配置，如 gp2 暂无私钥）单独计：
	// 不算 sent 也不算 failed，不下线 token——保证私钥就绪后这些设备能正常补发。
	stats := map[string]*fcmSendStat{}
	var deadTokens []string
	for _, r := range results {
		appID := r.token.ApplicationID
		if _, ok := stats[appID]; !ok {
			stats[appID] = &fcmSendStat{}
		}
		if dead := stats[appID].apply(r); dead != "" {
			deadTokens = append(deadTokens, dead)
		}
	}

	// 批量下线失效 token。
	if len(deadTokens) > 0 {
		_ = s.repo.DeactivateTokens(ctx, deadTokens)
	}

	// 写 push_record。
	now := time.Now()
	records := make([]model.PushRecord, 0, len(stats))
	var totalSent, totalFailed int
	for appID, st := range stats {
		totalSent += st.sent
		totalFailed += st.failed
		records = append(records, model.PushRecord{
			CampaignID:    campaignID,
			ApplicationID: appID,
			Sent:          st.sent,
			Failed:        st.failed,
			ErrorSample:   st.errSamp,
			FinishedAt:    &now,
		})
	}
	_ = s.repo.BatchUpsertPushRecords(ctx, campaignID, records)

	// 更新 campaign 统计与终态。
	finalStatus := model.CampaignDone
	if totalFailed > 0 && totalSent == 0 {
		finalStatus = model.CampaignFailed
	}
	_ = s.repo.UpdateCampaignFields(ctx, campaignID, map[string]any{
		"status":        finalStatus,
		"sent_at":       now,
		"success_count": totalSent,
		"failure_count": totalFailed,
	})
}

// campaignView 把 model.PushCampaign 转换为 PushCampaignView。
func (s *Service) campaignView(ctx context.Context, c *model.PushCampaign, appIDs []string) *PushCampaignView {
	kind := c.Kind
	if kind == "" {
		kind = model.CampaignKindChannel // 兼容 000007 之前无 kind 列的历史行
	}
	v := &PushCampaignView{
		ID:           c.ID,
		Kind:         kind,
		Name:         c.Name,
		Title:        c.Title,
		Body:         c.Body,
		ImageURL:     c.ImageURL,
		DeeplinkPath: c.DeeplinkPath,
		Status:       c.Status,
		ScheduledAt:  c.ScheduledAt,
		SentAt:       c.SentAt,
		TotalDevices: c.TotalDevices,
		SuccessCount: c.SuccessCount,
		FailureCount: c.FailureCount,
		CreatedBy:    c.CreatedBy,
		CreatedAt:    c.CreatedAt,
		TargetAppIDs: appIDs,

		RepeatEveryDays: c.RepeatEveryDays,
		RepeatEndAt:     c.RepeatEndAt,
		RepeatMaxRuns:   c.RepeatMaxRuns,
		RunCount:        c.RunCount,
		LastRunAt:       c.LastRunAt,
		ParentID:        c.ParentID,
	}
	if c.ExtraData != "" {
		var extra map[string]string
		if err := json.Unmarshal([]byte(c.ExtraData), &extra); err == nil {
			v.ExtraData = extra
		}
	}
	// 上架包活动：补目标 listing_id 列表（容错，失败留空不阻断）。
	if kind == model.CampaignKindListing {
		if ids, err := s.repo.GetCampaignTargetListingIDs(ctx, c.ID); err == nil {
			v.ListingIDs = ids
		}
	}
	return v
}

// validateCampaignInput 基础校验。
func validateCampaignInput(in PushCampaignInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return errBadRequest("name 不得为空")
	}
	if strings.TrimSpace(in.Title) == "" {
		return errBadRequest("title 不得为空")
	}
	if strings.TrimSpace(in.Body) == "" {
		return errBadRequest("body 不得为空")
	}
	if len(in.TargetAppIDs) == 0 {
		return errBadRequest("targetAppIds 不得为空")
	}
	// deeplinkPath 只能存相对路径，守 ADR-0002。
	if in.DeeplinkPath != "" && (strings.HasPrefix(in.DeeplinkPath, "http://") || strings.HasPrefix(in.DeeplinkPath, "https://")) {
		return errBadRequest("deeplinkPath 应为相对路径（不含域名），守 ADR-0002")
	}
	return nil
}

// marshalExtraData 把 map[string]string 序列化为 JSON 字符串存入 DB。
func marshalExtraData(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// extractTargetAppIDs 从 PushCampaignTarget 切片中提取 applicationId 列表。
func extractTargetAppIDs(targets []model.PushCampaignTarget) []string {
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.ApplicationID)
	}
	return out
}

// min 返回两整数的最小值（Go 1.21+ 内置，保留兼容）。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
