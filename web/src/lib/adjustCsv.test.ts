import { describe, expect, it } from 'vitest';
import { parseAdjustEventsCsv } from './adjustCsv';

describe('parseAdjustEventsCsv', () => {
  it('解析标准 Adjust 导出格式（带引号、token,name,unique）为 { name: token }', () => {
    const csv = ['token,name,unique', '"jrilpe","AddToCart","false"', '"wzb3fb","Login","false"'].join('\n');
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe', Login: 'wzb3fb' });
    expect(result.rows).toEqual([
      { name: 'AddToCart', token: 'jrilpe' },
      { name: 'Login', token: 'wzb3fb' },
    ]);
    expect(result.warnings).toEqual([]);
  });

  it('不带引号也能解析', () => {
    const csv = 'token,name,unique\njrilpe,AddToCart,false\nwzb3fb,Login,false';
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe', Login: 'wzb3fb' });
  });

  it('表头字段顺序不同也能按名定位（不是死按位置取值）', () => {
    const csv = ['name,token,unique', '"AddToCart","jrilpe","false"', '"Login","wzb3fb","false"'].join('\n');
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe', Login: 'wzb3fb' });
  });

  it('跳过空行与结尾多余换行', () => {
    const csv = 'token,name,unique\n\n"jrilpe","AddToCart","false"\n\n\n"wzb3fb","Login","false"\n\n';
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe', Login: 'wzb3fb' });
  });

  it('无法识别表头时按 token,name,unique 兜底解析', () => {
    const csv = '"jrilpe","AddToCart","false"\n"wzb3fb","Login","false"';
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe', Login: 'wzb3fb' });
    expect(result.warnings).toContain('未识别到 token/name 表头，按默认列顺序 token,name,unique 解析');
  });

  it('缺 token 或 name 的行会被跳过并记录 warning', () => {
    const csv = ['token,name,unique', '"jrilpe","AddToCart","false"', '"","BadRow","false"', ',,'].join('\n');
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ AddToCart: 'jrilpe' });
    expect(result.warnings.length).toBe(2);
  });

  it('同名事件重复出现时以最后一次为准，并给出 warning', () => {
    const csv = ['token,name,unique', '"aaa111","Login","false"', '"bbb222","Login","false"'].join('\n');
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({ Login: 'bbb222' });
    expect(result.warnings.some((w) => w.includes('重复'))).toBe(true);
  });

  it('空 CSV 返回空结果并给出 warning', () => {
    const result = parseAdjustEventsCsv('');
    expect(result.events).toEqual({});
    expect(result.warnings).toEqual(['CSV 内容为空']);
  });

  it('文档示例的完整场景（6 个事件）', () => {
    const csv = [
      'token,name,unique',
      '"jrilpe","AddToCart","false"',
      '"ny18sp","CompleteRegistration","false"',
      '"wzb3fb","Login","false"',
      '"4y3tr5","OldRegPurchase","false"',
      '"gyuu2f","Purchase","false"',
      '"975l72","TPFirstDeposit","false"',
    ].join('\n');
    const result = parseAdjustEventsCsv(csv);
    expect(result.events).toEqual({
      AddToCart: 'jrilpe',
      CompleteRegistration: 'ny18sp',
      Login: 'wzb3fb',
      OldRegPurchase: '4y3tr5',
      Purchase: 'gyuu2f',
      TPFirstDeposit: '975l72',
    });
  });
});
