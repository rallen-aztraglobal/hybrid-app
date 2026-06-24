package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/hybrid-app/server/internal/model"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("记录不存在")

// Repo 封装全部数据访问。
type Repo struct {
	db *gorm.DB
}

// New 创建 Repo。
func New(db *gorm.DB) *Repo { return &Repo{db: db} }

// DB 暴露底层连接（仅供 seed/迁移等少数场景）。
func (r *Repo) DB() *gorm.DB { return r.db }

// ---------- Brand ----------

// ListBrands 返回全部品牌（按 sort 升序），含域名与渠道计数。
func (r *Repo) ListBrands(ctx context.Context) ([]model.Brand, error) {
	var brands []model.Brand
	if err := r.db.WithContext(ctx).
		Preload("Domains", func(d *gorm.DB) *gorm.DB { return d.Order("position asc") }).
		Order("sort asc, id asc").Find(&brands).Error; err != nil {
		return nil, fmt.Errorf("查询品牌失败: %w", err)
	}
	return brands, nil
}

// GetBrandByCode 按 code 取品牌（含域名）。
func (r *Repo) GetBrandByCode(ctx context.Context, code string) (*model.Brand, error) {
	var b model.Brand
	err := r.db.WithContext(ctx).
		Preload("Domains", func(d *gorm.DB) *gorm.DB { return d.Order("position asc") }).
		Where("code = ?", code).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询品牌失败: %w", err)
	}
	return &b, nil
}

// GetBrandByID 按 id 取品牌。
func (r *Repo) GetBrandByID(ctx context.Context, id uint64) (*model.Brand, error) {
	var b model.Brand
	err := r.db.WithContext(ctx).First(&b, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询品牌失败: %w", err)
	}
	return &b, nil
}

// CountChannelsByBrand 返回每个 brand_id 的非归档渠道数量。
func (r *Repo) CountChannelsByBrand(ctx context.Context) (map[uint64]int64, error) {
	type row struct {
		BrandID uint64
		Cnt     int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).
		Select("brand_id, count(*) as cnt").
		Where("status <> ?", model.ChannelArchived).
		Group("brand_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计渠道数失败: %w", err)
	}
	m := make(map[uint64]int64, len(rows))
	for _, x := range rows {
		m[x.BrandID] = x.Cnt
	}
	return m, nil
}

// ReplaceBrandDomains 事务性替换某品牌的全部域名。
func (r *Repo) ReplaceBrandDomains(ctx context.Context, brandID uint64, domains []model.BrandDomain) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("brand_id = ?", brandID).Delete(&model.BrandDomain{}).Error; err != nil {
			return fmt.Errorf("清空品牌域名失败: %w", err)
		}
		if len(domains) == 0 {
			return nil
		}
		for i := range domains {
			domains[i].BrandID = brandID
		}
		if err := tx.Create(&domains).Error; err != nil {
			return fmt.Errorf("写入品牌域名失败: %w", err)
		}
		return nil
	})
}

// ---------- Channel ----------

// ChannelFilter 渠道列表筛选条件。
type ChannelFilter struct {
	BrandCode string
	Status    string
	Q         string
	Page      int
	PageSize  int
}

// ListChannels 分页/筛选/搜索渠道。返回当页数据与总数。
func (r *Repo) ListChannels(ctx context.Context, f ChannelFilter) ([]model.Channel, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Channel{}).
		Joins("JOIN brand ON brand.id = channel.brand_id")

	if f.BrandCode != "" {
		q = q.Where("brand.code = ?", f.BrandCode)
	}
	if f.Status != "" {
		q = q.Where("channel.status = ?", f.Status)
	}
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("channel.flavor_name LIKE ? OR channel.application_id LIKE ? OR channel.app_name LIKE ? OR channel.pal_code LIKE ?",
			like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计渠道总数失败: %w", err)
	}

	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 500 {
		f.PageSize = 50
	}
	var list []model.Channel
	if err := q.
		Preload("Brand").
		Preload("Domains", func(d *gorm.DB) *gorm.DB { return d.Order("position asc") }).
		Order("channel.id asc").
		Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询渠道失败: %w", err)
	}
	return list, total, nil
}

// GetChannel 按 id 取渠道（含品牌与域名）。
func (r *Repo) GetChannel(ctx context.Context, id uint64) (*model.Channel, error) {
	var ch model.Channel
	err := r.db.WithContext(ctx).
		Preload("Brand").
		Preload("Domains", func(d *gorm.DB) *gorm.DB { return d.Order("position asc") }).
		First(&ch, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询渠道失败: %w", err)
	}
	return &ch, nil
}

