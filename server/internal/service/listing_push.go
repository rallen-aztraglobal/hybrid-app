// Package service — 上架包推送发送流程（kind=listing 的 campaign）。
//
// 与渠道推送共用发送内核 dispatchFCMJobs / FCMManager，区别在三点：
//  1. 目标是 listing_id（而非 application_id）；
//  2. 受众**强制只含最近判定为 B 面的活跃设备**（repo.ActiveListingTokensBMode 硬过滤），
//     绝不把 B 面内容推给 A 面设备（可能是审核员）——这是本流程存在的首要安全约束；
//  3. 一律路由到独立的 listings Firebase 项目（fcmRouteKeyListings）。
package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hybrid-app/server/internal/model"
)

// fcmRouteKeyListings 是上架包推送在 FCMManager 里的路由键（对应独立 Firebase 项目）。
// 未配置私钥时 Send 返回 Skipped（不算失败），发送流程仍能完整跑通，待运维配好即可真发。
const fcmRouteKeyListings = "listings"

// CreateListingCampaignInput 是创建上架包推送活动的入参。
type CreateListingCampaignInput struct {
	Name         string
	Title        string
	Body         string
	ImageURL     string
	DeeplinkPath string
	ExtraData    map[string]string
	ListingIDs   []uint64 // 目标上架包
	CreatedBy    string
}

// CreateListingCampaign 创建一条 kind=listing 的推送活动（草稿）。
func (s *Service) CreateListingCampaign(ctx context.Context, in CreateListingCampaignInput) (*PushCampaignView, error) {
	if err := validateListingCampaign(in); err != nil {
		return nil, err
	}
	// 校验每个 listing 存在（防止误配无效 id）。
	for _, lid := range in.ListingIDs {
		if _, err := s.repo.GetListing(ctx, lid); err != nil {
			return nil, errBadRequest(fmt.Sprintf("上架包 id=%d 不存在", lid))
		}
	}

	c := &model.PushCampaign{
		Kind:         model.CampaignKindListing,
		Name:         in.Name,
		Title:        in.Title,
		Body:         in.Body,
		ImageURL:     in.ImageURL,
		DeeplinkPath: in.DeeplinkPath,
		ExtraData:    marshalExtraData(in.ExtraData),
		Status:       model.CampaignDraft,
		CreatedBy:    in.CreatedBy,
	}
	if err := s.repo.CreateCampaign(ctx, c); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceCampaignListingTargets(ctx, c.ID, in.ListingIDs); err != nil {
		return nil, err
	}
	return s.campaignView(ctx, c, nil), nil
}

// ListingCampaignAudience 预估某活动的 B 面可推送设备数（按 listing 分组）。
type ListingCampaignAudience struct {
	TotalDevices int              `json:"totalDevices"`
	ByListing    map[uint64]int64 `json:"byListing"`
}

// ListingCampaignAudienceFor 统计给定上架包集合的 B 面活跃设备数（发送前预览）。
func (s *Service) ListingCampaignAudienceFor(ctx context.Context, listingIDs []uint64) (*ListingCampaignAudience, error) {
	out := &ListingCampaignAudience{ByListing: map[uint64]int64{}}
	for _, lid := range listingIDs {
		n, err := s.repo.CountActiveListingTokensBMode(ctx, lid)
		if err != nil {
			return nil, err
		}
		out.ByListing[lid] = n
		out.TotalDevices += int(n)
	}
	return out, nil
}

