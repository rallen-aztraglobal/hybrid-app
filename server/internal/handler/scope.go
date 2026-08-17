// Package handler — 数据权限（角色可见的品牌/渠道范围）强制点的公共小工具
// （docs/admin/10-rbac.md「数据权限」节）。
package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/repo"
	"github.com/hybrid-app/server/internal/service"
)

// callerScope 解析当前请求调用者的数据范围。runner 静态 token 与未鉴权（理论不会到这里，
// 上层 RequirePerm/RequireActiveAccount 已挡）都返回「不限」——runner 不受数据权限约束
// （要拉全量 manifest 才能构建，见 docs/admin/10-rbac.md「强制点」）。
func (h *Handler) callerScope(c echo.Context) (auth.Scope, error) {
	claims := auth.FromContext(c)
	if claims == nil || claims.Type == auth.TokenRunner {
		return auth.FullScope(), nil
	}
	return h.rbac.EffectiveScope(c.Request().Context(), claims.UserID)
}

// assertChannelIDInScope 校验给定渠道 id 是否在调用者数据范围内，越界返回 404（不是 403——
// 不泄露「存在一个你看不到的渠道」这个事实，见「强制点」单体类一节）。范围全量时跳过查询。
// 用于不需要完整 Channel 记录、只是要做权限闸门的写/下载类端点
// （domains/icon/splash/res.zip/latest-apk 等）。
func (h *Handler) assertChannelIDInScope(c echo.Context, id uint64) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return err
	}
	if scope.AllBrands && scope.AllChannels {
		return nil
	}
	brandCode, err := h.repo.GetChannelBrandCode(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "渠道不存在")
		}
		return err
	}
	if !scope.ChannelAllowed(brandCode, id) {
		return echo.NewHTTPError(http.StatusNotFound, "渠道不存在")
	}
	return nil
}

// assertBrandCodeInScope 校验给定品牌 code 是否在调用者数据范围内，越界返回 404。
// 用于 GET/PUT /brands/:code/domains。
func (h *Handler) assertBrandCodeInScope(c echo.Context, code string) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return err
	}
	if !scope.BrandAllowed(code) {
		return echo.NewHTTPError(http.StatusNotFound, "品牌不存在")
	}
	return nil
}

// applyDeviceInputScope 把 scope 写入 in 的数据权限过滤字段（GET /devices 及其两个导出端点
// 共用）。scope 全量时不改动 in，保持零值语义（= 不限）。
func applyDeviceInputScope(in *service.ListDevicesInput, scope auth.Scope) {
	if scope.AllBrands && scope.AllChannels {
		return
	}
	in.ScopeRestricted = true
	in.ScopeAllBrands = scope.AllBrands
	in.ScopeBrandCodes = scope.BrandCodeList()
	in.ScopeAllChannels = scope.AllChannels
	in.ScopeChannelIDs = scope.ChannelIDList()
}
