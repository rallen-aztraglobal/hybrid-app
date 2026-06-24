package runner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hybrid-app/cli/internal/build"
	"github.com/hybrid-app/cli/internal/manifest"
	"github.com/hybrid-app/cli/internal/repo"
)

// fakeBackend 记录 runner 对后端的调用，便于断言。
type fakeBackend struct {
	mu        sync.Mutex
	jobs      []*manifest.BuildJob // 依次领取；空则返回 ID=0
	statuses  []manifest.JobStatusUpdate
	artifacts []manifest.BuildArtifact
}

func (f *fakeBackend) ClaimBuildJob(_ context.Context, _ string) (*manifest.BuildJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.jobs) == 0 {
		return &manifest.BuildJob{ID: 0}, nil
	}
	j := f.jobs[0]
	f.jobs = f.jobs[1:]
	return j, nil
}

func (f *fakeBackend) UpdateJobStatus(_ context.Context, _ int64, status, logExcerpt string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statuses = append(f.statuses, manifest.JobStatusUpdate{Status: status, LogExcerpt: logExcerpt})
	return nil
}

func (f *fakeBackend) AppendJobLog(_ context.Context, _ int64, _ string) error { return nil }

func (f *fakeBackend) RegisterJobArtifact(_ context.Context, _ int64, art manifest.BuildArtifact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.artifacts = append(f.artifacts, art)
	return nil
}

// fakeRepoWithAPK 构造最小仓库布局，并在某 flavor 的 release 输出目录放一个假 APK。
func fakeRepoWithAPK(t *testing.T, flavor, apkName, content string) *repo.Repo {
	t.Helper()
	root := t.TempDir()
	r := &repo.Repo{Root: root}
	dir := r.APKOutputDir(flavor)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, apkName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestDeliverArtifacts 验证产物落盘路径、URL 拼接与登记内容（核心 ADR-0008 行为）。
func TestDeliverArtifacts(t *testing.T) {
	r := fakeRepoWithAPK(t, "ap01018", "app-ap01018-release.apk", "APKDATA")
	artifactRoot := t.TempDir()
	be := &fakeBackend{}
	job := &manifest.BuildJob{ID: 7, Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.0.1"}

	opt := Options{ArtifactDir: artifactRoot, ArtifactBaseURL: "https://console.example.com/apks"}
	n, err := deliverArtifacts(context.Background(), r, be, job, "1.0.1", opt)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("应投递 1 个 APK，实得 %d", n)
	}

	// 落盘路径：<root>/ap/ap01018/1.0.1/app-ap01018-release.apk
	want := filepath.Join(artifactRoot, "ap", "ap01018", "1.0.1", "app-ap01018-release.apk")
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("产物未落到预期路径 %s: %v", want, err)
	}
	if string(data) != "APKDATA" {
		t.Errorf("产物内容不符: %q", string(data))
	}

	if len(be.artifacts) != 1 {
		t.Fatalf("应登记 1 个产物，实得 %d", len(be.artifacts))
	}
	art := be.artifacts[0]
	if art.ApkURL != "https://console.example.com/apks/ap/ap01018/1.0.1/app-ap01018-release.apk" {
		t.Errorf("ApkURL 不符: %q", art.ApkURL)
	}
	if art.Flavor != "ap01018" || art.VersionName != "1.0.1" || art.FileName != "app-ap01018-release.apk" {
		t.Errorf("产物元信息不符: %+v", art)
	}
	if art.SizeBytes != int64(len("APKDATA")) {
		t.Errorf("SizeBytes 不符: %d", art.SizeBytes)
	}
	if len(art.SHA256) != 64 {
		t.Errorf("SHA256 应为 64 位十六进制，实得 %q", art.SHA256)
	}
}

// TestDeliverArtifactsRelativeURL 验证未配置 base-url 时登记相对路径（前导 /）。
func TestDeliverArtifactsRelativeURL(t *testing.T) {
	r := fakeRepoWithAPK(t, "gpgzmkk042", "app-gpgzmkk042-release.apk", "X")
	be := &fakeBackend{}
	job := &manifest.BuildJob{ID: 1, Brand: "gp", Flavors: []string{"gpgzmkk042"}, VersionName: "2.0.0"}
	opt := Options{ArtifactDir: t.TempDir()} // 无 base-url
	if _, err := deliverArtifacts(context.Background(), r, be, job, "2.0.0", opt); err != nil {
		t.Fatal(err)
	}
	if got := be.artifacts[0].ApkURL; got != "/gp/gpgzmkk042/2.0.0/app-gpgzmkk042-release.apk" {
		t.Errorf("相对 URL 不符: %q", got)
	}
}

