package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/model"
)

// BrandView 是 GET /api/brands 的单条返回，含渠道计数与域名。
// PackagePrefix 供前端「新增渠道」时自动派生 applicationId（ADR-0009）。
type BrandView struct {
	ID            uint64 `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	PackagePrefix string `json:"packagePrefix"`
	Scheme        string `json:"scheme"`
	HMSEnabled    bool   `json:"hmsEnabled"`
	// SupportsChannels=false 的品牌只做上架包（ADR-0017）：Console 渠道页/打包中心不展示它，
	// 上架包抽屉的「归属品牌」仍可选它。
	SupportsChannels bool     `json:"supportsChannels"`
	AccentColor      string   `json:"accentColor"`
	Sort             int      `json:"sort"`
	ChannelCount     int64    `json:"channelCount"`
	Domains          []string `json:"domains"`
	// AdjustDeepLinkHost 该品牌 Adjust 品牌短链的 host（如 link.bingoplus.com），空串 = 未配置。
	AdjustDeepLinkHost string `json:"adjustDeepLinkHost"`
}

// ListBrands 返回全部大渠道（含渠道计数与默认域名），供前端顶部 Tab。
// 含只做上架包的品牌（SupportsChannels=false）——域名配置页与上架包抽屉需要它们，
// 渠道相关页面由前端按该字段过滤（ADR-0017）。
// scope 是调用者的数据范围（数据权限强制点：GET /brands 列表类查询层过滤，见
// docs/admin/10-rbac.md）：非全量范围时只返回 scope 内的品牌。
func (s *Service) ListBrands(ctx context.Context, scope auth.Scope) ([]BrandView, error) {
	brands, err := s.repo.ListBrands(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.CountChannelsByBrand(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BrandView, 0, len(brands))
	for i := range brands {
		b := &brands[i]
		if !scope.BrandAllowed(b.Code) {
			continue
		}
		out = append(out, brandView(b, counts[b.ID]))
	}
	return out, nil
}

// maxAdjustDeepLinkHostLen 与 brand.adjust_deeplink_host 列定义 VARCHAR(128) 对齐。
const maxAdjustDeepLinkHostLen = 128

// normalizeAdjustDeepLinkHost 校验并规范化 Adjust 品牌短链 host：trim 后转小写；空串合法
// （= 未配置）；非空必须是合法 host（不含 scheme/路径/端口/空白），长度不超过 128。
func normalizeAdjustDeepLinkHost(raw string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(raw))
	if h == "" {
		return "", nil
	}
	if len(h) > maxAdjustDeepLinkHostLen {
		return "", errBadRequest(fmt.Sprintf("adjustDeepLinkHost 长度不能超过 %d", maxAdjustDeepLinkHostLen))
	}
	// 与品牌域名同一套 host 规则（含点、各段合法、不含 scheme/路径/端口/空白）。
	if !domainutil.IsPlausibleHost(h) {
		return "", errBadRequest("adjustDeepLinkHost 必须是合法 host（如 link.bingoplus.com，不含 https://、路径或端口）")
	}
	return h, nil
}

// SetBrandAdjustHost 更新品牌 Adjust 品牌短链 host（PUT /api/brands/:code/adjust）。
func (s *Service) SetBrandAdjustHost(ctx context.Context, code string, rawHost string) (*BrandView, error) {
	brand, err := s.repo.GetBrandByCode(ctx, code)
	if err != nil {
		return nil, errNotFound(fmt.Sprintf("品牌 %q 不存在", code))
	}
	host, err := normalizeAdjustDeepLinkHost(rawHost)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateBrandFields(ctx, brand.ID, map[string]any{"adjust_deeplink_host": host}); err != nil {
		return nil, err
	}

	brand.AdjustDeepLinkHost = host
	counts, err := s.repo.CountChannelsByBrand(ctx)
	if err != nil {
		return nil, err
	}
	view := brandView(brand, counts[brand.ID])
	return &view, nil
}

// brandView 把 model.Brand 组装成 BrandView（只带 enabled 的品牌域名）。ListBrands 与
// SetBrandAdjustHost 共用，避免两处各拼一份 12 个字段的字面量、以后加字段漏改一边。
func brandView(b *model.Brand, channelCount int64) BrandView {
	domains := make([]string, 0, len(b.Domains))
	for _, d := range b.Domains {
		if d.Enabled {
			domains = append(domains, d.URL)
		}
	}
	return BrandView{
		ID:                 b.ID,
		Code:               b.Code,
		Name:               b.Name,
		PackagePrefix:      b.PackagePrefix,
		Scheme:             b.Scheme,
		HMSEnabled:         b.HMSEnabled,
		SupportsChannels:   b.SupportsChannels,
		AccentColor:        b.AccentColor,
		Sort:               b.Sort,
		ChannelCount:       channelCount,
		Domains:            domains,
		AdjustDeepLinkHost: b.AdjustDeepLinkHost,
	}
}

// Login 校验账号密码，成功返回用户。
func (s *Service) Login(ctx context.Context, username, password string) (*model.AdminUser, error) {
	u, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errBadRequest("用户名或密码错误")
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, errBadRequest("用户名或密码错误")
	}
	return u, nil
}
