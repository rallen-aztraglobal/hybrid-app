package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/httpx"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/service"
)

// ————————————————————————————————————————————————————————————
// 管理面：上架包 CRUD
// ————————————————————————————————————————————————————————————

// ListListings godoc
// @Summary  上架包列表（按平台/状态/关键词筛选，含品牌/域名/网关）
// @Tags     listings
// @Produce  json
// @Param    platform  query     string  false  "android / ios"
// @Param    status    query     string  false  "enabled/disabled/archived"
// @Param    q         query     string  false  "关键词（名称/包名）"
// @Success  200       {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings [get]
func (h *Handler) ListListings(c echo.Context) error {
	list, err := h.svc.ListListings(c.Request().Context(),
		c.QueryParam("platform"), c.QueryParam("status"), c.QueryParam("q"))
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"items": list, "total": len(list)})
}

// GetListing godoc
// @Summary  上架包详情
// @Tags     listings
// @Produce  json
// @Param    id   path      int  true  "上架包 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id} [get]
func (h *Handler) GetListing(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	l, err := h.svc.GetListing(c.Request().Context(), id)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, l)
}

// createListingReq 是新建上架包的请求体。AdjustAppToken 用指针区分「不传」与「传空串解绑」。
// OpenMode 留空则由 service 归一化为默认值 internal（内开）。
type createListingReq struct {
	BrandCode      string             `json:"brandCode"`
	Platform       string             `json:"platform"`
	BundleID       string             `json:"bundleId"`
	Name           string             `json:"name"`
	DisplayName    string             `json:"displayName"`
	StoreURL       string             `json:"storeUrl"`
	PalCode        string             `json:"palCode"`
	OpenMode       string             `json:"openMode"`
	AfDevKey       string             `json:"afDevKey"`
	AfAppID        string             `json:"afAppId"`
	AdjustAppToken *string            `json:"adjustAppToken"`
	AdjustEvents   model.AdjustEvents `json:"adjustEvents"`
	Remark         string             `json:"remark"`
}

// CreateListing godoc
// @Summary  新增上架包（默认继承品牌域名、gate_enabled=false 只有 A 面）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    body  body      createListingReq  true  "上架包字段"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings [post]
func (h *Handler) CreateListing(c echo.Context) error {
	var req createListingReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	l, err := h.svc.CreateListing(c.Request().Context(), service.CreateListingInput{
		BrandCode:      req.BrandCode,
		Platform:       req.Platform,
		BundleID:       req.BundleID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		StoreURL:       req.StoreURL,
		PalCode:        req.PalCode,
		OpenMode:       req.OpenMode,
		AfDevKey:       req.AfDevKey,
		AfAppID:        req.AfAppID,
		AdjustAppToken: req.AdjustAppToken,
		AdjustEvents:   req.AdjustEvents,
		Remark:         req.Remark,
		CreatedBy:      currentUserID(c),
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, l)
}

// updateListingReq 是更新请求体。全部字段可选（指针 nil = 不改）。OpenMode 非 nil 时按
// model.NormalizeOpenMode 归一化（空/非法值一律回落 internal）。
type updateListingReq struct {
	BrandCode      *string             `json:"brandCode"`
	Name           *string             `json:"name"`
	DisplayName    *string             `json:"displayName"`
	StoreURL       *string             `json:"storeUrl"`
	Status         *string             `json:"status"`
	PalCode        *string             `json:"palCode"`
	GateEnabled    *bool               `json:"gateEnabled"`
	OpenMode       *string             `json:"openMode"`
	AfDevKey       *string             `json:"afDevKey"`
	AfAppID        *string             `json:"afAppId"`
	AdjustAppToken *string             `json:"adjustAppToken"`
	AdjustEvents   *model.AdjustEvents `json:"adjustEvents"`
	Remark         *string             `json:"remark"`
}

// UpdateListing godoc
// @Summary  修改上架包（打开 gateEnabled 前服务端强制校验国家白名单非空）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    id    path      int               true  "上架包 ID"
// @Param    body  body      updateListingReq  true  "可选更新字段"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id} [put]
func (h *Handler) UpdateListing(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req updateListingReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	l, err := h.svc.UpdateListing(c.Request().Context(), id, service.UpdateListingInput{
		BrandCode:      req.BrandCode,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		StoreURL:       req.StoreURL,
		Status:         req.Status,
		PalCode:        req.PalCode,
		GateEnabled:    req.GateEnabled,
		OpenMode:       req.OpenMode,
		AfDevKey:       req.AfDevKey,
		AfAppID:        req.AfAppID,
		AdjustAppToken: req.AdjustAppToken,
		AdjustEvents:   req.AdjustEvents,
		Remark:         req.Remark,
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, l)
}

// DeleteListing godoc
// @Summary  删除上架包（连带域名/网关/日志级联删除）
// @Tags     listings
// @Produce  json
// @Param    id   path      int  true  "上架包 ID"
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id} [delete]
func (h *Handler) DeleteListing(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	if err := h.svc.DeleteListing(c.Request().Context(), id); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"deleted": true})
}

