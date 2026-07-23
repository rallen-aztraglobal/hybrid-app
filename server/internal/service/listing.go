// Package service — 上架包（listing）用例编排。
//
// 分工：
//   - listinggate.go 放纯判定 EvaluateGate（无 I/O，已穷举测试）；
//   - 本文件放需要 I/O 的编排：CRUD、网关规则保存校验、以及公开端点
//     EvaluateListingGate（查库 → 查 GeoIP → 调 EvaluateGate → 落日志 → 取 B 面域名）。
package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/model"
)

// ————————————————————————————————————————————————————————————
// 公开端点：AB 面判定
// ————————————————————————————————————————————————————————————

// GateRequest 是客户端调 POST /api/app/listing/gate 的输入（已从 HTTP 层解析）。
type GateRequest struct {
	Platform string // android / ios
	BundleID string // 包名
	Timezone string // 客户端上报的 IANA 时区名（可伪造，仅作收紧条件）
	IP       net.IP // 由 httpx.ClientIP 从可信代理链提取的真实客户端 IP
}

// GateResponse 是返回给客户端的判定结果。
//
// 刻意极简：只告诉客户端「进 A 还是 B」以及 B 面该开哪个 URL。
// 绝不返回判定原因、命中的规则、国家码等任何可反推规则的信息——审核方也会调这个接口。
type GateResponse struct {
	Mode string `json:"mode"`          // "A" | "B"
	URL  string `json:"url,omitempty"` // 仅 mode=B 时有值：B 面完整地址（主域名 + /?palcode=，客户端原样打开）
}

// EvaluateListingGate 是公开网关端点的核心编排。
//
// 任何一步出错都不向客户端暴露细节，且一律安全降级为 A 面：
// 包不存在、GeoIP 故障、域名取不到……后果都只是「少放一个用户进 B」，绝不会误放审核员。
func (s *Service) EvaluateListingGate(ctx context.Context, req GateRequest) (*GateResponse, error) {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	bundle := strings.TrimSpace(req.BundleID)
	if platform == "" || bundle == "" {
		return nil, errBadRequest("缺少 platform 或 bundleId")
	}

	// 取上架包（含网关规则）。查不到 → A 面（但要区分「未知包」与判 A，故用一个哨兵 listing）。
	listing, err := s.repo.GetListingByPlatformBundle(ctx, platform, bundle)
	if err != nil {
		// 未知包不落日志（无 listing_id 可挂），直接返回 A。
		return &GateResponse{Mode: model.GateModeA}, nil
	}

	// 已归档/停用的包一律 A 面。
	if listing.Status != model.ListingEnabled {
		s.logGate(ctx, listing.ID, req, "", model.GateModeA, "上架包非启用状态")
		return &GateResponse{Mode: model.GateModeA}, nil
	}

	// GeoIP 解析国家（resolver 可空 → 未知）。
	country := ""
	if s.geo != nil && req.IP != nil {
		if c, ok := s.geo.Country(req.IP); ok {
			country = c
		}
	}

	decision := EvaluateGate(GateInput{
		GateEnabled: listing.GateEnabled,
		Gate:        listing.Gate,
		Country:     country,
		Timezone:    req.Timezone,
		IP:          req.IP,
	})

	s.logGate(ctx, listing.ID, req, country, decision.Mode, decision.Reason)

	if decision.Mode != model.GateModeB {
		return &GateResponse{Mode: model.GateModeA}, nil
	}

	// B 面：取生效域名（覆盖优先，否则继承品牌），主域名给客户端。
	domains, err := s.effectiveListingDomains(ctx, listing)
	if err != nil || len(domains) == 0 {
		// 判为 B 却没有可用域名属于配置事故：安全起见退回 A，不给空 URL 让客户端裸奔。
		s.logGate(ctx, listing.ID, req, country, model.GateModeA, "判 B 但无可用域名，降级 A")
		return &GateResponse{Mode: model.GateModeA}, nil
	}

	return &GateResponse{Mode: model.GateModeB, URL: buildListingBSideURL(domains[0], listing.PalCode)}, nil
}

// logGate 落一条判定流水（容错：失败只吞掉，绝不影响判定返回）。
func (s *Service) logGate(ctx context.Context, listingID uint64, req GateRequest, country, mode, reason string) {
	if !s.cfg.GateLogEnable {
		return
	}
	ipStr := ""
	if req.IP != nil {
		ipStr = req.IP.String()
	}
	_ = s.repo.CreateGateLog(ctx, &model.ListingGateLog{
		ListingID: listingID,
		IP:        ipStr,
		Country:   country,
		Timezone:  req.Timezone,
		Decision:  mode,
		Reason:    reason,
	})
}

