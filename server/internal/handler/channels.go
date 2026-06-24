package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/service"
)

// ListChannels godoc
// @Summary  渠道列表（分页/搜索/筛选）
// @Tags     channels
// @Produce  json
// @Param    brand   query     string  false  "品牌 code"
// @Param    status  query     string  false  "状态 enabled/disabled/archived"
// @Param    q       query     string  false  "关键词（flavor/包名/应用名/palcode）"
// @Param    page    query     int     false  "页码，默认 1"
// @Param    pageSize query    int     false  "每页，默认 50"
// @Success  200     {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels [get]
func (h *Handler) ListChannels(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	list, total, err := h.svc.ListChannels(c.Request().Context(), repo.ChannelFilter{
		BrandCode: c.QueryParam("brand"),
		Status:    c.QueryParam("status"),
		Q:         c.QueryParam("q"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"items": list, "total": total})
}

// GetChannel godoc
// @Summary  渠道详情
// @Tags     channels
// @Produce  json
// @Param    id   path      int  true  "渠道 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id} [get]
func (h *Handler) GetChannel(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	ch, err := h.svc.GetChannel(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, ch)
}

// CreateChannel godoc
// @Summary  新增渠道（applicationId 派生自品牌包前缀+flavor；唯一性校验 applicationId 与 (brand,flavor)）
// @Tags     channels
// @Accept   json
// @Produce  json
// @Param    body  body      service.CreateChannelInput  true  "渠道字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels [post]
func (h *Handler) CreateChannel(c echo.Context) error {
	var in service.CreateChannelInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	ch, err := h.svc.CreateChannel(c.Request().Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, ch)
}

// UpdateChannel godoc
// @Summary  修改渠道
// @Tags     channels
// @Accept   json
// @Produce  json
// @Param    id    path      int                         true  "渠道 ID"
// @Param    body  body      service.UpdateChannelInput  true  "可选更新字段"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id} [put]
func (h *Handler) UpdateChannel(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var in service.UpdateChannelInput
	if err := c.Bind(&in); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	ch, err := h.svc.UpdateChannel(c.Request().Context(), id, in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, ch)
}

// DeleteChannel godoc
// @Summary  软删除渠道（置 archived）
// @Tags     channels
// @Produce  json
// @Param    id   path      int  true  "渠道 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id} [delete]
func (h *Handler) DeleteChannel(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.ArchiveChannel(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"archived": true})
}

// LatestApk godoc
// @Summary  渠道最新包：按该 flavor 取最近一次成功构建的 APK（ADR-0008 卡片「下载最新包」）
// @Tags     channels
// @Produce  json
// @Param    id   path      int  true  "渠道 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id}/latest-apk [get]
func (h *Handler) LatestApk(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	a, err := h.svc.LatestApkForChannel(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	if a == nil {
		return httpx.Fail(c, http.StatusNotFound, "该渠道暂无成功构建的 APK")
	}
	return httpx.OK(c, a)
}

type setChannelDomainsReq struct {
	InheritBrand bool                     `json:"inheritBrand"`
	Domains      []domainutil.DomainInput `json:"domains"`
}

// SetChannelDomains godoc
// @Summary  设置渠道域名覆盖 / 切回继承品牌
// @Tags     channels
// @Accept   json
// @Produce  json
// @Param    id    path      int                   true  "渠道 ID"
// @Param    body  body      setChannelDomainsReq  true  "inheritBrand=true 表示切回继承"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id}/domains [put]
func (h *Handler) SetChannelDomains(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req setChannelDomainsReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.SetChannelDomains(c.Request().Context(), id, req.InheritBrand, req.Domains)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}
