import type {
  AdminUser,
  AdminUserInput,
  ApiEnvelope,
  AuthUser,
  Brand,
  BrandCode,
  BuildArtifact,
  BuildJob,
  BuildJobRequest,
  BuildLogChunk,
  Channel,
  ChannelDevice,
  ChannelInput,
  DeviceFilter,
  DeviceListResult,
  DomainEntry,
  DomainInput,
  Listing,
  ListingCampaign,
  ListingCampaignInput,
  ListingCampaignSendResult,
  ListingGateLog,
  ListingGateTestResult,
  ListingInput,
  ListingOpenMode,
  ListResult,
  LoginResponse,
  MeResponse,
  PermCatalogModule,
  ProbeResult,
  PushAudience,
  PushCampaign,
  PushCampaignInput,
  PushSendResult,
  PushStatus,
  Role,
  RoleInput,
  RoleScope,
  ScopeDTO,
  Store,
  StoreInput,
  StoreUpdateInput,
} from './types';
import { FULL_ROLE_SCOPE } from './types';
import { mockDb } from './mock/db';
import { devicesByFilter, devicesByIds, queryDevices } from './mock/devices';
import { mockListingDb } from './mock/listings';
import { mockListingCampaignDb } from './mock/listingCampaigns';
import { mockRbacDb } from './mock/rbac';
import { BRAND_META } from './brands';

/**
 * API 客户端 —— 对齐**真实后端**（server/internal/handler 路由 + httpx.Envelope）。
 *
 * 第二轮：把第一轮的 demo/mock 接到真实 /api。策略：
 *  - 真实后端为**主**路径，每个端点做精确的「响应整形（adapter）」，把 Go 的字段
 *    形态（string[] 域名 / {items,total} / accessToken / 数字 id）转成 UI 类型。
 *  - 仅当真实请求**失败**（后端未起 / 网络断 / 端点尚未上线）时，透明回退到本地
 *    mock，保证前端可独立 `pnpm dev` 演示完整闭环（VITE_USE_MOCK=1 可强制 mock）。
 *  - ADR-0008 的 /build/jobs + 日志流、ADR-0009 的 packagePrefix 等「下一轮后端落地」
 *    的端点：客户端已按 ADR 契约实现，后端未就绪时由 fallback 给出**模拟**响应，UI 即开即用。
 */

const FORCE_MOCK = import.meta.env.VITE_USE_MOCK === '1';
const MOCK_LATENCY = 180; // 模拟网络延迟，让 loading 态可见

function delay<T>(value: T, ms = MOCK_LATENCY): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

/** 统一 fetch：带 JWT、解信封（成功 code=0）、非 2xx 抛错（带后端 message）。 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
  });
  // 后端错误也走 Envelope（{code,message}），优先取其 message 抛出。
  let json: ApiEnvelope<T> | undefined;
  try {
    json = (await res.json()) as ApiEnvelope<T>;
  } catch {
    /* 空响应体 */
  }
  // token 失效/过期（401）：清登录态并跳回登录页，而不是把空数据丢给页面
  // （否则用户会误以为"数据没了"）。仅豁免登录/刷新本身的 401（账号密码错 / refresh token
  // 过期，属业务错误，不代表当前登录态失效）。注意用路径前缀（`?` 之前）比对，避免带 query
  // 的写法（理论上这两个端点不带 query，此处仍做防御）被误判命中/漏判。
  // /auth/me 的 401 **必须**触发清态——账号被删/token 失效后启动时静默刷新 perms 失败，
  // 不能让用户顶着旧权限留在界面上，直到踩到下一个 401 才被踢下线。
  const pathname = path.split('?')[0];
  const exemptFrom401Handling = pathname === '/auth/login' || pathname === '/auth/refresh';
  if (res.status === 401 && token && !exemptFrom401Handling) {
    setToken(null);
    try {
      localStorage.removeItem('hybrid:auth'); // 清 zustand persist 的登录态
    } catch {
      /* ignore */
    }
    if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
      window.location.assign('/login');
    }
  }
  if (!res.ok) {
    throw new ApiError(json?.message || `HTTP ${res.status} ${res.statusText}`, res.status);
  }
  if (json && json.code !== 0 && json.code !== 200) {
    throw new ApiError(json.message || '请求失败', json.code);
  }
  return (json?.data as T) ?? (undefined as T);
}

/** 带 HTTP 状态码的错误，便于上层区分 401/404/409。 */
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/**
 * 单体资源（渠道 / 品牌域名等）404 的展示文案（10-rbac.md「数据权限」：越界返回 404 而非 403，
 * 不泄露「存在一个你看不到的资源」这一事实）。裸的「资源不存在」对运营是误导性文案——
 * 更常见的原因其实是角色数据范围被收紧了，而不是数据真的被删。这里统一措辞、
 * 保留后端原始 message（如有），并加一句数据范围提示；非 404 原样透传 err.message。
 * 覆盖端点：GET/PUT/DELETE /channels/:id、/channels/:id/{domains,icon,splash,...}、
 * GET/PUT /brands/:code/domains。
 */
export function friendlyNotFoundMessage(err: unknown, fallback = '操作失败'): string {
  if (err instanceof ApiError && err.status === 404) {
    const raw = err.message && !/^HTTP 404\b/.test(err.message) ? err.message : '未找到该资源';
    return `${raw}（可能已被删除，或不在你当前角色的数据范围内；如有疑问请联系管理员核对角色的品牌/渠道范围）`;
  }
  return err instanceof Error ? err.message : fallback;
}

/**
 * 包一层：FORCE_MOCK 直接走 mock；否则真实请求失败再回退 mock。
 *
 * 关键区分（接真实后端的正确性）：只在「后端不可达 / 端点尚未上线」时回退；
 * **业务错误（4xx，如 400 校验失败 / 409 唯一性冲突 / 401 未授权）必须如实抛给用户**，
 * 否则真实 409 会被 mock 的 upsert 掩盖成「保存成功」，违背唯一性护栏。
 * 视为「不可达」的情形：fetch 抛 TypeError（网络层）或 5xx / 404（端点缺失）。
 */
function shouldFallback(err: unknown): boolean {
  if (err instanceof ApiError) {
    // 404 = 端点未上线（下一轮后端落地前）；5xx = 服务异常 → 回退演示。其余 4xx 抛出。
    return err.status === 404 || err.status >= 500;
  }
  // fetch 网络错误（后端没起）→ TypeError。
  return true;
}

async function withFallback<T>(real: () => Promise<T>, mock: () => T): Promise<T> {
  if (FORCE_MOCK) return delay(mock());
  try {
    return await real();
  } catch (err) {
    if (!shouldFallback(err)) throw err;
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.warn('[api] 后端不可达/端点缺失，回退到本地 mock（来源 channels/*.csv）:', (err as Error)?.message);
    }
    return delay(mock());
  }
}

