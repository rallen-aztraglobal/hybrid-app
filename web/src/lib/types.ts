/**
 * 领域类型 —— 对齐**真实后端**（server/internal/...）的数据模型与 REST 契约。
 *
 * 本轮（第二轮）把第一轮的占位类型对齐到 Go 后端实际返回的字段：
 *  - 登录返回 { accessToken, refreshToken, expiresIn, role, username }（auth.go）
 *  - 品牌/品牌域名以 string[] 下发（brand.go / domain.go），前端用适配器转 DomainEntry[]
 *  - 渠道 id 为数字主键（channels.go 路径参数），前端统一以字符串持有
 *  - 构建：ADR-0008 的「打包任务」(build job) + 日志流 + 单 APK 下载 + 渠道最新包
 *  - 身份：ADR-0009 的 applicationId 由 品牌包前缀 + flavor 派生
 *  - Adjust 归因（08-adjust.md / ADR-0013）：Channel/ChannelInput 新增
 *    `adjustAppToken` / `adjustEvents` 两个可空字段，随渠道保存接口一起提交。
 *    **待办**：server 的 CreateChannelInput/UpdateChannelInput OpenAPI 契约尚未收录
 *    这两个字段（本文件手写先行带上，等后端补齐并重新生成 OpenAPI 后对齐/替换）。
 *
 * 后端 OpenAPI 就绪后（server 暴露 /swagger/doc.json），可用 `openapi-typescript`
 * 生成 `src/lib/api/schema.d.ts` 替换本文件；当前手写以保证编译期类型安全。
 */

export type BrandCode = 'ap' | 'bp' | 'gp';

export type ChannelStatus = 'enabled' | 'disabled' | 'archived';

/** 域名探测健康度。'unconfigured' 为前端态：该槽位尚未填写。 */
export type DomainHealth = 'ok' | 'warn' | 'down' | 'unknown' | 'unconfigured';

/**
 * 大渠道（品牌）。对应后端 service.BrandView。
 * 注意：后端 BrandView.domains 是 string[]（仅 URL，按 position 升序）；
 * 前端在 api 适配层转成 DomainEntry[] 供编辑器使用。
 */
export interface Brand {
  /** 后端数字主键（字符串化） */
  id?: string;
  code: BrandCode;
  name: string;
  /** deeplink scheme：gzone / bingo */
  scheme: string;
  /** 是否集成 HMS/OAID（仅 bp） */
  hmsEnabled: boolean;
  /** 后台 Tab 主题色（十六进制） */
  accentColor: string;
  /** 该品牌启用 + 停用的渠道总数（不含 archived） */
  channelCount: number;
  /** 品牌级默认域名（位置 0 = 主，1..3 = 备用）。由 string[] 适配而来。 */
  domains: DomainEntry[];
  /**
   * 品牌包前缀（ADR-0009）：ap→com.arenaplus / bp→com.bingoplus / gp→com.gamezone。
   * applicationId = packagePrefix + '.' + flavor。后端 BrandView 暂未下发时，
   * 前端用 BRAND_META 兜底（见 lib/brands.ts deriveApplicationId）。
   */
  packagePrefix?: string;
}

/** 单条域名（品牌级或渠道级覆盖通用）。 */
export interface DomainEntry {
  /** 0 = 主域名；1..3 = 备用 */
  position: number;
  url: string;
  enabled: boolean;
  /** 后台巡检结果（仅展示，不决定 APK 行为；见 ADR-0003） */
  health?: DomainHealth;
  latencyMs?: number;
}

/** 后端保存域名时的入参（domainutil.DomainInput）。 */
export interface DomainInput {
  position: number;
  url: string;
  enabled: boolean;
}

/** 保存域名后的探测结果（service.ProbeResult）。 */
export interface ProbeResult {
  url: string;
  ok: boolean;
  httpCode?: number;
  latencyMs?: number;
  note?: string;
}

/**
 * 应用商店（渠道包发布到的应用商店，如华为/应用宝/Google Play 等）。
 * 对应后端 model.Store。code 一经创建不可改（作为 flavor 后缀参与包名派生）。
 */
export interface Store {
  id: number;
  /** 商店代号，拼入 flavor 后缀（如 hw）；创建后不可改 */
  code: string;
  name: string;
  /** 展示排序，越小越靠前 */
  sort: number;
  status: 'enabled' | 'disabled';
  createdAt?: string;
  updatedAt?: string;
}

