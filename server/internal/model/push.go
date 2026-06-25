// Package model — 推送功能四张表（ADR-0012）。
package model

import "time"

// 推送活动状态机枚举。
const (
	CampaignDraft     = "draft"
	CampaignScheduled = "scheduled"
	CampaignSending   = "sending"
	CampaignDone      = "done"
	CampaignFailed    = "failed"
)

// PushDeviceToken 设备 token 注册库。
// APK 上报 (applicationId, device_token)，发送时按 applicationId 取活跃 token。
// ADR-0009：以 applicationId 为键，pal_code 仅作随 URL 传递字段。
type PushDeviceToken struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ApplicationID string    `gorm:"column:application_id;type:varchar(128);not null;index:idx_token_app_active,priority:1" json:"applicationId"`
	BrandCode     string    `gorm:"column:brand_code;type:varchar(16);not null" json:"brandCode"`
	DeviceToken   string    `gorm:"column:device_token;type:varchar(255);not null;uniqueIndex" json:"deviceToken"`
	PalCode       string    `gorm:"column:pal_code;type:varchar(64);not null;default:''" json:"palCode"`
	Platform      string    `gorm:"column:platform;type:varchar(16);not null;default:android" json:"platform"`
	ModelInfo     string    `gorm:"column:model_info;type:varchar(255)" json:"modelInfo"`
	IsActive      bool      `gorm:"column:is_active;not null;default:true;index:idx_token_app_active,priority:2" json:"isActive"`
	LastSeenAt    time.Time `gorm:"column:last_seen_at;not null" json:"lastSeenAt"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (PushDeviceToken) TableName() string { return "push_device_token" }

// PushCampaign 推送活动（内容 + 目标 + 状态机 draft|scheduled|sending|done|failed）。
// ExtraData 以 JSON 字符串存储（兼容 sqlite 与 mysql）。
// deeplink_path 存相对路径（如 /promo/618），绝不存域名（ADR-0002）。
type PushCampaign struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string     `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Title         string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Body          string     `gorm:"column:body;type:text;not null" json:"body"`
	ImageURL      string     `gorm:"column:image_url;type:varchar(512)" json:"imageUrl"`
	DeeplinkPath  string     `gorm:"column:deeplink_path;type:varchar(512)" json:"deeplinkPath"`
	ExtraData     string     `gorm:"column:extra_data;type:text" json:"extraData"` // JSON string map[string]string
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:draft;index:idx_campaign_status_scheduled,priority:1" json:"status"`
	ScheduledAt   *time.Time `gorm:"column:scheduled_at;index:idx_campaign_status_scheduled,priority:2" json:"scheduledAt,omitempty"`
	SentAt        *time.Time `gorm:"column:sent_at" json:"sentAt,omitempty"`
	TotalDevices  int        `gorm:"column:total_devices;not null;default:0" json:"totalDevices"`
	SuccessCount  int        `gorm:"column:success_count;not null;default:0" json:"successCount"`
	FailureCount  int        `gorm:"column:failure_count;not null;default:0" json:"failureCount"`
	CreatedBy     string     `gorm:"column:created_by;type:varchar(64)" json:"createdBy"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`

	Targets []PushCampaignTarget `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"-"`
	Records []PushRecord         `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"-"`
}

func (PushCampaign) TableName() string { return "push_campaign" }

// PushCampaignTarget 活动目标渠道（多对多：一个活动覆盖多个 applicationId）。
type PushCampaignTarget struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	CampaignID    uint64 `gorm:"column:campaign_id;not null;index" json:"campaignId"`
	ApplicationID string `gorm:"column:application_id;type:varchar(128);not null" json:"applicationId"`
}

func (PushCampaignTarget) TableName() string { return "push_campaign_target" }

// PushRecord 发送结果（按 application_id 汇总，不存逐 token 明细以控量）。
type PushRecord struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CampaignID    uint64     `gorm:"column:campaign_id;not null;index" json:"campaignId"`
	ApplicationID string     `gorm:"column:application_id;type:varchar(128);not null" json:"applicationId"`
	Sent          int        `gorm:"column:sent;not null;default:0" json:"sent"`
	Failed        int        `gorm:"column:failed;not null;default:0" json:"failed"`
	ErrorSample   string     `gorm:"column:error_sample;type:text" json:"errorSample,omitempty"`
	FinishedAt    *time.Time `gorm:"column:finished_at" json:"finishedAt,omitempty"`
}

func (PushRecord) TableName() string { return "push_record" }
