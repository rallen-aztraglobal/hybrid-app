import type { Listing, ListingGateLog, ListingGateTestResult, ListingInput } from '../types';

/**
 * 进程内 mock 上架包数据（后端未就绪时的演示态，不落库、刷新即重置）。
 * 种子数据对齐 09-listing.md：ColorStack（Android+iOS 同包名）+ DeckTallyPro（iOS）。
 * 与 lib/mock/db.ts 保持同一套约定：函数式、深拷贝返回、失败抛普通 Error（由 withFallback 消化）。
 */

let listings: Listing[] = [
  {
    id: 'lst-1',
    brandCode: 'gp',
    platform: 'android',
    bundleId: 'com.vividnest.colorstack5821',
    name: 'ColorStack',
    displayName: 'ColorStack - Color Puzzle',
    palCode: '1053259',
    storeUrl: 'https://play.google.com/store/apps/details?id=com.vividnest.colorstack5821',
    status: 'enabled',
    useBrandDomains: true,
    gateEnabled: true,
    afDevKey: 'af-dev-key-demo',
    afAppId: 'com.vividnest.colorstack5821',
    adjustAppToken: 'csk_demo_token',
    adjustEvents: { install: 'abc123', level_complete: 'def456' },
    remark: '',
    gate: { countries: ['PH', 'ID'], timezoneDeny: ['America/New_York'], ipAllowCidrs: [], ipDenyCidrs: [], updatedAt: '2026-06-01T02:00:00Z' },
    createdAt: '2026-05-01T02:00:00Z',
    updatedAt: '2026-06-01T02:00:00Z',
  },
  {
    id: 'lst-2',
    brandCode: 'gp',
    platform: 'ios',
    bundleId: 'com.vividnest.colorstack5821',
    name: 'ColorStack',
    displayName: 'ColorStack - Color Puzzle',
    palCode: '1053259',
    storeUrl: 'https://apps.apple.com/app/id0000000000',
    status: 'enabled',
    useBrandDomains: true,
    gateEnabled: false,
    afDevKey: 'af-dev-key-demo',
    afAppId: 'id0000000000',
    remark: '',
    gate: { countries: [], timezoneDeny: [], ipAllowCidrs: [], ipDenyCidrs: [] },
    createdAt: '2026-05-01T02:00:00Z',
    updatedAt: '2026-05-20T02:00:00Z',
  },
  {
    id: 'lst-3',
    brandCode: 'ap',
    platform: 'ios',
    bundleId: 'com.deck.tallypro',
    name: 'DeckTallyPro',
    displayName: 'Deck Tally Pro',
    palCode: '2087431',
    storeUrl: 'https://apps.apple.com/app/id0000000001',
    status: 'enabled',
    useBrandDomains: true,
    gateEnabled: false,
    afDevKey: '',
    afAppId: '',
    remark: '合规审核期仅展示 A 面，规则待运营配置完成后开启',
    gate: { countries: ['PH'], timezoneDeny: [], ipAllowCidrs: [], ipDenyCidrs: [] },
    createdAt: '2026-05-10T02:00:00Z',
    updatedAt: '2026-05-10T02:00:00Z',
  },
];
let seq = 3;

const gateLogsStore: Record<string, ListingGateLog[]> = {
  'lst-1': [
    { id: 'g1', ip: '203.0.113.10', country: 'PH', timezone: 'Asia/Manila', decision: 'B', reason: '国家 PH 命中全部放行条件', createdAt: '2026-07-19T10:00:00Z' },
    { id: 'g2', ip: '8.8.8.8', country: 'US', timezone: 'America/New_York', decision: 'A', reason: '国家 US 被硬编码强制 A 面', createdAt: '2026-07-19T09:50:00Z' },
    { id: 'g3', ip: '198.51.100.4', country: 'JP', timezone: 'Asia/Tokyo', decision: 'A', reason: '国家 JP 不在白名单内', createdAt: '2026-07-19T09:40:00Z' },
    { id: 'g4', ip: '203.0.113.55', country: 'PH', timezone: 'America/New_York', decision: 'A', reason: '命中时区黑名单 America/New_York', createdAt: '2026-07-19T09:30:00Z' },
  ],
};

/** 归一化网关草稿：国家去重转大写去空，其余去空白去空项，与真实后端 SetListingGate 前置处理保持一致口径。 */
function normalizeGateInput(input: ListingInput) {
  const countries = Array.from(new Set(input.gate.countries.map((c) => c.trim().toUpperCase()).filter(Boolean)));
  return {
    countries,
    timezoneDeny: input.gate.timezoneDeny.map((t) => t.trim()).filter(Boolean),
    ipAllowCidrs: input.gate.ipAllowCidrs.map((c) => c.trim()).filter(Boolean),
    ipDenyCidrs: input.gate.ipDenyCidrs.map((c) => c.trim()).filter(Boolean),
  };
}

