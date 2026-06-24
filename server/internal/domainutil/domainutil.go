// Package domainutil 实现域名清单的保存期校验（docs/admin/01 §5.7 + ADR-0003 的前提）。
// 规则：必须 https；主域名必填；备用 0~3 个（共 ≤4）；去重；URL 可解析为合法 host。
// 这是 APK 端容灾可靠的前提——脏域名进库会让 APK「乱换」。
package domainutil

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// MaxDomains 主 + 备用合计上限。
const MaxDomains = 4

// DomainInput 是一条待保存的域名（position 0=主，1..3=备用）。
type DomainInput struct {
	Position int    `json:"position"`
	URL      string `json:"url"`
	Enabled  bool   `json:"enabled"`
}

// Normalized 是规范化后的域名。
type Normalized struct {
	Position int
	URL      string // 规范化后的 https URL（去尾斜杠、host 小写）
	Host     string
	Enabled  bool
}

// ValidationError 聚合一次校验里的所有问题，便于一次性返回给前端。
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Issues, "; ")
}

// Validate 校验并规范化一组域名。返回按 position 升序的规范化清单。
// 不做网络探测（探测在 service 层异步做，不通也允许保存但告警）——本函数只做静态规则。
func Validate(inputs []DomainInput) ([]Normalized, error) {
	ve := &ValidationError{}

	if len(inputs) == 0 {
		ve.Issues = append(ve.Issues, "至少需要 1 个主域名")
		return nil, ve
	}
	if len(inputs) > MaxDomains {
		ve.Issues = append(ve.Issues, fmt.Sprintf("域名数量 %d 超过上限 %d（主+最多3备用）", len(inputs), MaxDomains))
	}

	out := make([]Normalized, 0, len(inputs))
	seenHost := map[string]bool{}
	seenPos := map[int]bool{}
	hasPrimary := false

	for i, in := range inputs {
		label := fmt.Sprintf("第%d个域名", i+1)
		raw := strings.TrimSpace(in.URL)
		if raw == "" {
			ve.Issues = append(ve.Issues, label+": URL 不能为空")
			continue
		}

		n, err := normalizeOne(raw)
		if err != nil {
			ve.Issues = append(ve.Issues, fmt.Sprintf("%s(%s): %v", label, raw, err))
			continue
		}
		n.Position = in.Position
		n.Enabled = in.Enabled

		if in.Position < 0 || in.Position > MaxDomains-1 {
			ve.Issues = append(ve.Issues, fmt.Sprintf("%s: position %d 越界（应为 0..%d）", label, in.Position, MaxDomains-1))
		}
		if seenPos[in.Position] {
			ve.Issues = append(ve.Issues, fmt.Sprintf("%s: position %d 重复", label, in.Position))
		}
		seenPos[in.Position] = true

		if seenHost[n.Host] {
			ve.Issues = append(ve.Issues, fmt.Sprintf("%s: 域名 %s 重复", label, n.Host))
			continue
		}
		seenHost[n.Host] = true

		if in.Position == 0 {
			hasPrimary = true
		}
		out = append(out, n)
	}

	if !hasPrimary {
		ve.Issues = append(ve.Issues, "缺少主域名（position=0）")
	}

	if len(ve.Issues) > 0 {
		return nil, ve
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

// normalizeOne 校验并规范化单个 URL。
func normalizeOne(raw string) (Normalized, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Normalized{}, fmt.Errorf("不是合法 URL: %w", err)
	}
	if u.Scheme != "https" {
		// 线上一律 https（usesCleartextTraffic 虽开着，但配置必须 https）。
		return Normalized{}, fmt.Errorf("必须以 https:// 开头（当前 scheme=%q）", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return Normalized{}, fmt.Errorf("缺少域名 host")
	}
	if !isPlausibleHost(host) {
		return Normalized{}, fmt.Errorf("域名 %q 非法或不可解析格式", host)
	}
	// 规范化：host 小写、去掉末尾斜杠、丢弃 query/fragment（域名清单只保留 scheme+host[:port]）。
	port := u.Port()
	normHost := strings.ToLower(host)
	rebuilt := "https://" + normHost
	if port != "" {
		rebuilt += ":" + port
	}
	return Normalized{URL: rebuilt, Host: normHost}, nil
}

// isPlausibleHost 判断是否为合法 IP 或形如 a.b.c 的域名（不真正发 DNS）。
func isPlausibleHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	if len(host) > 253 {
		return false
	}
	// 必须含点（顶级单 label 如 "localhost" 不接受为线上域名），且每段合法。
	if !strings.Contains(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, l := range labels {
		if l == "" || len(l) > 63 {
			return false
		}
		for i := 0; i < len(l); i++ {
			ch := l[i]
			isAlnum := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
			if !isAlnum && ch != '-' {
				return false
			}
		}
		if l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
	}
	// 末段（TLD）应为字母。
	tld := labels[len(labels)-1]
	for i := 0; i < len(tld); i++ {
		ch := tld[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')) {
			return false
		}
	}
	return true
}

// URLs 提取规范化清单里的 URL（按 position 升序），供 /api/app/config 直接返回。
func URLs(ns []Normalized) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		if n.Enabled {
			out = append(out, n.URL)
		}
	}
	return out
}
