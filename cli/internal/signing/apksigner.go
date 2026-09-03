package signing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// 口令传给 apksigner 子进程的环境变量名（ADR-0016：只经子进程环境变量传递，
// 绝不出现在命令行参数、日志或错误信息里）。变量名本身非机密。
const (
	envKeystorePass = "HYBRID_SIGN_KS_PASS"
	envKeyPass      = "HYBRID_SIGN_KEY_PASS"
)

// EnvApksignerPath 指定 apksigner 可执行文件路径，优先于 Android SDK 自动探测。
const EnvApksignerPath = "HYBRID_PACK_APKSIGNER"

// CertInfo 是 `apksigner verify --print-certs` 解析出的证书信息，仅用于日志展示
// （帮助运维核对「这次是不是真的用对了 key」），不含任何机密。
type CertInfo struct {
	// DN 是签名证书的 Distinguished Name，如 "CN=empty-app"。
	DN string
	// SHA1 是签名证书的 SHA-1 摘要（十六进制，含冒号分隔，与 apksigner 输出一致）。
	SHA1 string
}

// FindApksigner 定位 apksigner 可执行文件：
//  1. 环境变量 EnvApksignerPath 非空则直接采用；
//  2. 否则在 $ANDROID_HOME（或 $ANDROID_SDK_ROOT）/build-tools/<版本>/ 下按版本号
//     （数字分段比较，而非字符串字典序，避免 "9.0.0" 被判定大于 "34.0.0"）取最高版本的一个。
func FindApksigner() (string, error) {
	if p := strings.TrimSpace(os.Getenv(EnvApksignerPath)); p != "" {
		return p, nil
	}

	sdkRoot := strings.TrimSpace(os.Getenv("ANDROID_HOME"))
	if sdkRoot == "" {
		sdkRoot = strings.TrimSpace(os.Getenv("ANDROID_SDK_ROOT"))
	}
	if sdkRoot == "" {
		return "", fmt.Errorf("未设置 %s，且 ANDROID_HOME/ANDROID_SDK_ROOT 均未设置，无法定位 apksigner", EnvApksignerPath)
	}

	btDir := filepath.Join(sdkRoot, "build-tools")
	entries, err := os.ReadDir(btDir)
	if err != nil {
		return "", fmt.Errorf("读取 build-tools 目录 %s 失败: %w", btDir, err)
	}

	exeName := "apksigner"
	if runtime.GOOS == "windows" {
		exeName = "apksigner.bat"
	}

	var best string
	var bestVer []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ver := parseVersion(e.Name())
		if ver == nil {
			continue // 非纯数字版本号目录（如遗留的调试/预览目录），跳过
		}
		candidate := filepath.Join(btDir, e.Name(), exeName)
		if _, err := os.Stat(candidate); err != nil {
			continue // 该版本目录下没有 apksigner，跳过
		}
		if best == "" || compareVersions(ver, bestVer) > 0 {
			best = candidate
			bestVer = ver
		}
	}
	if best == "" {
		return "", fmt.Errorf("在 %s 下未找到任何含 apksigner 的 build-tools 版本目录", btDir)
	}
	return best, nil
}

