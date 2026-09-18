/**
 * 推送定时规则（渠道推送 / 上架包推送共用）：单次 / 每天 / 每 N 天 + 结束条件。
 *
 * 与后端契约（POST /push/campaigns/:id/schedule、/push/listing-campaigns/:id/schedule）：
 *   scheduledAt     首次发送时刻（ISO；周期任务之后表示「下次运行时间」）
 *   repeatEveryDays 0=单次，1=每天，N=每 N 天
 *   repeatEndAt     截止时刻（含）；前端取截止日期当天 23:59:59（本地时区）
 *   repeatMaxRuns   最多执行次数，0=不限
 * 周期任务在服务端按「首次时刻 + k×N 天」推进，每次触发派生一条子活动去发。
 */
import {
  ceilToStep,
  combineLocal,
  dateToKey,
  dateToTime,
  formatDateTimeCN,
  parseKey,
  parseLocalDateTime,
  toLocalDateTime,
} from './dateTime';

export type RepeatMode = 'once' | 'daily' | 'interval';
/** 定时任务控制动作（POST /push/campaigns/:id/<action>）。 */
export type PushScheduleAction = 'pause' | 'resume' | 'cancel';
export type EndMode = 'never' | 'until' | 'count';

export interface PushScheduleDraft {
  repeat: RepeatMode;
  /** 单次：'YYYY-MM-DDTHH:mm'。 */
  onceAt: string;
  /** 周期：首次日期 'YYYY-MM-DD'。 */
  startDate: string;
  /** 周期：每次发送时刻 'HH:mm'。 */
  time: string;
  /** interval 模式的 N（2..365）。 */
  everyDays: number;
  endMode: EndMode;
  /** endMode=until：截止日期 'YYYY-MM-DD'（含当天）。 */
  endDate: string;
  /** endMode=count：共执行几次。 */
  maxRuns: number;
}

export interface PushSchedulePayload {
  scheduledAt: string;
  repeatEveryDays: number;
  repeatEndAt?: string;
  repeatMaxRuns: number;
}

export const MAX_EVERY_DAYS = 365;
export const MAX_RUNS = 1000;
/** 定时至少要比当前晚这么多（给 cron 每分钟扫描留余量）。 */
const MIN_LEAD_MS = 60_000;

/** 默认：半小时后的下一个整点；周期默认每 3 天 / 不结束 / 次数模式预填 7 次。 */
export function defaultScheduleDraft(now = new Date()): PushScheduleDraft {
  const first = ceilToStep(new Date(now.getTime() + 30 * 60_000), 60);
  return {
    repeat: 'once',
    onceAt: toLocalDateTime(first),
    startDate: dateToKey(first),
    time: dateToTime(first),
    everyDays: 3,
    endMode: 'never',
    endDate: '',
    maxRuns: 7,
  };
}

export function everyDaysOf(d: PushScheduleDraft): number {
  if (d.repeat === 'once') return 0;
  if (d.repeat === 'daily') return 1;
  return d.everyDays;
}

export function firstRunOf(d: PushScheduleDraft): Date | null {
  return d.repeat === 'once' ? parseLocalDateTime(d.onceAt) : combineLocal(d.startDate, d.time);
}

function addDays(d: Date, n: number): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() + n, d.getHours(), d.getMinutes());
}

function endOfDay(key: string): Date | null {
  const p = parseKey(key);
  return p ? new Date(p.y, p.m, p.d, 23, 59, 59) : null;
}

/** 草稿里切换重复模式时，把单次/周期两套字段互相同步，避免切过去是空的。 */
export function switchRepeat(d: PushScheduleDraft, repeat: RepeatMode): PushScheduleDraft {
  if (repeat === d.repeat) return d;
  if (d.repeat === 'once') {
    const first = parseLocalDateTime(d.onceAt);
    return first ? { ...d, repeat, startDate: dateToKey(first), time: dateToTime(first) } : { ...d, repeat };
  }
  if (repeat === 'once') {
    const first = combineLocal(d.startDate, d.time);
    return first ? { ...d, repeat, onceAt: toLocalDateTime(first) } : { ...d, repeat };
  }
  return { ...d, repeat };
}