/** 新增商店入参（POST /api/stores）。 */
export interface StoreInput {
  code: string;
  name: string;
  sort?: number;
}

/** 编辑商店入参（PUT /api/stores/:id）；code 不可改故不在此列。 */
export interface StoreUpdateInput {
  name?: string;
  sort?: number;
  status?: Store['status'];
}

/** 小渠道（一个 Gradle product flavor）。对应后端 model.Channel。 */
export interface Channel {
  /** 后端数字主键（字符串化）；mock 模式下为 `<brand>-<flavor>` */
  id: string;
  brandCode: BrandCode;
  /** Gradle flavor 名，如 ap01018；带应用商店后缀时形如 `bpocmhuawei004_hw` */
  flavorName: string;
  /** 包名，全局唯一；ADR-0009 由 品牌包前缀 + flavor 派生（flavor 中的下划线会转为点号） */
  applicationId: string;
  /** URL 参数 /?palcode=，编译期烧录；ADR-0009：不再全局唯一、不再作解析键 */
  palCode: string;
  /** 桌面应用名 */
  appName: string;
  status: ChannelStatus;
  /** 是否继承品牌默认域名（见 ADR-0006） */
  useBrandDomains: boolean;
  /** use_brand_domains = false 时生效的渠道级覆盖 */
  domains?: DomainEntry[];
  iconMasterUrl?: string;
  splashUrl?: string;
  remark?: string;
  /** ADR-0008：该子渠道最近一次成功构建的 APK 下载地址（后端 latestApkUrl）。 */
  latestApkUrl?: string;
  updatedAt?: string;
  /** 关联应用商店主键（可空 = 无商店/默认）。 */
  storeId?: string | number | null;
  /** 关联应用商店（后端下发的精简视图，供列表展示商店标签）。 */
  store?: { id: string | number; code: string; name: string } | null;
  /**
   * Adjust App Token（08-adjust.md / ADR-0013）。空/未设置 = 该渠道未绑定 Adjust，
   * 打包时不集成、不发任何 Adjust 事件。非机密，随 APK 分发。
   */
  adjustAppToken?: string;
  /**
   * Adjust 事件映射 `{ 事件name: token }`，由前端解析 Adjust 导出的事件 CSV
   * （`token,name,unique`，`unique` 列丢弃）得到。server 只存不解析（见 CLAUDE.md 跨层契约）。
   */
  adjustEvents?: Record<string, string>;
}

/**
 * 新增/编辑渠道的表单载荷（前端聚合）。
 *
 * 注意后端把职责拆开了：基本信息走 POST/PUT /channels（CreateChannelInput/
 * UpdateChannelInput）；域名覆盖走 PUT /channels/:id/domains；图标/splash 走
 * multipart 子端点。api 层据此把本载荷拆成多次请求。
 *
 * ADR-0009：applicationId 不在表单手填，由 品牌包前缀 + flavorName 派生、只读。
 */
export interface ChannelInput {
  brandCode: BrandCode;
  /** 合成 flavor（基础 flavor + 可选 `_`+商店 code）；提交给后端的 flavorName。 */
  flavorName: string;
  /** 派生只读（packagePrefix + '.' + flavorName，下划线转点号）；提交时仍带上供后端校验一致性 */
  applicationId: string;
  palCode: string;
  appName: string;
  useBrandDomains: boolean;
  domains?: DomainEntry[];
  status?: ChannelStatus;
  /** 前端裁剪后的 1024² 主图（dataURL，后端 fan-out 全部密度） */
  iconMasterDataUrl?: string;
  splashDataUrl?: string;
  remark?: string;
  /** 应用商店主键（可空 = 无商店/默认）。 */
  storeId?: string | number | null;
  /** Adjust App Token；空字符串 = 不启用（见 Channel.adjustAppToken 注释）。 */
  adjustAppToken?: string;
  /** Adjust 事件映射 `{ name: token }`（见 Channel.adjustEvents 注释）。 */
  adjustEvents?: Record<string, string>;
}

/**
 * 构建任务（ADR-0008）。一次「打包中心」触发 = 一个 build job。
 * 对应后端（下一轮落地）的 POST /api/build/jobs 返回与 GET /api/build/jobs 列表项。
 *
 * 兼容旧的 build_record：字段语义一致，jobName 为新增（可填，留空用默认名）。
 */
