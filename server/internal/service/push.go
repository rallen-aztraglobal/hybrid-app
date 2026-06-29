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
	Name          string            `json:"name"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	ImageURL      string            `json:"imageUrl"`
	DeeplinkPath  string            `json:"deeplinkPath"`
	ExtraData     map[string]string `json:"extraData"`
	TargetAppIDs  []string          `json:"targetAppIds"`
}

// PushCampaignView 是对前端展示友好的活动响应（对应 API 契约 PushCampaign）。
type PushCampaignView struct {
	ID            uint64            `json:"id"`
	Name          string            `json:"name"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	ImageURL      string            `json:"imageUrl"`
	DeeplinkPath  string            `json:"deeplinkPath"`
	ExtraData     map[string]string `json:"extraData,omitempty"`
	TargetAppIDs  []string          `json:"targetAppIds"`
	Status        string            `json:"status"`
	ScheduledAt   *time.Time        `json:"scheduledAt,omitempty"`
	SentAt        *time.Time        `json:"sentAt,omitempty"`
	TotalDevices  int               `json:"totalDevices"`
	SuccessCount  int               `json:"successCount"`
	FailureCount  int               `json:"failureCount"`
	CreatedBy     string            `json:"createdBy"`
	CreatedAt     time.Time         `json:"createdAt"`
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
	Campaign PushCampaignView  `json:"campaign"`
	DryRun   bool              `json:"dryRun"`
	// Preview 仅 dryRun=true 时填充：预览各 appId 的预计触达数，不持久化。
	Preview  *DryRunPreview    `json:"preview,omitempty"`
}

// DryRunPreview dry-run 预览数据（只存在于响应，绝不写回 campaign 持久字段）。
type DryRunPreview struct {
	TotalDevices int                `json:"totalDevices"`
	ByApp        map[string]int     `json:"byApp"` // applicationId → 活跃 token 数
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

// CreateCampaign 创建推送活动草稿。
func (s *Service) CreateCampaign(ctx context.Context, in PushCampaignInput, createdBy string) (*PushCampaignView, error) {
	if err := validateCampaignInput(in); err != nil {
		return nil, err
	}
	c := &model.PushCampaign{
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

// ListCampaigns 查询推送活动列表。
func (s *Service) ListCampaigns(ctx context.Context, brand string) ([]PushCampaignView, error) {
	list, err := s.repo.ListCampaigns(ctx, repo.CampaignFilter{Brand: brand, Limit: 100})
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

// UpdateCampaign 修改草稿活动（仅 draft 可改）。
func (s *Service) UpdateCampaign(ctx context.Context, id uint64, in PushCampaignInput) (*PushCampaignView, error) {
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
func (s *Service) SendCampaign(ctx context.Context, id uint64, dryRun bool) (*PushSendResult, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}

	// 取目标 appIds（dry-run 与真发共同需要）。
	appIDs := extractTargetAppIDs(c.Targets)
	if len(appIDs) == 0 {
		return nil, errBadRequest("活动没有目标渠道（targetAppIds 为空）")
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

// PushAudience 预估目标活跃设备数（发送前展示）。
func (s *Service) PushAudience(ctx context.Context, appIDs []string) (*AudienceResult, error) {
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
			if _, err := s.SendCampaign(ctx, c.ID, false); err != nil {
				fmt.Printf("[push-cron] 活动 %d 触发发送失败: %v\n", c.ID, err)
			}
		}()
	}
}

// ---------- 内部辅助 ----------

// doSend 在 goroutine 中执行真实 FCM 发送（worker pool），结束后更新 campaign 统计。
func (s *Service) doSend(ctx context.Context, campaignID uint64, c *model.PushCampaign, tokenMap map[string][]model.PushDeviceToken, appIDs []string) {
	// 构建 FCM data payload。
	data := map[string]string{
		"deeplink_path": c.DeeplinkPath,
	}
	if c.ExtraData != "" {
		var extra map[string]string
		if err := json.Unmarshal([]byte(c.ExtraData), &extra); err == nil {
			for k, v := range extra {
				data[k] = v
			}
		}
	}

	type tokenJob struct {
		routeKey string
		token    model.PushDeviceToken
	}

	// 构建 applicationId → 路由键 映射（gp 拆分：gp2 溢出包路由到 gp2 项目）。
	// 数据源是已上传的 fcm/gp2/google-services.json；未上传则索引为空、全部退回品牌路由。
	appIndex := s.buildFCMAppIndex(ctx)

	// 汇总所有 token 任务，逐 token 解析路由键（命中 gp2 索引→gp2，否则品牌 code）。
	var jobs []tokenJob
	for appID, tokens := range tokenMap {
		_ = appID
		for _, t := range tokens {
			key := resolveFCMKey(appIndex, t.ApplicationID, t.BrandCode)
			jobs = append(jobs, tokenJob{routeKey: key, token: t})
		}
	}

	// worker pool。
	jobCh := make(chan tokenJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	type sendRes struct {
		appID        string
		token        string
		err          error
		unregistered bool
		skipped      bool
	}
	resCh := make(chan sendRes, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < min(fcmWorkerCount, len(jobs)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				// 为每个 token 附上 palcode（透传给 APK 端 URL 拼接）。
				d := make(map[string]string, len(data)+1)
				for k, v := range data {
					d[k] = v
				}
				d["palcode"] = j.token.PalCode
				r := s.fcm.Send(ctx, j.routeKey, j.token.DeviceToken, c.Title, c.Body, c.ImageURL, d)
				resCh <- sendRes{
					appID:        j.token.ApplicationID,
					token:        j.token.DeviceToken,
					err:          r.Err,
					unregistered: r.Unregistered,
					skipped:      r.Skipped,
				}
			}
		}()
	}
	wg.Wait()
	close(resCh)

	// 汇总结果。skipped（路由项目未配置，如 gp2 暂无私钥）单独计：不算 sent 也不算 failed，
	// 不下线 token——保证 gp2 私钥就绪后这些设备能正常补发。
	type appStat struct {
		sent    int
		failed  int
		skipped int
		errSamp string
	}
	stats := map[string]*appStat{}
	var deadTokens []string
	for r := range resCh {
		if _, ok := stats[r.appID]; !ok {
			stats[r.appID] = &appStat{}
		}
		switch {
		case r.skipped:
			stats[r.appID].skipped++
			if stats[r.appID].errSamp == "" {
				stats[r.appID].errSamp = "FCM 项目未配置，已跳过（如 gp2 暂无私钥）"
			}
		case r.err == nil:
			stats[r.appID].sent++
		default:
			stats[r.appID].failed++
			if stats[r.appID].errSamp == "" {
				stats[r.appID].errSamp = r.err.Error()
			}
			if r.unregistered {
				deadTokens = append(deadTokens, r.token)
			}
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
func (s *Service) campaignView(_ context.Context, c *model.PushCampaign, appIDs []string) *PushCampaignView {
	v := &PushCampaignView{
		ID:           c.ID,
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
