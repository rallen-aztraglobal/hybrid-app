// Package handler — 渠道设备上报 + 设备管理列表 + CSV 导出 HTTP 端点。
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/service"
)

// deviceExportTokenTTL 是 CSV 导出下载令牌的有效期：够用户点开下载即可，尽量短。
const deviceExportTokenTTL = 5 * time.Minute

// registerDeviceReq APK 上报设备信息请求体。
type registerDeviceReq struct {
	AppID      string `json:"appId"`
	PalCode    string `json:"palcode"`
	DeviceKey  string `json:"deviceKey"`
	DeviceName string `json:"deviceName"`
	GAID       string `json:"gaid"`
	ADID       string `json:"adid"`
	OAID       string `json:"oaid"`
}

// RegisterDevice godoc
// @Summary  APK 上报设备信息（公开端点）
// @Tags     device
// @Accept   json
// @Produce  json
// @Param    body  body      registerDeviceReq  true  "设备上报请求"
// @Success  200   {object}  httpx.Envelope
// @Router   /api/app/device/register [post]
func (h *Handler) RegisterDevice(c echo.Context) error {
	var req registerDeviceReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	in := service.RegisterDeviceInput{
		AppID: req.AppID, PalCode: req.PalCode, DeviceKey: req.DeviceKey,
		DeviceName: req.DeviceName, GAID: req.GAID, ADID: req.ADID, OAID: req.OAID,
	}
	if err := h.svc.RegisterDevice(c.Request().Context(), in); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"registered": true})
}

// deviceFilterInput 从查询参数解析设备筛选条件（不含分页），列表与两个按筛选导出端点共用。
// applicationId 支持逗号分隔多值（渠道多选）；device / packageName 为模糊关键字。
func deviceFilterInput(c echo.Context) service.ListDevicesInput {
	in := service.ListDevicesInput{
		DeviceKw:   c.QueryParam("device"),
		PackageKw:  c.QueryParam("packageName"),
		From:       c.QueryParam("from"),
		To:         c.QueryParam("to"),
		ActiveFrom: c.QueryParam("activeFrom"),
		ActiveTo:   c.QueryParam("activeTo"),
	}
	if raw := c.QueryParam("applicationId"); raw != "" {
		for _, v := range strings.Split(raw, ",") {
			if v = strings.TrimSpace(v); v != "" {
				in.ApplicationIDs = append(in.ApplicationIDs, v)
			}
		}
	}
	return in
}

// deviceListInput 从查询参数解析设备列表筛选条件（筛选 + 分页）。
func deviceListInput(c echo.Context) service.ListDevicesInput {
	in := deviceFilterInput(c)
	in.Page, _ = strconv.Atoi(c.QueryParam("page"))
	in.PageSize, _ = strconv.Atoi(c.QueryParam("pageSize"))
	return in
}

// ListDevices godoc
// @Summary  设备列表（分页/筛选，page:devices）
// @Tags     device
// @Produce  json
// @Param    applicationId  query  string  false  "按 applicationId 精确筛选，逗号分隔支持多选"
// @Param    device         query  string  false  "设备关键字（设备名/deviceKey/GAID/ADID 模糊）"
// @Param    packageName    query  string  false  "包名关键字（applicationId 模糊）"
// @Param    from           query  string  false  "起始日期 YYYY-MM-DD（含）"
// @Param    to             query  string  false  "结束日期 YYYY-MM-DD（含）"
// @Param    activeFrom     query  string  false  "最后活跃起始日期 YYYY-MM-DD（含）"
// @Param    activeTo       query  string  false  "最后活跃结束日期 YYYY-MM-DD（含）"
// @Param    page           query  int     false  "页码，默认 1"
// @Param    pageSize       query  int     false  "每页条数，默认 50，最大 200"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/devices [get]
func (h *Handler) ListDevices(c echo.Context) error {
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	in := deviceListInput(c)
	applyDeviceInputScope(&in, scope)
	result, err := h.svc.ListDevices(c.Request().Context(), in)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, result)
}

