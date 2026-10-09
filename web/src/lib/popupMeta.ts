/**
 * 弹窗模块的静态元数据与纯逻辑（docs/admin/12-popup.md）：
 *  - §1 位置能力矩阵（字段显隐 / 默认值 / 归一化依据）
 *  - §1.1 素材建议尺寸（二倍图）与上传后比对
 *  - §2 版本号 X.Y.Z ↔ versionCode 换算与校验
 *  - §7.3 频控表（写死在客户端，后台只读展示）
 * 全部为无副作用纯函数，便于单测；UI 层只消费这里的结论。
 */
import type { PopupPositionCode } from './types';

export const POPUP_POSITION_CODES: PopupPositionCode[] = ['P1', 'P2', 'P3', 'P4', 'P5', 'P6', 'P7', 'P8'];

/** 卡片数上限（P1/P3/P7 轮播最多 5 张，其余固定 1 张）。 */
export const MAX_CAROUSEL_CARDS = 5;
/** 便条文案硬上限（>8 拦截）与建议上限（>5 黄色提示）。 */
export const TAB_TEXT_HARD_MAX = 8;
export const TAB_TEXT_SOFT_MAX = 5;

/** 素材种类 → 建议尺寸。 */
export type ImageKind = 'card' | 'full' | 'ball' | 'thumb' | 'icon';

export interface ImageSpec {
  kind: ImageKind;
  /** 建议像素（二倍图）。 */
  w: number;
  h: number;
  /** 客户端显示 dp。 */
  dpW: number;
  dpH: number;
  /** 比例文案，如 3:4。 */
  ratio: string;
  /** 素材用途（提示文案用）。 */
  label: string;
}

export const IMAGE_SPECS: Record<ImageKind, ImageSpec> = {
  card: { kind: 'card', w: 600, h: 800, dpW: 300, dpH: 400, ratio: '3:4', label: '卡片图' },
  full: { kind: 'full', w: 720, h: 1280, dpW: 360, dpH: 640, ratio: '9:16', label: '全屏图' },
  ball: { kind: 'ball', w: 120, h: 120, dpW: 60, dpH: 60, ratio: '1:1', label: '悬浮球' },
  thumb: { kind: 'thumb', w: 80, h: 80, dpW: 40, dpH: 40, ratio: '1:1', label: '横幅缩略图' },
  icon: { kind: 'icon', w: 40, h: 40, dpW: 20, dpH: 20, ratio: '1:1', label: '便条图标' },
};

/** 「建议尺寸 600×800px（3:4 · 二倍图，显示 300×400dp）」；便条图标用短句「建议 40×40px」。 */
export function suggestedSizeText(spec: ImageSpec): string {
  if (spec.kind === 'icon') return `建议 ${spec.w}×${spec.h}px（${spec.ratio} · 二倍图，显示 ${spec.dpW}×${spec.dpH}dp）`;
  return `建议尺寸 ${spec.w}×${spec.h}px（${spec.ratio} · 二倍图，显示 ${spec.dpW}×${spec.dpH}dp）`;
}

/** 比例容差（相对误差）。 */
const RATIO_TOLERANCE = 0.02;
/** 像素容差：实际边长相对建议值偏差 ≤ 5% 视为相符。 */
const SIZE_TOLERANCE = 0.05;

/**
 * 上传后比对：返回 null = 相符；否则返回黄色提示文案。**只提示、不拦截保存**。
 * 先看比例（比例错会被 centerCrop 裁切），比例对再看尺寸（偏小会模糊、偏大拖慢加载）。
 */
export function checkImageSize(spec: ImageSpec, width: number, height: number): string | null {
  if (!width || !height) return null;
  const want = spec.w / spec.h;
  const got = width / height;
  if (Math.abs(got - want) / want > RATIO_TOLERANCE) {
    return `图片 ${width}×${height}px，比例与建议的 ${spec.ratio} 不符，客户端会按 centerCrop 裁切`;
  }
  const diff = (width - spec.w) / spec.w;
  if (diff < -SIZE_TOLERANCE) {
    return `图片 ${width}×${height}px 小于建议的 ${spec.w}×${spec.h}px，高清屏上可能偏模糊`;
  }
  if (diff > SIZE_TOLERANCE) {
    return `图片 ${width}×${height}px 大于建议的 ${spec.w}×${spec.h}px，会增加加载耗时与失败率`;
  }
  return null;
}

