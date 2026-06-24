package domainutil

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		inputs    []DomainInput
		wantErr   bool
		wantURLs  []string // 期望规范化后的（按 position）URL，仅在 !wantErr 时校验
	}{
		{
			name: "正常主+2备用",
			inputs: []DomainInput{
				{Position: 0, URL: "https://arenaplus.ph", Enabled: true},
				{Position: 1, URL: "https://arenaplus-cdn.com/", Enabled: true},
				{Position: 2, URL: "https://AP-Backup.net", Enabled: true},
			},
			wantURLs: []string{"https://arenaplus.ph", "https://arenaplus-cdn.com", "https://ap-backup.net"},
		},
		{
			name:    "空清单被拒",
			inputs:  nil,
			wantErr: true,
		},
		{
			name: "缺主域名被拒",
			inputs: []DomainInput{
				{Position: 1, URL: "https://a.com", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "非 https 被拒",
			inputs: []DomainInput{
				{Position: 0, URL: "http://insecure.com", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "超过4个被拒",
			inputs: []DomainInput{
				{Position: 0, URL: "https://a.com", Enabled: true},
				{Position: 1, URL: "https://b.com", Enabled: true},
				{Position: 2, URL: "https://c.com", Enabled: true},
				{Position: 3, URL: "https://d.com", Enabled: true},
				{Position: 4, URL: "https://e.com", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "重复域名被拒",
			inputs: []DomainInput{
				{Position: 0, URL: "https://same.com", Enabled: true},
				{Position: 1, URL: "https://SAME.com", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "重复position被拒",
			inputs: []DomainInput{
				{Position: 0, URL: "https://a.com", Enabled: true},
				{Position: 0, URL: "https://b.com", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "无效host被拒",
			inputs: []DomainInput{
				{Position: 0, URL: "https://nodot", Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "合法IP主域名通过",
			inputs: []DomainInput{
				{Position: 0, URL: "https://203.0.113.5", Enabled: true},
			},
			wantURLs: []string{"https://203.0.113.5"},
		},
		{
			name: "带端口规范化保留",
			inputs: []DomainInput{
				{Position: 0, URL: "https://Example.com:8443", Enabled: true},
			},
			wantURLs: []string{"https://example.com:8443"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate(tt.inputs)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望出错，但通过了: %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应出错: %v", err)
			}
			urls := URLs(got)
			if len(urls) != len(tt.wantURLs) {
				t.Fatalf("URL 数量 = %d，期望 %d (%v)", len(urls), len(tt.wantURLs), urls)
			}
			for i := range urls {
				if urls[i] != tt.wantURLs[i] {
					t.Errorf("URL[%d] = %q，期望 %q", i, urls[i], tt.wantURLs[i])
				}
			}
		})
	}
}

func TestValidateDisabledExcludedFromURLs(t *testing.T) {
	got, err := Validate([]DomainInput{
		{Position: 0, URL: "https://main.com", Enabled: true},
		{Position: 1, URL: "https://backup.com", Enabled: false},
	})
	if err != nil {
		t.Fatalf("不应出错: %v", err)
	}
	urls := URLs(got)
	if len(urls) != 1 || urls[0] != "https://main.com" {
		t.Fatalf("disabled 域名应被 URLs 过滤，得到 %v", urls)
	}
}