// GetDeviceExportToken godoc
// @Summary  签发设备 CSV 导出下载令牌（device:export，5 分钟有效）
// @Tags     device
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/devices/export-token [get]
func (h *Handler) GetDeviceExportToken(c echo.Context) error {
	token, err := h.authMgr.IssueScopedToken(service.DeviceExportScope, deviceExportTokenTTL)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{
		"token":     token,
		"expiresIn": int(deviceExportTokenTTL.Seconds()),
	})
}

// deviceCSVHeaders 设好 CSV 流式响应的公共响应头。
func deviceCSVHeaders(c echo.Context) {
	filename := fmt.Sprintf("devices_%s.csv", time.Now().Format("20060102_150405"))
	c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
}

// deviceXLSXHeaders 设好 XLSX 响应的公共响应头。
func deviceXLSXHeaders(c echo.Context) {
	filename := fmt.Sprintf("devices_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
}

// verifyDeviceExportToken 校验导出下载短 token（export-token 换取的一次性令牌），
// export.csv / export.xlsx 两个下载端点共用。
func (h *Handler) verifyDeviceExportToken(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return httpx.Fail(c, http.StatusUnauthorized, "缺少 token")
	}
	if _, err := h.authMgr.VerifyScopedToken(token, service.DeviceExportScope); err != nil {
		return httpx.Fail(c, http.StatusUnauthorized, "token 校验失败")
	}
	return nil
}

// ExportDevicesCSV godoc
// @Summary  设备 CSV 导出（按筛选条件流式下载；用 export-token 换来的一次性令牌鉴权，不接受 access token）
//
// 已知缺口（数据权限）：IssueScopedToken 按设计"不挂用户身份"（见 auth.Manager.IssueScopedToken
// 注释），这条端点走的是 scoped token 而非 JWT access token，handler 侧拿不到 auth.FromContext——
// 无法在这里还原发起下载的账号是谁、进而应用其数据范围。GET /devices（列表）与 POST
// /devices/export（勾选 id 导出）都在正常 JWT 鉴权下，已按调用者数据范围过滤/裁剪；这条
// 按筛选条件的批量下载暂未收窄，等价于任何持有 device:export 权限的账号都能导出全量。
// 收窄需要给 scoped token 加一个用户身份字段（改 IssueScopedToken 签名），本轮未做。
// @Tags     device
// @Produce  text/csv
// @Param    token          query  string  true   "GET /api/devices/export-token 换来的令牌"
// @Param    applicationId  query  string  false  "按 applicationId 精确筛选，逗号分隔支持多选"
// @Param    device         query  string  false  "设备关键字（设备名/deviceKey/GAID/ADID 模糊）"
// @Param    packageName    query  string  false  "包名关键字（applicationId 模糊）"
// @Param    from           query  string  false  "起始日期 YYYY-MM-DD（含）"
// @Param    to             query  string  false  "结束日期 YYYY-MM-DD（含）"
// @Param    activeFrom     query  string  false  "最后活跃起始日期 YYYY-MM-DD（含）"
// @Param    activeTo       query  string  false  "最后活跃结束日期 YYYY-MM-DD（含）"
// @Success  200  {file}  binary
// @Router   /api/devices/export.csv [get]
func (h *Handler) ExportDevicesCSV(c echo.Context) error {
	if err := h.verifyDeviceExportToken(c); err != nil {
		return err
	}

	in := deviceFilterInput(c)
	// 先设响应头（尚未提交响应）；真正的首字节写入在 service 层 streamDeviceCSV 里。
	// 参数校验失败（如 from/to 格式非法）发生在任何写入之前，此时响应仍未提交，可以正常走
	// 统一 JSON 错误响应；一旦已经开始流式写出，出错只能记日志、不能再改写响应体。
	deviceCSVHeaders(c)
	flush := func() { c.Response().Flush() }
	if err := h.svc.ExportDevicesCSV(c.Request().Context(), c.Response(), in, flush); err != nil {
		if !c.Response().Committed {
			return fail(c, err)
		}
		c.Logger().Errorf("导出设备 CSV 失败: %v", err)
		return nil
	}
	return nil
}

