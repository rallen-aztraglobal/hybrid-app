package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/perm"
	"github.com/hybrid-app/server/internal/repo"
)

// RunnerUsername 是构建机静态 token 注入的机器身份用户名（仅展示用，见 Middleware）。
// 该身份不落库、不挂角色；判定机器身份靠 Claims.Type == TokenRunner，不靠这个用户名
// （用户名是可注册的用户输入，不可作为身份边界，见 B1 修复）。
const RunnerUsername = "runner"

// ErrRoleMissing 表示账号存在但其 role_id 未指向任何有效角色（脏数据/角色被误删）。
// 与 repo.ErrNotFound（账号本身不存在）区分，RequirePerm 据此返回 403 而非 401，
// 避免把「账号不存在」这个误导性文案扣在一个真实存在的账号上（M2）。
var ErrRoleMissing = errors.New("账号未挂有效角色")

// PermCacheTTL 是权限集进程内缓存的 TTL（契约：30s，见 docs/admin/10-rbac.md）。
const PermCacheTTL = 30 * time.Second

// RoleInfo 是解析出的角色摘要，供登录/刷新/me 响应、中间件、service 层最小权限校验统一复用。
type RoleInfo struct {
	ID      uint64
	Name    string
	Builtin bool
}

type permCacheEntry struct {
	role  RoleInfo
	perms map[string]bool
	at    time.Time
}

// RBAC 按用户 ID 解析「角色 + 权限集 + 数据范围」，带 30s TTL 进程内缓存（角色/用户写操作后
// 调用 Invalidate 使其失效）。用户不存在（已被删除）时 Resolve 返回 repo.ErrNotFound，
// 天然实现「删号即失效」。
type RBAC struct {
	repo *repo.Repo

	mu         sync.RWMutex
	cache      map[uint64]permCacheEntry
	scopeCache map[uint64]scopeCacheEntry // 数据范围缓存（见 scope.go），与 cache 各自独立但同一 TTL 节奏
}

// NewRBAC 创建 RBAC 解析器。
func NewRBAC(r *repo.Repo) *RBAC {
	return &RBAC{repo: r, cache: map[uint64]permCacheEntry{}, scopeCache: map[uint64]scopeCacheEntry{}}
}

// RolePermCodes 返回某角色的权限 code 切片：builtin 角色恒为 perm.AllCodes()（不依赖
// role_permission 落库），非 builtin 角色从 role_permission 表读取。
//
// 这是「角色 → 权限集」判定的唯一实现，ResolveFresh（用户 → 角色 → 权限集）、
// service.roleView（角色详情展示）、service 侧最小权限校验（B2）都基于它，避免同一条
// 「builtin ? 全量 : 落库」判定分散拷贝三处、随时间漂移出不一致的口径。
func (rb *RBAC) RolePermCodes(ctx context.Context, role *model.Role) ([]string, error) {
	if role.Builtin {
		return perm.AllCodes(), nil
	}
	return rb.repo.RolePermCodes(ctx, role.ID)
}

// ResolveFresh 直接查库解析用户的角色摘要与权限集，不经过缓存、也不写入缓存。
// 供 service 层「现查现算」的最小权限校验复用（B2：角色/用户管理写操作低频，不需要也不应该
// 信任 30s 缓存），以及 Resolve 缓存未命中时的实际获取逻辑——两处共享同一套算法。
func (rb *RBAC) ResolveFresh(ctx context.Context, userID uint64) (RoleInfo, map[string]bool, error) {
	u, err := rb.repo.GetUserByID(ctx, userID)
	if err != nil {
		return RoleInfo{}, nil, err
	}
	role, err := rb.repo.GetRoleByID(ctx, u.RoleID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return RoleInfo{}, nil, ErrRoleMissing
		}
		return RoleInfo{}, nil, err
	}
	codes, err := rb.RolePermCodes(ctx, role)
	if err != nil {
		return RoleInfo{}, nil, err
	}
	permSet := make(map[string]bool, len(codes))
	for _, c := range codes {
		permSet[c] = true
	}
	return RoleInfo{ID: role.ID, Name: role.Name, Builtin: role.Builtin}, permSet, nil
}