// GetChannelByApplicationID 按 application_id 取渠道（供 /api/app/config 使用，ADR-0009：解析键改为 appId）。
func (r *Repo) GetChannelByApplicationID(ctx context.Context, appID string) (*model.Channel, error) {
	var ch model.Channel
	err := r.db.WithContext(ctx).
		Preload("Brand", func(d *gorm.DB) *gorm.DB { return d.Preload("Domains") }).
		Preload("Domains", func(d *gorm.DB) *gorm.DB { return d.Order("position asc") }).
		Where("application_id = ?", appID).First(&ch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("按 applicationId 查询渠道失败: %w", err)
	}
	return &ch, nil
}

// CountChannels 返回渠道总数（用于判断是否「首次部署、尚无渠道」以触发自动导入）。
func (r *Repo) CountChannels(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("统计渠道数失败: %w", err)
	}
	return n, nil
}

// CreateChannel 插入渠道。
func (r *Repo) CreateChannel(ctx context.Context, ch *model.Channel) error {
	if err := r.db.WithContext(ctx).Create(ch).Error; err != nil {
		return fmt.Errorf("创建渠道失败: %w", err)
	}
	return nil
}

// UpdateChannel 保存渠道字段。
func (r *Repo) UpdateChannel(ctx context.Context, ch *model.Channel) error {
	if err := r.db.WithContext(ctx).Save(ch).Error; err != nil {
		return fmt.Errorf("更新渠道失败: %w", err)
	}
	return nil
}

// UpdateChannelFields 局部更新渠道指定字段。
func (r *Repo) UpdateChannelFields(ctx context.Context, id uint64, fields map[string]any) error {
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("更新渠道字段失败: %w", err)
	}
	return nil
}

// UniqueCheck 唯一性检查参数；ExcludeID>0 时排除自身（用于更新）。
// ADR-0009：仅 applicationId 与 (brand, flavor) 唯一；pal_code 不再查重。
type UniqueCheck struct {
	ApplicationID string
	BrandID       uint64
	FlavorName    string
	ExcludeID     uint64
}

// UniqueConflicts 返回与给定值冲突的字段名清单（applicationId/flavor）。
// 空 slice 表示无冲突。这是「重复必须被拒」的核心查询。
func (r *Repo) UniqueConflicts(ctx context.Context, u UniqueCheck) ([]string, error) {
	var conflicts []string

	check := func(field string, cond *gorm.DB) error {
		if u.ExcludeID > 0 {
			cond = cond.Where("id <> ?", u.ExcludeID)
		}
		var cnt int64
		if err := cond.Count(&cnt).Error; err != nil {
			return fmt.Errorf("唯一性检查(%s)失败: %w", field, err)
		}
		if cnt > 0 {
			conflicts = append(conflicts, field)
		}
		return nil
	}

	if u.ApplicationID != "" {
		if err := check("applicationId",
			r.db.WithContext(ctx).Model(&model.Channel{}).Where("application_id = ?", u.ApplicationID)); err != nil {
			return nil, err
		}
	}
	if u.BrandID > 0 && u.FlavorName != "" {
		if err := check("flavor",
			r.db.WithContext(ctx).Model(&model.Channel{}).Where("brand_id = ? AND flavor_name = ?", u.BrandID, u.FlavorName)); err != nil {
			return nil, err
		}
	}
	return conflicts, nil
}

// ReplaceChannelDomains 事务性替换某渠道的域名覆盖。
func (r *Repo) ReplaceChannelDomains(ctx context.Context, channelID uint64, domains []model.ChannelDomain) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", channelID).Delete(&model.ChannelDomain{}).Error; err != nil {
			return fmt.Errorf("清空渠道域名失败: %w", err)
		}
		if len(domains) == 0 {
			return nil
		}
		for i := range domains {
			domains[i].ChannelID = channelID
		}
		if err := tx.Create(&domains).Error; err != nil {
			return fmt.Errorf("写入渠道域名失败: %w", err)
		}
		return nil
	})
}

// ---------- AdminUser ----------

// GetUserByUsername 按用户名取账号。
func (r *Repo) GetUserByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var u model.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询账号失败: %w", err)
	}
	return &u, nil
}

// CountUsers 返回账号总数（用于决定是否 bootstrap admin）。
func (r *Repo) CountUsers(ctx context.Context) (int64, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.AdminUser{}).Count(&n).Error; err != nil {
		return 0, fmt.Errorf("统计账号失败: %w", err)
	}
	return n, nil
}

