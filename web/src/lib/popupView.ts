/** 弹窗列表 / 抽屉用到的展示格式化（纯函数）。 */
import { pad2 } from './dateTime';

export function fmtLocalTime(s: string | null | undefined): string {
  return fmt(s ?? null);
}

function fmt(s: string | null): string {
  if (!s) return '';
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

/** 生效期两行展示：开始 / → 结束；空 = 不限。 */
export function formatRange(startAt: string | null, endAt: string | null): string[] {
  return [fmt(startAt) || '不限（立即）', `→ ${fmt(endAt) || '不限'}`];
}
