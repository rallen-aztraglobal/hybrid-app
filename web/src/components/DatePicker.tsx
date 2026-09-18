/**
 * 自绘日期选择器（替代原生 <input type="date">，风格对齐 .field-input + Select 的 popover）。
 * value 用本地时区拼接的 'YYYY-MM-DD'（不用 toISOString，避免时区偏移导致日期错一天）。
 *
 * 月历面板抽成 CalendarPanel 单独导出，供 DateTimePicker 复用同一套网格/禁用/今天高亮逻辑。
 */
import { useState } from 'react';
import { cn } from '@/lib/cn';
import { daysInMonth, firstWeekdayMon0, parseKey, toKey, todayKey } from '@/lib/dateTime';
import { usePopover } from '@/hooks/usePopover';
import { CalendarIcon, CloseIcon } from './icons';

interface DatePickerProps {
  value: string;
  onChange: (value: string) => void;
  min?: string;
  max?: string;
  placeholder?: string;
  className?: string;
  /** 是否显示清除按钮（必填场景关掉），默认 true。 */
  clearable?: boolean;
  /** 自定义触发器里展示的文字（默认直接显示 YYYY-MM-DD）。 */
  formatDisplay?: (value: string) => string;
}

const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];

// ── 月历面板（无外壳，调用方负责弹层容器） ────────────────────────────────────
export function CalendarPanel({
  value,
  onPick,
  min,
  max,
}: {
  value: string;
  onPick: (key: string) => void;
  min?: string;
  max?: string;
}) {
  // 面板随弹层挂载，初始视图定位到已选值（无值则今天）。
  const initial = parseKey(value) ?? parseKey(todayKey())!;
  const [viewY, setViewY] = useState(initial.y);
  const [viewM, setViewM] = useState(initial.m);

  function shiftMonth(delta: number) {
    const d = new Date(viewY, viewM + delta, 1);
    setViewY(d.getFullYear());
    setViewM(d.getMonth());
  }

  function inRange(key: string): boolean {
    if (min && key < min) return false;
    if (max && key > max) return false;
    return true;
  }

  const todayK = todayKey();
  const leadDays = firstWeekdayMon0(viewY, viewM);
  const curDays = daysInMonth(viewY, viewM);
  const prev = new Date(viewY, viewM - 1, 1);
  const next = new Date(viewY, viewM + 1, 1);
  const prevDays = daysInMonth(prev.getFullYear(), prev.getMonth());

  type Cell = { y: number; m: number; d: number; inMonth: boolean };
  const cells: Cell[] = [];
  for (let i = leadDays - 1; i >= 0; i--) {
    cells.push({ y: prev.getFullYear(), m: prev.getMonth(), d: prevDays - i, inMonth: false });
  }
  for (let d = 1; d <= curDays; d++) cells.push({ y: viewY, m: viewM, d, inMonth: true });
  while (cells.length < 42) {
    const idx = cells.length - (leadDays + curDays);
    cells.push({ y: next.getFullYear(), m: next.getMonth(), d: idx + 1, inMonth: false });
  }

  // 已到最小月份时禁用「上一月」，避免翻进整月不可选的空白页。
  const minP = min ? parseKey(min) : null;
  const prevDisabled = !!minP && (viewY < minP.y || (viewY === minP.y && viewM <= minP.m));
  const maxP = max ? parseKey(max) : null;
  const nextDisabled = !!maxP && (viewY > maxP.y || (viewY === maxP.y && viewM >= maxP.m));

  return (
    <div className="w-[252px]">
      <div className="flex items-center justify-between mb-2">
        <button
          type="button"
          onClick={() => shiftMonth(-1)}
          disabled={prevDisabled}
          className="grid place-items-center w-7 h-7 rounded-[8px] text-ink-2 hover:bg-panel-2 disabled:opacity-30 disabled:hover:bg-transparent"
          aria-label="上一月"
        >
          ‹
        </button>
        <span className="text-[13px] font-semibold text-ink">
          {viewY}年{viewM + 1}月
        </span>
        <button
          type="button"
          onClick={() => shiftMonth(1)}
          disabled={nextDisabled}
          className="grid place-items-center w-7 h-7 rounded-[8px] text-ink-2 hover:bg-panel-2 disabled:opacity-30 disabled:hover:bg-transparent"
          aria-label="下一月"
        >
          ›
        </button>
      </div>
      <div className="grid grid-cols-7 gap-y-1 mb-1">
        {WEEKDAYS.map((w) => (
          <span key={w} className="text-center text-[11px] text-muted font-medium">
            {w}
          </span>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-y-1">
        {cells.map((c, i) => {
          const key = toKey(c.y, c.m, c.d);
          const isSel = key === value;
          const isToday = key === todayK;
          const disabled = !inRange(key);
          return (
            <button
              type="button"
              key={i}
              disabled={disabled}
              onClick={() => {
                if (!disabled) onPick(key);
              }}
              className={cn(
                'mx-auto grid place-items-center w-7 h-7 rounded-full text-[12.5px] transition',
                !c.inMonth && 'text-muted/60',
                c.inMonth && !isSel && 'text-ink',
                !disabled && !isSel && 'hover:bg-panel-2',
                disabled && 'opacity-30 cursor-not-allowed',
                isToday && !isSel && 'border',
              )}
              style={
                isSel
                  ? { background: 'var(--brand)', color: '#fff' }
                  : isToday
                    ? { borderColor: 'var(--brand)' }
                    : undefined
              }
            >
              {c.d}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export function DatePicker({
  value,
  onChange,
  min,
  max,
  placeholder,
  className,
  clearable = true,
  formatDisplay,
}: DatePickerProps) {
  const { open, setOpen, wrapRef, popRef, popClass } = usePopover();
  const todayK = todayKey();
  const todayPickable = (!min || todayK >= min) && (!max || todayK <= max);

  function pick(key: string) {
    onChange(key);
    setOpen(false);
  }

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={cn('field-input flex items-center gap-2 text-left cursor-pointer', open && 'border-brand', className)}
      >
        <CalendarIcon className="w-4 h-4 flex-none text-muted" />
        <span className={cn('flex-1 truncate', !value && 'text-muted')}>
          {value ? (formatDisplay ? formatDisplay(value) : value) : placeholder || '选择日期'}
        </span>
        {value && clearable && (
          <span
            role="button"
            aria-label="清除日期"
            onClick={(e) => {
              e.stopPropagation();
              onChange('');
            }}
            className="grid place-items-center w-[18px] h-[18px] flex-none rounded-full text-muted hover:bg-panel-2 hover:text-ink-2"
          >
            <CloseIcon className="w-3 h-3" />
          </span>
        )}
      </button>
      {open && (
        <div
          ref={popRef}
          className={cn(popClass, 'rounded-[10px] border bg-white p-3 shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)]')}
          style={{ borderColor: 'var(--line)' }}
        >
          <CalendarPanel value={value} onPick={pick} min={min} max={max} />
          <div className="flex items-center justify-between mt-3 pt-2 border-t" style={{ borderColor: 'var(--line)' }}>
            {clearable ? (
              <button
                type="button"
                onClick={() => {
                  onChange('');
                  setOpen(false);
                }}
                className="text-[12px] text-ink-2 hover:text-ink px-2 py-1 rounded-[6px] hover:bg-panel-2"
              >
                清除
              </button>
            ) : (
              <span />
            )}
            <button
              type="button"
              disabled={!todayPickable}
              onClick={() => pick(todayK)}
              className="text-[12px] font-semibold px-2 py-1 rounded-[6px] hover:bg-panel-2 disabled:opacity-40"
              style={{ color: 'var(--brand)' }}
            >
              今天
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
