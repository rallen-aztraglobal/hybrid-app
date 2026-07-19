package httpx

import (
	"net"
	"net/http"
	"strings"
)

// 默认可信代理网段：回环 + RFC1918 私有网段 + RFC4193 IPv6 唯一本地地址 + IPv6 回环。
// 部署形态是 nginx 与 go-api 同机 docker compose（ADR-0010），代理必然来自这些网段。
var defaultTrustedCIDRs = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

// TrustedProxies 判断某个地址是否是我们自己的反向代理。
type TrustedProxies struct {
	nets []*net.IPNet
}

// NewTrustedProxies 解析 CIDR 列表；传空则用默认私有网段集合。
// 无法解析的条目直接跳过（配置写错不应让服务起不来，但会因此少信任一个代理，
// 后果是取到代理 IP 而非真实客户端 IP —— 偏保守，可接受）。
func NewTrustedProxies(cidrs []string) *TrustedProxies {
	if len(cidrs) == 0 {
		cidrs = defaultTrustedCIDRs
	}
	tp := &TrustedProxies{}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, n, err := net.ParseCIDR(c); err == nil {
			tp.nets = append(tp.nets, n)
		}
	}
	return tp
}

// Contains 判断 ip 是否落在任一可信网段内。
func (t *TrustedProxies) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, n := range t.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP 提取请求的真实客户端 IP。
//
// X-Forwarded-For 是客户端可以随便伪造的——攻击者直接发
// `X-Forwarded-For: 1.2.3.4`（伪装成菲律宾 IP）就能骗过朴素实现。
// 因此不能简单取 XFF 的第一段，必须结合「直连对端是否可信」来判断：
//
//  1. 直连对端（RemoteAddr）不在可信代理网段内 → 说明请求没经过我们的 nginx，
//     XFF 完全不可信，直接用 RemoteAddr，忽略所有头。
//  2. 直连对端可信 → 从 XFF **最右往左**扫，返回第一个不可信的地址。
//     右侧是离我们最近、由我们自己的代理逐跳追加的，可信；一旦遇到不可信地址，
//     说明它是最外层代理看到的真实对端，即客户端 IP。左侧更早的条目可能是客户端伪造的。
//  3. XFF 全是可信地址（或为空）→ 回落到 RemoteAddr。
//
// 这样即使客户端伪造 XFF，伪造值只会出现在链的左侧，扫描在遇到它之前就已停在
// 我们代理记录的真实对端上，伪造无效。
func ClientIP(r *http.Request, trusted *TrustedProxies) net.IP {
	remote := parseIP(r.RemoteAddr)

	// 情况 1：直连对端不可信，XFF 一律不采信。
	if !trusted.Contains(remote) {
		return remote
	}

	// 情况 2：逐跳右→左扫描。
	hops := forwardedHops(r)
	for i := len(hops) - 1; i >= 0; i-- {
		ip := parseIP(hops[i])
		if ip == nil {
			continue
		}
		if !trusted.Contains(ip) {
			return ip
		}
	}

	// 情况 3：没有可用的 XFF 条目。
	return remote
}

// forwardedHops 把可能出现多次的 X-Forwarded-For 头按出现顺序展开成一维列表。
// 多个代理各加一个头、或一个头里逗号分隔，两种形态都要覆盖。
func forwardedHops(r *http.Request) []string {
	var hops []string
	for _, h := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(h, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				hops = append(hops, part)
			}
		}
	}
	return hops
}

// parseIP 解析可能带端口、可能带 IPv6 方括号的地址串。
func parseIP(s string) net.IP {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// 优先按 host:port 拆；失败说明本身就是裸地址。
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	s = strings.Trim(s, "[]")
	return net.ParseIP(s)
}
