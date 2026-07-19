/**
 * 平台 Tabs（Android / iOS）—— 上架包管理页顶部筛选，样式仿照 BrandTabs（同一视觉语言），
 * 但上架包没有品牌域名可展示，副标题固定为「上架包」，计数按当前已加载的上架包数组算。
 */
import type { ListingPlatform } from '@/lib/types';
import { PLATFORM_META, PLATFORM_ORDER } from '@/lib/listingMeta';
import { cn } from '@/lib/cn';

export function PlatformTabs({
  counts,
  current,
  onSelect,
}: {
  counts: Record<ListingPlatform, number>;
  current: ListingPlatform;
  onSelect: (platform: ListingPlatform) => void;
}) {
  return (
    <div className="flex gap-2 mb-[18px] flex-wrap">
      {PLATFORM_ORDER.map((p) => {
        const meta = PLATFORM_META[p];
        const active = p === current;
        return (
          <button
            key={p}
            onClick={() => onSelect(p)}
            className={cn(
              'relative flex items-center gap-[10px] px-4 py-[11px] rounded-[12px] bg-panel border min-w-[160px] transition shadow-sm2',
              'hover:-translate-y-px hover:shadow-md2',
              active ? 'border-transparent' : 'border-line',
            )}
            data-active={active}
          >
            <span
              className="grid place-items-center w-[30px] h-[30px] rounded-[9px] text-white font-extrabold text-[13px] flex-none"
              style={{ background: meta.accentColor }}
            >
              {meta.label[0]}
            </span>
            <span className="text-left">
              <span className="block font-bold text-[14px] leading-[1.2]">{meta.label}</span>
              <span className="block text-[11.5px] text-muted">上架包</span>
            </span>
            <span
              className={cn(
                'ml-auto text-[12px] font-bold rounded-lg px-[9px] py-[3px]',
                active ? 'text-white' : 'text-ink-2 bg-bg',
              )}
              style={active ? { background: meta.accentColor } : undefined}
            >
              {counts[p] ?? 0}
            </span>
            {active && (
              <span
                aria-hidden
                className="pointer-events-none absolute inset-0 rounded-[12px]"
                style={{ boxShadow: `0 0 0 2px ${meta.accentColor} inset` }}
              />
            )}
          </button>
        );
      })}
    </div>
  );
}
