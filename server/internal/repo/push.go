// Package repo — 推送功能数据访问（ADR-0012）。
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hybrid-app/server/internal/model"
)

// ---------- PushDeviceToken ----------

// UpsertDeviceToken 按 device_token 做 upsert：
// 存在时更新 application_id / pal_code / last_seen_at / is_active=true；不存在时插入。
// 依赖唯一索引 device_token UNIQUE，用 GORM OnConflict clause 实现原子 upsert。
func (r *Repo) UpsertDeviceToken(ctx context.Context, t *model.PushDeviceToken) error {
	t.LastSeenAt = time.Now()
	t.IsActive = true
	// GORM OnConflict: 只更新关键字段，不改 created_at。
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "device_token"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"application_id", "brand_code", "pal_code",
				"platform", "model_info", "is_active", "last_seen_at",
			}),
		}).Create(t)
	if result.Error != nil {
		return fmt.Errorf("upsert device token 失败: %w", result.Error)
	}
	return nil
}

// DeactivateTokens 批量将一组 device_token 置为 is_active=false（FCM 返回 UNREGISTERED/INVALID_ARGUMENT）。
func (r *Repo) DeactivateTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Model(&model.PushDeviceToken{}).
		Where("device_token IN ?", tokens).
		Update("is_active", false).Error; err != nil {
		return fmt.Errorf("置失效 token 失败: %w", err)
	}
	return nil
}

// ActiveTokensByAppIDs 取给定 applicationId 集合的全部活跃 token，按 applicationId 分组返回。
func (r *Repo) ActiveTokensByAppIDs(ctx context.Context, appIDs []string) (map[string][]model.PushDeviceToken, error) {
	if len(appIDs) == 0 {
		return nil, nil
	}
	var list []model.PushDeviceToken
	if err := r.db.WithContext(ctx).
		Where("application_id IN ? AND is_active = ?", appIDs, true).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询活跃 token 失败: %w", err)
	}
	out := make(map[string][]model.PushDeviceToken, len(appIDs))
	for _, t := range list {
		out[t.ApplicationID] = append(out[t.ApplicationID], t)
	}
	return out, nil
}

// CountActiveTokensByAppIDs 统计各 applicationId 的活跃设备数（用于 audience 预估）。
func (r *Repo) CountActiveTokensByAppIDs(ctx context.Context, appIDs []string) (map[string]int64, error) {
	if len(appIDs) == 0 {
		return map[string]int64{}, nil
	}
	type row struct {
		ApplicationID string
		Cnt           int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&model.PushDeviceToken{}).
		Select("application_id, count(*) as cnt").
		Where("application_id IN ? AND is_active = ?", appIDs, true).
		Group("application_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计活跃设备数失败: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, x := range rows {
		out[x.ApplicationID] = x.Cnt
	}
	return out, nil
}

// ---------- PushCampaign ----------

// CreateCampaign 写入推送活动。
func (r *Repo) CreateCampaign(ctx context.Context, c *model.PushCampaign) error {
	if err := r.db.WithContext(ctx).Create(c).Error; err != nil {
		return fmt.Errorf("创建推送活动失败: %w", err)
	}
	return nil
}

// CampaignFilter 推送活动列表筛选。
type CampaignFilter struct {
	Brand string // 按 brand code 筛选（内联查 push_campaign_target → channel.brand_id）
	Limit int
}

// ListCampaigns 按条件查推送活动（倒序），预加载 Targets。
func (r *Repo) ListCampaigns(ctx context.Context, f CampaignFilter) ([]model.PushCampaign, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	q := r.db.WithContext(ctx).Model(&model.PushCampaign{}).
		Preload("Targets").
		Order("id desc").Limit(f.Limit)
	if f.Brand != "" {
		// 过滤：该活动至少有一个 target 属于指定 brand。
		q = q.Where(`id IN (
			SELECT DISTINCT pct.campaign_id
			FROM push_campaign_target pct
			JOIN channel ch ON ch.application_id = pct.application_id
			JOIN brand b ON b.id = ch.brand_id
			WHERE b.code = ?
		)`, f.Brand)
	}
	var list []model.PushCampaign
	if err := q.Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询推送活动失败: %w", err)
	}
	return list, nil
}

// GetCampaign 按 id 取推送活动，预加载 Targets 与 Records。
func (r *Repo) GetCampaign(ctx context.Context, id uint64) (*model.PushCampaign, error) {
	var c model.PushCampaign
	err := r.db.WithContext(ctx).
		Preload("Targets").
		Preload("Records").
		First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询推送活动失败: %w", err)
	}
	return &c, nil
}

