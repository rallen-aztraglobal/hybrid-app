package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// ListStores godoc
// @Summary  应用商店列表（含停用，按 sort 升序）
// @Tags     stores
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/stores [get]
func (h *Handler) ListStores(c echo.Context) error {
	list, err := h.svc.ListStores(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// CreateStore godoc
// @Summary  新增应用商店（code 全局唯一，创建后不可改）
// @Tags     stores
// @Accept   json
// @Produce  json
// @Param    body  body      service.CreateStoreInput  true  "商店字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/stores [post]
func (h *Handler) CreateStore(c echo.Context) error {
	var in service.CreateStoreInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	st, err := h.svc.CreateStore(c.Request().Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, st)
}

// UpdateStore godoc
// @Summary  修改应用商店（仅 name/sort/status；code 不可改）
// @Tags     stores
// @Accept   json
// @Produce  json
// @Param    id    path      int                       true  "商店 ID"
// @Param    body  body      service.UpdateStoreInput  true  "可选更新字段"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/stores/{id} [put]
func (h *Handler) UpdateStore(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.UpdateStoreInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	st, err := h.svc.UpdateStore(c.Request().Context(), id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, st)
}

// DeleteStore godoc
// @Summary  删除应用商店（已被渠道引用则拒绝，返回 409，请改为停用）
// @Tags     stores
// @Produce  json
// @Param    id   path      int  true  "商店 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/stores/{id} [delete]
func (h *Handler) DeleteStore(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.DeleteStore(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"deleted": true})
}