// effectiveListingDomains 返回上架包 B 面的生效域名：覆盖优先，否则继承品牌。
// 语义与 channel 的 effectiveDomains 一致（ADR-0006）。
func (s *Service) effectiveListingDomains(ctx context.Context, l *model.ListingApp) ([]string, error) {
	if !l.UseBrandDomains && len(l.Domains) > 0 {
		out := make([]string, 0, len(l.Domains))
		for _, d := range l.Domains {
			if d.Enabled {
				out = append(out, d.URL)
			}
		}
		return out, nil
	}
	brand, err := s.repo.GetBrandByID(ctx, l.BrandID)
	if err != nil {
		return nil, err
	}
	full, err := s.repo.GetBrandByCode(ctx, brand.Code)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(full.Domains))
	for _, d := range full.Domains {
		if d.Enabled {
			out = append(out, d.URL)
		}
	}
	return out, nil
}

// ————————————————————————————————————————————————————————————
// 管理面：CRUD
// ————————————————————————————————————————————————————————————

// ListListings 列表（含品牌/域名/网关关联）。
func (s *Service) ListListings(ctx context.Context, platform, status, q string) ([]model.ListingApp, error) {
	return s.repo.ListListings(ctx, repoListingFilter(platform, status, q))
}

// GetListing 详情。
func (s *Service) GetListing(ctx context.Context, id uint64) (*model.ListingApp, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, AsError(err)
	}
	return l, nil
}

// CreateListingInput 是新建上架包的入参。
type CreateListingInput struct {
	BrandCode      string
	Platform       string
	BundleID       string
	Name           string
	DisplayName    string
	StoreURL       string
	PalCode        string // 参考渠道包的 PAL_CODE，运营手填，必填、纯数字串（拼 B 面 /?palcode=）
	AfDevKey       string
	AfAppID        string
	AdjustAppToken *string
	AdjustEvents   model.AdjustEvents
	Remark         string
	CreatedBy      uint64
}

// CreateListing 新建上架包。校验平台/技术栈枚举、品牌存在、(platform,bundleId) 唯一。
// 新建时 gate_enabled 恒为 false（默认只有 A 面），需运营配好规则后再单独打开。
func (s *Service) CreateListing(ctx context.Context, in CreateListingInput) (*model.ListingApp, error) {
	platform, err := normalizePlatform(in.Platform)
	if err != nil {
		return nil, err
	}
	bundle := strings.TrimSpace(in.BundleID)
	if bundle == "" {
		return nil, errBadRequest("包名不能为空")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, errBadRequest("名称不能为空")
	}

	brand, err := s.repo.GetBrandByCode(ctx, in.BrandCode)
	if err != nil {
		return nil, errBadRequest(fmt.Sprintf("品牌 %q 不存在", in.BrandCode))
	}

	// 参考渠道包 PAL_CODE 必填、纯数字串（与渠道包表单同一口径）。
	palCode, err := normalizeListingPalCode(in.PalCode)
	if err != nil {
		return nil, err
	}

	n, err := s.repo.CountListingsByPlatformBundle(ctx, platform, bundle, 0)
	if err != nil {
		return nil, AsError(err)
	}
	if n > 0 {
		return nil, errConflict(fmt.Sprintf("%s 平台已存在包名 %s", platform, bundle))
	}

	adjustToken, err := normalizeAndValidateAdjust(in.AdjustAppToken, in.AdjustEvents)
	if err != nil {
		return nil, err
	}

	l := &model.ListingApp{
		BrandID:         brand.ID,
		Platform:        platform,
		BundleID:        bundle,
		Name:            strings.TrimSpace(in.Name),
		DisplayName:     strings.TrimSpace(in.DisplayName),
		StoreURL:        strings.TrimSpace(in.StoreURL),
		PalCode:         palCode,
		Status:          model.ListingEnabled,
		UseBrandDomains: true,  // 默认继承品牌域名
		GateEnabled:     false, // 默认只有 A 面
		AfDevKey:        strings.TrimSpace(in.AfDevKey),
		AfAppID:         strings.TrimSpace(in.AfAppID),
		AdjustAppToken:  adjustToken,
		AdjustEvents:    in.AdjustEvents,
		Remark:          strings.TrimSpace(in.Remark),
		CreatedBy:       in.CreatedBy,
	}
	if err := s.repo.CreateListing(ctx, l); err != nil {
		return nil, AsError(err)
	}
	return s.GetListing(ctx, l.ID)
}