// parseVersion 把形如 "35.0.0" 的目录名解析成数字分段；含非数字分段（如 "debug"）返回 nil。
func parseVersion(name string) []int {
	parts := strings.Split(name, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// compareVersions 按数字分段比较两个版本号，返回 -1/0/1；缺失分段视为 0（如 "9" vs "9.1"）。
func compareVersions(a, b []int) int {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}

// buildSignArgs 组装 `apksigner sign` 的命令行参数（不含 apksigner 自身路径）。
//
// 口令一律用 `env:<变量名>` 让 apksigner 自行从子进程环境变量读取——本函数返回的 args
// 里绝不出现口令明文，可安全打日志/写测试断言。
func buildSignArgs(key Key, apkPath, outPath string) []string {
	return []string{
		"sign",
		"--ks", key.File,
		"--ks-key-alias", key.Alias,
		"--ks-pass", "env:" + envKeystorePass,
		"--key-pass", "env:" + envKeyPass,
		"--v1-signing-enabled", "true",
		"--v2-signing-enabled", "true",
		"--v3-signing-enabled", "false",
		"--v4-signing-enabled", "false",
		"--out", outPath,
		apkPath,
	}
}

// buildVerifyArgs 组装 `apksigner verify` 的命令行参数。
//
// 必须带 --min-sdk-version 21：minSdk>=24 的 APK（本工程 minSdk 29）若不显式指定，
// apksigner 会跳过 v1 方案校验，导致「重签成功但 v1 其实没生效」被漏检。
func buildVerifyArgs(apkPath string) []string {
	return []string{"verify", "-v", "--min-sdk-version", "21", "--print-certs", apkPath}
}

var (
	reV1Verified = regexp.MustCompile(`(?i)v1 scheme[^:\n]*:\s*true`)
	reV2Verified = regexp.MustCompile(`(?i)v2 scheme[^:\n]*:\s*true`)
	reCertDN     = regexp.MustCompile(`(?m)^Signer #1 certificate DN:\s*(.+)\s*$`)
	reCertSHA1   = regexp.MustCompile(`(?m)^Signer #1 certificate SHA-1 digest:\s*(.+)\s*$`)
)

// parseVerifyOutput 解析 `apksigner verify -v --print-certs` 的输出，确认 v1/v2 均生效
// 并提取证书 DN / SHA-1（用于日志展示，核对「这次真的换对 key 了」）。
// v1 或 v2 未确认生效视为校验失败（返回 error），调用方不得继续投递该 APK。
func parseVerifyOutput(output string) (CertInfo, error) {
	if !reV1Verified.MatchString(output) {
		return CertInfo{}, fmt.Errorf("apksigner verify 输出未确认 v1 签名生效，输出:\n%s", strings.TrimSpace(output))
	}
	if !reV2Verified.MatchString(output) {
		return CertInfo{}, fmt.Errorf("apksigner verify 输出未确认 v2 签名生效，输出:\n%s", strings.TrimSpace(output))
	}
	info := CertInfo{}
	if m := reCertDN.FindStringSubmatch(output); len(m) == 2 {
		info.DN = strings.TrimSpace(m[1])
	}
	if m := reCertSHA1.FindStringSubmatch(output); len(m) == 2 {
		info.SHA1 = strings.TrimSpace(m[1])
	}
	return info, nil
}

// Verify 执行 `apksigner verify -v --min-sdk-version 21 --print-certs <apkPath>`，
// 确认 v1/v2 均生效并返回证书信息。
func Verify(ctx context.Context, apksignerPath, apkPath string) (CertInfo, error) {
	cmd := exec.CommandContext(ctx, apksignerPath, buildVerifyArgs(apkPath)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return CertInfo{}, fmt.Errorf("apksigner verify 执行失败: %w（输出: %s）", err, strings.TrimSpace(out.String()))
	}
	return parseVerifyOutput(out.String())
}

// Resign 用 key 对 apkPath 就地重签（v1+v2，v3/v4 关闭以匹配现有 Gradle 出包的签名方案组合），
// 成功后立即用 Verify 校验，返回证书信息供调用方打日志。
//
// 口令只经子进程环境变量传递，绝不出现在命令行参数、日志或错误信息里；apksigner sign 阶段的
// stdout/stderr 一律丢弃、不拼进错误信息，出错时只把 exec 层错误（如非零退出码）包装返回——
// 更详细的诊断信息由随后的 Verify 失败时给出（其输出不含口令）。
//
// 重签用「先写临时文件、成功后原子 rename 替换原 APK」的方式，中途失败不会破坏原产物。
func Resign(ctx context.Context, apksignerPath string, key Key, apkPath string, logf func(format string, args ...any)) (CertInfo, error) {
	log := logf
	if log == nil {
		log = func(string, ...any) {}
	}

	if strings.TrimSpace(apksignerPath) == "" {
		return CertInfo{}, fmt.Errorf("apksigner 路径为空")
	}
	if key.File == "" || key.Alias == "" || key.StorePassword == "" || key.KeyPassword == "" {
		return CertInfo{}, fmt.Errorf("签名 key %q 物料不完整（注册表须同时提供 file/alias/storePassword/keyPassword）", key.ID)
	}
	if _, err := os.Stat(key.File); err != nil {
		return CertInfo{}, fmt.Errorf("签名 key %q 的 keystore 文件未找到: %s", key.ID, key.File)
	}
	if _, err := os.Stat(apkPath); err != nil {
		return CertInfo{}, fmt.Errorf("待重签的 APK 未找到: %s", apkPath)
	}

	outPath := apkPath + ".resigned.tmp"
	_ = os.Remove(outPath) // 清理可能残留的上次临时文件

	cmd := exec.CommandContext(ctx, apksignerPath, buildSignArgs(key, apkPath, outPath)...)
	cmd.Env = append(os.Environ(),
		envKeystorePass+"="+key.StorePassword,
		envKeyPass+"="+key.KeyPassword,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	log("  执行 apksigner sign（key=%s，apk=%s）...", key.ID, filepath.Base(apkPath))
	if err := cmd.Run(); err != nil {
		_ = os.Remove(outPath)
		return CertInfo{}, fmt.Errorf("apksigner 签名失败（key=%s）: %w", key.ID, err)
	}
	if err := os.Rename(outPath, apkPath); err != nil {
		_ = os.Remove(outPath)
		return CertInfo{}, fmt.Errorf("重签后替换原 APK 失败: %w", err)
	}

	info, err := Verify(ctx, apksignerPath, apkPath)
	if err != nil {
		return CertInfo{}, fmt.Errorf("重签后校验失败（key=%s）: %w", key.ID, err)
	}
	return info, nil
}