// UpdateCampaign 保存推送活动（完整覆盖 + 替换 Targets）。
// 仅当 status=draft 可修改（业务层校验）。
func (r *Repo) UpdateCampaign(ctx context.Context, c *model.PushCampaign, targetAppIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(c).Error; err != nil {
			return fmt.Errorf("更新推送活动失败: %w", err)
		}
		// 替换 Targets。
		if err := tx.Where("campaign_id = ?", c.ID).Delete(&model.PushCampaignTarget{}).Error; err != nil {
			return fmt.Errorf("清空推送目标失败: %w", err)
		}
		if len(targetAppIDs) > 0 {
			targets := make([]model.PushCampaignTarget, 0, len(targetAppIDs))
			for _, appID := range targetAppIDs {
				targets = append(targets, model.PushCampaignTarget{CampaignID: c.ID, ApplicationID: appID})
			}
			if err := tx.Create(&targets).Error; err != nil {
				return fmt.Errorf("写入推送目标失败: %w", err)
			}
		}
		return nil
	})
}

// UpdateCampaignFields 局部更新推送活动字段。
func (r *Repo) UpdateCampaignFields(ctx context.Context, id uint64, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&model.PushCampaign{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("更新推送活动字段失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceCampaignTargets 事务性替换活动目标 appId 列表。
func (r *Repo) ReplaceCampaignTargets(ctx context.Context, campaignID uint64, appIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("campaign_id = ?", campaignID).Delete(&model.PushCampaignTarget{}).Error; err != nil {
			return fmt.Errorf("清空推送目标失败: %w", err)
		}
		if len(appIDs) == 0 {
			return nil
		}
		targets := make([]model.PushCampaignTarget, 0, len(appIDs))
		for _, appID := range appIDs {
			targets = append(targets, model.PushCampaignTarget{CampaignID: campaignID, ApplicationID: appID})
		}
		if err := tx.Create(&targets).Error; err != nil {
			return fmt.Errorf("写入推送目标失败: %w", err)
		}
		return nil
	})
}

// ListScheduledCampaigns 取 status=scheduled 且 scheduled_at<=now 的活动，供 cron 触发。
func (r *Repo) ListScheduledCampaigns(ctx context.Context) ([]model.PushCampaign, error) {
	var list []model.PushCampaign
	if err := r.db.WithContext(ctx).
		Preload("Targets").
		Where("status = ? AND scheduled_at <= ?", model.CampaignScheduled, time.Now()).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询定时活动失败: %w", err)
	}
	return list, nil
}

// ---------- PushRecord ----------

// UpsertPushRecord 按 (campaign_id, application_id) 累加发送结果。
// 简单实现：先删再插（活动发送期间不会并发写同一 appId 的 record）。
func (r *Repo) UpsertPushRecord(ctx context.Context, rec *model.PushRecord) error {
	now := time.Now()
	rec.FinishedAt = &now
	// 用 Save 实现 upsert（若有 ID 则 update，否则 insert）。
	if err := r.db.WithContext(ctx).Save(rec).Error; err != nil {
		return fmt.Errorf("upsert push record 失败: %w", err)
	}
	return nil
}

// BatchUpsertPushRecords 批量写入 push_record（替换已有的同活动同 appId 记录）。
func (r *Repo) BatchUpsertPushRecords(ctx context.Context, campaignID uint64, records []model.PushRecord) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清除旧记录（重试/dry-run 等场景需要幂等）。
		if err := tx.Where("campaign_id = ?", campaignID).Delete(&model.PushRecord{}).Error; err != nil {
			return fmt.Errorf("清空旧 push_record 失败: %w", err)
		}
		if err := tx.Create(&records).Error; err != nil {
			return fmt.Errorf("批量写入 push_record 失败: %w", err)
		}
		return nil
	})
}

// GetCampaignTargetAppIDs 取活动的所有目标 applicationId。
func (r *Repo) GetCampaignTargetAppIDs(ctx context.Context, campaignID uint64) ([]string, error) {
	var targets []model.PushCampaignTarget
	if err := r.db.WithContext(ctx).
		Where("campaign_id = ?", campaignID).
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("查询推送目标失败: %w", err)
	}
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		out = append(out, t.ApplicationID)
	}
	return out, nil
}
