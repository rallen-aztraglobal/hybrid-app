/**
 * 本地时区日期/时间小工具（自绘 DatePicker / TimePicker / DateTimePicker 共用）。
 *
 * 约定：日期键 'YYYY-MM-DD'、时刻 'HH:mm'、日期时间 'YYYY-MM-DDTHH:mm'，一律按**浏览器本地时区**
 * 拼接/解析——不用 toISOString（UTC 偏移会让日期错一天）。提交后端时再统一转 ISO。
 */

export const WEEKDAY_CN = ['日', '一', '二', '三', '四', '五', '六'];

export function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

/** 本地时区拼接 YYYY-MM-DD（m 为 0 基月份）。 */
export function toKey(y: number, m: number, d: number): string {
  return `${y}-${pad2(m + 1)}-${pad2(d)}`;
}

export function parseKey(key: string): { y: number; m: number; d: number } | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(key);
  if (!match) return null;
  return { y: Number(match[1]), m: Number(match[2]) - 1, d: Number(match[3]) };
}

export function dateToKey(d: Date): string {
  return toKey(d.getFullYear(), d.getMonth(), d.getDate());
}

export function todayKey(): string {
  return dateToKey(new Date());
}

export function parseTime(t: string): { h: number; mi: number } | null {
  const match = /^(\d{2}):(\d{2})$/.exec(t);
  if (!match) return null;
  const h = Number(match[1]);
  const mi = Number(match[2]);
  if (h > 23 || mi > 59) return null;
  return { h, mi };
}

export function dateToTime(d: Date): string {
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

/** 'YYYY-MM-DD' + 'HH:mm' → 本地 Date；任一非法返回 null。 */
export function combineLocal(dateKey: string, time: string): Date | null {
  const p = parseKey(dateKey);
  const t = parseTime(time);
  if (!p || !t) return null;
  return new Date(p.y, p.m, p.d, t.h, t.mi, 0, 0);
}

/** 'YYYY-MM-DDTHH:mm' → 本地 Date。 */
export function parseLocalDateTime(v: string): Date | null {
  const [d, t] = v.split('T');
  return d && t ? combineLocal(d, t) : null;
}

export function toLocalDateTime(d: Date): string {
  return `${dateToKey(d)}T${dateToTime(d)}`;
}

/** 该月 1 日是周几（0=周一...6=周日），日历网格起点对齐用。 */
export function firstWeekdayMon0(y: number, m: number): number {
  return (new Date(y, m, 1).getDay() + 6) % 7;
}

export function daysInMonth(y: number, m: number): number {
  return new Date(y, m + 1, 0).getDate();
}

/** 把时间向上取整到 step 分钟（秒/毫秒清零）。 */
export function ceilToStep(d: Date, stepMin: number): Date {
  const out = new Date(d);
  out.setSeconds(0, 0);
  const rem = out.getMinutes() % stepMin;
  if (rem !== 0 || d.getSeconds() > 0 || d.getMilliseconds() > 0) {
    out.setMinutes(out.getMinutes() + (stepMin - rem || stepMin));
  }
  return out;
}

/** 两个日期相差的自然日数（按本地日期，忽略时刻）。 */
export function dayDiff(from: Date, to: Date): number {
  const a = new Date(from.getFullYear(), from.getMonth(), from.getDate()).getTime();
  const b = new Date(to.getFullYear(), to.getMonth(), to.getDate()).getTime();
  return Math.round((b - a) / 86_400_000);
}

/** 「今天 / 明天 / 后天 / 9月22日」这类相对日期标签。 */
export function relativeDayLabel(d: Date, now = new Date()): string {
  const diff = dayDiff(now, d);
  if (diff === 0) return '今天';
  if (diff === 1) return '明天';
  if (diff === 2) return '后天';
  if (diff === -1) return '昨天';
  const sameYear = d.getFullYear() === now.getFullYear();
  return sameYear ? `${d.getMonth() + 1}月${d.getDate()}日` : `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

/** 「9月22日 周二 10:00」；跨年时带年份。 */
export function formatDateTimeCN(d: Date, now = new Date()): string {
  const y = d.getFullYear() === now.getFullYear() ? '' : `${d.getFullYear()}年`;
  return `${y}${d.getMonth() + 1}月${d.getDate()}日 周${WEEKDAY_CN[d.getDay()]} ${dateToTime(d)}`;
}

/** 距离现在多久（未来）：「3 分钟后 / 2 小时后 / 5 天后」；已过去返回「等待触发」（cron 下一轮扫描即发）。 */
export function timeUntil(d: Date, now = new Date()): string {
  const ms = d.getTime() - now.getTime();
  if (ms <= 0) return '等待触发';
  const min = Math.round(ms / 60_000);
  if (min < 60) return `${Math.max(min, 1)} 分钟后`;
  const h = Math.round(min / 60);
  if (h < 24) return `${h} 小时后`;
  return `${Math.round(h / 24)} 天后`;
}

/** 浏览器本地时区的 GMT 偏移标签（如 GMT+8）。 */
export function localTzLabel(): string {
  const off = -new Date().getTimezoneOffset();
  const sign = off >= 0 ? '+' : '-';
  const h = Math.floor(Math.abs(off) / 60);
  const m = Math.abs(off) % 60;
  return `GMT${sign}${h}${m ? `:${pad2(m)}` : ''}`;
}