// UpdateListingInput 是更新上架包的入参。platform/bundleId 不可改（改了等于换个包），故不在此。
type UpdateListingInput struct {
	BrandCode      *string // 非 nil = 改所属品牌（决定 B 面继承的域名清单）；platform/bundleId 仍不可改
	Name           *string
	DisplayName    *string
	StoreURL       *string
	Status         *string
	PalCode        *string // 非 nil = 改参考渠道包 PAL_CODE；仍强制非空纯数字
	GateEnabled    *bool
	AfDevKey       *string
	AfAppID        *string
	AdjustAppToken *string
	AdjustEvents   *model.AdjustEvents
	Remark         *string
}

// UpdateListing 局部更新。指针字段为 nil 表示不改。
// 打开 gate_enabled 前强制校验：必须已配置且国家白名单非空，否则拒绝——
// 避免「开了开关但规则是空的」这种最危险的误操作在服务端被放行。
func (s *Service) UpdateListing(ctx context.Context, id uint64, in UpdateListingInput) (*model.ListingApp, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, AsError(err)
	}

	fields := map[string]any{}
	if in.BrandCode != nil {
		brand, err := s.repo.GetBrandByCode(ctx, *in.BrandCode)
		if err != nil {
			return nil, errBadRequest(fmt.Sprintf("品牌 %q 不存在", *in.BrandCode))
		}
		fields["brand_id"] = brand.ID
	}
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return nil, errBadRequest("名称不能为空")
		}
		fields["name"] = strings.TrimSpace(*in.Name)
	}
	if in.DisplayName != nil {
		fields["display_name"] = strings.TrimSpace(*in.DisplayName)
	}
	if in.StoreURL != nil {
		fields["store_url"] = strings.TrimSpace(*in.StoreURL)
	}
	if in.PalCode != nil {
		palCode, err := normalizeListingPalCode(*in.PalCode)
		if err != nil {
			return nil, err
		}
		fields["pal_code"] = palCode
	}
	if in.Status != nil {
		st, err := normalizeStatus(*in.Status)
		if err != nil {
			return nil, err
		}
		fields["status"] = st
	}
	if in.AfDevKey != nil {
		fields["af_dev_key"] = strings.TrimSpace(*in.AfDevKey)
	}
	if in.AfAppID != nil {
		fields["af_app_id"] = strings.TrimSpace(*in.AfAppID)
	}
	if in.AdjustAppToken != nil || in.AdjustEvents != nil {
		token := l.AdjustAppToken
		if in.AdjustAppToken != nil {
			token = in.AdjustAppToken
		}
		events := l.AdjustEvents
		if in.AdjustEvents != nil {
			events = *in.AdjustEvents
		}
		normalizedToken, err := normalizeAndValidateAdjust(token, events)
		if err != nil {
			return nil, err
		}
		fields["adjust_app_token"] = normalizedToken
		fields["adjust_events"] = events
	}
	if in.Remark != nil {
		fields["remark"] = strings.TrimSpace(*in.Remark)
	}

	// 打开总开关前的安全校验。
	if in.GateEnabled != nil {
		if *in.GateEnabled {
			if err := s.assertGateReadyToEnable(l); err != nil {
				return nil, err
			}
		}
		fields["gate_enabled"] = *in.GateEnabled
	}

	if len(fields) == 0 {
		return s.GetListing(ctx, id)
	}
	if err := s.repo.UpdateListingFields(ctx, id, fields); err != nil {
		return nil, AsError(err)
	}
	return s.GetListing(ctx, id)
}

// normalizeListingPalCode 校验并规范化参考渠道包 PAL_CODE：去空白后必须非空且为纯数字串
// （与渠道包 PAL_CODE 同一口径，见 web validation）。返回规范化后的值。
func normalizeListingPalCode(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" {
		return "", errBadRequest("参考渠道包 PAL_CODE 必填")
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return "", errBadRequest("参考渠道包 PAL_CODE 应为纯数字串")
		}
	}
	return p, nil
}

