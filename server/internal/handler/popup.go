// Package handler — 马甲包弹窗模块 HTTP 端点（ADR-0019 / docs/admin/12-popup.md）。
package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

const popupEventsMaxBody = 256 << 10 // 100 条事件远小于此，防滥用

// ---------- 管理端 ----------

// ListPopupPositions godoc
// @Summary  弹窗位置开关列表（page:popups）
// @Tags     popup
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/positions [get]
func (h *Handler) ListPopupPositions(c echo.Context) error {
	list, err := h.svc.ListPopupPositions(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

type enabledReq struct {
	Enabled bool `json:"enabled"`
}

// SetPopupPosition godoc
// @Summary  开关弹窗位置（popup:edit + 全量数据范围，受限账号 403）
// @Tags     popup
// @Accept   json
// @Produce  json
// @Param    code  path  string      true  "位置码 P1~P8"
// @Param    body  body  enabledReq  true  "开关"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/positions/{code} [put]
func (h *Handler) SetPopupPosition(c echo.Context) error {
	var req enabledReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.SetPopupPosition(c.Request().Context(), scope, c.Param("code"), req.Enabled, currentUsername(c))
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// ListPopups godoc
// @Summary  弹窗列表（page:popups；按数据权限 ALL-match 过滤）
// @Tags     popup
// @Produce  json
// @Param    position  query  string  false  "位置码 P1~P8"
// @Param    brand     query  string  false  "品牌 code（含未限定品牌的弹窗）"
// @Param    status    query  string  false  "active|scheduled|ended|disabled"
// @Param    keyword   query  string  false  "名称关键字"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups [get]
func (h *Handler) ListPopups(c echo.Context) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	list, err := h.svc.ListPopups(c.Request().Context(), scope, service.PopupListQuery{
		Position: c.QueryParam("position"), Brand: c.QueryParam("brand"),
		Status: c.QueryParam("status"), Keyword: c.QueryParam("keyword"),
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, list)
}

// GetPopup godoc
// @Summary  弹窗详情（page:popups）
// @Tags     popup
// @Produce  json
// @Param    id  path  int  true  "弹窗 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/{id} [get]
func (h *Handler) GetPopup(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.GetPopup(c.Request().Context(), scope, id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// CreatePopup godoc
// @Summary  新建弹窗（popup:edit；保存时按位置能力矩阵归一化）
// @Tags     popup
// @Accept   json
// @Produce  json
// @Param    body  body  service.PopupInput  true  "弹窗内容"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups [post]
func (h *Handler) CreatePopup(c echo.Context) error {
	var in service.PopupInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.CreatePopup(c.Request().Context(), scope, in, currentUsername(c))
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, v)
}

// UpdatePopup godoc
// @Summary  全量更新弹窗（popup:edit；卡片按 id 原地合并，缺席的删除）
// @Tags     popup
// @Accept   json
// @Produce  json
// @Param    id    path  int                 true  "弹窗 ID"
// @Param    body  body  service.PopupInput  true  "弹窗内容"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/{id} [put]
func (h *Handler) UpdatePopup(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.PopupInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.UpdatePopup(c.Request().Context(), scope, id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// SetPopupEnabled godoc
// @Summary  弹窗行内快速开关（popup:edit）
// @Tags     popup
// @Accept   json
// @Produce  json
// @Param    id    path  int         true  "弹窗 ID"
// @Param    body  body  enabledReq  true  "开关"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/{id}/enabled [put]
func (h *Handler) SetPopupEnabled(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req enabledReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	v, err := h.svc.SetPopupEnabled(c.Request().Context(), scope, id, req.Enabled)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, v)
}

// DeletePopup godoc
// @Summary  软删弹窗（popup:edit；统计仍保留名称）
// @Tags     popup
// @Produce  json
// @Param    id  path  int  true  "弹窗 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/{id} [delete]
func (h *Handler) DeletePopup(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	if err := h.svc.DeletePopup(c.Request().Context(), scope, id); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"deleted": true})
}

// UploadPopupImage godoc
// @Summary  上传弹窗素材（popup:edit；png/jpeg/webp/gif，≤3MB，每次都是新文件名）
// @Tags     popup
// @Accept   multipart/form-data
// @Produce  json
// @Param    file  formData  file  true  "图片文件"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/upload-image [post]
func (h *Handler) UploadPopupImage(c echo.Context) error {
	// 请求体上限略大于 3MB 图片上限（含 multipart 开销），防止超大请求吃满内存/磁盘。
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 4<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 file 字段")
	}
	f, err := fh.Open()
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "打开上传文件失败")
	}
	defer f.Close()
	// 多读 1 字节用于判超限，避免把超大文件整个读进内存。
	data, err := io.ReadAll(io.LimitReader(f, 3<<20+1))
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "读取文件内容失败")
	}
	res, err := h.svc.UploadPopupImage(c.Request().Context(), data)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

