/** 编辑抽屉的两个只读区：频控（§7.3，写死在客户端）与「保存后多久生效」提示框。 */
import { cn } from '@/lib/cn';
import { FREQ_ROWS, POSITION_CAPS, currentFreqRowKey, isForcedMode } from '@/lib/popupMeta';
import type { PopupPositionCode } from '@/lib/types';
import { AlertIcon, ClockIcon, LockIcon } from '../icons';

export function PopupFrequency({ position, closable }: { position: PopupPositionCode; closable: boolean }) {
  const cur = currentFreqRowKey(position, isForcedMode(position, closable));
  return (
    <div className="rounded-[14px] border border-dashed border-[#cbd5e1] bg-[#f8fafc] p-[18px]">
      <h3 className="flex items-center gap-2 text-[13px] font-bold text-ink mb-[14px] flex-wrap">
        <span className="grid place-items-center w-5 h-5 rounded-md bg-[#94a3b8] text-white">
          <LockIcon className="w-3 h-3" />
        </span>
        频控
        <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-[#64748b] bg-[#e2e8f0] px-2 py-0.5 rounded-full">
          <LockIcon className="w-[11px] h-[11px]" />
          只读 · 写死在客户端
        </span>
      </h3>
      <table className="w-full border-collapse text-[12px] bg-white rounded-lg overflow-hidden border border-line">
        <thead>
          <tr>
            {['位置', '自动展示条件', '关闭后'].map((h) => (
              <th key={h} className="bg-[#f1f5f9] text-[#64748b] font-semibold text-left px-[10px] py-[7px] text-[11.5px]">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {FREQ_ROWS.map((r) => {
            const on = r.key === cur;
            return (
              <tr key={r.key} className={cn(on ? 'text-ink font-semibold bg-[#fafbff]' : 'text-[#64748b]')}>
                <td className="px-[10px] py-2 border-t border-line-2 whitespace-nowrap" style={on ? { boxShadow: 'inset 3px 0 0 var(--brand)' } : undefined}>
                  {r.label}
                </td>
                <td className="px-[10px] py-2 border-t border-line-2">{r.condition}</td>
                <td className="px-[10px] py-2 border-t border-line-2">{r.afterClose}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <div className="flex gap-[10px] text-[12px] mt-[10px] text-[#64748b]">
        <span className="flex-none w-14">触发时机</span>
        <b className="text-ink-2 font-semibold">{POSITION_CAPS[position].trigger}</b>
      </div>
      <div className="flex gap-[7px] items-center text-[12px] text-[#64748b] mt-[10px] pt-[10px] border-t border-dashed border-[#cbd5e1]">
        <LockIcon className="w-[13px] h-[13px] flex-none" />
        频控按设备记，写死在客户端，后台不提供配置；改动需发版。轮播按整个弹窗计一次频控；频控在真实渲染成功时写入。
      </div>
    </div>
  );
}

export function PopupEffectNote() {
  return (
    <div
      className="flex gap-[10px] p-[12px_13px] rounded-[10px] text-[12px] leading-[1.6] text-ink-2"
      style={{ background: '#fffbeb', border: '1px solid #fde68a' }}
    >
      <ClockIcon className="w-[17px] h-[17px] flex-none mt-px" style={{ color: '#d97706' }} />
      <div>
        <b className="text-ink">保存后多久生效？</b>（配置改动不需要重新打包）
        <ul className="list-disc ml-4 mt-[6px] mb-2 [&>li]:my-0.5">
          <li><b>冷启动</b>：必拉最新配置</li>
          <li><b>回前台</b>：距上次拉取 &gt; 5 分钟才重拉</li>
          <li><b>拉取失败</b>：沿用本地缓存配置；无缓存则本次不展示任何弹窗</li>
          <li><b>新素材</b>：拉到配置后后台预下载；未预热完成前沿用上一份配置</li>
        </ul>
        <div className="flex gap-[6px] items-center font-bold rounded-lg px-[10px] py-[6px] my-[6px]" style={{ color: '#92400e', background: '#fef3c7' }}>
          <AlertIcon className="w-[14px] h-[14px] flex-none" />
          最坏生效延迟 = 用户下次回前台。改完立刻看手机没变化，不是 bug。
        </div>
        <div>
          <b className="text-ink">紧急下线</b>（关开关）：最慢约 5 分钟 + 用户切换动作；需要更快须另做服务端推送，本期不含。
        </div>
        <details className="mt-2">
          <summary className="cursor-pointer font-semibold text-ink-2">可配置边界（PRD §6）</summary>
          <div className="grid grid-cols-2 gap-[10px] mt-2">
            <div className="bg-white border border-line rounded-lg px-[10px] py-2">
              <h4 className="text-[11.5px] font-bold mb-1" style={{ color: '#15803d' }}>随时改 · 按上述机制生效</h4>
              <p className="text-[11.5px] leading-[1.7]">图片 / 文案 / 跳转链接 · 轮播卡片增删与排序 · 生效时间段 · 定向条件 · 各位置开关与单个弹窗开关 · 优先级 · 便条图标与文案 · 倒计时开 / 关</p>
            </div>
            <div className="bg-white border border-line rounded-lg px-[10px] py-2">
              <h4 className="text-[11.5px] font-bold mb-1" style={{ color: '#b91c1c' }}>需要发版</h4>
              <p className="text-[11.5px] leading-[1.7]">频控数值 · 倒计时时长（固定 3 秒）· 便条尺寸 / 位置 / 两态切换逻辑 · 弹窗形态与交互 · 新增位置类型 · 埋点事件定义</p>
            </div>
          </div>
        </details>
      </div>
    </div>
  );
}
