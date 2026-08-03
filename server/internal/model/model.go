// Package model 定义渠道中台的全部持久化实体（对应 docs/admin/01-architecture.md §4）。
// 这些 GORM 模型同时用于：① 开发期 AutoMigrate（sqlite/mysql）；② 生产期由 golang-migrate 的 SQL 建表，
// 模型仅作读写映射。两边表名/列名保持一致。
package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// 渠道状态枚举。
const (
	ChannelEnabled  = "enabled"
	ChannelDisabled = "disabled"
	ChannelArchived = "archived"
)

// 应用商店状态枚举。
const (
	StoreEnabled  = "enabled"
	StoreDisabled = "disabled"
)

// 账号角色枚举（RBAC）。只有两档：admin 全部权限；user 除 Store/系统设置与渠道归档外的全部日常操作。
const (
	RoleAdmin = "admin" // 全部权限（含系统设置/商店管理/渠道归档）
	RoleUser  = "user"  // 日常业务操作：渠道读写、域名、打包、推送等；不含系统设置与渠道归档
)

// 域名健康状态枚举。
const (
	HealthOK       = "ok"
	HealthDown     = "down"
	HealthHijacked = "hijacked"
	HealthUnknown  = "unknown"
)

// Brand 大渠道（品牌）。code 全局唯一：ap/bp/gp。
// PackagePrefix 是 applicationId 的包前缀（ap→com.arenaplus / bp→com.bingoplus / gp→com.gamezone），
// 渠道 applicationId 由 PackagePrefix + "." + flavor 派生（ADR-0009），不手填。
type Brand struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code          string    `gorm:"column:code;type:varchar(16);not null;uniqueIndex" json:"code"`
	Name          string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	PackagePrefix string    `gorm:"column:package_prefix;type:varchar(128);not null;default:''" json:"packagePrefix"`
	Scheme        string    `gorm:"column:scheme;type:varchar(32);not null" json:"scheme"`
	HMSEnabled    bool      `gorm:"column:hms_enabled;not null;default:false" json:"hmsEnabled"`
	AccentColor   string    `gorm:"column:accent_color;type:varchar(16)" json:"accentColor"`
	Sort          int       `gorm:"column:sort;not null;default:0" json:"sort"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Domains  []BrandDomain `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE" json:"domains,omitempty"`
	Channels []Channel     `gorm:"foreignKey:BrandID" json:"-"`
}

// DeriveApplicationID 按 ADR-0009 从品牌包前缀与 flavor 派生 applicationId：<package_prefix>.<flavor>。
// 这是 applicationId 唯一的事实来源——不信任外部传入值。
// 商店后缀支持：flavor 中的 "_" 会被转换为 "."，使商店包 flavor（如 <base>_<storeCode>）
// 派生出点号分段的 appId（如 prefix.<base>.<storeCode>）。老数据 flavor 无下划线时结果不变（向后兼容）。
func (b *Brand) DeriveApplicationID(flavor string) string {
	return b.PackagePrefix + "." + strings.ReplaceAll(flavor, "_", ".")
}

func (Brand) TableName() string { return "brand" }

// BrandDomain 品牌级默认域名（小渠道默认继承）。position 0=主，1..3=备用。
type BrandDomain struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BrandID  uint64 `gorm:"column:brand_id;not null;uniqueIndex:uk_brand_pos" json:"brandId"`
	Position int    `gorm:"column:position;not null;uniqueIndex:uk_brand_pos" json:"position"`
	URL      string `gorm:"column:url;type:varchar(255);not null" json:"url"`
	Enabled  bool   `gorm:"column:enabled;not null;default:true" json:"enabled"`
}

func (BrandDomain) TableName() string { return "brand_domain" }

