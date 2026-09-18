/**
 * 推送定时规则编辑器（渠道推送 / 上架包推送共用）。
 *
 *   重复方式  [单次] [每天] [每 N 天]
 *   单次 → 发送时间（DateTimePicker）
 *   周期 → 开始日期（DatePicker）+ 发送时刻（TimePicker）+ 间隔 N（仅每 N 天）+ 结束条件
 *   底部摘要：一句话描述 + 接下来几次的具体时刻，发之前一眼确认「到底哪天几点发」。
 *
 * 纯受控组件：值是 PushScheduleDraft，校验规则见 lib/pushSchedule.ts；
 * showErrors 由父组件在首次提交后置 true（未提交前不打扰用户）。
 */
import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';
import { dateToTime, formatDateTimeCN, localTzLabel, relativeDayLabel, todayKey, WEEKDAY_CN } from '@/lib/dateTime';
import {
  describeDraft,
  everyDaysOf,
  MAX_EVERY_DAYS,
  MAX_RUNS,
  repeatLabel,
  switchRepeat,
  totalRunsOf,
  upcomingRuns,
  validateSchedule,
  type EndMode,
  type PushScheduleDraft,
  type RepeatMode,
} from '@/lib/pushSchedule';
import { DatePicker } from './DatePicker';
import { DateTimePicker } from './DateTimePicker';
import { TimePicker } from './TimePicker';
import { ClockIcon } from './icons';

const MINUTE_STEP = 5;

// ── 分段选择（与页面顶部「渠道推送 / 上架包推送」切换同一视觉语言，缩小一号） ──
function Segmented<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: readonly (readonly [T, string])[];
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex p-[3px] rounded-[10px] bg-panel-2 border border-line gap-[3px]" role="radiogroup">
      {options.map(([key, label]) => (
        <button
          key={key}
          type="button"
          role="radio"
          aria-checked={value === key}
          onClick={() => onChange(key)}
          className={cn(
            'flex-1 px-2 py-[6px] rounded-[8px] text-[12.5px] font-semibold transition whitespace-nowrap',
            value === key ? 'bg-panel text-ink shadow-sm2' : 'text-muted hover:text-ink',
          )}
        >
          {label}
        </button>
      ))}
    </div>
  );
}

// ── 数字步进器（− [n] +），输入框可直接键入，失焦时钳到 [min,max] ─────────────
function NumberStepper({
  value,
  onChange,
  min,
  max,
  suffix,
  invalid,
}: {
  value: number;
  onChange: (n: number) => void;
  min: number;
  max: number;
  suffix: string;
  invalid?: boolean;
}) {
  const clamp = (n: number) => Math.min(max, Math.max(min, Math.round(n)));
  const btn =
    'grid place-items-center w-8 h-[38px] text-[16px] text-ink-2 hover:bg-bg hover:text-ink transition disabled:opacity-30 disabled:hover:bg-transparent';
  return (
    <div className="flex items-center gap-2">
      <div
        className={cn(
          'inline-flex items-stretch rounded-[10px] border bg-panel-2 overflow-hidden',
          invalid ? 'border-down' : 'border-line',
        )}
      >
        <button type="button" className={btn} disabled={value <= min} onClick={() => onChange(clamp(value - 1))} aria-label="减少">
          −
        </button>
        <input
          type="text"
          inputMode="numeric"
          className="w-12 text-center text-[13px] font-semibold font-mono bg-transparent border-x border-line outline-none focus:bg-white"
          value={Number.isNaN(value) ? '' : String(value)}
          onChange={(e) => {
            const digits = e.target.value.replace(/\D/g, '').slice(0, 4);
            onChange(digits ? Number(digits) : NaN);
          }}
          onBlur={() => onChange(clamp(Number.isNaN(value) ? min : value))}
        />
        <button type="button" className={btn} disabled={value >= max} onClick={() => onChange(clamp(value + 1))} aria-label="增加">
          +
        </button>
      </div>
      <span className="text-[13px] text-ink-2">{suffix}</span>
    </div>
  );
}

function FieldLabel({ children, required }: { children: ReactNode; required?: boolean }) {
  return (
    <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
      {children}
      {required && <span className="text-down"> *</span>}
    </label>
  );
}

function FieldError({ msg }: { msg?: string }) {
  return msg ? <div className="mt-1 text-[12px] text-down">{msg}</div> : null;
}

const REPEAT_OPTIONS = [
  ['once', '单次'],
  ['daily', '每天'],
  ['interval', '每 N 天'],
] as const satisfies readonly (readonly [RepeatMode, string])[];

const END_OPTIONS = [
  ['never', '不结束'],
  ['until', '截止日期'],
  ['count', '执行次数'],
] as const satisfies readonly (readonly [EndMode, string])[];

