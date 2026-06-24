package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hybrid-app/server/internal/domainutil"
	"github.com/hybrid-app/server/internal/model"
)

// ProbeResult 单个域名保存期探测结果（不通也允许保存，仅红色告警）。
type ProbeResult struct {
	URL       string `json:"url"`
	OK        bool   `json:"ok"`
	HTTPCode  int    `json:"httpCode"`
	LatencyMs int    `json:"latencyMs"`
	Note      string `json:"note"`
}

// SetDomainsResult 保存域名后的返回（含探测结果，供前端展示健康提示）。
type SetDomainsResult struct {
	Domains []string      `json:"domains"`
	Probes  []ProbeResult `json:"probes"`
}

// SetBrandDomains 校验并替换品牌默认域名，保存后实时探测一次并触发 CDN 快照重生成。
func (s *Service) SetBrandDomains(ctx context.Context, code string, inputs []domainutil.DomainInput) (*SetDomainsResult, error) {
	brand, err := s.repo.GetBrandByCode(ctx, code)
	if err != nil {
		return nil, errNotFound(fmt.Sprintf("品牌 %q 不存在", code))
	}
	normalized, err := domainutil.Validate(inputs)
	if err != nil {
		return nil, &Error{Code: http.StatusBadRequest, Message: "域名校验失败: " + err.Error()}
	}

	rows := make([]model.BrandDomain, 0, len(normalized))
	for _, n := range normalized {
		rows = append(rows, model.BrandDomain{Position: n.Position, URL: n.URL, Enabled: n.Enabled})
	}
	if err := s.repo.ReplaceBrandDomains(ctx, brand.ID, rows); err != nil {
		return nil, err
	}

	// 保存即探测（异步思想：这里同步但带短超时；不通不阻断保存）。
	probes := s.probeAll(ctx, domainutil.URLs(normalized))

	// 触发受影响渠道的 CDN 配置快照重生成（容错，失败不影响保存）。
	s.regenSnapshotsForBrand(ctx, brand.ID)

	return &SetDomainsResult{Domains: domainutil.URLs(normalized), Probes: probes}, nil
}

// SetChannelDomains 设置小渠道域名覆盖；inputs 为空表示切回继承品牌（use_brand_domains=1）。
func (s *Service) SetChannelDomains(ctx context.Context, channelID uint64, inheritBrand bool, inputs []domainutil.DomainInput) (*SetDomainsResult, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if inheritBrand {
		// 切回继承：清空覆盖 + 置 use_brand_domains=1。
		if err := s.repo.ReplaceChannelDomains(ctx, ch.ID, nil); err != nil {
			return nil, err
		}
		if err := s.repo.UpdateChannelFields(ctx, ch.ID, map[string]any{"use_brand_domains": true}); err != nil {
			return nil, err
		}
		s.regenSnapshot(ctx, ch.ApplicationID)
		// 返回继承到的品牌域名，便于前端展示「当前生效」。
		eff, _ := s.effectiveDomains(ctx, ch.ID)
		return &SetDomainsResult{Domains: eff}, nil
	}

	normalized, err := domainutil.Validate(inputs)
	if err != nil {
		return nil, &Error{Code: http.StatusBadRequest, Message: "域名校验失败: " + err.Error()}
	}
	rows := make([]model.ChannelDomain, 0, len(normalized))
	for _, n := range normalized {
		rows = append(rows, model.ChannelDomain{Position: n.Position, URL: n.URL, Enabled: n.Enabled})
	}
	if err := s.repo.ReplaceChannelDomains(ctx, ch.ID, rows); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateChannelFields(ctx, ch.ID, map[string]any{"use_brand_domains": false}); err != nil {
		return nil, err
	}

	probes := s.probeAll(ctx, domainutil.URLs(normalized))
	s.regenSnapshot(ctx, ch.ApplicationID)
	return &SetDomainsResult{Domains: domainutil.URLs(normalized), Probes: probes}, nil
}

// GetBrandDomains 返回品牌默认域名 URL 清单（按 position 升序）。
func (s *Service) GetBrandDomains(ctx context.Context, code string) ([]string, error) {
	brand, err := s.repo.GetBrandByCode(ctx, code)
	if err != nil {
		return nil, errNotFound(fmt.Sprintf("品牌 %q 不存在", code))
	}
	out := make([]string, 0, len(brand.Domains))
	for _, d := range brand.Domains {
		if d.Enabled {
			out = append(out, d.URL)
		}
	}
	return out, nil
}

// effectiveDomains 计算渠道「实际生效」的域名清单：
// use_brand_domains=1 取品牌默认；否则取渠道覆盖（ADR-0006 的合并继承逻辑）。
func (s *Service) effectiveDomains(ctx context.Context, channelID uint64) ([]string, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if !ch.UseBrandDomains && len(ch.Domains) > 0 {
		out := make([]string, 0, len(ch.Domains))
		for _, d := range ch.Domains {
			if d.Enabled {
				out = append(out, d.URL)
			}
		}
		return out, nil
	}
	// 继承品牌。
	brand, err := s.repo.GetBrandByID(ctx, ch.BrandID)
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

// probeAll 对若干 URL 做保存期探测（短超时、容错）。
func (s *Service) probeAll(ctx context.Context, urls []string) []ProbeResult {
	results := make([]ProbeResult, 0, len(urls))
	client := &http.Client{Timeout: 3 * time.Second}
	for _, u := range urls {
		results = append(results, probeOne(ctx, client, u, s.cfg.DefaultProbePath))
	}
	return results
}

// probeOne 探测单个域名的健康端点。可达但非 200 也记录，便于前端红色告警。
func probeOne(ctx context.Context, client *http.Client, baseURL, probePath string) ProbeResult {
	start := time.Now()
	url := baseURL + probePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{URL: baseURL, OK: false, Note: "构造请求失败: " + err.Error()}
	}
	resp, err := client.Do(req)
	lat := int(time.Since(start).Milliseconds())
	if err != nil {
		return ProbeResult{URL: baseURL, OK: false, LatencyMs: lat, Note: "不可达: " + err.Error()}
	}
	defer resp.Body.Close()
	ok := resp.StatusCode == http.StatusOK
	note := ""
	if !ok {
		note = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return ProbeResult{URL: baseURL, OK: ok, HTTPCode: resp.StatusCode, LatencyMs: lat, Note: note}
}
