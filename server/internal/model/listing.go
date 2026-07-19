// Package model — 上架包（listing）模块的持久化实体。
//
// 「上架包」指正式上架 Google Play / App Store 的合规应用（当前两个：ColorStack / DeckTallyPro），
// 与 channel（小渠道 APK，走 CSV + flavor 打包）是两条独立产线，故独立建表、不复用 channel。
//
// 命名说明：本模块一律用 listing 前缀。仓库里已有的 store 表指的是「应用商店后缀」
// （华为/小米/Oppo，用于拼 applicationId 分段），与本模块无关，勿混淆。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// 上架包平台枚举。
const (
	ListingAndroid = "android"
	ListingIOS     = "ios"
)

// 上架包状态枚举（与渠道保持同一套语义）。
const (
	ListingEnabled  = "enabled"
	ListingDisabled = "disabled"
	ListingArchived = "archived"
)

// 上架包技术栈枚举（决定 CLI/构建与客户端 SDK 接入方式）。
const (
	TechFlutter       = "flutter"        // ColorStack：一套 Dart 覆盖 android+ios
	TechNativeIOS     = "native_ios"     // DeckTallyPro：原生 Swift，仅 iOS
	TechNativeAndroid = "native_android" // 预留
)

// ForcedACountries 是无条件强制走 A 面的国家码，无视 listing_gate 的任何配置。
//
// 这是硬编码的最后一道闸，不做成可配置项：中美两地是应用商店审核与合规风险最集中的来源，
// 一次误配的代价远高于灵活性收益。前端不提供这两个选项只是 UI 约定，绕过前端直接调 API
// 就破了，因此判定必须在服务端做（见 service 层 EvaluateGate）。
var ForcedACountries = map[string]struct{}{
	"CN": {},
	"US": {},
}

// IsForcedACountry 判断某国家码是否被硬编码强制走 A 面。入参需为大写 ISO-3166-1 alpha-2。
func IsForcedACountry(country string) bool {
	_, ok := ForcedACountries[country]
	return ok
}

// 网关判定结果枚举（客户端据此决定展示 A 面还是 B 面）。
const (
	GateModeA = "A" // 展示应用自身的合规内容
	GateModeB = "B" // 放行到配置的 web 地址
)

// StringList 是「JSON 字符串数组」列在 Go 侧的类型，用于网关的国家 / 时区 / CIDR 清单。
//
// 与 AdjustEvents 同源的设计：底层就是 []string，encoding/json 对具名切片类型的默认行为
// 已经是「按 JSON 数组序列化/反序列化」，故只需补 database/sql 的 Scanner/Valuer。
// 空切片存 NULL；「空」在各字段上的业务语义不统一（Countries 空 = 配置无效，其余空 = 不参与判定），
// 一律以 ListingGate 的字段文档为准，本类型只负责存取。
type StringList []string

// Value 实现 driver.Valuer：空切片存 NULL，非空序列化为 JSON 数组字符串。
func (l StringList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return nil, nil
	}
	b, err := json.Marshal([]string(l))
	if err != nil {
		return nil, fmt.Errorf("序列化 StringList 失败: %w", err)
	}
	return string(b), nil
}

// Scan 实现 sql.Scanner：NULL/空 → nil 切片；否则按 JSON 数组解析。
func (l *StringList) Scan(value any) error {
	if value == nil {
		*l = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("StringList 列不支持的底层类型: %T", value)
	}
	if len(raw) == 0 {
		*l = nil
		return nil
	}
	var s []string
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("解析 StringList 失败: %w", err)
	}
	*l = StringList(s)
	return nil
}