// CreateUser 新建账号。
func (r *Repo) CreateUser(ctx context.Context, u *model.AdminUser) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("创建账号失败: %w", err)
	}
	return nil
}

// ---------- BuildRecord ----------

// CreateBuildRecord 写入构建记录。
func (r *Repo) CreateBuildRecord(ctx context.Context, rec *model.BuildRecord) error {
	if err := r.db.WithContext(ctx).Create(rec).Error; err != nil {
		return fmt.Errorf("写入构建记录失败: %w", err)
	}
	return nil
}

// BuildRecordFilter 构建记录列表筛选。
type BuildRecordFilter struct {
	BrandCode string
	Status    string
	Limit     int
}

// ListBuildRecords 按品牌/状态查构建历史（倒序），预加载产物。
func (r *Repo) ListBuildRecords(ctx context.Context, f BuildRecordFilter) ([]model.BuildRecord, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	q := r.db.WithContext(ctx).Model(&model.BuildRecord{}).
		Preload("Artifacts", func(d *gorm.DB) *gorm.DB { return d.Order("flavor asc") }).
		Order("id desc").Limit(f.Limit)
	if f.BrandCode != "" {
		q = q.Where("brand_code = ?", f.BrandCode)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var list []model.BuildRecord
	if err := q.Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询构建记录失败: %w", err)
	}
	return list, nil
}

// GetBuildRecord 按 id 取单条构建记录（含产物）。
func (r *Repo) GetBuildRecord(ctx context.Context, id uint64) (*model.BuildRecord, error) {
	var rec model.BuildRecord
	err := r.db.WithContext(ctx).
		Preload("Artifacts", func(d *gorm.DB) *gorm.DB { return d.Order("flavor asc") }).
		First(&rec, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询构建记录失败: %w", err)
	}
	return &rec, nil
}

// ClaimQueuedBuild 让 runner 原子领取最早的一条 queued 记录：置 running + 记 started_at。
// 用事务 + 行级条件更新避免多 runner 抢同一条；无可领取任务时返回 (nil, nil)。
func (r *Repo) ClaimQueuedBuild(ctx context.Context, runner string) (*model.BuildRecord, error) {
	var rec model.BuildRecord
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取最早的一条 queued。
		if err := tx.Where("status = ?", model.BuildQueued).
			Order("id asc").First(&rec).Error; err != nil {
			return err // ErrRecordNotFound 由外层判定为「无任务」
		}
		// 条件更新：仅当仍是 queued 才置 running（防并发重复领取）。
		now := time.Now()
		res := tx.Model(&model.BuildRecord{}).
			Where("id = ? AND status = ?", rec.ID, model.BuildQueued).
			Updates(map[string]any{"status": model.BuildRunning, "operator": runner, "started_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 被别的 runner 抢走，回滚后让上层重试/视为无任务。
			return gorm.ErrRecordNotFound
		}
		rec.Status = model.BuildRunning
		rec.Operator = runner
		rec.StartedAt = now
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // 无可领取任务
	}
	if err != nil {
		return nil, fmt.Errorf("领取构建任务失败: %w", err)
	}
	return &rec, nil
}

