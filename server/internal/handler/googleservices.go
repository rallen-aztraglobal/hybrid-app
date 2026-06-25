// Package handler — google-services.json 分发端点（ADR-0012 §3 + §5）。
package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
)

// GetGoogleServices godoc
// @Summary  下载品牌 google-services.json（公开端点，非机密，CLI/构建机消费）
// @Description 返回该品牌合并的 google-services.json 原始 JSON 字节（非 Envelope）。未上传则 404；CLI 据此判断「未配置 FCM 跳过」。
// @Tags     app
// @Produce  application/json
// @Param    brand  query  string  true  "品牌 code：ap | bp | gp"
// @Success  200    {string}  string  "原始 JSON 内容"
// @Failure  400    {object}  httpx.Envelope
// @Failure  404    {object}  httpx.Envelope
// @Router   /api/app/google-services [get]
func (h *Handler) GetGoogleServices(c echo.Context) error {
	brand := c.QueryParam("brand")
	if brand == "" {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 brand 参数")
	}
	rc, err := h.svc.GetGoogleServices(c.Request().Context(), brand)
	if err != nil {
		return fail(c, err)
	}
	defer rc.Close()
	// 直接透传原始 JSON，不套 Envelope（CLI 拿到后原样写盘）。
	c.Response().Header().Set(echo.HeaderContentType, "application/json; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	_, err = io.Copy(c.Response(), rc)
	return err
}

// UploadGoogleServices godoc
// @Summary  上传品牌合并 google-services.json（operator+）
// @Description 接受 multipart file 字段 "file" 或 raw JSON body；校验 project_info/client 后存入 Storage（key=fcm/<brand>/google-services.json）。
// @Tags     push
// @Accept   multipart/form-data
// @Produce  json
// @Param    brand  query     string  true  "品牌 code：ap | bp | gp"
// @Param    file   formData  file    false "google-services.json 文件（multipart）"
// @Success  200    {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/google-services [post]
func (h *Handler) UploadGoogleServices(c echo.Context) error {
	brand := c.QueryParam("brand")
	if brand == "" {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 brand 参数")
	}

	// 优先取 multipart file 字段 "file"；不存在则用 raw body。
	var reader io.Reader
	fh, err := c.FormFile("file")
	if err == nil {
		f, err := fh.Open()
		if err != nil {
			return httpx.Fail(c, http.StatusBadRequest, "打开上传文件失败")
		}
		defer f.Close()
		reader = f
	} else {
		// raw body（Content-Type: application/json）。
		reader = c.Request().Body
	}

	result, err := h.svc.UploadGoogleServices(c.Request().Context(), brand, reader)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, result)
}
