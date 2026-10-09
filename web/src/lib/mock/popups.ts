import type {
  BrandCode,
  Popup,
  PopupInput,
  PopupListFilter,
  PopupPosition,
  PopupPositionCode,
  PopupRuntimePayload,
  PopupStatRow,
  PopupStatsQuery,
  PopupStatus,
} from '../types';
import { BRAND_META } from '../brands';
import { POPUP_POSITION_CODES, POSITION_CAPS, normalizeBehavior, versionToCode } from '../popupMeta';

/**
 * 进程内 mock 弹窗模块（12-popup.md）。后端未就绪（404 / 5xx / 网络不通）时由 api.ts 的
 * withFallback 回退到这里，保证前端可独立 `pnpm dev` 演示完整闭环。
 * 约定同其它 mock：函数式、深拷贝返回、校验失败抛 Error。
 */

/** 生成渐变 SVG 海报（data URL），当作「上传后的图片」。 */
export function posterDataUrl(w: number, h: number, c1: string, c2: string, title: string, sub = ''): string {
  const fs = Math.round(Math.min(w, h) * 0.11);
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">` +
    `<defs><linearGradient id="g" x1="0" y1="0" x2="0.6" y2="1"><stop offset="0" stop-color="${c1}"/><stop offset="1" stop-color="${c2}"/></linearGradient></defs>` +
    `<rect width="${w}" height="${h}" fill="url(#g)"/>` +
    `<circle cx="${w * 0.72}" cy="${h * 0.22}" r="${Math.min(w, h) * 0.34}" fill="rgba(255,255,255,.22)"/>` +
    `<text x="${w * 0.08}" y="${h * 0.78}" font-family="sans-serif" font-weight="900" font-size="${fs}" fill="#fff">${title}</text>` +
    `<text x="${w * 0.08}" y="${h * 0.78 + fs * 1.3}" font-family="sans-serif" font-weight="600" font-size="${Math.round(fs * 0.5)}" fill="rgba(255,255,255,.92)">${sub}</text>` +
    `</svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

const card34 = (c1: string, c2: string, t: string, s?: string) => posterDataUrl(600, 800, c1, c2, t, s);
const full916 = (c1: string, c2: string, t: string, s?: string) => posterDataUrl(720, 1280, c1, c2, t, s);
const sq = (c1: string, c2: string, t: string, n = 120) => posterDataUrl(n, n, c1, c2, t);

const NOW = () => Date.now();

function mkCard(id: number, sort: number, imageUrl: string, linkUrl = '', buttonText = '', title = '', description = '') {
  return { id, sort, imageUrl, linkUrl, buttonText, title, description };
}

let seq = 20;
let cardSeq = 200;

let positions: PopupPosition[] = POPUP_POSITION_CODES.map((code) => ({
  code,
  name: POSITION_CAPS[code].name,
  enabled: POSITION_CAPS[code].defaultPositionOn,
  updatedAt: '2026-10-08T18:22:00+08:00',
  updatedBy: 'Daly',
}));

type Stored = Omit<Popup, 'status' | 'positionEnabled'> & { deleted?: boolean };

const base = (over: Partial<Stored> & Pick<Stored, 'id' | 'name' | 'position' | 'cards'>): Stored => ({
  enabled: true,
  priority: 50,
  startAt: null,
  endAt: null,
  brandCodes: [],
  appIds: [],
  minVersion: '',
  maxVersion: '',
  userType: 'all',
  countries: [],
  tabEnabled: false,
  tabIconUrl: '',
  tabText: '',
  countdown: false,
  maskClosable: true,
  closable: true,
  openMode: 'webview',
  autoplaySeconds: 5,
  resumeGapMinutes: 30,
  badge: false,
  createdBy: 'Daly',
  createdAt: '2026-09-20T10:00:00+08:00',
  updatedAt: '2026-10-08T18:22:00+08:00',
  ...over,
});

let store: Stored[] = [
  base({
    id: 1, name: '中秋活动 · 月饼礼金', position: 'P1', priority: 90, startAt: '2026-09-28T00:00:00+08:00', endAt: '2026-10-12T23:59:00+08:00',
    brandCodes: ['gp'], minVersion: '1.0.3', countries: ['PH'], tabEnabled: true, tabText: '中秋礼金',
    cards: [
      mkCard(11, 0, card34('#1e1b4b', '#b45309', 'Mid-Autumn', 'Daily bonus'), '/promo/midautumn', 'Join now'),
      mkCard(12, 1, card34('#ff8a00', '#8b2cf5', 'First Deposit', '100% bonus'), '/promo/deposit', 'Join now'),
      mkCard(13, 2, card34('#0ea5e9', '#a855f7', 'Lucky Spin', '3 free spins'), '/promo/spin', 'Join now'),
    ],
  }),
  base({
    id: 2, name: '首存 100% 加赠', position: 'P1', priority: 80, startAt: '2026-10-01T00:00:00+08:00', endAt: '2026-10-31T23:59:00+08:00',
    brandCodes: ['ap', 'bp'], minVersion: '1.0.0', userType: 'new', countries: ['PH'], tabEnabled: true, tabText: '首存加赠', countdown: true,
    cards: [mkCard(21, 0, card34('#ff8a00', '#ff3d6e', 'Deposit 100%', 'Up to P5,000'), '/promo/first-deposit', 'Deposit now')],
  }),
  base({
    id: 3, name: '在线客服入口', position: 'P2', priority: 50, startAt: '2026-09-01T00:00:00+08:00', endAt: '2026-12-31T23:59:00+08:00',
    brandCodes: ['ap', 'bp', 'gp'], badge: true, maskClosable: false,
    cards: [mkCard(31, 0, sq('#06b6d4', '#3b82f6', 'Help'), '/service')],
  }),
  base({
    id: 4, name: '周末返水 10% 横幅', position: 'P3', priority: 40, startAt: '2026-10-11T00:00:00+08:00', endAt: '2026-10-26T23:59:00+08:00',
    brandCodes: ['bp'], minVersion: '1.0.5', userType: 'old', countries: ['PH'], tabEnabled: true, tabText: '周末返水', maskClosable: false, autoplaySeconds: 5,
    cards: [
      mkCard(41, 0, sq('#047857', '#bef264', 'Rebate', 80), '/promo/rebate', '', 'Weekend rebate 10% · auto credited Monday'),
      mkCard(42, 1, sq('#0f172a', '#22d3ee', 'Cup', 80), '/promo/cup', '', 'Tongits Cup · P1,000,000 prize pool'),
    ],
  }),
  base({
    id: 5, name: '强制换包 · 旧版停用通知', position: 'P4', enabled: false, priority: 100, startAt: '2026-10-09T10:00:00+08:00', endAt: '2026-11-30T23:59:00+08:00',
    brandCodes: ['ap'], maxVersion: '1.0.5', closable: false, maskClosable: false, openMode: 'store',
    cards: [mkCard(51, 0, card34('#1d4ed8', '#7dd3fc', 'Please Update', 'Old version ends 11/30'), 'market://details?id=com.arenaplus.ap01159', 'Update now', 'Update available', 'Please update to continue.')],
  }),
  base({
    id: 6, name: '10/05 系统维护通知', position: 'P5', priority: 60, startAt: '2026-10-03T00:00:00+08:00', endAt: '2026-10-05T06:00:00+08:00',
    brandCodes: [], maskClosable: false,
    cards: [mkCard(61, 0, '', '', '', 'Scheduled maintenance Oct 5, 02:00–04:00 (GMT+8)')],
  }),
  base({
    id: 7, name: '退出挽留 · 回来领 ₱50', position: 'P6', priority: 30, startAt: '2026-10-15T00:00:00+08:00', endAt: '2026-10-31T23:59:00+08:00',
    brandCodes: ['gp'], minVersion: '1.0.3', userType: 'old', countries: ['PH'],
    cards: [mkCard(71, 0, card34('#be185d', '#fb923c', 'Come back', 'Claim P50'), '/promo/comeback', 'Claim P50')],
  }),
  base({
    id: 8, name: '国庆大促 · 全屏插屏', position: 'P7', priority: 95, startAt: '2026-10-01T00:00:00+08:00', endAt: '2026-10-07T23:59:00+08:00',
    brandCodes: ['ap', 'gp'], minVersion: '1.0.3', countries: ['PH'], tabEnabled: true, tabText: '国庆大促', countdown: true, maskClosable: false,
    cards: [
      mkCard(81, 0, full916('#7f1d1d', '#f59e0b', 'Golden Week', 'Big promo'), '/promo/golden-week', 'Join now'),
      mkCard(82, 1, full916('#ff8a00', '#8b2cf5', 'Deposit', 'Bonus'), '/promo/deposit', 'Join now'),
    ],
  }),
  base({
    id: 9, name: '每日签到便条', position: 'P8', priority: 20, startAt: '2026-09-15T00:00:00+08:00', endAt: '2026-12-31T23:59:00+08:00',
    brandCodes: ['ap', 'bp', 'gp'], maskClosable: false,
    cards: [mkCard(91, 0, sq('#f59e0b', '#f97316', 'Go', 40), '/checkin', '', '签到领钱')],
  }),
];

function computeStatus(p: Stored): PopupStatus {
  if (!p.enabled) return 'disabled';
  const t = NOW();
  if (p.endAt && new Date(p.endAt).getTime() < t) return 'ended';
  if (p.startAt && new Date(p.startAt).getTime() > t) return 'scheduled';
  return 'active';
}

function view(p: Stored): Popup {
  const { deleted: _d, ...rest } = p;
  void _d;
  const pos = positions.find((x) => x.code === p.position);
  return JSON.parse(JSON.stringify({ ...rest, status: computeStatus(p), positionEnabled: !!pos?.enabled })) as Popup;
}

const live = () => store.filter((p) => !p.deleted);

function validateInput(input: PopupInput): void {
  const caps = POSITION_CAPS[input.position];
  if (!caps) throw new Error('未知位置码');
  if (!input.name.trim()) throw new Error('弹窗名称必填');
  if (input.cards.length > caps.maxCards) throw new Error(`最多 ${caps.maxCards} 张`);
  if (input.cards.length === 0) throw new Error('至少需要 1 张卡片');
  if (input.startAt && input.endAt && new Date(input.endAt) <= new Date(input.startAt)) {
    throw new Error('结束时间必须晚于开始时间');
  }
}

export const mockPopupDb = {
  listPositions(): PopupPosition[] {
    return JSON.parse(JSON.stringify(positions)) as PopupPosition[];
  },

  setPosition(code: PopupPositionCode, enabled: boolean): PopupPosition {
    const p = positions.find((x) => x.code === code);
    if (!p) throw new Error('未知位置码');
    p.enabled = enabled;
    p.updatedAt = new Date().toISOString();
    p.updatedBy = 'mock';
    return { ...p };
  },

  list(filter: PopupListFilter = {}): Popup[] {
    const kw = filter.keyword?.trim().toLowerCase();
    return live()
      .map(view)
      .filter(
        (p) =>
          (!filter.position || p.position === filter.position) &&
          (!filter.brand || p.brandCodes.length === 0 || p.brandCodes.includes(filter.brand)) &&
          (!filter.status || p.status === filter.status) &&
          (!kw || p.name.toLowerCase().includes(kw) || String(p.id) === kw),
      )
      .sort((a, b) => b.id - a.id);
  },

  get(id: number): Popup {
    const p = live().find((x) => x.id === id);
    if (!p) throw new Error('弹窗不存在');
    return view(p);
  },

  save(id: number | undefined, input: PopupInput): Popup {
    validateInput(input);
    const b = normalizeBehavior(input);
    const existing = id != null ? live().find((x) => x.id === id) : undefined;
    if (id != null && !existing) throw new Error('弹窗不存在');
    // 卡片按 id 合并：带 id 的原地更新，不带 id 的新建，缺席的删除
    const cards = input.cards.map((c, i) => ({
      id: c.id && existing?.cards.some((o) => o.id === c.id) ? c.id : (cardSeq += 1),
      sort: i,
      imageUrl: c.imageUrl,
      linkUrl: c.linkUrl,
      buttonText: c.buttonText,
      title: c.title,
      description: c.description,
    }));
    const next: Stored = {
      ...(existing ?? { id: (seq += 1), createdBy: 'mock', createdAt: new Date().toISOString() }),
      ...input,
      ...b,
      cards,
      updatedAt: new Date().toISOString(),
    } as Stored;
    if (existing) store = store.map((x) => (x.id === existing.id ? next : x));
    else store = [...store, next];
    return view(next);
  },

  setEnabled(id: number, enabled: boolean): Popup {
    const p = live().find((x) => x.id === id);
    if (!p) throw new Error('弹窗不存在');
    p.enabled = enabled;
    return view(p);
  },

  remove(id: number): void {
    const p = live().find((x) => x.id === id);
    if (!p) throw new Error('弹窗不存在');
    p.deleted = true;
  },

  /** 确定性伪随机统计（按 popupId 播种），数字稳定不乱跳。 */
  stats(q: PopupStatsQuery): PopupStatRow[] {
    const days = daysBetween(q.from, q.to);
    return store
      .filter(
        (p) =>
          (!q.position || p.position === q.position) &&
          (!q.popupId || p.id === q.popupId) &&
          (!q.brand || p.brandCodes.length === 0 || p.brandCodes.includes(q.brand)) &&
          (!q.appId || brandOfAppId(q.appId) === undefined || p.brandCodes.length === 0 || p.brandCodes.includes(brandOfAppId(q.appId)!)),
      )
      .map((p) => genStat(p, days, q.from));
  },

  runtimePreview(appId: string, country?: string): PopupRuntimePayload {
    const brand = brandOfAppId(appId);
    const popups = live()
      .filter((p) => {
        const pos = positions.find((x) => x.code === p.position);
        if (!pos?.enabled || !p.enabled) return false;
        if (p.endAt && new Date(p.endAt).getTime() < NOW()) return false;
        if (p.brandCodes.length && (!brand || !p.brandCodes.includes(brand))) return false;
        if (p.appIds.length && !p.appIds.includes(appId)) return false;
        if (p.countries.length && !(country && p.countries.includes(country.toUpperCase()))) return false;
        return true;
      })
      .map((p) => ({
        id: p.id,
        position: p.position,
        priority: p.priority,
        startAt: p.startAt ? new Date(p.startAt).getTime() : 0,
        endAt: p.endAt ? new Date(p.endAt).getTime() : 0,
        minVersionCode: versionToCode(p.minVersion) ?? 0,
        maxVersionCode: versionToCode(p.maxVersion) ?? 0,
        userType: p.userType,
        openMode: p.openMode,
        closable: p.closable,
        maskClosable: p.maskClosable,
        countdown: p.countdown,
        tabEnabled: p.tabEnabled,
        tabIconUrl: p.tabIconUrl || p.cards[0]?.imageUrl || '',
        tabText: p.tabText,
        autoplaySeconds: p.autoplaySeconds,
        resumeGapMinutes: p.resumeGapMinutes,
        badge: p.badge,
        cards: p.cards.map((c) => ({
          id: c.id ?? 0,
          imageUrl: c.imageUrl,
          linkUrl: c.linkUrl,
          buttonText: c.buttonText,
          title: c.title,
          description: c.description,
        })),
      }));
    return { appId, configVersion: 'mock' + popups.length.toString(16).padStart(12, '0'), serverTime: NOW(), tzOffsetMinutes: 480, popups };
  },
};

function brandOfAppId(appId: string): BrandCode | undefined {
  return (['ap', 'bp', 'gp'] as BrandCode[]).find((b) => appId.startsWith(BRAND_META[b].packagePrefix + '.'));
}

function daysBetween(from?: string, to?: string): number {
  if (!from || !to) return 7;
  const a = Date.parse(from);
  const b = Date.parse(to);
  if (Number.isNaN(a) || Number.isNaN(b)) return 7;
  return Math.max(1, Math.round((b - a) / 86_400_000) + 1);
}

function rnd(seed: number): () => number {
  let s = seed % 2147483647;
  if (s <= 0) s += 2147483646;
  return () => (s = (s * 16807) % 2147483647) / 2147483647;
}

function genStat(p: Stored, days: number, from?: string): PopupStatRow {
  const r = rnd(p.id * 7919 + 13);
  const caps = POSITION_CAPS[p.position];
  const n = p.cards.length;
  const trigger = Math.round((2500 + r() * 4000) * days);
  const loadFails = Math.round(trigger * (p.id === 2 ? 0.097 : 0.004 + r() * 0.02));
  const showRate = p.id === 2 ? 0.56 : 0.72 + r() * 0.25;
  const displays = Math.round(trigger * showRate);
  // 轮播：后面的卡片曝光逐张衰减
  const cardImps: number[] = [];
  let cur = displays;
  for (let i = 0; i < n; i++) {
    cardImps.push(Math.round(cur));
    cur *= 0.42 + r() * 0.2;
  }
  const impressions = cardImps.reduce((a, b) => a + b, 0) + Math.round(displays * 0.1);
  const ctrBase = p.id === 2 ? 0.026 : 0.04 + r() * 0.06;
  const cardClicks = cardImps.map((im, i) => Math.round(im * ctrBase * (1 - i * 0.2)));
  const clicks = cardClicks.reduce((a, b) => a + b, 0);
  const hasClose = p.position !== 'P8' && p.closable;
  const closes = hasClose ? Math.round(displays * (0.55 + r() * 0.4)) : 0;
  const tabCapable = caps.tab !== 'no' && p.tabEnabled && p.closable;
  const collapses = tabCapable ? Math.round(closes * (0.8 + r() * 0.2)) : 0;
  const tabImpressions = tabCapable ? Math.round(collapses * 1.05) : p.position === 'P8' ? Math.round(displays) : 0;
  const tabClicks = tabCapable ? Math.round(collapses * (0.08 + r() * 0.2)) : 0;
  const tabDismisses = tabImpressions ? Math.round(tabImpressions * (0.05 + r() * 0.3)) : 0;
  const slides = n > 1 ? Math.round(displays * 0.6) : 0;
  const closeBtn = 0.55 + r() * 0.2;
  const closeMask = p.maskClosable ? 0.1 + r() * 0.1 : 0;
  const closeByMethod = {
    button: Math.round(closes * closeBtn),
    mask: Math.round(closes * closeMask),
    back: Math.max(0, closes - Math.round(closes * closeBtn) - Math.round(closes * closeMask)),
  };
  const filtered = {
    frequency: Math.round(trigger * (0.5 + r())),
    mutex: caps.mask ? Math.round(trigger * 0.2 * r()) : 0,
    targeting: Math.round(trigger * 0.5 * r()),
    time: Math.round(trigger * 0.1 * r()),
  };
  const div = (a: number, b: number) => (b > 0 ? a / b : 0);
  const impsFirst = cardImps[0] ?? 0;
  const base0 = from ? new Date(from) : new Date(Date.now() - (days - 1) * 86_400_000);
  const daily = Array.from({ length: days }, (_, i) => {
    const d = new Date(base0.getTime() + i * 86_400_000);
    const f = (0.85 + r() * 0.3) / days;
    return {
      date: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
      trigger: Math.round(trigger * f),
      displays: Math.round(displays * f),
      impressions: Math.round(impressions * f),
      clicks: Math.round(clicks * f),
    };
  });
  return {
    popupId: p.id,
    name: p.name,
    position: p.position,
    deleted: !!p.deleted,
    trigger,
    displays,
    impressions,
    clicks,
    closes,
    loadFails,
    collapses,
    tabImpressions,
    tabClicks,
    tabDismisses,
    slides,
    filtered,
    closeByMethod,
    rates: {
      showRate: div(displays, trigger),
      ctr: div(clicks, impressions),
      closeRate: div(closes, impsFirst),
      loadFailRate: div(loadFails, trigger),
      carouselDepth: div(impressions, impsFirst),
      recoveryRate: div(tabClicks, collapses),
      tabAbandonRate: div(tabDismisses, tabImpressions),
      totalCtr: div(clicks + tabClicks, impressions),
    },
    cards: p.cards.map((c, i) => ({
      cardId: c.id ?? 0,
      cardIndex: i,
      imageUrl: c.imageUrl,
      impressions: cardImps[i] ?? 0,
      clicks: cardClicks[i] ?? 0,
      ctr: div(cardClicks[i] ?? 0, cardImps[i] ?? 0),
    })),
    daily,
  };
}

/** mock 上传：读文件成 data URL 并量出真实尺寸，key 仿 `popup/images/<hash>-<ms>.<ext>`。 */
export async function mockUploadPopupImage(file: File) {
  const url = await new Promise<string>((res, rej) => {
    const fr = new FileReader();
    fr.onload = () => res(fr.result as string);
    fr.onerror = () => rej(new Error('读取文件失败'));
    fr.readAsDataURL(file);
  });
  const { width, height } = await new Promise<{ width: number; height: number }>((res) => {
    const img = new Image();
    img.onload = () => res({ width: img.naturalWidth, height: img.naturalHeight });
    img.onerror = () => res({ width: 0, height: 0 });
    img.src = url;
  });
  const ext = (file.name.split('.').pop() || 'png').toLowerCase();
  const hash = Array.from(url.slice(-64))
    .reduce((h, ch) => (h * 31 + ch.charCodeAt(0)) >>> 0, 7)
    .toString(16)
    .padStart(8, '0')
    .slice(0, 12);
  return { url, key: `popup/images/${hash}-${Date.now()}.${ext}`, width, height, size: file.size };
}
