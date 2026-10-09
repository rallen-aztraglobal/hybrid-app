/**
 * 弹窗数据看板的纯逻辑（12-popup.md §5）：日期范围、指标适用性、阈值着色、KPI 汇总。
 * 比率以服务端 rates 为准（小数，0.82 = 82%）；KPI 汇总行按同口径用各行计数重算。
 */
import type { PopupPositionCode, PopupStatRow } from './types';
import { dateToKey } from './dateTime';
import { POSITION_CAPS } from './popupMeta';

export type RangePreset = 'today' | '7d' | '30d' | 'custom';

/** 最大跨度（含首尾共 92 天，对 ≤92 天的后端校验取保守口径）。 */
export const MAX_RANGE_DAYS = 92;

/** 服务端默认统计时区（POPUP_TZ 默认值）；首次请求前使用，之后以响应里的 tz 为准。 */
export const DEFAULT_STATS_TZ = 'Asia/Manila';

/** 某时刻在指定时区的日历日 YYYY-MM-DD；时区非法时回退本地。 */
export function dateKeyInTz(d: Date, tz: string): string {
  try {
    return new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(d);
  } catch {
    return dateToKey(d);
  }
}

/** 今日 / 近 N 天按统计时区归日（与服务端 stat_date 口径一致）。 */
export function rangeForPreset(
  preset: Exclude<RangePreset, 'custom'>,
  now = new Date(),
  tz: string = DEFAULT_STATS_TZ,
): { from: string; to: string } {
  const to = dateKeyInTz(now, tz);
  if (preset === 'today') return { from: to, to };
  const days = preset === '7d' ? 7 : 30;
  const base = keyToUtc(to) as number;
  const f = new Date(base - (days - 1) * 86_400_000);
  const from = `${f.getUTCFullYear()}-${String(f.getUTCMonth() + 1).padStart(2, '0')}-${String(f.getUTCDate()).padStart(2, '0')}`;
  return { from, to };
}

function keyToUtc(key: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(key);
  if (!m) return null;
  return Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
}

/** 含首尾的天数；非法返回 null。 */
export function rangeDays(from: string, to: string): number | null {
  const a = keyToUtc(from);
  const b = keyToUtc(to);
  if (a === null || b === null) return null;
  return Math.round((b - a) / 86_400_000) + 1;
}

/** 返回错误文案；合法返回 null。 */
export function validateRange(from: string, to: string): string | null {
  const n = rangeDays(from, to);
  if (n === null) return '请选择起止日期';
  if (n < 1) return '结束日期不能早于开始日期';
  if (n > MAX_RANGE_DAYS) return `日期跨度不能超过 ${MAX_RANGE_DAYS} 天（当前 ${n} 天）`;
  return null;
}

// ---------------------------------------------------------------------------
// 指标
// ---------------------------------------------------------------------------

export type RateKey =
  | 'showRate'
  | 'ctr'
  | 'closeRate'
  | 'loadFailRate'
  | 'carouselDepth'
  | 'recoveryRate'
  | 'tabAbandonRate'
  | 'totalCtr';

export type ThresholdLevel = '' | 'warn' | 'crit';

/** 阈值（按原型）：展示率 <60%、点击率 <3%、关闭率 >90%、便条弃用率 >30% 黄；加载失败率 >5% 红。 */
export function thresholdLevel(key: RateKey, rate: number | null): ThresholdLevel {
  if (rate === null) return '';
  switch (key) {
    case 'showRate':
      return rate < 0.6 ? 'warn' : '';
    case 'ctr':
      return rate < 0.03 ? 'warn' : '';
    case 'closeRate':
      return rate > 0.9 ? 'warn' : '';
    case 'tabAbandonRate':
      return rate > 0.3 ? 'warn' : '';
    case 'loadFailRate':
      return rate > 0.05 ? 'crit' : '';
    default:
      return '';
  }
}

