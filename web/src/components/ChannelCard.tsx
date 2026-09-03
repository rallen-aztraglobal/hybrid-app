/**
 * 渠道卡片 —— 复刻原型 .card：图标/应用名/flavor/包名/PAL_CODE/线上版本号/域名健康点/状态 + 悬浮操作。
 * 健康点取「该渠道实际生效的域名清单」（继承品牌或自身覆盖）的健康度。
 * 「线上版本号」是人工备忘（包太多记不清上次发的版本），直接放在卡片正面、有权限时就地可改。
 */
import { useEffect, useRef, useState } from 'react';
import type { Brand, Channel, DomainEntry } from '@/lib/types';
import { iconInitials } from '@/lib/brands';
import { friendlyNotFoundMessage } from '@/lib/api';
import { useSaveChannelLiveVersion, useSigningKeys } from '@/hooks/queries';
import { apkFileName } from '@/lib/text';
import { cn } from '@/lib/cn';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { AppIcon, HealthDot, SigningKeyBadge } from './ui';
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
  const canEdit = useAuthStore((s) => s.hasPerm(PERM.CHANNEL_EDIT));
  const canArchive = useAuthStore((s) => s.hasPerm(PERM.CHANNEL_ARCHIVE));
  const { data: signingKeys } = useSigningKeys();
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
          <div className="flex flex-wrap items-center gap-1.5 mt-1">
            <span
              className="inline-block text-[11.5px] font-mono px-2 py-0.5 rounded-md"
              style={{ color: 'var(--brand-ink)', background: 'rgba(99,102,241,.1)' }}
            >
              {channel.flavorName}
            </span>
            {channel.store?.name && (
              <span
                className="inline-block text-[11px] font-semibold px-2 py-0.5 rounded-md text-[#92681a] bg-[#fef3c7]"
                title={`应用商店：${channel.store.name}`}
              >
                {channel.store.name}
              </span>
            )}
            <SigningKeyBadge signingKey={channel.signingKey} keys={signingKeys} />
          </div>
        </div>
      </div>

      <div className="mt-[13px] flex flex-col gap-[7px]">
        <Row k="包名" v={channel.applicationId} />
        <Row k="PAL_CODE" v={channel.palCode} />
        <LiveVersionRow channel={channel} editable={canEdit} />
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
          {/* 编辑/归档：悬浮显现，靠左；无权限直接不渲染（10-rbac.md） */}
          {(canEdit || canArchive) && (
            <div className="flex gap-[6px] opacity-0 group-hover:opacity-100 transition">
              {canEdit && (
                <MiniBtn title="编辑" onClick={onEdit}>
                  <EditIcon className="w-[15px] h-[15px]" />
                </MiniBtn>
              )}
              {canArchive && (
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
              )}
            </div>
          )}
          {/* 下载最新包：纯图标、固定在最右（小屏不挤文字）。tooltip 说明用途。 */}
          {channel.latestApkUrl && (
            <a
              href={channel.latestApkUrl}
              download={apkFileName(channel.latestApkUrl)}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              title="下载该渠道最近一次成功构建的 APK"
              aria-label="下载最新包"
              className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-brand hover:bg-[rgba(99,102,241,.06)] transition"
            >
              <DownloadIcon className="w-[15px] h-[15px]" />
            </a>
          )}
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

/**
 * 「线上版本号」行：人工备忘、就地编辑。
 * 有 channel:edit 权限 → 一个贴合行高的小输入框：回车/失焦保存（只 PUT 这一个字段）、Esc 放弃；
 * 无权限 → 只读展示。保存成功后由 query 失效刷新列表，草稿随之同步；编辑中不被外部刷新打断。
 */
function LiveVersionRow({ channel, editable }: { channel: Channel; editable: boolean }) {
  const save = useSaveChannelLiveVersion();
  const current = channel.liveVersion ?? '';
  const [draft, setDraft] = useState(current);
  const [focused, setFocused] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  // Esc 放弃：blur 会紧跟着触发，用 ref 让本次 blur 跳过提交（state 更新是异步的，闭包里拿不到复位值）。
  const cancelRef = useRef(false);

  useEffect(() => {
    if (!focused) setDraft(current);
  }, [current, focused]);

  if (!editable) return <Row k="线上版本" v={current || '未记录'} />;

  function commit() {
    const next = draft.trim();
    setDraft(next);
    if (next === current) return;
    save.mutate(
      { id: channel.id, liveVersion: next },
      {
        onSuccess: () => setErr(null),
        // 越界渠道后端故意 404（10-rbac.md「数据权限」）：文案走统一转换，并把草稿退回原值。
        onError: (e) => {
          setErr(friendlyNotFoundMessage(e, '保存失败'));
          setDraft(current);
        },
      },
    );
  }

  return (
    <div className="flex items-center gap-2 text-[12px]">
      <span className="text-muted w-16 flex-none">线上版本</span>
      <input
        className={cn(
          'font-mono text-ink-2 flex-1 min-w-0 h-[24px] px-1.5 -ml-1.5 rounded-md bg-transparent transition',
          'border border-transparent hover:border-line focus:border-brand focus:bg-panel focus:outline-none',
          'placeholder:font-sans placeholder:text-muted/70 disabled:opacity-60',
          err ? 'border-[#fecaca]' : '',
        )}
        placeholder="未记录，点此填写"
        maxLength={32}
        value={draft}
        disabled={save.isPending}
        title="线上版本号（人工备忘）：回车或失焦保存，Esc 放弃"
        aria-label="线上版本号"
        onFocus={() => setFocused(true)}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.currentTarget.blur();
          } else if (e.key === 'Escape') {
            cancelRef.current = true;
            e.currentTarget.blur();
          }
        }}
        onBlur={() => {
          setFocused(false);
          if (cancelRef.current) {
            cancelRef.current = false;
            setDraft(current);
            return;
          }
          commit();
        }}
      />
      {save.isPending && <span className="text-[11px] text-muted flex-none">保存中…</span>}
      {err && !save.isPending && (
        <span className="text-[11px] text-down flex-none max-w-[45%] truncate" title={err}>
          {err}
        </span>
      )}
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
