// Package build 跨平台调用 Gradle wrapper 执行打包，并收集产物 APK。
//
// 严格复刻 package.sh 的行为（cli-go.md 要求）：
//   - task 名 = assemble + Cap(flavor) + Release，其中 Cap 仅首字母大写、其余原样；
//   - 一次调用可传多个 task；附带 -PtestEvents=<bool> 开关；
//   - Windows 调 gradlew.bat，macOS/Linux 调 ./gradlew（用 filepath/runtime 判定）。
//
// 绝不修改 Gradle 逻辑，只调用它。签名仍由 Gradle 读取 local.properties 完成，
// 本包不接触 keystore 与任何口令。
package build

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hybrid-app/cli/internal/repo"
)

// Cap 首字母大写、其余字符原样保留，与 package.sh 的 cap() 完全一致：
//
//	cap() { echo "$(tr '[:lower:]' '[:upper:]' <<< "${1:0:1}")${1:1}"; }
//
// 注意：是按「首个字节」处理（渠道名均为 ASCII），不做整词标题化，避免改变后续字符。
func Cap(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	c := b[0]
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	return string(c) + s[1:]
}

// TaskName 返回某 flavor 的 release 打包 task 名：assemble<Cap>Release。
func TaskName(flavor string) string {
	return "assemble" + Cap(flavor) + "Release"
}

// Options 控制一次构建。
type Options struct {
	Flavors    []string
	TestEvents bool
	// VersionName 非空时透传 -PversionName=<v> 给 gradlew（app/build.gradle 据此设
	// versionName/versionCode；留空则沿用 build.gradle 的默认值，不传该 -P）。
	// 仅校验形如 X.Y.Z（与 build.gradle 的正则一致），避免把非法值传进 Gradle。
	VersionName string
	// ExtraArgs 透传给 gradlew 的附加参数（如 --offline）。
	ExtraArgs []string
	// Stdout/Stderr 为 nil 时分别用 os.Stdout/os.Stderr。
	Stdout io.Writer
	Stderr io.Writer
	// CaptureTailLines>0 时，额外把输出末尾 N 行收集进 Result.LogTail（供构建记录上传）。
	CaptureTailLines int
}

// Result 是一次构建的结果。
type Result struct {
	Tasks    []string
	ExitCode int
	LogTail  string
}

// Args 组装传给 gradlew 的完整参数（不含 gradlew 自身）。
func (o Options) Args() []string {
	args := make([]string, 0, len(o.Flavors)+2+len(o.ExtraArgs))
	for _, f := range o.Flavors {
		args = append(args, TaskName(f))
	}
	args = append(args, fmt.Sprintf("-PtestEvents=%t", o.TestEvents))
	// 版本号：仅当显式指定时透传，沿用 build.gradle 的 -PversionName 机制（不改 Gradle 逻辑）。
	if v := strings.TrimSpace(o.VersionName); v != "" {
		args = append(args, "-PversionName="+v)
	}
	args = append(args, o.ExtraArgs...)
	return args
}

// GradlewPath 返回本平台应使用的 wrapper 路径。
func GradlewPath(r *repo.Repo) string {
	return r.Gradlew(runtime.GOOS == "windows")
}

// Run 执行 gradlew 打包。工作目录设为仓库根（gradlew 需在根目录运行）。
func Run(ctx context.Context, r *repo.Repo, opt Options) (*Result, error) {
	if len(opt.Flavors) == 0 {
		return nil, fmt.Errorf("未指定任何渠道（flavor）")
	}
	if v := strings.TrimSpace(opt.VersionName); v != "" && !ValidVersionName(v) {
		return nil, fmt.Errorf("versionName 必须为 X.Y.Z 格式（如 1.0.1），当前: %q", v)
	}
	gw := GradlewPath(r)
	if _, err := os.Stat(gw); err != nil {
		return nil, fmt.Errorf("未找到 gradle wrapper: %s", gw)
	}

	args := opt.Args()
	res := &Result{Tasks: tasksOf(opt.Flavors)}

	// Windows 下 .bat 需经 cmd 执行；其余直接执行脚本。
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", append([]string{"/c", gw}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, gw, args...)
	}
	cmd.Dir = r.Root
	cmd.Env = os.Environ()
	cmd.Stdin = nil

	stdout := opt.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opt.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	var tail *tailBuffer
	if opt.CaptureTailLines > 0 {
		tail = newTailBuffer(opt.CaptureTailLines)
		stdout = io.MultiWriter(stdout, tail)
		stderr = io.MultiWriter(stderr, tail)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	if tail != nil {
		res.LogTail = tail.String()
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if asExit(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, fmt.Errorf("gradlew 退出码 %d", res.ExitCode)
		}
		return res, fmt.Errorf("执行 gradlew 失败: %w", runErr)
	}
	return res, nil
}

func tasksOf(flavors []string) []string {
	out := make([]string, len(flavors))
	for i, f := range flavors {
		out[i] = TaskName(f)
	}
	return out
}

// CollectAPKs 扫描某 flavor 的 release 输出目录，返回 *.apk 的绝对路径。
func CollectAPKs(r *repo.Repo, flavor string) ([]string, error) {
	dir := r.APKOutputDir(flavor)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取产物目录 %s 失败: %w", dir, err)
	}
	var apks []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".apk") {
			apks = append(apks, filepath.Join(dir, e.Name()))
		}
	}
	return apks, nil
}

// tailBuffer 是只保留最后 N 行的环形缓冲，用于日志摘要。
type tailBuffer struct {
	max   int
	lines []string
	buf   strings.Builder
}

func newTailBuffer(n int) *tailBuffer { return &tailBuffer{max: n} }

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf.Write(p)
	// 按行切分累积内容，仅保留尾部 max 行。
	sc := bufio.NewScanner(strings.NewReader(t.buf.String()))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var all []string
	for sc.Scan() {
		all = append(all, sc.Text())
	}
	if len(all) > t.max {
		all = all[len(all)-t.max:]
	}
	t.lines = all
	return len(p), nil
}

func (t *tailBuffer) String() string { return strings.Join(t.lines, "\n") }
