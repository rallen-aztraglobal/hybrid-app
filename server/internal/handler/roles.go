// Package handler — 角色管理 + 权限点 catalog HTTP 端点（docs/admin/10-rbac.md）。
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/perm"
	"github.com/hybrid-app/server/internal/service"
)

// PermsCatalog godoc
// @Summary  权限点清单(登录即可，供前端渲染角色勾选树)
// @Tags     roles
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/perms/catalog [get]
func (h *Handler) PermsCatalog(c echo.Context) error {
	return httpx.OK(c, perm.Catalog())
}

// ListRoles godoc
// @Summary  角色列表
// @Tags     roles
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/roles [get]
func (h *Handler) ListRoles(c echo.Context) error {
	list, err := h.svc.ListRoles(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// CreateRole godoc
// @Summary  新增角色
// @Tags     roles
// @Accept   json
// @Produce  json
// @Param    body  body      service.RoleInput  true  "角色字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/roles [post]
func (h *Handler) CreateRole(c echo.Context) error {
	var in service.RoleInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	role, err := h.svc.CreateRole(c.Request().Context(), currentUserID(c), in)
	if err != nil {
		return fail(c, err)
	}
	h.rbac.Invalidate()
	return httpx.Created(c, role)
}

// UpdateRole godoc
// @Summary  修改角色(builtin 不可改)
// @Tags     roles
// @Accept   json
// @Produce  json
// @Param    id    path      int                 true  "角色 ID"
// @Param    body  body      service.RoleInput   true  "角色字段"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/roles/{id} [put]
func (h *Handler) UpdateRole(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.RoleInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	role, err := h.svc.UpdateRole(c.Request().Context(), currentUserID(c), id, in)
	if err != nil {
		return fail(c, err)
	}
	h.rbac.Invalidate()
	return httpx.OK(c, role)
}

// DeleteRole godoc
// @Summary  删除角色(builtin 不可删；仍有用户挂靠返回 409；非超管仅能删权限集 ⊆ 自身的角色)
// @Tags     roles
// @Produce  json
// @Param    id  path  int  true  "角色 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/roles/{id} [delete]
func (h *Handler) DeleteRole(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.DeleteRole(c.Request().Context(), currentUserID(c), id); err != nil {
		return fail(c, err)
	}
	h.rbac.Invalidate()
	return httpx.OK(c, map[string]any{"deleted": true})
}
