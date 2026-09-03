package signing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadParsesMultipleKeysWithCommentsAndMissingFields 验证注册表解析：
// 多个 id、注释行（# / !）、以及某 id 缺字段时仍能正常解析出已给出的字段。
func TestLoadParsesMultipleKeysWithCommentsAndMissingFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "signing-keys.properties")
	content := `# 签名 key 注册表（ADR-0016）：runner 按渠道 signingKey 查此表用 apksigner 重签。
! 感叹号也是注释

emptyapp.file=/opt/hybrid/store-emptyapp.keystore
emptyapp.alias=emptyapp
emptyapp.storePassword=sp1
emptyapp.keyPassword=kp1

# 第二把 key，故意缺 keyPassword，验证缺字段不影响解析其它字段
partial.file=/opt/hybrid/partial.keystore
partial.alias=partial-alias
partial.storePassword=sp2
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load 不应报错: %v", err)
	}
	if len(reg) != 2 {
		t.Fatalf("应解析出 2 个 id，实得 %d: %+v", len(reg), reg)
	}

	k, ok := reg.Lookup("emptyapp")
	if !ok {
		t.Fatal("应能查到 emptyapp")
	}
	if k.ID != "emptyapp" || k.File != "/opt/hybrid/store-emptyapp.keystore" || k.Alias != "emptyapp" ||
		k.StorePassword != "sp1" || k.KeyPassword != "kp1" {
		t.Errorf("emptyapp 解析不符: %+v", k)
	}

	p, ok := reg.Lookup("partial")
	if !ok {
		t.Fatal("应能查到 partial")
	}
	if p.File != "/opt/hybrid/partial.keystore" || p.Alias != "partial-alias" || p.StorePassword != "sp2" {
		t.Errorf("partial 已给字段解析不符: %+v", p)
	}
	if p.KeyPassword != "" {
		t.Errorf("partial 未给的 keyPassword 应为空串，实得 %q", p.KeyPassword)
	}

	if _, ok := reg.Lookup("nope"); ok {
		t.Error("不存在的 id 不应查到")
	}
}

// TestLoadMissingFileReturnsEmptyRegistry 验证注册表文件不存在时返回空注册表而非报错
// （构建机尚未烧入任何非默认 key 时的正常状态）。
func TestLoadMissingFileReturnsEmptyRegistry(t *testing.T) {
	reg, err := Load(filepath.Join(t.TempDir(), "no-such-file.properties"))
	if err != nil {
		t.Fatalf("文件不存在不应报错: %v", err)
	}
	if len(reg) != 0 {
		t.Errorf("应为空注册表，实得 %d 条", len(reg))
	}
	if _, ok := reg.Lookup("anything"); ok {
		t.Error("空注册表不应查到任何 id")
	}
}

// TestRegistryPathFromEnv 验证路径解析：环境变量优先，未设置则用默认路径。
func TestRegistryPathFromEnv(t *testing.T) {
	t.Setenv(EnvRegistryPath, "")
	if got := RegistryPathFromEnv(); got != DefaultRegistryPath {
		t.Errorf("未设置环境变量时应回退默认路径，实得 %q", got)
	}
	t.Setenv(EnvRegistryPath, "/tmp/custom-signing-keys.properties")
	if got := RegistryPathFromEnv(); got != "/tmp/custom-signing-keys.properties" {
		t.Errorf("应优先用环境变量指定路径，实得 %q", got)
	}
}
