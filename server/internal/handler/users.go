package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// ListUsers godoc
// @Summary  账号列表（admin-only；不含密码哈希；含 protected 标记永久 admin）
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
// @Summary  新增普通用户（admin-only；角色恒为 user，不接受创建 admin）
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

// ResetUserPassword godoc
// @Summary  重置普通用户密码（admin-only；不能重置管理员密码）
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

// DeleteUser godoc
// @Summary  删除普通用户（admin-only；软删除；不能删除管理员）
// @Tags     users
// @Produce  json
// @Param    id   path      int  true  "账号 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id} [delete]
func (h *Handler) DeleteUser(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.DeleteUser(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"deleted": true})
}
