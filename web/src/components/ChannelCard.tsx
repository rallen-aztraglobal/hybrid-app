/**
 * 渠道卡片 —— 复刻原型 .card：图标/应用名/flavor/包名/PAL_CODE/域名健康点/状态 + 悬浮操作。
 * 健康点取「该渠道实际生效的域名清单」（继承品牌或自身覆盖）的健康度。
 */
import type { Brand, Channel, DomainEntry } from '@/lib/types';
import { iconInitials } from '@/lib/brands';
import { apkFileName } from '@/lib/text';
import { cn } from '@/lib/cn';
import { AppIcon, HealthDot } from './ui';
import { DownloadIcon, EditIcon, TrashIcon } from './icons';

export function ChannelCard({
  channel,
  brand,
  onEdit,
  onArchive,
}: {
  channel: Channel;
  brand: Brand;
  onEdit: () => void;
  onArchive: () => void;
}) {
  const on = channel.status === 'enabled';
  // 实际生效域名：继承品牌则用 brand.domains，否则用渠道覆盖
  const effective: DomainEntry[] = channel.useBrandDomains
    ? brand.domains
    : channel.domains ?? [];

  return (
    <div
      className={cn(
        'group relative bg-panel border border-line rounded-card p-4 shadow-sm2 transition overflow-hidden',
        'hover:-translate-y-[3px] hover:shadow-md2 hover:border-[#dfe6f0]',
      )}
    >
      {/* 顶部品牌色条（hover 显现） */}
      <span
        className="absolute top-0 left-0 right-0 h-[3px] opacity-0 group-hover:opacity-100 transition"
        style={{ background: brand.accentColor }}
      />

      <span
        className={cn(
          'absolute top-[14px] right-[14px] text-[11px] font-semibold px-[9px] py-[3px] rounded-full',
          on ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
        )}
      >
        {on ? '启用中' : channel.status === 'archived' ? '已归档' : '已停用'}
      </span>

      <div className="flex gap-[13px] items-start">
        <AppIcon
          initials={iconInitials(channel.appName, channel.brandCode)}
          hex={brand.accentColor}
          src={channel.iconMasterUrl}
        />
        <div className="min-w-0 flex-1 pr-16">
          <div className="font-bold text-[14.5px] truncate" title={channel.appName}>
            {channel.appName}
          </div>
          <span
            className="inline-block mt-1 text-[11.5px] font-mono px-2 py-0.5 rounded-md"
            style={{ color: 'var(--brand-ink)', background: 'rgba(99,102,241,.1)' }}
          >
            {channel.flavorName}
          </span>
        </div>
      </div>

      <div className="mt-[13px] flex flex-col gap-[7px]">
        <Row k="包名" v={channel.applicationId} />
        <Row k="PAL_CODE" v={channel.palCode} />
      </div>

      <div className="mt-[14px] pt-[13px] border-t border-line-2 flex items-center gap-[10px]">
        <div className="flex items-center gap-[6px] text-[11.5px] text-muted" title="主域名 + 备用域名健康状态">
          {effective.length > 0 ? (
            effective.map((d) => <HealthDot key={d.position} health={d.health ?? 'unknown'} />)
          ) : (
            <HealthDot health="unconfigured" />
          )}
          <span>
            {effective.length} 域名{channel.useBrandDomains ? '（继承）' : '（覆盖）'}
          </span>
        </div>
        <div className="ml-auto flex gap-[6px] items-center">
          {channel.latestApkUrl && (
            <a
              href={channel.latestApkUrl}
              download={apkFileName(channel.latestApkUrl)}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              title="下载该渠道最近一次成功构建的 APK"
              className="inline-flex items-center gap-1 text-[11.5px] font-semibold px-[9px] py-[5px] rounded-lg border border-line bg-panel text-brand hover:bg-[rgba(99,102,241,.06)] transition"
            >
              <DownloadIcon className="w-[14px] h-[14px]" />
              下载最新包
            </a>
          )}
          <div className="flex gap-[6px] opacity-0 group-hover:opacity-100 transition">
            <MiniBtn title="编辑" onClick={onEdit}>
              <EditIcon className="w-[15px] h-[15px]" />
            </MiniBtn>
            <MiniBtn
              title="归档"
              danger
              onClick={(e) => {
                e.stopPropagation();
                if (confirm('确认归档该渠道？（软删除，保留归因历史）')) onArchive();
              }}
            >
              <TrashIcon className="w-[15px] h-[15px]" />
            </MiniBtn>
          </div>
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
