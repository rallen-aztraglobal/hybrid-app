/**
 * 投放列表：筛选（位置 / 品牌 / 状态 / 关键字）+ 弹窗列表（原型视图 1）。
 * 行内：缩略图（轮播张数角标）、名称、位置徽章（P4 强制「不可关闭」红徽章、位置未开启提示）、
 * 状态、优先级、生效期、定向摘要、行为小标签、启用开关、编辑 / 数据 / 删除。
 */
import { useEffect, useMemo, useState } from 'react';
import { useDeletePopup, usePopups, useSetPopupEnabled } from '@/hooks/queries';
import { BRAND_META, CHANNEL_BRAND_ORDER } from '@/lib/brands';
import { cn } from '@/lib/cn';
import { POPUP_POSITION_CODES, POSITION_CAPS, STATUS_LABEL, isForcedMode } from '@/lib/popupMeta';
import { formatRange } from '@/lib/popupView';
import { toast } from '@/store/toastStore';
import type { BrandCode, Popup, PopupPositionCode, PopupStatus } from '@/lib/types';
import { AlertIcon, ChartIcon, EditIcon, EyeIcon, LockIcon, PlusIcon, RefreshIcon, SearchIcon, TabNoteIcon, TimerIcon, TrashIcon } from '../icons';
import { ConfirmDialog } from '../ConfirmDialog';
import { Button, Select, Switch } from '../ui';
import { BrandTag, PopupThumb, PosBadge, ForceBadge, StatusBadge, Tag } from './bits';

const STATUS_OPTIONS: { value: '' | PopupStatus; label: string }[] = [
  { value: '', label: '全部状态' },
  { value: 'active', label: `● ${STATUS_LABEL.active}` },
  { value: 'scheduled', label: `● ${STATUS_LABEL.scheduled}` },
  { value: 'ended', label: `● ${STATUS_LABEL.ended}` },
  { value: 'disabled', label: `● ${STATUS_LABEL.disabled}` },
];

const BC = 'inline-flex items-center gap-[3px] text-[11px] font-semibold px-[6px] py-[2px] rounded-[5px] mr-[3px] my-px whitespace-nowrap';

function TargetingCell({ p }: { p: Popup }) {
  const ver = [p.minVersion && `≥${p.minVersion}`, p.maxVersion && `<${p.maxVersion}`].filter(Boolean).join(' 且 ');
  const user = { all: '新老用户', new: '新用户', old: '老用户' }[p.userType];
  return (
    <>
      <div className="flex gap-1 flex-wrap my-0.5">
        {p.brandCodes.length ? p.brandCodes.map((b) => <BrandTag key={b} code={b} />) : <Tag>全部品牌</Tag>}
        <Tag>{p.appIds.length ? `${p.appIds.length} 个包` : '全部包'}</Tag>
      </div>
      <div className="flex gap-1 flex-wrap my-0.5">
        <Tag>{ver ? `版本 ${ver}` : '全版本'}</Tag>
        <Tag>{user}</Tag>
        <Tag>{p.countries.length ? `地区 ${p.countries.join(' / ')}` : '全部地区'}</Tag>
      </div>
    </>
  );
}

function BehaviorCell({ p }: { p: Popup }) {
  const tags: JSX.Element[] = [];
  if (isForcedMode(p.position, p.closable)) {
    tags.push(
      <span key="force" className={cn(BC, 'bg-[#fee2e2] text-[#b91c1c]')}>
        <LockIcon className="w-[11px] h-[11px]" />
        不可关闭
      </span>,
    );
  }
  if (p.tabEnabled) {
    tags.push(
      <span key="tab" className={cn(BC, 'bg-[#eef2ff] text-brand-ink')} title={`关闭后收起为便条：「${p.tabText}」`}>
        <TabNoteIcon className="w-3 h-3" />
        便条
      </span>,
    );
  }
  if (p.countdown) {
    tags.push(
      <span key="cd" className={cn(BC, 'bg-[#eef2ff] text-brand-ink')} title="倒计时关闭 · 时长固定 3 秒">
        <TimerIcon className="w-3 h-3" />
        3s
      </span>,
    );
  }
  if (p.badge) tags.push(<span key="badge" className={cn(BC, 'bg-[#eef2ff] text-brand-ink')}>红点</span>);
  if (p.position === 'P3') tags.push(<span key="ap" className={cn(BC, 'bg-[#eef2ff] text-brand-ink')} title="自动轮播间隔"><RefreshIcon className="w-[11px] h-[11px]" />{p.autoplaySeconds}s</span>);
  return tags.length ? <>{tags}</> : <span className="text-muted">—</span>;
}

