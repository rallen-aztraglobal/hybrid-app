// Package service — 数据权限（角色可见的品牌/渠道范围）在查询层/写操作前的落地
// （docs/admin/10-rbac.md「数据权限」节，强制点清单）。
package service

import (
	"context"
	"fmt"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/repo"
)

// ApplyChannelScope 把 scope 写入 f 的数据权限过滤字段。scope 是「全部」（AllBrands &&
// AllChannels）时不改动 f、保持零值语义（= 不限），兼容不涉及数据权限的其它调用路径。
// 导出给 handler 包复用（GET /channels 在 handler 层直接构造 repo.ChannelFilter）。
func ApplyChannelScope(f *repo.ChannelFilter, scope auth.Scope) {
	if scope.AllBrands && scope.AllChannels {
		return
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = scope.AllBrands
	f.ScopeBrandCodes = scope.BrandCodeList()
	f.ScopeAllChannels = scope.AllChannels
	f.ScopeChannelIDs = scope.ChannelIDList()
}

// applyBuildRecordScope 同 ApplyChannelScope，用于构建记录（只有品牌维度）。
func applyBuildRecordScope(f *repo.BuildRecordFilter, scope auth.Scope) {
	if scope.AllBrands {
		return
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = false
	f.ScopeBrandCodes = scope.BrandCodeList()
}

// applyListingScope 同 ApplyChannelScope，用于上架包（只有品牌维度，见 repo.ListingFilter 注释）。
func applyListingScope(f *repo.ListingFilter, scope auth.Scope) {
	if scope.AllBrands {
		return
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = false
	f.ScopeBrandCodes = scope.BrandCodeList()
}

// applyCampaignScope 同 ApplyChannelScope，用于推送活动列表（ALL-match 安全边界，见
// repo.CampaignFilter 注释；与 Brand 字段的 ANY-match 用户筛选语义不同）。
func applyCampaignScope(f *repo.CampaignFilter, scope auth.Scope) {
	if scope.AllBrands {
		return
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = false
	f.ScopeBrandCodes = scope.BrandCodeList()
}

// applyDeviceScope 同 ApplyChannelScope，用于设备列表/导出。渠道维度需要把 scope.ChannelIDs
// 解析成 application_id 集合（channel_device 表按 application_id 存储，不是 channel_id）。
func (s *Service) applyDeviceScope(ctx context.Context, f *repo.DeviceFilter, scope auth.Scope) error {
	if scope.AllBrands && scope.AllChannels {
		return nil
	}
	f.ScopeRestricted = true
	f.ScopeAllBrands = scope.AllBrands
	f.ScopeBrandCodes = scope.BrandCodeList()
	f.ScopeAllChannels = scope.AllChannels
	if !scope.AllChannels {
		appIDs, err := s.repo.ApplicationIDsByChannelIDs(ctx, scope.ChannelIDList())
		if err != nil {
			return err
		}
		f.ScopeAppIDs = appIDs
	}
	return nil
}

// assertAppIDsInScope 校验一批 applicationId 全部在 scope 范围内；scope 全量时直接放行。
// 有一个越界即整体拒绝（推送:"活动的 brand 在范围内才能建/改/发"，与 build jobs 的
// "有一个越界整单拒绝" 同一口径）。appId 解析不出渠道（脏数据/已删）也视为越界，fail-closed。
func (s *Service) assertAppIDsInScope(ctx context.Context, scope auth.Scope, appIDs []string) error {
	if scope.AllBrands && scope.AllChannels {
		return nil
	}
	info, err := s.repo.ChannelBrandsByApplicationIDs(ctx, appIDs)
	if err != nil {
		return err
	}
	for _, id := range appIDs {
		row, ok := info[id]
		if !ok || !scope.ChannelAllowed(row.BrandCode, row.ChannelID) {
			return errForbidden(fmt.Sprintf("目标 %q 不在你的数据范围内", id))
		}
	}
	return nil
}

// filterAppIDsByScope 保留 appIDs 中「按数据权限可见」的那些；不报错，静默丢弃越界项
// （用于 GET /push/audience 这类预览类端点，见 docs/admin/10-rbac.md）。
func (s *Service) filterAppIDsByScope(ctx context.Context, scope auth.Scope, appIDs []string) []string {
	if scope.AllBrands && scope.AllChannels || len(appIDs) == 0 {
		return appIDs
	}
	info, err := s.repo.ChannelBrandsByApplicationIDs(ctx, appIDs)
	if err != nil {
		return nil // fail-closed：查询出错时不返回未经过滤的数据，宁可空
	}
	out := make([]string, 0, len(appIDs))
	for _, id := range appIDs {
		if row, ok := info[id]; ok && scope.ChannelAllowed(row.BrandCode, row.ChannelID) {
			out = append(out, id)
		}
	}
	return out
}