/**
 * 写操作专用：只有 VITE_USE_MOCK=1 显式开启时才走 mock；真实模式下即便后端 404/5xx
 * 也直接把错误原样抛给调用方，绝不能悄悄落到 mock 兜底再对着真后端发 DELETE/PUT。
 *
 * 背景（评审「应修2」）：RBAC 的 rolesApi/usersApi 列表读接口用 withFallback——后端
 * RBAC 路由未就绪时回退到 mock 的演示数据（id 从 1 开始）。但如果写操作也用同一套
 * withFallback，会出现「列表因为 404/5xx 显示了 mock 的 id=2，用户点删除/改角色时，
 * 单条写请求 DELETE/PUT /api/users/2 却是发去真实后端」的场景——一旦真库里恰好也有
 * id=2 的账号，就会被误删/误改，而不是「本来就应该失败」。因此角色/用户的
 * create/update/remove/resetPassword 一律走这里：真实模式下 100% 直达后端，
 * 失败就如实报错，不静默兜底伪装成功。
 */
async function realOnly<T>(real: () => Promise<T>, mock: () => T): Promise<T> {
  if (FORCE_MOCK) return delay(mock());
  return real();
}

// ---------- token 存取（与 auth store 共用 localStorage key） ----------
const TOKEN_KEY = 'hybrid:token';
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}
export function setToken(token: string | null): void {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

// =========================================================================
// 适配器：后端形态 → UI 类型
// =========================================================================

/** 后端 string[] 域名 → DomainEntry[]（position 即下标，主在前）。 */
function domainsFromUrls(urls: string[] | null | undefined, probes?: ProbeResult[]): DomainEntry[] {
  const byUrl = new Map((probes ?? []).map((p) => [p.url, p]));
  return (urls ?? []).map((url, i) => {
    const p = byUrl.get(url);
    return {
      position: i,
      url,
      enabled: true,
      health: p ? (p.ok ? 'ok' : 'down') : 'unknown',
      latencyMs: p?.latencyMs,
    };
  });
}

/** DomainEntry[] → 后端 DomainInput[]（去空、规范化 position）。 */
function domainsToInput(domains: DomainEntry[]): DomainInput[] {
  return domains
    .filter((d) => d.url.trim())
    .map((d) => ({ position: d.position, url: d.url.trim(), enabled: d.enabled }));
}

/** 后端 service.BrandView → UI Brand（domains: string[] → DomainEntry[]）。 */
interface BrandViewDTO {
  id?: number;
  code: BrandCode;
  name: string;
  scheme: string;
  hmsEnabled: boolean;
  accentColor: string;
  channelCount: number;
  domains: string[];
  packagePrefix?: string;
}
function adaptBrand(b: BrandViewDTO): Brand {
  return {
    id: b.id != null ? String(b.id) : undefined,
    code: b.code,
    name: b.name,
    scheme: b.scheme,
    hmsEnabled: b.hmsEnabled,
    // 后端 accentColor 与原型略有出入；UI 配色以 BRAND_META 为权威（与 docs/admin/ui 一致）。
    accentColor: BRAND_META[b.code]?.accentColor ?? b.accentColor,
    channelCount: b.channelCount,
    domains: domainsFromUrls(b.domains),
    packagePrefix: b.packagePrefix || BRAND_META[b.code]?.packagePrefix,
  };
}

/** 后端 model.Channel（数字 id + brandId）→ UI Channel（字符串 id + brandCode）。 */
interface ChannelDTO {
  id: number;
  brandCode?: BrandCode;
  brand?: { code?: BrandCode } | null;
  flavorName: string;
  applicationId: string;
  palCode: string;
  appName: string;
  status: Channel['status'];
  useBrandDomains: boolean;
  domains?: { position: number; url: string; enabled: boolean }[] | null;
  iconMasterUrl?: string;
  splashUrl?: string;
  remark?: string;
  liveVersion?: string;
  latestApkUrl?: string;
  updatedAt?: string;
  storeId?: number | null;
  store?: { id: number; code: string; name: string } | null;
  /**
   * Adjust 归因（08-adjust.md / ADR-0013）。**待办**：server 的 Create/UpdateChannelInput
   * OpenAPI 契约尚未收录这两个字段，此处先行手写对接，后端补齐并重新生成 OpenAPI 后需对齐/替换。
   */
  adjustAppToken?: string | null;
  adjustEvents?: Record<string, string> | null;
}
function adaptChannel(c: ChannelDTO, brandHint?: BrandCode): Channel {
  const brandCode = (c.brandCode ?? c.brand?.code ?? brandHint ?? 'ap') as BrandCode;
  return {
    id: String(c.id),
    brandCode,
    flavorName: c.flavorName,
    applicationId: c.applicationId,
    palCode: c.palCode,
    appName: c.appName,
    status: c.status,
    useBrandDomains: c.useBrandDomains,
    domains: c.domains?.length
      ? c.domains.map((d) => ({ position: d.position, url: d.url, enabled: d.enabled, health: 'unknown' as const }))
      : undefined,
    iconMasterUrl: c.iconMasterUrl || undefined,
    splashUrl: c.splashUrl || undefined,
    remark: c.remark || undefined,
    liveVersion: c.liveVersion || undefined,
    latestApkUrl: c.latestApkUrl || undefined,
    updatedAt: c.updatedAt,
    storeId: c.storeId ?? null,
    store: c.store ?? null,
    adjustAppToken: c.adjustAppToken || undefined,
    adjustEvents: c.adjustEvents && Object.keys(c.adjustEvents).length ? c.adjustEvents : undefined,
  };
}

/** 后端 model.ListingDomain（09-listing.md）。 */
interface ListingDomainDTO {
  id?: number;
  listingId?: number;
  position: number;
  url: string;
  enabled: boolean;
}

/** 后端 model.ListingGate（一对一）；countries/timezoneDeny/ipAllowCidrs/ipDenyCidrs 空清单存 NULL。 */
interface ListingGateDTO {
  id?: number;
  listingId?: number;
  countries: string[] | null;
  timezoneDeny: string[] | null;
  ipAllowCidrs: string[] | null;
  ipDenyCidrs: string[] | null;
  updatedAt?: string;
}

/** 后端 model.ListingApp（含 Preload 的 Brand/Domains/Gate）。 */
interface ListingDTO {
  id: number;
  brandId?: number;
  platform: Listing['platform'];
  bundleId: string;
  name: string;
  displayName: string;
  storeUrl: string;
  status: Listing['status'];
  /** 后端字段；打开方式（09-listing.md），可能缺失（未接口对齐/旧数据）时前端兜底为 internal。 */
  openMode?: ListingOpenMode | null;
  useBrandDomains: boolean;
  palCode?: string | null;
  gateEnabled: boolean;
  afDevKey: string;
  afAppId: string;
  adjustAppToken?: string | null;
  adjustEvents?: Record<string, string> | null;
  remark: string;
  createdAt?: string;
  updatedAt?: string;
  brand?: { id?: number; code?: BrandCode; name?: string } | null;
  domains?: ListingDomainDTO[] | null;
  gate?: ListingGateDTO | null;
}

/** 后端 model.ListingGateLog。 */
interface ListingGateLogDTO {
  id: number;
  ip: string;
  country: string;
  timezone: string;
  decision: 'A' | 'B';
  reason: string;
  createdAt: string;
}

/** 后端 ListingApp → UI Listing（数字 id 字符串化，domains/gate 展开为 UI 友好形态）。 */
function adaptListing(l: ListingDTO): Listing {
  return {
    id: String(l.id),
    // ListListings/GetListing 恒 Preload("Brand")，理论上不会缺失；缺失时回落 'ap' 仅为类型安全兜底。
    brandCode: (l.brand?.code ?? 'ap') as BrandCode,
    platform: l.platform,
    bundleId: l.bundleId,
    name: l.name,
    displayName: l.displayName || '',
    storeUrl: l.storeUrl || '',
    status: l.status,
    openMode: l.openMode === 'external' ? 'external' : 'internal',
    useBrandDomains: l.useBrandDomains,
    palCode: l.palCode || undefined,
    gateEnabled: l.gateEnabled,
    afDevKey: l.afDevKey || '',
    afAppId: l.afAppId || '',
    adjustAppToken: l.adjustAppToken || undefined,
    adjustEvents: l.adjustEvents && Object.keys(l.adjustEvents).length ? l.adjustEvents : undefined,
    remark: l.remark || '',
    domains: l.domains?.length
      ? l.domains.map((d) => ({ position: d.position, url: d.url, enabled: d.enabled, health: 'unknown' as const }))
      : undefined,
    gate: l.gate
      ? {
          countries: l.gate.countries ?? [],
          timezoneDeny: l.gate.timezoneDeny ?? [],
          ipAllowCidrs: l.gate.ipAllowCidrs ?? [],
          ipDenyCidrs: l.gate.ipDenyCidrs ?? [],
          updatedAt: l.gate.updatedAt,
        }
      : undefined,
    createdAt: l.createdAt,
    updatedAt: l.updatedAt,
  };
}

function adaptGateLog(g: ListingGateLogDTO): ListingGateLog {
  return {
    id: String(g.id),
    ip: g.ip || '',
    country: g.country || '',
    timezone: g.timezone || '',
    decision: g.decision,
    reason: g.reason || '',
    createdAt: g.createdAt,
  };
}

/**
 * 登录/刷新/me 响应体里嵌套的数据范围（10-rbac.md「数据权限」）→ UI：ScopeDTO.channelIds
 * 是数字数组，RoleScope.channelIds 是字符串（与其它「id 全部字符串化」的 UI 类型一致）。
 * 注意角色端点 /api/roles 用的是**平铺**字段，转换见 adaptRoleScope/roleScopeToPayload。
 */
function adaptScope(s: ScopeDTO | null | undefined): RoleScope {
  if (!s) return FULL_ROLE_SCOPE;
  return {
    allBrands: s.allBrands,
    brands: (s.brands ?? []) as BrandCode[],
    allChannels: s.allChannels,
    channelIds: (s.channelIds ?? []).map(String),
  };
}

// =========================================================================
// 鉴权
// =========================================================================
export const authApi = {
  async login(username: string, password: string): Promise<AuthUser> {
    return withFallback<AuthUser>(
      async () => {
        const r = await request<LoginResponse>('/auth/login', {
          method: 'POST',
          body: JSON.stringify({ username, password }),
        });
        return {
          username: r.username,
          role: r.role,
          perms: r.perms ?? [],
          scope: adaptScope(r.scope),
          token: r.accessToken,
          refreshToken: r.refreshToken,
        };
      },
      // mock：仅接受种子账号（与真实后端 bootstrap 账号一致：admin / admin12345）
      () => mockRbacDb.login(username, password),
    );
  },

  /** 启动时/角色变更后刷新 role+perms+scope（10-rbac.md：无需重登即可生效）。失败由调用方静默兜底。 */
  me(): Promise<{ role: AuthUser['role']; perms: string[]; scope: RoleScope }> {
    return withFallback(
      async () => {
        const r = await request<MeResponse>('/auth/me');
        return { role: r.role, perms: r.perms ?? [], scope: adaptScope(r.scope) };
      },
      () => mockRbacDb.me(getToken()),
    );
  },
};

// =========================================================================
// 大渠道 + 域名
// =========================================================================
export const brandApi = {
  list(): Promise<Brand[]> {
    return withFallback(
      async () => (await request<BrandViewDTO[]>('/brands')).map(adaptBrand),
      () => mockDb.listBrands(),
    );
  },
  getDomains(code: BrandCode): Promise<DomainEntry[]> {
    return withFallback(
      async () => {
        // 后端返回 { domains: string[] }
        const r = await request<{ domains: string[] }>(`/brands/${code}/domains`);
        return domainsFromUrls(r.domains);
      },
      () => mockDb.getBrandDomains(code),
    );
  },
  updateDomains(code: BrandCode, domains: DomainEntry[]): Promise<DomainEntry[]> {
    return withFallback(
      async () => {
        // 后端返回 { domains: string[], probes: ProbeResult[] }
        const r = await request<{ domains: string[]; probes?: ProbeResult[] }>(`/brands/${code}/domains`, {
          method: 'PUT',
          body: JSON.stringify({ domains: domainsToInput(domains) }),
        });
        return domainsFromUrls(r.domains, r.probes);
      },
      () => mockDb.setBrandDomains(code, domains),
    );
  },
};

// =========================================================================
// 小渠道 CRUD
// =========================================================================
export const channelApi = {
  list(): Promise<Channel[]> {
    return withFallback(
      async () => {
        // 后端 { items, total }；一次性拉大页（中台规模可控）。
        // 注意 repo 把 pageSize 上限钳在 500（>500 会被重置为默认 50），故取 500。
        const r = await request<ListResult<ChannelDTO>>('/channels?pageSize=500');
        return r.items.map((c) => adaptChannel(c));
      },
      () => mockDb.listChannels(),
    );
  },
  get(id: string): Promise<Channel | undefined> {
    return withFallback(
      async () => adaptChannel(await request<ChannelDTO>(`/channels/${id}`)),
      () => mockDb.getChannel(id),
    );
  },
  create(input: ChannelInput): Promise<Channel> {
    return withFallback(
      async () => {
        // 1) 基本信息（后端 CreateChannelInput：不含域名/图标）。
        // adjustAppToken/adjustEvents：08-adjust.md §6 跨层契约——随渠道保存接口一起提交，
        // server 只存不解析；OpenAPI 契约待后端补齐后对齐（见 ChannelDTO 注释）。
        const created = await request<ChannelDTO>('/channels', {
          method: 'POST',
          body: JSON.stringify({
            brandCode: input.brandCode,
            flavorName: input.flavorName.trim(),
            applicationId: input.applicationId.trim(),
            palCode: input.palCode.trim(),
            appName: input.appName.trim(),
            remark: input.remark ?? '',
            liveVersion: input.liveVersion?.trim() ?? '',
            storeId: input.storeId ?? null,
            adjustAppToken: input.adjustAppToken?.trim() ?? '',
            adjustEvents: input.adjustEvents ?? {},
          }),
        });
        const ch = adaptChannel(created, input.brandCode);
        // 2) 副作用：域名覆盖 / 图标 / splash（容错，不阻断主流程）。
        await applyChannelSideEffects(ch.id, input);
        return ch;
      },
      () => mockDb.upsertChannel(inputToChannel(input)),
    );
  },
  update(id: string, input: ChannelInput): Promise<Channel> {
    return withFallback(
      async () => {
        const updated = await request<ChannelDTO>(`/channels/${id}`, {
          method: 'PUT',
          body: JSON.stringify({
            // applicationId 派生只读：flavor 不可改（编辑态禁用），故不下发 flavorName/applicationId。
            // storeId 同理：编辑态商店下拉禁用，不下发（避免误改导致包名/flavor 语义错位）。
            palCode: input.palCode.trim(),
            appName: input.appName.trim(),
            status: input.status,
            remark: input.remark ?? '',
            liveVersion: input.liveVersion?.trim() ?? '',
            // adjustAppToken/adjustEvents：08-adjust.md §6 跨层契约，随保存接口一起提交。
            adjustAppToken: input.adjustAppToken?.trim() ?? '',
            adjustEvents: input.adjustEvents ?? {},
          }),
        });
        const ch = adaptChannel(updated, input.brandCode);
        await applyChannelSideEffects(ch.id, input);
        return ch;
      },
      () => mockDb.upsertChannel({ ...inputToChannel(input), id }),
    );
  },
  /**
   * 卡片就地改「线上版本号」：只 PUT 这一个字段（后端 UpdateChannelInput 指针字段，未传的不动），
   * 不会连带覆盖 palCode/appName 等——避免列表页与抽屉并发编辑时互相踩。空串 = 清空。
   */
  updateLiveVersion(id: string, liveVersion: string): Promise<Channel> {
    const v = liveVersion.trim();
    return withFallback(
      async () => {
        const updated = await request<ChannelDTO>(`/channels/${id}`, {
          method: 'PUT',
          body: JSON.stringify({ liveVersion: v }),
        });
        return adaptChannel(updated);
      },
      () => {
        const cur = mockDb.getChannel(id);
        if (!cur) throw new Error('渠道不存在');
        return mockDb.upsertChannel({ ...cur, liveVersion: v || undefined });
      },
    );
  },
  archive(id: string): Promise<void> {
    return withFallback(
      async () => {
        await request<{ archived: boolean }>(`/channels/${id}`, { method: 'DELETE' });
      },
      () => mockDb.archiveChannel(id),
    );
  },
};

// =========================================================================
// 设备管理：GET /api/devices（渠道包内 Android 设备注册流水）
// =========================================================================

/** CSV 单元格转义（含逗号/引号/换行时套双引号，内部引号转义为两个双引号）。 */
function csvCell(v: string | number | undefined): string {
  const s = v == null ? '' : String(v);
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
}

const DEVICE_CSV_HEADER = ['设备名称', 'GAID', 'ADID', 'OAID', '应用名', 'PAL_CODE', '包名', '品牌', '注册时间', '最后活跃时间'];

function devicesToCsv(items: ChannelDevice[]): string {
  const lines = [DEVICE_CSV_HEADER.map(csvCell).join(',')];
  for (const d of items) {
    lines.push(
      [d.deviceName, d.gaid, d.adid, d.oaid, d.appName, d.palCode, d.applicationId, d.brandCode, d.createdAt, d.updatedAt]
        .map(csvCell)
        .join(','),
    );
  }
  return lines.join('\r\n');
}

/** 触发浏览器下载一个 Blob（临时 <a> 元素 click，随后回收）。 */
function triggerBlobDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

/** CSV 文本 → Blob 下载（前置 BOM 保证 Excel 正确识别 UTF-8 中文）。 */
function downloadCsvText(csv: string, filename: string): void {
  triggerBlobDownload(new Blob([`﻿${csv}`], { type: 'text/csv;charset=utf-8;' }), filename);
}

/** 设备导出/列表共用的筛选字段（不含分页）。 */
export type DeviceExportFilter = Pick<
  DeviceFilter,
  'applicationIds' | 'device' | 'packageName' | 'from' | 'to' | 'activeFrom' | 'activeTo'
>;

/** 设备筛选参数 → URLSearchParams（空值不带；渠道多选逗号拼接）。 */
function deviceFilterParams(filter: DeviceExportFilter): URLSearchParams {
  const params = new URLSearchParams();
  if (filter.applicationIds?.length) params.set('applicationId', filter.applicationIds.join(','));
  if (filter.device) params.set('device', filter.device);
  if (filter.packageName) params.set('packageName', filter.packageName);
  if (filter.from) params.set('from', filter.from);
  if (filter.to) params.set('to', filter.to);
  if (filter.activeFrom) params.set('activeFrom', filter.activeFrom);
  if (filter.activeTo) params.set('activeTo', filter.activeTo);
  return params;
}

export const deviceApi = {
  /** 设备列表（分页 + 筛选）。翻页过深 / 其它业务错误由后端 400 中文 message 如实抛出，不回退 mock。 */
  listDevices(filter: DeviceFilter): Promise<DeviceListResult> {
    const params = deviceFilterParams(filter);
    params.set('page', String(filter.page));
    params.set('pageSize', String(filter.pageSize));
    return withFallback(
      () => request<DeviceListResult>(`/devices?${params.toString()}`),
      () => queryDevices(filter),
    );
  },

  /**
   * 导出勾选设备（POST /api/devices/export，body {ids}，≤1000）。
   * 真实模式：fetch → Blob → 临时 <a download> 触发下载；mock 模式：前端拼 CSV。
   */
  async exportDevicesSelected(ids: number[]): Promise<void> {
    if (FORCE_MOCK) {
      downloadCsvText(devicesToCsv(devicesByIds(ids)), 'devices_selected.csv');
      return;
    }
    try {
      const token = getToken();
      const res = await fetch('/api/devices/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ ids }),
      });
      if (!res.ok) {
        let message = `HTTP ${res.status} ${res.statusText}`;
        try {
          const json = (await res.json()) as ApiEnvelope<unknown>;
          if (json?.message) message = json.message;
        } catch {
          /* 非 JSON 错误体，保留默认 message */
        }
        throw new ApiError(message, res.status);
      }
      triggerBlobDownload(await res.blob(), 'devices_selected.xlsx');
    } catch (err) {
      if (!shouldFallback(err)) throw err;
      downloadCsvText(devicesToCsv(devicesByIds(ids)), 'devices_selected.csv');
    }
  },

  /**
   * 按当前筛选导出全部（GET /api/devices/export-token 取一次性 token，再拼
   * /api/devices/export.xlsx?token=… 原生 <a href> 下载，大文件不过 JS 内存）。
   * 导出为美化 XLSX（中文表头/列宽/冻结首行）；原始 CSV 通道 export.csv 服务端仍保留。
   */
  async exportDevicesByFilter(filter: DeviceExportFilter): Promise<void> {
    if (FORCE_MOCK) {
      downloadCsvText(devicesToCsv(devicesByFilter(filter)), 'devices_export.csv');
      return;
    }
    try {
      const { token } = await request<{ token: string; expiresIn: number }>('/devices/export-token');
      const params = deviceFilterParams(filter);
      params.set('token', token);
      const a = document.createElement('a');
      a.href = `/api/devices/export.csv?${params.toString()}`;
      a.rel = 'noopener';
      document.body.appendChild(a);
      a.click();
      a.remove();
    } catch (err) {
      if (!shouldFallback(err)) throw err;
      downloadCsvText(devicesToCsv(devicesByFilter(filter)), 'devices_export.csv');
    }
  },
};

