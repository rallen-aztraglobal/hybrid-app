/**
 * 上架包卡片 —— 仿照 ChannelCard：名称/包名/平台/PAL_CODE 徽章/状态 + AB 面徽标（醒目）+ 悬浮操作。
 * gate_enabled 徽标是本卡片与 ChannelCard 最大的差异点，务必醒目（绿=AB 面已开 / 灰=仅 A 面）。
 */
import type { Listing } from '@/lib/types';
import { PLATFORM_META } from '@/lib/listingMeta';
import { cn } from '@/lib/cn';
import { AppIcon } from './ui';
import { EditIcon, TrashIcon } from './icons';

export function ListingCard({
  listing,
  onEdit,
  onDelete,
}: {
  listing: Listing;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const platformMeta = PLATFORM_META[listing.platform];
  const on = listing.status === 'enabled';
  const initials = listing.name.replace(/[^A-Za-z]/g, '').slice(0, 2).toUpperCase() || listing.name.slice(0, 2).toUpperCase();

  return (
    <div
      className={cn(
        'group relative bg-panel border border-line rounded-card p-4 shadow-sm2 transition overflow-hidden',
        'hover:-translate-y-[3px] hover:shadow-md2 hover:border-[#dfe6f0]',
      )}
    >
      {/* 顶部平台色条（hover 显现） */}
      <span
        className="absolute top-0 left-0 right-0 h-[3px] opacity-0 group-hover:opacity-100 transition"
        style={{ background: platformMeta.accentColor }}
      />

      <div className="flex items-start justify-between gap-2">
        <span
          className={cn(
            'text-[11px] font-semibold px-[9px] py-[3px] rounded-full',
            on ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
          )}
        >
          {on ? '启用中' : listing.status === 'archived' ? '已归档' : '已停用'}
        </span>
        {/* AB 面网关醒目徽标：绿=已开（真实用户可能命中 B 面），灰=仅 A 面（安全态）。 */}
        <span
          className={cn(
            'text-[11px] font-bold px-[9px] py-[3px] rounded-full whitespace-nowrap',
            listing.gateEnabled ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
          )}
          title={listing.gateEnabled ? 'AB 面网关已开启：命中规则的设备会看到 B 面' : '网关关闭：该包永远只显示 A 面'}
        >
          {listing.gateEnabled ? '● AB 面已开' : '○ 仅 A 面'}
        </span>
      </div>

      <div className="flex gap-[13px] items-start mt-2">
        <AppIcon initials={initials} hex={platformMeta.accentColor} size={52} radius={14} />
        <div className="min-w-0 flex-1">
          <div className="font-bold text-[14.5px] truncate" title={listing.name}>
            {listing.name}
          </div>
          <div className="flex flex-wrap items-center gap-1.5 mt-1">
            <span
              className="inline-block text-[11px] font-semibold px-2 py-0.5 rounded-md"
              style={{ color: platformMeta.accentColor, background: `${platformMeta.accentColor}1a` }}
            >
              {platformMeta.label}
            </span>
            {listing.palCode && (
              <span
                className="inline-block text-[11px] font-semibold px-2 py-0.5 rounded-md text-ink-2 bg-panel-2 border border-line mono"
                title="继承自参考渠道包的 PAL_CODE"
              >
                palcode {listing.palCode}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="mt-[13px] flex flex-col gap-[7px]">
        <Row k="包名" v={listing.bundleId} />
        <Row k="品牌" v={listing.brandCode.toUpperCase()} />
        <Row k="打开方式" v={listing.openMode === 'external' ? '外开（系统浏览器）' : '内开（App 内 WebView）'} />
      </div>

      <div className="mt-[14px] pt-[13px] border-t border-line-2 flex items-center gap-[10px]">
        <div className="text-[11.5px] text-muted truncate flex-1" title={listing.remark || undefined}>
          {listing.remark || ' '}
        </div>
        <div className="flex gap-[6px] opacity-0 group-hover:opacity-100 transition">
          <MiniBtn title="编辑" onClick={onEdit}>
            <EditIcon className="w-[15px] h-[15px]" />
          </MiniBtn>
          <MiniBtn
            title="删除"
            danger
            onClick={(e) => {
              e.stopPropagation();
              if (
                confirm(
                  `确认删除「${listing.name}」（${platformMeta.label}）？此操作不可恢复，将级联删除其域名覆盖 / 网关规则 / 判定流水。`,
                )
              ) {
                onDelete();
              }
            }}
          >
            <TrashIcon className="w-[15px] h-[15px]" />
          </MiniBtn>
        </div>
      </div>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-center gap-2 text-[12px]">
      <span className="text-muted w-16 flex-none">{k}</span>
      <span className="font-mono text-ink-2 truncate" title={v}>
        {v}
      </span>
    </div>
  );
}

function MiniBtn({
  children,
  danger,
  onClick,
  title,
}: {
  children: React.ReactNode;
  danger?: boolean;
  onClick: (e: React.MouseEvent) => void;
  title: string;
}) {
  return (
    <button
      title={title}
      onClick={onClick}
      className={cn(
        'grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 transition',
        danger ? 'hover:text-down hover:border-[#fecaca] hover:bg-[#fef2f2]' : 'hover:bg-bg hover:text-ink',
      )}
    >
      {children}
    </button>
  );
}
