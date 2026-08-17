// Package service — 推送活动业务逻辑（ADR-0012）。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	list, err := s.repo.ListCampaigns(ctx, f)
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
	list, err := s.repo.ListCampaigns(ctx, repo.CampaignFilter{Kind: model.CampaignKindListing, Limit: 100})
	if err != nil {
		return nil, err
	}
	out := make([]PushCampaignView, 0, len(list))
	for i := range list {
		out = append(out, *s.campaignView(ctx, &list[i], nil))
	}
	return out, nil
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

// ScheduleCampaign 设置定时发送时间（draft → scheduled）。
func (s *Service) ScheduleCampaign(ctx context.Context, id uint64, scheduledAt time.Time) (*PushCampaignView, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status != model.CampaignDraft {
		return nil, errBadRequest(fmt.Sprintf("活动状态为 %s，仅 draft 可设置定时", c.Status))
	}
	if scheduledAt.Before(time.Now()) {
		return nil, errBadRequest("scheduledAt 不得早于当前时间")
	}
	if err := s.repo.UpdateCampaignFields(ctx, id, map[string]any{
		"status":       model.CampaignScheduled,
		"scheduled_at": scheduledAt,
	}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignScheduled
	c.ScheduledAt = &scheduledAt
	appIDs := extractTargetAppIDs(c.Targets)
	return s.campaignView(ctx, c, appIDs), nil
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

	// 置 sending。
	if err := s.repo.UpdateCampaignFields(ctx, id, map[string]any{
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
func (s *Service) RunScheduledCampaigns(ctx context.Context) {
	campaigns, err := s.repo.ListScheduledCampaigns(ctx)
	if err != nil {
		fmt.Printf("[push-cron] 查询定时活动失败: %v\n", err)
		return
	}
	for _, c := range campaigns {
		c := c
		go func() {
			// cron 是系统触发，不是人类请求，不受数据权限约束（活动的 brand 范围已在创建/设置
			// 定时时校验过一次，见 CreateCampaign/UpdateCampaign）。
			if _, err := s.SendCampaign(ctx, auth.FullScope(), c.ID, false); err != nil {
				fmt.Printf("[push-cron] 活动 %d 触发发送失败: %v\n", c.ID, err)
			}
		}()
	}
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