// ---------------------------------------------------------------------------
// 能力矩阵（§1）
// ---------------------------------------------------------------------------

export type CapLevel = 'no' | 'optional' | 'required';

export interface PositionCaps {
  code: PopupPositionCode;
  name: string;
  /** 形态一句话描述（位置开关条用）。 */
  desc: string;
  /** 建议优先级徽章（PRD §2）。 */
  priority: 'P0' | 'P1' | 'P2';
  /** 遮罩类（同一时刻只允许一个）。 */
  mask: boolean;
  /** 卡片数上限。 */
  maxCards: number;
  /** 图片：no 不用 / optional 可选 / required 必填。 */
  image: CapLevel;
  /** 主图规格；P5 无图为 null。 */
  imageSpec: ImageSpec | null;
  /** card.title 行为：none 无 / optional 可选 / required 必填 / ifNoImage 无图时必填（P4）。 */
  title: 'none' | 'optional' | 'required' | 'ifNoImage';
  /** 是否有 按钮文案/标题/描述 这组营销文案（P1/P4/P6/P7）。 */
  cta: boolean;
  /** 便条态：no / optional（默认开）。P4 强制模式下恒 false。 */
  tab: 'no' | 'optional';
  /** 倒计时：no / optional；默认值见 defaultCountdown。 */
  countdown: 'no' | 'optional';
  /** 是否可设为强制模式（仅 P4）。 */
  forceable: boolean;
  /** 遮罩点击关闭是否可配。 */
  maskClosable: 'no' | 'optional';
  /** 位置开关的服务端默认值。 */
  defaultPositionOn: boolean;
  /** 触发时机（只读展示，§7.4）。 */
  trigger: string;
}