// =========================================================================
// 上架包（09-listing.md / ADR-0014）：ColorStack / DeckTallyPro 等独立合规应用
// =========================================================================
export const listingApi = {
  /** 全量上架包（含品牌/域名/网关关联）；数量小，一次拉全量，页面自行按平台/状态/关键词过滤。 */
  list(): Promise<Listing[]> {
    return withFallback(
      async () => {
        const r = await request<ListResult<ListingDTO>>('/listings');
        return r.items.map(adaptListing);
      },
      () => mockListingDb.list(),
    );
  },
  get(id: string): Promise<Listing | undefined> {
    return withFallback(
      async () => adaptListing(await request<ListingDTO>(`/listings/${id}`)),
      () => mockListingDb.get(id),
    );
  },
  create(input: ListingInput): Promise<Listing> {
    return withFallback(
      async () => {
        const created = await request<ListingDTO>('/listings', {
          method: 'POST',
          body: JSON.stringify({
            brandCode: input.brandCode,
            platform: input.platform,
            bundleId: input.bundleId.trim(),
            name: input.name.trim(),
            displayName: input.displayName.trim(),
            storeUrl: input.storeUrl.trim(),
            openMode: input.openMode,
            palCode: input.palCode.trim(),
            afDevKey: input.afDevKey.trim(),
            afAppId: input.afAppId.trim(),
            // *string：始终携带（空串亦然）以「声明意图」，与 updateListingReq 的解绑约定保持一致。
            adjustAppToken: input.adjustAppToken?.trim() ?? '',
            adjustEvents: input.adjustEvents ?? {},
            remark: input.remark.trim(),
          }),
        });
        const id = String(created.id);
        await applyListingSideEffects(id, input);
        return (await listingApi.get(id)) ?? adaptListing(created);
      },
      () => mockListingDb.create(input),
    );
  },
  update(id: string, input: ListingInput): Promise<Listing> {
    return withFallback(
      async () => {
        await request<ListingDTO>(`/listings/${id}`, {
          method: 'PUT',
          body: JSON.stringify({
            brandCode: input.brandCode,
            name: input.name.trim(),
            displayName: input.displayName.trim(),
            storeUrl: input.storeUrl.trim(),
            status: input.status,
            openMode: input.openMode,
            palCode: input.palCode.trim(),
            afDevKey: input.afDevKey.trim(),
            afAppId: input.afAppId.trim(),
            adjustAppToken: input.adjustAppToken?.trim() ?? '',
            adjustEvents: input.adjustEvents ?? {},
            remark: input.remark.trim(),
          }),
        });
        await applyListingSideEffects(id, input);
        return (await listingApi.get(id))!;
      },
      () => mockListingDb.update(id, input),
    );
  },
  remove(id: string): Promise<void> {
    return withFallback(
      async () => {
        await request<{ deleted: boolean }>(`/listings/${id}`, { method: 'DELETE' });
      },
      () => mockListingDb.remove(id),
    );
  },
  /** 后台试算：用指定 IP/时区跑一次判定，返回原因供保存前自查（与线上端点相反，线上不回原因）。 */
  testGate(id: string, ip: string, timezone: string): Promise<ListingGateTestResult> {
    return withFallback(
      () =>
        request<ListingGateTestResult>(`/listings/${id}/gate/test`, {
          method: 'POST',
          body: JSON.stringify({ ip, timezone }),
        }),
      () => mockListingDb.testGate(id, ip, timezone),
    );
  },
  /** 判定流水（排查「为什么没进 B 面」）。 */
  gateLogs(id: string, limit = 100): Promise<ListingGateLog[]> {
    return withFallback(
      async () => {
        const r = await request<ListResult<ListingGateLogDTO>>(`/listings/${id}/gate/logs?limit=${limit}`);
        return r.items.map(adaptGateLog);
      },
      () => mockListingDb.gateLogs(id, limit),
    );
  },
};