// SendListingCampaign 立即（或 dry-run）发送一条上架包推送活动。
// 受众强制为各目标上架包的 B 面活跃设备。
func (s *Service) SendListingCampaign(ctx context.Context, id uint64, dryRun bool) (*PushSendResult, error) {
	c, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Kind != model.CampaignKindListing {
		return nil, errBadRequest("该活动不是上架包推送（kind != listing）")
	}

	listingIDs, err := s.repo.GetCampaignTargetListingIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(listingIDs) == 0 {
		return nil, errBadRequest("活动没有目标上架包")
	}

	// 收集各上架包的 B 面设备（硬过滤，只含 last_gate_mode='B'）。
	tokensByListing := map[uint64][]model.PushDeviceToken{}
	total := 0
	for _, lid := range listingIDs {
		byPlatform, err := s.repo.ActiveListingTokensBMode(ctx, lid)
		if err != nil {
			return nil, fmt.Errorf("查询上架包 %d 的 B 面设备失败: %w", lid, err)
		}
		for _, ts := range byPlatform {
			tokensByListing[lid] = append(tokensByListing[lid], ts...)
			total += len(ts)
		}
	}

	if dryRun {
		preview := map[string]int{}
		for lid, ts := range tokensByListing {
			preview[fmt.Sprintf("listing:%d", lid)] = len(ts)
		}
		return &PushSendResult{DryRun: true, Preview: &DryRunPreview{TotalDevices: total, ByApp: preview}}, nil
	}

	if c.Status == model.CampaignSending || c.Status == model.CampaignDone {
		return nil, errBadRequest(fmt.Sprintf("活动已处于 %s 状态，不可重复发送", c.Status))
	}
	if !s.cfg.PushEnabled {
		return nil, NewError(http.StatusUnprocessableEntity, "推送未启用：PUSH_ENABLED=false")
	}

	if err := s.repo.UpdateCampaignFields(ctx, id, map[string]any{
		"status":        model.CampaignSending,
		"total_devices": total,
	}); err != nil {
		return nil, err
	}
	c.Status = model.CampaignSending
	c.TotalDevices = total

	go s.doSendListing(context.Background(), id, c, tokensByListing)

	return &PushSendResult{DryRun: false}, nil
}

// doSendListing 执行上架包推送的真实发送（worker pool），按 listing_id 汇总并落 push_record。
func (s *Service) doSendListing(ctx context.Context, campaignID uint64, c *model.PushCampaign, tokensByListing map[uint64][]model.PushDeviceToken) {
	data := buildPushData(c)

	// 所有 token 一律路由到 listings 项目。
	var jobs []fcmJob
	for _, tokens := range tokensByListing {
		for _, t := range tokens {
			jobs = append(jobs, fcmJob{routeKey: fcmRouteKeyListings, token: t})
		}
	}

	results := s.dispatchFCMJobs(ctx, jobs, c, data)

	// 按 listing_id 汇总。
	stats := map[uint64]*fcmSendStat{}
	var deadTokens []string
	for _, r := range results {
		var lid uint64
		if r.token.ListingID != nil {
			lid = *r.token.ListingID
		}
		if _, ok := stats[lid]; !ok {
			stats[lid] = &fcmSendStat{}
		}
		if dead := stats[lid].apply(r); dead != "" {
			deadTokens = append(deadTokens, dead)
		}
	}

	if len(deadTokens) > 0 {
		_ = s.repo.DeactivateTokens(ctx, deadTokens)
	}

	now := time.Now()
	records := make([]model.PushRecord, 0, len(stats))
	var totalSent, totalFailed int
	for lid, st := range stats {
		totalSent += st.sent
		totalFailed += st.failed
		id := lid
		records = append(records, model.PushRecord{
			CampaignID:  campaignID,
			ListingID:   &id,
			Sent:        st.sent,
			Failed:      st.failed,
			ErrorSample: st.errSamp,
			FinishedAt:  &now,
		})
	}
	_ = s.repo.BatchUpsertPushRecords(ctx, campaignID, records)

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

// validateListingCampaign 校验上架包推送活动入参。
func validateListingCampaign(in CreateListingCampaignInput) error {
	if in.Name == "" || in.Title == "" || in.Body == "" {
		return errBadRequest("name/title/body 不能为空")
	}
	if len(in.ListingIDs) == 0 {
		return errBadRequest("至少选择一个目标上架包")
	}
	return nil
}
