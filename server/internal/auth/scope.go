package auth

import (
	"context"
	"errors"
	"time"

	"github.com/hybrid-app/server/internal/repo"
)

// Scope 是角色的有效数据范围（RoleEffectiveScope 计算结果），回答「能对哪些品牌/渠道数据做事」
// ——正交于权限点回答的「能不能做这件事」（docs/admin/10-rbac.md「数据权限」节）。
//
// AllBrands/AllChannels=true 时对应集合字段无意义（可能为 nil）。ChannelIDs 在计算阶段就已经
// 与 Brands 求过交（品牌范围收窄后，原本勾选但已不在允许品牌内的渠道不会出现在这里），
// 调用方直接用 ChannelAllowed 判定即可，不需要自己再手动求一次交集。
type Scope struct {
	AllBrands   bool
	Brands      map[string]bool
	AllChannels bool
	ChannelIDs  map[uint64]bool
}

// FullScope 返回「不限」的数据范围（AllBrands=true, AllChannels=true）。
// builtin 角色、runner 机器身份、以及不涉及数据权限的调用点（如角色/用户/商店管理，
// 这些本就不受数据权限约束）都用它。
func FullScope() Scope { return Scope{AllBrands: true, AllChannels: true} }

// BrandAllowed 判断某品牌 code 是否在范围内。
func (s Scope) BrandAllowed(code string) bool {
	if s.AllBrands {
		return true
	}
	return s.Brands[code]
}

// ChannelAllowed 判断某渠道（属于 brandCode）是否在范围内：先过品牌关，再过渠道关。
func (s Scope) ChannelAllowed(brandCode string, channelID uint64) bool {
	if !s.BrandAllowed(brandCode) {
		return false
	}
	if s.AllChannels {
		return true
	}
	return s.ChannelIDs[channelID]
}

// BrandCodeList 返回 Brands 集合的切片形式（AllBrands=true 时通常为空，调用方应先查 AllBrands）。
func (s Scope) BrandCodeList() []string {
	out := make([]string, 0, len(s.Brands))
	for code := range s.Brands {
		out = append(out, code)
	}
	return out
}

// ChannelIDList 同上，渠道 id 版本。
func (s Scope) ChannelIDList() []uint64 {
	out := make([]uint64, 0, len(s.ChannelIDs))
	for id := range s.ChannelIDs {
		out = append(out, id)
	}
	return out
}

// SubsetOf 判断 s 是否是 other 的子集：品牌与渠道两个维度都要满足。
// other 某维度是「全部」，则 s 该维度天然是子集（无论 s 是否也是全部）；
// other 该维度不是「全部」而 s 是「全部」，则 s 一定不是子集（全部 ⊄ 有限集合）；
// 两边都不是「全部」时，逐项判断 s 的集合是否 ⊆ other 的集合。
// 供 service 层最小权限校验复用：非超管建/改角色时，数据范围必须 ⊆ 调用者自身范围。
func (s Scope) SubsetOf(other Scope) bool {
	if !other.AllBrands {
		if s.AllBrands {
			return false
		}
		for code := range s.Brands {
			if !other.Brands[code] {
				return false
			}
		}
	}
	if !other.AllChannels {
		if s.AllChannels {
			return false
		}
		for id := range s.ChannelIDs {
			if !other.ChannelIDs[id] {
				return false
			}
		}
	}
	return true
}

// scopeCacheEntry 缓存某用户的有效数据范围（与权限集缓存同一个 30s TTL 节奏，见 rbac.go）。
type scopeCacheEntry struct {
	scope Scope
	at    time.Time
}

// EffectiveScope 返回用户对应角色的有效数据范围，带 30s TTL 进程内缓存
// （与 Resolve 的权限集缓存是两张独立的表，但 TTL/失效时机一致：Invalidate() 会同时清空两者）。
func (rb *RBAC) EffectiveScope(ctx context.Context, userID uint64) (Scope, error) {
	rb.mu.RLock()
	if e, ok := rb.scopeCache[userID]; ok && time.Since(e.at) < PermCacheTTL {
		rb.mu.RUnlock()
		return e.scope, nil
	}
	rb.mu.RUnlock()

	u, err := rb.repo.GetUserByID(ctx, userID)
	if err != nil {
		return Scope{}, err
	}
	scope, err := rb.RoleEffectiveScope(ctx, u.RoleID)
	if err != nil {
		return Scope{}, err
	}

	rb.mu.Lock()
	rb.scopeCache[userID] = scopeCacheEntry{scope: scope, at: time.Now()}
	rb.mu.Unlock()
	return scope, nil
}

// RoleEffectiveScope 计算某角色（不经过用户，也不经过缓存）的有效数据范围。这是「角色 → 数据范围」
// 判定的唯一实现，EffectiveScope（用户路径）与 service 层最小权限校验（角色/用户管理时比较
// 「目标角色范围 ⊆ 调用者范围」）都基于它，避免像最初的权限集那样散成多处再回头收敛（见
// docs/admin/10-rbac.md「数据权限」节）：
//  1. 角色 builtin=true（超级管理员）→ 全部品牌 + 全部渠道，忽略其余字段；
//  2. scope_all_brands=true → 全部品牌；否则 = role_brand 列表；
//  3. scope_all_channels=true → 上述品牌下的全部渠道；否则 = role_channel 里且所属品牌仍在
//     允许品牌内的渠道（与品牌范围求交，品牌收窄后原勾选的渠道自动失效，不需要额外清理）。
func (rb *RBAC) RoleEffectiveScope(ctx context.Context, roleID uint64) (Scope, error) {
	role, err := rb.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return Scope{}, ErrRoleMissing
		}
		return Scope{}, err
	}
	if role.Builtin {
		return FullScope(), nil
	}

	scope := Scope{AllBrands: role.ScopeAllBrands, AllChannels: role.ScopeAllChannels}
	if !role.ScopeAllBrands {
		codes, err := rb.repo.RoleBrandCodes(ctx, roleID)
		if err != nil {
			return Scope{}, err
		}
		scope.Brands = make(map[string]bool, len(codes))
		for _, c := range codes {
			scope.Brands[c] = true
		}
	}
	if !role.ScopeAllChannels {
		rows, err := rb.repo.RoleChannelBrandCodes(ctx, roleID)
		if err != nil {
			return Scope{}, err
		}
		ids := make(map[uint64]bool, len(rows))
		for _, row := range rows {
			// 与品牌范围求交：品牌范围收窄后，原本勾选但已不在允许品牌内的渠道在这里被排除。
			if scope.BrandAllowed(row.BrandCode) {
				ids[row.ChannelID] = true
			}
		}
		scope.ChannelIDs = ids
	}
	return scope, nil
}
