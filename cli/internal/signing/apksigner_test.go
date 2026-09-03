package signing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindApksignerPicksHighestNumericVersion 在临时 build-tools 目录下放 34.0.0/35.0.0/9.0.0，
// 验证按「数字分段比较」选中 35.0.0（若按字符串字典序会误选 9.0.0）。
func TestFindApksignerPicksHighestNumericVersion(t *testing.T) {
	sdkRoot := t.TempDir()
	btDir := filepath.Join(sdkRoot, "build-tools")
	versions := []string{"34.0.0", "35.0.0", "9.0.0"}
	for _, v := range versions {
		dir := filepath.Join(btDir, v)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "apksigner"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// 一个非纯数字版本目录，不应干扰选择。
	if err := os.MkdirAll(filepath.Join(btDir, "debug"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvApksignerPath, "")
	t.Setenv("ANDROID_HOME", sdkRoot)
	t.Setenv("ANDROID_SDK_ROOT", "")

	got, err := FindApksigner()
	if err != nil {
		t.Fatalf("FindApksigner 不应报错: %v", err)
	}
	want := filepath.Join(btDir, "35.0.0", "apksigner")
	if got != want {
		t.Errorf("应选中最高版本 35.0.0，实得 %q（期望 %q）", got, want)
	}
}

// TestFindApksignerEnvOverride 验证 EnvApksignerPath 优先于 SDK 自动探测。
func TestFindApksignerEnvOverride(t *testing.T) {
	t.Setenv(EnvApksignerPath, "/custom/path/apksigner")
	got, err := FindApksigner()
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if got != "/custom/path/apksigner" {
		t.Errorf("应优先用环境变量指定路径，实得 %q", got)
	}
}

// TestFindApksignerNoSDK 验证既无环境变量又无 SDK 时报错。
func TestFindApksignerNoSDK(t *testing.T) {
	t.Setenv(EnvApksignerPath, "")
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	if _, err := FindApksigner(); err == nil {
		t.Fatal("既无环境变量又无 SDK 时应报错")
	}
}

// TestBuildSignArgsNeverContainsPasswords 验证签名命令行参数里绝不出现口令明文，
// 口令一律用 env:<变量名> 间接引用（ADR-0016 安全红线）。
func TestBuildSignArgsNeverContainsPasswords(t *testing.T) {
	key := Key{
		ID:            "emptyapp",
		File:          "/opt/hybrid/store-emptyapp.keystore",
		Alias:         "emptyapp",
		StorePassword: "super-secret-store-pass",
		KeyPassword:   "super-secret-key-pass",
	}
	args := buildSignArgs(key, "/tmp/app-ap01018-release.apk", "/tmp/app-ap01018-release.apk.resigned.tmp")

	joined := strings.Join(args, "\x00")
	if strings.Contains(joined, key.StorePassword) {
		t.Errorf("命令行参数不应包含 store 口令明文: %v", args)
	}
	if strings.Contains(joined, key.KeyPassword) {
		t.Errorf("命令行参数不应包含 key 口令明文: %v", args)
	}
	if !containsPair(args, "--ks-pass", "env:"+envKeystorePass) {
		t.Errorf("--ks-pass 应指向 env:%s，实得 args=%v", envKeystorePass, args)
	}
	if !containsPair(args, "--key-pass", "env:"+envKeyPass) {
		t.Errorf("--key-pass 应指向 env:%s，实得 args=%v", envKeyPass, args)
	}
	// v1/v2 开启，v3/v4 关闭。
	if !containsPair(args, "--v1-signing-enabled", "true") || !containsPair(args, "--v2-signing-enabled", "true") {
		t.Errorf("应开启 v1/v2 签名: %v", args)
	}
	if !containsPair(args, "--v3-signing-enabled", "false") || !containsPair(args, "--v4-signing-enabled", "false") {
		t.Errorf("应关闭 v3/v4 签名: %v", args)
	}
}

// TestBuildVerifyArgsIncludesMinSdkVersion 验证 verify 命令带 --min-sdk-version 21——
// 缺这个参数 apksigner 在 minSdk>=24 的 APK 上会跳过 v1 校验（本工程 minSdk 29）。
func TestBuildVerifyArgsIncludesMinSdkVersion(t *testing.T) {
	args := buildVerifyArgs("/tmp/app-ap01018-release.apk")
	if !containsPair(args, "--min-sdk-version", "21") {
		t.Errorf("verify 参数应含 --min-sdk-version 21，实得 %v", args)
	}
	if !contains(args, "--print-certs") {
		t.Errorf("verify 参数应含 --print-certs，实得 %v", args)
	}
}

// TestParseVerifyOutputSuccess 验证正常输出下能确认 v1/v2 生效并提取证书 DN/SHA-1。
func TestParseVerifyOutputSuccess(t *testing.T) {
	out := `Verifies
Verified using v1 scheme (JAR signing): true
Verified using v2 scheme (APK Signature Scheme v2): true
Verified using v3 scheme (APK Signature Scheme v3): false
Verified using v4 scheme (APK Signature Scheme v4): false
Verified for SourceStamp: false
Number of signers: 1
Signer #1 certificate DN: CN=empty-app
Signer #1 certificate SHA-256 digest: aabbccddeeff...
Signer #1 certificate SHA-1 digest: af:ea:ec:41:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff
Signer #1 certificate MD5 digest: 00112233445566778899aabbccddeeff
`
	info, err := parseVerifyOutput(out)
	if err != nil {
		t.Fatalf("正常输出不应报错: %v", err)
	}
	if info.DN != "CN=empty-app" {
		t.Errorf("DN 解析不符: %q", info.DN)
	}
	if info.SHA1 != "af:ea:ec:41:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff" {
		t.Errorf("SHA-1 解析不符: %q", info.SHA1)
	}
}

// TestParseVerifyOutputV1False 验证 v1 未生效时判为失败（例如漏加 --min-sdk-version 21 导致跳过校验）。
func TestParseVerifyOutputV1False(t *testing.T) {
	out := `Verifies
Verified using v1 scheme (JAR signing): false
Verified using v2 scheme (APK Signature Scheme v2): true
`
	if _, err := parseVerifyOutput(out); err == nil {
		t.Fatal("v1 未生效时应报错")
	}
}

// TestParseVerifyOutputV2False 验证 v2 未生效时判为失败。
func TestParseVerifyOutputV2False(t *testing.T) {
	out := `Verifies
Verified using v1 scheme (JAR signing): true
Verified using v2 scheme (APK Signature Scheme v2): false
`
	if _, err := parseVerifyOutput(out); err == nil {
		t.Fatal("v2 未生效时应报错")
	}
}

func containsPair(args []string, flag, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == val {
			return true
		}
	}
	return false
}

func contains(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}