// Channel 小渠道（= 一个 Gradle product flavor）。
// 唯一性约束（ADR-0009）：application_id 全局唯一（派生自 <品牌包前缀>.<flavor>）、(brand_id, flavor_name) 唯一。
// pal_code 不再全局唯一：营销 palcode 可跨品牌重复，仅作 URL 参数 /?palcode= 与编译期烧录。
type Channel struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BrandID         uint64 `gorm:"column:brand_id;not null;uniqueIndex:uk_brand_flavor" json:"brandId"`
	FlavorName      string `gorm:"column:flavor_name;type:varchar(64);not null;uniqueIndex:uk_brand_flavor" json:"flavorName"`
	ApplicationID   string `gorm:"column:application_id;type:varchar(128);not null;uniqueIndex" json:"applicationId"`
	PalCode         string `gorm:"column:pal_code;type:varchar(64);not null;index" json:"palCode"`
	AppName         string `gorm:"column:app_name;type:varchar(128);not null" json:"appName"`
	Status          string `gorm:"column:status;type:varchar(16);not null;default:enabled" json:"status"`
	UseBrandDomains bool   `gorm:"column:use_brand_domains;not null;default:true" json:"useBrandDomains"`
	IconMasterURL   string `gorm:"column:icon_master_url;type:varchar(255)" json:"iconMasterUrl"`
	IconSetURL      string `gorm:"column:icon_set_url;type:varchar(255)" json:"iconSetUrl"`
	SplashURL       string `gorm:"column:splash_url;type:varchar(255)" json:"splashUrl"`
	Remark          string `gorm:"column:remark;type:varchar(255)" json:"remark"`
	CreatedBy       uint64 `gorm:"column:created_by" json:"createdBy"`

	// StoreID 可选：渠道所属应用商店（华为/小米/Oppo 等）。非空时 flavor 形如 <base>_<store.Code>，
	// applicationId 据此派生出点号分段（见 Brand.DeriveApplicationID）。
	StoreID *uint64 `gorm:"column:store_id;index" json:"storeId,omitempty"`
	Store   *Store  `gorm:"foreignKey:StoreID" json:"store,omitempty"`

	// AdjustAppToken / AdjustEvents：Adjust 归因绑定（ADR-0013 / docs/admin/08-adjust.md §2）。
	// 均可空 = 该渠道未绑定 Adjust（合法状态，打包时不集成、不发事件）。
	// JSON 字段名 adjustAppToken / adjustEvents 与本结构体其余字段一致走 camelCase；
	// DB 列名 adjust_app_token / adjust_events 保持 snake_case（gorm column 标签，见下）不变。
	// AdjustAppToken 用指针：列允许 NULL，指针可原生对应 NULL/无值，避免历史行 NULL 的扫描问题。
	AdjustAppToken *string      `gorm:"column:adjust_app_token;type:varchar(64)" json:"adjustAppToken,omitempty"`
	AdjustEvents   AdjustEvents `gorm:"column:adjust_events;type:text" json:"adjustEvents,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Brand   *Brand          `gorm:"foreignKey:BrandID" json:"brand,omitempty"`
	Domains []ChannelDomain `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"domains,omitempty"`

	// LatestApkURL 非持久化：按 flavor 取最近一次成功构建的 APK 下载地址（ADR-0008，列表/卡片「下载最新包」）。
	LatestApkURL string `gorm:"-" json:"latestApkUrl,omitempty"`
}

func (Channel) TableName() string { return "channel" }

// ChannelDomain 小渠道级域名覆盖（use_brand_domains=false 时生效）。
type ChannelDomain struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID uint64 `gorm:"column:channel_id;not null;uniqueIndex:uk_ch_pos" json:"channelId"`
	Position  int    `gorm:"column:position;not null;uniqueIndex:uk_ch_pos" json:"position"`
	URL       string `gorm:"column:url;type:varchar(255);not null" json:"url"`
	Enabled   bool   `gorm:"column:enabled;not null;default:true" json:"enabled"`
}

func (ChannelDomain) TableName() string { return "channel_domain" }

// Store 应用商店（华为/小米/Oppo 等）。code 全局唯一、小写字母数字，一旦使用（被渠道引用）即不可改，
// 因为它已进入 applicationId 分段（见 Brand.DeriveApplicationID）。
type Store struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"column:code;type:varchar(16);not null;uniqueIndex" json:"code"`
	Name      string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Sort      int       `gorm:"column:sort;not null;default:0" json:"sort"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:enabled" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (Store) TableName() string { return "store" }

// DomainHealth 域名健康巡检结果（仅监控展示，不直接决定 APK 行为）。
type DomainHealth struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	URL       string    `gorm:"column:url;type:varchar(255);not null;index:idx_url_time" json:"url"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:unknown" json:"status"`
	HTTPCode  int       `gorm:"column:http_code" json:"httpCode"`
	LatencyMs int       `gorm:"column:latency_ms" json:"latencyMs"`
	CheckedAt time.Time `gorm:"column:checked_at;autoCreateTime;index:idx_url_time" json:"checkedAt"`
}

func (DomainHealth) TableName() string { return "domain_health" }

// 构建任务/记录状态机枚举（ADR-0008）。
const (
	BuildQueued  = "queued"  // 已入队，等待 runner 领取
	BuildRunning = "running" // runner 已领取，构建中
	BuildSuccess = "success" // 构建成功
	BuildFailed  = "failed"  // 构建失败
)

