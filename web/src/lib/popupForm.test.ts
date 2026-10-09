import { describe, expect, it } from 'vitest';
import {
  blankCard,
  blankForm,
  localToRFC3339,
  rfc3339ToLocal,
  switchPosition,
  toPopupInput,
  validatePopupForm,
  type PopupForm,
} from './popupForm';

function okForm(over: Partial<PopupForm> = {}): PopupForm {
  const f = blankForm('P1');
  f.name = '国庆首存';
  f.cards = [{ ...blankCard(), imageUrl: 'https://cdn/x.png', meta: { width: 600, height: 800 } }];
  f.tabText = 'Bonus';
  return { ...f, ...over };
}

describe('时间格式', () => {
  it('本地 <-> RFC3339 往返', () => {
    const s = localToRFC3339('2026-10-01T08:30');
    expect(s).toMatch(/^2026-10-01T08:30:00[+-]\d{2}:\d{2}$/);
    expect(rfc3339ToLocal(s)).toBe('2026-10-01T08:30');
    expect(localToRFC3339('')).toBeNull();
    expect(rfc3339ToLocal(null)).toBe('');
  });
});

describe('validatePopupForm', () => {
  it('合法表单无错误', () => {
    expect(validatePopupForm(okForm()).errors).toEqual([]);
  });
  it('名称 / 优先级 / 时间 / 版本', () => {
    const r = validatePopupForm(
      okForm({ name: '', priority: '1000', startAt: '2026-10-02T00:00', endAt: '2026-10-01T00:00', minVersion: '1.0', maxVersion: '' }),
    );
    expect(r.errors.length).toBe(4);
  });
  it('版本下限需小于上限', () => {
    const r = validatePopupForm(okForm({ minVersion: '1.0.5', maxVersion: '1.0.5' }));
    expect(r.errors.some((e) => e.includes('小于'))).toBe(true);
  });
  it('必须上传图片（P1）', () => {
    const f = okForm();
    f.cards = [blankCard()];
    expect(validatePopupForm(f).errors.some((e) => e.includes('上传图片'))).toBe(true);
  });
  it('尺寸不符只警告不拦截', () => {
    const f = okForm();
    f.cards[0].meta = { width: 800, height: 800 };
    const r = validatePopupForm(f);
    expect(r.errors).toEqual([]);
    expect(r.warnings.length).toBe(1);
  });
  it('P3 / P5 文案必填', () => {
    const p3 = switchPosition(okForm(), 'P3');
    expect(validatePopupForm(p3).errors.some((e) => e.includes('文案必填'))).toBe(true);
    const p5 = switchPosition(okForm(), 'P5');
    expect(validatePopupForm(p5).errors.some((e) => e.includes('文案必填'))).toBe(true);
  });
  it('P8 文案：>5 警告，>8 拦截', () => {
    const p8 = switchPosition(okForm(), 'P8');
    p8.cards[0].meta = { width: 40, height: 40 };
    p8.cards[0].title = '123456';
    let r = validatePopupForm(p8);
    expect(r.errors).toEqual([]);
    expect(r.warnings.length).toBe(1);
    p8.cards[0].title = '123456789';
    r = validatePopupForm(p8);
    expect(r.errors.some((e) => e.includes('8'))).toBe(true);
  });
  it('便条态开启后文案必填且 ≤8', () => {
    expect(validatePopupForm(okForm({ tabText: '' })).errors.length).toBe(1);
    expect(validatePopupForm(okForm({ tabText: '123456789' })).errors.length).toBe(1);
    expect(validatePopupForm(okForm({ tabText: '123456' })).warnings.length).toBe(1);
  });
  it('P4 强制模式不校验便条文案；无图须有标题', () => {
    const f = switchPosition(okForm({ tabText: '' }), 'P4');
    f.closable = false;
    f.cards = [blankCard()];
    expect(validatePopupForm(f).errors.some((e) => e.includes('标题'))).toBe(true);
    expect(validatePopupForm(f).errors.some((e) => e.includes('便条'))).toBe(false);
  });
  it('受限账号必须选品牌 / 渠道包', () => {
    const r = validatePopupForm(okForm(), { allBrands: false, allChannels: false });
    expect(r.errors.length).toBe(2);
  });
  it('轮播间隔 / 回前台间隔范围', () => {
    expect(validatePopupForm(okForm({ resumeGapMinutes: '3' })).errors.length).toBe(1);
    const p3 = switchPosition(okForm(), 'P3');
    p3.cards[0].title = 'x';
    p3.autoplaySeconds = '31';
    expect(validatePopupForm(p3).errors.length).toBe(1);
  });
  it('链接与跳转方式冲突', () => {
    const f = okForm();
    f.cards[0].linkUrl = '/promo';
    f.openMode = 'browser';
    expect(validatePopupForm(f).errors.length).toBe(1);
  });
});

describe('toPopupInput', () => {
  it('提交前归一化并按下标重写 sort，版本规范化', () => {
    const f = okForm({ minVersion: '1.0.3', countdown: true, maskClosable: true });
    f.cards = [
      { ...blankCard(), id: 34, imageUrl: 'a', buttonText: ' Go ' },
      { ...blankCard(), imageUrl: 'b' },
    ];
    const p = toPopupInput(f);
    expect(p.cards.map((c) => c.sort)).toEqual([0, 1]);
    expect(p.cards[0].id).toBe(34);
    expect(p.cards[1].id).toBeUndefined();
    expect(p.cards[0].buttonText).toBe('Go');
    expect(p.minVersion).toBe('1.0.3');
    expect(p.startAt).toBeNull();
  });
  it('非轮播位置仅保留第 1 张；P5 不带图；不适用字段回默认', () => {
    const f = switchPosition(okForm(), 'P5');
    f.cards = [
      { ...blankCard(), title: 'Notice', imageUrl: 'x', buttonText: 'x' },
      { ...blankCard(), title: 'drop' },
    ];
    f.tabEnabled = true;
    const p = toPopupInput(f);
    expect(p.cards.length).toBe(1);
    expect(p.cards[0].imageUrl).toBe('');
    expect(p.cards[0].buttonText).toBe('');
    expect(p.tabEnabled).toBe(false);
    expect(p.closable).toBe(true);
  });
  it('P4 强制模式', () => {
    const f = switchPosition(okForm(), 'P4');
    f.closable = false;
    f.tabEnabled = true;
    const p = toPopupInput(f);
    expect(p.closable).toBe(false);
    expect(p.tabEnabled).toBe(false);
    expect(p.maskClosable).toBe(false);
    expect(p.tabText).toBe('');
  });
});
