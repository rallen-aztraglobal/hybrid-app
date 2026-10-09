import { describe, expect, it } from 'vitest';
import {
  IMAGE_SPECS,
  POSITION_CAPS,
  behaviorDefaults,
  checkImageSize,
  codeToVersion,
  currentFreqRowKey,
  imageKeyLabel,
  isValidVersion,
  normalizeBehavior,
  normalizeCountry,
  suggestedSizeText,
  validateLink,
  versionToCode,
} from './popupMeta';

describe('版本号换算', () => {
  it('X.Y.Z <-> versionCode', () => {
    expect(versionToCode('1.0.3')).toBe(10003);
    expect(versionToCode('2.10.15')).toBe(21015);
    expect(codeToVersion(10003)).toBe('1.0.3');
    expect(codeToVersion(0)).toBe('');
  });
  it('非法版本', () => {
    expect(versionToCode('1.0')).toBeNull();
    expect(versionToCode('1.100.0')).toBeNull();
    expect(versionToCode('100.0.0')).toBeNull();
    expect(versionToCode('99.99.99')).toBe(999999);
    expect(versionToCode('a.b.c')).toBeNull();
    expect(isValidVersion('')).toBe(true);
    expect(isValidVersion('1.0.3')).toBe(true);
    expect(isValidVersion('1.0')).toBe(false);
  });
});

describe('能力矩阵归一化', () => {
  const base = (position: keyof typeof POSITION_CAPS) => ({
    position,
    tabEnabled: true,
    countdown: true,
    maskClosable: true,
    closable: false,
    badge: true,
    autoplaySeconds: 9,
    resumeGapMinutes: 99,
  });
  it('非 P1/P7 倒计时恒 false，非 P4 closable 恒 true', () => {
    const n = normalizeBehavior(base('P6'));
    expect(n.countdown).toBe(false);
    expect(n.closable).toBe(true);
    expect(n.tabEnabled).toBe(false);
  });
  it('P4 强制模式：便条 / 遮罩关闭 / 倒计时全部关', () => {
    const n = normalizeBehavior(base('P4'));
    expect(n.closable).toBe(false);
    expect(n.tabEnabled).toBe(false);
    expect(n.maskClosable).toBe(false);
    expect(n.countdown).toBe(false);
  });
  it('P4 非强制保留便条与遮罩关闭', () => {
    const n = normalizeBehavior({ ...base('P4'), closable: true });
    expect(n.tabEnabled).toBe(true);
    expect(n.maskClosable).toBe(true);
  });
  it('P7 无遮罩点击配置；轮播间隔 / 回前台间隔 / 红点仅各自位置生效', () => {
    const n = normalizeBehavior(base('P7'));
    expect(n.maskClosable).toBe(false);
    expect(n.countdown).toBe(true);
    expect(n.badge).toBe(false);
    expect(n.autoplaySeconds).toBe(5);
    expect(n.resumeGapMinutes).toBe(30);
    expect(normalizeBehavior(base('P3')).autoplaySeconds).toBe(9);
    expect(normalizeBehavior(base('P1')).resumeGapMinutes).toBe(99);
    expect(normalizeBehavior(base('P2')).badge).toBe(true);
  });
  it('默认值：P1 倒计时关、P7 开、便条态默认开', () => {
    expect(behaviorDefaults('P1').countdown).toBe(false);
    expect(behaviorDefaults('P7').countdown).toBe(true);
    expect(behaviorDefaults('P1').tabEnabled).toBe(true);
    expect(behaviorDefaults('P6').tabEnabled).toBe(false);
    expect(behaviorDefaults('P2').badge).toBe(true);
  });
});

describe('建议尺寸与比对', () => {
  it('文案', () => {
    expect(suggestedSizeText(IMAGE_SPECS.card)).toBe('建议尺寸 600×800px（3:4 · 二倍图，显示 300×400dp）');
    expect(suggestedSizeText(IMAGE_SPECS.icon)).toContain('建议 40×40px');
  });
  it('相符 / 比例不符 / 偏小 / 偏大', () => {
    expect(checkImageSize(IMAGE_SPECS.card, 600, 800)).toBeNull();
    expect(checkImageSize(IMAGE_SPECS.card, 606, 800)).toBeNull();
    expect(checkImageSize(IMAGE_SPECS.card, 800, 800)).toContain('比例');
    expect(checkImageSize(IMAGE_SPECS.card, 300, 400)).toContain('偏模糊');
    expect(checkImageSize(IMAGE_SPECS.card, 1200, 1600)).toContain('加载');
    expect(checkImageSize(IMAGE_SPECS.full, 720, 1280)).toBeNull();
  });
});

describe('链接 / 国家码 / 频控行', () => {
  it('链接规则', () => {
    expect(validateLink('', 'webview')).toBeNull();
    expect(validateLink('/promo/a', 'webview')).toBeNull();
    expect(validateLink('/promo/a', 'browser')).not.toBeNull();
    expect(validateLink('https://a.com/x', 'browser')).toBeNull();
    expect(validateLink('market://details?id=a', 'store')).toBeNull();
    expect(validateLink('market://details?id=a', 'webview')).not.toBeNull();
    expect(validateLink('javascript:alert(1)', 'webview')).not.toBeNull();
    expect(validateLink('MARKET://details?id=a', 'store')).not.toBeNull();
    expect(validateLink('https://', 'browser')).not.toBeNull();
    expect(validateLink('http:///x', 'browser')).not.toBeNull();
  });
  it('国家码', () => {
    expect(normalizeCountry(' ph ')).toBe('PH');
    expect(normalizeCountry('PHL')).toBeNull();
  });
  it('P4 频控行按强制切换', () => {
    expect(currentFreqRowKey('P4', true)).toBe('P4F');
    expect(currentFreqRowKey('P4', false)).toBe('P4');
    expect(currentFreqRowKey('P1', false)).toBe('P1');
  });
  it('图片 key 文件名', () => {
    expect(imageKeyLabel('https://cdn.x.com/popup/images/ab12-1760.png?x=1')).toBe('popup/images/ab12-1760.png');
  });
});
