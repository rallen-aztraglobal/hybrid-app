import { describe, expect, it } from 'vitest';
import { compareVersions, highestVersion, isVersionLowerThanCurrent, parseVersion } from './version';

describe('parseVersion', () => {
  it('解析合法的 X.Y.Z', () => {
    expect(parseVersion('1.3.8')).toEqual([1, 3, 8]);
    expect(parseVersion('0.0.0')).toEqual([0, 0, 0]);
  });

  it('非法格式返回 null（不抛异常）', () => {
    expect(parseVersion('1.2')).toBeNull();
    expect(parseVersion('not-a-version')).toBeNull();
    expect(parseVersion('1.2.3.4')).toBeNull();
    expect(parseVersion('')).toBeNull();
  });
});

describe('compareVersions', () => {
  it('按数值比较，而非字符串字典序（1.10.0 > 1.2.0）', () => {
    expect(compareVersions(parseVersion('1.10.0')!, parseVersion('1.2.0')!)).toBe(1);
  });

  it('2.0.0 > 1.99.99', () => {
    expect(compareVersions(parseVersion('2.0.0')!, parseVersion('1.99.99')!)).toBe(1);
  });

  it('1.3.8 == 1.3.8', () => {
    expect(compareVersions(parseVersion('1.3.8')!, parseVersion('1.3.8')!)).toBe(0);
  });

  it('1.3.7 < 1.3.8', () => {
    expect(compareVersions(parseVersion('1.3.7')!, parseVersion('1.3.8')!)).toBe(-1);
  });
});

describe('highestVersion', () => {
  it('取语义版本最高的一个，而非数组最后一项', () => {
    expect(highestVersion(['1.0.0', '2.0.0', '1.9.0'])).toBe('2.0.0');
  });

  it('忽略无法解析的非法版本号', () => {
    expect(highestVersion(['not-a-version', '1.5.0', 'also-bad'])).toBe('1.5.0');
  });

  it('全部非法或为空数组时返回 null', () => {
    expect(highestVersion([])).toBeNull();
    expect(highestVersion(['bad', 'also-bad'])).toBeNull();
  });
});

describe('isVersionLowerThanCurrent', () => {
  it('当前版本为 null（暂无成功构建）时永远放行', () => {
    expect(isVersionLowerThanCurrent('0.0.1', null)).toBe(false);
  });

  it('低于当前版本时返回 true', () => {
    expect(isVersionLowerThanCurrent('1.3.7', '1.3.8')).toBe(true);
  });

  it('等于或高于当前版本时返回 false（允许）', () => {
    expect(isVersionLowerThanCurrent('1.3.8', '1.3.8')).toBe(false);
    expect(isVersionLowerThanCurrent('1.3.9', '1.3.8')).toBe(false);
    expect(isVersionLowerThanCurrent('1.10.0', '1.2.0')).toBe(false);
  });
});
