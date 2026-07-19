/**
 * 通用 UI 原子组件 —— 复刻 docs/admin/ui/index.html 的 .btn / .switch / .hdot / .hp
 * / .app-icon 等视觉，组件化复用。样式走 index.css 的 @layer components + 行内变量。
 */
import {
  forwardRef,
  useEffect,
  useRef,
  useState,
  type AnchorHTMLAttributes,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react';
import { cn } from '@/lib/cn';
import type { DomainHealth } from '@/lib/types';
import { iconGradient } from '@/lib/brands';
import { apkFileName } from '@/lib/text';
import { DownloadIcon } from './icons';

// ---------- Button ----------
type BtnVariant = 'primary' | 'ghost';
interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: BtnVariant;
}
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'ghost', className, children, ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      className={cn('btn', variant === 'primary' ? 'btn-primary' : 'btn-ghost', className)}
      {...rest}
    >
      {children}
    </button>
  );
});

// ---------- Select（自定义下拉，替代浏览器原生 <select>，与 field-input 同风格）----------
export interface SelectOption {
  value: string;
  label: ReactNode;
}
interface SelectProps {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
  className?: string;
}
export function Select({ value, onChange, options, placeholder, disabled, className }: SelectProps) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const wrapRef = useRef<HTMLDivElement>(null);

  const selected = options.find((o) => o.value === value);

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
        onKeyDown={(e) => {
          if (disabled) return;
          if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            if (!open) {
              setOpen(true);
              setActive(options.findIndex((o) => o.value === value));
            } else if (e.key === 'ArrowDown') {
              setActive((i) => Math.min(options.length - 1, i + 1));
            } else if (active >= 0) {
              pick(options[active].value);
            }
          } else if (e.key === 'ArrowUp' && open) {
            e.preventDefault();
            setActive((i) => Math.max(0, i - 1));
          }
        }}
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
        <ul
          role="listbox"
          className="absolute z-50 left-0 right-0 mt-1 max-h-[280px] overflow-y-auto rounded-[10px] border bg-white py-1 shadow-[0_12px_28px_-8px_rgba(0,0,0,0.25)]"
          style={{ borderColor: 'var(--line)' }}
        >
          {options.length === 0 && (
            <li className="px-3 py-[9px] text-[13px] text-muted select-none">无可选项</li>
          )}
          {options.map((o, i) => {
            const isSel = o.value === value;
            return (
              <li
                key={o.value}
                role="option"
                aria-selected={isSel}
                onMouseEnter={() => setActive(i)}
                onClick={() => pick(o.value)}
                className={cn(
                  'flex items-center justify-between gap-2 px-3 py-[9px] text-[13px] cursor-pointer',
                  (active === i || isSel) && 'bg-panel-2',
                  isSel ? 'font-semibold' : 'text-ink',
                )}
                style={isSel ? { color: 'var(--brand)' } : undefined}
              >
                <span className="truncate">{o.label}</span>
                {isSel && (
                  <svg
                    className="w-4 h-4 flex-none"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
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
      )}
    </div>
  );
}

// ---------- IconButton ----------
export function IconButton({
  className,
  children,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      className={cn(
        'grid place-items-center w-[38px] h-[38px] rounded-[10px] border bg-panel text-ink-2 transition',
        'border-line hover:bg-bg hover:text-ink',
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

// ---------- 健康点（卡片用小圆点） ----------
const HDOT_COLOR: Record<DomainHealth, string> = {
  ok: 'var(--ok)',
  warn: 'var(--warn)',
  down: 'var(--down)',
  unknown: '#e2e8f0',
  unconfigured: '#e2e8f0',
};
const HDOT_RING: Partial<Record<DomainHealth, string>> = {
  ok: 'rgba(34,197,94,.15)',
  warn: 'rgba(245,158,11,.15)',
  down: 'rgba(239,68,68,.15)',
};
export function HealthDot({ health }: { health: DomainHealth }) {
  const ring = HDOT_RING[health];
  return (
    <span
      className="inline-block w-[9px] h-[9px] rounded-full"
      style={{
        background: HDOT_COLOR[health],
        boxShadow: ring ? `0 0 0 3px ${ring}` : undefined,
      }}
      title={healthLabel(health)}
    />
  );
}

// ---------- 健康徽标（域名行用胶囊） ----------
export function HealthPill({ health }: { health: DomainHealth }) {
  const map: Record<DomainHealth, { cls: string; txt: string }> = {
    ok: { cls: 'text-[#15803d] bg-[#dcfce7]', txt: '● 正常' },
    warn: { cls: 'text-[#92681a] bg-[#fef3c7]', txt: '● 延迟高' },
    down: { cls: 'text-[#b91c1c] bg-[#fee2e2]', txt: '● 不可达' },
    unknown: { cls: 'text-muted bg-[#f1f5f9]', txt: '○ 未探测' },
    unconfigured: { cls: 'text-muted bg-[#f1f5f9]', txt: '○ 未配' },
  };
  const { cls, txt } = map[health];
  return (
    <span className={cn('text-[11px] font-semibold px-[9px] py-1 rounded-full whitespace-nowrap flex-none', cls)}>
      {txt}
    </span>
  );
}

export function healthLabel(h: DomainHealth): string {
  return { ok: '正常', warn: '延迟高', down: '不可达', unknown: '未探测', unconfigured: '未配' }[h];
}

// ---------- Switch ----------
export function Switch({
  checked,
  onChange,
  disabled,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  /** 可选：置灰且不响应点击（原生 button disabled 属性天然阻止 onClick 触发）。 */
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative w-[42px] h-6 rounded-full flex-none transition',
        checked ? 'bg-brand' : 'bg-[#cbd5e1]',
        disabled && 'opacity-40 cursor-not-allowed',
      )}
    >
      <span
        className={cn(
          'absolute top-[2px] w-5 h-5 rounded-full bg-white shadow-sm2 transition-all',
          checked ? 'left-5' : 'left-[2px]',
        )}
      />
    </button>
  );
}

// ---------- AppIcon（渐变方块 + 首字母 / 自定义图） ----------
export function AppIcon({
  initials,
  hex,
  src,
  size = 52,
  radius = 14,
}: {
  initials: string;
  hex: string;
  src?: string;
  size?: number;
  radius?: number;
}) {
  // 无状态降级（评审 W4）：底层始终是渐变+首字母；有 src 时叠一张图盖住，
  // 图加载失败(onError)即隐藏自身 → 自动露出底层首字母，不会出现「裂图」占位。
  return (
    <div
      className="relative grid place-items-center text-white font-extrabold flex-none shadow-sm2 overflow-hidden"
      style={{
        width: size,
        height: size,
        borderRadius: radius,
        background: iconGradient(hex),
        fontSize: size * 0.33,
      }}
    >
      <span style={{ textShadow: '0 1px 2px rgba(0,0,0,.25)' }}>{initials}</span>
      {src && (
        <img
          src={src}
          alt={initials}
          className="absolute inset-0 w-full h-full object-cover"
          style={{ background: '#fff' }}
          onError={(e) => {
            (e.currentTarget as HTMLImageElement).style.display = 'none';
          }}
        />
      )}
    </div>
  );
}

// ---------- APK 下载按钮（ADR-0008：nginx 静态产物 /apks/...） ----------
interface ApkDownloadProps extends AnchorHTMLAttributes<HTMLAnchorElement> {
  url: string;
  label?: string;
  variant?: 'primary' | 'ghost';
}
export function ApkDownloadButton({ url, label = '下载 APK', variant = 'ghost', className, ...rest }: ApkDownloadProps) {
  return (
    <a
      href={url}
      // 同源静态产物直接下载；download 提示浏览器存盘并用末段文件名。
      download={apkFileName(url)}
      target="_blank"
      rel="noopener noreferrer"
      className={cn('btn', variant === 'primary' ? 'btn-primary' : 'btn-ghost', 'text-[12px]', className)}
      onClick={(e) => e.stopPropagation()}
      {...rest}
    >
      <DownloadIcon className="w-4 h-4" />
      {label}
    </a>
  );
}

// ---------- Note（提示框） ----------
export function Note({ children }: { children: ReactNode }) {
  return (
    <div
      className="flex gap-[10px] p-[12px_13px] rounded-[10px] text-[12px] leading-[1.55] text-ink-2"
      style={{ background: 'rgba(99,102,241,.06)', border: '1px solid rgba(99,102,241,.18)' }}
    >
      {children}
    </div>
  );
}

// ---------- 段落小标题（抽屉里编号小节） ----------
export function SectionHeading({ num, children }: { num: ReactNode; children: ReactNode }) {
  return (
    <h3 className="flex items-center gap-2 text-[13px] font-bold text-ink mb-[14px]">
      <span className="grid place-items-center w-5 h-5 rounded-md bg-brand text-white text-[11px] font-extrabold">
        {num}
      </span>
      {children}
    </h3>
  );
}
