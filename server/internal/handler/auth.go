package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/httpx"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	Role         string `json:"role"`
	Username     string `json:"username"`
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
	return httpx.OK(c, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: h.authMgr.AccessTTLSeconds(), Role: u.Role, Username: u.Username,
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
	return httpx.OK(c, tokenResp{
		AccessToken: access, RefreshToken: refresh,
		ExpiresIn: h.authMgr.AccessTTLSeconds(), Role: u.Role, Username: u.Username,
	})
}