// UpdateBuildFields 局部更新构建记录字段。
func (r *Repo) UpdateBuildFields(ctx context.Context, id uint64, fields map[string]any) error {
	res := r.db.WithContext(ctx).Model(&model.BuildRecord{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("更新构建记录失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendBuildLog 把一段日志追加到构建记录的 log 字段（runner 增量上报）。
// 用 DB 端字符串拼接避免读改写竞态；按方言选择拼接语法（mysql 用 CONCAT，sqlite 用 ||）。
func (r *Repo) AppendBuildLog(ctx context.Context, id uint64, chunk string) error {
	var expr clause.Expr
	if r.db.Dialector.Name() == "mysql" {
		expr = gorm.Expr("CONCAT(COALESCE(log, ''), ?)", chunk)
	} else {
		expr = gorm.Expr("COALESCE(log, '') || ?", chunk)
	}
	res := r.db.WithContext(ctx).Model(&model.BuildRecord{}).Where("id = ?", id).
		Update("log", expr)
	if res.Error != nil {
		return fmt.Errorf("追加构建日志失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateBuildArtifact 写入一条 APK 产物记录。
func (r *Repo) CreateBuildArtifact(ctx context.Context, a *model.BuildArtifact) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		return fmt.Errorf("写入构建产物失败: %w", err)
	}
	return nil
}

// LatestArtifactURLsByFlavors 批量取一批 flavor 各自「最近一次成功构建」的 APK URL（供渠道列表 latestApkUrl 列）。
// 返回 map[flavor]apkUrl，没有成功产物的 flavor 不在 map 中。空入参返回空 map。
func (r *Repo) LatestArtifactURLsByFlavors(ctx context.Context, flavors []string) (map[string]string, error) {
	out := map[string]string{}
	if len(flavors) == 0 {
		return out, nil
	}
	// 每个 flavor 取 created_at 最大的成功产物：用子查询定位最大时间再回连。
	type row struct {
		Flavor string
		ApkURL string
	}
	var rows []row
	sub := r.db.WithContext(ctx).Model(&model.BuildArtifact{}).
		Select("build_artifact.flavor AS flavor, max(build_artifact.created_at) AS mx").
		Joins("JOIN build_record br ON br.id = build_artifact.build_record_id").
		Where("br.status = ? AND build_artifact.flavor IN ?", model.BuildSuccess, flavors).
		Group("build_artifact.flavor")
	if err := r.db.WithContext(ctx).
		Table("build_artifact ba").
		Select("ba.flavor AS flavor, ba.apk_url AS apk_url").
		Joins("JOIN (?) t ON t.flavor = ba.flavor AND t.mx = ba.created_at", sub).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询最新产物失败: %w", err)
	}
	for _, x := range rows {
		// 同一时间戳可能多条（理论上罕见），保留先到的即可。
		if _, ok := out[x.Flavor]; !ok {
			out[x.Flavor] = x.ApkURL
		}
	}
	return out, nil
}

// LatestArtifactForFlavor 取某 flavor 最近一次成功构建的 APK 产物（按 created_at 倒序）。
// 仅统计来自 success 记录的产物。无则返回 (nil, nil)。
func (r *Repo) LatestArtifactForFlavor(ctx context.Context, flavor string) (*model.BuildArtifact, error) {
	var a model.BuildArtifact
	err := r.db.WithContext(ctx).
		Joins("JOIN build_record br ON br.id = build_artifact.build_record_id").
		Where("build_artifact.flavor = ? AND br.status = ?", flavor, model.BuildSuccess).
		Order("build_artifact.created_at desc, build_artifact.id desc").
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 flavor=%s 最新产物失败: %w", flavor, err)
	}
	return &a, nil
}

// ---------- AuditLog ----------

// CreateAuditLog 写入审计日志（失败不应阻断主流程，调用方决定如何处理 err）。
func (r *Repo) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("写入审计日志失败: %w", err)
	}
	return nil
}

// ---------- DomainHealth ----------

// SaveDomainHealth 写入一次域名巡检结果。
func (r *Repo) SaveDomainHealth(ctx context.Context, h *model.DomainHealth) error {
	if err := r.db.WithContext(ctx).Create(h).Error; err != nil {
		return fmt.Errorf("写入域名健康记录失败: %w", err)
	}
	return nil
}

// LatestDomainHealth 返回每个 URL 最近一次巡检结果。
func (r *Repo) LatestDomainHealth(ctx context.Context) ([]model.DomainHealth, error) {
	// 子查询取每个 url 的最大 checked_at。
	var list []model.DomainHealth
	sub := r.db.WithContext(ctx).Model(&model.DomainHealth{}).
		Select("url, max(checked_at) as mx").Group("url")
	if err := r.db.WithContext(ctx).
		Table("domain_health dh").
		Joins("JOIN (?) t ON t.url = dh.url AND t.mx = dh.checked_at", sub).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("查询最新域名健康失败: %w", err)
	}
	return list, nil
}

// AllConfiguredDomains 返回库里配置过的所有去重域名 URL（品牌 + 渠道覆盖），供巡检使用。
func (r *Repo) AllConfiguredDomains(ctx context.Context) ([]string, error) {
	var brandURLs, chURLs []string
	if err := r.db.WithContext(ctx).Model(&model.BrandDomain{}).
		Where("enabled = ?", true).Distinct().Pluck("url", &brandURLs).Error; err != nil {
		return nil, fmt.Errorf("查询品牌域名失败: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.ChannelDomain{}).
		Where("enabled = ?", true).Distinct().Pluck("url", &chURLs).Error; err != nil {
		return nil, fmt.Errorf("查询渠道域名失败: %w", err)
	}
	seen := map[string]bool{}
	var out []string
	for _, u := range append(brandURLs, chURLs...) {
		if !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out, nil
}