// Resolve 返回用户的角色摘要与权限 code 集合（map，便于 O(1) 判定），命中 30s 缓存则不查库。
func (rb *RBAC) Resolve(ctx context.Context, userID uint64) (RoleInfo, map[string]bool, error) {
	rb.mu.RLock()
	if e, ok := rb.cache[userID]; ok && time.Since(e.at) < PermCacheTTL {
		rb.mu.RUnlock()
		return e.role, e.perms, nil
	}
	rb.mu.RUnlock()

	info, permSet, err := rb.ResolveFresh(ctx, userID)
	if err != nil {
		return RoleInfo{}, nil, err
	}

	rb.mu.Lock()
	rb.cache[userID] = permCacheEntry{role: info, perms: permSet, at: time.Now()}
	rb.mu.Unlock()
	return info, permSet, nil
}

// Invalidate 清空全部缓存（权限集 + 数据范围）。角色权限/数据范围、用户角色发生写操作后调用；
// 账号量级小，全量失效即可保证正确性，无需按用户精确失效。
func (rb *RBAC) Invalidate() {
	rb.mu.Lock()
	rb.cache = map[uint64]permCacheEntry{}
	rb.scopeCache = map[uint64]scopeCacheEntry{}
	rb.mu.Unlock()
}

// RequirePerm 返回一个中间件：any-of 语义，命中 codes 中任一权限点即放行。
//
// runner 静态 token 身份（Claims.Type == TokenRunner，只有 Middleware 校验过静态令牌后才会
// 注入这个 type，普通签发路径不可能产生，故不可伪造）不查库、不挂角色，只隐式持有
// perm.RunnerPerm：仅当 codes 含 perm.RunnerPerm 时放行。这不再局限于最初的 4 条机器接口——
// 构建机 claim 任务后还要拉 GET /build/manifest、/build/records* 与 GET /channels/:id/res.zip
// 才能渲染 flavor 资源，这些读接口的路由映射里也 any-of 了 perm.RunnerPerm（见 routes.go）；
// 除此之外的接口一律拒绝——机器身份不是可以碰任意写接口的泛化角色。判定字段刻意不用
// Username：Username 是用户可控输入（注册时校验，见 service.CreateUser 的保留用户名拦截），
// 不能作为身份边界。
//
// 普通用户按 Claims.UserID 经 Resolve 加载权限集；用户不存在（已被删除）返回 401；
// 用户存在但角色缺失（脏数据）返回 403 并给出明确文案（M2：不再与「用户不存在」混为一谈误导排查）。
func (rb *RBAC) RequirePerm(codes ...string) echo.MiddlewareFunc {
	need := make(map[string]bool, len(codes))
	for _, c := range codes {
		need[c] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := FromContext(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "未鉴权")
			}
			if claims.Type == TokenRunner {
				if need[perm.RunnerPerm] {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "权限不足")
			}
			_, perms, err := rb.Resolve(c.Request().Context(), claims.UserID)
			if err != nil {
				if errors.Is(err, ErrRoleMissing) {
					return echo.NewHTTPError(http.StatusForbidden, "账号未挂角色或角色已被删除，请联系管理员处理")
				}
				if errors.Is(err, repo.ErrNotFound) {
					return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在或已被删除")
				}
				return echo.NewHTTPError(http.StatusInternalServerError, "权限解析失败")
			}
			for code := range need {
				if perms[code] {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "权限不足")
		}
	}
}

// RequireActiveAccount 是比 RequirePerm 更轻量的中间件：不要求任何具体权限点，只确认账号仍存在。
//
// 用于「登录即可」类端点（/auth/me、/perms/catalog、/brands、/stores 的 GET）——这几条不进
// RequirePerm，天然不会触发 Resolve 的 401 判定；若不额外校验，被删账号的旧 access token
// 会在其自然过期前一直能读这几条（应修4）。
//
// runner 静态 token 对这几条一律拒绝（403）：/perms/catalog、/auth/me 对机器身份没有意义，
// /brands、/stores 是人工维护渠道用的基础数据，构建机拉取全量配置走的是 GET /build/manifest
// （已在 RequirePerm 放行），不需要也不应该额外开这几条口子。
func (rb *RBAC) RequireActiveAccount() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := FromContext(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "未鉴权")
			}
			if claims.Type == TokenRunner {
				return echo.NewHTTPError(http.StatusForbidden, "权限不足")
			}
			if _, err := rb.repo.GetUserByID(c.Request().Context(), claims.UserID); err != nil {
				if errors.Is(err, repo.ErrNotFound) {
					return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在或已被删除")
				}
				return echo.NewHTTPError(http.StatusInternalServerError, "账号校验失败")
			}
			return next(c)
		}
	}
}