/**
 * 上架包副作用：域名覆盖 → 网关规则 → AB 面总开关，顺序不可乱：
 * 打开总开关前后端会重新查库校验「国家白名单非空」（ADR-0014），
 * 故必须等网关规则的 PUT 先落库成功，才能提交开关这一步。
 * 与 Channel 的「best-effort、失败仅告警」不同：这里任何一步失败都直接抛出，
 * 不静默吞掉——AB 面网关是合规相关功能，域名/规则没保存成功却让用户以为成功了不可接受。
 */
async function applyListingSideEffects(id: string, input: ListingInput): Promise<void> {
  if (input.useBrandDomains) {
    await request(`/listings/${id}/domains`, {
      method: 'PUT',
      body: JSON.stringify({ inheritBrand: true, domains: [] }),
    });
  } else if (input.domains && input.domains.some((d) => d.url.trim())) {
    await request(`/listings/${id}/domains`, {
      method: 'PUT',
      body: JSON.stringify({ inheritBrand: false, domains: domainsToInput(input.domains) }),
    });
  }

  const countries = Array.from(new Set(input.gate.countries.map((c) => c.trim().toUpperCase()).filter(Boolean)));
  if (countries.length > 0) {
    await request(`/listings/${id}/gate`, {
      method: 'PUT',
      body: JSON.stringify({
        countries,
        timezoneDeny: input.gate.timezoneDeny.map((t) => t.trim()).filter(Boolean),
        ipAllowCidrs: input.gate.ipAllowCidrs.map((c) => c.trim()).filter(Boolean),
        ipDenyCidrs: input.gate.ipDenyCidrs.map((c) => c.trim()).filter(Boolean),
      }),
    });
  }

  // 总开关：必须晚于网关规则保存（见上），否则国家白名单尚未落库会被后端 400 拒绝。
  await request(`/listings/${id}`, { method: 'PUT', body: JSON.stringify({ gateEnabled: input.gateEnabled }) });
}

