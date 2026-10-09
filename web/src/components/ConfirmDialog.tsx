import type { ReactNode } from 'react';
import { Button } from './ui';

/** 页面内确认框（替代 window.confirm）。 */
export function ConfirmDialog({
  open,
  title,
  children,
  confirmText = '确认',
  danger,
  busy,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  children?: ReactNode;
  confirmText?: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-[80] grid place-items-center" style={{ background: 'rgba(15,23,42,.45)', backdropFilter: 'blur(2px)' }} onClick={onCancel}>
      <div
        role="dialog"
        aria-modal="true"
        className="w-[420px] max-w-[92vw] rounded-card bg-panel border border-line shadow-lg2 p-5 animate-fade"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="text-[15px] font-bold text-ink mb-2">{title}</div>
        <div className="text-[13px] text-ink-2 leading-relaxed">{children}</div>
        <div className="flex justify-end gap-2 mt-5">
          <Button onClick={onCancel} disabled={busy}>取消</Button>
          <Button
            variant="primary"
            disabled={busy}
            onClick={onConfirm}
            style={danger ? { background: '#dc2626', boxShadow: '0 8px 20px -8px rgba(220,38,38,.7)' } : undefined}
          >
            {busy ? '处理中…' : confirmText}
          </Button>
        </div>
      </div>
    </div>
  );
}