export interface BuildJob {
  id: string;
  /** 任务名：留空用默认 `<品牌code>-<versionName>-<YYYYMMDD-HHmm>`（ADR-0008） */
  jobName: string;
  brandCode: BrandCode;
  flavors: string[];
  versionName: string;
  testEvents: boolean;
  status: BuildStatus;
  operator?: string;
  /** 每个 APK 一条产物（含下载地址）。 */
  artifacts?: BuildArtifact[];
  /** 日志摘要（列表页展示用；详情/实时日志走 logs 流）。 */
  logExcerpt?: string;
  startedAt: string;
  finishedAt?: string;
}

export type BuildStatus = 'queued' | 'running' | 'success' | 'failed';

/** 单个 APK 产物（ADR-0008 build_artifact）。 */
export interface BuildArtifact {
  flavor: string;
  versionName: string;
  /** nginx 静态下载地址 /apks/<brand>/<flavor>/<versionName>/app-<flavor>-release.apk */
  apkUrl: string;
  sizeBytes?: number;
  builtAt?: string;
}

/** 触发打包任务的请求体（ADR-0008：渠道 + versionName + 任务名 + 测试事件）。 */
export interface BuildJobRequest {
  brandCode: BrandCode;
  flavors: string[];
  /** X.Y.Z，前端校验 */
  versionName: string;
  /** 可空：留空后端用默认名 */
  jobName?: string;
  testEvents: boolean;
}

/** 一段构建日志（GET /api/build/jobs/:id/logs?after= 增量拉取）。 */
export interface BuildLogChunk {
  /** 增量游标：下次请求带上以拉取新增行 */
  cursor: number;
  lines: string[];
  /** 任务是否已结束（结束后停止轮询） */
  done: boolean;
  status: BuildStatus;
}

/** 登录态用户。token = accessToken（请求头携带）。 */
export interface AuthUser {
  username: string;
  role: 'admin' | 'operator' | 'viewer';
  /** access token（写入请求头 Authorization: Bearer） */
  token: string;
  /** refresh token（用于续期；本轮仅持有，未做自动刷新调度） */
  refreshToken?: string;
}

/** 后端登录端点返回体（auth.go tokenResp）。 */
export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  role: AuthUser['role'];
  username: string;
}

/** 统一响应信封：{ code, message, data }（httpx.Envelope，成功 code=0）。 */
export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

/** 列表信封：后端渠道列表返回 { items, total }。 */
export interface ListResult<T> {
  items: T[];
  total: number;
}

/**
 * 图标九宫格槽位定义（见 ADR-0005 / 03-build-and-icon-pipeline.md §2）。
 * variant 决定遮罩形态；px 是该 density 的边长。
 */
export type IconVariant = 'square' | 'round' | 'foreground';

export interface IconSlot {
  variant: IconVariant;
  dpi: 'mdpi' | 'hdpi' | 'xhdpi' | 'xxhdpi' | 'xxxhdpi';
  px: number;
  /** 单独覆盖该槽的图（dataURL）；为空则用主图 fan-out 的预览 */
  overrideDataUrl?: string;
}

/**
 * APK 运行时配置（GET /api/app/config?palcode=）。后端返回裸 JSON（无信封）。
 * 与编译期 assets/bootstrap.json 同构，便于三级取用合并（见 ADR-0002）。
 */
export interface AppRuntimeConfig {
  palcode: string;
  /** 主域名在前，最多 3 备用 */
  domains: string[];
  probePath: string;
  configVersion: number;
  ttlSeconds: number;
}

// =========================================================================
// 推送管理（07-push.md）
// =========================================================================

/** FCM 推送功能门控（GET /api/push/status）。 */
export interface PushStatus {
  enabled: boolean;
  brands: Record<BrandCode, boolean>;
}

/** 推送活动状态机。 */
export type PushCampaignStatus = 'draft' | 'scheduled' | 'sending' | 'done' | 'failed';

/**
 * 推送活动（对应后端 push_campaign 表）。
 * id 类型与 Channel 等保持一致（后端数字主键，前端字符串化）。
 */
