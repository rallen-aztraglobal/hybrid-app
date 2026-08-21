/**
 * 自绘日期选择器（替代原生 <input type="date">，风格对齐 .field-input + Select 的 popover）。
 * value 用本地时区拼接的 'YYYY-MM-DD'（不用 toISOString，避免时区偏移导致日期错一天）。
 */
import { useEffect, useRef, useState } from 'react';
import { cn } from '@/lib/cn';
import { CalendarIcon, CloseIcon } from './icons';

interface DatePickerProps {
  value: string;
  onChange: (value: string) => void;
  min?: string;
  max?: string;
  placeholder?: string;
  className?: string;
}

const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日'];

function pad2(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}

/** 本地时区拼接 YYYY-MM-DD，避免 toISOString 的 UTC 偏移坑。 */
function toKey(y: number, m: number, d: number): string {
  return `${y}-${pad2(m + 1)}-${pad2(d)}`;
}

function parseKey(key: string): { y: number; m: number; d: number } | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(key);
  if (!match) return null;
  return { y: Number(match[1]), m: Number(match[2]) - 1, d: Number(match[3]) };
}

function todayKey(): string {
  const now = new Date();
  return toKey(now.getFullYear(), now.getMonth(), now.getDate());
}

/** 该月 1 日是周几（0=周一...6=周日），用于网格起点对齐。 */
function firstWeekdayMon0(y: number, m: number): number {
  const jsDay = new Date(y, m, 1).getDay(); // 0=周日
  return (jsDay + 6) % 7;
}

function daysInMonth(y: number, m: number): number {
  return new Date(y, m + 1, 0).getDate();
}

export function DatePicker({ value, onChange, min, max, placeholder, className }: DatePickerProps) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  const parsed = parseKey(value);
  const initial = parsed ?? parseKey(todayKey())!;
  const [viewY, setViewY] = useState(initial.y);
  const [viewM, setViewM] = useState(initial.m);

  useEffect(() => {
    if (!open) return;
    const p = parseKey(value) ?? parseKey(todayKey())!;
    setViewY(p.y);
    setViewM(p.m);
  }, [open, value]);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  function prevMonth() {
    setViewM((m) => {
      if (m === 0) {
        setViewY((y) => y - 1);
        return 11;
      }
      return m - 1;
    });
  }
  function nextMonth() {
    setViewM((m) => {
      if (m === 11) {
        setViewY((y) => y + 1);
        return 0;
      }
      return m + 1;
    });
  }

  function inRange(key: string): boolean {
    if (min && key < min) return false;
    if (max && key > max) return false;
    return true;
  }

  function pick(y: number, m: number, d: number) {
    const key = toKey(y, m, d);
    if (!inRange(key)) return;
    onChange(key);
    setViewY(y);
    setViewM(m);
    setOpen(false);
  }

  const todayK = todayKey();
  const leadDays = firstWeekdayMon0(viewY, viewM);
  const curDays = daysInMonth(viewY, viewM);
  const prevM = viewM === 0 ? 11 : viewM - 1;
  const prevY = viewM === 0 ? viewY - 1 : viewY;
  const prevDays = daysInMonth(prevY, prevM);
  const nextM = viewM === 11 ? 0 : viewM + 1;
  const nextY = viewM === 11 ? viewY + 1 : viewY;

  type Cell = { y: number; m: number; d: number; inMonth: boolean };
  const cells: Cell[] = [];
  for (let i = leadDays - 1; i >= 0; i--) {
    cells.push({ y: prevY, m: prevM, d: prevDays - i, inMonth: false });
  }
  for (let d = 1; d <= curDays; d++) {
    cells.push({ y: viewY, m: viewM, d, inMonth: true });
  }
  while (cells.length < 42) {
    const idx = cells.length - (leadDays + curDays);
    cells.push({ y: nextY, m: nextM, d: idx + 1, inMonth: false });
  }

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={cn('field-input flex items-center gap-2 text-left cursor-pointer', className)}
      >
        <CalendarIcon className="w-4 h-4 flex-none text-muted" />
        <span className={cn('flex-1 truncate', !value && 'text-muted')}>
          {value || placeholder || '选择日期'}
        </span>
        {value && (
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
          className="absolute z-50 left-0 mt-1 w-[280px] rounded-[10px] border bg-white p-3 shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)]"
          style={{ borderColor: 'var(--line)' }}
        >
          <div className="flex items-center justify-between mb-2">
            <button
              type="button"
              onClick={prevMonth}
              className="grid place-items-center w-7 h-7 rounded-[8px] text-ink-2 hover:bg-panel-2"
              aria-label="上一月"
            >
              ‹
            </button>
            <span className="text-[13px] font-semibold text-ink">
              {viewY}年{viewM + 1}月
            </span>
            <button
              type="button"
              onClick={nextMonth}
              className="grid place-items-center w-7 h-7 rounded-[8px] text-ink-2 hover:bg-panel-2"
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
                  onClick={() => pick(c.y, c.m, c.d)}
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
          <div className="flex items-center justify-between mt-3 pt-2 border-t" style={{ borderColor: 'var(--line)' }}>
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
            <button
              type="button"
              onClick={() => {
                const t = parseKey(todayK)!;
                pick(t.y, t.m, t.d);
              }}
              className="text-[12px] font-semibold px-2 py-1 rounded-[6px] hover:bg-panel-2"
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