/** 域名 + 网关规则 + 总开关副作用，顺序与真实后端一致：域名 → 网关规则 → 开关（开关校验国家白名单）。 */
function applySideEffects(id: string, input: ListingInput): void {
  const idx = listings.findIndex((l) => l.id === id);
  if (idx < 0) return;
  listings[idx] = {
    ...listings[idx],
    useBrandDomains: input.useBrandDomains,
    domains: input.useBrandDomains ? undefined : (input.domains ?? []).filter((d) => d.url.trim()),
  };
  const gate = normalizeGateInput(input);
  if (gate.countries.length > 0) {
    listings[idx] = { ...listings[idx], gate: { ...gate, updatedAt: new Date().toISOString() } };
  }
  if (input.gateEnabled && (listings[idx].gate?.countries.length ?? 0) === 0) {
    throw new Error('打开 AB 面前必须先配置国家白名单（且不能为空）');
  }
  listings[idx] = { ...listings[idx], gateEnabled: input.gateEnabled };
}

export const mockListingDb = {
  list(): Listing[] {
    return listings.filter((l) => l.status !== 'archived').map((l) => ({ ...l }));
  },

  get(id: string): Listing | undefined {
    const l = listings.find((x) => x.id === id);
    return l ? { ...l } : undefined;
  },

  create(input: ListingInput): Listing {
    if (listings.some((l) => l.platform === input.platform && l.bundleId.trim() === input.bundleId.trim())) {
      throw new Error(`${input.platform} 平台已存在包名 ${input.bundleId}`);
    }
    seq += 1;
    const id = `lst-${seq}`;
    const created: Listing = {
      id,
      brandCode: input.brandCode,
      platform: input.platform,
      bundleId: input.bundleId.trim(),
      name: input.name.trim(),
      displayName: input.displayName.trim(),
      palCode: input.palCode.trim() || undefined,
      storeUrl: input.storeUrl.trim(),
      status: 'enabled',
      useBrandDomains: true,
      gateEnabled: false,
      afDevKey: input.afDevKey.trim(),
      afAppId: input.afAppId.trim(),
      adjustAppToken: input.adjustAppToken?.trim() || undefined,
      adjustEvents: input.adjustEvents && Object.keys(input.adjustEvents).length ? input.adjustEvents : undefined,
      remark: input.remark.trim(),
      gate: { countries: [], timezoneDeny: [], ipAllowCidrs: [], ipDenyCidrs: [] },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    listings = [created, ...listings];
    applySideEffects(id, input);
    return { ...listings.find((l) => l.id === id)! };
  },

  update(id: string, input: ListingInput): Listing {
    const idx = listings.findIndex((l) => l.id === id);
    if (idx < 0) throw new Error('上架包不存在');
    listings[idx] = {
      ...listings[idx],
      brandCode: input.brandCode,
      name: input.name.trim(),
      displayName: input.displayName.trim(),
      palCode: input.palCode.trim() || undefined,
      storeUrl: input.storeUrl.trim(),
      status: input.status,
      afDevKey: input.afDevKey.trim(),
      afAppId: input.afAppId.trim(),
      adjustAppToken: input.adjustAppToken?.trim() || undefined,
      adjustEvents: input.adjustEvents && Object.keys(input.adjustEvents).length ? input.adjustEvents : undefined,
      remark: input.remark.trim(),
      updatedAt: new Date().toISOString(),
    };
    applySideEffects(id, input);
    return { ...listings.find((l) => l.id === id)! };
  },

  remove(id: string): void {
    listings = listings.filter((l) => l.id !== id);
    delete gateLogsStore[id];
  },

  /** 用给定 IP/时区跑一次判定（演示逻辑，非真实 GeoIP）：已配置国家的第一个视为「命中」演示样本。 */
  testGate(id: string, ip: string, timezone: string): ListingGateTestResult {
    const l = listings.find((x) => x.id === id);
    if (!l) throw new Error('上架包不存在');
    if (!l.gateEnabled) return { mode: 'A', reason: 'AB 面总开关未开启', country: '' };
    const trimmedIp = ip.trim();
    if (/^(1\.1\.1\.1|8\.8\.8\.8)$/.test(trimmedIp)) {
      return { mode: 'A', reason: '国家 US 被硬编码强制 A 面（演示：该 IP 模拟解析为 US）', country: 'US' };
    }
    const countries = l.gate?.countries ?? [];
    if (countries.length === 0) return { mode: 'A', reason: '国家白名单为空（配置无效）', country: '' };
    const demoCountry = countries[0];
    const tz = timezone.trim();
    // 时区黑名单：命中即强制 A（与 IP 黑名单同类；空上报或空清单不拦）。
    if (tz && (l.gate?.timezoneDeny ?? []).includes(tz)) {
      return { mode: 'A', reason: `命中时区黑名单 ${tz}`, country: demoCountry };
    }
    return { mode: 'B', reason: `国家 ${demoCountry} 命中全部放行条件（演示）`, country: demoCountry };
  },

  gateLogs(id: string, limit: number): ListingGateLog[] {
    return (gateLogsStore[id] ?? []).slice(0, limit > 0 ? limit : 50);
  },
};