export const POSITION_CAPS: Record<PopupPositionCode, PositionCaps> = {
  P1: {
    code: 'P1', name: '启动弹窗', desc: '遮罩 + 居中卡片，可轮播', priority: 'P0',
    mask: true, maxCards: 5, image: 'required', imageSpec: IMAGE_SPECS.card, title: 'optional', cta: true,
    tab: 'optional', countdown: 'optional', forceable: false, maskClosable: 'optional', defaultPositionOn: true,
    trigger: '冷启动首屏加载完成后；或回前台且距上次进入后台 > 回前台间隔（仅评估 P1）',
  },
  P2: {
    code: 'P2', name: '悬浮球', desc: '常驻浮标，可拖动吸边', priority: 'P0',
    mask: false, maxCards: 1, image: 'required', imageSpec: IMAGE_SPECS.ball, title: 'none', cta: false,
    tab: 'no', countdown: 'no', forceable: false, maskClosable: 'no', defaultPositionOn: true,
    trigger: '首屏加载完成后（非遮罩类，可与遮罩类共存）',
  },
  P3: {
    code: 'P3', name: '底部横幅', desc: '贴底条幅，自动轮播', priority: 'P1',
    mask: false, maxCards: 5, image: 'optional', imageSpec: IMAGE_SPECS.thumb, title: 'required', cta: false,
    tab: 'optional', countdown: 'no', forceable: false, maskClosable: 'no', defaultPositionOn: false,
    trigger: '首屏加载完成后（非遮罩类）；启用前需 H5 让出底部高度',
  },
  P4: {
    code: 'P4', name: '更新 / 公告', desc: '遮罩 + 居中卡片，可设强制', priority: 'P1',
    mask: true, maxCards: 1, image: 'optional', imageSpec: IMAGE_SPECS.card, title: 'ifNoImage', cta: true,
    tab: 'optional', countdown: 'no', forceable: true, maskClosable: 'optional', defaultPositionOn: false,
    trigger: '冷启动首屏加载完成后；强制 P4 永远最先，其余按优先级',
  },
  P5: {
    code: 'P5', name: '顶部通栏', desc: '贴顶细条，纯文字 + 关闭', priority: 'P2',
    mask: false, maxCards: 1, image: 'no', imageSpec: null, title: 'required', cta: false,
    tab: 'no', countdown: 'no', forceable: false, maskClosable: 'no', defaultPositionOn: false,
    trigger: '首屏加载完成后（非遮罩类）',
  },
  P6: {
    code: 'P6', name: '退出挽留', desc: '返回键拦截，形态同启动弹窗', priority: 'P2',
    mask: true, maxCards: 1, image: 'required', imageSpec: IMAGE_SPECS.card, title: 'optional', cta: true,
    tab: 'no', countdown: 'no', forceable: false, maskClosable: 'optional', defaultPositionOn: false,
    trigger: '顶层页按返回键时（消费这次返回；关闭后再按返回正常退出）',
  },
  P7: {
    code: 'P7', name: '全屏插屏', desc: '整屏图片，可轮播', priority: 'P2',
    mask: true, maxCards: 5, image: 'required', imageSpec: IMAGE_SPECS.full, title: 'optional', cta: true,
    tab: 'optional', countdown: 'optional', forceable: false, maskClosable: 'no', defaultPositionOn: false,
    trigger: '冷启动首屏加载完成后，与 P1 / P4 互斥',
  },
  P8: {
    code: 'P8', name: '便条', desc: '贴边扁条 28dp，单点跳转 / 长按关闭', priority: 'P0',
    mask: false, maxCards: 1, image: 'required', imageSpec: IMAGE_SPECS.icon, title: 'required', cta: false,
    tab: 'no', countdown: 'no', forceable: false, maskClosable: 'no', defaultPositionOn: true,
    trigger: '首屏加载完成后（非遮罩类，常驻入口）',
  },
};

export function isPopupPosition(s: string): s is PopupPositionCode {
  return (POPUP_POSITION_CODES as string[]).includes(s);
}

/** 倒计时默认值：P1 默认关、P7 默认开（PRD §11 #5/#6）。 */
export function defaultCountdown(p: PopupPositionCode): boolean {
  return p === 'P7';
}

/** 该位置可否有「便条态」（P4 强制模式另算）。 */
export function supportsTab(p: PopupPositionCode): boolean {
  return POSITION_CAPS[p].tab !== 'no';
}

/** P4 且不可关闭 = 强制模式。 */
export function isForcedMode(position: PopupPositionCode, closable: boolean): boolean {
  return position === 'P4' && !closable;
}

/** 卡片主图对应的规格（P5 无图返回 null）。 */
export function cardImageSpec(p: PopupPositionCode): ImageSpec | null {
  return POSITION_CAPS[p].imageSpec;
}

/** 该位置卡片是否展示「图片」上传位。P8 的图标本身就是卡片图。 */
export function hasCardImage(p: PopupPositionCode): boolean {
  return POSITION_CAPS[p].image !== 'no';
}

// ---------------------------------------------------------------------------
// 行为字段归一化（与服务端一致：不适用字段强制回默认）
// ---------------------------------------------------------------------------

export interface BehaviorFields {
  position: PopupPositionCode;
  tabEnabled: boolean;
  countdown: boolean;
  maskClosable: boolean;
  closable: boolean;
  badge: boolean;
  autoplaySeconds: number;
  resumeGapMinutes: number;
}

export const DEFAULT_AUTOPLAY_SECONDS = 5;
export const DEFAULT_RESUME_GAP_MINUTES = 30;

/**
 * §1：服务端保存时按矩阵归一化。前端先做一遍，保证「所见即所存」：
 *  - 非 P1/P7 的 countdown 恒 false；非 P4 的 closable 恒 true；
 *  - P4 强制模式下 tabEnabled / maskClosable / countdown 恒 false；
 *  - 不支持便条态的位置 tabEnabled 恒 false；无遮罩点击配置的位置 maskClosable 恒 false；
 *  - badge 仅 P2；autoplay 仅 P3；resumeGap 仅 P1，其余回默认。
 */