export interface ScheduleError {
  field: 'first' | 'everyDays' | 'endDate' | 'maxRuns';
  message: string;
}

export function validateSchedule(d: PushScheduleDraft, now = new Date()): ScheduleError | null {
  const first = firstRunOf(d);
  if (!first) {
    return { field: 'first', message: d.repeat === 'once' ? '请选择发送时间' : '请选择开始日期和发送时刻' };
  }
  if (first.getTime() < now.getTime() + MIN_LEAD_MS) {
    return {
      field: 'first',
      message: d.repeat === 'once' ? '发送时间须晚于当前时间' : '首次发送时刻已过，请改为稍晚的时刻或之后的日期',
    };
  }
  if (d.repeat === 'interval' && (!Number.isInteger(d.everyDays) || d.everyDays < 2 || d.everyDays > MAX_EVERY_DAYS)) {
    return { field: 'everyDays', message: `间隔天数须在 2 ~ ${MAX_EVERY_DAYS} 之间` };
  }
  if (d.repeat !== 'once') {
    if (d.endMode === 'until') {
      if (!d.endDate) return { field: 'endDate', message: '请选择截止日期' };
      if (d.endDate < d.startDate) return { field: 'endDate', message: '截止日期不能早于开始日期' };
    }
    if (d.endMode === 'count' && (!Number.isInteger(d.maxRuns) || d.maxRuns < 1 || d.maxRuns > MAX_RUNS)) {
      return { field: 'maxRuns', message: `执行次数须在 1 ~ ${MAX_RUNS} 之间` };
    }
  }
  return null;
}

export function toSchedulePayload(d: PushScheduleDraft): PushSchedulePayload {
  const first = firstRunOf(d)!;
  const every = everyDaysOf(d);
  const payload: PushSchedulePayload = { scheduledAt: first.toISOString(), repeatEveryDays: every, repeatMaxRuns: 0 };
  if (every > 0) {
    if (d.endMode === 'until') payload.repeatEndAt = endOfDay(d.endDate)?.toISOString();
    if (d.endMode === 'count') payload.repeatMaxRuns = d.maxRuns;
  }
  return payload;
}

/** 总执行次数（不结束 → null）。 */
export function totalRunsOf(d: PushScheduleDraft): number | null {
  const every = everyDaysOf(d);
  if (every === 0) return 1;
  if (d.endMode === 'count') return d.maxRuns;
  if (d.endMode === 'until') {
    const first = firstRunOf(d);
    const end = endOfDay(d.endDate);
    if (!first || !end || end < first) return 0;
    let n = 0;
    for (let t = first; t <= end && n <= MAX_RUNS; t = addDays(t, every)) n++;
    return n;
  }
  return null;
}

/** 接下来的最多 limit 次发送时刻（受结束条件约束）。 */
export function upcomingRuns(d: PushScheduleDraft, limit = 4): Date[] {
  const first = firstRunOf(d);
  const every = everyDaysOf(d);
  if (!first || !Number.isFinite(every)) return []; // 步进器输入中途可能是 NaN
  if (every === 0) return [first];
  const total = totalRunsOf(d);
  const n = total == null ? limit : Math.min(limit, total);
  return Array.from({ length: n }, (_, i) => addDays(first, i * every));
}

/** 最后一次发送时刻（不结束 → null）。 */
export function lastRunOf(d: PushScheduleDraft): Date | null {
  const first = firstRunOf(d);
  const total = totalRunsOf(d);
  if (!first || !total) return null;
  return addDays(first, (total - 1) * everyDaysOf(d));
}

