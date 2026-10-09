import { useToastStore } from '@/store/toastStore';
import { cn } from '@/lib/cn';
import { AlertIcon, CheckIcon } from './icons';

/** 全局 toast 容器，挂在 AppShell 里一次即可。 */
export function Toaster() {
  const items = useToastStore((s) => s.items);
  const dismiss = useToastStore((s) => s.dismiss);
  if (items.length === 0) return null;
  return (
    <div className="fixed left-1/2 bottom-7 -translate-x-1/2 z-[300] flex flex-col items-center gap-2 pointer-events-none">
      {items.map((t) => (
        <div
          key={t.id}
          onClick={() => dismiss(t.id)}
          className="pointer-events-auto flex items-start gap-[9px] max-w-[min(720px,92vw)] rounded-xl px-4 py-[11px] text-[13px] text-white shadow-lg2 cursor-pointer animate-fade"
          style={{ background: '#0f172a' }}
        >
          {t.kind === 'err' ? (
            <AlertIcon className={cn('w-4 h-4 flex-none mt-px')} style={{ color: '#f87171' }} />
          ) : (
            <CheckIcon className="w-4 h-4 flex-none mt-px" style={{ color: '#4ade80' }} />
          )}
          <span className="break-words">{t.message}</span>
        </div>
      ))}
    </div>
  );
}
