// Package handler 是 HTTP 层（Echo）。保持「薄」：解析请求 → 调 service → 统一封装响应。
package handler

import (
	"net/http"
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
		trustedProxies: httpx.NewTrustedProxies(cfg.TrustedProxyCIDRs),
	}
}

// RequireEnabled 校验当前 access token 对应账号仍处于启用状态。
//
// 背景：JWT access token 一旦签发，在自然过期（ADR-0008）前仅凭 auth.Manager.Middleware
// 的签名校验即可通过——管理员把某账号禁用后，该账号已持有的 token 并不会因此失效。
// 这里对每个受保护请求重新查一次库，把「禁用」做成立即生效的服务端强制检查，
// 而不是等 token 自然过期（详见用户管理文档「禁用账号行为」一节）。
// 构建机静态令牌（RunnerToken）注入的机器身份不对应 admin_user 记录（UserID 恒为 0），跳过此检查。
func (h *Handler) RequireEnabled(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims := auth.FromContext(c)
		if claims == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "未鉴权")
		}
		if claims.UserID == 0 {
			return next(c)
		}
		u, err := h.repo.GetUserByID(c.Request().Context(), claims.UserID)
		if err != nil || !u.Enabled {
			return echo.NewHTTPError(http.StatusUnauthorized, "账号已被禁用")
		}
		return next(c)
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
