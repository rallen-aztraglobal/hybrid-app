package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// buildListingBSideURL 把参考渠道包 PAL_CODE 与自定义附加参数拼到 B 面主域名后，
// palcode 段与渠道 APK 启动加载 ${DOMAIN}/?palcode=${PAL_CODE} 同一口径，附加参数紧随其后。
// 客户端拿到即原样打开，不自行拼参（客户端零内置 B 面配置）。
// extraQuery 须是 normalizeListingExtraQuery 规范化后的值（已转义、无前导 ?），此处原样拼接。
// 两者都为空（历史行 AutoMigrate 默认空串）时原样返回域名，不产生悬空的 ?。
func buildListingBSideURL(domain, palCode, extraQuery string) string {
	parts := make([]string, 0, 2)
	if palCode != "" {
		parts = append(parts, "palcode="+url.QueryEscape(palCode))
	}
	if extraQuery != "" {
		parts = append(parts, extraQuery)
	}
	if len(parts) == 0 {
		return domain
	}
	return strings.TrimRight(domain, "/") + "/?" + strings.Join(parts, "&")
}

// listingExtraQueryMaxLen 与 listing_app.extra_query 列宽一致。
const listingExtraQueryMaxLen = 1024

// normalizeListingExtraQuery 校验并规范化 B 面自定义附加参数。
//
// 运营直接从 ? 开始手输（如 "?channel=fb&lang=en"），前导 ?/& 可有可无。规范化规则：
//   - 去空白与前导 ?，丢弃空段（"a=1&&b=2&" → "a=1&b=2"），保留运营输入的参数顺序；
//   - query 中合法的字符（含 / : = 等）原样保留，空格、中文、# 等非法字符按 UTF-8 转义，
//     保证下发的 URL 两端客户端都能直接解析；已有的 %XX 转义须合法，否则报错；
//   - 参数名不能为空，且不能是 palcode（大小写不敏感）——palcode 只由 PAL_CODE 字段提供，
//     重复出现时 B 面 web 取哪个值不确定。
//
// 空串合法，表示不加附加参数。
func normalizeListingExtraQuery(raw string) (string, error) {
	s := strings.TrimLeft(strings.TrimSpace(raw), "?&")
	if s == "" {
		return "", nil
	}
	pairs := make([]string, 0, 4)
	for _, seg := range strings.Split(s, "&") {
		if seg == "" {
			continue
		}
		escaped, err := escapeQuerySegment(seg)
		if err != nil {
			return "", errBadRequest(fmt.Sprintf("B 面附加参数 %q 有误：%s", seg, err.Error()))
		}
		rawKey, _, _ := strings.Cut(escaped, "=")
		key, err := url.QueryUnescape(rawKey)
		if err != nil {
			return "", errBadRequest(fmt.Sprintf("B 面附加参数 %q 有误：参数名转义不合法", seg))
		}
		if strings.TrimSpace(key) == "" {
			return "", errBadRequest(fmt.Sprintf("B 面附加参数 %q 缺少参数名", seg))
		}
		if strings.EqualFold(strings.TrimSpace(key), "palcode") {
			return "", errBadRequest("B 面附加参数里不能再写 palcode，请填在「参考渠道包 PAL_CODE」字段")
		}
		pairs = append(pairs, escaped)
	}
	out := strings.Join(pairs, "&")
	if len(out) > listingExtraQueryMaxLen {
		return "", errBadRequest(fmt.Sprintf("B 面附加参数过长（转义后 %d 字符，上限 %d）", len(out), listingExtraQueryMaxLen))
	}
	return out, nil
}

// escapeQuerySegment 对一个 key=value 段做「最小转义」：RFC 3986 query 允许的字符原样保留，
// 其余字节（空格、非 ASCII、# 等）转成 %XX；已有的 % 必须跟两位十六进制，否则报错。
// '&' 已被调用方用作分隔符，不会出现在 seg 中。
func escapeQuerySegment(seg string) (string, error) {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		switch {
		case c == '%':
			if i+2 >= len(seg) || !isHexDigit(seg[i+1]) || !isHexDigit(seg[i+2]) {
				return "", errors.New("% 后须跟两位十六进制（要表示 % 本身请写 %25）")
			}
			b.WriteString(seg[i : i+3])
			i += 2
		case isQueryChar(c):
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
		}
	}
	return b.String(), nil
}

// isQueryChar 判断字节是否属于 RFC 3986 query 可直接出现的字符（unreserved / sub-delims / : @ / ?）。
func isQueryChar(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return strings.IndexByte("-._~!$'()*+,;=:@/?", c) >= 0
}

func isHexDigit(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

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
