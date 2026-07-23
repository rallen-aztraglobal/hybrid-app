// Package repo — 上架包（listing）模块的数据访问。
// 与 channel 的写法保持一致：预加载关联、局部字段更新、NotFound 归一化。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/model"
)

// ListingFilter 是上架包列表的筛选条件。
type ListingFilter struct {
	Platform string // android / ios，空 = 不筛
	Status   string // enabled/disabled/archived，空 = 不筛（但默认排除 archived，见下）
	Q        string // 关键词：名称/包名
}

// ListListings 返回上架包列表，预加载品牌、域名与网关，便于前端一次渲染卡片。
// 默认排除已归档（除非显式按 status=archived 查），与渠道列表口径一致。
func (r *Repo) ListListings(ctx context.Context, f ListingFilter) ([]model.ListingApp, error) {
	q := r.db.WithContext(ctx).
		Preload("Brand").
		Preload("Domains", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Gate").
		Model(&model.ListingApp{})

	if f.Platform != "" {
		q = q.Where("platform = ?", f.Platform)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	} else {
		q = q.Where("status <> ?", model.ListingArchived)
	}
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("name LIKE ? OR display_name LIKE ? OR bundle_id LIKE ?", like, like, like)
	}

	var list []model.ListingApp
	if err := q.Order("platform asc, id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询上架包失败: %w", err)
	}
	return list, nil
}

// GetListing 按 id 取上架包（含关联）。
func (r *Repo) GetListing(ctx context.Context, id uint64) (*model.ListingApp, error) {
	var l model.ListingApp
	err := r.db.WithContext(ctx).
		Preload("Brand").
		Preload("Domains", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Gate").
		First(&l, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询上架包失败: %w", err)
	}
	return &l, nil
}

// GetListingByPlatformBundle 按 (platform, bundle_id) 取上架包——这是客户端网关请求的解析键。
func (r *Repo) GetListingByPlatformBundle(ctx context.Context, platform, bundleID string) (*model.ListingApp, error) {
	var l model.ListingApp
	err := r.db.WithContext(ctx).
		Preload("Brand").
		Preload("Domains", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Preload("Gate").
		Where("platform = ? AND bundle_id = ?", platform, bundleID).
		First(&l).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询上架包失败: %w", err)
	}
	return &l, nil
}

// CountListingsByPlatformBundle 统计 (platform, bundle_id) 冲突数，排除 excludeID（更新时用）。
// 用于在 Service 层做唯一性校验，给出比 DB 约束更友好的错误信息。
func (r *Repo) CountListingsByPlatformBundle(ctx context.Context, platform, bundleID string, excludeID uint64) (int64, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&model.ListingApp{}).
		Where("platform = ? AND bundle_id = ?", platform, bundleID)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&n).Error; err != nil {
		return 0, fmt.Errorf("统计上架包唯一性失败: %w", err)
	}
	return n, nil
}

// CreateListing 新建上架包。
func (r *Repo) CreateListing(ctx context.Context, l *model.ListingApp) error {
	if err := r.db.WithContext(ctx).Create(l).Error; err != nil {
		return fmt.Errorf("创建上架包失败: %w", err)
	}
	return nil
}

// UpdateListingFields 局部更新上架包字段。platform/bundle_id 一旦创建不建议改，调用方自行约束。
func (r *Repo) UpdateListingFields(ctx context.Context, id uint64, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&model.ListingApp{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("更新上架包失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteListing 删除上架包（级联删除 domains/gate/gate_log 由外键 ON DELETE CASCADE 处理）。
func (r *Repo) DeleteListing(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Delete(&model.ListingApp{}, id)
	if res.Error != nil {
		return fmt.Errorf("删除上架包失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceListingDomains 整体替换某上架包的 B 面域名覆盖（先删后插，事务内）。
// 传空切片 = 清空覆盖（配合把 use_brand_domains 置 true 回到继承）。
func (r *Repo) ReplaceListingDomains(ctx context.Context, listingID uint64, domains []model.ListingDomain) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("listing_id = ?", listingID).Delete(&model.ListingDomain{}).Error; err != nil {
			return fmt.Errorf("清除旧上架包域名失败: %w", err)
		}
		if len(domains) == 0 {
			return nil
		}
		for i := range domains {
			domains[i].ListingID = listingID
		}
		if err := tx.Create(&domains).Error; err != nil {
			return fmt.Errorf("写入上架包域名失败: %w", err)
		}
		return nil
	})
}

// UpsertListingGate 写入/更新某上架包的网关规则（一对一，按 listing_id upsert）。
func (r *Repo) UpsertListingGate(ctx context.Context, gate *model.ListingGate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ListingGate
		err := tx.Where("listing_id = ?", gate.ListingID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(gate).Error; err != nil {
				return fmt.Errorf("创建网关规则失败: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("查询网关规则失败: %w", err)
		}
		gate.ID = existing.ID
		if err := tx.Model(&model.ListingGate{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"countries":      gate.Countries,
			"timezone_deny":  gate.TimezoneDeny,
			"ip_allow_cidrs": gate.IPAllowCIDRs,
			"ip_deny_cidrs":  gate.IPDenyCIDRs,
		}).Error; err != nil {
			return fmt.Errorf("更新网关规则失败: %w", err)
		}
		return nil
	})
}

// CreateGateLog 落一条网关判定流水（容错：调用方通常忽略其错误，不阻断判定返回）。
func (r *Repo) CreateGateLog(ctx context.Context, l *model.ListingGateLog) error {
	if err := r.db.WithContext(ctx).Create(l).Error; err != nil {
		return fmt.Errorf("写入网关判定流水失败: %w", err)
	}
	return nil
}

// ListGateLogs 取某上架包最近的判定流水（倒序），供 Console 排查。
func (r *Repo) ListGateLogs(ctx context.Context, listingID uint64, limit int) ([]model.ListingGateLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var list []model.ListingGateLog
	if err := r.db.WithContext(ctx).
		Where("listing_id = ?", listingID).
		Order("id desc").
		Limit(limit).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询网关判定流水失败: %w", err)
	}
	return list, nil
}