export function PushScheduleEditor({
  value,
  onChange,
  showErrors,
}: {
  value: PushScheduleDraft;
  onChange: (next: PushScheduleDraft) => void;
  showErrors?: boolean;
}) {
  const now = new Date();
  const set = (patch: Partial<PushScheduleDraft>) => onChange({ ...value, ...patch });
  const err = showErrors ? validateSchedule(value, now) : null;
  const errOf = (f: string) => (err?.field === f ? err.message : undefined);

  const recurring = value.repeat !== 'once';
  const today = todayKey();
  // 开始日期选今天时，发送时刻不能早于「现在」（向上对齐到步长）
  const minTimeToday =
    value.startDate === today
      ? dateToTime(new Date(Math.ceil((now.getTime() + 60_000) / (MINUTE_STEP * 60_000)) * MINUTE_STEP * 60_000))
      : undefined;

  const runs = upcomingRuns(value, 4);
  const total = totalRunsOf(value);
  const summary = describeDraft(value, now);

  return (
    <div className="flex flex-col gap-[14px]">
      <div>
        <FieldLabel>重复方式</FieldLabel>
        <Segmented value={value.repeat} options={REPEAT_OPTIONS} onChange={(r) => onChange(switchRepeat(value, r))} />
      </div>

      {!recurring ? (
        <div>
          <FieldLabel required>发送时间</FieldLabel>
          <DateTimePicker
            value={value.onceAt}
            onChange={(v) => set({ onceAt: v })}
            min={now}
            minuteStep={MINUTE_STEP}
            invalid={!!errOf('first')}
          />
          <FieldError msg={errOf('first')} />
        </div>
      ) : (
        <>
          {value.repeat === 'interval' && (
            <div>
              <FieldLabel required>间隔</FieldLabel>
              <div className="flex items-center gap-2 flex-wrap">
                <NumberStepper
                  value={value.everyDays}
                  onChange={(n) => set({ everyDays: n })}
                  min={2}
                  max={MAX_EVERY_DAYS}
                  suffix="天发送一次"
                  invalid={!!errOf('everyDays')}
                />
              </div>
              <div className="flex gap-1.5 mt-2">
                {[2, 3, 7, 14].map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => set({ everyDays: n })}
                    className={cn(
                      'text-[11.5px] px-2 py-[3px] rounded-[7px] border transition',
                      value.everyDays === n
                        ? 'border-brand text-brand bg-[rgba(99,102,241,.06)] font-semibold'
                        : 'border-line text-ink-2 hover:border-brand hover:text-brand',
                    )}
                  >
                    {n === 7 ? '每周' : n === 14 ? '每两周' : `${n} 天`}
                  </button>
                ))}
              </div>
              <FieldError msg={errOf('everyDays')} />
            </div>
          )}

          <div className="grid grid-cols-2 gap-2.5">
            <div className="min-w-0">
              <FieldLabel required>开始日期</FieldLabel>
              <DatePicker
                value={value.startDate}
                onChange={(v) => set({ startDate: v })}
                min={today}
                clearable={false}
                placeholder="选择日期"
                className={errOf('first') ? 'border-down' : undefined}
                formatDisplay={(k) => {
                  const d = new Date(`${k}T00:00:00`);
                  return `${relativeDayLabel(d, now)} 周${WEEKDAY_CN[d.getDay()]}`;
                }}
              />
            </div>
            <div className="min-w-0">
              <FieldLabel required>发送时刻</FieldLabel>
              <TimePicker
                value={value.time}
                onChange={(v) => set({ time: v })}
                minuteStep={MINUTE_STEP}
                min={minTimeToday}
                presets={['09:00', '12:00', '20:00']}
                className={errOf('first') ? 'border-down' : undefined}
              />
            </div>
          </div>
          <FieldError msg={errOf('first')} />

          <div>
            <FieldLabel>结束条件</FieldLabel>
            <Segmented value={value.endMode} options={END_OPTIONS} onChange={(m) => set({ endMode: m })} />
            {value.endMode === 'until' && (
              <div className="mt-2">
                <DatePicker
                  value={value.endDate}
                  onChange={(v) => set({ endDate: v })}
                  min={value.startDate || today}
                  clearable={false}
                  placeholder="选择截止日期（含当天）"
                  className={errOf('endDate') ? 'border-down' : undefined}
                />
                <FieldError msg={errOf('endDate')} />
              </div>
            )}
            {value.endMode === 'count' && (
              <div className="mt-2">
                <NumberStepper
                  value={value.maxRuns}
                  onChange={(n) => set({ maxRuns: n })}
                  min={1}
                  max={MAX_RUNS}
                  suffix="次后自动结束"
                  invalid={!!errOf('maxRuns')}
                />
                <FieldError msg={errOf('maxRuns')} />
              </div>
            )}
          </div>
        </>
      )}

      {/* 摘要 + 接下来几次：发之前一眼看清到底哪天几点发 */}
      {summary && (
        <div className="rounded-[10px] border border-[rgba(99,102,241,.2)] bg-[rgba(99,102,241,.05)] px-3 py-2.5">
          <div className="flex items-start gap-2 text-[12.5px] text-ink leading-relaxed">
            <ClockIcon className="w-4 h-4 text-brand flex-none mt-[2px]" />
            <span>{summary}</span>
          </div>
          {recurring && runs.length > 0 && (
            <ol className="mt-2 ml-6 flex flex-col gap-1">
              {runs.map((d, i) => (
                <li key={d.getTime()} className="flex items-center gap-2 text-[12px]">
                  <span
                    className={cn('w-1.5 h-1.5 rounded-full flex-none', i === 0 ? 'bg-brand' : 'bg-[#c7cbe0]')}
                  />
                  <span className={cn('tabular-nums', i === 0 ? 'text-ink font-semibold' : 'text-ink-2')}>
                    {formatDateTimeCN(d, now)}
                  </span>
                  <span className="text-muted text-[11px]">第 {i + 1} 次</span>
                </li>
              ))}
              {(total == null || total > runs.length) && (
                <li className="text-[11.5px] text-muted pl-3.5">
                  {total == null ? `… 之后${repeatLabel(everyDaysOf(value))}继续，直到手动停止` : `… 共 ${total} 次`}
                </li>
              )}
            </ol>
          )}
          <div className="mt-1.5 ml-6 text-[11px] text-muted">按本机时区（{localTzLabel()}）；服务端每分钟扫描一次，最多延迟约 1 分钟。</div>
        </div>
      )}
    </div>
  );
}
