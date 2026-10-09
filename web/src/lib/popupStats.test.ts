import { describe, expect, it } from 'vitest';
import type { PopupStatRow } from './types';
import { dateKeyInTz, aggregateKpis, rangeDays, rangeForPreset, rateValue, thresholdLevel, validateRange, pct } from './popupStats';

function row(over: Partial<PopupStatRow> = {}): PopupStatRow {
  return {
    popupId: 1, name: 'x', position: 'P1', deleted: false,
    trigger: 1000, displays: 820, impressions: 1310, clicks: 96, closes: 640, loadFails: 12,
    collapses: 600, tabImpressions: 610, tabClicks: 75, tabDismisses: 40, slides: 530,
    filtered: { frequency: 1, mutex: 2, targeting: 3, time: 4 },
    closeByMethod: { button: 1, mask: 1, back: 1 },
    rates: { showRate: 0.82, ctr: 0.073, closeRate: 0.78, loadFailRate: 0.012, carouselDepth: 1.6, recoveryRate: 0.125, tabAbandonRate: 0.066, totalCtr: 0.13 },
    cards: [], daily: [],
    ...over,
  };
}

describe('阈值着色', () => {
  it('黄 / 红边界', () => {
    expect(thresholdLevel('showRate', 0.59)).toBe('warn');
    expect(thresholdLevel('showRate', 0.6)).toBe('');
    expect(thresholdLevel('ctr', 0.029)).toBe('warn');
    expect(thresholdLevel('ctr', 0.03)).toBe('');
    expect(thresholdLevel('closeRate', 0.91)).toBe('warn');
    expect(thresholdLevel('closeRate', 0.9)).toBe('');
    expect(thresholdLevel('tabAbandonRate', 0.31)).toBe('warn');
    expect(thresholdLevel('loadFailRate', 0.051)).toBe('crit');
    expect(thresholdLevel('loadFailRate', 0.05)).toBe('');
    expect(thresholdLevel('showRate', null)).toBe('');
  });
});

describe('指标适用性', () => {
  it('P1 全部适用', () => {
    expect(rateValue(row(), 'recoveryRate')).toBe(0.125);
    expect(rateValue(row(), 'carouselDepth')).toBe(1.6);
  });
  it('P2 无便条 / 无轮播 -> null', () => {
    const r = row({ position: 'P2' });
    expect(rateValue(r, 'recoveryRate')).toBeNull();
    expect(rateValue(r, 'totalCtr')).toBeNull();
    expect(rateValue(r, 'carouselDepth')).toBeNull();
    expect(rateValue(r, 'tabAbandonRate')).toBeNull();
    expect(rateValue(r, 'ctr')).toBe(0.073);
  });
  it('P8 无关闭率，有便条弃用率', () => {
    const r = row({ position: 'P8' });
    expect(rateValue(r, 'closeRate')).toBeNull();
    expect(rateValue(r, 'tabAbandonRate')).toBe(0.066);
    expect(rateValue(r, 'recoveryRate')).toBeNull();
  });
  it('分母为 0 -> null', () => {
    expect(rateValue(row({ trigger: 0 }), 'showRate')).toBeNull();
    expect(rateValue(row({ collapses: 0 }), 'recoveryRate')).toBeNull();
    expect(rateValue(row({ impressions: 0 }), 'ctr')).toBeNull();
  });
});

describe('KPI 汇总', () => {
  it('按契约公式求和后相除', () => {
    const k = aggregateKpis([row(), row({ popupId: 2, trigger: 500, displays: 100, impressions: 200, clicks: 4, collapses: 0, tabClicks: 0 })]);
    expect(k.trigger).toBe(1500);
    expect(k.showRate).toBeCloseTo(920 / 1500);
    expect(k.ctr).toBeCloseTo(100 / 1510);
    expect(k.recoveryRate).toBeCloseTo(75 / 600);
    expect(aggregateKpis([]).showRate).toBeNull();
    expect(pct(null)).toBe('—');
    expect(pct(0.1234)).toBe('12.3%');
  });
});

describe('日期范围', () => {
  it('预设', () => {
    const now = new Date(2026, 9, 9);
    expect(rangeForPreset('today', now)).toEqual({ from: '2026-10-09', to: '2026-10-09' });
    expect(rangeForPreset('7d', now, 'Asia/Manila')).toEqual({ from: '2026-10-03', to: '2026-10-09' });
    expect(rangeForPreset('30d', now).from).toBe('2026-09-10');
  });
  it('跨度校验', () => {
    expect(rangeDays('2026-10-03', '2026-10-09')).toBe(7);
    expect(validateRange('2026-10-03', '2026-10-09')).toBeNull();
    expect(validateRange('2026-10-09', '2026-10-03')).not.toBeNull();
    expect(validateRange('2026-01-01', '2026-04-02')).toBeNull(); // 92 天
    expect(validateRange('2026-01-01', '2026-04-03')).not.toBeNull(); // 93 天
    expect(validateRange('', '2026-04-03')).not.toBeNull();
  });
});

describe('未知位置码 / displaysAll / 时区', () => {
  it('未知位置码所有比率为 null，不抛错', () => {
    const r = row({ position: '' as never });
    for (const k of ['showRate', 'ctr', 'closeRate', 'loadFailRate', 'carouselDepth', 'recoveryRate', 'tabAbandonRate', 'totalCtr'] as const) {
      expect(rateValue(r, k)).toBeNull();
    }
  });
  it('关闭率 / 轮播深度的 — 判定用 displaysAll', () => {
    expect(rateValue(row({ displaysAll: 0, impressions: 50 }), 'closeRate')).toBeNull();
    expect(rateValue(row({ displaysAll: 10, impressions: 0 }), 'closeRate')).toBe(0.78);
    expect(rateValue(row({ displaysAll: 0 }), 'carouselDepth')).toBeNull();
    expect(rateValue(row({ impressions: 0 }), 'closeRate')).toBeNull(); // 字段缺失回退旧逻辑
  });
  it('预设按指定时区归日', () => {
    const now = new Date('2026-10-08T20:00:00Z'); // Manila 已是 10-09
    expect(dateKeyInTz(now, 'Asia/Manila')).toBe('2026-10-09');
    expect(dateKeyInTz(now, 'UTC')).toBe('2026-10-08');
    expect(rangeForPreset('7d', now, 'Asia/Manila')).toEqual({ from: '2026-10-03', to: '2026-10-09' });
    expect(rangeForPreset('today', now, 'UTC')).toEqual({ from: '2026-10-08', to: '2026-10-08' });
  });
});