// ————————————————————————————————————————————————————————————
// 管理面：域名覆盖 & 网关规则
// ————————————————————————————————————————————————————————————

type setListingDomainsReq struct {
	InheritBrand bool                     `json:"inheritBrand"`
	Domains      []domainutil.DomainInput `json:"domains"`
}

// SetListingDomains godoc
// @Summary  设置上架包 B 面域名覆盖 / 切回继承品牌
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    id    path      int                   true  "上架包 ID"
// @Param    body  body      setListingDomainsReq  true  "inheritBrand=true 表示切回继承"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id}/domains [put]
func (h *Handler) SetListingDomains(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req setListingDomainsReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.SetListingDomains(c.Request().Context(), id, req.InheritBrand, req.Domains)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

type setListingGateReq struct {
	Countries    []string `json:"countries"`
	TimezoneDeny []string `json:"timezoneDeny"`
	IPAllowCIDRs []string `json:"ipAllowCidrs"`
	IPDenyCIDRs  []string `json:"ipDenyCidrs"`
}

// SetListingGate godoc
// @Summary  保存上架包 AB 面网关规则（国家白名单必填、拒绝 CN/US）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    id    path      int                true  "上架包 ID"
// @Param    body  body      setListingGateReq  true  "网关规则"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id}/gate [put]
func (h *Handler) SetListingGate(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req setListingGateReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	gate, err := h.svc.SetListingGate(c.Request().Context(), id, service.SetGateInput{
		Countries:    req.Countries,
		TimezoneDeny: req.TimezoneDeny,
		IPAllowCIDRs: req.IPAllowCIDRs,
		IPDenyCIDRs:  req.IPDenyCIDRs,
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, gate)
}

type testListingGateReq struct {
	IP       string `json:"ip"`
	Timezone string `json:"timezone"`
}

// TestListingGate godoc
// @Summary  用指定 IP/时区试算 AB 面判定（后台自查，返回原因）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    id    path      int                 true  "上架包 ID"
// @Param    body  body      testListingGateReq  true  "试算输入"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id}/gate/test [post]
func (h *Handler) TestListingGate(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req testListingGateReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.TestListingGate(c.Request().Context(), id, service.GateTestInput{IP: req.IP, Timezone: req.Timezone})
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

// ListGateLogs godoc
// @Summary  上架包网关判定流水（排查「为什么没进 B 面」）
// @Tags     listings
// @Produce  json
// @Param    id     path      int  true   "上架包 ID"
// @Param    limit  query     int  false  "条数，默认 100，上限 500"
// @Success  200    {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/listings/{id}/gate/logs [get]
func (h *Handler) ListGateLogs(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	logs, err := h.svc.ListGateLogs(c.Request().Context(), id, limit)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"items": logs, "total": len(logs)})
}

// ————————————————————————————————————————————————————————————
// 公开端点：AB 面判定（客户端 App 消费）
// ————————————————————————————————————————————————————————————

type gateReq struct {
	Platform string `json:"platform"`
	BundleID string `json:"bundleId"`
	Timezone string `json:"timezone"`
}

// ListingGate godoc
// @Summary  AB 面判定（公开，客户端 App 启动调用，不缓存）
// @Description 服务端按请求真实 IP 查国家 + 校验时区/IP 规则，返回 mode=A（展示应用内容）或 mode=B（放行到配置 URL）。
// @Description mode=B 时额外带 openMode（internal=内开/external=外开），告诉客户端用哪种方式打开该 URL；mode=A 时不含该字段。
// @Description 出于安全：不返回任何判定原因/国家/命中规则，审核方也无法从响应反推规则。
// @Tags     app
// @Accept   json
// @Produce  json
// @Param    body  body      gateReq  true  "platform/bundleId/timezone"
// @Success  200   {object}  service.GateResponse
// @Router   /api/app/listing/gate [post]
func (h *Handler) ListingGate(c echo.Context) error {
	var req gateReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	res, err := h.svc.EvaluateListingGate(c.Request().Context(), service.GateRequest{
		Platform: req.Platform,
		BundleID: req.BundleID,
		Timezone: req.Timezone,
		IP:       httpx.ClientIP(c.Request(), h.trustedProxies),
	})
	if err != nil {
		return fail(c, err)
	}
	// 明确不缓存：判定因 IP 而异，且需实时反映后台规则变更（与 /api/app/config 的强缓存相反）。
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(http.StatusOK, res)
}

// ————————————————————————————————————————————————————————————
// 管理面：上架包推送（强制只发 B 面设备）
// ————————————————————————————————————————————————————————————

type createListingCampaignReq struct {
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	ImageURL     string            `json:"imageUrl"`
	DeeplinkPath string            `json:"deeplinkPath"`
	ExtraData    map[string]string `json:"extraData"`
	ListingIDs   []uint64          `json:"listingIds"`
}

// CreateListingCampaign godoc
// @Summary  创建上架包推送活动（草稿；发送时强制只发 B 面设备）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    body  body      createListingCampaignReq  true  "活动字段 + 目标上架包 ids"
// @Success  201   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/listing-campaigns [post]
func (h *Handler) CreateListingCampaign(c echo.Context) error {
	var req createListingCampaignReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	campaign, err := h.svc.CreateListingCampaign(c.Request().Context(), service.CreateListingCampaignInput{
		Name:         req.Name,
		Title:        req.Title,
		Body:         req.Body,
		ImageURL:     req.ImageURL,
		DeeplinkPath: req.DeeplinkPath,
		ExtraData:    req.ExtraData,
		ListingIDs:   req.ListingIDs,
		CreatedBy:    currentUsername(c),
	})
	if err != nil {
		return fail(c, err)
	}
	return httpx.Created(c, campaign)
}

// ListListingCampaigns godoc
// @Summary  上架包推送活动列表（kind=listing）
// @Tags     listings
// @Produce  json
// @Success  200  {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/listing-campaigns [get]
func (h *Handler) ListListingCampaigns(c echo.Context) error {
	list, err := h.svc.ListListingCampaigns(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"items": list, "total": len(list)})
}

