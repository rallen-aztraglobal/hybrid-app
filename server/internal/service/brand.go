package service

import (
	"context"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
)

// BrandView 是 GET /api/brands 的单条返回，含渠道计数与域名。
// PackagePrefix 供前端「新增渠道」时自动派生 applicationId（ADR-0009）。
type BrandView struct {
	ID            uint64   `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	PackagePrefix string   `json:"packagePrefix"`
	Scheme        string   `json:"scheme"`
	HMSEnabled    bool     `json:"hmsEnabled"`
	AccentColor   string   `json:"accentColor"`
	Sort          int      `json:"sort"`
	ChannelCount  int64    `json:"channelCount"`
	Domains       []string `json:"domains"`
}

// ListBrands 返回三个大渠道（含渠道计数与默认域名），供前端顶部 Tab。
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
		domains := make([]string, 0, len(b.Domains))
		for _, d := range b.Domains {
			if d.Enabled {
				domains = append(domains, d.URL)
			}
		}
		out = append(out, BrandView{
			ID:            b.ID,
			Code:          b.Code,
			Name:          b.Name,
			PackagePrefix: b.PackagePrefix,
			Scheme:        b.Scheme,
			HMSEnabled:    b.HMSEnabled,
			AccentColor:   b.AccentColor,
			Sort:          b.Sort,
			ChannelCount:  counts[b.ID],
			Domains:       domains,
		})
	}
	return out, nil
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