export function normalizeBehavior<T extends BehaviorFields>(f: T): T {
  const caps = POSITION_CAPS[f.position];
  const closable = f.position === 'P4' ? f.closable : true;
  const forced = isForcedMode(f.position, closable);
  return {
    ...f,
    closable,
    countdown: caps.countdown === 'optional' && !forced ? f.countdown : false,
    tabEnabled: caps.tab === 'optional' && !forced ? f.tabEnabled : false,
    maskClosable: caps.maskClosable === 'optional' && !forced ? f.maskClosable : false,
    badge: f.position === 'P2' ? f.badge : false,
    autoplaySeconds: f.position === 'P3' ? f.autoplaySeconds : DEFAULT_AUTOPLAY_SECONDS,
    resumeGapMinutes: f.position === 'P1' ? f.resumeGapMinutes : DEFAULT_RESUME_GAP_MINUTES,
  };
}

/** 切换位置时的行为字段默认值（便条态默认开；P1 倒计时关、P7 开；遮罩点击默认可关；P2 红点默认开）。 */
export function behaviorDefaults(position: PopupPositionCode): Omit<BehaviorFields, 'position'> {
  const caps = POSITION_CAPS[position];
  return {
    tabEnabled: caps.tab === 'optional',
    countdown: caps.countdown === 'optional' && defaultCountdown(position),
    maskClosable: caps.maskClosable === 'optional',
    closable: true,
    badge: position === 'P2',
    autoplaySeconds: DEFAULT_AUTOPLAY_SECONDS,
    resumeGapMinutes: DEFAULT_RESUME_GAP_MINUTES,
  };
}

// ---------------------------------------------------------------------------
// 版本号（API 收发 X.Y.Z；存 major*10000+minor*100+patch，与 app/build.gradle 一致）
// ---------------------------------------------------------------------------

const VERSION_RE = /^(\d{1,2})\.(\d{1,2})\.(\d{1,2})$/;

/** 'X.Y.Z' → versionCode；非法返回 null。minor / patch 必须 ≤ 99，否则换算会撞码。 */
export function versionToCode(v: string): number | null {
  const m = VERSION_RE.exec(v.trim());
  if (!m) return null;
  return Number(m[1]) * 10000 + Number(m[2]) * 100 + Number(m[3]);
}

/** versionCode → 'X.Y.Z'；0 / 非法返回空串（= 不限）。 */
export function codeToVersion(code: number): string {
  if (!Number.isFinite(code) || code <= 0) return '';
  const major = Math.floor(code / 10000);
  const minor = Math.floor((code % 10000) / 100);
  const patch = code % 100;
  return `${major}.${minor}.${patch}`;
}

/** 空串合法（不限）；否则必须是 X.Y.Z。 */
export function isValidVersion(v: string): boolean {
  return v.trim() === '' || versionToCode(v) !== null;
}

// ---------------------------------------------------------------------------
// 链接规则（§2.2）
// ---------------------------------------------------------------------------

