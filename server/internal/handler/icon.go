package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
)

// openUpload 取出 multipart 表单里的文件字段 file。
func openUpload(c echo.Context, field string) (io.ReadCloser, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, err
	}
	return fh.Open()
}

// UploadIcon godoc
// @Summary  上传裁剪后的主图(1024²) → imaging 生成全密度 → 返回各槽位预览
// @Tags     icon
// @Accept   multipart/form-data
// @Produce  json
// @Param    id          path      int     true   "渠道 ID"
// @Param    file        formData  file    true   "主图 PNG/JPEG"
// @Param    background  formData  string  false  "自适应图标背景色 #RRGGBB"
// @Success  200         {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id}/icon [post]
func (h *Handler) UploadIcon(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.assertChannelIDInScope(c, id); err != nil {
		return err
	}
	f, err := openUpload(c, "file")
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 file 字段")
	}
	defer f.Close()

	res, err := h.svc.UploadIcon(c.Request().Context(), id, f, c.FormValue("background"))
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

// UploadSplash godoc
// @Summary  上传 splash 源图
// @Tags     icon
// @Accept   multipart/form-data
// @Produce  json
// @Param    id    path      int   true  "渠道 ID"
// @Param    file  formData  file  true  "splash 源图"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/channels/{id}/splash [post]
func (h *Handler) UploadSplash(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.assertChannelIDInScope(c, id); err != nil {
		return err
	}
	f, err := openUpload(c, "file")
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "缺少 file 字段")
	}
	defer f.Close()

	url, err := h.svc.UploadSplash(c.Request().Context(), id, f)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"splashUrl": url})
}

// GetResZip godoc
// @Summary  下载整套 res 资源 zip（CLI/构建机用；越界返回 404，runner 不受数据权限约束）
// @Tags     icon
// @Produce  application/zip
// @Param    id   path  int  true  "渠道 ID"
// @Success  200  {file}  binary
// @Security BearerAuth
// @Router   /api/channels/{id}/res.zip [get]
func (h *Handler) GetResZip(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.assertChannelIDInScope(c, id); err != nil {
		return err
	}
	rc, err := h.svc.ResZip(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	defer rc.Close()
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="res.zip"`)
	return c.Stream(http.StatusOK, "application/zip", rc)
}
