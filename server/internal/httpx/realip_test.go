package httpx

import (
	"net/http"
	"testing"
)

// req 构造一个带 RemoteAddr 与若干 XFF 头的请求。
func req(remoteAddr string, xff ...string) *http.Request {
	r := &http.Request{
		RemoteAddr: remoteAddr,
		Header:     http.Header{},
	}
	for _, v := range xff {
		r.Header.Add("X-Forwarded-For", v)
	}
	return r
}

func TestClientIP(t *testing.T) {
	trusted := NewTrustedProxies(nil) // 默认私有网段

	cases := []struct {
		name       string
		remoteAddr string
		xff        []string
		want       string
	}{
		{
			name:       "无代理直连，无 XFF",
			remoteAddr: "203.0.113.7:51234",
			want:       "203.0.113.7",
		},
		{
			// 攻击面核心：请求没经过我们的 nginx（对端是公网地址），
			// 此时 XFF 是攻击者自己写的，必须完全忽略。
			name:       "直连时伪造 XFF 必须被忽略",
			remoteAddr: "203.0.113.7:51234",
			xff:        []string{"1.2.3.4"},
			want:       "203.0.113.7",
		},
		{
			name:       "经一层可信 nginx",
			remoteAddr: "127.0.0.1:8080",
			xff:        []string{"203.0.113.7"},
			want:       "203.0.113.7",
		},
		{
			// 客户端伪造了链最左侧的一段，真实对端由 nginx 追加在右侧。
			// 右→左扫描应停在 203.0.113.7，而不是采信伪造的 1.2.3.4。
			name:       "经可信代理但客户端伪造了左侧条目",
			remoteAddr: "127.0.0.1:8080",
			xff:        []string{"1.2.3.4, 203.0.113.7"},
			want:       "203.0.113.7",
		},
		{
			name:       "多层可信代理，取最右侧的不可信地址",
			remoteAddr: "10.0.0.5:8080",
			xff:        []string{"203.0.113.7, 10.0.0.9, 172.16.0.3"},
			want:       "203.0.113.7",
		},
		{
			name:       "多个同名 XFF 头按序展开",
			remoteAddr: "127.0.0.1:8080",
			xff:        []string{"1.2.3.4", "203.0.113.7, 10.0.0.9"},
			want:       "203.0.113.7",
		},
		{
			name:       "XFF 全是可信地址，回落 RemoteAddr",
			remoteAddr: "127.0.0.1:8080",
			xff:        []string{"10.0.0.9, 192.168.1.1"},
			want:       "127.0.0.1",
		},
		{
			name:       "可信代理但无 XFF，回落 RemoteAddr",
			remoteAddr: "127.0.0.1:8080",
			want:       "127.0.0.1",
		},
		{
			name:       "XFF 含非法条目应跳过",
			remoteAddr: "127.0.0.1:8080",
			xff:        []string{"not-an-ip, 203.0.113.7, garbage"},
			want:       "203.0.113.7",
		},
		{
			name:       "IPv6 客户端经可信代理",
			remoteAddr: "[::1]:8080",
			xff:        []string{"2001:db8::1"},
			want:       "2001:db8::1",
		},
		{
			name:       "IPv6 直连带方括号端口",
			remoteAddr: "[2001:db8::2]:51234",
			want:       "2001:db8::2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClientIP(req(tc.remoteAddr, tc.xff...), trusted)
			if got == nil {
				t.Fatalf("ClientIP 返回 nil，期望 %s", tc.want)
			}
			if got.String() != tc.want {
				t.Errorf("ClientIP = %s，期望 %s", got, tc.want)
			}
		})
	}
}

func TestClientIPMalformedRemoteAddr(t *testing.T) {
	trusted := NewTrustedProxies(nil)
	// RemoteAddr 解析不出 IP 时返回 nil，上层网关据此判未知国家 → A 面。
	if got := ClientIP(req("garbage"), trusted); got != nil {
		t.Errorf("非法 RemoteAddr 应返回 nil，实际 %s", got)
	}
}

func TestTrustedProxiesCustomCIDR(t *testing.T) {
	// 自定义可信网段：把一个公网段声明为可信代理（例如前置了云厂商 LB）。
	trusted := NewTrustedProxies([]string{"198.51.100.0/24"})
	got := ClientIP(req("198.51.100.10:8080", "203.0.113.7"), trusted)
	if got == nil || got.String() != "203.0.113.7" {
		t.Errorf("自定义可信网段未生效，得到 %v", got)
	}
	// 默认私有网段此时不再可信（已被自定义列表整体替换），XFF 应被忽略。
	got = ClientIP(req("127.0.0.1:8080", "203.0.113.7"), trusted)
	if got == nil || got.String() != "127.0.0.1" {
		t.Errorf("自定义列表应替换默认值，得到 %v", got)
	}
}

func TestTrustedProxiesIgnoresInvalidCIDR(t *testing.T) {
	// 配置写错的条目跳过即可，不应 panic，也不应让其余条目失效。
	trusted := NewTrustedProxies([]string{"not-a-cidr", "10.0.0.0/8"})
	got := ClientIP(req("10.0.0.5:8080", "203.0.113.7"), trusted)
	if got == nil || got.String() != "203.0.113.7" {
		t.Errorf("合法条目应仍生效，得到 %v", got)
	}
}
