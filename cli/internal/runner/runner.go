// Package runner 实现 ADR-0008 的「服务器端构建」执行体：构建机上的 hybrid-pack runner。
//
// 它轮询后端 build-job 队列，领取任务后在仓库 checkout 上执行：
//
//	pull（拉最新配置→渲染 CSV/res/bootstrap）
//	  → ./gradlew assemble<Flavor>Release（+签名，复用现有 Gradle 机制，零改动）
//	  → 按渠道核对签名 key（ADR-0016）：非默认 key 的渠道用 apksigner 就地重签 v1+v2 并校验
//	  → 把产出 APK 落到目标产物目录（默认 nginx 共享卷 /var/www/apks/<brand>/<flavor>/<versionName>/）
//	  → 回传状态/日志/产物给后端
//
// 安全红线（CLAUDE.md 护栏 4 / ADR-0008）：keystore 路径与口令只从环境/secret 读取，
// 绝不上传后端、绝不进任何请求体、绝不打印口令值；产物登记只含非机密元信息。同样适用于
// ADR-0016 的「签名 key 注册表」：只存在于构建机本地文件，不进 manifest 之外的任何请求体。
package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hybrid-app/cli/internal/api"
	"github.com/hybrid-app/cli/internal/build"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/render"
	"github.com/hybrid-app/cli/internal/repo"
	"github.com/hybrid-app/cli/internal/signing"
)

// DefaultArtifactDir 是产物落盘的默认根目录（ADR-0008：与构建机共享的 nginx /apks 卷）。
const DefaultArtifactDir = "/var/www/apks"

// Backend 抽象 runner 需要的后端能力，便于测试用 fake 替换（真实实现是 *api.Client）。
type Backend interface {
	ClaimBuildJob(ctx context.Context, runnerID string) (*manifest.BuildJob, error)
	UpdateJobStatus(ctx context.Context, jobID int64, status, logExcerpt string) error
	AppendJobLog(ctx context.Context, jobID int64, chunk string) error
	RegisterJobArtifact(ctx context.Context, jobID int64, art manifest.BuildArtifact) error
}

// 编译期断言：真实 Client 满足 Backend。
var _ Backend = (*api.Client)(nil)

// Options 配置 runner 行为。
type Options struct {
	// RunnerID 本构建机标识（用于审计/任务归属，非机密）。空则用主机名。
	RunnerID string
	// ArtifactDir 产物根目录；空则用 DefaultArtifactDir。
	ArtifactDir string
	// ArtifactBaseURL 产物公网下载前缀（nginx 暴露），如 https://console.example.com/apks
	// 或相对 /apks。与 ArtifactDir 下的相对路径拼成 BuildArtifact.ApkURL。空则只登记相对路径。
	ArtifactBaseURL string
	// PollInterval 队列为空时的轮询间隔；<=0 用 5s。
	PollInterval time.Duration
	// Once 为 true 时只领取并处理一个任务（含「无任务即返回」），用于 CI/单测/手动触发。
	Once bool
	// CaptureTailLines 回传给后端的日志尾行数；<=0 用 200。
	CaptureTailLines int
	// Keystore 从环境读取的签名材料（绝不上传）；为 nil 时不主动注入（依赖构建机已有 local.properties）。
	Keystore *Keystore
	// Source 是 pull 阶段的 manifest 来源（真实后端 = *api.Client）。必填。
	Source api.ManifestSource
	// SigningRegistryPath 签名 key 注册表文件路径（ADR-0016）；空则用
	// signing.RegistryPathFromEnv()（环境变量 HYBRID_PACK_SIGNING_KEYS，未设置时用
	// /opt/hybrid/signing-keys.properties）。仅测试/特殊部署需要覆盖默认解析逻辑时才设置。
	SigningRegistryPath string
	// Apksigner apksigner 可执行文件路径；空则用 signing.FindApksigner()（环境变量
	// HYBRID_PACK_APKSIGNER 或按 ANDROID_HOME/build-tools 自动探测最高版本）。
	Apksigner string
	// buildFn 执行实际打包；为 nil 时用 build.Run（生产路径）。仅测试会注入桩。
	buildFn func(ctx context.Context, r *repo.Repo, opt build.Options) (*build.Result, error)
	// Logf 进度回调（可为 nil）。
	Logf func(format string, args ...any)
}