// =========================================================================
// 上架包推送（09-listing.md §6）：ColorStack/DeckTallyPro 独立推送流程。
// 与渠道推送（pushApi）共用后端发送内核，但目标是 listingIds，且发送时服务端
// **强制只投递最近判定为 B 面的活跃设备**（无 UI 选项可绕过），故本层只透传 dryRun。
// =========================================================================

/** 后端 PushCampaignView（kind=listing）。数字 id/listingIds 需字符串化以对齐 UI 约定。 */
interface ListingCampaignDTO {
  id: number;
  kind: string;
  name: string;
  title: string;
  body: string;
  imageUrl?: string;
  deeplinkPath?: string;
  extraData?: Record<string, string>;
  listingIds?: number[];
  status: ListingCampaign['status'];
  sentAt?: string;
  totalDevices: number;
  successCount: number;
  failureCount: number;
  createdBy?: string;
  createdAt: string;
}
function adaptListingCampaign(c: ListingCampaignDTO): ListingCampaign {
  return {
    id: String(c.id),
    kind: 'listing',
    name: c.name,
    title: c.title,
    body: c.body,
    imageUrl: c.imageUrl || undefined,
    deeplinkPath: c.deeplinkPath || undefined,
    extraData: c.extraData && Object.keys(c.extraData).length ? c.extraData : undefined,
    listingIds: (c.listingIds ?? []).map(String),
    status: c.status,
    sentAt: c.sentAt || undefined,
    totalDevices: c.totalDevices,
    successCount: c.successCount,
    failureCount: c.failureCount,
    createdBy: c.createdBy || undefined,
    createdAt: c.createdAt,
  };
}

