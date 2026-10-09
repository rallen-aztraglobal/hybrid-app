/**
 * 位置开关条（原型 .pos-panel）：8 个位置、建议优先级徽章、各位置弹窗数、开关。
 * 位置开关是全局的：需 popup:edit + 全量数据范围，其余只读；后端 403 时 toast 提示。
 */
import { useAllPopups, usePopupPositions, useSetPopupPosition } from '@/hooks/queries';
import { ApiError } from '@/lib/api';
import { cn } from '@/lib/cn';
import { POPUP_POSITION_CODES, POSITION_CAPS } from '@/lib/popupMeta';
import { toast } from '@/store/toastStore';
import type { PopupPositionCode } from '@/lib/types';
import { LayersIcon, LockIcon } from '../icons';
import { Switch } from '../ui';
import { PriorityBadge } from './bits';

export function PopupPositionBar({ canEdit, fullScope }: { canEdit: boolean; fullScope: boolean }) {
  const { data: positions } = usePopupPositions();
  const { data: popups } = useAllPopups();
  const setPos = useSetPopupPosition();

  const canToggle = canEdit && fullScope;
  const countOf = (code: PopupPositionCode) => (popups ?? []).filter((p) => p.position === code).length;
  const enabledOf = (code: PopupPositionCode) => positions?.find((p) => p.code === code)?.enabled ?? false;

  async function toggle(code: PopupPositionCode, enabled: boolean) {
    try {
      await setPos.mutateAsync({ code, enabled });
      toast.ok(`${code} ${POSITION_CAPS[code].name} 已${enabled ? '开启' : '关闭'}`);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) toast.err('仅全量数据权限账号可修改');
      else toast.err(err instanceof Error ? err.message : '修改位置开关失败');
    }
  }

  const lockText = !canEdit
    ? '当前账号无编辑权限，仅可查看'
    : !fullScope
      ? '当前账号无全量数据权限，仅可查看'
      : '仅全量数据权限的账号可修改';

  return (
    <div className="section-card !p-4 pb-3 mb-4">
      <div className="flex items-center gap-[10px] flex-wrap mb-3">
        <h3 className="flex items-center gap-[7px] text-[13.5px] font-bold">
          <LayersIcon className="w-4 h-4 text-brand" />
          位置开关
          <span className="font-normal text-muted text-[11.5px]">· 全局，按位置开 / 关；关闭后该位置下所有弹窗及其便条停止下发</span>
        </h3>
        <span
          className={cn(
            'ml-auto inline-flex items-center gap-[5px] text-[11.5px] font-semibold px-[10px] py-[3px] rounded-full',
            canToggle ? 'text-[#92681a] bg-[#fef3c7]' : 'text-[#b91c1c] bg-[#fee2e2]',
          )}
        >
          <LockIcon className="w-3 h-3" />
          {lockText}
        </span>
      </div>
      <div className="grid gap-[10px] grid-cols-2 min-[1380px]:grid-cols-4">
        {POPUP_POSITION_CODES.map((code) => {
          const caps = POSITION_CAPS[code];
          const on = enabledOf(code);
          return (
            <div
              key={code}
              className={cn(
                'flex items-start gap-[10px] px-3 py-[11px] rounded-xl border transition',
                on ? 'bg-white' : 'bg-panel-2 border-line',
              )}
              style={on ? { borderColor: 'rgba(99,102,241,.35)', boxShadow: '0 0 0 1px rgba(99,102,241,.1) inset' } : undefined}
            >
              <span
                className={cn('grid place-items-center flex-none w-8 h-8 rounded-[9px] font-mono font-extrabold text-[12px]', on ? 'text-white' : 'bg-[#eef2f7] text-[#64748b]')}
                style={on ? { background: 'linear-gradient(135deg,var(--brand),#8b5cf6)', boxShadow: '0 6px 14px -6px rgba(99,102,241,.8)' } : undefined}
              >
                {code}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-[6px] font-bold text-[13px]">
                  {caps.name}
                  <PriorityBadge level={caps.priority} />
                </div>
                <div className="text-[11.5px] text-muted mt-0.5 truncate" title={caps.desc}>
                  {caps.desc}
                </div>
                <div className="text-[11px] text-ink-2 mt-1">
                  {countOf(code)} 个弹窗 · {on ? <span className="text-[#15803d] font-bold">已开启</span> : '未开启'}
                </div>
              </div>
              <div
                title={canToggle ? (on ? '点击关闭该位置' : '点击开启该位置') : `无权限：${lockText}`}
                className="pt-0.5"
              >
                <Switch checked={on} disabled={!canToggle || setPos.isPending || !positions} onChange={(v) => void toggle(code, v)} />
              </div>
            </div>
          );
        })}
      </div>
      <div className="text-[11.5px] text-muted mt-[10px]">
        默认开启 P1 / P2 / P8，其余位置代码已实现、默认关闭。开关属运营（还要不要投），频控属产品（写死在客户端，见编辑页只读区）。
      </div>
    </div>
  );
}
