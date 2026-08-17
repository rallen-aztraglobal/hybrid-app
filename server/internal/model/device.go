// Package model — 渠道设备上报表。
package model

import "time"

// ChannelDevice 渠道设备上报记录：APK 启动时上报一次安装设备信息（GAID/ADID/OAID 等），
// 用 device_key（客户端安装 UUID）做幂等去重键，供投放/归因侧导出使用。
//
// app_name/brand_code 是注册时刻 channel 表的快照（免 JOIN 直接导出 CSV），后续渠道改名
// 不会回溯更新已落库的历史行——这是设计取舍，不是 bug。
type ChannelDevice struct {
	ID uint64 `gorm:"primaryKey;autoIncrement;index:idx_app_created,priority:3;index:idx_created,priority:2" json:"id"`

	// DeviceKey 客户端安装 UUID，唯一去重键。
	DeviceKey string `gorm:"column:device_key;type:varchar(64);not null;uniqueIndex" json:"deviceKey"`

	ApplicationID string `gorm:"column:application_id;type:varchar(128);not null;index:idx_app_created,priority:1" json:"applicationId"`
	BrandCode     string `gorm:"column:brand_code;type:varchar(16);not null;default:''" json:"brandCode"`
	PalCode       string `gorm:"column:pal_code;type:varchar(64);not null;default:''" json:"palCode"`
	// AppName 注册时渠道名快照，导出免 JOIN。
	AppName    string `gorm:"column:app_name;type:varchar(128);not null;default:''" json:"appName"`
	DeviceName string `gorm:"column:device_name;type:varchar(128);not null;default:''" json:"deviceName"`

	// GAID/ADID/OAID 统一小写存储；opt-out（全 0 GAID）或未采集时存空串。
	GAID string `gorm:"column:gaid;type:varchar(64);not null;default:''" json:"gaid"`
	ADID string `gorm:"column:adid;type:varchar(64);not null;default:''" json:"adid"`
	OAID string `gorm:"column:oaid;type:varchar(64);not null;default:''" json:"oaid"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_app_created,priority:2;index:idx_created,priority:1" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (ChannelDevice) TableName() string { return "channel_device" }