export const listingCampaignApi = {
  /** 上架包推送活动列表（kind=listing，GET /api/push/listing-campaigns）。 */
  listCampaigns(): Promise<ListingCampaign[]> {
    return withFallback(
      async () => {
        const r = await request<ListResult<ListingCampaignDTO>>('/push/listing-campaigns');
        return r.items.map(adaptListingCampaign);
      },
      () => mockListingCampaignDb.list(),
    );
  },

  /** 创建草稿（POST /api/push/listing-campaigns）。注意：后端无编辑端点，创建后内容不可再改。 */
  createCampaign(input: ListingCampaignInput): Promise<ListingCampaign> {
    return withFallback(
      async () => {
        const created = await request<ListingCampaignDTO>('/push/listing-campaigns', {
          method: 'POST',
          body: JSON.stringify({
            name: input.name,
            title: input.title,
            body: input.body,
            imageUrl: input.imageUrl ?? '',
            deeplinkPath: input.deeplinkPath ?? '',
            extraData: input.extraData ?? {},
            // 后端 ListingIDs 为 []uint64；UI 侧统一以字符串持有 Listing.id，这里转数字。
            listingIds: input.listingIds.map((id) => Number(id)),
          }),
        });
        return adaptListingCampaign(created);
      },
      () => mockListingCampaignDb.create(input),
    );
  },

  /**
   * 发送（POST /api/push/listing-campaigns/:id/send）。
   * dryRun=true → 预览各目标上架包的 B 面活跃设备数；dryRun=false → 异步真发（强制只投 B 面设备）。
   */
  sendCampaign(id: string, dryRun: boolean): Promise<ListingCampaignSendResult> {
    return withFallback(
      () =>
        request<ListingCampaignSendResult>(`/push/listing-campaigns/${id}/send`, {
          method: 'POST',
          body: JSON.stringify({ dryRun }),
        }),
      () => mockListingCampaignDb.send(id, dryRun),
    );
  },
};

// =========================================================================
// 应用商店（渠道包发布商店：华为 / 应用宝 / Google Play 等）
// =========================================================================

/** 进程内 mock 商店清单（后端未就绪时的演示态，不落库、刷新即重置）。 */
let mockStores: Store[] = [];
let mockStoreSeq = 1;

export const storeApi = {
  /** 全量商店（含 disabled），sort 升序。 */
  list(): Promise<Store[]> {
    return withFallback(
      () => request<Store[]>('/stores'),
      () => mockStores.slice().sort((a, b) => a.sort - b.sort),
    );
  },
  create(input: StoreInput): Promise<Store> {
    return withFallback(
      () =>
        request<Store>('/stores', {
          method: 'POST',
          body: JSON.stringify({ code: input.code.trim(), name: input.name.trim(), sort: input.sort ?? 0 }),
        }),
      () => {
        const s: Store = {
          id: mockStoreSeq++,
          code: input.code.trim(),
          name: input.name.trim(),
          sort: input.sort ?? mockStores.length,
          status: 'enabled',
        };
        mockStores = [...mockStores, s];
        return s;
      },
    );
  },
  update(id: number, input: StoreUpdateInput): Promise<Store> {
    return withFallback(
      () =>
        request<Store>(`/stores/${id}`, {
          method: 'PUT',
          body: JSON.stringify(input),
        }),
      () => {
        const idx = mockStores.findIndex((s) => s.id === id);
        if (idx < 0) throw new ApiError('商店不存在', 404);
        mockStores[idx] = { ...mockStores[idx], ...input };
        return { ...mockStores[idx] };
      },
    );
  },
  remove(id: number): Promise<void> {
    return withFallback(
      async () => {
        await request<{ deleted: boolean }>(`/stores/${id}`, { method: 'DELETE' });
      },
      () => {
        mockStores = mockStores.filter((s) => s.id !== id);
      },
    );
  },
};

// =========================================================================
// RBAC：权限点清单 + 角色 + 用户（10-rbac.md，唯一契约文档）
// =========================================================================

/** 后端 role 视图（数字 id）。 */
interface RoleDTO {
  id: number;
  name: string;
  description: string;
  builtin: boolean;
  permCodes: string[];
  userCount: number;
  /**
   * 数据权限（10-rbac.md「数据权限」节）在 /api/roles 上是**平铺字段**，
   * 与登录/刷新/me 响应体里嵌套的 `scope: {allBrands,...}` 形状不同——后端两处契约本就不一致，
   * 这里按 10-rbac.md「API」一节的角色端点契约逐字对齐；builtin 角色恒等价于 FULL_ROLE_SCOPE。
   */
  scopeAllBrands?: boolean;
  brandCodes?: string[];
  scopeAllChannels?: boolean;
  channelIds?: number[];
}
/** 角色端点的平铺数据范围字段 → UI 的 RoleScope（字段缺失时按「全部」兜底，同 adaptScope）。 */
function adaptRoleScope(r: RoleDTO): RoleScope {
  if (r.scopeAllBrands === undefined && r.scopeAllChannels === undefined) return FULL_ROLE_SCOPE;
  return {
    allBrands: !!r.scopeAllBrands,
    brands: (r.brandCodes ?? []) as BrandCode[],
    allChannels: !!r.scopeAllChannels,
    channelIds: (r.channelIds ?? []).map(String),
  };
}
/** UI 的 RoleScope → 角色端点的平铺入参（POST/PUT 整体替换语义）。 */
function roleScopeToPayload(scope: RoleScope) {
  return {
    scopeAllBrands: scope.allBrands,
    brandCodes: scope.allBrands ? [] : scope.brands,
    scopeAllChannels: scope.allChannels,
    channelIds: scope.allChannels ? [] : scope.channelIds.map((id) => Number(id)),
  };
}
function adaptRole(r: RoleDTO): Role {
  return {
    id: String(r.id),
    name: r.name,
    description: r.description ?? '',
    builtin: r.builtin,
    permCodes: r.permCodes ?? [],
    userCount: r.userCount ?? 0,
    scope: r.builtin ? FULL_ROLE_SCOPE : adaptRoleScope(r),
  };
}

/** 后端 admin_user 视图（数字 id/roleId）。 */
interface AdminUserDTO {
  id: number;
  username: string;
  roleId: number;
  roleName: string;
  builtinRole: boolean;
  createdAt: string;
}
function adaptAdminUser(u: AdminUserDTO): AdminUser {
  return {
    id: String(u.id),
    username: u.username,
    roleId: String(u.roleId),
    roleName: u.roleName,
    builtinRole: u.builtinRole,
    createdAt: u.createdAt,
  };
}

/** 权限点清单：登录即可拉取，渲染角色管理的勾选树。 */
export const permsApi = {
  catalog(): Promise<PermCatalogModule[]> {
    return withFallback(
      () => request<PermCatalogModule[]>('/perms/catalog'),
      () => mockRbacDb.catalog(),
    );
  },
};

/**
 * 角色 CRUD（需 role:manage）。列表读走 withFallback（后端 RBAC 路由未就绪时回退演示数据）；
 * 写操作一律走 realOnly（见其注释）——真实模式下不兜底，避免拿 mock 的 id 打真后端。
 */
