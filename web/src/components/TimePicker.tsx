/**
 * 自绘时刻选择器（替代原生 <input type="time">）：「时 / 分」双滚动列，风格对齐 DatePicker。
 * value 为本地时刻 'HH:mm'。
 *
 * TimeColumns 单独导出，DateTimePicker 把它和 CalendarPanel 拼在同一个弹层里。
 */
import { useLayoutEffect, useRef } from 'react';
import { cn } from '@/lib/cn';
import { pad2, parseTime } from '@/lib/dateTime';
import { usePopover } from '@/hooks/usePopover';
import { ClockIcon } from './icons';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

// ── 单列（选中项滚到列中间；只滚列容器本身，不牵动页面滚动） ─────────────────
function Column({
  label,
  items,
  selected,
  isDisabled,
  onPick,
}: {
  label: string;
  items: number[];
  selected: number | null;
  isDisabled: (n: number) => boolean;
  onPick: (n: number) => void;
}) {
  const listRef = useRef<HTMLDivElement>(null);
  const mountedRef = useRef(false);

  useLayoutEffect(() => {
    const list = listRef.current;
    if (!list || selected == null) return;
    const el = list.querySelector<HTMLElement>(`[data-v="${selected}"]`);
    if (!el) return;
    // 首次打开直接定位，之后切换再平滑滚动
    list.scrollTo({
      top: el.offsetTop - list.clientHeight / 2 + el.clientHeight / 2,
      behavior: mountedRef.current ? 'smooth' : 'auto',
    });
    mountedRef.current = true;
  }, [selected]);

  return (
    <div className="flex flex-col w-[54px]">
      <div className="text-center text-[11px] text-muted font-medium mb-1">{label}</div>
      <div
        ref={listRef}
        className="relative h-[220px] overflow-y-auto overscroll-contain flex flex-col gap-0.5 pr-0.5 [scrollbar-width:thin]"
      >
        {items.map((n) => {
          const sel = n === selected;
          const disabled = isDisabled(n);
          return (
            <button
              type="button"
              key={n}
              data-v={n}
              disabled={disabled}
              onClick={() => onPick(n)}
              className={cn(
                'flex-none h-7 rounded-[7px] text-[12.5px] font-mono tabular-nums transition',
                sel ? 'text-white font-semibold' : 'text-ink hover:bg-panel-2',
                disabled && 'opacity-30 cursor-not-allowed hover:bg-transparent',
              )}
              style={sel ? { background: 'var(--brand)' } : undefined}
            >
              {pad2(n)}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export interface TimeColumnsProps {
  value: string;
  onChange: (value: string) => void;
  /** 分钟步长（默认 1）。 */
  minuteStep?: number;
  /** 可选最早时刻 'HH:mm'（当天已过去的时刻置灰）。 */
  min?: string;
}

/** 时 / 分双列面板。选了小时但分钟被 min 卡住时，自动把分钟抬到最早可选值。 */
export function TimeColumns({ value, onChange, minuteStep = 1, min }: TimeColumnsProps) {
  const cur = parseTime(value);
  const minT = min ? parseTime(min) : null;
  const minutes = Array.from({ length: Math.ceil(60 / minuteStep) }, (_, i) => i * minuteStep);

  const hourDisabled = (h: number) => {
    if (!minT) return false;
    if (h < minT.h) return true;
    // 该小时内所有步长分钟都早于 min 时也置灰
    return h === minT.h && !minutes.some((m) => m >= minT.mi);
  };
  const minuteDisabled = (m: number) => !!minT && !!cur && cur.h === minT.h && m < minT.mi;

  function firstMinuteFor(h: number, preferred: number): number {
    if (minT && h === minT.h && preferred < minT.mi) return minutes.find((m) => m >= minT.mi) ?? preferred;
    return preferred;
  }

  return (
    <div className="flex gap-1.5 justify-center">
      <Column
        label="时"
        items={HOURS}
        selected={cur?.h ?? null}
        isDisabled={hourDisabled}
        onPick={(h) => {
          // 分钟对齐到步长，避免选了 13 点后仍残留 07 这种不在列表里的值
          const mi = cur ? Math.floor(cur.mi / minuteStep) * minuteStep : 0;
          onChange(`${pad2(h)}:${pad2(firstMinuteFor(h, mi))}`);
        }}
      />
      <Column
        label="分"
        items={minutes}
        selected={cur ? cur.mi : null}
        isDisabled={minuteDisabled}
        onPick={(mi) => onChange(`${pad2(cur?.h ?? (minT?.h ?? 0))}:${pad2(mi)}`)}
      />
    </div>
  );
}

interface TimePickerProps extends TimeColumnsProps {
  placeholder?: string;
  className?: string;
  /** 弹层底部的快捷时刻（如 ['09:00','12:00','20:00']）。 */
  presets?: string[];
}

export function TimePicker({ value, onChange, minuteStep = 1, min, placeholder, className, presets }: TimePickerProps) {
  const { open, setOpen, wrapRef, popRef, popClass } = usePopover();

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={cn('field-input flex items-center gap-2 text-left cursor-pointer', open && 'border-brand', className)}
      >
        <ClockIcon className="w-4 h-4 flex-none text-muted" />
        <span className={cn('flex-1 truncate font-mono tabular-nums', !value && 'text-muted font-sans')}>
          {value || placeholder || '选择时间'}
        </span>
      </button>
      {open && (
        <div
          ref={popRef}
          className={cn(popClass, 'w-[236px] rounded-[10px] border bg-white p-3 shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)]')}
          style={{ borderColor: 'var(--line)' }}
        >
          <TimeColumns value={value} onChange={onChange} minuteStep={minuteStep} min={min} />
          <div className="flex items-center gap-1 mt-2 pt-2 border-t" style={{ borderColor: 'var(--line)' }}>
            {presets?.map((p) => {
              const disabled = !!min && p < min;
              return (
                <button
                  key={p}
                  type="button"
                  disabled={disabled}
                  onClick={() => onChange(p)}
                  className={cn(
                    'text-[11.5px] font-mono px-1.5 py-0.5 rounded-[6px] hover:bg-panel-2 disabled:opacity-30',
                    p === value ? 'text-brand font-semibold' : 'text-ink-2',
                  )}
                >
                  {p}
                </button>
              );
            })}
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="ml-auto text-[12px] font-semibold px-2 py-1 rounded-[6px] hover:bg-panel-2"
              style={{ color: 'var(--brand)' }}
            >
              完成
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
