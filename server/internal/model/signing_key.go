// Package model：签名 key 注册表（固定常量表，非数据库表）。
//
// 背景：一批已上架商店的渠道当年用另一把 keystore（CN=empty-app）签名，商店按包名绑定证书、
// 证书不能变；其余渠道用默认 key（release-key，CN=bingo）。Gradle 构建逻辑不动（ADR-0004）——
// 构建机 runner 在打包后按渠道用 apksigner 重签。
//
// keystore 与口令**绝不**进 DB/API/前端（CLAUDE.md 护栏 4）：服务端只存并下发一个 key ID 字符串
// （见 Channel.SigningKey），密钥材料全在构建机镜像里（deploy/Dockerfile.builder）。
//
// 新增一把签名 key：① 这里加一条 SigningKeyInfo；② 构建机镜像烧入对应 keystore。二者缺一不可——
// 只加①而不加②，runner 重签时会因找不到 keystore 而失败；只加②而不加①，后台无法选择该 key
// （校验会拒绝未注册的 signingKey）。
package model

// SigningKeyInfo 一把签名 key 的公开元信息（不含任何密钥材料，仅证书指纹供核对）。
type SigningKeyInfo struct {
	ID         string `json:"id"` // "" = 默认（构建机 Gradle signingConfigs.release 那把）
	Name       string `json:"name"`
	CertSHA1   string `json:"certSha1"`
	CertSHA256 string `json:"certSha256"`
	IsDefault  bool   `json:"isDefault"`
}

// signingKeyRegistry 固定注册表。顺序即前端下拉展示顺序。
var signingKeyRegistry = []SigningKeyInfo{
	{
		ID:         "",
		Name:       "默认（release-key，CN=bingo）",
		CertSHA1:   "c52c6e053310d6d29f990589c7159557332e52b0",
		CertSHA256: "943f7ceda1974b70b83d180572d11cc9856bcbedf3f4272c3a61bb30c8e3060d",
		IsDefault:  true,
	},
	{
		ID:         "emptyapp",
		Name:       "商店老 key（empty-app，2025-09 小米/OPPO 批次）",
		CertSHA1:   "afeaec41d6e41fb1e2da30060505320bceb0b666",
		CertSHA256: "d078768a9801bd78ceb56db5585903a11ae4d6d7bca03bfd5a46bbadc55a6bbe",
		IsDefault:  false,
	},
}

// SigningKeys 返回全部已注册的签名 key（含默认项），供 GET /api/signing-keys 与入参校验使用。
func SigningKeys() []SigningKeyInfo {
	out := make([]SigningKeyInfo, len(signingKeyRegistry))
	copy(out, signingKeyRegistry)
	return out
}

// IsKnownSigningKey 判断 id 是否已注册（空串恒合法，代表默认 key）。
func IsKnownSigningKey(id string) bool {
	if id == "" {
		return true
	}
	for _, k := range signingKeyRegistry {
		if k.ID == id {
			return true
		}
	}
	return false
}