// ListingApp 一个上架包。
//
// 唯一性：(platform, bundle_id) 唯一，而非 bundle_id 全局唯一 —— Flutter 包在 Android 与 iOS 上
// 用的是同一个包名（如 ColorStack 两端都是 com.vividnest.colorstack5821），全局唯一会误杀。
//
// 域名：BrandID + UseBrandDomains 复刻 Channel 的继承语义（ADR-0006）——
// 默认继承所属品牌的域名清单，需要时用 ListingDomain 覆盖。B 面最终落到的 web 与小渠道包同一套。
type ListingApp struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BrandID  uint64 `gorm:"column:brand_id;not null;index" json:"brandId"`
	Platform string `gorm:"column:platform;type:varchar(16);not null;uniqueIndex:uk_platform_bundle" json:"platform"`
	BundleID string `gorm:"column:bundle_id;type:varchar(128);not null;uniqueIndex:uk_platform_bundle" json:"bundleId"`

	Name        string `gorm:"column:name;type:varchar(64);not null" json:"name"`        // 内部代号，如 ColorStack
	DisplayName string `gorm:"column:display_name;type:varchar(128)" json:"displayName"` // 商店展示名
	Tech        string `gorm:"column:tech;type:varchar(24);not null" json:"tech"`        // flutter / native_ios / native_android
	StoreURL    string `gorm:"column:store_url;type:varchar(512)" json:"storeUrl"`       // 商店详情页链接
	Status      string `gorm:"column:status;type:varchar(16);not null;default:enabled" json:"status"`

	// UseBrandDomains=true 时 B 面域名继承品牌，false 时用 ListingDomain 覆盖。
	UseBrandDomains bool `gorm:"column:use_brand_domains;not null;default:true" json:"useBrandDomains"`

	// GateEnabled 是 AB 面总开关。false = 该包永远只有 A 面（网关直接返回 mode=A，不做任何判定）。
	// 这是「默认安全」的第一道闸：新建的上架包默认关闭，运营确认规则无误后再手动打开。
	GateEnabled bool `gorm:"column:gate_enabled;not null;default:false" json:"gateEnabled"`

	// AppsFlyer 归因。DevKey 按账号维度、AppID 按应用维度（iOS 形如 id123456789，Android 为包名）。
	// 均可空 = 该包未接 AF。A 面也初始化 SDK（避免「集成了却不用」在审核侧显得可疑）。
	AfDevKey string `gorm:"column:af_dev_key;type:varchar(64)" json:"afDevKey"`
	AfAppID  string `gorm:"column:af_app_id;type:varchar(64)" json:"afAppId"`

	// Adjust 归因。复用 channel 既有的类型与跨层契约（ADR-0013 / docs/admin/08-adjust.md）：
	// server 只做形状校验、不解析 CSV，原样读出给客户端配置。
	AdjustAppToken *string      `gorm:"column:adjust_app_token;type:varchar(64)" json:"adjustAppToken,omitempty"`
	AdjustEvents   AdjustEvents `gorm:"column:adjust_events;type:text" json:"adjustEvents,omitempty"`

	Remark    string    `gorm:"column:remark;type:varchar(255)" json:"remark"`
	CreatedBy uint64    `gorm:"column:created_by" json:"createdBy"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`

	Brand   *Brand          `gorm:"foreignKey:BrandID" json:"brand,omitempty"`
	Domains []ListingDomain `gorm:"foreignKey:ListingID;constraint:OnDelete:CASCADE" json:"domains,omitempty"`
	Gate    *ListingGate    `gorm:"foreignKey:ListingID;constraint:OnDelete:CASCADE" json:"gate,omitempty"`
}

func (ListingApp) TableName() string { return "listing_app" }

// ListingDomain 上架包级 B 面域名覆盖（use_brand_domains=false 时生效）。
// position 0=主，1..n=备用，语义与 ChannelDomain 完全一致。
type ListingDomain struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ListingID uint64 `gorm:"column:listing_id;not null;uniqueIndex:uk_listing_pos" json:"listingId"`
	Position  int    `gorm:"column:position;not null;uniqueIndex:uk_listing_pos" json:"position"`
	URL       string `gorm:"column:url;type:varchar(255);not null" json:"url"`
	Enabled   bool   `gorm:"column:enabled;not null;default:true" json:"enabled"`
}