export const rolesApi = {
  list(): Promise<Role[]> {
    return withFallback(
      async () => (await request<RoleDTO[]>('/roles')).map(adaptRole),
      () => mockRbacDb.listRoles(),
    );
  },
  create(input: RoleInput): Promise<Role> {
    return realOnly(
      async () =>
        adaptRole(
          await request<RoleDTO>('/roles', {
            method: 'POST',
            // 数据范围走平铺字段（见 roleScopeToPayload）——嵌套成 {scope:{...}} 后端不认，
            // 会被静默丢弃成「零品牌零渠道」，角色看起来配了全部品牌实则什么都看不到。
            body: JSON.stringify({
              name: input.name,
              description: input.description,
              permCodes: input.permCodes,
              ...roleScopeToPayload(input.scope),
            }),
          }),
        ),
      () => mockRbacDb.createRole(input),
    );
  },
  update(id: string, input: RoleInput): Promise<Role> {
    return realOnly(
      async () =>
        adaptRole(
          await request<RoleDTO>(`/roles/${id}`, {
            method: 'PUT',
            body: JSON.stringify({
              name: input.name,
              description: input.description,
              permCodes: input.permCodes,
              ...roleScopeToPayload(input.scope),
            }),
          }),
        ),
      () => mockRbacDb.updateRole(id, input),
    );
  },
  remove(id: string): Promise<void> {
    return realOnly(
      async () => {
        await request(`/roles/${id}`, { method: 'DELETE' });
      },
      () => mockRbacDb.removeRole(id),
    );
  },
};

/**
 * 后台用户 CRUD + 重置密码（需 user:manage）。列表读走 withFallback；
 * 写操作一律走 realOnly（见其注释）——真实模式下不兜底，避免拿 mock 的 id 打真后端。
 */
export const usersApi = {
  list(): Promise<AdminUser[]> {
    return withFallback(
      async () => (await request<AdminUserDTO[]>('/users')).map(adaptAdminUser),
      () => mockRbacDb.listUsers(),
    );
  },
  create(input: AdminUserInput): Promise<AdminUser> {
    return realOnly(
      async () =>
        adaptAdminUser(
          await request<AdminUserDTO>('/users', {
            method: 'POST',
            body: JSON.stringify({ username: input.username.trim(), password: input.password, roleId: Number(input.roleId) || input.roleId }),
          }),
        ),
      () => mockRbacDb.createUser(input),
    );
  },
  updateRole(id: string, roleId: string): Promise<AdminUser> {
    return realOnly(
      async () =>
        adaptAdminUser(
          await request<AdminUserDTO>(`/users/${id}`, {
            method: 'PUT',
            body: JSON.stringify({ roleId: Number(roleId) || roleId }),
          }),
        ),
      () => mockRbacDb.updateUserRole(id, roleId),
    );
  },
  resetPassword(id: string, password: string): Promise<void> {
    return realOnly(
      async () => {
        await request(`/users/${id}/reset-password`, { method: 'POST', body: JSON.stringify({ password }) });
      },
      () => mockRbacDb.resetPassword(id, password),
    );
  },
  remove(id: string): Promise<void> {
    return realOnly(
      async () => {
        await request(`/users/${id}`, { method: 'DELETE' });
      },
      // mock 层用当前登录用户名判断「不能删除自己」（真实后端从 JWT 里取，前端仅本地演示用）。
      () => {
        let currentUsername: string | null = null;
        try {
          const raw = localStorage.getItem('hybrid:auth');
          if (raw) currentUsername = (JSON.parse(raw)?.state?.user?.username as string | undefined) ?? null;
        } catch {
          /* ignore */
        }
        mockRbacDb.removeUser(id, currentUsername);
      },
    );
  },
};

/**
 * 渠道副作用：域名覆盖 + 图标 + splash。后端把它们拆成独立端点，这里串起来。
 * 每步独立容错：单步失败仅 console 警告，不让整次保存回滚（基本信息已落库）。
 */
async function applyChannelSideEffects(id: string, input: ChannelInput): Promise<void> {
  // 域名：继承 → inheritBrand=true；覆盖 → 提交渠道级清单。
  try {
    if (input.useBrandDomains) {
      await request(`/channels/${id}/domains`, {
        method: 'PUT',
        body: JSON.stringify({ inheritBrand: true, domains: [] }),
      });
    } else if (input.domains && input.domains.some((d) => d.url.trim())) {
      await request(`/channels/${id}/domains`, {
        method: 'PUT',
        body: JSON.stringify({ inheritBrand: false, domains: domainsToInput(input.domains) }),
      });
    }
  } catch (err) {
    if (import.meta.env.DEV) console.warn('[api] 设置渠道域名失败:', (err as Error)?.message);
  }
  // 图标主图：dataURL → multipart 上传（后端 fan-out 全密度）。
  if (input.iconMasterDataUrl?.startsWith('data:')) {
    try {
      await uploadDataUrl(`/channels/${id}/icon`, input.iconMasterDataUrl, 'icon.png');
    } catch (err) {
      if (import.meta.env.DEV) console.warn('[api] 上传图标失败:', (err as Error)?.message);
    }
  }
  // splash 源图。
  if (input.splashDataUrl?.startsWith('data:')) {
    try {
      await uploadDataUrl(`/channels/${id}/splash`, input.splashDataUrl, 'splash.png');
    } catch (err) {
      if (import.meta.env.DEV) console.warn('[api] 上传 splash 失败:', (err as Error)?.message);
    }
  }
}