// TestRunOnceEmptyQueue 验证 Once 模式遇空队列即返回（不阻塞）。
func TestRunOnceEmptyQueue(t *testing.T) {
	root := t.TempDir()
	r := &repo.Repo{Root: root}
	// 准备 local.properties 让 ensureSigning 通过（无环境 secret 路径）。
	writeSigningLocalProps(t, r)
	be := &fakeBackend{} // 空 jobs → ID=0
	err := Run(context.Background(), r, be, Options{Once: true, Source: nopSource{}})
	if err != nil {
		t.Fatalf("空队列 Once 不应报错: %v", err)
	}
}

// TestEnsureSigningFailsWithoutMaterial 验证既无环境 secret 又无 local.properties 时拒绝启动。
func TestEnsureSigningFailsWithoutMaterial(t *testing.T) {
	r := &repo.Repo{Root: t.TempDir()}
	err := Run(context.Background(), r, &fakeBackend{}, Options{Once: true, Source: nopSource{}})
	if err == nil {
		t.Fatal("缺少签名材料时应拒绝启动")
	}
}

// TestKeystoreFromEnvMaterializes 验证环境 secret 注入 local.properties 且不丢 sdk.dir。
func TestKeystoreFromEnvMaterializes(t *testing.T) {
	root := t.TempDir()
	r := &repo.Repo{Root: root}
	// keystore 文件需真实存在（verifyKeystoreFile 校验）。
	ksPath := filepath.Join(root, "release.jks")
	if err := os.WriteFile(ksPath, []byte("ks"), 0o600); err != nil {
		t.Fatal(err)
	}
	// 预置一个含 sdk.dir 的 local.properties，验证合并保留。
	if err := os.WriteFile(r.LocalProperties(), []byte("sdk.dir=/opt/android-sdk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ks := &Keystore{File: ksPath, StorePassword: "sp", KeyAlias: "rel", KeyPassword: "kp"}
	if err := ensureSigning(r, Options{Keystore: ks}); err != nil {
		t.Fatalf("注入应成功: %v", err)
	}
	props := readProps(r.LocalProperties())
	if props["sdk.dir"] != "/opt/android-sdk" {
		t.Errorf("sdk.dir 应保留: %q", props["sdk.dir"])
	}
	for _, k := range signingKeys {
		if props[k] == "" {
			t.Errorf("签名键 %s 应被注入", k)
		}
	}
}

// TestRunOnceFullLoop 端到端验证单任务全流程：claim→pull→build(桩)→deliver→回传 success。
// build 步骤用桩替代真实 gradlew：桩按 gradle 行为在 release 输出目录写出 APK。
func TestRunOnceFullLoop(t *testing.T) {
	root := t.TempDir()
	r := &repo.Repo{Root: root}
	// settings.gradle + channels/ 让 pull 渲染落地（RenderManifest 写 CSV）。
	if err := os.MkdirAll(filepath.Join(root, "channels"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.gradle"), []byte("// t"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSigningLocalProps(t, r)

	be := &fakeBackend{jobs: []*manifest.BuildJob{
		{ID: 99, Brand: "ap", Flavors: []string{"ap01018"}, TestEvents: false, VersionName: "1.0.1"},
	}}

	// manifest 源：返回一个 ap 渠道（pull 会写 CSV + bootstrap）。
	src := mapSource{m: &manifest.Manifest{
		Brand:        "ap",
		ConfigURL:    "https://cdn.example.com/cfg",
		BrandDomains: []string{"https://arenaplus.ph"},
		Channels: []manifest.Channel{
			{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "111", AppName: "AP", UseBrandDomains: true},
		},
	}}

	artifactRoot := t.TempDir()
	var gotBuildOpt build.Options
	opt := Options{
		Once:            true,
		Source:          src,
		ArtifactDir:     artifactRoot,
		ArtifactBaseURL: "/apks",
		buildFn: func(_ context.Context, rr *repo.Repo, bo build.Options) (*build.Result, error) {
			gotBuildOpt = bo
			// 模拟 gradlew：在每个 flavor 的 release 目录写出 APK。
			for _, f := range bo.Flavors {
				dir := rr.APKOutputDir(f)
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return nil, err
				}
				if err := os.WriteFile(filepath.Join(dir, "app-"+f+"-release.apk"), []byte("apk:"+f), 0o644); err != nil {
					return nil, err
				}
			}
			return &build.Result{LogTail: "BUILD SUCCESSFUL"}, nil
		},
	}

	if err := Run(context.Background(), r, be, opt); err != nil {
		t.Fatalf("全流程不应报错: %v", err)
	}

	// 1) pull 渲染：CSV 与 bootstrap.json 已写出。
	if _, err := os.Stat(r.ChannelsCSV("ap")); err != nil {
		t.Errorf("pull 未写出 CSV: %v", err)
	}
	if _, err := os.Stat(r.FlavorBootstrap("ap", "ap01018")); err != nil {
		t.Errorf("pull 未写出 bootstrap.json: %v", err)
	}

	// 2) build 收到了 job.versionName（透传 -PversionName 的前提）。
	if gotBuildOpt.VersionName != "1.0.1" {
		t.Errorf("build 应收到 job.versionName=1.0.1，实得 %q", gotBuildOpt.VersionName)
	}
	if got := gotBuildOpt.Args(); !contains(got, "-PversionName=1.0.1") {
		t.Errorf("gradle 参数应含 -PversionName=1.0.1，实得 %v", got)
	}

	// 3) 产物落到 <root>/ap/ap01018/1.0.1/ 并登记。
	want := filepath.Join(artifactRoot, "ap", "ap01018", "1.0.1", "app-ap01018-release.apk")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("产物未落到预期路径 %s: %v", want, err)
	}
	if len(be.artifacts) != 1 {
		t.Fatalf("应登记 1 个产物，实得 %d", len(be.artifacts))
	}
	if be.artifacts[0].ApkURL != "/apks/ap/ap01018/1.0.1/app-ap01018-release.apk" {
		t.Errorf("ApkURL 不符: %q", be.artifacts[0].ApkURL)
	}

	// 4) 末状态为 success。
	if got := be.statuses[len(be.statuses)-1].Status; got != manifest.StatusSuccess {
		t.Errorf("末状态应为 success，实得 %q", got)
	}
}

// TestRunOnceBuildFailureReported 验证打包失败时回传 failed 且不登记产物。
func TestRunOnceBuildFailureReported(t *testing.T) {
	root := t.TempDir()
	r := &repo.Repo{Root: root}
	if err := os.MkdirAll(filepath.Join(root, "channels"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.gradle"), []byte("// t"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSigningLocalProps(t, r)
	be := &fakeBackend{jobs: []*manifest.BuildJob{{ID: 5, Brand: "ap", Flavors: []string{"ap01018"}, VersionName: "1.0.1"}}}
	src := mapSource{m: &manifest.Manifest{Brand: "ap", BrandDomains: []string{"https://x"}, Channels: []manifest.Channel{{Flavor: "ap01018", ApplicationId: "com.arenaplus.ap01018", PalCode: "1", AppName: "A", UseBrandDomains: true}}}}
	opt := Options{
		Once: true, Source: src, ArtifactDir: t.TempDir(),
		buildFn: func(_ context.Context, _ *repo.Repo, bo build.Options) (*build.Result, error) {
			return &build.Result{LogTail: "FAILURE: task failed", ExitCode: 1}, errAssemble
		},
	}
	if err := Run(context.Background(), r, be, opt); err != nil {
		t.Fatalf("Once 模式下单任务失败不应让 Run 返回错误（失败已回传），实得: %v", err)
	}
	if got := be.statuses[len(be.statuses)-1].Status; got != manifest.StatusFailed {
		t.Errorf("末状态应为 failed，实得 %q", got)
	}
	if len(be.artifacts) != 0 {
		t.Errorf("失败时不应登记产物，实得 %d", len(be.artifacts))
	}
}

var errAssemble = errSentinel("gradlew 退出码 1")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// mapSource 是返回固定 manifest 的测试源。
type mapSource struct{ m *manifest.Manifest }

func (s mapSource) Manifest(_ context.Context, _ string) (*manifest.Manifest, error) { return s.m, nil }
func (s mapSource) DownloadResZip(_ context.Context, _, _ string) ([]byte, error)    { return nil, nil }

// nopSource 是 pull 阶段不被触达时的占位 manifest 源（Once 空队列下不会调用）。
type nopSource struct{}

func (nopSource) Manifest(_ context.Context, _ string) (*manifest.Manifest, error) {
	return &manifest.Manifest{}, nil
}
func (nopSource) DownloadResZip(_ context.Context, _, _ string) ([]byte, error) { return nil, nil }

// writeSigningLocalProps 写一个含齐 4 个签名键 + keystore 文件的 local.properties。
func writeSigningLocalProps(t *testing.T, r *repo.Repo) {
	t.Helper()
	ks := filepath.Join(r.Root, "release.jks")
	if err := os.WriteFile(ks, []byte("ks"), 0o600); err != nil {
		t.Fatal(err)
	}
	content := "KEYSTORE_FILE=" + ks + "\nKEYSTORE_PASSWORD=x\nKEY_ALIAS=a\nKEY_PASSWORD=y\n"
	if err := os.WriteFile(r.LocalProperties(), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
