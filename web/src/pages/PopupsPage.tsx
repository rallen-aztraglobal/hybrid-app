/**
 * 弹窗管理（12-popup.md / docs/admin/ui/popup.html）—— 马甲包原生弹窗的后台配置与数据看板。
 *
 * 子 Tab：
 *  - 投放列表：位置开关条 + 筛选 + 列表；新建 / 编辑走右侧抽屉（左表单 + 右手机实时预览）。
 *  - 数据看板：KPI + 11 指标主表 + 拦截 / 关闭方式分布；列表行「数据」按钮跳转并筛选该弹窗。
 * 权限：page:popups 可见页面；popup:edit 才能新建 / 编辑 / 删除 / 开关 / 上传；
 * 位置开关还要求全量数据范围（品牌 / 渠道受限账号只读，后端 403 时 toast 提示）。
 */
import { useMemo, useState } from 'react';
import { useAllPopups } from '@/hooks/queries';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { cn } from '@/lib/cn';
import { InfoIcon, ChartIcon, ListIcon } from '@/components/icons';
import { PopupDashboard } from '@/components/popup/PopupDashboard';
import { PopupDrawer } from '@/components/popup/PopupDrawer';
import { PopupList } from '@/components/popup/PopupList';
import { PopupPositionBar } from '@/components/popup/PopupPositionBar';
import { PopupRuntimePreview } from '@/components/popup/PopupRuntimePreview';

type SubTab = 'list' | 'data';

export function PopupsPage() {
  const [tab, setTab] = useState<SubTab>('list');
  const [drawer, setDrawer] = useState<'new' | number | null>(null);
  const [dashPopup, setDashPopup] = useState<number | null>(null);

  const hasPerm = useAuthStore((s) => s.hasPerm);
  const scope = useAuthStore((s) => s.user?.scope);
  const canEdit = hasPerm(PERM.POPUP_EDIT);
  // scope 缺失（旧后端 / 超管兜底）按全量处理；任一维度受限即视为「非全量」。
  const scopeFlags = useMemo(
    () => ({ allBrands: scope?.allBrands ?? true, allChannels: scope?.allChannels ?? true }),
    [scope],
  );
  const fullScope = scopeFlags.allBrands && scopeFlags.allChannels;

  const { data: all } = useAllPopups();

  return (
    <div>
      <div className="flex items-center gap-[14px] mb-[18px] flex-wrap">
        <div className="inline-flex gap-1 p-1 rounded-xl bg-[#e8ecf3]">
          {(
            [
              ['list', '投放列表', ListIcon, all?.length],
              ['data', '数据看板', ChartIcon, undefined],
            ] as const
          ).map(([k, label, Icon, n]) => (
            <button
              key={k}
              onClick={() => setTab(k)}
              className={cn(
                'flex items-center gap-2 px-[15px] py-2 rounded-[9px] text-[13px] font-semibold transition',
                tab === k ? 'bg-panel text-brand-ink shadow-sm2' : 'text-ink-2 hover:text-ink',
              )}
            >
              <Icon className="w-4 h-4" />
              {label}
              {n != null && <span className="text-[11px] px-[7px] rounded-full" style={{ background: 'rgba(99,102,241,.12)', color: 'var(--brand-ink)' }}>{n}</span>}
            </button>
          ))}
        </div>
        <div className="ml-auto flex items-center gap-[6px] text-[12px] text-muted">
          <InfoIcon className="w-[14px] h-[14px]" />
          马甲包原生弹窗 · 配置改动无需重新打包，按客户端拉取时机生效
        </div>
      </div>

      {tab === 'list' ? (
        <>
          <PopupPositionBar canEdit={canEdit} fullScope={fullScope} />
          <PopupList
            canEdit={canEdit}
            onNew={() => setDrawer('new')}
            onEdit={(id) => setDrawer(id)}
            onData={(id) => {
              setDashPopup(id);
              setTab('data');
            }}
          />
          <PopupRuntimePreview />
        </>
      ) : (
        <PopupDashboard popupId={dashPopup} onPopupChange={setDashPopup} />
      )}

      <PopupDrawer target={drawer} canEdit={canEdit} scope={scopeFlags} onClose={() => setDrawer(null)} />
    </div>
  );
}
