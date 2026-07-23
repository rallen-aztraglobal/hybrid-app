package service

import (
	"net"
	"strings"
	"testing"

	"github.com/hybrid-app/server/internal/model"
)

// gate 构造一条规则，简化用例书写。第二参数是「时区黑名单」（命中即强制 A）。
func gate(countries, tzDeny, allow, deny []string) *model.ListingGate {
	return &model.ListingGate{
		Countries:    model.StringList(countries),
		TimezoneDeny: model.StringList(tzDeny),
		IPAllowCIDRs: model.StringList(allow),
		IPDenyCIDRs:  model.StringList(deny),
	}
}

func ip(s string) net.IP { return net.ParseIP(s) }

func TestEvaluateGate(t *testing.T) {
	phOnly := gate([]string{"PH"}, nil, nil, nil)

	cases := []struct {
		name string
		in   GateInput
		want string
	}{
		// —— 放行的唯一路径 ——
		{
			name: "总开关开 + 国家命中 → B",
			in:   GateInput{GateEnabled: true, Gate: phOnly, Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "多国白名单命中其一 → B",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH", "ID"}, nil, nil, nil), Country: "ID", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "国家码大小写不敏感 → B",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"ph"}, nil, nil, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},

		// —— 总开关 ——
		{
			name: "总开关关闭时即便条件全中也 → A",
			in:   GateInput{GateEnabled: false, Gate: phOnly, Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "总开关开但无规则行 → A",
			in:   GateInput{GateEnabled: true, Gate: nil, Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},

		// —— 硬编码强制 A 面（最关键的一组）——
		{
			name: "美国 IP 即便在白名单里也 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH", "US"}, nil, nil, nil), Country: "US", IP: ip("8.8.8.8")},
			want: model.GateModeA,
		},
		{
			name: "中国 IP 即便在白名单里也 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH", "CN"}, nil, nil, nil), Country: "CN", IP: ip("114.114.114.114")},
			want: model.GateModeA,
		},
		{
			name: "美国 IP 即便在 IP 白名单里也 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"US"}, nil, []string{"8.8.8.0/24"}, nil), Country: "US", IP: ip("8.8.8.8")},
			want: model.GateModeA,
		},
		{
			name: "小写 us 同样被强制 A 面",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"us"}, nil, nil, nil), Country: "us", IP: ip("8.8.8.8")},
			want: model.GateModeA,
		},

		// —— IP 黑名单 ——
		{
			name: "命中黑名单 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, nil, []string{"112.198.0.0/16"}), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "黑名单填单个 IP 也生效",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, nil, []string{"112.198.0.1"}), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "未命中黑名单不影响放行",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, nil, []string{"203.0.113.0/24"}), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},

		// —— 国家未知 / 白名单 ——
		{
			name: "国家未知 → A",
			in:   GateInput{GateEnabled: true, Gate: phOnly, Country: "", IP: ip("192.168.1.1")},
			want: model.GateModeA,
		},
		{
			name: "国家不在白名单 → A",
			in:   GateInput{GateEnabled: true, Gate: phOnly, Country: "JP", IP: ip("203.0.113.7")},
			want: model.GateModeA,
		},
		{
			// 关键安全语义：空白名单是配置无效，不是「不限国家」。
			name: "国家白名单为空 → A（不得解释为不限国家）",
			in:   GateInput{GateEnabled: true, Gate: gate(nil, nil, nil, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},

		// —— 时区黑名单（选填，命中即强制 A）——
		{
			name: "时区命中黑名单 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{"America/New_York"}, nil, nil), Country: "PH", Timezone: "America/New_York", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "时区不在黑名单 → B（放行不受影响）",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{"America/New_York"}, nil, nil), Country: "PH", Timezone: "Asia/Manila", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "配了时区黑名单但客户端没上报时区 → B（黑名单只在明确命中时收紧）",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{"America/New_York"}, nil, nil), Country: "PH", Timezone: "", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "时区黑名单大小写/空白不敏感 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{" America/New_York "}, nil, nil), Country: "PH", Timezone: "America/New_York", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},

		// —— IP 白名单（选填）——
		{
			name: "IP 白名单命中 → B",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, []string{"112.198.0.0/16"}, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "IP 不在白名单 → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, []string{"203.0.113.0/24"}, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "配了 IP 白名单但取不到客户端 IP → A",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, []string{"112.198.0.0/16"}, nil), Country: "PH", IP: nil},
			want: model.GateModeA,
		},

		// —— 条件之间是 AND ——
		{
			name: "国家中 + 时区命中黑名单 → A（黑名单短路）",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{"Asia/Tokyo"}, nil, nil), Country: "PH", Timezone: "Asia/Tokyo", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "国家中 + 时区不在黑名单 但 IP 不在白名单 → A（AND 语义）",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, []string{"203.0.113.0/24"}, nil), Country: "PH", Timezone: "Asia/Manila", IP: ip("112.198.0.1")},
			want: model.GateModeA,
		},
		{
			name: "国家中 + 时区不在黑名单 + IP 在白名单 → B",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, []string{"America/New_York"}, []string{"112.198.0.0/16"}, nil), Country: "PH", Timezone: "Asia/Manila", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},

		// —— 健壮性 ——
		{
			name: "非法 CIDR 条目跳过，合法条目仍生效",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{"PH"}, nil, []string{"这不是CIDR", "112.198.0.0/16"}, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
		{
			name: "国家码带空白仍能匹配",
			in:   GateInput{GateEnabled: true, Gate: gate([]string{" PH "}, nil, nil, nil), Country: "PH", IP: ip("112.198.0.1")},
			want: model.GateModeB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateGate(tc.in)
			if got.Mode != tc.want {
				t.Errorf("判定为 %s（原因：%s），期望 %s", got.Mode, got.Reason, tc.want)
			}
			if got.Reason == "" {
				t.Error("判定结果必须带原因，否则线上无法排查")
			}
		})
	}
}

