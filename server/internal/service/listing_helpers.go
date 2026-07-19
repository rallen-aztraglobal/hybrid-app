package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// repoListingFilter 把 handler 传来的原始筛选参数规范化为 repo 过滤器。
func repoListingFilter(platform, status, q string) repo.ListingFilter {
	return repo.ListingFilter{
		Platform: strings.ToLower(strings.TrimSpace(platform)),
		Status:   strings.ToLower(strings.TrimSpace(status)),
		Q:        strings.TrimSpace(q),
	}
}

// normalizePlatform 校验平台枚举并归一化为小写。
func normalizePlatform(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	switch p {
	case model.ListingAndroid, model.ListingIOS:
		return p, nil
	default:
		return "", errBadRequest("platform 仅支持 android / ios")
	}
}

// normalizeTech 校验技术栈枚举，并检查与平台的相容性：
// native_ios 只能用于 ios，native_android 只能用于 android；flutter 两端皆可。
func normalizeTech(raw, platform string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(raw))
	switch t {
	case model.TechFlutter:
		return t, nil
	case model.TechNativeIOS:
		if platform != model.ListingIOS {
			return "", errBadRequest("native_ios 技术栈只能用于 ios 平台")
		}
		return t, nil
	case model.TechNativeAndroid:
		if platform != model.ListingAndroid {
			return "", errBadRequest("native_android 技术栈只能用于 android 平台")
		}
		return t, nil
	default:
		return "", errBadRequest("tech 仅支持 flutter / native_ios / native_android")
	}
}

// normalizeStatus 校验上架包状态枚举。
func normalizeStatus(raw string) (string, error) {
	st := strings.ToLower(strings.TrimSpace(raw))
	switch st {
	case model.ListingEnabled, model.ListingDisabled, model.ListingArchived:
		return st, nil
	default:
		return "", errBadRequest("status 仅支持 enabled / disabled / archived")
	}
}

// normalizeAndValidateAdjust 复用 channel 既有的 Adjust 校验（token 长度 + events 形状），
// 并**返回规范化后的 token**：空串/纯空白 → nil（对应 DB NULL，语义「未绑定/解绑」），
// 与 channel 侧写入口径一致。此前只校验不取规范化结果，导致空串被原样落库为 ""（而非 NULL），
// 让下游按 IS NULL 判断「是否绑定」的消费者误判——这里修正为使用规范化结果。
func normalizeAndValidateAdjust(token *string, events model.AdjustEvents) (*string, error) {
	var normalized *string
	if token != nil {
		n, err := normalizeAdjustAppToken(*token)
		if err != nil {
			return nil, err
		}
		normalized = n
	}
	if len(events) > 0 {
		if err := validateAdjustEvents(map[string]string(events)); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

// normalizeCountries 把国家码去空白、转大写、去重，并拒绝被硬编码强制 A 面的国家（CN/US）。
//
// 拒绝而非静默丢弃：如果运营填了 US 却被我们悄悄删掉，他会以为「配了 US 用户能进 B」，
// 与实际行为（强制 A）背离。直接报错让他明确知道这两个国家不可能进 B 面。
func normalizeCountries(raw []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, c := range raw {
		cc := strings.ToUpper(strings.TrimSpace(c))
		if cc == "" {
			continue
		}
		if len(cc) != 2 {
			return nil, errBadRequest(fmt.Sprintf("国家码 %q 非法（需 ISO-3166-1 两位字母，如 PH）", c))
		}
		if model.IsForcedACountry(cc) {
			return nil, errBadRequest(fmt.Sprintf("国家 %s 被强制走 A 面，不可加入白名单", cc))
		}
		if _, dup := seen[cc]; dup {
			continue
		}
		seen[cc] = struct{}{}
		out = append(out, cc)
	}
	return out, nil
}

// validateCIDRs 校验 CIDR 清单语法，允许「单个 IP」简写（无掩码）。空清单合法。
func validateCIDRs(raw []string, label string) error {
	for _, c := range raw {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(c); err == nil {
			continue
		}
		if net.ParseIP(c) != nil {
			continue
		}
		return errBadRequest(fmt.Sprintf("%s 含非法条目 %q（需 CIDR 如 1.2.3.0/24 或单个 IP）", label, c))
	}
	return nil
}

// trimNonEmpty 去空白并剔除空串，保序。
func trimNonEmpty(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