// assertGateReadyToEnable 校验一个上架包是否具备打开 AB 面的前提。
func (s *Service) assertGateReadyToEnable(l *model.ListingApp) error {
	if l.Gate == nil || len(l.Gate.Countries) == 0 {
		return errBadRequest("打开 AB 面前必须先配置国家白名单（且不能为空）")
	}
	return nil
}

// DeleteListing 删除上架包（连带域名/网关/日志级联删除）。
func (s *Service) DeleteListing(ctx context.Context, id uint64) error {
	if err := s.repo.DeleteListing(ctx, id); err != nil {
		return AsError(err)
	}
	return nil
}

// ————————————————————————————————————————————————————————————
// 管理面：域名覆盖 & 网关规则
// ————————————————————————————————————————————————————————————

// SetListingDomains 设置上架包 B 面域名覆盖；inheritBrand=true 表示切回继承品牌。
// 复用 channel 的域名校验（domainutil）与探测（probeAll），行为一致。
func (s *Service) SetListingDomains(ctx context.Context, id uint64, inheritBrand bool, inputs []domainutil.DomainInput) (*SetDomainsResult, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, AsError(err)
	}

	if inheritBrand {
		if err := s.repo.ReplaceListingDomains(ctx, l.ID, nil); err != nil {
			return nil, AsError(err)
		}
		if err := s.repo.UpdateListingFields(ctx, l.ID, map[string]any{"use_brand_domains": true}); err != nil {
			return nil, AsError(err)
		}
		reloaded, _ := s.repo.GetListing(ctx, l.ID)
		eff, _ := s.effectiveListingDomains(ctx, reloaded)
		return &SetDomainsResult{Domains: eff}, nil
	}

	normalized, err := domainutil.Validate(inputs)
	if err != nil {
		return nil, &Error{Code: http.StatusBadRequest, Message: "域名校验失败: " + err.Error()}
	}
	rows := make([]model.ListingDomain, 0, len(normalized))
	for _, nd := range normalized {
		rows = append(rows, model.ListingDomain{Position: nd.Position, URL: nd.URL, Enabled: nd.Enabled})
	}
	if err := s.repo.ReplaceListingDomains(ctx, l.ID, rows); err != nil {
		return nil, AsError(err)
	}
	if err := s.repo.UpdateListingFields(ctx, l.ID, map[string]any{"use_brand_domains": false}); err != nil {
		return nil, AsError(err)
	}
	probes := s.probeAll(ctx, domainutil.URLs(normalized))
	return &SetDomainsResult{Domains: domainutil.URLs(normalized), Probes: probes}, nil
}

// SetGateInput 是保存网关规则的入参。
type SetGateInput struct {
	Countries    []string
	TimezoneDeny []string // 时区黑名单：命中即强制 A（选填）
	IPAllowCIDRs []string
	IPDenyCIDRs  []string
}

// SetListingGate 保存网关规则，做完整校验：
//   - 国家白名单必填非空，且不得包含被硬编码强制 A 面的国家（CN/US）——即便存了也无效，
//     这里直接拒绝，避免运营误以为配了就生效；
//   - 国家码规范化为大写、去重；
//   - CIDR 语法校验（允许单个 IP 简写）；
//   - 时区黑名单做基本非空清洗（不强制解析 IANA 库，容忍新时区名）。
func (s *Service) SetListingGate(ctx context.Context, id uint64, in SetGateInput) (*model.ListingGate, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, AsError(err)
	}

	countries, err := normalizeCountries(in.Countries)
	if err != nil {
		return nil, err
	}
	if len(countries) == 0 {
		return nil, errBadRequest("国家白名单必填，且至少一个国家")
	}

	if err := validateCIDRs(in.IPAllowCIDRs, "IP 白名单"); err != nil {
		return nil, err
	}
	if err := validateCIDRs(in.IPDenyCIDRs, "IP 黑名单"); err != nil {
		return nil, err
	}
	gate := &model.ListingGate{
		ListingID:    l.ID,
		Countries:    model.StringList(countries),
		TimezoneDeny: model.StringList(trimNonEmpty(in.TimezoneDeny)),
		IPAllowCIDRs: model.StringList(trimNonEmpty(in.IPAllowCIDRs)),
		IPDenyCIDRs:  model.StringList(trimNonEmpty(in.IPDenyCIDRs)),
	}
	if err := s.repo.UpsertListingGate(ctx, gate); err != nil {
		return nil, AsError(err)
	}
	return gate, nil
}

// GateTestInput 是「用某 IP + 时区试算」的入参（Console 保存前预览判定结果）。
type GateTestInput struct {
	IP       string
	Timezone string
}

