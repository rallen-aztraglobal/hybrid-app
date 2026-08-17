// Package repo — 渠道设备上报数据访问。
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hybrid-app/server/internal/model"
)

// UpsertChannelDevice 按 device_key 做 upsert：
//   - 存在：merge——gaid/adid/oaid/device_name 来值非空才覆盖，application_id/brand_code/
//     pal_code/app_name 直接覆盖为最新（渠道快照随最近一次上报刷新），created_at 永不覆盖；
//   - 不存在：插入。
//
// 不用 MySQL 方言的条件更新（ON DUPLICATE KEY / ON CONFLICT DO UPDATE），改用「先 SELECT 再
// Save」的朴素实现，保证 sqlite（单测）与 mysql（生产）行为一致。
func (r *Repo) UpsertChannelDevice(ctx context.Context, d *model.ChannelDevice) error {
	var existing model.ChannelDevice
	err := r.db.WithContext(ctx).Where("device_key = ?", d.DeviceKey).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := r.db.WithContext(ctx).Create(d).Error; err != nil {
			return fmt.Errorf("创建设备上报记录失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询设备上报记录失败: %w", err)
	}

	// merge：来值非空才覆盖。
	if d.GAID != "" {
		existing.GAID = d.GAID
	}
	if d.ADID != "" {
		existing.ADID = d.ADID
	}
	if d.OAID != "" {
		existing.OAID = d.OAID
	}
	if d.DeviceName != "" {
		existing.DeviceName = d.DeviceName
	}
	// 渠道快照字段直接覆盖为最新（不判空——appId/brandCode/palCode/appName 均已在
	// service 层从 channel 记录取得，代表「当前」事实）。
	existing.ApplicationID = d.ApplicationID
	existing.BrandCode = d.BrandCode
	existing.PalCode = d.PalCode
	existing.AppName = d.AppName
	// existing.CreatedAt 保留原值不动。

	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return fmt.Errorf("更新设备上报记录失败: %w", err)
	}
	*d = existing
	return nil
}

// DeviceFilter 设备列表/导出筛选条件。
// From/To 为半开区间 [From, To)；均为 nil 表示不限。
//
// Scope* 是数据权限过滤（docs/admin/10-rbac.md）：opt-in，零值不生效（兼容既有调用点）。
// channel_device 表按 application_id 维度存储（不是 channel_id），渠道范围在 service 层已被
// 解析成 ScopeAppIDs（channel id → application_id）后传下来，这里直接按 application_id 过滤。
type DeviceFilter struct {
	ApplicationID string
	From          *time.Time
	To            *time.Time
	Page          int
	PageSize      int

	ScopeRestricted  bool
	ScopeAllBrands   bool
	ScopeBrandCodes  []string
	ScopeAllChannels bool
	ScopeAppIDs      []string
}

// applyDeviceFilter 把筛选条件应用到查询（不含分页），供列表与导出共用。
func applyDeviceFilter(q *gorm.DB, f DeviceFilter) *gorm.DB {
	if f.ApplicationID != "" {
		q = q.Where("application_id = ?", f.ApplicationID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	if f.ScopeRestricted {
		if !f.ScopeAllBrands {
			if len(f.ScopeBrandCodes) == 0 {
				q = q.Where("1 = 0")
			} else {
				q = q.Where("brand_code IN ?", f.ScopeBrandCodes)
			}
		}
		if !f.ScopeAllChannels {
			if len(f.ScopeAppIDs) == 0 {
				q = q.Where("1 = 0")
			} else {
				q = q.Where("application_id IN ?", f.ScopeAppIDs)
			}
		}
	}
	return q
}

// ListChannelDevices 分页/筛选设备列表（仿 ChannelFilter 用法：Count + Offset/Limit）。
// 排序 created_at DESC, id DESC（最新上报在前）。
func (r *Repo) ListChannelDevices(ctx context.Context, f DeviceFilter) ([]model.ChannelDevice, int64, error) {
	q := applyDeviceFilter(r.db.WithContext(ctx).Model(&model.ChannelDevice{}), f)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计设备总数失败: %w", err)
	}

	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 200 {
		f.PageSize = 50
	}
	var list []model.ChannelDevice
	if err := q.
		Order("created_at desc, id desc").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询设备列表失败: %w", err)
	}
	return list, total, nil
}

// deviceExportColumns 导出 CSV 用的固定选列顺序（与 ExportDeviceRows 的 Scan 顺序一一对应）。
const deviceExportColumns = "device_name, gaid, adid, oaid, app_name, pal_code, application_id, brand_code, created_at"

// ExportDeviceRows 按筛选条件返回设备导出游标（*sql.Rows，Order id ASC，不分页），
// 供 service 层流式生成 CSV，避免整表加载进内存。调用方负责 Close()。
func (r *Repo) ExportDeviceRows(ctx context.Context, f DeviceFilter) (*sql.Rows, error) {
	q := applyDeviceFilter(r.db.WithContext(ctx).Model(&model.ChannelDevice{}), f)
	rows, err := q.Select(deviceExportColumns).Order("id asc").Rows()
	if err != nil {
		return nil, fmt.Errorf("查询设备导出游标失败: %w", err)
	}
	return rows, nil
}

// DevicesByIDs 按 id 列表取设备记录（按 id 升序），供「勾选导出」端点使用。
func (r *Repo) DevicesByIDs(ctx context.Context, ids []uint64) ([]model.ChannelDevice, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.ChannelDevice
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("id asc").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("按 id 查询设备失败: %w", err)
	}
	return list, nil
}