// build 执行打包：默认走 build.Run，测试可经 buildFn 注入桩。
func (o Options) build(ctx context.Context, r *repo.Repo, bo build.Options) (*build.Result, error) {
	if o.buildFn != nil {
		return o.buildFn(ctx, r, bo)
	}
	return build.Run(ctx, r, bo)
}

// manifestSource 返回 pull 阶段使用的 manifest 来源。
func (o Options) manifestSource() (api.ManifestSource, error) {
	if o.Source == nil {
		return nil, fmt.Errorf("runner 未配置 manifest 来源（Source）")
	}
	return o.Source, nil
}

// signingRegistryPath 返回签名 key 注册表路径：显式配置优先，否则走环境变量/默认路径。
func (o Options) signingRegistryPath() string {
	if strings.TrimSpace(o.SigningRegistryPath) != "" {
		return o.SigningRegistryPath
	}
	return signing.RegistryPathFromEnv()
}

// apksignerPath 返回 apksigner 可执行文件路径：显式配置优先，否则自动探测。
func (o Options) apksignerPath() (string, error) {
	if strings.TrimSpace(o.Apksigner) != "" {
		return o.Apksigner, nil
	}
	return signing.FindApksigner()
}

func (o Options) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

func (o Options) pollInterval() time.Duration {
	if o.PollInterval <= 0 {
		return 5 * time.Second
	}
	return o.PollInterval
}

func (o Options) tailLines() int {
	if o.CaptureTailLines <= 0 {
		return 200
	}
	return o.CaptureTailLines
}

func (o Options) artifactDir() string {
	if strings.TrimSpace(o.ArtifactDir) != "" {
		return o.ArtifactDir
	}
	return DefaultArtifactDir
}

// Run 启动 runner 主循环，直到 ctx 取消（或 Once 模式处理完一个任务）。
func Run(ctx context.Context, r *repo.Repo, be Backend, opt Options) error {
	if opt.RunnerID == "" {
		if h, err := os.Hostname(); err == nil {
			opt.RunnerID = h
		} else {
			opt.RunnerID = "runner"
		}
	}
	// 启动前确保签名材料就位（仅从环境/secret，绝不上传）。失败即拒绝启动，避免打出未签名包。
	if err := ensureSigning(r, opt); err != nil {
		return err
	}
	opt.logf("runner 已启动（id=%s，产物目录=%s，轮询=%s）", opt.RunnerID, opt.artifactDir(), opt.pollInterval())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job, err := be.ClaimBuildJob(ctx, opt.RunnerID)
		if err != nil {
			if opt.Once {
				return fmt.Errorf("领取任务失败: %w", err)
			}
			opt.logf("领取任务失败（将重试）: %v", err)
			if !sleep(ctx, opt.pollInterval()) {
				return ctx.Err()
			}
			continue
		}
		if job == nil || job.ID == 0 {
			if opt.Once {
				opt.logf("队列为空，无任务可领取")
				return nil
			}
			if !sleep(ctx, opt.pollInterval()) {
				return ctx.Err()
			}
			continue
		}

		opt.logf("领取任务 #%d（brand=%s，flavors=%d，version=%s）", job.ID, job.Brand, len(job.Flavors), job.VersionName)
		if err := processJob(ctx, r, be, job, opt); err != nil {
			opt.logf("任务 #%d 失败: %v", job.ID, err)
			// 失败已在 processJob 内回传后端；这里仅记录，继续取下一个。
		} else {
			opt.logf("任务 #%d 成功", job.ID)
		}

		if opt.Once {
			return nil
		}
	}
}