// SendListingCampaign godoc
// @Summary  发送上架包推送活动（dryRun 预览受众；真发只投 B 面设备）
// @Tags     listings
// @Accept   json
// @Produce  json
// @Param    id    path      int                true  "活动 ID"
// @Param    body  body      sendCampaignReq    false "dryRun"
// @Success  200   {object}  httpx.Envelope
// @Security BearerAuth
// @Router   /api/push/listing-campaigns/{id}/send [post]
func (h *Handler) SendListingCampaign(c echo.Context) error {
	id, err := paramID(c)
	if err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "非法 id")
	}
	var req sendCampaignReq
	_ = c.Bind(&req)
	res, err := h.svc.SendListingCampaign(c.Request().Context(), id, req.DryRun)
	if err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, res)
}

type registerListingTokenReq struct {
	Platform    string `json:"platform"`
	BundleID    string `json:"bundleId"`
	DeviceToken string `json:"deviceToken"`
	GateMode    string `json:"gateMode"` // 客户端上报的最近 AB 面判定结果 A/B（决定是否进 B 面推送受众）
	Model       string `json:"model"`
}

// RegisterListingToken godoc
// @Summary  上架包设备 push token 注册（公开，带 AB 面判定结果）
// @Description 客户端拿到 gate 判定后注册 FCM/APNs token 并上报 gateMode；上架包推送强制只发 B 面设备。
// @Tags     app
// @Accept   json
// @Produce  json
// @Param    body  body      registerListingTokenReq  true  "platform/bundleId/deviceToken/gateMode"
// @Success  200   {object}  httpx.Envelope
// @Router   /api/app/listing/register-token [post]
func (h *Handler) RegisterListingToken(c echo.Context) error {
	var req registerListingTokenReq
	if err := c.Bind(&req); err != nil {
		return httpx.Fail(c, http.StatusBadRequest, "请求参数解析失败")
	}
	if err := h.svc.RegisterListingDeviceToken(c.Request().Context(), service.RegisterListingTokenInput{
		Platform:    req.Platform,
		BundleID:    req.BundleID,
		DeviceToken: req.DeviceToken,
		GateMode:    req.GateMode,
		ModelInfo:   req.Model,
	}); err != nil {
		return fail(c, err)
	}
	return httpx.OK(c, map[string]any{"registered": true})
}
