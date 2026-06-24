/**
 * 域名编辑器 —— 复刻原型「域名配置」：1 主 + 最多 3 备用，每行健康徽标。
 * 复用于：渠道抽屉（覆盖域名）与域名配置页（品牌默认域名）。
 * 失焦时做轻量形态校验（validateDomainUrl），把无效项的健康标记为 down。
 */
import type { DomainEntry } from '@/lib/types';
import { validateDomainUrl } from '@/lib/validation';
import { HealthPill } from './ui';
import { cn } from '@/lib/cn';

const TAGS = ['主', '备用1', '备用2', '备用3'];

export function DomainEditor({
  domains,
  onChange,
  disabled,
  maxBackups = 3,
}: {
  domains: DomainEntry[];
  onChange: (next: DomainEntry[]) => void;
  disabled?: boolean;
  maxBackups?: number;
}) {
  // 规范化为 position 0..maxBackups 的固定行（缺失补空）
  const rows: DomainEntry[] = Array.from({ length: maxBackups + 1 }, (_, pos) => {
    const found = domains.find((d) => d.position === pos);
    return found ?? { position: pos, url: '', enabled: true, health: 'unconfigured' };
  });

  function update(pos: number, url: string) {
    const next = rows.map((r) =>
      r.position === pos
        ? { ...r, url, health: url.trim() ? r.health ?? 'unknown' : ('unconfigured' as const) }
        : r,
    );
    onChange(next.filter((r) => r.url.trim() || r.position === 0));
  }

  function validateOnBlur(pos: number) {
    const row = rows.find((r) => r.position === pos);
    if (!row || !row.url.trim()) return;
    const err = validateDomainUrl(row.url.trim());
    const next = rows.map((r) =>
      r.position === pos ? { ...r, health: err ? ('down' as const) : (r.health === 'down' ? 'unknown' : r.health) } : r,
    );
    onChange(next.filter((r) => r.url.trim() || r.position === 0));
  }

  return (
    <div className={cn(disabled && 'opacity-50 pointer-events-none')}>
      {rows.map((row) => (
        <div key={row.position} className="flex items-center gap-[10px] mb-[10px]">
          <span
            className={cn(
              'text-[11px] font-bold w-[52px] flex-none text-center py-[5px] rounded-[7px]',
              row.position === 0 ? 'text-[var(--brand-ink)]' : 'text-[#64748b] bg-[#f1f5f9]',
            )}
            style={row.position === 0 ? { background: 'rgba(99,102,241,.12)' } : undefined}
          >
            {TAGS[row.position]}
          </span>
          <input
            value={row.url}
            onChange={(e) => update(row.position, e.target.value)}
            onBlur={() => validateOnBlur(row.position)}
            placeholder={row.position === 0 ? 'https://your-domain.com' : 'https://（可选）'}
            className="flex-1 font-mono text-[12.5px] rounded-[9px] border border-line bg-panel px-[11px] py-[9px] focus:outline-none focus:border-brand focus:shadow-[0_0_0_3px_rgba(99,102,241,.12)]"
          />
          <HealthPill health={row.health ?? 'unconfigured'} />
        </div>
      ))}
    </div>
  );
}