/** 「每天 10:00」「每 3 天 10:00」「单次」。 */
export function repeatLabel(everyDays: number, at?: Date | null): string {
  const t = at ? ` ${dateToTime(at)}` : '';
  if (!everyDays) return '单次';
  return everyDays === 1 ? `每天${t}` : `每 ${everyDays} 天${t}`;
}

/** 编辑器底部的一句话摘要。 */
export function describeDraft(d: PushScheduleDraft, now = new Date()): string {
  const first = firstRunOf(d);
  const every = everyDaysOf(d);
  if (!first || !Number.isFinite(every)) return '';
  if (every === 0) return `将于 ${formatDateTimeCN(first, now)} 发送 1 次`;
  const parts = [`从 ${formatDateTimeCN(first, now)} 起，${repeatLabel(every, first)} 发送`];
  const total = totalRunsOf(d);
  const last = lastRunOf(d);
  if (total == null) parts.push('直到手动停止');
  else if (last) parts.push(`共 ${total} 次，最后一次 ${formatDateTimeCN(last, now)}`);
  return parts.join('，');
}

// ── 活动（后端返回）侧的展示辅助 ─────────────────────────────────────────────

interface RepeatFields {
  status: string;
  scheduledAt?: string;
  repeatEveryDays?: number;
  repeatEndAt?: string;
  repeatMaxRuns?: number;
  runCount?: number;
}

export function isRecurring(c: RepeatFields): boolean {
  return (c.repeatEveryDays ?? 0) > 0;
}

/** 仍在等待触发的定时任务（单次待发 / 周期进行中 / 周期暂停）。 */
export function isActiveSchedule(c: RepeatFields): boolean {
  return c.status === 'scheduled' || c.status === 'paused';
}

/** 「已执行 2 / 5 次」「已执行 2 次 · 截止 10月1日」「已执行 0 次 · 不限」。 */
export function runProgressLabel(c: RepeatFields): string {
  const ran = c.runCount ?? 0;
  if (c.repeatMaxRuns) return `已执行 ${ran} / ${c.repeatMaxRuns} 次`;
  if (c.repeatEndAt) {
    const e = new Date(c.repeatEndAt);
    return `已执行 ${ran} 次 · 截止 ${e.getMonth() + 1}月${e.getDate()}日`;
  }
  return `已执行 ${ran} 次 · 不限期`;
}

/**
 * 纯函数：从 prev 起按 everyDays 步进，返回第一个晚于 now 的时刻（resume / mock 用，
 * 与服务端 nextRunAfter 语义一致：漏掉的周期不补发）。
 */
export function nextRunAfter(prev: Date, now: Date, everyDays: number): Date {
  let t = addDays(prev, everyDays);
  while (t.getTime() <= now.getTime()) t = addDays(t, everyDays);
  return t;
}

/** 对活动应用 pause/resume/cancel 状态机（mock 与服务端规则一致），非法迁移抛错。 */
export function applyScheduleAction<T extends RepeatFields>(c: T, action: PushScheduleAction, now = new Date()): T {
  const every = c.repeatEveryDays ?? 0;
  if (action === 'pause') {
    if (c.status !== 'scheduled' || every === 0) throw new Error('只有进行中的周期任务可以暂停');
    return { ...c, status: 'paused' };
  }
  if (action === 'cancel') {
    if (c.status !== 'scheduled' && c.status !== 'paused') throw new Error('只有等待中的定时任务可以取消');
    return { ...c, status: 'cancelled' };
  }
  if (c.status !== 'paused') throw new Error('只有已暂停的任务可以恢复');
  let next = c.scheduledAt ? new Date(c.scheduledAt) : now;
  if (next.getTime() <= now.getTime()) next = nextRunAfter(next, now, every);
  const ran = c.runCount ?? 0;
  if ((c.repeatMaxRuns && ran >= c.repeatMaxRuns) || (c.repeatEndAt && next > new Date(c.repeatEndAt))) {
    throw new Error('周期已结束，无剩余执行');
  }
  return { ...c, status: 'scheduled', scheduledAt: next.toISOString() };
}
