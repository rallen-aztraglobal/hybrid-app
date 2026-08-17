/**
 * AB 面网关区块 —— 上架包抽屉里的「重点」小节（ADR-0014）：
 *  1) 总开关：countries 草稿为空时禁用（前端先拦一道，后端仍是最终闸）；
 *  2) 国家白名单（必填）：常用国家 chips 多选 + 手输补充，绝不提供 CN/US 选项，
 *     手输命中会被就地拒绝并提示；
 *  3) IP 白/黑名单（选填）：多行文本域，每行一个 CIDR / IP；
 *  4) 时区黑名单（选填）：chips 多选 + 手输，命中即强制 A（放在 IP 名单之后）；
 *  5) 试算工具 + 判定流水：均依赖已存在的 listingId（后端端点要求 :id），
 *     新建未保存时以提示替代。
 *
 * 注意：本区块只维护「草稿」状态，不直接发请求——由外层 ListingDrawer 的统一「保存」
 * 提交（api 层保证域名 → 网关规则 → 总开关的正确顺序，见 lib/api.ts applyListingSideEffects）。
 * 因此试算 / 判定流水明确基于「上次保存」的规则，草稿未保存时可能与试算结果不一致。
 */
import { useState } from 'react';
import type { ListingGateLog, ListingGateTestResult } from '@/lib/types';
import { COUNTRY_OPTIONS, FORCED_A_COUNTRIES, TIMEZONE_OPTIONS } from '@/lib/listingMeta';
import { useListingGateLogs, useTestListingGate } from '@/hooks/queries';
import { cn } from '@/lib/cn';
import { Button, Note, Switch } from './ui';
import { InfoIcon } from './icons';

export interface ListingGateDraft {
  countries: string[];
  timezoneDeny: string[];
  ipAllowText: string;
  ipDenyText: string;
}

