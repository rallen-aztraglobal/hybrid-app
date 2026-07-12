/**
 * 表单字段外壳（label + hint + error）—— 从 ChannelDrawer 抽出，供其自身与
 * AdjustSection 等子块复用同一视觉规范（避免子组件反向 import ChannelDrawer 造成循环依赖）。
 */
import type { ReactNode } from 'react';

export function Field({
  label,
  required,
  hint,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  hint?: string;
  error?: string;
  children: ReactNode;
}) {
  return (
    <div className="mb-[13px] last:mb-0">
      <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
        {label} {required && <span className="text-down">*</span>}{' '}
        {hint && <span className="font-normal text-muted text-[11.5px]">{hint}</span>}
      </label>
      {children}
      {error && <div className="mt-1 text-[12px] text-down">{error}</div>}
    </div>
  );
}