// 强制 A 面的国家清单是安全底线，不允许被配置绕过。
// 这里对 ForcedACountries 的每一项做一次「即便配置全开也必须是 A」的验证。
func TestForcedACountriesCannotBeBypassed(t *testing.T) {
	for country := range model.ForcedACountries {
		t.Run(country, func(t *testing.T) {
			// 构造一份「尽可能放行」的配置：该国在白名单、IP 也在白名单、无黑名单。
			in := GateInput{
				GateEnabled: true,
				Gate:        gate([]string{country}, nil, []string{"0.0.0.0/0"}, nil),
				Country:     country,
				IP:          ip("1.2.3.4"),
			}
			got := EvaluateGate(in)
			if got.Mode != model.GateModeA {
				t.Errorf("%s 必须强制 A 面，实际 %s（原因：%s）", country, got.Mode, got.Reason)
			}
			if !strings.Contains(got.Reason, "强制 A 面") {
				t.Errorf("原因应指明是硬编码强制，实际：%s", got.Reason)
			}
		})
	}
}

// 兜底不变量：GateEnabled=false 时，无论其余输入怎么组合，结果恒为 A。
func TestGateDisabledAlwaysA(t *testing.T) {
	inputs := []GateInput{
		{GateEnabled: false},
		{GateEnabled: false, Gate: gate([]string{"PH"}, nil, []string{"0.0.0.0/0"}, nil), Country: "PH", Timezone: "Asia/Manila", IP: ip("112.198.0.1")},
		{GateEnabled: false, Country: "PH"},
	}
	for i, in := range inputs {
		if got := EvaluateGate(in); got.Mode != model.GateModeA {
			t.Errorf("用例 %d：总开关关闭时必须为 A，实际 %s", i, got.Mode)
		}
	}
}