export const THRESHOLD_TIPS: Partial<Record<RateKey, string>> = {
  showRate: '展示率 < 60%：预加载失败或频控过紧（工程问题）',
  ctr: '点击率 < 3%：素材 / 文案问题',
  closeRate: '关闭率 > 90%：内容与用户预期不符',
  loadFailRate: '加载失败率 > 5%：图片过大或 CDN 问题',
  tabAbandonRate: '便条弃用率 > 30%：便条构成干扰，考虑缩小或换位置',
};

/**
 * 某一行某个比率的取值：不适用或分母为 0 返回 null（界面显示「—」）。
 * 适用性按位置能力：轮播深度仅多卡位置；找回率 / 总点击率需支持便条态；
 * 便条弃用率除便条态位置外，独立 P8 本身就是便条；P8 无「关闭」这一概念（长按走 tab_dismiss）。
 */
export function rateValue(row: PopupStatRow, key: RateKey): number | null {
  // 未知位置码（库里残留 / 新版后端新增）：按「不适用」兜底，绝不抛错
  const caps = POSITION_CAPS[row.position];
  if (!caps) return null;
  const r = row.rates;
  // 展示次数（含便条重开）；旧后端无该字段时回退到曝光数
  const shown = row.displaysAll ?? row.impressions;
  switch (key) {
    case 'showRate':
      return row.trigger > 0 ? r.showRate : null;
    case 'ctr':
      return row.impressions > 0 ? r.ctr : null;
    case 'closeRate':
      return row.position === 'P8' || shown === 0 ? null : r.closeRate;
    case 'loadFailRate':
      return row.trigger > 0 ? r.loadFailRate : null;
    case 'carouselDepth':
      return caps.maxCards > 1 && shown > 0 ? r.carouselDepth : null;
    case 'recoveryRate':
      return caps.tab !== 'no' && row.collapses > 0 ? r.recoveryRate : null;
    case 'tabAbandonRate':
      return (caps.tab !== 'no' || row.position === 'P8') && row.tabImpressions > 0 ? r.tabAbandonRate : null;
    case 'totalCtr':
      return caps.tab !== 'no' && row.impressions > 0 ? r.totalCtr : null;
  }
}

export interface KpiTotals {
  trigger: number;
  impressions: number;
  clicks: number;
  displays: number;
  collapses: number;
  tabClicks: number;
  showRate: number | null;
  ctr: number | null;
  recoveryRate: number | null;
}

function div(a: number, b: number): number | null {
  return b > 0 ? a / b : null;
}

/** KPI 卡：触发 / 曝光 / 点击 / 展示率 / 点击率 / 找回率，按契约公式对各行求和再相除。 */
export function aggregateKpis(rows: PopupStatRow[]): KpiTotals {
  const t = { trigger: 0, impressions: 0, clicks: 0, displays: 0, collapses: 0, tabClicks: 0 };
  for (const r of rows) {
    t.trigger += r.trigger;
    t.impressions += r.impressions;
    t.clicks += r.clicks;
    t.displays += r.displays;
    t.collapses += r.collapses;
    t.tabClicks += r.tabClicks;
  }
  return {
    ...t,
    showRate: div(t.displays, t.trigger),
    ctr: div(t.clicks, t.impressions),
    recoveryRate: div(t.tabClicks, t.collapses),
  };
}

export function sumFiltered(rows: PopupStatRow[]) {
  const s = { frequency: 0, mutex: 0, targeting: 0, time: 0 };
  for (const r of rows) {
    s.frequency += r.filtered.frequency;
    s.mutex += r.filtered.mutex;
    s.targeting += r.filtered.targeting;
    s.time += r.filtered.time;
  }
  return s;
}

/** 百分比文案；null → 「—」。 */
export function pct(v: number | null, digits = 1): string {
  return v === null ? '—' : `${(v * 100).toFixed(digits)}%`;
}

export function fmtInt(n: number | null | undefined): string {
  return n == null ? '—' : Math.round(n).toLocaleString('en-US');
}

/** 位置过滤辅助（看板位置筛选的可选项）。 */
export function positionLabel(code: PopupPositionCode): string {
  return `${code} · ${POSITION_CAPS[code]?.name ?? '未知'}`;
}
