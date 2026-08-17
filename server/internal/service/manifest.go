package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/auth"
	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// BuildManifest 是 GET /api/build/manifest?brand= 的响应（docs/admin/01 §5.5）。
// CLI 据此重写 channels/<brand>.csv + res + 每 flavor 的 assets/bootstrap.json，Gradle 零改动（ADR-0004）。
type BuildManifest struct {
	Brand         string            `json:"brand"`
	Scheme        string            `json:"scheme"`
	HMSEnabled    bool              `json:"hmsEnabled"`
	BrandDomains  []string          `json:"brandDomains"`  // 品牌默认域名（继承用）
	ConfigBaseURL string            `json:"configBaseUrl"` // CDN 配置端点前缀（烧录进 bootstrap.json）
	GeneratedAt   string            `json:"generatedAt"`
	Channels      []ManifestChannel `json:"channels"`
}

// ManifestChannel 单个渠道的全量信息。
//
// AdjustAppToken/AdjustEvents（ADR-0013 / docs/admin/08-adjust.md §5）：原样带出该渠道的 Adjust 绑定，
// 供 CLI pull/build 渲染 app/adjust-tokens.json（键=applicationId）；JSON 字段名 adjustAppToken /
// adjustEvents，与渠道 CRUD API 及本结构体其余字段一致走 camelCase。空 token 表示该渠道未绑定 Adjust，
// CLI 侧应据此跳过、不写入 adjust-tokens.json（保持依赖恒在 classpath、按 flavor 探测开关的心智）。
type ManifestChannel struct {
	FlavorName        string            `json:"flavorName"`
	ApplicationID     string            `json:"applicationId"`
	PalCode           string            `json:"palCode"`
	AppName           string            `json:"appName"`
	Status            string            `json:"status"`
	EffectiveDomains  []string          `json:"effectiveDomains"` // 合并继承后的域名（写 bootstrap.json 兜底）
	ResZipURL         string            `json:"resZipUrl"`        // 资源 zip 地址（CLI 下载解压到 flavor 目录）
	SplashURL         string            `json:"splashUrl"`
	ConfigSnapshotURL string            `json:"configSnapshotUrl"` // 该渠道的 CDN 配置快照地址
	AdjustAppToken    string            `json:"adjustAppToken,omitempty"`
	AdjustEvents      map[string]string `json:"adjustEvents,omitempty"`
}

// CSVLine 渲染成与现有 channels/*.csv 字节级兼容的一行：flavor|applicationId|palCode|appName。
func (m ManifestChannel) CSVLine() string {
	return strings.Join([]string{m.FlavorName, m.ApplicationID, m.PalCode, m.AppName}, "|")
}

// appConfigBaseURL 返回烧进 bootstrap.json 的运行时配置端点基址。
// APK 用 GET ${base}?appId=<applicationId> 拉「该渠道当前的 web 域名列表」（ADR-0002/0009）。
// 优先 APP_CONFIG_BASE_URL（如 https://api.<域名>/api/app/config）；未设则回退对象存储前缀
// （旧静态快照方案，但那样烧进 APK 的是相对地址、拉不到实时配置、改域名不热更）。
func (s *Service) appConfigBaseURL() string {
	if b := strings.TrimRight(strings.TrimSpace(s.cfg.AppConfigBaseURL), "/"); b != "" {
		return b
	}
	return s.storage.PublicURL("app-config")
}

// BuildManifestForBrand 组装某品牌的构建 manifest。
// scope 是调用者的数据范围（数据权限强制点：GET /build/manifest 按范围裁剪，见
// docs/admin/10-rbac.md）：runner 恒 auth.FullScope()（不受数据权限约束，要拉全量才能构建）；
// 非全量范围的人类账号请求一个不在范围内的品牌时，channels 会被过滤到空（不做硬性 404——
// 品牌枚举本身不敏感，裁剪掉的是渠道明细）。
func (s *Service) BuildManifestForBrand(ctx context.Context, scope auth.Scope, brandCode string) (*BuildManifest, error) {
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

	// 拉取该品牌所有 enabled 渠道（archived/disabled 不下发给构建），按数据范围裁剪。
	chFilter := repo.ChannelFilter{
		BrandCode: brandCode,
		Status:    model.ChannelEnabled,
		PageSize:  500,
	}
	ApplyChannelScope(&chFilter, scope)
	list, _, err := s.repo.ListChannels(ctx, chFilter)
	if err != nil {
		return nil, err
	}

	out := &BuildManifest{
		Brand:         brand.Code,
		Scheme:        brand.Scheme,
		HMSEnabled:    brand.HMSEnabled,
		BrandDomains:  brandDomains,
		ConfigBaseURL: s.appConfigBaseURL(),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for i := range list {
		ch := &list[i]
		eff, err := s.effectiveDomains(ctx, ch.ID)
		if err != nil {
			return nil, err
		}
		adjustToken := ""
		if ch.AdjustAppToken != nil {
			adjustToken = *ch.AdjustAppToken
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
			AdjustAppToken:    adjustToken,
			AdjustEvents:      map[string]string(ch.AdjustEvents),
		})
	}
	return out, nil
}