function hasHost(v: string): boolean {
  // 不用 new URL：WHATWG 会把 http:///x 容错成 host=x
  return /^https?:\/\/[^/?#\s@]+/i.test(v);
}

/** 返回错误文案；合法返回 null。空 = 整图不可点，合法。 */
export function validateLink(link: string, openMode: 'webview' | 'browser' | 'store'): string | null {
  const v = link.trim();
  if (!v) return null;
  if (v.length > 512) return '跳转链接不能超过 512 字符';
  if (v.startsWith('/')) {
    if (v.startsWith('//')) return '站内路径不能以 // 开头';
    return openMode === 'webview' ? null : '站内相对路径 /path 仅支持「WebView 内」跳转方式';
  }
  if (/^https?:\/\/\S+$/i.test(v) && hasHost(v)) return null;
  if (/^market:\/\/\S+$/.test(v)) return openMode === 'store' ? null : 'market:// 仅支持「应用商店」跳转方式';
  return '链接需为 /站内路径、http(s):// 或 market://（商店方式）';
}

// ---------------------------------------------------------------------------
// 国家码
// ---------------------------------------------------------------------------

export const COMMON_COUNTRIES = ['PH', 'ID', 'VN', 'TH', 'MY', 'SG', 'IN'];

/** 规范化为大写 ISO2；非法返回 null。 */
export function normalizeCountry(s: string): string | null {
  const v = s.trim().toUpperCase();
  return /^[A-Z]{2}$/.test(v) ? v : null;
}

// ---------------------------------------------------------------------------
// 频控表（§7.3，写死在客户端，后台只读展示）
// ---------------------------------------------------------------------------

export interface FreqRow {
  key: string;
  position: PopupPositionCode;
  label: string;
  condition: string;
  afterClose: string;
  /** P4 两行用：true = 强制行。 */
  forced?: boolean;
}

export const FREQ_ROWS: FreqRow[] = [
  { key: 'P1', position: 'P1', label: 'P1 启动弹窗', condition: '当日未自动展示过（每日 1 次）', afterClose: '便条态开 → 收起为便条（当日有效）；当日不再自动弹' },
  { key: 'P2', position: 'P2', label: 'P2 悬浮球', condition: '本会话未关闭', afterClose: '本会话（进程存活期）不再出现' },
  { key: 'P3', position: 'P3', label: 'P3 底部横幅', condition: '不在「关闭后 24h」窗口内', afterClose: '便条态开 → 便条（24h 有效）；24h 内不再自动出现' },
  { key: 'P4', position: 'P4', label: 'P4 更新 / 公告（普通）', condition: '当日未展示过', afterClose: '便条态开 → 便条（当日有效）', forced: false },
  { key: 'P4F', position: 'P4', label: 'P4 更新 / 公告（强制）', condition: '每次冷启动', afterClose: '不可关闭（无 X、遮罩无效、返回键不关弹窗）', forced: true },
  { key: 'P5', position: 'P5', label: 'P5 顶部通栏', condition: '当日未关闭', afterClose: '当日不再出现' },
  { key: 'P6', position: 'P6', label: 'P6 退出挽留', condition: '当日未展示过 ∧ 本会话未展示过', afterClose: '直接消失' },
  { key: 'P7', position: 'P7', label: 'P7 全屏插屏', condition: '当日未自动展示过', afterClose: '同 P1' },
  { key: 'P8', position: 'P8', label: 'P8 便条', condition: '当日未被长按关闭', afterClose: '—' },
];

/** 当前编辑的弹窗对应高亮哪一行。 */
export function currentFreqRowKey(position: PopupPositionCode, forced: boolean): string {
  return position === 'P4' ? (forced ? 'P4F' : 'P4') : position;
}

// ---------------------------------------------------------------------------
// 列表状态文案
// ---------------------------------------------------------------------------

export const STATUS_LABEL: Record<'active' | 'scheduled' | 'ended' | 'disabled', string> = {
  active: '生效中',
  scheduled: '未开始',
  ended: '已结束',
  disabled: '已关闭',
};

/** 字符数（按 Unicode 码点，与服务端 rune 计数一致）。 */
export function charLen(s: string): number {
  return Array.from(s ?? '').length;
}

/** 从图片 URL / key 取展示用文件名：优先 popup/images/ 之后的路径。 */
export function imageKeyLabel(urlOrKey: string): string {
  if (!urlOrKey) return '';
  if (urlOrKey.startsWith('data:')) return '(本地预览 · mock)';
  const i = urlOrKey.indexOf('popup/images/');
  if (i >= 0) return urlOrKey.slice(i).split(/[?#]/)[0];
  try {
    return new URL(urlOrKey).pathname.replace(/^\//, '');
  } catch {
    return urlOrKey;
  }
}
