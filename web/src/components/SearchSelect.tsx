/**
 * 带模糊搜索的自定义下拉（复用 ui.tsx Select 的 popover/键盘导航/outside-click 写法，
 * 额外支持：选项两行展示（主文案 + 小号 muted 副文案）+ 面板顶部固定搜索框过滤）。
 * 用于渠道选择等「选项带长附加信息、易被截断」的场景（见 DevicesPage 筛选栏）。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { cn } from '@/lib/cn';
import { SearchIcon } from './icons';

export interface SearchSelectOption {
  value: string;
  label: string;
  /** 小号 muted 副文案，如 `palCode · applicationId`；空值选项（如「全部渠道」）不传。 */
  sub?: string;
}

interface SearchSelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SearchSelectOption[];
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}

/** 大小写不敏感包含匹配：命中 label 或 sub 任一即算匹配。 */
function matches(o: SearchSelectOption, kw: string): boolean {
  if (!kw) return true;
  const k = kw.toLowerCase();
  return o.label.toLowerCase().includes(k) || (o.sub ?? '').toLowerCase().includes(k);
}

export function SearchSelect({ value, onChange, options, placeholder, disabled, className }: SearchSelectProps) {
  const [open, setOpen] = useState(false);
  const [kw, setKw] = useState('');
  const [active, setActive] = useState(-1);
  const wrapRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const selected = options.find((o) => o.value === value);
  const filtered = useMemo(() => options.filter((o) => matches(o, kw)), [options, kw]);

  useEffect(() => {
    if (!open) return;
    setKw('');
    setActive(options.findIndex((o) => o.value === value));
    // 打开后自动 focus 搜索框
    const t = window.setTimeout(() => searchRef.current?.focus(), 0);
    return () => window.clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

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

  const pick = (v: string) => {
    onChange(v);
    setOpen(false);
  };

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        disabled={disabled}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => !disabled && setOpen((o) => !o)}
        className={cn(
          'field-input flex items-center justify-between gap-2 text-left',
          disabled && 'opacity-50 cursor-not-allowed',
          !disabled && 'cursor-pointer',
          className,
        )}
      >
        <span className={cn('truncate', !selected && 'text-muted')}>
          {selected ? selected.label : placeholder ?? '请选择…'}
        </span>
        <svg
          className={cn('w-4 h-4 flex-none transition-transform text-ink-2', open && 'rotate-180')}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      {open && (
        <div
          className="absolute z-50 left-0 mt-1 w-[380px] max-w-[min(380px,90vw)] rounded-[10px] border bg-white shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)] overflow-hidden"
          style={{ borderColor: 'var(--line)' }}
        >
          <div className="sticky top-0 z-10 p-2 border-b bg-white" style={{ borderColor: 'var(--line)' }}>
            <div className="relative">
              <SearchIcon className="absolute left-2.5 top-1/2 -translate-y-1/2 w-[14px] h-[14px] text-muted pointer-events-none" />
              <input
                ref={searchRef}
                value={kw}
                onChange={(e) => {
                  setKw(e.target.value);
                  setActive(0);
                }}
                onKeyDown={(e) => {
                  if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    setActive((i) => Math.min(filtered.length - 1, i + 1));
                  } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    setActive((i) => Math.max(0, i - 1));
                  } else if (e.key === 'Enter') {
                    e.preventDefault();
                    if (active >= 0 && active < filtered.length) pick(filtered[active].value);
                  } else if (e.key === 'Escape') {
                    setOpen(false);
                  }
                }}
                placeholder="搜索应用名 / PAL_CODE / 包名…"
                className="w-full rounded-[8px] border pl-8 pr-2 py-[7px] text-[12.5px] outline-none"
                style={{ borderColor: 'var(--line)', background: 'var(--panel-2)' }}
              />
            </div>
          </div>
          <ul role="listbox" className="max-h-[320px] overflow-y-auto py-1">
            {filtered.length === 0 && (
              <li className="px-3 py-[14px] text-[13px] text-muted select-none text-center">无匹配渠道</li>
            )}
            {filtered.map((o, i) => {
              const isSel = o.value === value;
              return (
                <li
                  key={o.value}
                  role="option"
                  aria-selected={isSel}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => pick(o.value)}
                  className={cn(
                    'flex items-center justify-between gap-2 px-3 py-[8px] cursor-pointer',
                    (active === i || isSel) && 'bg-panel-2',
                  )}
                >
                  <span className="min-w-0 flex-1">
                    <span
                      className={cn('block truncate text-[13px]', isSel ? 'font-semibold' : 'text-ink')}
                      style={isSel ? { color: 'var(--brand)' } : undefined}
                      title={o.label}
                    >
                      {o.label}
                    </span>
                    {o.sub && (
                      <span className="mono block truncate text-[11px] text-muted mt-[1px]" title={o.sub}>
                        {o.sub}
                      </span>
                    )}
                  </span>
                  {isSel && (
                    <svg
                      className="w-4 h-4 flex-none"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="var(--brand)"
                      strokeWidth="2.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <polyline points="20 6 9 17 4 12" />
                    </svg>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
