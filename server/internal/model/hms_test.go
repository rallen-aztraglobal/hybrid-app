package model

import "testing"

// TestEffectiveHMSEnabled 覆盖渠道级 HMS 开关的三态语义：
// 显式配置优先；未配置（NULL）回落到与 app/build.gradle 一致的默认规则
// 「品牌整体开 HMS，或 flavor 以 _hw 结尾的华为商店包」。
func TestEffectiveHMSEnabled(t *testing.T) {
	on, off := true, false
	cases := []struct {
		name     string
		flavor   string
		explicit *bool
		brandHMS bool
		want     bool
	}{
		{"未配置+非华为包+品牌不开 → 不集成", "ap01036", nil, false, false},
		{"未配置+华为商店包 → 集成", "johnc2010092_hw", nil, false, true},
		{"未配置+品牌整体开（bp） → 集成", "bp001", nil, true, true},
		{"显式开+非华为后缀（上架华为商店的老渠道） → 集成", "ap01018", &on, false, true},
		{"显式关+品牌整体开 → 不集成（覆盖默认规则）", "bp001", &off, true, false},
		{"显式关+华为商店包 → 不集成", "johnc2010092_hw", &off, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ch := &Channel{FlavorName: c.flavor, HMSEnabled: c.explicit}
			if got := ch.EffectiveHMSEnabled(c.brandHMS); got != c.want {
				t.Errorf("EffectiveHMSEnabled(%v) = %v，期望 %v", c.brandHMS, got, c.want)
			}
		})
	}
}