/** dataURL → Blob → multipart POST（字段名 file，与后端 openUpload 一致）。 */
async function uploadDataUrl(path: string, dataUrl: string, filename: string): Promise<void> {
  const blob = await (await fetch(dataUrl)).blob();
  const fd = new FormData();
  fd.append('file', blob, filename);
  const token = getToken();
  const res = await fetch(`/api${path}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: fd,
  });
  if (!res.ok) throw new ApiError(`HTTP ${res.status}`, res.status);
}

// =========================================================================
// 打包编排 + 构建记录（ADR-0008）
// =========================================================================
export const buildApi = {
  /** 构建任务列表（构建记录页）。 */
  listJobs(brand?: BrandCode): Promise<BuildJob[]> {
    const q = brand ? `?brand=${brand}` : '';
    return withFallback(
      async () => {
        // 优先新端点 /build/jobs；后端未上线时（404）回退兼容旧 /build/records。
        try {
          return await request<BuildJob[]>(`/build/jobs${q}`);
        } catch (err) {
          if (err instanceof ApiError && err.status === 404) {
            const recs = await request<BuildRecordDTO[]>(`/build/records${q}`);
            return recs.map(adaptRecordToJob);
          }
          throw err;
        }
      },
      () => mockDb.listBuildJobs(brand),
    );
  },

  /** 触发打包任务（ADR-0008：渠道 + versionName + 任务名 + 测试事件）。 */
  submitJob(req: BuildJobRequest): Promise<BuildJob> {
    return withFallback(
      async () => {
        // 适配后端 service.CreateBuildJobInput 的字段名：brand（非 brandCode）/ name（非 jobName）。
        const rec = await request<BuildRecordDTO>('/build/jobs', {
          method: 'POST',
          body: JSON.stringify({
            brand: req.brandCode,
            flavors: req.flavors,
            versionName: req.versionName,
            name: req.jobName,
            testEvents: req.testEvents,
          }),
        });
        return adaptRecordToJob(rec);
      },
      () => mockDb.createBuildJob(req),
    );
  },

  /**
   * 增量拉取任务日志（轮询 logs 流；done=true 时停止）。
   * 对齐后端真实端点 GET /api/build/records/:id/logs?offset=（返回 BuildLogSegment{offset,next,log,done}，
   * 不含 status）。done 时再取一次记录拿到真实终态 success/failed（评审 W1）。
   */
  getJobLogs(jobId: string, after = 0): Promise<BuildLogChunk> {
    return withFallback(
      async () => {
        const seg = await request<{ offset: number; next: number; log: string; done: boolean }>(
          `/build/records/${jobId}/logs?offset=${after}`,
        );
        let status: BuildLogChunk['status'] = seg.done ? 'success' : 'running';
        if (seg.done) {
          // 段接口不含状态：done 后取记录确定 success/failed。
          try {
            const rec = await request<BuildRecordDTO>(`/build/records/${jobId}`);
            status = (rec.status as BuildLogChunk['status']) || 'success';
          } catch {
            /* 取记录失败则按 success 兜底，不影响日志展示 */
          }
        }
        const lines = seg.log ? seg.log.replace(/\n+$/, '').split('\n') : [];
        return { cursor: seg.next, lines, done: seg.done, status };
      },
      () => mockDb.jobLogs(jobId, after),
    );
  },
};

/** 旧 build_record 形态（兼容回退用）。 */
interface BuildRecordDTO {
  id: number;
  name?: string; // 任务名（后端默认 <brand>-<versionName>-<时间>，可被前端覆盖）
  brandCode: BrandCode;
  flavors: string; // JSON 字符串
  testEvents: boolean;
  status: 'queued' | 'running' | 'success' | 'failed';
  operator?: string;
  versionName?: string;
  apkUrls?: string; // JSON 字符串
  logExcerpt?: string;
  startedAt: string;
  finishedAt?: string;
}
function adaptRecordToJob(r: BuildRecordDTO): BuildJob {
  const flavors = safeParseArray(r.flavors);
  const apkUrls = safeParseArray(r.apkUrls);
  const artifacts: BuildArtifact[] = apkUrls.map((url, i) => ({
    flavor: flavors[i] ?? flavors[0] ?? '',
    versionName: r.versionName ?? '',
    apkUrl: url,
  }));
  return {
    id: String(r.id),
    jobName: r.name || `${r.brandCode}-${r.versionName ?? ''}`,
    brandCode: r.brandCode,
    flavors,
    versionName: r.versionName ?? '',
    testEvents: r.testEvents,
    status: r.status,
    operator: r.operator,
    artifacts,
    logExcerpt: r.logExcerpt,
    startedAt: r.startedAt,
    finishedAt: r.finishedAt,
  };
}
function safeParseArray(s: string | undefined): string[] {
  if (!s) return [];
  try {
    const v = JSON.parse(s);
    return Array.isArray(v) ? v : [];
  } catch {
    return [];
  }
}

/** 给 UI 用的品牌色（即便后端返回也用本地 meta 兜底色）。 */
export function brandAccent(code: BrandCode): string {
  return BRAND_META[code].accentColor;
}

// =========================================================================
// 推送管理（07-push.md）
// =========================================================================
export const pushApi = {
  /** 功能门控：读 FCM 配置状态（feature gate）。 */
  getStatus(): Promise<PushStatus> {
    return withFallback(
      () => request<PushStatus>('/push/status'),
      () => mockDb.getPushStatus(),
    );
  },

  /** 推送活动列表（按品牌过滤）。 */
  listCampaigns(brand?: BrandCode): Promise<PushCampaign[]> {
    const q = brand ? `?brand=${brand}` : '';
    return withFallback(
      () => request<PushCampaign[]>(`/push/campaigns${q}`),
      () => mockDb.listPushCampaigns(brand),
    );
  },

  /** 推送活动详情 + push_record 列表。 */
  getCampaign(id: string): Promise<PushCampaign & { records: import('./types').PushRecord[] }> {
    return withFallback(
      () => request<PushCampaign & { records: import('./types').PushRecord[] }>(`/push/campaigns/${id}`),
      () => {
        const c = mockDb.getPushCampaign(id);
        if (!c) throw new Error(`Campaign ${id} not found`);
        return c;
      },
    );
  },

  /** 创建草稿活动。 */
  createCampaign(input: PushCampaignInput): Promise<PushCampaign> {
    return withFallback(
      () => request<PushCampaign>('/push/campaigns', { method: 'POST', body: JSON.stringify(input) }),
      () => mockDb.createPushCampaign(input),
    );
  },

  /** 更新草稿活动。 */
  updateCampaign(id: string, input: PushCampaignInput): Promise<PushCampaign> {
    return withFallback(
      () => request<PushCampaign>(`/push/campaigns/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
      () => mockDb.updatePushCampaign(id, input),
    );
  },

  /** 立即发送（dryRun=true 时不实际发送，仅预演）。返回新响应结构 PushSendResult。 */
  sendCampaign(id: string, dryRun = false): Promise<PushSendResult> {
    return withFallback(
      () => request<PushSendResult>(`/push/campaigns/${id}/send`, { method: 'POST', body: JSON.stringify({ dryRun }) }),
      () => mockDb.sendPushCampaign(id, dryRun),
    );
  },

  /** 定时发送。 */
  scheduleCampaign(id: string, scheduledAt: string): Promise<PushCampaign> {
    return withFallback(
      () =>
        request<PushCampaign>(`/push/campaigns/${id}/schedule`, {
          method: 'POST',
          body: JSON.stringify({ scheduledAt }),
        }),
      () => mockDb.schedulePushCampaign(id, scheduledAt),
    );
  },

  /** 上传推送封面图（multipart）→ { url }。 */
  async uploadImage(dataUrl: string): Promise<string> {
    if (FORCE_MOCK) {
      // mock：原样回返 dataUrl，演示用
      return delay(dataUrl);
    }
    try {
      const blob = await (await fetch(dataUrl)).blob();
      const fd = new FormData();
      fd.append('file', blob, 'push-cover.png');
      const token = getToken();
      const res = await fetch('/api/push/upload-image', {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : undefined,
        body: fd,
      });
      if (!res.ok) throw new ApiError(`HTTP ${res.status}`, res.status);
      const json = (await res.json()) as ApiEnvelope<{ url: string }>;
      return json.data.url;
    } catch (err) {
      if (!shouldFallback(err)) throw err;
      return delay(dataUrl);
    }
  },

  /** 触达预估（发送前展示目标设备数）。 */
  getAudience(appIds: string[]): Promise<PushAudience> {
    const q = appIds.map((id) => encodeURIComponent(id)).join(',');
    return withFallback(
      () => request<PushAudience>(`/push/audience?appIds=${q}`),
      () => mockDb.getPushAudience(appIds),
    );
  },
};

// ---------- 辅助：ChannelInput → Channel（mock 落库用） ----------
function inputToChannel(input: ChannelInput): Channel {
  const id = `${input.brandCode}-${input.flavorName}`;
  return {
    id,
    brandCode: input.brandCode,
    flavorName: input.flavorName.trim(),
    applicationId: input.applicationId.trim(),
    palCode: input.palCode.trim(),
    appName: input.appName.trim(),
    status: input.status ?? 'enabled',
    useBrandDomains: input.useBrandDomains,
    domains: input.useBrandDomains ? undefined : input.domains,
    iconMasterUrl: input.iconMasterDataUrl,
    splashUrl: input.splashDataUrl,
    remark: input.remark,
    liveVersion: input.liveVersion?.trim() || undefined,
    storeId: input.storeId ?? null,
    adjustAppToken: input.adjustAppToken?.trim() || undefined,
    adjustEvents: input.adjustEvents && Object.keys(input.adjustEvents).length ? input.adjustEvents : undefined,
  };
}
