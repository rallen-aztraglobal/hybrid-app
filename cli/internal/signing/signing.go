// Package signing 实现 ADR-0016「多签名 key」：一批已上架商店的老渠道
// （ap01018~ap01022、gzmkt031）当年是用另一把 key 签的，商店按包名绑定证书、不能换；
// 而 Gradle 逻辑一行不动（护栏 #1），仍只用默认 key 出包。
//
// 本包提供：
//   - 构建机本地「签名 key 注册表」（properties 风格文件，路径经环境变量 HYBRID_PACK_SIGNING_KEYS
//     告知），只在构建机本地存在，绝不进 git / DB / 配置 API / 对象存储 / 前端（护栏 #4）；
//   - 用 apksigner 对已出包的 APK 按渠道重签（v1+v2）并校验，供 runner 在 assemble 成功后、
//     投递产物前调用。
//
// 密钥物料（keystore 路径、口令）只从注册表文件读取，全程只经子进程环境变量传给 apksigner，
// 绝不出现在命令行参数、日志或错误信息里。
package signing

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

// 注册表文件位置的环境变量与默认路径（ADR-0016 / deploy/Dockerfile.builder）。
const (
	// EnvRegistryPath 指定注册表文件路径；未设置时用 DefaultRegistryPath。
	EnvRegistryPath = "HYBRID_PACK_SIGNING_KEYS"
	// DefaultRegistryPath 是 build-runner 镜像烧入注册表的固定路径。
	DefaultRegistryPath = "/opt/hybrid/signing-keys.properties"
)

// Key 是注册表中一把签名 key 的本地物料。只从构建机本地文件读取，绝不上传/回传后端。
type Key struct {
	// ID 是 Console/manifest 里 channel.signingKey 引用的标识，如 "emptyapp"。
	ID string
	// File 是 keystore 文件的绝对路径（构建机本地/镜像内烧入）。
	File string
	// Alias 是 keystore 内待使用的 key alias。
	Alias string
	// StorePassword 是 keystore（store）口令。
	StorePassword string
	// KeyPassword 是 key 口令。
	KeyPassword string
}

// Registry 是「id → Key」的签名 key 注册表。
type Registry map[string]Key

// Lookup 按 ID 查找一把签名 key。
func (r Registry) Lookup(id string) (Key, bool) {
	k, ok := r[id]
	return k, ok
}

// RegistryPathFromEnv 返回应使用的注册表文件路径：环境变量 EnvRegistryPath 优先，
// 未设置则用 DefaultRegistryPath。
func RegistryPathFromEnv() string {
	if p := strings.TrimSpace(os.Getenv(EnvRegistryPath)); p != "" {
		return p
	}
	return DefaultRegistryPath
}

// Load 从 path 读取签名 key 注册表（properties 风格）。
//
// 格式：每行 `<id>.file=<keystore 绝对路径>`、`<id>.alias=`、`<id>.storePassword=`、
// `<id>.keyPassword=`，'#' 或 '!' 开头为注释行，一个文件可含多个 id（各自四行，顺序不限）。
// 文件不存在视为「空注册表」（非错误）——构建机尚未烧入任何非默认 key 时，所有渠道理应
// 都用默认 key，此时不应因为找不到文件而报错；真正的 fail-closed 发生在调用方 Lookup
// 某个渠道要求的 ID 却查不到时（ADR-0016 决策 4）。
func Load(path string) (Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return nil, fmt.Errorf("读取签名 key 注册表 %s 失败: %w", path, err)
	}
	return parseRegistry(data), nil
}

// parseRegistry 解析 properties 风格内容。未知字段名（非 file/alias/storePassword/
// keyPassword）与不含 '.' 的行会被忽略；同一 id 可缺某些字段（调用方在真正使用前校验完整性）。
func parseRegistry(data []byte) Registry {
	reg := Registry{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		dot := strings.LastIndexByte(key, '.')
		if dot <= 0 || dot == len(key)-1 {
			continue // 无「id.field」结构，忽略
		}
		id := key[:dot]
		field := key[dot+1:]
		k := reg[id]
		k.ID = id
		switch field {
		case "file":
			k.File = val
		case "alias":
			k.Alias = val
		case "storePassword":
			k.StorePassword = val
		case "keyPassword":
			k.KeyPassword = val
		default:
			continue // 未知字段，忽略（不影响其余字段解析）
		}
		reg[id] = k
	}
	return reg
}
