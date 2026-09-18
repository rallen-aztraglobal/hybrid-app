/**
 * 自绘日期时间选择器（替代原生 <input type="datetime-local">）。
 *
 * 弹层 = 左侧月历（CalendarPanel）+ 右侧「时/分」滚动列（TimeColumns）+ 底部快捷项。
 * value 为本地时区 'YYYY-MM-DDTHH:mm'（'' = 未选）；min 为最早可选时刻（Date），
 * 早于它的日期/时刻置灰，选日期时若时刻落到 min 之前会自动抬到最早可选值。
 * 选择即生效（不需要「确定」），点「完成」/ 外部 / Esc 收起。
 */
import { cn } from '@/lib/cn';
import {
  ceilToStep,
  combineLocal,
  dateToKey,
  dateToTime,
  formatDateTimeCN,
  parseLocalDateTime,
  timeUntil,
  toLocalDateTime,
} from '@/lib/dateTime';
import { usePopover } from '@/hooks/usePopover';
import { CalendarPanel } from './DatePicker';
import { TimeColumns } from './TimePicker';
import { CalendarIcon } from './icons';

interface DateTimePickerProps {
  value: string;
  onChange: (value: string) => void;
  /** 最早可选时刻（含）。 */
  min?: Date;
  /** 分钟步长，默认 5（推送类场景无需精确到分）。 */
  minuteStep?: number;
  placeholder?: string;
  className?: string;
  invalid?: boolean;
  /** 底部快捷项，默认给「1 小时后 / 今晚 20:00 / 明早 10:00」。传 [] 关闭。 */
  presets?: { label: string; get: () => Date }[];
}

const DEFAULT_PRESETS: { label: string; get: () => Date }[] = [
  { label: '1 小时后', get: () => new Date(Date.now() + 3_600_000) },
  {
    label: '今晚 20:00',
    get: () => {
      const d = new Date();
      d.setHours(20, 0, 0, 0);
      return d;
    },
  },
  {
    label: '明早 10:00',
    get: () => {
      const d = new Date();
      d.setDate(d.getDate() + 1);
      d.setHours(10, 0, 0, 0);
      return d;
    },
  },
];

export function DateTimePicker({
  value,
  onChange,
  min,
  minuteStep = 5,
  placeholder,
  className,
  invalid,
  presets = DEFAULT_PRESETS,
}: DateTimePickerProps) {
  const { open, setOpen, wrapRef, popRef, popClass } = usePopover();
  const current = parseLocalDateTime(value);
  const minAligned = min ? ceilToStep(min, minuteStep) : null;
  const minKey = minAligned ? dateToKey(minAligned) : undefined;
  const dateKey = current ? dateToKey(current) : '';
  const time = current ? dateToTime(current) : '';

  /** 统一出口：对齐步长 + 不早于 min。 */
  function commit(d: Date) {
    let out = ceilToStep(d, minuteStep);
    if (minAligned && out < minAligned) out = minAligned;
    onChange(toLocalDateTime(out));
  }

  function pickDate(key: string) {
    const d = combineLocal(key, time || (minAligned && key === minKey ? dateToTime(minAligned) : '10:00'));
    if (d) commit(d);
  }

  function pickTime(t: string) {
    const key = dateKey || minKey || dateToKey(new Date());
    const d = combineLocal(key, t);
    if (d) commit(d);
  }

  const now = new Date();

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={cn(
          'field-input flex items-center gap-2 text-left cursor-pointer',
          open && 'border-brand',
          invalid && 'border-down',
          className,
        )}
      >
        <CalendarIcon className="w-4 h-4 flex-none text-muted" />
        <span className={cn('flex-1 truncate', !current && 'text-muted')}>
          {current ? formatDateTimeCN(current, now) : placeholder || '选择日期和时间'}
        </span>
        {current && <span className="flex-none text-[11px] text-muted">{timeUntil(current, now)}</span>}
      </button>

      {open && (
        <div
          ref={popRef}
          role="dialog"
          className={cn(popClass, 'rounded-[12px] border bg-white p-3 shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)]')}
          style={{ borderColor: 'var(--line)' }}
        >
          <div className="flex gap-3">
            <CalendarPanel value={dateKey} onPick={pickDate} min={minKey} />
            <div className="w-px bg-line-2 flex-none" />
            <TimeColumns
              value={time}
              onChange={pickTime}
              minuteStep={minuteStep}
              min={minAligned && dateKey === minKey ? dateToTime(minAligned) : undefined}
            />
          </div>
          <div className="flex items-center gap-1.5 mt-3 pt-2.5 border-t flex-wrap" style={{ borderColor: 'var(--line)' }}>
            {presets.map((p) => {
              const d = ceilToStep(p.get(), minuteStep);
              const disabled = !!minAligned && d < minAligned;
              return (
                <button
                  key={p.label}
                  type="button"
                  disabled={disabled}
                  onClick={() => commit(d)}
                  className="text-[11.5px] px-2 py-1 rounded-[7px] border border-line text-ink-2 hover:border-brand hover:text-brand transition disabled:opacity-35 disabled:hover:border-line disabled:hover:text-ink-2"
                >
                  {p.label}
                </button>
              );
            })}
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="ml-auto text-[12px] font-semibold px-3 py-1 rounded-[7px] text-white"
              style={{ background: 'var(--brand)' }}
            >
              完成
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