// GateTestResult 是试算结果，与线上判定不同，这里**返回原因**——因为是后台运营自查，不是客户端。
type GateTestResult struct {
	Mode    string `json:"mode"`
	Reason  string `json:"reason"`
	Country string `json:"country"`
}

// TestListingGate 用给定 IP/时区对某上架包跑一次判定，返回带原因的结果供运营核对。
// 不落判定日志（这是后台试算，不是真实设备请求）。
func (s *Service) TestListingGate(ctx context.Context, id uint64, in GateTestInput) (*GateTestResult, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, AsError(err)
	}
	ip := net.ParseIP(strings.TrimSpace(in.IP))
	if ip == nil {
		return nil, errBadRequest("IP 格式非法")
	}
	country := ""
	if s.geo != nil {
		if c, ok := s.geo.Country(ip); ok {
			country = c
		}
	}
	decision := EvaluateGate(GateInput{
		GateEnabled: l.GateEnabled,
		Gate:        l.Gate,
		Country:     country,
		Timezone:    in.Timezone,
		IP:          ip,
	})
	return &GateTestResult{Mode: decision.Mode, Reason: decision.Reason, Country: country}, nil
}

// ListGateLogs 取判定流水供 Console 排查。
func (s *Service) ListGateLogs(ctx context.Context, id uint64, limit int) ([]model.ListingGateLog, error) {
	list, err := s.repo.ListGateLogs(ctx, id, limit)
	if err != nil {
		return nil, AsError(err)
	}
	return list, nil
}

// ————————————————————————————————————————————————————————————
// 上架包推送：设备 token 注册（带 AB 面判定结果）
// ————————————————————————————————————————————————————————————

// RegisterListingTokenInput 是上架包设备注册 push token 的入参。
// 客户端在拿到 AB 面判定结果后带上 GateMode，服务端据此标记该设备当前属 A 还是 B 面。
type RegisterListingTokenInput struct {
	Platform    string // android / ios
	BundleID    string
	DeviceToken string
	GateMode    string // 客户端上报的最近判定结果 A / B；非法值归一化为空（视为 A，不进 B 面推送）
	ModelInfo   string
}

// RegisterListingDeviceToken 注册/更新上架包设备 token，并记录其最近 AB 面判定结果。
//
// 安全要点：LastGateMode 决定该设备是否进入上架包推送受众（只推 B 面设备）。这里只信任
// 客户端上报为 "B" 的值——但客户端伪造 B 只会让它多收到 B 面推送，不构成把内容推给审核员的风险
// （真正的审核员设备判定为 A，不会上报 B）。反向的安全线（A 面设备不收 B 面推送）由发送侧
// repo.ActiveListingTokensBMode 的硬编码过滤保证。
func (s *Service) RegisterListingDeviceToken(ctx context.Context, in RegisterListingTokenInput) error {
	platform, err := normalizePlatform(in.Platform)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.DeviceToken) == "" {
		return errBadRequest("deviceToken 不得为空")
	}
	listing, err := s.repo.GetListingByPlatformBundle(ctx, platform, strings.TrimSpace(in.BundleID))
	if err != nil {
		return errBadRequest("platform+bundleId 对应上架包不存在")
	}

	// 只接受 A / B 两个合法值，其余一律归一化为空（等价 A：不进 B 面推送）。
	gateMode := strings.ToUpper(strings.TrimSpace(in.GateMode))
	if gateMode != model.GateModeA && gateMode != model.GateModeB {
		gateMode = ""
	}
	now := time.Now()

	brandCode := ""
	if listing.Brand != nil {
		brandCode = listing.Brand.Code
	}
	lid := listing.ID
	dt := &model.PushDeviceToken{
		ApplicationID: listing.BundleID, // 上架包用 bundleId 占位，真正定位靠 ListingID
		ListingID:     &lid,
		BrandCode:     brandCode,
		DeviceToken:   strings.TrimSpace(in.DeviceToken),
		PalCode:       listing.PalCode, // 继承自参考渠道包，随 B 面 URL 传递（repo 已按 RefChannel 填充）
		Platform:      platform,
		LastGateMode:  gateMode,
		LastGateAt:    &now,
		ModelInfo:     in.ModelInfo,
	}
	if err := s.repo.UpsertDeviceToken(ctx, dt); err != nil {
		return fmt.Errorf("注册上架包 token 失败: %w", err)
	}
	return nil
}
