import type { BrandCode, Channel } from './types';

/**
 * CSV 解析 —— 事实来源是仓库根的 channels/<brand>.csv。
 *
 * 字段：flavorName|applicationId|palCode|appName（# 开头为注释）。
 * 与 app/build.gradle 的 loadChannels 解析逻辑一致（split('|', -1)、跳过 # 与空行、
 * 至少 4 段）。本函数在前端复用，把 CSV 文本还原为 Channel[]，作为后端未就绪时
 * 的 mock 数据源，确保「以现有 channels/*.csv 为事实来源」。
 */
export function parseChannelsCsv(text: string, brand: BrandCode): Channel[] {
  const out: Channel[] = [];
  let seq = 0;
  for (const rawLine of text.split('\n')) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const p = line.split('|');
    if (p.length < 4) continue;
    const flavorName = p[0].trim();
    const applicationId = p[1].trim();
    const palCode = p[2].trim();
    const appName = p.slice(3).join('|').trim(); // 容错：应用名理论上不含 |
    seq += 1;
    out.push({
      id: `${brand}-${flavorName}`,
      brandCode: brand,
      flavorName,
      applicationId,
      palCode,
      appName,
      // mock：现网默认全部启用、继承品牌域名；少量在 data.ts 里被标记停用以演示筛选
      status: 'enabled',
      useBrandDomains: true,
    });
  }
  return out;
}