export interface PushCampaign {
  id: string;
  name: string;
  title: string;
  body: string;
  imageUrl?: string;
  /** 相对 path，如 /promo/618；不含域名（ADR-0002）。 */
  deeplinkPath?: string;
  extraData?: Record<string, string>;
  /** 目标渠道包 applicationId 集合。 */
  targetAppIds: string[];
  status: PushCampaignStatus;
  scheduledAt?: string;
  sentAt?: string;
  totalDevices: number;
  successCount: number;
  failureCount: number;
  createdBy?: string;
  createdAt: string;
}

/** 推送活动创建/编辑载荷（POST/PUT /api/push/campaigns）。 */
export interface PushCampaignInput {
  name: string;
  title: string;
  body: string;
  imageUrl?: string;
  deeplinkPath?: string;
  extraData?: Record<string, string>;
  targetAppIds: string[];
}

/** 单 applicationId 的发送结果汇总（GET /api/push/campaigns/:id → records）。 */
export interface PushRecord {
  applicationId: string;
  sent: number;
  failed: number;
  errorSample?: string;
  finishedAt?: string;
}

/** 触达预估（GET /api/push/audience?appIds=...）。 */
export interface PushAudience {
  totalDevices: number;
  byApp: Record<string, number>;
}

// =========================================================================
// 上架包管理（docs/admin/09-listing.md / ADR-0014）
// =========================================================================

/** 上架包所属平台。 */
export type ListingPlatform = 'android' | 'ios';

/** 上架包状态，与渠道同一套语义（enabled/disabled 可切换；archived 走硬删除或软归档）。 */
export type ListingStatus = 'enabled' | 'disabled' | 'archived';

/**
 * AB 面网关规则（listing_gate，与上架包一对一）。
 * countries 必填非空才算「已配置」——空清单在后端语义上等同「配置无效」，不是「不限国家」。
 * CN/US 被服务端硬编码强制 A 面，前端也绝不提供这两个国家作为可选项（ADR-0014）。
 */
export interface ListingGateRule {
  countries: string[];
  /** 时区黑名单：命中即强制 A 面（选填，与 IP 黑名单同类）。 */
  timezoneDeny: string[];
  ipAllowCidrs: string[];
  ipDenyCidrs: string[];
  updatedAt?: string;
}

