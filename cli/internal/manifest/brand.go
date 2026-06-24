package manifest

// 已知大渠道（品牌）元数据。
//
// 仅用于 CLI 的展示与校验（品牌名、顺序），与 app/build.gradle 的 brandConfig
// 以及 package.sh 的 brand_name 保持一致。注意：这不是 Gradle 构建机制的一部分，
// 不影响 flavor 生成；域名/scheme/hms 等真实构建参数仍由 Gradle 与后台负责。
var brandNames = map[string]string{
	"ap": "ArenaPlus",
	"bp": "BingoPlus",
	"gp": "GameZone",
}

// brandOrder 固定品牌展示顺序，与 package.sh 的 "ap bp gp" 一致。
var brandOrder = []string{"ap", "bp", "gp"}

// brandPackagePrefix 是各品牌的 applicationId 包前缀（ADR-0009）。
// applicationId = <prefix>.<flavor>，由派生保证「后缀 == flavor」且全局唯一。
// 与现有 channels/*.csv 的包名前缀一致（com.arenaplus.* / com.bingoplus.* / com.gamezone.*）。
var brandPackagePrefix = map[string]string{
	"ap": "com.arenaplus",
	"bp": "com.bingoplus",
	"gp": "com.gamezone",
}

// BrandPackagePrefix 返回品牌的包前缀；未知品牌返回空串。
func BrandPackagePrefix(code string) string {
	return brandPackagePrefix[code]
}

// DeriveApplicationID 按 ADR-0009 派生 applicationId：<品牌包前缀>.<flavor>。
// 品牌未知或 flavor 为空时返回空串（调用方据此判定无法派生）。
func DeriveApplicationID(brand, flavor string) string {
	prefix := brandPackagePrefix[brand]
	if prefix == "" || flavor == "" {
		return ""
	}
	return prefix + "." + flavor
}

// KnownBrands 返回受支持的品牌代码（有序）。
func KnownBrands() []string {
	out := make([]string, len(brandOrder))
	copy(out, brandOrder)
	return out
}

// IsKnownBrand 判断品牌代码是否受支持。
func IsKnownBrand(code string) bool {
	_, ok := brandNames[code]
	return ok
}

// BrandName 返回品牌展示名；未知品牌回退为原始代码。
func BrandName(code string) string {
	if n, ok := brandNames[code]; ok {
		return n
	}
	return code
}
