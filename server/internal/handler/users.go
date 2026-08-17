// Package handler — 用户管理 HTTP 端点（docs/admin/10-rbac.md）。
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// ListUsers godoc
// @Summary  账号列表
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

type createUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   uint64 `json:"roleId"`
}

// CreateUser godoc
// @Summary  新增账号
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    body  body      createUserReq  true  "账号字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users [post]
func (h *Handler) CreateUser(c echo.Context) error {
	var req createUserReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	u, err := h.svc.CreateUser(c.Request().Context(), currentUserID(c), service.CreateUserInput{
		Username: req.Username, Password: req.Password, RoleID: req.RoleID,
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, u)
}

type updateUserReq struct {
	RoleID uint64 `json:"roleId"`
}

// UpdateUser godoc
// @Summary  修改账号角色
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id    path      int            true  "账号 ID"
// @Param    body  body      updateUserReq  true  "新角色 ID"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id} [put]
func (h *Handler) UpdateUser(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req updateUserReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	u, err := h.svc.UpdateUser(c.Request().Context(), currentUserID(c), id, service.UpdateUserInput{RoleID: req.RoleID})
	if err != nil {
		return fail(c, err)
	}
	h.rbac.Invalidate()
	return httpx.OK(c, u)
}

type resetPasswordReq struct {
	Password string `json:"password"`
}

// ResetUserPassword godoc
// @Summary  重置账号密码
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    id    path      int                true  "账号 ID"
// @Param    body  body      resetPasswordReq   true  "新密码"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id}/reset-password [post]
func (h *Handler) ResetUserPassword(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req resetPasswordReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	if err := h.svc.ResetUserPassword(c.Request().Context(), currentUserID(c), id, req.Password); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"reset": true})
}

// DeleteUser godoc
// @Summary  删除账号(不能删除自己；不能删除最后一个超级管理员)
// @Tags     users
// @Produce  json
// @Param    id  path  int  true  "账号 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/users/{id} [delete]
func (h *Handler) DeleteUser(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.DeleteUser(c.Request().Context(), currentUserID(c), id); err != nil {
		return fail(c, err)
	}
	h.rbac.Invalidate()
	return httpx.OK(c, map[string]any{"deleted": true})
}