// ExportDevicesXLSX godoc
// @Summary  设备 XLSX 导出（美化表头/列宽/冻结首行；按筛选条件下载，鉴权同 export.csv）
//
// 数据权限已知缺口同 ExportDevicesCSV（scoped token 不挂用户身份，本条暂未按调用者数据范围过滤）。
// @Tags     device
// @Produce  application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param    token          query  string  true   "GET /api/devices/export-token 换来的令牌"
// @Param    applicationId  query  string  false  "按 applicationId 精确筛选，逗号分隔支持多选"
// @Param    device         query  string  false  "设备关键字（设备名/deviceKey/GAID/ADID 模糊）"
// @Param    packageName    query  string  false  "包名关键字（applicationId 模糊）"
// @Param    from           query  string  false  "起始日期 YYYY-MM-DD（含）"
// @Param    to             query  string  false  "结束日期 YYYY-MM-DD（含）"
// @Param    activeFrom     query  string  false  "最后活跃起始日期 YYYY-MM-DD（含）"
// @Param    activeTo       query  string  false  "最后活跃结束日期 YYYY-MM-DD（含）"
// @Success  200  {file}  binary
// @Router   /api/devices/export.xlsx [get]
func (h *Handler) ExportDevicesXLSX(c echo.Context) error {
	if err := h.verifyDeviceExportToken(c); err != nil {
		return err
	}

	in := deviceFilterInput(c)
	// XLSX 是 zip 容器，无法边生成边发；参数错误都在写响应体之前抛出，可正常走 JSON 错误。
	deviceXLSXHeaders(c)
	if err := h.svc.ExportDevicesXLSX(c.Request().Context(), c.Response(), in); err != nil {
		if !c.Response().Committed {
			return fail(c, err)
		}
		c.Logger().Errorf("导出设备 XLSX 失败: %v", err)
		return nil
	}
	return nil
}

// exportDevicesByIDsReq 勾选 id 导出请求体。format 可选 "xlsx"（默认，美化 Excel）或 "csv"（原始）。
type exportDevicesByIDsReq struct {
	IDs    []uint64 `json:"ids"`
	Format string   `json:"format"`
}

// ExportDevicesByIDs godoc
// @Summary  按勾选 id 导出设备 CSV（device:export，同构流式响应）
// @Tags     device
// @Accept   json
// @Produce  text/csv
// @Param    body  body  exportDevicesByIDsReq  true  "id 列表（最多 1000 个）"
// @Success  200  {file}  binary
// @Security BearerAuth
// @Router   /api/devices/export [post]
func (h *Handler) ExportDevicesByIDs(c echo.Context) error {
	var req exportDevicesByIDsReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	// 这条走正常 JWT 鉴权（不同于 export.csv/xlsx 的 scoped token），能还原调用者身份，
	// 按数据范围过滤勾选的 id（越界的静默丢弃，见 service.filterDevicesByScope）。
	scope, err := h.callerScope(c)
	if err != nil {
		return fail(c, err)
	}
	if req.Format == "csv" {
		deviceCSVHeaders(c)
		flush := func() { c.Response().Flush() }
		if err := h.svc.ExportDevicesCSVByIDs(c.Request().Context(), c.Response(), scope, req.IDs, flush); err != nil {
			if !c.Response().Committed {
				return fail(c, err)
			}
			c.Logger().Errorf("按 id 导出设备 CSV 失败: %v", err)
			return nil
		}
		return nil
	}
	deviceXLSXHeaders(c)
	if err := h.svc.ExportDevicesXLSXByIDs(c.Request().Context(), c.Response(), scope, req.IDs); err != nil {
		if !c.Response().Committed {
			return fail(c, err)
		}
		c.Logger().Errorf("按 id 导出设备 XLSX 失败: %v", err)
		return nil
	}
	return nil
}