// processJob 执行单个任务的完整流程，并把状态/日志/产物回传后端。
func processJob(ctx context.Context, r *repo.Repo, be Backend, job *manifest.BuildJob, opt Options) error {
	// 基本校验。
	if len(job.Flavors) == 0 {
		_ = be.UpdateJobStatus(ctx, job.ID, manifest.StatusFailed, "任务未指定任何 flavor")
		return fmt.Errorf("任务 #%d 未指定 flavor", job.ID)
	}
	if v := strings.TrimSpace(job.VersionName); v != "" && !build.ValidVersionName(v) {
		msg := fmt.Sprintf("versionName 非法（需 X.Y.Z）: %q", v)
		_ = be.UpdateJobStatus(ctx, job.ID, manifest.StatusFailed, msg)
		return fmt.Errorf("任务 #%d %s", job.ID, msg)
	}

	// 标记 running。
	_ = be.UpdateJobStatus(ctx, job.ID, manifest.StatusRunning, "")

	// 实时日志流：阶段标记 + gradle 输出按行节流回传，前端终端近实时滚动（ADR-0008）。
	ls := newJobLogStreamer(ctx, be, job.ID)
	defer ls.Close()
	// step 同时写 runner 控制台（docker logs）与回传后端（前端终端）。
	step := func(format string, a ...any) {
		msg := fmt.Sprintf(format, a...)
		opt.logf("%s", msg)
		ls.line(msg)
	}

	// 1) pull：拉最新配置渲染回本地（构建机消费 source-of-truth，ADR-0004/0008）。
	// manifest 只拉一次：既用于渲染，也留给后面「2b) 按渠道重签」查每个 flavor 的 signingKey
	// （ADR-0016），避免重复请求后端。
	src, err := opt.manifestSource()
	if err != nil {
		fail(ctx, be, job.ID, "", err)
		return err
	}
	step("→ [#%d] pull %s ...", job.ID, job.Brand)
	m, err := src.Manifest(ctx, job.Brand)
	if err != nil {
		fail(ctx, be, job.ID, "", fmt.Errorf("拉取 %s manifest 失败: %w", job.Brand, err))
		return err
	}
	if _, err := render.RenderManifest(ctx, r, src, m, render.Options{
		Logf: func(f string, a ...any) { step("    "+f, a...) },
	}); err != nil {
		fail(ctx, be, job.ID, "", fmt.Errorf("pull 失败: %w", err))
		return err
	}

	// 1b) 拉品牌 google-services.json → app/google-services.json（推送的编译期注入；ADR-0012 第 6b 节）。
	// 「有则落地、无则跳过」，不阻断未启用推送的品牌；漏掉它会导致 google-services 插件不应用、
	// APK 无 Firebase 配置、运行时拿不到 FCM token（与交互式 pull 命令保持一致，见 pull.go）。
	if err := render.PullGoogleServices(ctx, r, src, job.Brand, render.Options{
		Logf: func(f string, a ...any) { step("    "+f, a...) },
	}); err != nil {
		step("    警告: google-services.json 写入失败（%v），本次构建将无推送能力", err)
	}

	// 2) gradlew assemble<Flavor>Release（+签名）。stdout/stderr 实时回传 + 末尾摘要随结果回传。
	step("→ [#%d] assemble %v (version=%s) ...", job.ID, job.Flavors, job.VersionName)
	res, buildErr := opt.build(ctx, r, build.Options{
		Flavors:          job.Flavors,
		TestEvents:       job.TestEvents,
		VersionName:      job.VersionName, // runner 用 job.versionName
		CaptureTailLines: opt.tailLines(),
		Stdout:           io.MultiWriter(os.Stdout, ls), // 控制台 + 前端终端
		Stderr:           io.MultiWriter(os.Stderr, ls),
	})
	logTail := ""
	if res != nil {
		logTail = res.LogTail
	}
	if buildErr != nil {
		fail(ctx, be, job.ID, logTail, fmt.Errorf("打包失败: %w", buildErr))
		return buildErr
	}

	// 2b) 按渠道重签（ADR-0016）：Gradle 仍只出默认 key 的包（护栏 #1 不动），一批已上架商店、
	// 当年用另一把 key 签名的老渠道（manifest.Channel.SigningKey 非空）在此用 apksigner 重签
	// v1+v2 并校验。fail-closed：渠道要求的 key 在构建机注册表里没有 → 整个任务失败，绝不能把
	// 默认签名的包当商店包投递（同包名双证书会导致商店拒收或用户无法覆盖升级）。
	step("→ [#%d] 按渠道核对签名 key ...", job.ID)
	if err := resignArtifacts(ctx, r, job, m, opt, step); err != nil {
		fail(ctx, be, job.ID, logTail, err)
		return err
	}

	// 3) 收集产物 → 落到目标产物目录 → 登记。versionName 用于路径与登记元信息。
	version := strings.TrimSpace(job.VersionName)
	if version == "" {
		version = build.ReadVersionName(r) // 任务未指定版本时回读 build.gradle 默认
	}
	count, err := deliverArtifacts(ctx, r, be, job, version, opt)
	if err != nil {
		fail(ctx, be, job.ID, logTail, fmt.Errorf("投递产物失败: %w", err))
		return err
	}
	if count == 0 {
		err := fmt.Errorf("构建成功但未找到任何 APK 产物")
		fail(ctx, be, job.ID, logTail, err)
		return err
	}

	// 4) 成功回传。
	step("✓ [#%d] 完成，投递 %d 个 APK", job.ID, count)
	_ = be.UpdateJobStatus(ctx, job.ID, manifest.StatusSuccess, logTail)
	return nil
}