// PopupStats godoc
// @Summary  弹窗统计（page:popups；受限账号只统计其范围内渠道包）
// @Tags     popup
// @Produce  json
// @Param    from      query  string  false  "起始日期 YYYY-MM-DD（含），默认近 7 天"
// @Param    to        query  string  false  "结束日期 YYYY-MM-DD（含），跨度 ≤ 92 天"
// @Param    brand     query  string  false  "品牌 code"
// @Param    appId     query  string  false  "渠道 applicationId"
// @Param    popupId   query  int     false  "弹窗 ID"
// @Param    position  query  string  false  "位置码"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/stats [get]
func (h *Handler) PopupStats(c echo.Context) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	q := service.PopupStatsQuery{
		From: c.QueryParam("from"), To: c.QueryParam("to"),
		Brand: c.QueryParam("brand"), AppID: c.QueryParam("appId"), Position: c.QueryParam("position"),
	}
	if raw := strings.TrimSpace(c.QueryParam("popupId")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return httpx.Fail(c, http.StatusBadRequest, "popupId 非法")
		}
		q.PopupID = id
	}
	res, err := h.svc.PopupStats(c.Request().Context(), scope, q)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

// PopupRuntimePreview godoc
// @Summary  运行时预览（page:popups）：返回与 App 端完全相同的 payload，country 用参数模拟
// @Tags     popup
// @Produce  json
// @Param    appId    query  string  true   "渠道 applicationId"
// @Param    country  query  string  false  "模拟国家码（ISO-3166 二位）"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/popups/runtime-preview [get]
func (h *Handler) PopupRuntimePreview(c echo.Context) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.PopupRuntimePreview(c.Request().Context(), scope, c.QueryParam("appId"), c.QueryParam("country"))
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

// ---------- App 端公开 ----------

// AppPopups godoc
// @Summary  弹窗运行时配置（App 公开消费，不缓存）
// @Description 按 appId 解析品牌 / 包定向，并按请求真实 IP 的国家做地区定向；版本 / 新老用户 / 生效时间由客户端过滤。
// @Description 裸 JSON（非 Envelope），Cache-Control: no-store（地区定向按 IP，不可被 CDN 共享缓存）。未知 / 已归档 appId → 404。
// @Tags     app
// @Produce  json
// @Param    appId  query  string  true  "渠道 applicationId"
// @Success  200  {object}  service.PopupAppConfig
// @Router   /api/app/popups [get]
func (h *Handler) AppPopups(c echo.Context) error {
	appID := c.QueryParam("appId")
	if appID == "" {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 appId 参数")
	}
	// 成功与失败（含 404）都不缓存。
	c.Response().Header().Set("Cache-Control", "no-store")
	cfg, err := h.svc.PopupConfigForIP(c.Request().Context(), appID, httpx.ClientIP(c.Request(), h.trustedProxies))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(http.StatusOK, cfg)
}

// AppPopupEvents godoc
// @Summary  弹窗埋点批量上报（App 公开，batchId 幂等，每批 ≤100 条）
// @Tags     app
// @Accept   json
// @Produce  json
// @Param    body  body  service.PopupEventsRequest  true  "批次"
// @Success  200   {object}  httpx.Envelope
// @Router   /api/app/popups/events [post]
func (h *Handler) AppPopupEvents(c echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, popupEventsMaxBody)
	var req service.PopupEventsRequest
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.IngestPopupEvents(c.Request().Context(), req)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}
