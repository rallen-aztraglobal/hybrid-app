// Package doctor 做打包前预检：JDK、Android SDK、keystore(local.properties)、后台连通性。
//
// 安全（CLAUDE.md 护栏 4）：doctor 只检查 keystore 相关「键是否存在、文件是否可定位」，
// 绝不读取或打印任何口令值，更不会把它们带到任何网络请求里。
package doctor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/config"
	"github.com/hybrid-app/cli/internal/repo"
)

// Level 是检查结果级别。
type Level int

const (
	OK Level = iota
	Warn
	Fail
)

func (l Level) Symbol() string {
	switch l {
	case OK:
		return "✓"
	case Warn:
		return "!"
	default:
		return "✗"
	}
}

func (l Level) String() string {
	switch l {
	case OK:
		return "OK"
	case Warn:
		return "WARN"
	default:
		return "FAIL"
	}
}

// Check 是一项检查的结果。
type Check struct {
	Name   string
	Level  Level
	Detail string
}

// Report 是全部检查结果。
type Report struct {
	Checks []Check
}

// HasFail 返回是否存在致命问题（影响打包）。
func (r *Report) HasFail() bool {
	for _, c := range r.Checks {
		if c.Level == Fail {
			return true
		}
	}
	return false
}

func (r *Report) add(name string, lvl Level, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Level: lvl, Detail: detail})
}

// Run 执行全部预检。cfg 可为 nil（则跳过后台连通性）。
func Run(ctx context.Context, r *repo.Repo, cfg *config.Config) *Report {
	rep := &Report{}
	checkJDK(rep)
	checkAndroidSDK(rep, r)
	checkKeystore(rep, r)
	checkBackend(ctx, rep, cfg)
	return rep
}

var javaVersionRe = regexp.MustCompile(`version "?(\d+)(?:\.(\d+))?`)

func checkJDK(rep *Report) {
	bin := "java"
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		cand := filepath.Join(jh, "bin", "java")
		if _, err := os.Stat(cand); err == nil {
			bin = cand
		}
	}
	out, err := exec.Command(bin, "-version").CombinedOutput()
	if err != nil {
		rep.add("JDK", Fail, "未检测到可用的 java（请安装 JDK 17 并设置 JAVA_HOME）")
		return
	}
	text := string(out)
	major := 0
	if m := javaVersionRe.FindStringSubmatch(text); m != nil {
		fmt.Sscanf(m[1], "%d", &major)
	}
	first := strings.SplitN(strings.TrimSpace(text), "\n", 2)[0]
	switch {
	case major == 17:
		rep.add("JDK", OK, first)
	case major == 0:
		rep.add("JDK", Warn, "无法解析 java 版本: "+first)
	case major >= 11:
		rep.add("JDK", Warn, fmt.Sprintf("检测到 JDK %d；本工程要求 JDK 17（见 CLAUDE.md）", major))
	default:
		rep.add("JDK", Fail, fmt.Sprintf("检测到 JDK %d，过低；需要 JDK 17", major))
	}
}

func checkAndroidSDK(rep *Report, r *repo.Repo) {
	// 优先环境变量，其次 local.properties 的 sdk.dir。
	sdk := firstNonEmpty(os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT"))
	src := "环境变量"
	if sdk == "" {
		if v := readProp(r.LocalProperties(), "sdk.dir"); v != "" {
			sdk = v
			src = "local.properties(sdk.dir)"
		}
	}
	if sdk == "" {
		rep.add("Android SDK", Fail, "未找到 SDK（设置 ANDROID_HOME 或 local.properties 的 sdk.dir）")
		return
	}
	if fi, err := os.Stat(sdk); err != nil || !fi.IsDir() {
		rep.add("Android SDK", Fail, fmt.Sprintf("SDK 路径不存在: %s（来源: %s）", sdk, src))
		return
	}
	// 进一步看是否像个 SDK（含 platform-tools 或 platforms）。
	likely := false
	for _, sub := range []string{"platform-tools", "platforms", "build-tools"} {
		if _, err := os.Stat(filepath.Join(sdk, sub)); err == nil {
			likely = true
			break
		}
	}
	if !likely {
		rep.add("Android SDK", Warn, fmt.Sprintf("目录存在但未见 platform-tools/platforms: %s", sdk))
		return
	}
	rep.add("Android SDK", OK, fmt.Sprintf("%s（来源: %s）", sdk, src))
}

// checkKeystore 校验签名配置完整性，但绝不读取口令值。
func checkKeystore(rep *Report, r *repo.Repo) {
	lp := r.LocalProperties()
	if _, err := os.Stat(lp); err != nil {
		rep.add("Keystore", Fail, "缺少 local.properties（签名配置所在）")
		return
	}
	props := readProps(lp)
	var missing []string
	for _, k := range []string{"KEYSTORE_FILE", "KEYSTORE_PASSWORD", "KEY_ALIAS", "KEY_PASSWORD"} {
		if strings.TrimSpace(props[k]) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		rep.add("Keystore", Fail, "local.properties 缺少: "+strings.Join(missing, ", "))
		return
	}
	// 校验 keystore 文件可定位（相对路径以仓库根为基准，与 Gradle file() 一致）。
	ksPath := props["KEYSTORE_FILE"]
	if !filepath.IsAbs(ksPath) {
		ksPath = filepath.Join(r.Root, ksPath)
	}
	if _, err := os.Stat(ksPath); err != nil {
		rep.add("Keystore", Warn, fmt.Sprintf("签名键齐全，但 keystore 文件未找到: %s", props["KEYSTORE_FILE"]))
		return
	}
	// 仅报告文件名，不泄露完整路径与任何口令。
	rep.add("Keystore", OK, "签名配置完整（keystore: "+filepath.Base(props["KEYSTORE_FILE"])+"，口令已就位·未读取）")
}

func checkBackend(ctx context.Context, rep *Report, cfg *config.Config) {
	if cfg == nil || strings.TrimSpace(cfg.Server) == "" {
		rep.add("后台连通性", Warn, "未配置后台地址（hybrid-pack login --server <URL>），跳过")
		return
	}
	cl := api.New(cfg.Server, cfg.Token)
	if err := cl.Ping(ctx); err != nil {
		rep.add("后台连通性", Warn, fmt.Sprintf("无法连通 %s：%v", cfg.Server, err))
		return
	}
	auth := "未登录"
	if strings.TrimSpace(cfg.Token) != "" {
		auth = "已登录(" + cfg.Operator + ")"
	}
	rep.add("后台连通性", OK, fmt.Sprintf("%s 可达（%s）", cfg.Server, auth))
}

// ---- 小工具：读取 .properties（key=value，# 注释），只取值不解释语义 ----

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
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		out[k] = v
	}
	return out
}

func readProp(path, key string) string {
	return readProps(path)[key]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