// deliverArtifacts 把每个 flavor 的 release APK 复制到产物目录并登记，返回投递的 APK 数。
//
// 目标路径（ADR-0008）：<artifactDir>/<brand>/<flavor>/<versionName>/<原文件名>。
func deliverArtifacts(ctx context.Context, r *repo.Repo, be Backend, job *manifest.BuildJob, version string, opt Options) (int, error) {
	total := 0
	for _, flavor := range job.Flavors {
		apks, err := build.CollectAPKs(r, flavor)
		if err != nil {
			return total, err
		}
		for _, srcPath := range apks {
			fileName := filepath.Base(srcPath)
			relDir := filepath.Join(job.Brand, flavor, version)
			dstDir := filepath.Join(opt.artifactDir(), relDir)
			if err := os.MkdirAll(dstDir, 0o755); err != nil {
				return total, fmt.Errorf("创建产物目录 %s 失败: %w", dstDir, err)
			}
			dstPath := filepath.Join(dstDir, fileName)
			size, sum, err := copyFile(srcPath, dstPath)
			if err != nil {
				return total, fmt.Errorf("复制 %s 失败: %w", fileName, err)
			}
			// URL 用正斜杠（HTTP 路径），与本地 filepath 分隔符无关。
			relURL := path(job.Brand, flavor, version, fileName)
			art := manifest.BuildArtifact{
				Flavor:      flavor,
				VersionName: version,
				FileName:    fileName,
				ApkURL:      joinURL(opt.ArtifactBaseURL, relURL),
				SizeBytes:   size,
				SHA256:      sum,
			}
			if err := be.RegisterJobArtifact(ctx, job.ID, art); err != nil {
				return total, fmt.Errorf("登记产物 %s 失败: %w", fileName, err)
			}
			opt.logf("    投递 %s → %s（%d 字节）", fileName, dstPath, size)
			total++
		}
	}
	return total, nil
}

