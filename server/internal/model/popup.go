// Package model — 马甲包弹窗模块（ADR-0019 / docs/admin/12-popup.md §2）。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 弹窗位置码（PRD 编号）。
const (
	PopupP1 = "P1"
	PopupP2 = "P2"
	PopupP3 = "P3"
	PopupP4 = "P4"
	PopupP5 = "P5"
	PopupP6 = "P6"
	PopupP7 = "P7"
	PopupP8 = "P8"
)

// 弹窗跳转方式。
const (
	PopupOpenWebview = "webview"
	PopupOpenBrowser = "browser"
	PopupOpenStore   = "store"
)

// 用户类型定向。
const (
	PopupUserAll = "all"
	PopupUserNew = "new"
	PopupUserOld = "old"
)

// 注：brand_codes / app_ids / countries 复用 listing.go 的 StringList（text 列存 JSON 数组）。

// Popup 弹窗（软删：统计仍能显示已删弹窗名称）。
// 注意：不对布尔列写 default:true——GORM 会把 Go 零值 false 静默改写成 true（见 rbac 文档的教训），
// 所有布尔列由 service 层显式赋值。
type Popup struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"column:name;type:varchar(64);not null;default:''" json:"name"`
	Position string `gorm:"column:position;type:varchar(8);not null;default:'';index" json:"position"`
	Enabled  bool   `gorm:"column:enabled;not null" json:"enabled"`
	Priority int    `gorm:"column:priority;not null;default:0" json:"priority"`

	StartAt *time.Time `gorm:"column:start_at" json:"startAt,omitempty"`
	EndAt   *time.Time `gorm:"column:end_at" json:"endAt,omitempty"`

	BrandCodes StringList `gorm:"column:brand_codes;type:text" json:"brandCodes"`
	AppIDs     StringList `gorm:"column:app_ids;type:text" json:"appIds"`
	// 版本区间：≥ min、< max，0 = 不限；存 major*10000+minor*100+patch。
	MinVersionCode int        `gorm:"column:min_version_code;not null;default:0" json:"minVersionCode"`
	MaxVersionCode int        `gorm:"column:max_version_code;not null;default:0" json:"maxVersionCode"`
	UserType       string     `gorm:"column:user_type;type:varchar(8);not null;default:all" json:"userType"`
	Countries      StringList `gorm:"column:countries;type:text" json:"countries"`

	TabEnabled   bool   `gorm:"column:tab_enabled;not null" json:"tabEnabled"`
	TabIconURL   string `gorm:"column:tab_icon_url;type:varchar(512);not null;default:''" json:"tabIconUrl"`
	TabText      string `gorm:"column:tab_text;type:varchar(32);not null;default:''" json:"tabText"`
	Countdown    bool   `gorm:"column:countdown;not null" json:"countdown"`
	MaskClosable bool   `gorm:"column:mask_closable;not null" json:"maskClosable"`
	Closable     bool   `gorm:"column:closable;not null" json:"closable"`
	OpenMode     string `gorm:"column:open_mode;type:varchar(16);not null;default:webview" json:"openMode"`

	AutoplaySeconds  int  `gorm:"column:autoplay_seconds;not null;default:5" json:"autoplaySeconds"`
	ResumeGapMinutes int  `gorm:"column:resume_gap_minutes;not null;default:30" json:"resumeGapMinutes"`
	Badge            bool `gorm:"column:badge;not null" json:"badge"`

	CreatedBy string         `gorm:"column:created_by;type:varchar(64);not null;default:''" json:"createdBy"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`

	Cards []PopupCard `gorm:"foreignKey:PopupID" json:"-"`
}

func (Popup) TableName() string { return "popup" }

// PopupCard 素材卡片；卡片 ID 即埋点维度，编辑时原地更新保持 ID 不变。
type PopupCard struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	PopupID     uint64 `gorm:"column:popup_id;not null;index" json:"popupId"`
	Sort        int    `gorm:"column:sort;not null;default:0" json:"sort"`
	ImageURL    string `gorm:"column:image_url;type:varchar(512);not null;default:''" json:"imageUrl"`
	LinkURL     string `gorm:"column:link_url;type:varchar(512);not null;default:''" json:"linkUrl"`
	ButtonText  string `gorm:"column:button_text;type:varchar(32);not null;default:''" json:"buttonText"`
	Title       string `gorm:"column:title;type:varchar(64);not null;default:''" json:"title"`
	Description string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"`
}

func (PopupCard) TableName() string { return "popup_card" }

// PopupPosition 位置开关（全局，启动 seed 8 行，已存在的不覆盖）。
type PopupPosition struct {
	Code      string    `gorm:"column:code;type:varchar(8);primaryKey" json:"code"`
	Enabled   bool      `gorm:"column:enabled;not null" json:"enabled"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	UpdatedBy string    `gorm:"column:updated_by;type:varchar(64);not null;default:''" json:"updatedBy"`
}

func (PopupPosition) TableName() string { return "popup_position" }

// PopupStatDaily 埋点日聚合（不落原始事件）。
//
// 唯一键 (stat_date, popup_id, card_id, card_index, application_id, app_version, event, dim)，
// MySQL utf8mb4 索引字节估算：10*4 + 8 + 8 + 4 + 128*4 + 32*4 + 32*4 + 16*4 = 904 < 3072。
type PopupStatDaily struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	StatDate      string `gorm:"column:stat_date;type:varchar(10);not null;uniqueIndex:uk_popup_stat,priority:1" json:"statDate"`
	PopupID       uint64 `gorm:"column:popup_id;not null;uniqueIndex:uk_popup_stat,priority:2" json:"popupId"`
	CardID        uint64 `gorm:"column:card_id;not null;default:0;uniqueIndex:uk_popup_stat,priority:3" json:"cardId"`
	CardIndex     int    `gorm:"column:card_index;not null;default:0;uniqueIndex:uk_popup_stat,priority:4" json:"cardIndex"`
	ApplicationID string `gorm:"column:application_id;type:varchar(128);not null;uniqueIndex:uk_popup_stat,priority:5" json:"applicationId"`
	AppVersion    string `gorm:"column:app_version;type:varchar(32);not null;default:'';uniqueIndex:uk_popup_stat,priority:6" json:"appVersion"`
	Event         string `gorm:"column:event;type:varchar(32);not null;uniqueIndex:uk_popup_stat,priority:7" json:"event"`
	Dim           string `gorm:"column:dim;type:varchar(16);not null;default:'';uniqueIndex:uk_popup_stat,priority:8" json:"dim"`
	Count         int64  `gorm:"column:count;not null;default:0" json:"count"`
}

func (PopupStatDaily) TableName() string { return "popup_stat_daily" }

// PopupEventBatch 埋点上报幂等表：同 batch_id 重试直接成功、不重复累加；cron 清 7 天前。
type PopupEventBatch struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BatchID   string    `gorm:"column:batch_id;type:varchar(64);not null;uniqueIndex" json:"batchId"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index" json:"createdAt"`
}

func (PopupEventBatch) TableName() string { return "popup_event_batch" }