export function ListingGateSection({
  listingId,
  canGate,
  gateEnabled,
  onGateEnabledChange,
  draft,
  onDraftChange,
}: {
  /** null = 尚未保存过（新建未提交），此时试算/流水不可用。 */
  listingId: string | null;
  /** listing:gate（10-rbac.md：AB 面网关配置与试算）；无权限时本区块整体只读。 */
  canGate: boolean;
  gateEnabled: boolean;
  onGateEnabledChange: (v: boolean) => void;
  draft: ListingGateDraft;
  onDraftChange: (next: ListingGateDraft) => void;
}) {
  const canEnable = draft.countries.length > 0;
  // 只在「当前关闭、且还不能开启」时禁用——已开启的开关任何时候都允许关闭，不因草稿变化被锁死。
  const switchDisabled = !canGate || (!gateEnabled && !canEnable);

  function patch(partial: Partial<ListingGateDraft>) {
    onDraftChange({ ...draft, ...partial });
  }

  return (
    <div className="flex flex-col gap-4">
      {!canGate && (
        <Note>
          <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
          <div>当前账号无 AB 面网关权限（listing:gate），以下配置为只读展示，试算工具已隐藏。</div>
        </Note>
      )}
      <div className="flex items-center gap-[10px] p-[11px_13px] bg-panel-2 border border-line rounded-[10px]">
        <Switch checked={gateEnabled} onChange={onGateEnabledChange} disabled={switchDisabled} />
        <div className="flex-1">
          <div className="text-[13px] font-semibold">开启 AB 面网关</div>
          <div className={cn('text-[11.5px]', !canEnable && 'text-down')}>
            {canEnable
              ? '开启后，命中下方规则的设备访问 B 面；关闭则该包永远只有 A 面'
              : '请先配置国家白名单（必填，且不能为空）才能开启'}
          </div>
        </div>
        {gateEnabled && (
          <span className="flex-none text-[11px] font-bold px-[9px] py-[3px] rounded-full text-[#15803d] bg-[#dcfce7]">
            已开启
          </span>
        )}
      </div>

      <div className={cn(!canGate && 'opacity-60 pointer-events-none')}>
        <div className="mb-[6px] text-[12.5px] font-semibold text-ink-2">
          国家白名单 <span className="text-down">*</span>{' '}
          <span className="font-normal text-muted text-[11.5px]">
            · ISO-3166-1 两位码；CN / US 强制走 A 面，不提供选项也不可手输加入
          </span>
        </div>
        <CountryPicker value={draft.countries} onChange={(countries) => patch({ countries })} />
      </div>

      <div className={cn('grid grid-cols-2 gap-3', !canGate && 'opacity-60 pointer-events-none')}>
        <div>
          <div className="mb-[6px] text-[12.5px] font-semibold text-ink-2">
            IP 白名单 <span className="font-normal text-muted text-[11.5px]">· 选填，每行一个 CIDR / IP</span>
          </div>
          <textarea
            className="field-input mono h-[92px] resize-none"
            placeholder={'203.0.113.0/24\n198.51.100.5'}
            value={draft.ipAllowText}
            onChange={(e) => patch({ ipAllowText: e.target.value })}
          />
        </div>
        <div>
          <div className="mb-[6px] text-[12.5px] font-semibold text-ink-2">
            IP 黑名单 <span className="font-normal text-muted text-[11.5px]">· 选填，命中即强制 A 面</span>
          </div>
          <textarea
            className="field-input mono h-[92px] resize-none"
            placeholder={'已知审核机房网段…'}
            value={draft.ipDenyText}
            onChange={(e) => patch({ ipDenyText: e.target.value })}
          />
        </div>
      </div>

      <div className={cn(!canGate && 'opacity-60 pointer-events-none')}>
        <div className="mb-[6px] text-[12.5px] font-semibold text-ink-2">
          时区黑名单 <span className="font-normal text-muted text-[11.5px]">· 选填；命中即强制 A 面（客户端上报，与 IP 黑名单同类）</span>
        </div>
        <TimezonePicker value={draft.timezoneDeny} onChange={(timezoneDeny) => patch({ timezoneDeny })} />
      </div>

      <Note>
        <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
        <div>
          判定顺序（任一步判 A 即短路，条件一律 AND）：总开关 → CN/US 强制 A → IP 黑名单 → 时区黑名单(选填) →
          国家未知则 A → 国家白名单 → IP 白名单(选填) → 全部通过才是 B 面。修改需点击抽屉底部「保存」才会生效；
          下方「试算」与「判定流水」反映的是<b>已保存</b>的规则，与尚未保存的草稿可能不一致。后端不接受「清空」国家白名单
          （必须始终非空），若已保存过规则、这里又清空到 0 个再保存，本次提交会<b>跳过网关规则这一项、保留上次保存的值</b>，
          不会被清空——如需彻底变更，请直接替换成新的国家清单。
        </div>
      </Note>

      {listingId ? (
        <>
          {canGate && <GateTestTool listingId={listingId} />}
          <GateLogsPanel listingId={listingId} />
        </>
      ) : (
        <Note>
          <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
          <div>保存该上架包后，即可使用「试算」工具与「判定流水」核对 AB 面判定结果。</div>
        </Note>
      )}
    </div>
  );
}