// resignArtifacts 按渠道把已出包（默认 key 签名）的 APK 重签成 manifest 指定的签名 key（如有）。
//
// 只处理 job.Flavors 中 manifest.Channel.SigningKey 非空的渠道；空则是默认 key，跳过（无需
// 打日志，避免刷屏）。fail-closed（ADR-0016 决策 4）：渠道要求的 key 在构建机注册表里查不到，
// 整个任务失败，错误信息点明渠道名、key id 与注册表路径，方便运维知道该去 build-runner
// 镜像里烧哪把 key。
func resignArtifacts(ctx context.Context, r *repo.Repo, job *manifest.BuildJob, m *manifest.Manifest, opt Options, step func(format string, a ...any)) error {
	chByFlavor := make(map[string]manifest.Channel, len(m.Channels))
	for _, ch := range m.Channels {
		chByFlavor[ch.Flavor] = ch
	}

	registryPath := opt.signingRegistryPath()
	var registry signing.Registry
	var registryLoaded bool
	var apksignerPath string // 懒加载：只有真的需要重签才去定位 apksigner（避免测试/无需求场景依赖 ANDROID_HOME）

	for _, flavor := range job.Flavors {
		ch, ok := chByFlavor[flavor]
		if !ok {
			// 理论上不会发生（job.Flavors 来自同一份 manifest 渲染）；按默认签名处理并记警告。
			step("    警告: manifest 未找到渠道 %s，按默认签名处理", flavor)
			continue
		}
		keyID := strings.TrimSpace(ch.SigningKey)
		if keyID == "" {
			continue // 默认 key，Gradle 已直接签好，无需处理
		}

		if !registryLoaded {
			reg, err := signing.Load(registryPath)
			if err != nil {
				return fmt.Errorf("加载签名 key 注册表 %s 失败: %w", registryPath, err)
			}
			registry = reg
			registryLoaded = true
		}
		key, ok := registry.Lookup(keyID)
		if !ok {
			return fmt.Errorf("渠道 %s 需要签名 key %q，但构建机注册表（%s）中没有这把 key；"+
				"请在 build-runner 镜像中烧入（deploy/Dockerfile.builder，ADR-0016）", flavor, keyID, registryPath)
		}

		if apksignerPath == "" {
			p, err := opt.apksignerPath()
			if err != nil {
				return fmt.Errorf("定位 apksigner 失败: %w", err)
			}
			apksignerPath = p
		}

		apks, err := build.CollectAPKs(r, flavor)
		if err != nil {
			return err
		}
		if len(apks) == 0 {
			// 无产物：交给后续 deliverArtifacts 统一报「未找到任何 APK 产物」，这里不重复报错。
			continue
		}
		for _, apkPath := range apks {
			info, err := signing.Resign(ctx, apksignerPath, key, apkPath, func(f string, a ...any) { step("    "+f, a...) })
			if err != nil {
				return fmt.Errorf("渠道 %s 用签名 key %q 重签 %s 失败: %w", flavor, keyID, filepath.Base(apkPath), err)
			}
			step("✓ [%s] 已用签名 key %s 重签（CN=%s，SHA-1=%s）", flavor, keyID, info.DN, info.SHA1)
		}
	}
	return nil
}

// fail 把失败状态与日志摘要回传后端（best-effort），用于 processJob 各失败分支。
func fail(ctx context.Context, be Backend, jobID int64, logTail string, cause error) {
	msg := cause.Error()
	if logTail != "" {
		msg = logTail + "\n---\n" + msg
	}
	_ = be.UpdateJobStatus(ctx, jobID, manifest.StatusFailed, msg)
}

// copyFile 把 src 复制到 dst（原子：先写 .tmp 再 rename），返回字节数与 sha256（十六进制）。
func copyFile(src, dst string) (int64, string, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, "", err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, h), in)
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return 0, "", closeErr
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

// sleep 在 ctx 可取消的前提下睡 d；返回 false 表示被取消。
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// path 用正斜杠拼 HTTP 相对路径（不依赖平台分隔符）。
func path(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, "/")
}

// joinURL 把下载前缀与相对路径拼成完整 URL；base 为空时返回相对路径（前导 /apks 由部署决定）。
func joinURL(base, rel string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "/" + rel
	}
	return base + "/" + rel
}
