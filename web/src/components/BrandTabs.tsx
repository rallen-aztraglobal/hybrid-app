/**
 * 大渠道 Tabs —— 复刻原型 .tabs/.tab（3 大渠道、品牌色、计数）。
 * 渠道管理与打包中心共用；选中态写入 uiStore.currentBrand。
 */
import type { Brand, BrandCode } from '@/lib/types';
import { cn } from '@/lib/cn';

export function BrandTabs({
  brands,
  current,
  onSelect,
}: {
  brands: Brand[];
  current: BrandCode;
  onSelect: (code: BrandCode) => void;
}) {
  return (
    <div className="flex gap-2 mb-[18px] flex-wrap">
      {brands.map((b) => {
        const active = b.code === current;
        return (
          <button
            key={b.code}
            onClick={() => onSelect(b.code)}
            style={{ ['--accent' as string]: b.accentColor }}
            className={cn(
              'relative flex items-center gap-[10px] px-4 py-[11px] rounded-[12px] bg-panel border min-w-[172px] transition shadow-sm2',
              'hover:-translate-y-px hover:shadow-md2',
              active ? 'border-transparent' : 'border-line',
            )}
            data-active={active}
          >
            <span
              className="grid place-items-center w-[30px] h-[30px] rounded-[9px] text-white font-extrabold text-[13px] flex-none"
              style={{ background: b.accentColor }}
            >
              {b.name[0]}
            </span>
            <span className="text-left">
              <span className="block font-bold text-[14px] leading-[1.2]">{b.name}</span>
              <span className="block text-[11.5px] text-muted">
                {b.code} · {b.domains[0]?.url.replace(/^https?:\/\//, '') ?? '未配域名'}
              </span>
            </span>
            <span
              className={cn(
                'ml-auto text-[12px] font-bold rounded-lg px-[9px] py-[3px]',
                active ? 'text-white' : 'text-ink-2 bg-bg',
              )}
              style={active ? { background: b.accentColor } : undefined}
            >
              {b.channelCount}
            </span>
            {active && (
              <span
                aria-hidden
                className="pointer-events-none absolute inset-0 rounded-[12px]"
                style={{ boxShadow: `0 0 0 2px ${b.accentColor} inset` }}
              />
            )}
          </button>
        );
      })}
    </div>
  );
}
