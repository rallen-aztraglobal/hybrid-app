/**
 * 数据看板（原型视图 3，口径见 12-popup.md §5）。
 * 日期范围（今日 / 近 7 天 / 近 30 天 / 自定义，跨度 ≤ 92 天）+ 品牌 / 渠道包 / 位置 / 弹窗筛选；
 * KPI 卡、主表 11 个指标（阈值着色，不适用显示「—」）、行展开卡片级明细、
 * 拦截分布与关闭方式分布横向堆叠条、口径说明。
 */
import { Fragment, useEffect, useMemo, useState, type ReactNode } from 'react';
import { useAllPopups, useChannels, usePopupStats } from '@/hooks/queries';
import { BRAND_META, CHANNEL_BRAND_ORDER } from '@/lib/brands';
import { cn } from '@/lib/cn';
import { POPUP_POSITION_CODES } from '@/lib/popupMeta';
import {
  THRESHOLD_TIPS,
  DEFAULT_STATS_TZ,
  aggregateKpis,
  fmtInt,
  pct,
  positionLabel,
  rangeForPreset,
  rateValue,
  sumFiltered,
  thresholdLevel,
  validateRange,
  type RangePreset,
  type RateKey,
} from '@/lib/popupStats';
import type { BrandCode, PopupPositionCode, PopupStatRow } from '@/lib/types';
import { AlertIcon, CalendarIcon, ChevronDownIcon, CloseIcon, LayersIcon, ZapIcon } from '../icons';
import { DatePicker } from '../DatePicker';
import { SearchSelect } from '../SearchSelect';
import { Select } from '../ui';
import { PopupThumb, PosBadge } from './bits';

const SERIES = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100'];
const FILTER_NAMES = ['频控', '互斥', '定向', '时间'];
const CLOSE_NAMES = ['关闭按钮', '遮罩', '返回键'];

