package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// ListUsers godoc
// @Summary  账号列表（admin-only；不含密码哈希）
// @Tags     users
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users [get]
func (h *Handler) ListUsers(c echo.Context) error {
	list, err := h.svc.ListUsers(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// CreateUser godoc
// @Summary  新增账号（admin-only；角色仅 admin/user）
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    body  body      service.CreateUserInput  true  "账号字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users [post]
func (h *Handler) CreateUser(c echo.Context) error {
	var in service.CreateUserInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	u, err := h.svc.CreateUser(c.Request().Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, u)
}

// UpdateUser godoc
// @Summary  修改账号角色/启用状态（admin-only；不能改自己、不能移除最后一个启用中的 admin）
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id    path      int                      true  "账号 ID"
// @Param    body  body      service.UpdateUserInput  true  "可选更新字段"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id} [put]
func (h *Handler) UpdateUser(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.UpdateUserInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	u, err := h.svc.UpdateUser(c.Request().Context(), currentUserID(c), id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, u)
}

// ResetUserPassword godoc
// @Summary  重置指定账号密码（admin-only；与登录同一套 bcrypt 哈希）
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id    path      int                         true  "账号 ID"
// @Param    body  body      service.ResetPasswordInput  true  "新密码"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id}/reset-password [post]
func (h *Handler) ResetUserPassword(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.ResetPasswordInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	if err := h.svc.ResetUserPassword(c.Request().Context(), id, in); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"reset": true})
}
