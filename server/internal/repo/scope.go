// Package repo — 数据权限（角色可见的品牌/渠道范围）的数据访问，docs/admin/10-rbac.md「数据权限」节。
package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/model"
)

// ---------- role_brand / role_channel ----------

// RoleBrandCodes 返回某角色的品牌范围 code 列表（仅 role.scope_all_brands=false 时有意义）。
func (r *Repo) RoleBrandCodes(ctx context.Context, roleID uint64) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).Model(&model.RoleBrand{}).
		Where("role_id = ?", roleID).Pluck("brand_code", &codes).Error; err != nil {
		return nil, fmt.Errorf("查询角色品牌范围失败: %w", err)
	}
	return codes, nil
}

// RoleChannelBrandRow 是 RoleChannelBrandCodes 的单行结果：某渠道 id + 其所属品牌 code
// （供 auth.RBAC.RoleEffectiveScope 与品牌范围求交）。
type RoleChannelBrandRow struct {
	ChannelID uint64
	BrandCode string
}

// RoleChannelBrandCodes 返回某角色的渠道范围列表，附带每个渠道所属的品牌 code
// （仅 role.scope_all_channels=false 时有意义；渠道若已被物理删除，因 FK ON DELETE CASCADE
// 该行也已随之清理，这里查到的都是仍存在的渠道）。
func (r *Repo) RoleChannelBrandCodes(ctx context.Context, roleID uint64) ([]RoleChannelBrandRow, error) {
	var rows []RoleChannelBrandRow
	if err := r.db.WithContext(ctx).
		Table("role_channel rc").
		Select("rc.channel_id AS channel_id, b.code AS brand_code").
		Joins("JOIN channel ch ON ch.id = rc.channel_id").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("rc.role_id = ?", roleID).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询角色渠道范围失败: %w", err)
	}
	return rows, nil
}

// ReplaceRoleBrands 事务性替换某角色的品牌范围。
func (r *Repo) ReplaceRoleBrands(ctx context.Context, roleID uint64, codes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleBrand{}).Error; err != nil {
			return fmt.Errorf("清空角色品牌范围失败: %w", err)
		}
		if len(codes) == 0 {
			return nil
		}
		rows := make([]model.RoleBrand, 0, len(codes))
		for _, code := range codes {
			rows = append(rows, model.RoleBrand{RoleID: roleID, BrandCode: code})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("写入角色品牌范围失败: %w", err)
		}
		return nil
	})
}

// ReplaceRoleChannels 事务性替换某角色的渠道范围。
func (r *Repo) ReplaceRoleChannels(ctx context.Context, roleID uint64, channelIDs []uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleChannel{}).Error; err != nil {
			return fmt.Errorf("清空角色渠道范围失败: %w", err)
		}
		if len(channelIDs) == 0 {
			return nil
		}
		rows := make([]model.RoleChannel, 0, len(channelIDs))
		for _, id := range channelIDs {
			rows = append(rows, model.RoleChannel{RoleID: roleID, ChannelID: id})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("写入角色渠道范围失败: %w", err)
		}
		return nil
	})
}

// ---------- 渠道 ↔ 品牌解析（供数据权限强制点复用） ----------

// ChannelBrandCodesByIDs 批量按渠道 id 取其所属品牌 code（用于校验 RoleInput.ChannelIDs
// 都是真实存在的渠道，以及后续做品牌范围求交）。不存在的 id 不会出现在返回的 map 中。
func (r *Repo) ChannelBrandCodesByIDs(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   uint64
		Code string
	}
	if err := r.db.WithContext(ctx).
		Table("channel ch").
		Select("ch.id AS id, b.code AS code").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("ch.id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询渠道品牌失败: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row.Code
	}
	return out, nil
}

// ChannelBrandInfo 是 ChannelBrandsByApplicationIDs 的单条结果。
type ChannelBrandInfo struct {
	ChannelID uint64
	BrandCode string
}

// ChannelBrandsByApplicationIDs 批量按 applicationId 取其渠道 id + 所属品牌 code
// （供推送/受众等按 appId 操作的场景做数据权限校验/过滤）。不存在对应渠道的 appId 不会出现在
// 返回的 map 中（调用方应把「查不到」视为不可见，fail-closed）。
func (r *Repo) ChannelBrandsByApplicationIDs(ctx context.Context, appIDs []string) (map[string]ChannelBrandInfo, error) {
	out := map[string]ChannelBrandInfo{}
	if len(appIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ApplicationID string
		ChannelID     uint64
		Code          string
	}
	if err := r.db.WithContext(ctx).
		Table("channel ch").
		Select("ch.application_id AS application_id, ch.id AS channel_id, b.code AS code").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("ch.application_id IN ?", appIDs).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询渠道品牌失败: %w", err)
	}
	for _, row := range rows {
		out[row.ApplicationID] = ChannelBrandInfo{ChannelID: row.ChannelID, BrandCode: row.Code}
	}
	return out, nil
}

// ApplicationIDsByChannelIDs 批量按渠道 id 取其 applicationId（供设备等以 applicationId
// 为维度存储的表按渠道范围过滤：先把渠道范围转成 applicationId 集合再 IN 查询）。
func (r *Repo) ApplicationIDsByChannelIDs(ctx context.Context, ids []uint64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var appIDs []string
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).
		Where("id IN ?", ids).Pluck("application_id", &appIDs).Error; err != nil {
		return nil, fmt.Errorf("批量查询渠道 applicationId 失败: %w", err)
	}
	return appIDs, nil
}

// GetChannelBrandCode 返回某渠道所属品牌的 code（轻量查询：只用于数据权限校验的「渠道是否在
// 范围内」判定，不加载整条 Channel 记录）。渠道不存在返回 ErrNotFound。
func (r *Repo) GetChannelBrandCode(ctx context.Context, channelID uint64) (string, error) {
	var row struct{ Code string }
	tx := r.db.WithContext(ctx).
		Table("channel ch").
		Select("b.code AS code").
		Joins("JOIN brand b ON b.id = ch.brand_id").
		Where("ch.id = ?", channelID).
		Take(&row)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("查询渠道品牌失败: %w", tx.Error)
	}
	return row.Code, nil
}