// ---------- 国家白名单选择器 ----------
function CountryPicker({ value, onChange }: { value: string[]; onChange: (next: string[]) => void }) {
  const [input, setInput] = useState('');
  const [warn, setWarn] = useState<string | null>(null);
  const selected = new Set(value.map((c) => c.toUpperCase()));
  const extra = value.filter((c) => !COUNTRY_OPTIONS.some((o) => o.code === c.toUpperCase()));

  function toggle(code: string) {
    setWarn(null);
    if (selected.has(code)) onChange(value.filter((c) => c.toUpperCase() !== code));
    else onChange([...value, code]);
  }

  function addCustom() {
    const code = input.trim().toUpperCase();
    setInput('');
    if (!code) return;
    if (!/^[A-Z]{2}$/.test(code)) {
      setWarn(`「${code}」不是合法的两位国家码（ISO-3166-1，如 PH）`);
      return;
    }
    if (FORCED_A_COUNTRIES.includes(code)) {
      setWarn(`${code} 被强制走 A 面，不可加入白名单`);
      return;
    }
    if (selected.has(code)) {
      setWarn(`${code} 已在白名单中`);
      return;
    }
    setWarn(null);
    onChange([...value, code]);
  }

  return (
    <div>
      <div className="flex flex-wrap gap-[6px] mb-2">
        {COUNTRY_OPTIONS.map((o) => (
          <button
            key={o.code}
            type="button"
            onClick={() => toggle(o.code)}
            className={cn('chip px-[10px] py-1 text-[12px]', selected.has(o.code) && 'chip-on')}
          >
            {o.name} {o.code}
          </button>
        ))}
      </div>
      {extra.length > 0 && (
        <div className="flex flex-wrap gap-[6px] mb-2">
          {extra.map((code) => (
            <button
              key={code}
              type="button"
              onClick={() => toggle(code.toUpperCase())}
              className="chip chip-on mono px-[10px] py-1 text-[12px]"
              title="点击移除"
            >
              {code.toUpperCase()} ×
            </button>
          ))}
        </div>
      )}
      <div className="flex gap-2">
        <input
          className="field-input mono flex-1"
          placeholder="手输补充国家码，如 KH（回车或点击添加）"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              addCustom();
            }
          }}
        />
        <Button onClick={addCustom}>添加</Button>
      </div>
      {warn && <div className="mt-1 text-[12px] text-down">{warn}</div>}
      <div className="mt-2 text-[11.5px] text-muted">
        已选 {value.length} 个国家{value.length === 0 && '（必填；为空则无法开启 AB 面，后端也会拒绝保存）'}
      </div>
    </div>
  );
}

// ---------- 时区黑名单选择器 ----------
function TimezonePicker({ value, onChange }: { value: string[]; onChange: (next: string[]) => void }) {
  const [input, setInput] = useState('');
  const selected = new Set(value);
  const extra = value.filter((t) => !TIMEZONE_OPTIONS.includes(t));

  function toggle(tz: string) {
    if (selected.has(tz)) onChange(value.filter((t) => t !== tz));
    else onChange([...value, tz]);
  }

  function addCustom() {
    const tz = input.trim();
    setInput('');
    if (!tz || selected.has(tz)) return;
    onChange([...value, tz]);
  }

  return (
    <div>
      <div className="flex flex-wrap gap-[6px] mb-2">
        {TIMEZONE_OPTIONS.map((tz) => (
          <button
            key={tz}
            type="button"
            onClick={() => toggle(tz)}
            className={cn('chip mono px-[10px] py-1 text-[12px]', selected.has(tz) && 'chip-on')}
          >
            {tz}
          </button>
        ))}
      </div>
      {extra.length > 0 && (
        <div className="flex flex-wrap gap-[6px] mb-2">
          {extra.map((tz) => (
            <button
              key={tz}
              type="button"
              onClick={() => toggle(tz)}
              className="chip chip-on mono px-[10px] py-1 text-[12px]"
              title="点击移除"
            >
              {tz} ×
            </button>
          ))}
        </div>
      )}
      <div className="flex gap-2">
        <input
          className="field-input mono flex-1"
          placeholder="手输 IANA 时区，如 Asia/Manila（回车或点击添加）"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              addCustom();
            }
          }}
        />
        <Button onClick={addCustom}>添加</Button>
      </div>
      <div className="mt-2 text-[11.5px] text-muted">选填；命中黑名单的时区强制走 A 面，为空表示不按时区拦截。</div>
    </div>
  );
}

