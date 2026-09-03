// Package manifest 定义 CLI 与后端共享的数据结构。
//
// 这些结构体是 `go.work` 工作区内 server 与 cli 的「共用契约」（见 ADR-0001 / ADR-0007）：
// 后端 `GET /api/build/manifest` 的响应、APK 端 `GET /api/app/config` 的响应、
// 以及编译期烧录的 assets/bootstrap.json，都以此为准，避免两端定义漂移。
//
// 在 server 落地之前，CLI 先在此独立维护一份；server 建成后可将本包提升为共享包。
package manifest

// Manifest 是 `GET /api/build/manifest?brand=ap` 的响应主体（data 字段）。
//
// 一次拉全某品牌的全部启用渠道 + 域名 + PAL_CODE + 资源 zip 地址，
// CLI 据此重写 channels/<brand>.csv、下载解压 res、写每个 flavor 的 bootstrap.json。
type Manifest struct {
	// Brand 大渠道（品牌）代码：ap / bp / gp。
	Brand string `json:"brand"`
	// BrandName 品牌展示名：ArenaPlus / BingoPlus / GameZone。
	BrandName string `json:"brandName"`
	// ConfigURL 运行时配置端点（抗封基础设施上的静态快照），写入 bootstrap.json。
	// 见 ADR-0002：APK 启动时 GET 此地址拿最新域名清单。
	ConfigURL string `json:"configUrl"`
	// BrandDomains 品牌级默认域名清单 [主, 备用1..3]，作为继承渠道的兜底来源。
	BrandDomains []string `json:"brandDomains"`
	// Channels 该品牌下全部启用的小渠道。
	Channels []Channel `json:"channels"`
	// ConfigVersion 后台配置版本号，便于漂移检测与日志。
	ConfigVersion int `json:"configVersion,omitempty"`
}

// Channel 是一个小渠道（= 一个 Gradle product flavor）的完整打包信息。
type Channel struct {
	// Flavor Gradle flavor 名，CSV 第 1 列，如 ap01018。
	Flavor string `json:"flavor"`
	// ApplicationId 包名，CSV 第 2 列。ADR-0009：派生自 <品牌包前缀>.<flavor>，
	// 是唯一标识，也是 APK 拉取域名配置的解析键（GET /api/app/config?appId=）。
	ApplicationId string `json:"applicationId"`
	// PalCode 渠道营销码，CSV 第 3 列。ADR-0009：不再全局唯一（跨品牌可重复），
	// 仅编译期烧录进 BuildConfig.PAL_CODE 并作 URL 参数 /?palcode=，不再作身份/解析键。
	PalCode string `json:"palCode"`
	// AppName 桌面应用名，CSV 第 4 列。
	AppName string `json:"appName"`
	// UseBrandDomains 是否继承品牌默认域名（见 ADR-0006）。
	UseBrandDomains bool `json:"useBrandDomains"`
	// Domains 小渠道级域名覆盖（仅 UseBrandDomains=false 时有意义）。
	Domains []string `json:"domains,omitempty"`
	// ResZipURL 整套 res 资源 zip 的下载地址（对象存储），CLI 下载解压到 flavor 目录。
	ResZipURL string `json:"resZipUrl,omitempty"`
	// ResZipSHA256 res.zip 的 sha256 校验和（十六进制），用于校验下载完整性。
	ResZipSHA256 string `json:"resZipSha256,omitempty"`
	// AdjustAppToken Adjust 归因 App Token（ADR-0013 / docs/admin/08-adjust.md §2）。
	// 空串 = 未在 Console 给该渠道绑定 Adjust，CLI 渲染 adjust-tokens.json 时跳过此渠道，
	// 打包出的 flavor 运行时 AdjustBootstrap 全程 no-op（不集成、不发事件）。
	AdjustAppToken string `json:"adjustAppToken,omitempty"`
	// AdjustEvents Adjust 事件 name → token 的映射，由后台解析「上传的事件 CSV
	// （token,name,unique）」得到（unique 列已在后台丢弃）。CLI 不再解析，原样透传进
	// adjust-tokens.json 的 events 字段。AdjustAppToken 为空时该字段无意义（可为空）。
	AdjustEvents map[string]string `json:"adjustEvents,omitempty"`
	// SigningKey 该渠道应使用的签名 key ID（ADR-0016）。空串 = 默认 key（Gradle
	// signingConfigs.release 直接出包，不重签）；非空则是构建机本地「签名 key 注册表」
	// （internal/signing）里的一个 ID，如 "emptyapp"——用于一批已上架商店、当年用另一把
	// key 签名、商店按包名绑定证书不可更换的老渠道。Gradle 逻辑不动（护栏 #1）：runner 在
	// assemble 成功后、投递产物前，用 apksigner 按此 ID 对该渠道的 APK 重签（v1+v2）。
	// 密钥材料本身（keystore/口令）不随 manifest 下发，只在构建机注册表里，绝不上传/回传。
	SigningKey string `json:"signingKey,omitempty"`
}

// AdjustTokenEntry 是 app/adjust-tokens.json 中单个 applicationId 对应的 Adjust 配置
// （ADR-0013 §3 / docs/admin/08-adjust.md §3）。
//
// 键用 applicationId（ADR-0009 唯一标识）；app/build.gradle 的自包含旁路块按
// `variant.applicationId` 查表取 appToken/events 注入 BuildConfig，字段名
// appToken/events 需与该 Groovy 消费逻辑保持一致，不可改名。
type AdjustTokenEntry struct {
	AppToken string            `json:"appToken"`
	Events   map[string]string `json:"events"`
}

