// Package service — 上架包 AB 面网关判定。
//
// 本文件只放「纯判定」：输入是已经解析好的国家/时区/IP 与规则，输出是 A 还是 B，不做任何 I/O。
// 这样 8 步判定顺序可以被穷举测试，而不需要起 DB 或造 HTTP 请求。
// I/O 编排（查库、查 GeoIP、落判定日志、取域名）在 listing.go 的 EvaluateListingGate 里。
package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/hybrid-app/server/internal/model"
)

// GateInput 是判定所需的全部输入，全部已解析完毕。
type GateInput struct {
	// GateEnabled 取自 ListingApp.GateEnabled（AB 面总开关）。
	GateEnabled bool
	// Gate 是该上架包的放行规则；为 nil 表示未配置，判 A。
	Gate *model.ListingGate
	// Country 是服务端按请求 IP 查 GeoIP 得到的国家码；空串 = 未知。
	Country string
	// Timezone 是客户端上报的 IANA 时区名；可伪造，只作收紧条件。
	Timezone string
	// IP 是提取出的真实客户端 IP；可能为 nil（RemoteAddr 非法等）。
	IP net.IP
}

// GateDecision 是判定结果。Reason 只用于运维排查与落日志，不返回给客户端——
// 把「你为什么没进 B 面」告诉客户端等于把规则泄露给审核方。
type GateDecision struct {
	Mode   string
	Reason string
}

func decideA(format string, args ...any) GateDecision {
	return GateDecision{Mode: model.GateModeA, Reason: fmt.Sprintf(format, args...)}
}

// EvaluateGate 执行 AB 面判定。
//
// 判定顺序与短路语义见 model.ListingGate 的文档（那里是唯一事实来源）。
// 核心不变量：**任何不确定的情形都返回 A**。新增分支时务必保持这一点——
// 判错成 A 只是少一个用户进 B 面，判错成 B 可能让审核员看到 B 面内容。
func EvaluateGate(in GateInput) GateDecision {
	// 1. 总开关。关闭时规则字段一律无意义，直接返回。
	if !in.GateEnabled {
		return decideA("AB 面总开关未开启")
	}

	// 2. 规则缺失。gate_enabled=true 却没有规则行属于异常状态，按 A 处理。
	if in.Gate == nil {
		return decideA("未配置放行规则")
	}

	country := strings.ToUpper(strings.TrimSpace(in.Country))

	// 3. 硬编码强制 A 面的国家（CN/US），无视任何配置。
	if model.IsForcedACountry(country) {
		return decideA("国家 %s 被硬编码强制 A 面", country)
	}

	// 4. IP 黑名单：命中即强制 A 面（钉死已知审核机房网段）。
	if hit, cidr := matchCIDR(in.IP, in.Gate.IPDenyCIDRs); hit {
		return decideA("命中 IP 黑名单 %s", cidr)
	}

	// 5. 国家未知（GeoIP 未加载、私有地址、库里查不到）→ fail-closed。
	if country == "" {
		return decideA("无法解析出国家")
	}

	// 6. 国家白名单。空清单属于配置无效（Countries 是必填项），同样判 A，
	//    绝不解释成「不限国家」——那会让一次误删配置变成全量放行。
	if len(in.Gate.Countries) == 0 {
		return decideA("国家白名单为空（配置无效）")
	}
	if !containsFold(in.Gate.Countries, country) {
		return decideA("国家 %s 不在白名单内", country)
	}

	// 7. 时区白名单（选填）。非空才参与判定。
	if len(in.Gate.Timezones) > 0 {
		tz := strings.TrimSpace(in.Timezone)
		if tz == "" {
			return decideA("已配置时区白名单但客户端未上报时区")
		}
		if !containsFold(in.Gate.Timezones, tz) {
			return decideA("时区 %s 不在白名单内", tz)
		}
	}

	// 8. IP 白名单（选填）。非空时在国家白名单之上再收紧一层。
	if len(in.Gate.IPAllowCIDRs) > 0 {
		if in.IP == nil {
			return decideA("已配置 IP 白名单但无法取得客户端 IP")
		}
		if hit, _ := matchCIDR(in.IP, in.Gate.IPAllowCIDRs); !hit {
			return decideA("IP %s 不在白名单内", in.IP)
		}
	}

	// 9. 全部条件通过。
	return GateDecision{Mode: model.GateModeB, Reason: fmt.Sprintf("国家 %s 命中全部放行条件", country)}
}

// matchCIDR 判断 ip 是否落在任一 CIDR 内，返回命中的那条便于写进判定日志。
// 无法解析的 CIDR 条目跳过：配置里一条写错不应让整个清单失效。
// 注意 ip 为 nil 时一律不命中——对黑名单而言这是「不拦」，但后续的未知国家分支会兜住。
func matchCIDR(ip net.IP, cidrs model.StringList) (bool, string) {
	if ip == nil {
		return false, ""
	}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			// 兼容运营直接填单个 IP（不带掩码）的写法。
			if single := net.ParseIP(c); single != nil && single.Equal(ip) {
				return true, c
			}
			continue
		}
		if n.Contains(ip) {
			return true, c
		}
	}
	return false, ""
}

// containsFold 判断 list 中是否存在与 target 忽略大小写相等的项（两侧去空白）。
// 国家码与时区名在录入时可能大小写不一，判定时不该因此漏放。
func containsFold(list model.StringList, target string) bool {
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return true
		}
	}
	return false
}
