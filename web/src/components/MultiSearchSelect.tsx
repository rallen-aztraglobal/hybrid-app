/**
 * 带模糊搜索的多选下拉（SearchSelect 的多选变体，视觉/交互风格保持一致）：
 * 选项行为复选框语义——点击切换选中、面板不关闭；触发按钮显示「已选 N 项」；
 * 面板底部提供「清空」。空选中集 = 不限（如「全部渠道」）。
 * 用于设备管理页「渠道」多选筛选。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { cn } from '@/lib/cn';
import { SearchIcon } from './icons';
import type { SearchSelectOption } from './SearchSelect';

interface MultiSearchSelectProps {
  values: string[];
  onChange: (values: string[]) => void;
  options: SearchSelectOption[];
  /** 空选中集时按钮展示的文案（如「全部渠道」）。 */
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  searchPlaceholder?: string;
}

/** 大小写不敏感包含匹配：命中 label 或 sub 任一即算匹配。 */
function matches(o: SearchSelectOption, kw: string): boolean {
  if (!kw) return true;
  const k = kw.toLowerCase();
  return o.label.toLowerCase().includes(k) || (o.sub ?? '').toLowerCase().includes(k);
}

export function MultiSearchSelect({
  values,
  onChange,
  options,
  placeholder,
  disabled,
  className,
  searchPlaceholder,
}: MultiSearchSelectProps) {
  const [open, setOpen] = useState(false);
  const [kw, setKw] = useState('');
  const [active, setActive] = useState(-1);
  const wrapRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  const selectedSet = useMemo(() => new Set(values), [values]);
  const filtered = useMemo(() => options.filter((o) => matches(o, kw)), [options, kw]);

  const buttonLabel = useMemo(() => {
    if (values.length === 0) return placeholder ?? '请选择…';
    if (values.length === 1) {
      const hit = options.find((o) => o.value === values[0]);
      return hit ? hit.label : values[0];
    }
    return `已选 ${values.length} 项`;
  }, [values, options, placeholder]);

  useEffect(() => {
    if (!open) return;
    setKw('');
    setActive(-1);
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

  const toggle = (v: string) => {
    if (selectedSet.has(v)) onChange(values.filter((x) => x !== v));
    else onChange([...values, v]);
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
        <span className={cn('truncate', values.length === 0 && 'text-muted')}>{buttonLabel}</span>
        <span className="flex items-center gap-1 flex-none">
          {values.length > 1 && (
            <span
              className="rounded-full px-[7px] py-[1px] text-[11px] font-semibold text-white"
              style={{ background: 'var(--brand)' }}
            >
              {values.length}
            </span>
          )}
          <svg
            className={cn('w-4 h-4 transition-transform text-ink-2', open && 'rotate-180')}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </span>
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
                    if (active >= 0 && active < filtered.length) toggle(filtered[active].value);
                  } else if (e.key === 'Escape') {
                    setOpen(false);
                  }
                }}
                placeholder={searchPlaceholder ?? '搜索…'}
                className="w-full rounded-[8px] border pl-8 pr-2 py-[7px] text-[12.5px] outline-none"
                style={{ borderColor: 'var(--line)', background: 'var(--panel-2)' }}
              />
            </div>
          </div>
          <ul role="listbox" aria-multiselectable className="max-h-[300px] overflow-y-auto py-1">
            {filtered.length === 0 && (
              <li className="px-3 py-[14px] text-[13px] text-muted select-none text-center">无匹配选项</li>
            )}
            {filtered.map((o, i) => {
              const isSel = selectedSet.has(o.value);
              return (
                <li
                  key={o.value}
                  role="option"
                  aria-selected={isSel}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => toggle(o.value)}
                  className={cn(
                    'flex items-center gap-2.5 px-3 py-[8px] cursor-pointer',
                    active === i && 'bg-panel-2',
                  )}
                >
                  <span
                    className={cn(
                      'w-[15px] h-[15px] flex-none rounded-[4px] border flex items-center justify-center',
                    )}
                    style={
                      isSel
                        ? { background: 'var(--brand)', borderColor: 'var(--brand)' }
                        : { borderColor: 'var(--line)' }
                    }
                  >
                    {isSel && (
                      <svg
                        className="w-[11px] h-[11px]"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="#fff"
                        strokeWidth="3"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <polyline points="20 6 9 17 4 12" />
                      </svg>
                    )}
                  </span>
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
                </li>
              );
            })}
          </ul>
          <div
            className="flex items-center justify-between px-3 py-2 border-t text-[12px] bg-white"
            style={{ borderColor: 'var(--line)' }}
          >
            <span className="text-muted">已选 {values.length} 项{values.length === 0 ? '（不限）' : ''}</span>
            <button
              type="button"
              disabled={values.length === 0}
              onClick={() => onChange([])}
              className={cn(
                'font-semibold',
                values.length === 0 ? 'text-muted cursor-default' : 'cursor-pointer',
              )}
              style={values.length > 0 ? { color: 'var(--brand)' } : undefined}
            >
              清空
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
