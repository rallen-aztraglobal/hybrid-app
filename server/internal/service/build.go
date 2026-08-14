package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hybrid-app/server/internal/model"
	"github.com/hybrid-app/server/internal/repo"
)

// versionNameRe 校验 versionName 形如 X.Y.Z（三段纯数字，ADR-0008）。
var versionNameRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// validVersionName 返回 versionName 是否形如 X.Y.Z。
func validVersionName(v string) bool { return versionNameRe.MatchString(v) }

// semver 是解析后的 X.Y.Z 三段版本号（均为非负整数），供数值比较而非字符串字典序比较
// （字典序会把 "1.10.0" 误判为小于 "1.2.0"）。
type semver [3]int

// parseSemver 解析形如 X.Y.Z 的版本号。历史脏数据/非法格式返回 ok=false，绝不 panic。
func parseSemver(v string) (sv semver, ok bool) {
	if !validVersionName(v) {
		return semver{}, false
	}
	parts := strings.SplitN(v, ".", 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, false
		}
		sv[i] = n
	}
	return sv, true
}

// compareSemver 返回 a 相对 b 的大小：-1 表示 a<b，0 表示相等，1 表示 a>b（逐段数值比较，非字符串比较）。
func compareSemver(a, b semver) int {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// CurrentVersion 取某品牌全部 success 构建记录里「语义版本最高」的 versionName
// （不是按 created_at/id 取最新一条——后者可能因手误提交了一个更低的版本号）。
// 忽略无法解析的历史脏版本号，不中断查询；一个合法版本都没有时 found=false，
// 视为「当前版本不存在」，据此放行第一次构建。
//
// 这是 CreateBuildJob 版本校验与 GetCurrentVersion（GET /api/build/current-version，
// 供打包中心展示）共用的唯一实现——前端不应该、也不需要自己在一份可能被分页截断的
// 构建记录列表上重新算一遍「最高版本」，两处算出来的结果必须永远一致。
func (s *Service) CurrentVersion(ctx context.Context, brandCode string) (name string, found bool) {
	names, err := s.repo.ListSuccessfulVersionNames(ctx, brandCode)
	if err != nil {
		// 查询失败不应阻断正常打包流程：按「当前版本不存在」处理，跳过本次版本比较（其余校验仍然生效）。
		log.Printf("[build] 查询品牌 %s 当前版本失败，本次跳过版本比较: %v", brandCode, err)
		return "", false
	}
	var best semver
	for _, n := range names {
		sv, ok := parseSemver(n)
		if !ok {
			log.Printf("[build] 品牌 %s 存在无法解析的历史版本号 %q，已忽略", brandCode, n)
			continue
		}
		if !found || compareSemver(sv, best) > 0 {
			name, best, found = n, sv, true
		}
	}
	return name, found
}

// CreateBuildJobInput 是 POST /api/build/jobs 入参（Web「打包中心」触发）。
type CreateBuildJobInput struct {
	Brand       string   `json:"brand" validate:"required"`
	Flavors     []string `json:"flavors" validate:"required"`
	VersionName string   `json:"versionName" validate:"required"`
	TestEvents  bool     `json:"testEvents"`
	Name        string   `json:"name"` // 可空，默认 <brand>-<versionName>-<YYYYMMDD-HHmm>
}

// defaultJobName 生成默认任务名 <brand>-<versionName>-<YYYYMMDD-HHmm>（ADR-0008）。
func defaultJobName(brand, versionName string, t time.Time) string {
	return fmt.Sprintf("%s-%s-%s", brand, versionName, t.Format("20060102-1504"))
}

// CreateBuildJob 校验并入队一条构建任务（状态 queued），等待构建机领取。
// 校验：versionName X.Y.Z、品牌存在、flavors 非空且均属于该品牌的 enabled 渠道。
func (s *Service) CreateBuildJob(ctx context.Context, in CreateBuildJobInput) (*model.BuildRecord, error) {
	in.Brand = strings.TrimSpace(in.Brand)
	in.VersionName = strings.TrimSpace(in.VersionName)
	in.Name = strings.TrimSpace(in.Name)

	if in.Brand == "" || len(in.Flavors) == 0 || in.VersionName == "" {
		return nil, errBadRequest("brand / flavors / versionName 必填")
	}
	if !validVersionName(in.VersionName) {
		return nil, errBadRequest(fmt.Sprintf("versionName %q 非法（应形如 X.Y.Z）", in.VersionName))
	}
	brand, err := s.repo.GetBrandByCode(ctx, in.Brand)
	if err != nil {
		return nil, errBadRequest(fmt.Sprintf("品牌 %q 不存在", in.Brand))
	}

	// 版本号不能低于该品牌当前（最高语义版本的成功构建）版本；相等允许，更高允许（YW 需求）。
	// newSv 必然可解析：上面 validVersionName 已校验过 in.VersionName 的格式。
	newSv, _ := parseSemver(in.VersionName)
	if current, ok := s.CurrentVersion(ctx, brand.Code); ok {
		if curSv, curOK := parseSemver(current); curOK && compareSemver(newSv, curSv) < 0 {
			return nil, errBadRequest(fmt.Sprintf("versionName %q 不能低于当前版本 %s", in.VersionName, current))
		}
	}

	// 去重 + 归一 flavors，并校验都属于该品牌的 enabled 渠道（避免给构建机不存在/未启用的 flavor）。
	known, _, err := s.repo.ListChannels(ctx, repo.ChannelFilter{
		BrandCode: brand.Code, Status: model.ChannelEnabled, PageSize: 500,
	})
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]bool, len(known))
	for i := range known {
		enabled[known[i].FlavorName] = true
	}
	seen := map[string]bool{}
	flavors := make([]string, 0, len(in.Flavors))
	for _, f := range in.Flavors {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		if !enabled[f] {
			return nil, errBadRequest(fmt.Sprintf("flavor %q 不是品牌 %s 下的 enabled 渠道", f, brand.Code))
		}
		seen[f] = true
		flavors = append(flavors, f)
	}
	if len(flavors) == 0 {
		return nil, errBadRequest("flavors 去重后为空")
	}

	name := in.Name
	if name == "" {
		name = defaultJobName(brand.Code, in.VersionName, time.Now())
	}
	flavorsJSON, _ := json.Marshal(flavors)

	rec := &model.BuildRecord{
		Name:        name,
		BrandCode:   brand.Code,
		Flavors:     string(flavorsJSON),
		TestEvents:  in.TestEvents,
		Status:      model.BuildQueued,
		VersionName: in.VersionName,
		ApkURLs:     "[]",
	}
	if err := s.repo.CreateBuildRecord(ctx, rec); err != nil {
		return nil, err
	}
	return s.repo.GetBuildRecord(ctx, rec.ID)
}