// EffectiveDomains 返回该渠道实际生效的域名清单：
// 继承品牌默认（UseBrandDomains=true）则用 brandDomains，否则用自身覆盖。
func (c Channel) EffectiveDomains(brandDomains []string) []string {
	if c.UseBrandDomains || len(c.Domains) == 0 {
		return brandDomains
	}
	return c.Domains
}

// Bootstrap 是写入每个 flavor 的 assets/bootstrap.json 的结构。
//
// 严格对应 docs/admin/03 的约定与 ADR-0002 的「编译期兜底（仅首启）」、ADR-0009 的「解析键改 appId」：
//   - AppID：APK 拉取配置的解析键（GET /api/app/config?appId=<AppID>），= BuildConfig.APPLICATION_ID。
//   - Palcode：仅用于拼加载 URL 的 /?palcode= 参数（ADR-0009 用途不变，但不再作解析键）。
//   - ConfigURL：运行时配置端点；DefaultDomains：从未成功拉取过时的兜底域名。
//
// 字段名 appId / configUrl / palcode / defaultDomains 与 Android 端读取保持一致。
// 注意：APP 实际可由 BuildConfig.APPLICATION_ID 直接得到 appId，这里冗余写入是为了让
// bootstrap.json 自包含、便于排查与离线核对（ADR-0009「bootstrap.json 让 APK 能按 appId 拉配置」）。
type Bootstrap struct {
	AppID          string   `json:"appId"`
	ConfigURL      string   `json:"configUrl"`
	Palcode        string   `json:"palcode"`
	DefaultDomains []string `json:"defaultDomains"`
}

// AppConfig 是 `GET /api/app/config?appId=` 的响应（APK 公开消费，ADR-0009 解析键改 appId）。
// 这里保留定义是为了 doctor / status 在需要时复用同一契约。
type AppConfig struct {
	AppID         string   `json:"appId"`
	Domains       []string `json:"domains"`
	ProbePath     string   `json:"probePath"`
	ConfigVersion int      `json:"configVersion"`
	TTLSeconds    int      `json:"ttlSeconds"`
}

// BuildRecord 是 `POST /api/build/records` 的请求体（CLI 回传构建结果）。
//
// 安全红线（CLAUDE.md / cli-go.md）：keystore、签名口令等敏感信息绝不出现在此结构、
// 也绝不进入任何上传路径。本结构只含渠道、状态、产物地址、日志摘要等非敏感信息。
type BuildRecord struct {
	BrandCode   string   `json:"brandCode"`
	Flavors     []string `json:"flavors"`
	TestEvents  bool     `json:"testEvents"`
	Status      string   `json:"status"` // running / success / failed
	Operator    string   `json:"operator,omitempty"`
	VersionName string   `json:"versionName,omitempty"`
	APKNames    []string `json:"apkNames,omitempty"` // 产物文件名（非本地绝对路径，不泄露机器信息）
	LogExcerpt  string   `json:"logExcerpt,omitempty"`
}

// 构建状态常量。
const (
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// ---- 服务器端构建队列（ADR-0008）：runner 与后端的共享契约 ----

// BuildJob 是 `POST /api/build/jobs/claim` 领取到的一个待构建任务（data 字段）。
//
// 由 Web「打包中心」触发入队，构建机（hybrid-pack runner）消费：pull→assemble→签名→
// 落产物→回传。队列为空时后端返回 ID==0 的空对象（或空 data），runner 据此判定无任务。
type BuildJob struct {
	// ID 任务唯一 ID；0 表示「当前无可领取任务」。
	ID int64 `json:"id"`
	// Brand 大渠道（品牌）code：ap / bp / gp。
	Brand string `json:"brand"`
	// Flavors 本任务要打包的小渠道（flavor）列表。
	Flavors []string `json:"flavors"`
	// TestEvents 是否开启测试事件（透传 -PtestEvents）。
	TestEvents bool `json:"testEvents"`
	// VersionName 版本号 X.Y.Z（透传 -PversionName；runner 用 job.versionName）。
	VersionName string `json:"versionName"`
	// TaskName 任务展示名（可空；ADR-0008 默认 <品牌code>-<versionName>-<时间>）。
	TaskName string `json:"taskName,omitempty"`
}

// JobStatusUpdate 是 `POST /api/build/jobs/{id}/status` 的请求体。
type JobStatusUpdate struct {
	Status     string `json:"status"` // running / success / failed
	LogExcerpt string `json:"logExcerpt,omitempty"`
}

// JobLogChunk 是 `POST /api/build/jobs/{id}/logs` 的请求体（流式日志，best-effort）。
type JobLogChunk struct {
	Chunk string `json:"chunk"`
}

// BuildArtifact 是 `POST /api/build/jobs/{id}/artifacts` 的请求体（登记单个 APK）。
//
// 安全：只含非机密的产物元信息——绝不含 keystore / 口令 / 本地绝对路径。
// ApkURL 是 nginx 静态下载地址（ADR-0008：/apks/<brand>/<flavor>/<versionName>/...）。
type BuildArtifact struct {
	Flavor      string `json:"flavor"`
	VersionName string `json:"versionName"`
	FileName    string `json:"fileName"`
	ApkURL      string `json:"apkUrl"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256,omitempty"`
}