// BuildRecord 构建记录。一次「打包」= 一条 build_record，承载状态机（queued/running/success/failed）与日志。
// 产物以 build_artifact 子表记录（每个 flavor 一条）。Flavors/ApkURLs 用 JSON 字符串存储以兼容 sqlite/mysql。
type BuildRecord struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"column:name;type:varchar(128)" json:"name"` // 任务名，空则默认 <brand>-<versionName>-<YYYYMMDD-HHmm>
	BrandCode   string     `gorm:"column:brand_code;type:varchar(16);not null" json:"brandCode"`
	Flavors     string     `gorm:"column:flavors;type:text;not null" json:"flavors"` // JSON 数组字符串
	TestEvents  bool       `gorm:"column:test_events;not null;default:false" json:"testEvents"`
	Status      string     `gorm:"column:status;type:varchar(16);not null" json:"status"` // queued/running/success/failed
	Operator    string     `gorm:"column:operator;type:varchar(64)" json:"operator"`
	VersionName string     `gorm:"column:version_name;type:varchar(32)" json:"versionName"`
	ApkURLs     string     `gorm:"column:apk_urls;type:text" json:"apkUrls"` // JSON 数组字符串（汇总，兼容旧字段）
	Log         string     `gorm:"column:log;type:longtext" json:"log,omitempty"`  // 完整构建日志（runner 增量上报追加）
	LogExcerpt  string     `gorm:"column:log_excerpt;type:text" json:"logExcerpt"` // 日志摘要（末段，列表展示用）
	StartedAt   time.Time  `gorm:"column:started_at;autoCreateTime" json:"startedAt"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finishedAt"`

	Artifacts []BuildArtifact `gorm:"foreignKey:BuildRecordID;constraint:OnDelete:CASCADE" json:"artifacts,omitempty"`
}

func (BuildRecord) TableName() string { return "build_record" }

// BuildArtifact 单个 APK 产物（每个 flavor 一条）。apk_url 指向 nginx 静态目录（ADR-0008，APK 不走对象存储）。
type BuildArtifact struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BuildRecordID uint64    `gorm:"column:build_record_id;not null;index" json:"buildRecordId"`
	Flavor        string    `gorm:"column:flavor;type:varchar(64);not null;index:idx_flavor_time" json:"flavor"`
	VersionName   string    `gorm:"column:version_name;type:varchar(32)" json:"versionName"`
	ApkURL        string    `gorm:"column:apk_url;type:varchar(512);not null" json:"apkUrl"`
	Size          int64     `gorm:"column:size;not null;default:0" json:"size"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;index:idx_flavor_time" json:"createdAt"`
}

func (BuildArtifact) TableName() string { return "build_artifact" }

// AdminUser 后台账号。V1 只有一个永久 admin（bootstrap 创建，见 seed.EnsureBootstrapAdmin），
// User Management 只管理 role=user 的普通账号（见 docs/admin/10-user-management.md）。
//
// 删除用户是软删除（DeletedAt 非空 = 已删除）而非物理删除：channel.created_by /
// listing_app.created_by / audit_log.user_id 均引用 admin_user.id 且无级联，物理删除会
// 让这些外键字段悬空、破坏审计与归属追溯。GORM 对带 DeletedAt 字段的模型：
//   - 常规查询（First/Find，包括 GetUserByUsername 登录查询、GetUserByID 鉴权查询、
//     ListUsers 列表查询）自动加 `deleted_at IS NULL` 过滤，已删除账号天然「查不到」——
//     登录自然失败、已签发的 token 在下一次 RequireActiveAccount 检查时自然被判无效，
//     都不需要额外代码；
//   - .Delete() 自动变成 UPDATE ... SET deleted_at=now()，不是真正的 DELETE。
//   - 只有显式 .Unscoped() 才能看到/操作已删除的行（repo.CreateOrReactivateUser 用它
//     查找「同用户名的已删除行」并原地复活，而不是被 username 的全局唯一索引挡住）。
//
// 复活同用户名账号后，该行的 id 与创建时间保持不变——历史的 created_by 引用仍然指向
// 同一个 id，这是刻意的简化：新账号被视为「延续」了这个用户名此前的身份，而不是分配
// 一个全新的 id。见 docs/admin/10-user-management.md「用户名复用」一节。
type AdminUser struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"column:username;type:varchar(64);not null;uniqueIndex" json:"username"`
	PasswordHash string         `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Role         string         `gorm:"column:role;type:varchar(16);not null;default:user" json:"role"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AdminUser) TableName() string { return "admin_user" }

// AuditLog 审计日志。Detail 用 JSON 字符串。
type AuditLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"column:user_id" json:"userId"`
	Action    string    `gorm:"column:action;type:varchar(64);not null" json:"action"`
	Target    string    `gorm:"column:target;type:varchar(128)" json:"target"`
	Detail    string    `gorm:"column:detail;type:text" json:"detail"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_log" }

// AllModels 返回全部模型指针，供 AutoMigrate 使用。
func AllModels() []any {
	return []any{
		&Brand{},
		&BrandDomain{},
		&Store{},
		&Channel{},
		&ChannelDomain{},
		&DomainHealth{},
		&BuildRecord{},
		&BuildArtifact{},
		&AdminUser{},
		&AuditLog{},
		// 推送功能（ADR-0012）
		&PushDeviceToken{},
		&PushCampaign{},
		&PushCampaignTarget{},
		&PushRecord{},
		// 上架包 + AB 面网关
		&ListingApp{},
		&ListingDomain{},
		&ListingGate{},
		&ListingGateLog{},
	}
}