// ClaimBuild 让构建机领取下一条 queued 任务（原子）。无任务时返回 (nil, nil)。
func (s *Service) ClaimBuild(ctx context.Context, runner string) (*model.BuildRecord, error) {
	runner = strings.TrimSpace(runner)
	if runner == "" {
		runner = "runner"
	}
	rec, err := s.repo.ClaimQueuedBuild(ctx, runner)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return s.repo.GetBuildRecord(ctx, rec.ID)
}

// ReportBuildStatusInput runner 上报构建状态（running/success/failed）。
type ReportBuildStatusInput struct {
	Status     string `json:"status" validate:"required"`
	LogExcerpt string `json:"logExcerpt"` // 可选：末段日志摘要，列表展示用
}

// ReportBuildStatus runner 更新任务状态；进入终态（success/failed）时记 finished_at。
func (s *Service) ReportBuildStatus(ctx context.Context, id uint64, in ReportBuildStatusInput) (*model.BuildRecord, error) {
	switch in.Status {
	case model.BuildRunning, model.BuildSuccess, model.BuildFailed:
	default:
		return nil, errBadRequest("status 非法（running/success/failed）")
	}
	fields := map[string]any{"status": in.Status}
	if in.LogExcerpt != "" {
		fields["log_excerpt"] = clamp(in.LogExcerpt, 4000)
	}
	if in.Status == model.BuildSuccess || in.Status == model.BuildFailed {
		fields["finished_at"] = time.Now()
	}
	if err := s.repo.UpdateBuildFields(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetBuildRecord(ctx, id)
}

// AppendBuildLog runner 增量上报日志（追加到 log 字段）。
func (s *Service) AppendBuildLog(ctx context.Context, id uint64, chunk string) error {
	if chunk == "" {
		return nil
	}
	return s.repo.AppendBuildLog(ctx, id, chunk)
}

// AddBuildArtifactInput runner 上报一个产出的 APK。
type AddBuildArtifactInput struct {
	Flavor      string `json:"flavor" validate:"required"`
	VersionName string `json:"versionName"`
	ApkURL      string `json:"apkUrl" validate:"required"`
	Size        int64  `json:"size"`
}

// AddBuildArtifact 记录一条 APK 产物，并把其 URL 汇总进 build_record.apk_urls（兼容旧字段/列表展示）。
func (s *Service) AddBuildArtifact(ctx context.Context, recordID uint64, in AddBuildArtifactInput) (*model.BuildArtifact, error) {
	in.Flavor = strings.TrimSpace(in.Flavor)
	in.ApkURL = strings.TrimSpace(in.ApkURL)
	if in.Flavor == "" || in.ApkURL == "" {
		return nil, errBadRequest("flavor / apkUrl 必填")
	}
	rec, err := s.repo.GetBuildRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	ver := strings.TrimSpace(in.VersionName)
	if ver == "" {
		ver = rec.VersionName
	}
	a := &model.BuildArtifact{
		BuildRecordID: recordID,
		Flavor:        in.Flavor,
		VersionName:   ver,
		ApkURL:        in.ApkURL,
		Size:          in.Size,
	}
	if err := s.repo.CreateBuildArtifact(ctx, a); err != nil {
		return nil, err
	}

	// 把 apkUrl 追加进汇总数组（容错：解析失败则重建）。
	var urls []string
	if rec.ApkURLs != "" {
		_ = json.Unmarshal([]byte(rec.ApkURLs), &urls)
	}
	urls = append(urls, in.ApkURL)
	if b, err := json.Marshal(urls); err == nil {
		_ = s.repo.UpdateBuildFields(ctx, recordID, map[string]any{"apk_urls": string(b)})
	}
	return a, nil
}

// ListBuildRecords 构建历史（按品牌/状态筛选）。
func (s *Service) ListBuildRecords(ctx context.Context, brandCode, status string, limit int) ([]model.BuildRecord, error) {
	return s.repo.ListBuildRecords(ctx, repo.BuildRecordFilter{BrandCode: brandCode, Status: status, Limit: limit})
}

// GetBuildRecord 单条构建记录（含产物）。
func (s *Service) GetBuildRecord(ctx context.Context, id uint64) (*model.BuildRecord, error) {
	return s.repo.GetBuildRecord(ctx, id)
}

// BuildLogSegment 是分段日志返回（offset 之后的内容 + 新游标 + 是否已结束）。
type BuildLogSegment struct {
	Offset int    `json:"offset"` // 本段起始字节偏移
	Next   int    `json:"next"`   // 下次请求应传的 offset
	Log    string `json:"log"`    // [offset, next) 区间的日志
	Done   bool   `json:"done"`   // 构建是否已进入终态（前端据此停止轮询）
}

// BuildLog 返回从 offset 起的日志分段，供前端流式/分段拉取（轮询 next 直到 done）。
func (s *Service) BuildLog(ctx context.Context, id uint64, offset int) (*BuildLogSegment, error) {
	rec, err := s.repo.GetBuildRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	full := rec.Log
	if offset < 0 {
		offset = 0
	}
	if offset > len(full) {
		offset = len(full)
	}
	seg := full[offset:]
	done := rec.Status == model.BuildSuccess || rec.Status == model.BuildFailed
	return &BuildLogSegment{
		Offset: offset,
		Next:   offset + len(seg),
		Log:    seg,
		Done:   done,
	}, nil
}

// LatestApkForChannel 取某渠道（按其 flavor）最近一次成功构建的 APK 产物。无则返回 (nil, nil)。
func (s *Service) LatestApkForChannel(ctx context.Context, channelID uint64) (*model.BuildArtifact, error) {
	ch, err := s.repo.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return s.repo.LatestArtifactForFlavor(ctx, ch.FlavorName)
}

// clamp 截断字符串到最多 n 字节（保留末段，列表摘要用）。
func clamp(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