func (ListingDomain) TableName() string { return "listing_domain" }

// ListingGate 一个上架包的 AB 面放行规则（与 ListingApp 一对一）。
//
// 判定一律在服务端做，客户端不内置任何 B 面地址 —— 审核方拿到安装包也翻不出域名。
//
// 判定顺序（任一步判为 A 即短路返回，不再往下走）：
//
//  1. ListingApp.GateEnabled=false        → A（总开关关闭时，本结构体的所有字段都无意义）
//  2. 国家 ∈ ForcedACountries（CN/US）     → A（硬编码，无视本结构体配置）
//  3. 命中 IPDenyCIDRs                    → A
//  4. GeoIP 解析不出国家                   → A（自然 fail-closed：未知国家不可能在必填白名单里）
//  5. 国家 ∉ Countries                    → A
//  6. Timezones 非空 且 时区 ∉ Timezones    → A
//  7. IPAllowCIDRs 非空 且 IP ∉ 其中        → A
//  8. 以上全部通过                         → B
//
// 条件之间一律是 AND，没有「任一命中即放行」的模式 —— 放宽的口子只会在误配时放进审核员。
//
// 必填/选填（由 Service 层校验强制，不靠前端）：
//   - Countries 必填非空。空清单不代表「不限国家」，而是配置无效，Service 拒绝保存；
//     真要全关就把 GateEnabled 置 false，语义唯一、不产生歧义。
//   - Timezones / IPAllowCIDRs / IPDenyCIDRs 选填，为空 = 该维度不参与判定。
type ListingGate struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ListingID uint64 `gorm:"column:listing_id;not null;uniqueIndex" json:"listingId"`

	// Countries 必填：ISO-3166-1 alpha-2 大写国家码白名单，如 ["PH","ID"]。
	// 由服务端按请求 IP 查 GeoIP 得出，客户端无从干预。
	// 即使运营把 CN/US 塞进来也无效——第 2 步的硬编码闸在它之前。
	Countries StringList `gorm:"column:countries;type:text" json:"countries"`

	// Timezones 选填：IANA 时区名白名单，如 ["Asia/Manila"]。
	// 由客户端上报，可伪造，故只能作为收紧条件叠加在国家判定之上，永远不单独作准。
	Timezones StringList `gorm:"column:timezones;type:text" json:"timezones"`

	// IPAllowCIDRs 选填：非空时，请求 IP 还必须落在其中之一（在国家白名单之上再收紧一层）。
	IPAllowCIDRs StringList `gorm:"column:ip_allow_cidrs;type:text" json:"ipAllowCidrs"`
	// IPDenyCIDRs 选填：命中即强制 A 面，用于钉死已知的审核机房网段。
	IPDenyCIDRs StringList `gorm:"column:ip_deny_cidrs;type:text" json:"ipDenyCidrs"`

	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (ListingGate) TableName() string { return "listing_gate" }

// ListingGateLog 网关判定流水。上线后排查「为什么这台设备没进 B 面」全靠它，
// 故连拒绝原因一并落库。仅作运维观测，不参与判定逻辑。
type ListingGateLog struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	ListingID uint64 `gorm:"column:listing_id;not null;index:idx_listing_time" json:"listingId"`

	IP       string `gorm:"column:ip;type:varchar(64)" json:"ip"`
	Country  string `gorm:"column:country;type:varchar(8)" json:"country"`    // GeoIP 解析结果，未知为空
	Timezone string `gorm:"column:timezone;type:varchar(64)" json:"timezone"` // 客户端上报值

	Decision string `gorm:"column:decision;type:varchar(4);not null" json:"decision"` // A / B
	Reason   string `gorm:"column:reason;type:varchar(128)" json:"reason"`            // 命中/拒绝的具体原因

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_listing_time" json:"createdAt"`
}

func (ListingGateLog) TableName() string { return "listing_gate_log" }