/** 上架包（对应后端 model.ListingApp，含品牌/域名/网关关联）。 */
export interface Listing {
  /** 后端数字主键（字符串化） */
  id: string;
  /** 域名继承的所属品牌；上架包本身与品牌小渠道包是两条独立产线（09-listing.md）。 */
  brandCode: BrandCode;
  platform: ListingPlatform;
  /** 包名；(platform, bundleId) 唯一，而非全局唯一（Flutter 双端可同包名）。 */
  bundleId: string;
  /** 内部代号，如 ColorStack */
  name: string;
  /** 商店展示名 */
  displayName: string;
  storeUrl: string;
  status: ListingStatus;
  /** 是否继承所属品牌的默认域名（ADR-0006 继承语义） */
  useBrandDomains: boolean;
  /** 参考渠道包的 PAL_CODE（运营手填，纯数字串），拼 B 面 /?palcode=。 */
  palCode?: string;
  /** AB 面总开关；打开前后端强制校验国家白名单非空（ADR-0014）。新建恒为 false。 */
  gateEnabled: boolean;
  /** AppsFlyer 账号级 Dev Key，可空 = 未接 AF */
  afDevKey: string;
  /** AppsFlyer 应用级 App ID（iOS 形如 id123456789，Android 为包名） */
  afAppId: string;
  /** Adjust App Token（复用 ADR-0013 跨层契约），空 = 未启用 Adjust */
  adjustAppToken?: string;
  /** Adjust 事件映射 `{ 事件name: token }` */
  adjustEvents?: Record<string, string>;
  remark: string;
  /** use_brand_domains=false 时生效的渠道级覆盖域名 */
  domains?: DomainEntry[];
  /** AB 面放行规则；未配置时为 undefined（前端据此判断「尚未配置国家白名单」） */
  gate?: ListingGateRule;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * 新增/编辑上架包的表单载荷（前端聚合）。api 层按后端端点拆分为多次请求提交：
 * 基本信息 → POST/PUT /listings；域名 → PUT /listings/:id/domains；
 * 网关规则 → PUT /listings/:id/gate；总开关 → PUT /listings/:id { gateEnabled }
 * （必须晚于网关规则保存，见 ADR-0014「打开前校验国家白名单非空」）。
 */
export interface ListingInput {
  brandCode: BrandCode;
  /** 新建必填；编辑态只读不下发（换平台等于换包）。 */
  platform: ListingPlatform;
  /** 新建必填；编辑态只读不下发。 */
  bundleId: string;
  name: string;
  displayName: string;
  storeUrl: string;
  /** 参考渠道包的 PAL_CODE（运营手填，纯数字串），必填；拼 B 面 /?palcode=。 */
  palCode: string;
  /** 新建时后端恒置 enabled，此字段仅编辑态下发。 */
  status: ListingStatus;
  remark: string;
  afDevKey: string;
  afAppId: string;
  /** 空字符串 = 不启用/解绑（与 Channel 的约定一致）。 */
  adjustAppToken?: string;
  adjustEvents?: Record<string, string>;
  useBrandDomains: boolean;
  domains?: DomainEntry[];
  /** 期望的 AB 面总开关目标态；api 层负责在网关规则保存之后再提交这一步。 */
  gateEnabled: boolean;
  gate: {
    countries: string[];
    timezoneDeny: string[];
    ipAllowCidrs: string[];
    ipDenyCidrs: string[];
  };
}

/** 网关试算结果（POST /listings/:id/gate/test，后台自查专用，含原因）。 */
export interface ListingGateTestResult {
  mode: 'A' | 'B';
  reason: string;
  country: string;
}

/** 网关判定流水一条记录（GET /listings/:id/gate/logs）。 */
export interface ListingGateLog {
  id: string;
  ip: string;
  country: string;
  timezone: string;
  decision: 'A' | 'B';
  reason: string;
  createdAt: string;
}

/**
 * POST /api/push/campaigns/:id/send 的响应体（envelope.data）。
 * dry-run 时 campaign 保持原状态（draft），preview 带触达数；
 * 真发时 campaign 状态变为 sending/done，preview 无意义（可忽略）。
 */
export interface PushSendResult {
  campaign: PushCampaign;
  dryRun: boolean;
  preview?: {
    totalDevices: number;
    byApp: Record<string, number>;
  };
}

// =========================================================================
// 上架包推送（09-listing.md §6）：ColorStack / DeckTallyPro 独立推送流程。
// 与渠道推送共用状态机字段与发送内核，差异仅两点：目标是 listingIds（而非
// targetAppIds）；发送时服务端**强制只投递最近判定为 B 面的活跃设备**
// （repo.ActiveListingTokensBMode 硬过滤，前端不提供、也不需要任何受众筛选项）。
// =========================================================================

/** 上架包推送活动创建载荷（POST /api/push/listing-campaigns）。 */
export interface ListingCampaignInput {
  name: string;
  title: string;
  body: string;
  imageUrl?: string;
  deeplinkPath?: string;
  extraData?: Record<string, string>;
  /** 目标上架包 Listing.id 集合；提交真实后端时转 number[]（后端字段 listingIds:[]uint64）。 */
  listingIds: string[];
}

/**
 * 上架包推送活动（对应后端 PushCampaignView，kind 恒为 'listing'）。
 * 注意：后端**没有编辑草稿的端点**（只有创建/列表/发送三个），Console 前端据此把
 * 创建后的活动视为不可再改——如需调整内容只能放弃当前草稿、重新创建一条。
 */
export interface ListingCampaign {
  id: string;
  kind: 'listing';
  name: string;
  title: string;
  body: string;
  imageUrl?: string;
  deeplinkPath?: string;
  extraData?: Record<string, string>;
  listingIds: string[];
  status: PushCampaignStatus;
  sentAt?: string;
  totalDevices: number;
  successCount: number;
  failureCount: number;
  createdBy?: string;
  createdAt: string;
}

/**
 * POST /api/push/listing-campaigns/:id/send 响应体（09-listing.md §6）。
 * dryRun=true：preview 为各目标上架包的 B 面活跃设备数，byApp 的 key 形如
 * `listing:<id>`（`<id>` 即 Listing.id）；dryRun=false：真发异步进行，响应仅
 * `{dryRun:false}`，需轮询 GET .../listing-campaigns 看 status/successCount/failureCount。
 */
export interface ListingCampaignSendResult {
  dryRun: boolean;
  preview?: {
    totalDevices: number;
    byApp: Record<string, number>;
  };
}
