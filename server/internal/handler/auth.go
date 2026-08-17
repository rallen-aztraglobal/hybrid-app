package handler

import (
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/model"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// roleSummary 是登录/刷新/me 响应里的角色摘要（docs/admin/10-rbac.md）。
type roleSummary struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

// scopeSummary 是登录/刷新/me 响应里的数据范围摘要（docs/admin/10-rbac.md「数据权限」节，
// 字段名与前端约定的形状逐字一致：{allBrands, brands, allChannels, channelIds}）。
type scopeSummary struct {
	AllBrands   bool     `json:"allBrands"`
	Brands      []string `json:"brands"`
	AllChannels bool     `json:"allChannels"`
	ChannelIDs  []uint64 `json:"channelIds"`
}

type tokenResp struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int          `json:"expiresIn"`
	Username     string       `json:"username"`
	Role         roleSummary  `json:"role"`
	Perms        []string     `json:"perms"` // 超级管理员返回完整 catalog code 列表
	Scope        scopeSummary `json:"scope"`
}

// authInfo 把当前用户的角色摘要 + 权限集 + 数据范围组装成响应片段，Login/Refresh/Me 共用。
func (h *Handler) authInfo(c echo.Context, u *model.AdminUser) (roleSummary, []string, scopeSummary, error) {
	ctx := c.Request().Context()
	info, permSet, err := h.rbac.Resolve(ctx, u.ID)
	if err != nil {
		return roleSummary{}, nil, scopeSummary{}, err
	}
	perms := make([]string, 0, len(permSet))
	for code := range permSet {
		perms = append(perms, code)
	}
	sort.Strings(perms)

	scope, err := h.rbac.EffectiveScope(ctx, u.ID)
	if err != nil {
		return roleSummary{}, nil, scopeSummary{}, err
	}
	ss := scopeSummary{
		AllBrands: scope.AllBrands, Brands: scope.BrandCodeList(),
		AllChannels: scope.AllChannels, ChannelIDs: scope.ChannelIDList(),
	}
	return roleSummary{ID: info.ID, Name: info.Name, Builtin: info.Builtin}, perms, ss, nil
}

// Login godoc
// @Summary  账号密码登录
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      loginReq  true  "账号密码"
// @Success  200   {object}  httpx.Envelope{data=tokenResp}
// @Router   /api/auth/login [post]
func (h *Handler) Login(c echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	u, err := h.svc.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return fail(c, err)
	}
	access, refresh, err := h.authMgr.Issue(u)
	if err != nil {
		return fail(c, err)
	}
	role, perms, scope, err := h.authInfo(c, u)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, tokenResp{
		AccessToken: access, RefreshToken: refresh, ExpiresIn: h.authMgr.AccessTTLSeconds(),
		Username: u.Username, Role: role, Perms: perms, Scope: scope,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

// Refresh godoc
// @Summary  刷新 access token
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body  body      refreshReq  true  "refresh token"
// @Success  200   {object}  httpx.Envelope{data=tokenResp}
// @Router   /api/auth/refresh [post]
func (h *Handler) Refresh(c echo.Context) error {
	var req refreshReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	claims, err := h.authMgr.Parse(req.RefreshToken)
	if err != nil || claims.Type != auth.TokenRefresh {
		return httpx.Fail(c, http.StatusUnauthorized, "refresh token 无效")
	}
	u, err := h.repo.GetUserByUsername(c.Request().Context(), claims.Username)
	if err != nil {
		return fail(c, err)
	}
	access, refresh, err := h.authMgr.Issue(u)
	if err != nil {
		return fail(c, err)
	}
	role, perms, scope, err := h.authInfo(c, u)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, tokenResp{
		AccessToken: access, RefreshToken: refresh, ExpiresIn: h.authMgr.AccessTTLSeconds(),
		Username: u.Username, Role: role, Perms: perms, Scope: scope,
	})
}

type meResp struct {
	Username string       `json:"username"`
	Role     roleSummary  `json:"role"`
	Perms    []string     `json:"perms"`
	Scope    scopeSummary `json:"scope"`
}

// Me godoc
// @Summary  当前登录用户的角色与权限(前端启动时拉新，角色被改后无需重登即可生效)
// @Tags     auth
// @Produce  json
// @Success  200  {object}  httpx.Envelope{data=meResp}
// @Security BearerAuth
// @Router   /api/auth/me [get]
func (h *Handler) Me(c echo.Context) error {
	claims := auth.FromContext(c)
	if claims == nil {
		return httpx.Fail(c, http.StatusUnauthorized, "未鉴权")
	}
	u, err := h.repo.GetUserByID(c.Request().Context(), claims.UserID)
	if err != nil {
		return httpx.Fail(c, http.StatusUnauthorized, "账号不存在或已被删除")
	}
	role, perms, scope, err := h.authInfo(c, u)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, meResp{Username: u.Username, Role: role, Perms: perms, Scope: scope})
}
