package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/httpx"
)

// ListBrands godoc
// @Summary  大渠道列表（含渠道计数、accent 色、默认域名）
// @Tags     brands
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/brands [get]
func (h *Handler) ListBrands(c echo.Context) error {
	views, err := h.svc.ListBrands(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, views)
}

// GetBrandDomains godoc
// @Summary  品牌默认域名清单
// @Tags     brands
// @Produce  json
// @Param    code  path      string  true  "品牌 code (ap/bp/gp)"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/brands/{code}/domains [get]
func (h *Handler) GetBrandDomains(c echo.Context) error {
	domains, err := h.svc.GetBrandDomains(c.Request().Context(), c.Param("code"))
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"domains": domains})
}

type setDomainsReq struct {
	Domains []domainutil.DomainInput `json:"domains"`
}

// SetBrandDomains godoc
// @Summary  更新品牌默认域名（主+最多3备用，强校验）
// @Tags     brands
// @Accept   json
// @Produce  json
// @Param    code  path      string         true  "品牌 code"
// @Param    body  body      setDomainsReq  true  "域名清单"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/brands/{code}/domains [put]
func (h *Handler) SetBrandDomains(c echo.Context) error {
	var req setDomainsReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.SetBrandDomains(c.Request().Context(), c.Param("code"), req.Domains)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}