// ---------- 试算工具 ----------
function GateTestTool({ listingId }: { listingId: string }) {
  const test = useTestListingGate();
  const [ip, setIp] = useState('');
  const [timezone, setTimezone] = useState('');
  const [result, setResult] = useState<ListingGateTestResult | null>(null);

  async function run() {
    if (!ip.trim()) return;
    try {
      const r = await test.mutateAsync({ id: listingId, ip: ip.trim(), timezone: timezone.trim() });
      setResult(r);
    } catch {
      setResult(null);
    }
  }

  return (
    <div className="rounded-[10px] border border-line bg-panel-2 p-3">
      <div className="text-[12.5px] font-semibold text-ink-2 mb-2">用某 IP 试算（基于当前已保存的规则，保存前自查用）</div>
      <div className="flex flex-wrap gap-2">
        <input
          className="field-input mono flex-1 min-w-[160px]"
          placeholder="IP，如 203.0.113.5"
          value={ip}
          onChange={(e) => setIp(e.target.value)}
        />
        <input
          className="field-input mono flex-1 min-w-[160px]"
          placeholder="时区（可选），如 Asia/Manila"
          value={timezone}
          onChange={(e) => setTimezone(e.target.value)}
        />
        <Button variant="primary" onClick={run} disabled={test.isPending || !ip.trim()}>
          {test.isPending ? '试算中…' : '试算'}
        </Button>
      </div>
      {test.isError && <div className="mt-2 text-[12px] text-down">{(test.error as Error)?.message ?? '试算失败'}</div>}
      {result && (
        <div className="mt-3 flex items-center gap-2 flex-wrap">
          <span
            className={cn(
              'text-[12px] font-bold px-[10px] py-1 rounded-full',
              result.mode === 'B' ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
            )}
          >
            {result.mode === 'B' ? '放行 B 面' : '仍走 A 面'}
          </span>
          {result.country && <span className="text-[12px] text-ink-2 font-mono">国家 {result.country}</span>}
          <span className="text-[12px] text-muted">{result.reason}</span>
        </div>
      )}
    </div>
  );
}

// ---------- 判定流水 ----------
const DECISION_LABEL: Record<string, string> = { A: '仍走 A 面', B: '放行 B 面' };

function GateLogsPanel({ listingId }: { listingId: string }) {
  const [open, setOpen] = useState(false);
  const { data, isFetching, refetch } = useListingGateLogs(listingId, open);
  const logs: ListingGateLog[] = data ?? [];

  return (
    <div>
      <button type="button" onClick={() => setOpen((o) => !o)} className="chip">
        {open ? '收起判定流水' : '查看判定流水'}
      </button>
      {open && (
        <div className="mt-3 rounded-[10px] border border-line overflow-hidden">
          <div className="flex items-center justify-between px-3 py-2 bg-panel-2 border-b border-line">
            <span className="text-[12px] text-muted">最近 {logs.length} 条 · 排查「为什么没进 B 面」</span>
            <button className="text-[12px] text-brand hover:underline" onClick={() => void refetch()} disabled={isFetching}>
              {isFetching ? '刷新中…' : '刷新'}
            </button>
          </div>
          <div className="max-h-[280px] overflow-auto">
            <table className="w-full text-[12px]">
              <thead>
                <tr className="bg-panel-2 text-ink-2">
                  <th className="text-left font-semibold px-3 py-2">判定</th>
                  <th className="text-left font-semibold px-3 py-2">IP</th>
                  <th className="text-left font-semibold px-3 py-2">国家</th>
                  <th className="text-left font-semibold px-3 py-2">时区</th>
                  <th className="text-left font-semibold px-3 py-2">原因</th>
                  <th className="text-left font-semibold px-3 py-2">时间</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-t border-line-2">
                    <td className="px-3 py-[7px]">
                      <span
                        className={cn(
                          'text-[11px] font-bold px-2 py-0.5 rounded-full whitespace-nowrap',
                          log.decision === 'B' ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
                        )}
                      >
                        {DECISION_LABEL[log.decision] ?? log.decision}
                      </span>
                    </td>
                    <td className="px-3 py-[7px] font-mono text-ink-2">{log.ip || '—'}</td>
                    <td className="px-3 py-[7px] font-mono text-ink-2">{log.country || '未知'}</td>
                    <td className="px-3 py-[7px] font-mono text-ink-2">{log.timezone || '—'}</td>
                    <td className="px-3 py-[7px] text-ink-2">{log.reason}</td>
                    <td className="px-3 py-[7px] text-muted whitespace-nowrap">{formatTime(log.createdAt)}</td>
                  </tr>
                ))}
                {logs.length === 0 && !isFetching && (
                  <tr>
                    <td colSpan={6} className="text-center text-muted py-6">
                      暂无判定记录
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false });
  } catch {
    return iso;
  }
}
