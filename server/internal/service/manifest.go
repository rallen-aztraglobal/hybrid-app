package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// BuildManifest 是 GET /api/build/manifest?brand= 的响应（docs/admin/01 §5.5）。
// CLI 据此重写 channels/<brand>.csv + res + 每 flavor 的 assets/bootstrap.json，Gradle 零改动（ADR-0004）。
type BuildManifest struct {
	Brand         string            `json:"brand"`
	Scheme        string            `json:"scheme"`
	HMSEnabled    bool              `json:"hmsEnabled"`
	BrandDomains  []string          `json:"brandDomains"` // 品牌默认域名（继承用）
	ConfigBaseURL string            `json:"configBaseUrl"`// CDN 配置端点前缀（烧录进 bootstrap.json）
	GeneratedAt   string            `json:"generatedAt"`
	Channels      []ManifestChannel `json:"channels"`
}

// ManifestChannel 单个渠道的全量信息。
type ManifestChannel struct {
	FlavorName     string   `json:"flavorName"`
	ApplicationID  string   `json:"applicationId"`
	PalCode        string   `json:"palCode"`
	AppName        string   `json:"appName"`
	Status         string   `json:"status"`
	EffectiveDomains []string `json:"effectiveDomains"` // 合并继承后的域名（写 bootstrap.json 兜底）
	ResZipURL      string   `json:"resZipUrl"`        // 资源 zip 地址（CLI 下载解压到 flavor 目录）
	SplashURL      string   `json:"splashUrl"`
	ConfigSnapshotURL string `json:"configSnapshotUrl"` // 该渠道的 CDN 配置快照地址
}

// CSVLine 渲染成与现有 channels/*.csv 字节级兼容的一行：flavor|applicationId|palCode|appName。
func (m ManifestChannel) CSVLine() string {
	return strings.Join([]string{m.FlavorName, m.ApplicationID, m.PalCode, m.AppName}, "|")
}

// BuildManifestForBrand 组装某品牌的构建 manifest。
func (s *Service) BuildManifestForBrand(ctx context.Context, brandCode string) (*BuildManifest, error) {
	brand, err := s.repo.GetBrandByCode(ctx, brandCode)
	if err != nil {
		return nil, errNotFound(fmt.Sprintf("品牌 %q 不存在", brandCode))
	}

	brandDomains := make([]string, 0, len(brand.Domains))
	for _, d := range brand.Domains {
		if d.Enabled {
			brandDomains = append(brandDomains, d.URL)
		}
	}

	// 拉取该品牌所有 enabled 渠道（archived/disabled 不下发给构建）。
	list, _, err := s.repo.ListChannels(ctx, repo.ChannelFilter{
		BrandCode: brandCode,
		Status:    model.ChannelEnabled,
		PageSize:  500,
	})
	if err != nil {
		return nil, err
	}

	out := &BuildManifest{
		Brand:         brand.Code,
		Scheme:        brand.Scheme,
		HMSEnabled:    brand.HMSEnabled,
		BrandDomains:  brandDomains,
		ConfigBaseURL: s.storage.PublicURL("app-config"),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for i := range list {
		ch := &list[i]
		eff, err := s.effectiveDomains(ctx, ch.ID)
		if err != nil {
			return nil, err
		}
		out.Channels = append(out.Channels, ManifestChannel{
			FlavorName:        ch.FlavorName,
			ApplicationID:     ch.ApplicationID,
			PalCode:           ch.PalCode,
			AppName:           ch.AppName,
			Status:            ch.Status,
			EffectiveDomains:  eff,
			ResZipURL:         ch.IconSetURL,
			SplashURL:         ch.SplashURL,
			ConfigSnapshotURL: s.SnapshotURL(ch.ApplicationID),
		})
	}
	return out, nil
}