function Spark({ values, labels, unit }: { values: number[]; labels: string[]; unit: string }) {
  if (values.length < 2) return <div className="h-[30px]" />;
  const mn = Math.min(...values);
  const mx = Math.max(...values);
  const H = 30;
  const pts = values.map((v, k) => [k * (100 / (values.length - 1)), H - 4 - ((v - mn) / (mx - mn || 1)) * (H - 9)] as const);
  const d = pts.map((p) => `${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ');
  return (
    <svg viewBox="0 0 100 30" preserveAspectRatio="none" className="block w-full h-[30px] mt-[6px]">
      <polygon points={`0,30 ${d} 100,30`} fill="rgba(42,120,214,.10)" />
      <polyline points={d} fill="none" stroke="#2a78d6" strokeWidth="2" vectorEffect="non-scaling-stroke" strokeLinejoin="round" strokeLinecap="round" />
      {pts.map((p, k) => (
        <rect key={k} x={p[0] - 8} y="0" width="16" height="30" fill="transparent">
          <title>{`${labels[k]} · ${unit === '%' ? (values[k] * 100).toFixed(1) : fmtInt(values[k])}${unit}`}</title>
        </rect>
      ))}
    </svg>
  );
}

/** 横向堆叠条。 */
function StackBar({ values, names, total }: { values: number[]; names: string[]; total?: number }) {
  const tot = total ?? (values.reduce((a, b) => a + b, 0) || 1);
  return (
    <div className="flex gap-0.5 h-[14px] rounded overflow-hidden bg-panel">
      {values.map((v, k) =>
        v ? (
          <i
            key={k}
            className="block h-full min-w-[3px] transition hover:brightness-110"
            style={{ flex: v, background: SERIES[k] }}
            title={`${names[k]} · ${fmtInt(v)}（${((v / tot) * 100).toFixed(1)}%）`}
          />
        ) : null,
      )}
    </div>
  );
}

function RateCell({ row, k, digits = 1 }: { row: PopupStatRow; k: RateKey; digits?: number }) {
  const v = rateValue(row, k);
  if (v === null) return <td className="px-3 py-[11px] text-right text-muted tabular-nums">—</td>;
  const lv = thresholdLevel(k, v);
  return (
    <td className="px-3 py-[11px] text-right tabular-nums whitespace-nowrap">
      <span
        className={cn('inline-flex items-center gap-1 px-[6px] py-px rounded-[5px] font-semibold', lv === 'warn' && 'bg-[#fef3c7] text-[#92400e]', lv === 'crit' && 'bg-[#fee2e2] text-[#b91c1c]')}
        title={lv ? THRESHOLD_TIPS[k] : undefined}
      >
        {lv && <AlertIcon className="w-[11px] h-[11px]" strokeWidth={2.4} />}
        {k === 'carouselDepth' ? v.toFixed(2) : pct(v, digits)}
      </span>
    </td>
  );
}

function DetailRow({ row }: { row: PopupStatRow }) {
  const first = row.cards[0]?.impressions || 0;
  const depth = rateValue(row, 'carouselDepth');
  return (
    <tr>
      <td colSpan={13} className="bg-[#fafbff] pl-[46px] pr-3 pb-[14px]">
        <div className="bg-white border border-line rounded-xl px-[14px] py-3 mt-0.5">
          <div className="flex items-center gap-[10px] mb-2 text-[12.5px] flex-wrap">
            <b>卡片级明细</b>
            <span className="inline-flex items-center gap-[6px] px-[10px] py-[5px] text-[11.5px] rounded-[10px] text-ink-2" style={{ background: 'rgba(99,102,241,.06)', border: '1px solid rgba(99,102,241,.18)' }}>
              轮播<b className="text-ink">按卡片记曝光</b>：滑到且停留 <b className="text-ink">≥1s</b> 才计一次；频控按整个弹窗计（滑几张都算一次展示）
            </span>
          </div>
          <table className="w-full text-[12.5px] border-separate border-spacing-0">
            <thead>
              <tr>
                {['#', '素材', '卡片 ID', '曝光', '点击', 'CTR', '曝光留存（相对第 1 张）'].map((h, i) => (
                  <th key={h} className={cn('text-[11.5px] font-semibold text-muted px-3 py-2 border-b border-line', i >= 3 && i <= 5 ? 'text-right' : 'text-left')}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {row.cards.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-3 py-3 text-muted text-center">暂无卡片级数据</td>
                </tr>
              )}
              {row.cards.map((c) => {
                const keep = first > 0 ? c.impressions / first : 0;
                return (
                  <tr key={`${c.cardId}-${c.cardIndex}`}>
                    <td className="px-3 py-[7px] font-bold">{c.cardIndex + 1}</td>
                    <td className="px-3 py-[7px]">
                      <PopupThumb position={row.position} imageUrl={c.imageUrl} scale={0.7} />
                    </td>
                    <td className="px-3 py-[7px] font-mono">#{c.cardId}</td>
                    <td className="px-3 py-[7px] text-right tabular-nums">{fmtInt(c.impressions)}</td>
                    <td className="px-3 py-[7px] text-right tabular-nums">{fmtInt(c.clicks)}</td>
                    <td className="px-3 py-[7px] text-right tabular-nums">{pct(c.impressions > 0 ? c.ctr : null, 2)}</td>
                    <td className="px-3 py-[7px]">
                      <span className="inline-block w-[150px] h-2 rounded bg-[#eef2f7] align-middle mr-2 overflow-hidden">
                        <i className="block h-full rounded" style={{ width: `${(keep * 100).toFixed(1)}%`, background: SERIES[0] }} />
                      </span>
                      <span className="font-mono text-[11.5px]">{(keep * 100).toFixed(0)}%</span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {depth !== null && row.cards.length > 1 && (
            <div className="text-[11.5px] text-muted mt-[5px] leading-[1.55]">
              轮播深度 {depth.toFixed(2)}：第 2 张起曝光留存 {first > 0 ? ((row.cards[1].impressions / first) * 100).toFixed(0) : '—'}%
              {row.cards[2] ? `，第 3 张 ${first > 0 ? ((row.cards[2].impressions / first) * 100).toFixed(0) : '—'}%` : ''} → 低价值素材别放后面，或减少张数。
            </div>
          )}
        </div>
      </td>
    </tr>
  );
}

export function PopupDashboard({ popupId, onPopupChange }: { popupId: number | null; onPopupChange: (id: number | null) => void }) {
  const init = rangeForPreset('7d', new Date(), DEFAULT_STATS_TZ);
  const [preset, setPreset] = useState<RangePreset>('7d');
  const [from, setFrom] = useState(init.from);
  const [to, setTo] = useState(init.to);
  const [brand, setBrand] = useState<'' | BrandCode>('');
  const [appId, setAppId] = useState('');
  const [position, setPosition] = useState<'' | PopupPositionCode>('');
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const { data: channels } = useChannels();
  const { data: popups } = useAllPopups();

  const rangeErr = validateRange(from, to);
  const query = useMemo(
    () => ({ from, to, brand: brand || undefined, appId: appId || undefined, position: position || undefined, popupId: popupId ?? undefined }),
    [from, to, brand, appId, position, popupId],
  );
  const { data: statsRes, isLoading, isFetching } = usePopupStats(query, !rangeErr);
  const rows = statsRes?.rows;
  const tz = statsRes?.tz ?? DEFAULT_STATS_TZ;

  // 拿到服务端时区后，非自定义的预设按该时区重新归日
  useEffect(() => {
    if (preset === 'custom') return;
    const r = rangeForPreset(preset, new Date(), tz);
    setFrom(r.from);
    setTo(r.to);
  }, [tz, preset]);

  function pickPreset(p: Exclude<RangePreset, 'custom'>) {
    const r = rangeForPreset(p, new Date(), tz);
    setPreset(p);
    setFrom(r.from);
    setTo(r.to);
  }

  const list = rows ?? [];
  const kpi = aggregateKpis(list);
  const flt = sumFiltered(list);
  const fltArr = [flt.frequency, flt.mutex, flt.targeting, flt.time];
  const closeRows = list.filter((r) => r.closes > 0);

  // 日趋势（跨弹窗按日汇总），给 KPI 迷你折线用
  const series = useMemo(() => {
    const m = new Map<string, { trigger: number; displays: number; impressions: number; clicks: number }>();
    for (const r of list)
      for (const d of r.daily) {
        const a = m.get(d.date) ?? { trigger: 0, displays: 0, impressions: 0, clicks: 0 };
        a.trigger += d.trigger;
        a.displays += d.displays;
        a.impressions += d.impressions;
        a.clicks += d.clicks;
        m.set(d.date, a);
      }
    const days = [...m.entries()].sort((a, b) => a[0].localeCompare(b[0]));
    return {
      labels: days.map((d) => d[0]),
      trigger: days.map((d) => d[1].trigger),
      impressions: days.map((d) => d[1].impressions),
      clicks: days.map((d) => d[1].clicks),
      show: days.map((d) => (d[1].trigger ? d[1].displays / d[1].trigger : 0)),
      ctr: days.map((d) => (d[1].impressions ? d[1].clicks / d[1].impressions : 0)),
    };
  }, [list]);

  const kpis: { label: string; value: string; formula: string; spark?: { v: number[]; unit: string } }[] = [
    { label: '触发', value: fmtInt(kpi.trigger), formula: 'popup_trigger', spark: { v: series.trigger, unit: '' } },
    { label: '曝光', value: fmtInt(kpi.impressions), formula: 'popup_impression（卡片级）', spark: { v: series.impressions, unit: '' } },
    { label: '点击', value: fmtInt(kpi.clicks), formula: 'popup_click', spark: { v: series.clicks, unit: '' } },
    { label: '展示率', value: pct(kpi.showRate), formula: '展示次数 / 触发', spark: { v: series.show, unit: '%' } },
    { label: '点击率', value: pct(kpi.ctr, 2), formula: '点击 / 曝光', spark: { v: series.ctr, unit: '%' } },
    { label: '找回率', value: pct(kpi.recoveryRate), formula: '便条展开 / 收起为便条' },
  ];

  const popupOpts = [{ value: '', label: '全部弹窗' }, ...(popups ?? []).map((p) => ({ value: String(p.id), label: `#${p.id} ${p.name}` }))];
  const chanOpts = [
    { value: '', label: '全部渠道包' },
    ...(channels ?? [])
      .filter((c) => c.status !== 'archived' && (!brand || c.brandCode === brand))
      .map((c) => ({ value: c.applicationId, label: c.appName, sub: c.applicationId })),
  ];
  const th = 'text-[11.5px] font-semibold text-muted px-3 py-[10px] bg-panel-2 border-b border-line whitespace-nowrap';

  return (
    <div>
      <div className="flex items-center gap-3 mb-[14px] flex-wrap">
        <div className="flex gap-[10px] items-center flex-wrap">
          <div className="flex gap-[6px] items-center">
            {([['today', '今日'], ['7d', '近 7 天'], ['30d', '近 30 天'], ['custom', '自定义']] as const).map(([k, n]) => (
              <button key={k} className={cn('chip', preset === k && 'chip-on')} onClick={() => (k === 'custom' ? setPreset('custom') : pickPreset(k))}>
                {n}
              </button>
            ))}
            {preset === 'custom' ? (
              <div className="flex items-center gap-[6px]">
                <div className="w-[150px]"><DatePicker value={from} onChange={(v) => v && setFrom(v)} max={to || undefined} clearable={false} /></div>
                <span className="text-muted">~</span>
                <div className="w-[150px]"><DatePicker value={to} onChange={(v) => v && setTo(v)} min={from || undefined} clearable={false} /></div>
              </div>
            ) : (
              <span className="chip !cursor-default inline-flex items-center gap-[6px]">
                <CalendarIcon className="w-[14px] h-[14px]" />
                <span className="font-mono">{from} ~ {to}</span>
              </span>
            )}
          </div>
          <div className="flex gap-[6px] items-center pl-[10px] border-l border-line">
            <button className={cn('chip', brand === '' && 'chip-on')} onClick={() => { setBrand(''); }}>全部品牌</button>
            {CHANNEL_BRAND_ORDER.map((b) => (
              <button
                key={b}
                className={cn('chip inline-flex items-center gap-[6px]', brand === b && 'chip-on')}
                onClick={() => {
                  setBrand(b);
                  setAppId('');
                }}
              >
                <i className="w-2 h-2 rounded-full inline-block" style={{ background: BRAND_META[b].accentColor }} />
                {BRAND_META[b].name}
              </button>
            ))}
          </div>
          <div className="flex gap-2 items-center pl-[10px] border-l border-line">
            <div className="w-[220px]"><SearchSelect value={appId} onChange={setAppId} options={chanOpts} placeholder="全部渠道包" /></div>
            <div className="w-[170px]">
              <Select
                value={position}
                onChange={(v) => setPosition(v as '' | PopupPositionCode)}
                options={[{ value: '', label: '全部位置' }, ...POPUP_POSITION_CODES.map((c) => ({ value: c, label: positionLabel(c) }))]}
              />
            </div>
            <div className="w-[210px]"><SearchSelect value={popupId ? String(popupId) : ''} onChange={(v) => onPopupChange(v ? Number(v) : null)} options={popupOpts} placeholder="全部弹窗" /></div>
          </div>
        </div>
        <div className="ml-auto text-[11.5px] text-muted">埋点带 palcode / 包名 / 版本 / 时间戳；本地缓存 + 失败重试{isFetching && !isLoading ? ' · 刷新中…' : ''}</div>
      </div>

      {popupId && (
        <div className="mb-3 inline-flex items-center gap-2 text-[12.5px] bg-[#eef2ff] text-brand-ink font-semibold px-[10px] py-[5px] rounded-lg">
          仅看弹窗 #{popupId}
          <button onClick={() => onPopupChange(null)} className="grid place-items-center w-4 h-4 rounded hover:bg-[rgba(99,102,241,.15)]" title="清除">
            <CloseIcon className="w-[11px] h-[11px]" />
          </button>
        </div>
      )}

      {rangeErr && <div className="mb-3 rounded-[10px] border border-[#fecaca] bg-[#fef2f2] text-[#b91c1c] text-[12.5px] px-3 py-2 font-semibold">{rangeErr}</div>}

      <div className="grid gap-3 mb-4 grid-cols-3 min-[1380px]:grid-cols-6">
        {kpis.map((k) => (
          <div key={k.label} className="rounded-card border border-line bg-panel shadow-sm2 px-[14px] pt-[14px] pb-2">
            <div className="text-[12px] text-ink-2 font-semibold">{k.label}</div>
            <div className="text-[23px] font-extrabold tracking-[-.3px] mt-[3px] tabular-nums text-ink">{k.value}</div>
            <div className="text-[10.5px] text-muted font-mono">{k.formula}</div>
            {k.spark ? <Spark values={k.spark.v} labels={series.labels} unit={k.spark.unit} /> : <div className="h-[30px]" />}
          </div>
        ))}
      </div>

      <div className="rounded-card border border-line bg-panel shadow-sm2 overflow-auto">
        <table className="w-full border-separate border-spacing-0 text-[12.5px]">
          <thead>
            <tr>
              <th className={th} />
              <th className={cn(th, 'text-left')}>弹窗</th>
              {[
                ['触发', 'popup_trigger'],
                ['曝光', 'popup_impression（卡片级）'],
                ['展示率', '展示次数 / 触发'],
                ['点击', ''],
                ['点击率', '点击 / 曝光'],
                ['关闭率', '关闭 / 首张曝光(展示次数)'],
                ['加载失败率', 'load_fail / 触发'],
                ['轮播深度', '曝光 / 首张曝光，平均看到几张'],
                ['找回率', '便条展开 / 收起为便条'],
                ['便条弃用率', '长按关闭 / 便条曝光'],
                ['总点击率', '(弹窗点击 + 便条点击) / 曝光'],
              ].map(([h, tip]) => (
                <th key={h} className={cn(th, 'text-right')} title={tip || undefined}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={13} className="text-center text-muted py-[50px]">加载中…</td></tr>
            ) : list.length === 0 ? (
              <tr><td colSpan={13} className="text-center text-muted py-[50px]">该筛选下暂无数据</td></tr>
            ) : (
              list.map((r) => {
                const open = expanded.has(r.popupId);
                return (
                  <Fragment key={r.popupId}>
                    <tr className="hover:bg-[#fafbff] [&>td]:border-b [&>td]:border-line-2">
                      <td className="px-3 py-[11px]">
                        <button
                          title="卡片级明细"
                          onClick={() => setExpanded((s) => { const n = new Set(s); if (n.has(r.popupId)) n.delete(r.popupId); else n.add(r.popupId); return n; })}
                          className={cn('grid place-items-center w-[22px] h-[22px] rounded-md text-muted hover:bg-bg hover:text-ink transition', open && 'rotate-180 !text-brand')}
                        >
                          <ChevronDownIcon className="w-[15px] h-[15px]" strokeWidth={2.2} />
                        </button>
                      </td>
                      <td className="px-3 py-[11px]">
                        <div className="flex items-center gap-[10px] min-w-[210px]">
                          <PopupThumb position={r.position} imageUrl={r.cards[0]?.imageUrl} count={r.cards.length} scale={0.82} />
                          <div>
                            <div className="font-bold text-[13px] text-ink">
                              {r.name}
                              {r.deleted && <span className="ml-1 text-[10.5px] font-bold text-muted bg-[#f1f5f9] px-[6px] py-px rounded-[5px]">已删除</span>}
                            </div>
                            <div className="mt-0.5"><PosBadge code={r.position} /></div>
                          </div>
                        </div>
                      </td>
                      <td className="px-3 py-[11px] text-right tabular-nums">{fmtInt(r.trigger)}</td>
                      <td className="px-3 py-[11px] text-right tabular-nums">{fmtInt(r.impressions)}</td>
                      <RateCell row={r} k="showRate" />
                      <td className="px-3 py-[11px] text-right tabular-nums">{fmtInt(r.clicks)}</td>
                      <RateCell row={r} k="ctr" digits={2} />
                      <RateCell row={r} k="closeRate" />
                      <RateCell row={r} k="loadFailRate" />
                      <RateCell row={r} k="carouselDepth" />
                      <RateCell row={r} k="recoveryRate" />
                      <RateCell row={r} k="tabAbandonRate" />
                      <RateCell row={r} k="totalCtr" />
                    </tr>
                    {open && <DetailRow row={r} />}
                  </Fragment>
                );
              })
            )}
          </tbody>
        </table>
        <div className="flex gap-[14px] items-center text-[11.5px] text-muted px-[14px] py-[10px] border-t border-line-2 flex-wrap">
          <span className="inline-flex items-center gap-1 px-[6px] py-px rounded-[5px] font-semibold bg-[#fef3c7] text-[#92400e]"><AlertIcon className="w-[11px] h-[11px]" />黄</span>
          需关注：展示率 &lt;60% · 点击率 &lt;3% · 关闭率 &gt;90% · 便条弃用率 &gt;30%
          <span className="inline-flex items-center gap-1 px-[6px] py-px rounded-[5px] font-semibold bg-[#fee2e2] text-[#b91c1c]"><AlertIcon className="w-[11px] h-[11px]" />红</span>
          异常：加载失败率 &gt;5%
          <span className="ml-auto">点击行首箭头展开卡片级明细 · 「—」= 该位置不适用或分母为 0</span>
        </div>
      </div>

      <div className="mt-4 flex gap-[10px] p-[12px_13px] rounded-[10px] text-[12px] leading-[1.6] text-ink-2" style={{ background: 'rgba(99,102,241,.06)', border: '1px solid rgba(99,102,241,.18)' }}>
        <ZapIcon className="w-[17px] h-[17px] text-brand flex-none mt-px" />
        <div>
          <b className="text-ink">展示率低 → 工程问题</b>（预加载失败 / 频控过紧）；<b className="text-ink">点击率低 → 素材问题</b>。两者分开看，别反复换图却没用。先压缩图片 / 查 CDN，而不是急着换素材。
        </div>
      </div>

      <div className="grid gap-4 mt-4 grid-cols-1 min-[1100px]:grid-cols-2">
        <div className="rounded-card border border-line bg-panel shadow-sm2 px-[18px] py-4">
          <h3 className="text-[13.5px] font-bold flex items-center gap-2"><LayersIcon className="w-[15px] h-[15px]" />拦截分布 <span className="font-normal text-muted text-[11.5px]">popup_filtered 按原因</span></h3>
          <div className="text-[11.5px] text-muted mt-[3px] mb-3">看频控是不是卡太死。频控写死在客户端，调整需发版，建议先按默认策略跑两周再固化。</div>
          <Legend names={FILTER_NAMES} values={fltArr} />
          <Srow name="全部" bold total={fltArr.reduce((a, b) => a + b, 0)}><StackBar values={fltArr} names={FILTER_NAMES} /></Srow>
          {list.map((r) => {
            const v = [r.filtered.frequency, r.filtered.mutex, r.filtered.targeting, r.filtered.time];
            return (
              <Srow key={r.popupId} name={`${r.position} · ${r.name}`} total={v.reduce((a, b) => a + b, 0)}>
                <StackBar values={v} names={FILTER_NAMES} />
              </Srow>
            );
          })}
        </div>
        <div className="rounded-card border border-line bg-panel shadow-sm2 px-[18px] py-4">
          <h3 className="text-[13.5px] font-bold flex items-center gap-2"><CloseIcon className="w-[15px] h-[15px]" />关闭方式分布 <span className="font-normal text-muted text-[11.5px]">popup_close.method</span></h3>
          <div className="text-[11.5px] text-muted mt-[3px] mb-3">返回键占比高通常意味着倒计时期间用户想走（倒计时期间返回键全程可关）。便条长按关闭不计入此处。</div>
          <Legend names={CLOSE_NAMES} />
          {closeRows.length === 0 ? (
            <div className="text-center text-muted py-8">暂无</div>
          ) : (
            closeRows.map((r) => {
              const v = [r.closeByMethod.button, r.closeByMethod.mask, r.closeByMethod.back];
              return (
                <Srow key={r.popupId} name={`${r.position} · ${r.name}`} total={r.closes}>
                  <StackBar values={v} names={CLOSE_NAMES} />
                </Srow>
              );
            })
          )}
        </div>
      </div>

      <details className="mt-4 rounded-card border border-line bg-panel shadow-sm2 px-4 py-3">
        <summary className="cursor-pointer font-bold text-[13px]">指标口径（12-popup.md §5）</summary>
        <table className="w-full text-[12.5px] mt-[10px] border-separate border-spacing-0">
          <thead>
            <tr>{['指标', '公式', '看什么'].map((h) => <th key={h} className="text-left text-[11.5px] font-semibold text-muted px-3 py-[10px] bg-panel-2 border-b border-line">{h}</th>)}</tr>
          </thead>
          <tbody>
            {[
              ['展示次数', 'Σ impression（card_index=0 ∧ 自动弹出）', '自动弹出且真实渲染的次数'],
              ['展示率', '展示次数 / 触发', '低 → 预加载失败或频控设太紧（工程问题）'],
              ['点击率', '点击 / 曝光（卡片级）', '低 → 素材或文案问题'],
              ['关闭率', '关闭 / 首张曝光(含便条重开)', '高 → 内容与用户预期不符'],
              ['加载失败率', 'load_fail / 触发', '高 → 图片太大或 CDN 有问题'],
              ['轮播深度', '曝光 / 首张曝光', '低 → 第 2 张之后基本没人看，别配太多'],
              ['找回率', '便条展开 / 收起为便条', '高 → 内容有价值但弹出时机不对，该调触发而非换图'],
              ['便条弃用率', '长按关闭 / 便条曝光', '高 → 便条本身构成干扰，该缩小或换位置'],
              ['总点击率', '(弹窗点击 + 便条点击) / 曝光', '便条带来的增量，判断这个机制值不值得留'],
              ['拦截分布', 'filtered 按原因分组（频控 / 互斥 / 定向 / 时间）', '看频控是不是卡太死'],
            ].map((r) => (
              <tr key={r[0]} className="[&>td]:border-b [&>td]:border-line-2">
                <td className="px-3 py-[9px] font-bold">{r[0]}</td>
                <td className="px-3 py-[9px] font-mono text-[12px]">{r[1]}</td>
                <td className="px-3 py-[9px]">{r[2]}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="text-[11.5px] text-muted mt-2">日期按服务端统计时区（{tz}）归日；分母为 0 时服务端返回 0，看板显示「—」。</div>
      </details>
    </div>
  );
}

function Legend({ names, values }: { names: string[]; values?: number[] }) {
  return (
    <div className="flex gap-[14px] flex-wrap text-[12px] text-ink-2 mb-3">
      {names.map((n, k) => (
        <span key={n} className="inline-flex items-center gap-[6px]">
          <i className="w-[10px] h-[10px] rounded-[3px] inline-block" style={{ background: SERIES[k] }} />
          {n}
          {values && <b className="tabular-nums">{fmtInt(values[k])}</b>}
        </span>
      ))}
    </div>
  );
}

function Srow({ name, total, bold, children }: { name: string; total: number; bold?: boolean; children: ReactNode }) {
  return (
    <div className="grid items-center gap-[10px] mb-[9px] text-[12px]" style={{ gridTemplateColumns: '150px 1fr 70px' }}>
      <span className={cn('truncate font-semibold', bold ? 'text-ink font-extrabold' : 'text-ink-2')} title={name}>{name}</span>
      {children}
      <span className="text-right text-muted tabular-nums">{fmtInt(total)}</span>
    </div>
  );
}
