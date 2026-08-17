// Package handler 是 HTTP 层（Echo）。保持「薄」：解析请求 → 调 service → 统一封装响应。
package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/config"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/service"
)

// Handler 持有依赖。
type Handler struct {
	cfg     *config.Config
	svc     *service.Service
	authMgr *auth.Manager
	repo    *repo.Repo
	// rbac 按用户 ID 解析角色 + 权限集（30s TTL 进程内缓存），供路由中间件与
	// 登录/刷新/me 响应组装 role+perms 复用（docs/admin/10-rbac.md）。
	rbac *auth.RBAC
	// trustedProxies 用于从 X-Forwarded-For 安全提取真实客户端 IP（上架包网关判定用）。
	trustedProxies *httpx.TrustedProxies
}

// New 创建 Handler。
func New(cfg *config.Config, svc *service.Service, authMgr *auth.Manager, r *repo.Repo) *Handler {
	return &Handler{
		cfg:            cfg,
		svc:            svc,
		authMgr:        authMgr,
		repo:           r,
		rbac:           auth.NewRBAC(r),
		trustedProxies: httpx.NewTrustedProxies(cfg.TrustedProxyCIDRs),
	}
}

// currentUserID 从鉴权上下文取当前用户 ID；未登录/无 claims 时返回 0。
func currentUserID(c echo.Context) uint64 {
	if claims := auth.FromContext(c); claims != nil {
		return claims.UserID
	}
	return 0
}

// currentUsername 从鉴权上下文取当前用户名；未登录/无 claims 时返回空串。
func currentUsername(c echo.Context) string {
	if claims := auth.FromContext(c); claims != nil {
		return claims.Username
	}
	return ""
}

// fail 把 service 错误映射为统一响应。
func fail(c echo.Context, err error) error {
	e := service.AsError(err)
	return httpx.Fail(c, e.Code, e.Message)
}

// paramID 解析路径参数 :id。
func paramID(c echo.Context) (uint64, error) {
	return strconv.ParseUint(c.Param("id"), 10, 64)
}
