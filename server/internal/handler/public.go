package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
)

// AppConfig godoc
// @Summary  运行时配置（APK 公开消费，强缓存）— 返回该渠道最新域名清单 + 探测端点
// @Description APK 启动用不变的 applicationId（BuildConfig.APPLICATION_ID）拉最新域名（ADR-0002/0009）。可被 CDN 强缓存；后台域名变更后刷新快照。
// @Tags     app
// @Produce  json
// @Param    appId  query     string  true  "渠道 applicationId（解析键）"
// @Success  200      {object}  github_com_hybrid-app_server_internal_service.AppConfig
// @Router   /api/app/config [get]
func (h *Handler) AppConfig(c echo.Context) error {
	appID := c.QueryParam("appId")
	if appID == "" {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 appId 参数")
	}
	cfg, err := h.svc.AppConfigForAppId(c.Request().Context(), appID)
	if err != nil {
		return fail(c, err)
	}
	// 强缓存：APK 与 CDN 都可缓存 ttlSeconds。这是公开端点，直接返回裸 JSON（非 Envelope），
	// 与 docs/admin/01 §5.6 的响应示例一致，便于生成静态快照。
	c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cfg.TTLSeconds))
	return c.JSON(http.StatusOK, cfg)
}

// healthzResp 是 APK 探针校验「确实是我们站点」的约定 JSON（ADR-0003）。
type healthzResp struct {
	OK    bool   `json:"ok"`
	Brand string `json:"brand,omitempty"`
	V     int    `json:"v"`
}

// Healthz godoc
// @Summary  业务健康端点（供 APK 探针校验是我们站点）
// @Description 返回约定 JSON {"ok":true,"brand":"ap","v":1}。带 ?brand= 时回显品牌，供按品牌校验。
// @Tags     app
// @Produce  json
// @Param    brand  query     string  false  "品牌 code（回显，便于 APK 校验 brand 匹配）"
// @Success  200    {object}  healthzResp
// @Router   /healthz [get]
func (h *Handler) Healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, healthzResp{OK: true, Brand: c.QueryParam("brand"), V: 1})
}