export function PopupList({
  canEdit,
  onNew,
  onEdit,
  onData,
}: {
  canEdit: boolean;
  onNew: () => void;
  onEdit: (id: number) => void;
  onData: (id: number) => void;
}) {
  const [position, setPosition] = useState<'' | PopupPositionCode>('');
  const [brand, setBrand] = useState<'' | BrandCode>('');
  const [status, setStatus] = useState<'' | PopupStatus>('');
  const [keyword, setKeyword] = useState('');
  const [kwDebounced, setKwDebounced] = useState('');
  useEffect(() => {
    const t = window.setTimeout(() => setKwDebounced(keyword), 300);
    return () => window.clearTimeout(t);
  }, [keyword]);
  const [delTarget, setDelTarget] = useState<Popup | null>(null);

  const filter = useMemo(
    () => ({ position: position || undefined, brand: brand || undefined, status: status || undefined, keyword: kwDebounced.trim() || undefined }),
    [position, brand, status, kwDebounced],
  );
  const { data: list, isLoading } = usePopups(filter);
  const setEnabled = useSetPopupEnabled();
  const del = useDeletePopup();

  async function toggle(p: Popup, enabled: boolean) {
    try {
      await setEnabled.mutateAsync({ id: p.id, enabled });
      toast.ok(`「${p.name}」已${enabled ? '启用' : '关闭'}`);
    } catch (err) {
      toast.err(err instanceof Error ? err.message : '操作失败');
    }
  }

  async function doDelete() {
    if (!delTarget) return;
    try {
      await del.mutateAsync(delTarget.id);
      toast.ok(`已删除「${delTarget.name}」`);
      setDelTarget(null);
    } catch (err) {
      toast.err(err instanceof Error ? err.message : '删除失败');
    }
  }

  return (
    <>
      <div className="flex items-center gap-3 mb-[14px] flex-wrap">
        <div className="flex gap-[10px] items-center flex-wrap">
          <div className="w-[190px]">
            <Select
              value={position}
              onChange={(v) => setPosition(v as '' | PopupPositionCode)}
              options={[
                { value: '', label: '全部位置' },
                ...POPUP_POSITION_CODES.map((c) => ({ value: c, label: `${c} · ${POSITION_CAPS[c].name}` })),
              ]}
            />
          </div>
          <div className="flex gap-[6px] items-center pl-[10px] border-l border-line">
            <button className={cn('chip', brand === '' && 'chip-on')} onClick={() => setBrand('')}>
              全部品牌
            </button>
            {CHANNEL_BRAND_ORDER.map((b) => (
              <button key={b} className={cn('chip inline-flex items-center gap-[6px]', brand === b && 'chip-on')} onClick={() => setBrand(b)}>
                <i className="w-2 h-2 rounded-full inline-block" style={{ background: BRAND_META[b].accentColor }} />
                {BRAND_META[b].name}
              </button>
            ))}
          </div>
          <div className="w-[150px] pl-[10px] border-l border-line box-content">
            <Select value={status} onChange={(v) => setStatus(v as '' | PopupStatus)} options={STATUS_OPTIONS} />
          </div>
        </div>
        <div className="ml-auto flex gap-[10px] items-center">
          <div className="flex items-center gap-2 bg-panel border border-line rounded-[10px] px-[11px] py-[7px] w-[210px] text-muted">
            <SearchIcon className="w-[15px] h-[15px]" />
            <input
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="搜索名称"
              className="border-none bg-transparent outline-none w-full text-ink text-[13px] placeholder:text-muted"
            />
          </div>
          {canEdit && (
            <Button variant="primary" onClick={onNew}>
              <PlusIcon />
              新建弹窗
            </Button>
          )}
        </div>
      </div>

      <div className="rounded-card border border-line bg-panel shadow-sm2 overflow-auto">
        {isLoading ? (
          <div className="text-center text-muted py-[50px]">加载中…</div>
        ) : !list || list.length === 0 ? (
          <div className="text-center text-muted py-[50px]">该筛选下暂无弹窗</div>
        ) : (
          <table className="w-full border-separate border-spacing-0 text-[12.5px]">
            <thead>
              <tr>
                {['素材', '名称', '位置', '状态', '优先级', '生效时间', '定向', '行为', '启用', '操作'].map((h) => (
                  <th key={h} className="text-left text-[11.5px] font-semibold text-muted px-3 py-[10px] bg-panel-2 border-b border-line whitespace-nowrap">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {list.map((p) => (
                <tr key={p.id} className="hover:bg-[#fafbff] [&>td]:border-b [&>td]:border-line-2 last:[&>td]:border-b-0">
                  <td className="px-3 py-[11px] align-middle">
                    <PopupThumb position={p.position} imageUrl={p.cards[0]?.imageUrl} count={p.cards.length} />
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <div className="font-bold text-[13px] text-ink">{p.name}</div>
                    <div className="text-[11px] text-muted font-mono">#{p.id}</div>
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <div className="whitespace-nowrap">
                      <PosBadge code={p.position} />
                      {isForcedMode(p.position, p.closable) && <ForceBadge label="不可关闭" />}
                    </div>
                    {!p.positionEnabled && (
                      <div className="flex items-center gap-1 text-[11px] text-[#b45309] mt-[5px] font-semibold whitespace-nowrap">
                        <AlertIcon className="w-3 h-3" />
                        位置未开启 · 不下发
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <StatusBadge status={p.status} />
                  </td>
                  <td className="px-3 py-[11px] align-middle font-mono font-bold">{p.priority}</td>
                  <td className="px-3 py-[11px] align-middle font-mono text-[11.5px] text-ink-2 whitespace-nowrap leading-[1.6]">
                    {formatRange(p.startAt, p.endAt).map((l, i) => (
                      <div key={i}>{l}</div>
                    ))}
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <TargetingCell p={p} />
                  </td>
                  <td className="px-3 py-[11px] align-middle max-w-[150px]">
                    <BehaviorCell p={p} />
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <span title={canEdit ? '单个弹窗开关：关闭即停投' : '无 popup:edit 权限'}>
                      <Switch checked={p.enabled} disabled={!canEdit || setEnabled.isPending} onChange={(v) => void toggle(p, v)} />
                    </span>
                  </td>
                  <td className="px-3 py-[11px] align-middle">
                    <div className="flex gap-[6px]">
                      <IconBtn title={canEdit ? '编辑' : '查看'} onClick={() => onEdit(p.id)}>
                        {canEdit ? <EditIcon className="w-[15px] h-[15px]" /> : <EyeIcon className="w-[15px] h-[15px]" />}
                      </IconBtn>
                      <IconBtn title="数据" onClick={() => onData(p.id)}>
                        <ChartIcon className="w-[15px] h-[15px]" />
                      </IconBtn>
                      {canEdit && (
                        <IconBtn title="删除" danger onClick={() => setDelTarget(p)}>
                          <TrashIcon className="w-[15px] h-[15px]" />
                        </IconBtn>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <ConfirmDialog
        open={!!delTarget}
        title="删除弹窗"
        confirmText="删除"
        danger
        busy={del.isPending}
        onConfirm={() => void doDelete()}
        onCancel={() => setDelTarget(null)}
      >
        确认删除「{delTarget?.name}」？客户端下次拉取配置后将不再展示（含其便条）。历史统计仍会保留该弹窗名称。
      </ConfirmDialog>
    </>
  );
}

function IconBtn({
  children,
  title,
  danger,
  onClick,
}: {
  children: React.ReactNode;
  title: string;
  danger?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={cn(
        'grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 transition hover:bg-bg hover:text-ink',
        danger && 'hover:!text-down hover:!border-[#fecaca] hover:!bg-[#fef2f2]',
      )}
    >
      {children}
    </button>
  );
}
