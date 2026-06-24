package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hybrid-app/cli/internal/repo"
)

// 签名材料的环境变量名（ADR-0008：keystore 只作构建机 secret，从环境/挂载注入）。
// 注意：CLI 绝不打印这些值，也绝不把它们放进任何回传请求体。
const (
	EnvKeystoreFile     = "HYBRID_PACK_KEYSTORE_FILE"     // keystore 文件路径（构建机本地/挂载）
	EnvKeystorePassword = "HYBRID_PACK_KEYSTORE_PASSWORD" // store 口令
	EnvKeyAlias         = "HYBRID_PACK_KEY_ALIAS"         // key alias
	EnvKeyPassword      = "HYBRID_PACK_KEY_PASSWORD"      // key 口令
)

// Keystore 是从环境读取的签名材料（绝不上传、绝不打印口令值）。
type Keystore struct {
	File          string
	StorePassword string
	KeyAlias      string
	KeyPassword   string
}

// KeystoreFromEnv 从环境变量读取签名材料；任一关键项缺失则返回 nil（表示「环境未提供」，
// 此时依赖构建机已有的 local.properties）。仅在四项齐全时才返回非 nil。
func KeystoreFromEnv() *Keystore {
	ks := &Keystore{
		File:          strings.TrimSpace(os.Getenv(EnvKeystoreFile)),
		StorePassword: os.Getenv(EnvKeystorePassword),
		KeyAlias:      strings.TrimSpace(os.Getenv(EnvKeyAlias)),
		KeyPassword:   os.Getenv(EnvKeyPassword),
	}
	if ks.File == "" || ks.StorePassword == "" || ks.KeyAlias == "" || ks.KeyPassword == "" {
		return nil
	}
	return ks
}

// signingKeys 是 app/build.gradle signingConfigs.release 读取的 4 个 local.properties 键。
var signingKeys = []string{"KEYSTORE_FILE", "KEYSTORE_PASSWORD", "KEY_ALIAS", "KEY_PASSWORD"}

// ensureSigning 确保 release 签名可用（不改 app/build.gradle，只准备它读取的 local.properties）。
//
//   - 若 opt.Keystore 非 nil（环境提供了 secret）：把 4 个签名键写入/合并进 local.properties
//     （保留既有 sdk.dir 等其他键），并校验 keystore 文件存在。
//   - 若 opt.Keystore 为 nil：校验构建机已有的 local.properties 含齐 4 个签名键且 keystore 文件存在。
//
// 任一情况不满足都返回错误，拒绝启动 runner——避免打出无法签名的 release 任务。
// 全程不打印任何口令值。
func ensureSigning(r *repo.Repo, opt Options) error {
	lp := r.LocalProperties()

	if opt.Keystore != nil {
		if err := materializeLocalProperties(lp, opt.Keystore); err != nil {
			return fmt.Errorf("写入签名配置失败: %w", err)
		}
		opt.logf("已从环境注入签名配置到 local.properties（keystore: %s，口令未打印）", filepath.Base(opt.Keystore.File))
		return verifyKeystoreFile(r, opt.Keystore.File)
	}

	// 无环境 secret：依赖构建机既有 local.properties。
	props := readProps(lp)
	var missing []string
	for _, k := range signingKeys {
		if strings.TrimSpace(props[k]) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("缺少签名配置：未设置 %s 等环境变量，且 local.properties 缺少 %s（ADR-0008：keystore 须作为构建机 secret 注入）",
			EnvKeystoreFile, strings.Join(missing, ", "))
	}
	return verifyKeystoreFile(r, props["KEYSTORE_FILE"])
}

// verifyKeystoreFile 校验 keystore 文件可定位（相对路径以仓库根为基准，与 Gradle file() 一致）。
func verifyKeystoreFile(r *repo.Repo, ksPath string) error {
	p := ksPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.Root, p)
	}
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("keystore 文件未找到: %s（请检查注入路径）", ksPath)
	}
	return nil
}

// materializeLocalProperties 把签名键写入 local.properties（合并保留其他键，如 sdk.dir）。
// 文件权限 0600（含口令，按机密处理）。
func materializeLocalProperties(path string, ks *Keystore) error {
	props := readProps(path)
	// 注入/覆盖签名键。
	props["KEYSTORE_FILE"] = ks.File
	props["KEYSTORE_PASSWORD"] = ks.StorePassword
	props["KEY_ALIAS"] = ks.KeyAlias
	props["KEY_PASSWORD"] = ks.KeyPassword

	// 稳定顺序输出：先 sdk.dir（若有），再签名键，再其余键。
	var b strings.Builder
	b.WriteString("# 由 hybrid-pack runner 注入（含签名口令，机密文件，勿入库）。\n")
	written := map[string]bool{}
	emit := func(k string) {
		if v, ok := props[k]; ok && !written[k] {
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(v)
			b.WriteByte('\n')
			written[k] = true
		}
	}
	emit("sdk.dir")
	for _, k := range signingKeys {
		emit(k)
	}
	for k := range props {
		emit(k)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// readProps 读取 .properties（key=value，# / ! 注释行跳过）。只取值不解释语义。
func readProps(path string) map[string]string {
	out := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 0 {
			continue
		}
		out[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
	}
	return out
}
