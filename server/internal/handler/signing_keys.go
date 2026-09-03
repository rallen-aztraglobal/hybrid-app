package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/model"
)

// ListSigningKeys godoc
// @Summary  签名 key 注册表（固定清单，含默认项）
// @Description  仅返回 ID/名称/证书指纹等公开元信息，不含任何密钥材料（密钥全在构建机镜像里，见 model.SigningKeyInfo）。
// @Tags     signing-keys
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/signing-keys [get]
func (h *Handler) ListSigningKeys(c echo.Context) error {
	return httpx.OK(c, model.SigningKeys())
}
