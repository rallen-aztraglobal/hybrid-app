/**
 * 领域类型 —— 对齐**真实后端**（server/internal/...）的数据模型与 REST 契约。
 *
 * 本轮（第二轮）把第一轮的占位类型对齐到 Go 后端实际返回的字段：
 *  - 登录返回 { accessToken, refreshToken, expiresIn, role, username }（auth.go）
 *  - 品牌/品牌域名以 string[] 下发（brand.go / domain.go），前端用适配器转 DomainEntry[]
 *  - 渠道 id 为数字主键（channels.go 路径参数），前端统一以字符串持有
 *  - 构建：ADR-0008 的「打包任务」(build job) + 日志流 + 单 APK 下载 + 渠道最新包
 *  - 身份：ADR-0009 的 applicationId 由 品牌包前缀 + flavor 派生
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
