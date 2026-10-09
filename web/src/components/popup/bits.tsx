/**
 * 弹窗模块的小件：位置徽章、状态点、优先级徽章、分段控件、缩略图（对齐 docs/admin/ui/popup.html）。
 */
import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';
import { BRAND_META } from '@/lib/brands';
import { POSITION_CAPS, STATUS_LABEL } from '@/lib/popupMeta';
import type { BrandCode, PopupPositionCode, PopupStatus } from '@/lib/types';
import { MegaphoneIcon } from '../icons';

export function PosBadge({ code }: { code: PopupPositionCode }) {
  return (
    <span className="inline-flex items-center gap-[5px] text-[12px] font-semibold text-ink-2 bg-[#f1f5f9] pl-1 pr-2 py-[3px] rounded-[7px] whitespace-nowrap">
      <b className="font-mono text-[11px] bg-white text-brand-ink px-[5px] rounded-[5px] shadow-sm2">{code || '?'}</b>
      {POSITION_CAPS[code]?.name ?? '未知'}
    </span>
  );
}

const PRIO_CLS = {
  P0: 'bg-[#fee2e2] text-[#b91c1c]',
  P1: 'bg-[#fef3c7] text-[#92681a]',
  P2: 'bg-[#f1f5f9] text-[#64748b]',
} as const;

export function PriorityBadge({ level }: { level: 'P0' | 'P1' | 'P2' }) {
  return (
    <span
      className={cn('font-mono text-[10.5px] font-extrabold px-[6px] py-px rounded-[5px] tracking-[.2px]', PRIO_CLS[level])}
      title="建议优先级（PRD §2）"
    >
      {level}
    </span>
  );
}

const STATUS_STYLE: Record<PopupStatus, { dot: string; ring?: string; text: string }> = {
  active: { dot: 'var(--ok)', ring: 'rgba(34,197,94,.16)', text: '#15803d' },
  scheduled: { dot: 'var(--brand)', ring: 'rgba(99,102,241,.15)', text: 'var(--brand-ink)' },
  ended: { dot: '#cbd5e1', text: 'var(--muted)' },
  disabled: { dot: '#64748b', text: '#64748b' },
};

export function StatusBadge({ status }: { status: PopupStatus }) {
  const st = STATUS_STYLE[status];
  return (
    <span className="inline-flex items-center gap-[7px] text-[12px] font-semibold whitespace-nowrap" style={{ color: st.text }}>
      <i
        className="w-2 h-2 rounded-full inline-block"
        style={{ background: st.dot, boxShadow: st.ring ? `0 0 0 3px ${st.ring}` : undefined }}
      />
      {STATUS_LABEL[status]}
    </span>
  );
}

export function ForceBadge({ label = '强制' }: { label?: string }) {
  return (
    <span
      className="inline-flex items-center gap-[3px] text-[11px] font-extrabold text-white bg-[#dc2626] px-[7px] py-[2px] rounded-[6px] ml-1"
      style={{ boxShadow: '0 4px 10px -4px rgba(220,38,38,.7)' }}
    >
      {label}
    </span>
  );
}

export function BrandTag({ code }: { code: BrandCode }) {
  const meta = BRAND_META[code];
  return (
    <span className="inline-flex items-center gap-1 text-[11px] font-bold font-mono text-ink-2 bg-white border border-line px-[7px] py-px rounded-[5px] whitespace-nowrap">
      <i className="w-[7px] h-[7px] rounded-full inline-block" style={{ background: meta?.accentColor ?? '#94a3b8' }} />
      {code}
    </span>
  );
}

export function Tag({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1 text-[11px] text-ink-2 bg-[#f1f5f9] px-[7px] py-px rounded-[5px] whitespace-nowrap">
      {children}
    </span>
  );
}

/** 分段控件（原型 .seg）。 */
export function Seg<T extends string>({
  value,
  options,
  onChange,
  disabled,
}: {
  value: T;
  options: { value: T; label: string }[];
  onChange: (v: T) => void;
  disabled?: boolean;
}) {
  return (
    <div className="inline-flex flex-wrap gap-[2px] p-[3px] rounded-[10px] bg-[#eef2f7]">
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          disabled={disabled}
          onClick={() => onChange(o.value)}
          className={cn(
            'px-3 py-[6px] rounded-lg text-[12.5px] font-semibold transition disabled:opacity-50 disabled:cursor-not-allowed',
            value === o.value ? 'bg-white text-brand-ink shadow-sm2' : 'text-ink-2',
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

/**
 * 缩略图：按位置决定形状（P2/P8 圆形、P3 方形、P7 9:16、P5 纯文字、其余 3:4）。
 */
export function PopupThumb({
  position,
  imageUrl,
  count = 1,
  scale = 1,
}: {
  position: PopupPositionCode;
  imageUrl?: string;
  count?: number;
  scale?: number;
}) {
  if (position === 'P5') {
    return (
      <div
        className="flex-none flex flex-col items-center justify-center text-[10px] font-bold text-[#64748b] bg-[#f1f5f9] border border-dashed border-[#cbd5e1] rounded-[9px]"
        style={{ width: 46 * scale, height: 46 * scale }}
      >
        <MegaphoneIcon className="w-4 h-4" />
        纯文字
      </div>
    );
  }
  const round = position === 'P2' || position === 'P8';
  const [w, h] =
    position === 'P7' ? [34, 60] : position === 'P3' ? [46, 46] : round ? [44, 44] : [42, 56];
  return (
    <div
      className={cn(
        'relative flex-none overflow-hidden bg-[#e2e8f0] shadow-sm2',
        round ? 'rounded-full border-2 border-white' : position === 'P3' ? 'rounded-[9px]' : 'rounded-[7px]',
      )}
      style={{ width: w * scale, height: h * scale, boxShadow: round ? '0 0 0 1px var(--line), var(--shadow-sm)' : undefined }}
    >
      {imageUrl ? (
        <img src={imageUrl} alt="" className="w-full h-full object-cover" loading="lazy" />
      ) : (
        <div className="w-full h-full grid place-items-center text-[#94a3b8] text-[10px]">无图</div>
      )}
      {count > 1 && (
        <span className="absolute right-[2px] bottom-[2px] bg-[rgba(15,23,42,.8)] text-white text-[9.5px] font-bold px-1 rounded leading-[15px]">
          {count} 张
        </span>
      )}
    </div>
  );
}
