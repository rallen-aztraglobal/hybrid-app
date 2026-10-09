/**
 * 新建 / 编辑弹窗抽屉（原型视图 2）：左表单 + 右手机实时预览。
 * 分区：1 基本与投放 / 2 素材 / 3 定向 / 4 行为 / 频控（只读）/ 生效机制提示。
 * 字段按位置能力矩阵（lib/popupMeta）动态显隐与默认值；校验见 lib/popupForm.validatePopupForm，
 * 与后端规则一致（后端 400 的 message 直接 toast）。无 popup:edit 时整张表单只读。
 */
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useChannels, useBrands, usePopupDetail, usePopupPositions, useSavePopup } from '@/hooks/queries';
import { BRAND_META, CHANNEL_BRAND_ORDER } from '@/lib/brands';
import { cn } from '@/lib/cn';
import {
  blankForm,
  formFromPopup,
  switchPosition,
  toPopupInput,
  validatePopupForm,
  type PopupForm,
  type ScopeFlags,
} from '@/lib/popupForm';
import {
  COMMON_COUNTRIES,
  IMAGE_SPECS,
  POPUP_POSITION_CODES,
  POSITION_CAPS,
  TAB_TEXT_HARD_MAX,
  TAB_TEXT_SOFT_MAX,
  behaviorDefaults,
  charLen,
  isForcedMode,
  isValidVersion,
  normalizeCountry,
} from '@/lib/popupMeta';
import { toast } from '@/store/toastStore';
import { fmtLocalTime } from '@/lib/popupView';
import type { BrandCode, PopupOpenMode, PopupPositionCode, PopupUserType } from '@/lib/types';
import { AlertIcon, CheckIcon, ClockIcon, CloseIcon, LockIcon, SaveCheckIcon, TimerIcon } from '../icons';
import { DateTimePicker } from '../DateTimePicker';
import { MultiSearchSelect } from '../MultiSearchSelect';
import { Button, Select, Switch } from '../ui';
import { Seg } from './bits';
import { CountInput, ImageUpload, PopupMaterial, UploadInfo } from './PopupMaterial';
import { PopupPreview } from './PopupPreview';
import { PopupEffectNote, PopupFrequency } from './PopupReadonly';

function Section({ num, title, hint, children }: { num: number; title: string; hint?: string; children: ReactNode }) {
  return (
    <div className="section-card">
      <h3 className="flex items-center gap-2 flex-wrap text-[13px] font-bold text-ink mb-[14px]">
        <span className="grid place-items-center w-5 h-5 rounded-md bg-brand text-white text-[11px] font-extrabold">{num}</span>
        {title}
        {hint && <span className="font-normal text-muted text-[11.5px]">{hint}</span>}
      </h3>
      {children}
    </div>
  );
}

function F({ label, req, hint, error, children }: { label: string; req?: boolean; hint?: string; error?: string; children: ReactNode }) {
  return (
    <div className="mb-[13px] last:mb-0">
      <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
        {label} {req && <span className="text-down">*</span>}{' '}
        {hint && <span className="font-normal text-muted text-[11.5px]">{hint}</span>}
      </label>
      {children}
      {error && <div className="mt-1 text-[12px] text-down font-semibold">{error}</div>}
    </div>
  );
}

function SwitchRow({ checked, onChange, disabled, title, sub }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean; title: string; sub: string }) {
  return (
    <div className="flex items-center gap-[10px] px-[13px] py-[11px] bg-panel-2 border border-line rounded-[10px]">
      <Switch checked={checked} onChange={onChange} disabled={disabled} />
      <div>
        <div className="text-[13px] font-semibold">{title}</div>
        <div className="text-[11.5px] text-muted">{sub}</div>
      </div>
    </div>
  );
}

function BehRow({ title, tag, sub, children, dis }: { title: ReactNode; tag?: ReactNode; sub: ReactNode; children: ReactNode; dis?: boolean }) {
  return (
    <div className="flex items-start gap-[14px] py-3 border-t border-line-2 first:border-t-0 first:pt-[2px]">
      <div className={cn(dis && 'opacity-50')}>
        <div className="text-[13px] font-semibold flex items-center gap-2 flex-wrap">
          {title}
          {tag}
        </div>
        <div className="text-[11.5px] text-muted mt-0.5 leading-[1.55]">{sub}</div>
      </div>
      <div className="ml-auto flex-none pt-0.5">{children}</div>
    </div>
  );
}

const Fixed = ({ children }: { children: ReactNode }) => (
  <span className="inline-flex items-center gap-1 text-[11px] font-medium text-[#64748b] bg-[#f1f5f9] px-[7px] py-px rounded-[5px]">
    <LockIcon className="w-[11px] h-[11px]" />
    {children}
  </span>
);
const DisTag = () => <span className="text-[10.5px] font-bold text-[#b91c1c] bg-[#fee2e2] px-[6px] py-px rounded-[5px]">强制模式下禁用</span>;

function CountryInput({ value, onChange }: { value: string[]; onChange: (v: string[]) => void }) {
  const [text, setText] = useState('');
  const [bad, setBad] = useState(false);
  function add(raw: string) {
    const parts = raw.split(/[\s,，;；]+/).filter(Boolean);
    if (!parts.length) return;
    const next = [...value];
    let invalid = false;
    for (const p of parts) {
      const c = normalizeCountry(p);
      if (!c) invalid = true;
      else if (!next.includes(c)) next.push(c);
    }
    setBad(invalid);
    if (!invalid) setText('');
    onChange(next);
  }
  return (
    <>
      <div className="flex gap-[6px] flex-wrap items-center p-[6px] border border-line rounded-[10px] bg-panel-2">
        {value.map((c) => (
          <span key={c} className="inline-flex items-center gap-[5px] font-mono text-[12px] font-bold bg-[#eef2ff] text-brand-ink pl-[9px] pr-[5px] py-[3px] rounded-[7px]">
            {c}
            <button type="button" className="grid place-items-center w-4 h-4 rounded hover:bg-[rgba(99,102,241,.15)]" onClick={() => onChange(value.filter((x) => x !== c))}>
              <CloseIcon className="w-[10px] h-[10px]" />
            </button>
          </span>
        ))}
        <input
          value={text}
          onChange={(e) => {
            setText(e.target.value.toUpperCase());
            setBad(false);
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ',') {
              e.preventDefault();
              add(text);
            } else if (e.key === 'Backspace' && !text && value.length) onChange(value.slice(0, -1));
          }}
          onBlur={() => text && add(text)}
          placeholder="输入如 PH，回车添加"
          className="border-none bg-transparent outline-none text-[12.5px] p-1 min-w-[130px] flex-1 font-mono"
        />
      </div>
      {bad && <div className="mt-1 text-[12px] text-down font-semibold">国家码需为 ISO-3166 二位字母（如 PH）</div>}
      <div className="flex gap-[6px] flex-wrap items-center mt-[6px]">
        <span className="text-[11.5px] text-muted">快捷：</span>
        {COMMON_COUNTRIES.map((c) => {
          const on = value.includes(c);
          return (
            <button
              key={c}
              type="button"
              onClick={() => onChange(on ? value.filter((x) => x !== c) : [...value, c])}
              className={cn('font-mono text-[11.5px] font-bold px-[9px] py-[2px] rounded-lg border transition', on ? 'bg-ink text-white border-ink' : 'bg-panel text-ink-2 border-line hover:bg-bg')}
            >
              {c}
            </button>
          );
        })}
      </div>
      <div className="text-[11.5px] text-muted mt-[5px] leading-[1.55]">
        按请求 IP 判定国家；<b className="text-ink-2">判不出则不命中</b>（不兜底放行）。留空 = 全部地区。
      </div>
    </>
  );
}

export function PopupDrawer({
  target,
  canEdit,
  scope,
  onClose,
}: {
  target: 'new' | number | null;
  canEdit: boolean;
  scope: ScopeFlags;
  onClose: () => void;
}) {
  const open = target !== null;
  const editingId = typeof target === 'number' ? target : null;
  const { data: detail, isError } = usePopupDetail(editingId);
  const { data: positions } = usePopupPositions();
  const { data: brandList } = useBrands();
  const { data: channels } = useChannels();
  const save = useSavePopup();

  const [form, setForm] = useState<PopupForm | null>(null);
  const [submitted, setSubmitted] = useState(false);
  const initFor = useRef<string | number | null>(null);

  // 打开 / 切换目标时初始化表单（编辑态等详情就绪后只初始化一次，避免回刷覆盖用户编辑）
  useEffect(() => {
    if (target === null) {
      initFor.current = null;
      setForm(null);
      return;
    }
    setSubmitted(false);
    if (target === 'new') {
      initFor.current = 'new';
      setForm(blankForm());
    } else {
      initFor.current = null;
      setForm(null);
    }
  }, [target]);
  useEffect(() => {
    if (editingId !== null && detail && detail.id === editingId && initFor.current !== editingId) {
      initFor.current = editingId;
      setForm(formFromPopup(detail));
    }
  }, [detail, editingId]);

  const result = useMemo(() => (form ? validatePopupForm(form, scope) : { errors: [], warnings: [] }), [form, scope]);

  const brandCodes: BrandCode[] = useMemo(() => {
    const list = (brandList ?? []).filter((b) => b.supportsChannels !== false).map((b) => b.code);
    return list.length ? list : CHANNEL_BRAND_ORDER;
  }, [brandList]);

  const channelBrand = useMemo(() => new Map((channels ?? []).map((c) => [c.applicationId, c.brandCode])), [channels]);
  const chanOptions = useMemo(() => {
    const brands = form?.brandCodes.length ? form.brandCodes : brandCodes;
    return (channels ?? [])
      .filter((c) => c.status !== 'archived' && brands.includes(c.brandCode))
      .map((c) => ({ value: c.applicationId, label: c.appName, sub: c.applicationId }));
  }, [channels, form?.brandCodes, brandCodes]);

  if (!open) {
    return <DrawerShell open={false} title="" sub="" onClose={onClose} body={null} foot={null} />;
  }

  const set = (patch: Partial<PopupForm>) => setForm((f) => (f ? { ...f, ...patch } : f));

  async function doSave() {
    if (!form) return;
    setSubmitted(true);
    const r = validatePopupForm(form, scope);
    if (r.errors.length) {
      toast.err(`无法保存：${r.errors.join('；')}`);
      return;
    }
    try {
      await save.mutateAsync({ id: form.id, input: toPopupInput(form) });
      toast.ok('已保存。客户端按拉取时机生效：冷启动必拉，回前台 >5 分钟重拉；最坏 = 用户下次回前台');
      onClose();
    } catch (err) {
      toast.err(err instanceof Error ? err.message : '保存失败');
    }
  }

  const pos = form?.position ?? 'P1';
  const caps = POSITION_CAPS[pos];
  const forced = !!form && isForcedMode(pos, form.closable);
  const posOff = positions ? positions.find((p) => p.code === pos)?.enabled === false : false;

  const body = !form ? (
    <div className="text-center text-muted py-20">{isError ? '加载弹窗详情失败' : '加载中…'}</div>
  ) : (
    <div className="grid gap-[22px] items-start" style={{ gridTemplateColumns: 'minmax(0,1fr) 330px' }}>
      <fieldset disabled={!canEdit || save.isPending} className="min-w-0 border-0 p-0 m-0 flex flex-col gap-4">
        {!canEdit && (
          <div className="flex items-center gap-2 text-[12.5px] font-semibold text-[#b91c1c] bg-[#fee2e2] rounded-[10px] px-3 py-2">
            <LockIcon className="w-[14px] h-[14px]" />
            当前账号无 popup:edit 权限，仅可查看
          </div>
        )}
        {submitted && result.errors.length > 0 && (
          <div className="rounded-[10px] border border-[#fecaca] bg-[#fef2f2] text-[#b91c1c] text-[12.5px] leading-[1.7] px-[13px] py-[10px]">
            <b>无法保存，请修正：</b>
            <ul className="list-disc ml-4">
              {result.errors.map((e) => (
                <li key={e}>{e}</li>
              ))}
            </ul>
          </div>
        )}
        {result.warnings.length > 0 && (
          <div className="rounded-[10px] border border-[#fde68a] bg-[#fffbeb] text-[12px] leading-[1.7] px-[13px] py-[10px]" style={{ color: '#92400e' }}>
            <div className="flex items-center gap-[6px] font-bold">
              <AlertIcon className="w-[14px] h-[14px]" />
              提示（不拦截保存）
            </div>
            <ul className="list-disc ml-4">
              {result.warnings.map((e) => (
                <li key={e}>{e}</li>
              ))}
            </ul>
          </div>
        )}

        {/* 1 基本与投放 */}
        <Section num={1} title="基本与投放">
          <F label="弹窗名称" req hint="仅后台可见，用于列表与数据看板">
            <CountInput value={form.name} onChange={(v) => set({ name: v })} hard={64} placeholder="例如 中秋活动 · 月饼礼金" />
          </F>
          <div className="grid grid-cols-2 gap-3">
            <F label="弹窗位置" req hint="切换后下方字段与右侧预览随之变化">
              <Select
                value={form.position}
                onChange={(v) => setForm((f) => (f ? switchPosition(f, v as PopupPositionCode) : f))}
                options={POPUP_POSITION_CODES.map((c) => ({
                  value: c,
                  label: `${c} · ${POSITION_CAPS[c].name}${positions && positions.find((p) => p.code === c)?.enabled === false ? '（位置未开启）' : ''}`,
                }))}
              />
              {posOff && (
                <div className="flex gap-[5px] items-start text-[11.5px] leading-[1.55] mt-[5px]" style={{ color: '#b45309' }}>
                  <AlertIcon className="w-[13px] h-[13px] flex-none mt-[2px]" />
                  <span>{pos} 位置当前全局关闭，保存后不会下发；需在「投放列表 › 位置开关」开启</span>
                </div>
              )}
            </F>
            <F label="优先级" hint="0–999，数字越大越优先">
              <input type="number" min={0} max={999} className="field-input" value={form.priority} onChange={(e) => set({ priority: e.target.value })} />
              <div className="text-[11.5px] text-muted mt-[5px] leading-[1.55]">多弹窗同时命中只展示最高的一个，其余记为「互斥拦截」（同一时刻只允许一个遮罩类弹窗）</div>
            </F>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <F label="生效开始" hint="留空 = 不限">
              <div className="flex gap-2 items-center">
                <div className="flex-1 min-w-0">
                  <DateTimePicker value={form.startAt} onChange={(v) => set({ startAt: v })} placeholder="不限（立即生效）" presets={[]} minuteStep={1} />
                </div>
                {form.startAt && <ClearBtn onClick={() => set({ startAt: '' })} />}
              </div>
            </F>
            <F label="生效结束" hint="留空 = 不限">
              <div className="flex gap-2 items-center">
                <div className="flex-1 min-w-0">
                  <DateTimePicker
                    value={form.endAt}
                    onChange={(v) => set({ endAt: v })}
                    placeholder="不限（长期有效）"
                    presets={[]}
                    minuteStep={1}
                    invalid={!!form.startAt && !!form.endAt && form.endAt <= form.startAt}
                  />
                </div>
                {form.endAt && <ClearBtn onClick={() => set({ endAt: '' })} />}
              </div>
            </F>
          </div>
          <SwitchRow
            checked={form.enabled}
            onChange={(v) => set({ enabled: v })}
            title="启用"
            sub="关闭即停投，弹窗与其便条同时停止出现；按拉取时机生效（最慢约 5 分钟 + 用户切换动作）"
          />
        </Section>

        {/* 2 素材 */}
        <PopupMaterial form={form} setForm={(u) => setForm((f) => (f ? u(f) : f))} canEdit={canEdit} />

        {/* 3 定向 */}
        <Section num={3} title="定向" hint="· 全部条件同时满足才命中">
          <F label="品牌" hint={`多选；不选 = 全部品牌${scope.allBrands ? '' : '（当前账号数据范围受限，必须选择）'}`}>
            <div className="flex gap-[6px] flex-wrap">
              {brandCodes.map((b) => {
                const on = form.brandCodes.includes(b);
                const color = BRAND_META[b]?.accentColor ?? '#6366f1';
                return (
                  <button
                    key={b}
                    type="button"
                    onClick={() => {
                      const next = on ? form.brandCodes.filter((x) => x !== b) : [...form.brandCodes, b];
                      // 品牌收窄时，同步剔除已不属于所选品牌的渠道包
                      const appIds = next.length ? form.appIds.filter((a) => !channelBrand.has(a) || next.includes(channelBrand.get(a)!)) : form.appIds;
                      set({ brandCodes: next, appIds });
                    }}
                    className="inline-flex items-center gap-[6px] px-[11px] py-[6px] rounded-full border text-[12.5px] font-semibold transition disabled:opacity-60"
                    style={on ? { borderColor: color, background: `color-mix(in srgb, ${color} 9%, #fff)`, color } : { borderColor: 'var(--line)', background: '#fff', color: 'var(--ink-2)' }}
                  >
                    <span className="grid place-items-center w-[14px] h-[14px] rounded-[4px] border-[1.5px]" style={on ? { background: color, borderColor: color, color: '#fff' } : { borderColor: '#cbd5e1' }}>
                      {on && <CheckIcon className="w-[10px] h-[10px]" strokeWidth={3.2} />}
                    </span>
                    {BRAND_META[b]?.name ?? b}
                    <span className="font-mono text-[11px] opacity-70">{b}</span>
                  </button>
                );
              })}
            </div>
          </F>
          <F label="渠道包" hint={`按 applicationId 匹配，多选；不选 = 全部${scope.allChannels ? '' : '（当前账号渠道范围受限，必须选择）'}`}>
            <MultiSearchSelect
              values={form.appIds}
              onChange={(v) => set({ appIds: v })}
              options={chanOptions}
              placeholder="全部渠道包"
              searchPlaceholder="搜索渠道名 / applicationId"
            />
            <div className="text-[11.5px] text-muted mt-[5px]">
              已选 <b className="text-ink-2">{form.appIds.length}</b> 个包；候选来自{form.brandCodes.length ? '所选品牌' : '全部品牌'}下的渠道（共 {chanOptions.length} 个）
            </div>
          </F>
          <div className="grid grid-cols-2 gap-3">
            <F label="App 版本 ≥" hint="X.Y.Z，留空不限" error={isValidVersion(form.minVersion) ? undefined : '格式需为 X.Y.Z（如 1.0.3，各段 0–99）'}>
              <input className="field-input font-mono text-[12.5px]" value={form.minVersion} placeholder="1.0.3" onChange={(e) => set({ minVersion: e.target.value })} />
            </F>
            <F label="App 版本 <" hint="X.Y.Z，留空不限" error={isValidVersion(form.maxVersion) ? undefined : '格式需为 X.Y.Z（如 2.0.0，各段 0–99）'}>
              <input className="field-input font-mono text-[12.5px]" value={form.maxVersion} placeholder="2.0.0" onChange={(e) => set({ maxVersion: e.target.value })} />
            </F>
          </div>
          <F label="新老用户">
            <Seg<PopupUserType>
              value={form.userType}
              onChange={(v) => set({ userType: v })}
              options={[
                { value: 'all', label: '全部' },
                { value: 'new', label: '新用户' },
                { value: 'old', label: '老用户' },
              ]}
            />
            <div className="text-[11.5px] text-muted mt-[5px] leading-[1.55]">
              <b className="text-ink-2">新用户 = 本机首次启动的那次会话</b>（客户端按本地首次启动记录判定）
            </div>
          </F>
          <F label="地区" hint="ISO 国家码多选，可空 = 全部">
            <CountryInput value={form.countries} onChange={(v) => set({ countries: v })} />
          </F>
        </Section>

        {/* 4 行为 */}
        <Section num={4} title="行为" hint={`· 只显示 ${pos} 适用的配置项`}>
          {pos === 'P4' && (
            <>
              <BehRow title="是否可关闭" sub="关 = 强制模式：无关闭按钮、遮罩点击无效、返回键不关弹窗；每次冷启动都弹">
                <Switch
                  checked={form.closable}
                  onChange={(v) => {
                    if (v) {
                      const d = behaviorDefaults('P4');
                      set({ closable: true, tabEnabled: d.tabEnabled, maskClosable: d.maskClosable });
                    } else set({ closable: false, tabEnabled: false, maskClosable: false, countdown: false });
                  }}
                />
              </BehRow>
              {forced && (
                <div className="flex gap-[10px] rounded-[10px] border border-[#fecaca] bg-[#fef2f2] text-[#b91c1c] text-[12.5px] leading-[1.6] px-[13px] py-[11px] mt-0.5 mb-[10px]">
                  <AlertIcon className="w-4 h-4 flex-none mt-0.5" />
                  <div>
                    <b>强制模式：仅用于「不换包就不能用」的情况，日常活动严禁使用。</b>
                    <br />
                    已自动禁用：收起为便条 / 遮罩点击关闭 / 倒计时；返回键也不关弹窗（全模块唯一例外），连按两次返回仍可退出 App。
                  </div>
                </div>
              )}
            </>
          )}

          {caps.tab === 'optional' && (
            <>
              <BehRow
                title="关闭后收起为便条"
                tag={forced ? <DisTag /> : undefined}
                dis={forced}
                sub="开 = 关闭后贴右侧边缘收起为便条，单点重新展开、长按当日不再出现；关 = 关闭即消失"
              >
                <Switch checked={form.tabEnabled && !forced} disabled={forced} onChange={(v) => set({ tabEnabled: v })} />
              </BehRow>
              {form.tabEnabled && !forced && (
                <div className="grid gap-3 items-start bg-panel-2 border border-line rounded-[10px] p-3 -mt-0.5 mb-[10px]" style={{ gridTemplateColumns: 'auto 1fr 1fr' }}>
                  <div>
                    <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">便条图标</label>
                    <ImageUpload
                      src={form.tabIconUrl || ''}
                      spec={IMAGE_SPECS.icon}
                      canEdit={canEdit}
                      width={64}
                      onUploaded={({ url, meta }) => set({ tabIconUrl: url, tabIconMeta: meta })}
                    />
                    {form.tabIconUrl && canEdit && (
                      <button type="button" className="text-[11.5px] text-muted hover:text-down mt-1" onClick={() => set({ tabIconUrl: '', tabIconMeta: undefined })}>
                        移除
                      </button>
                    )}
                  </div>
                  <div className="min-w-0">
                    <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">图标规格</label>
                    <UploadInfo spec={IMAGE_SPECS.icon} url={form.tabIconUrl} meta={form.tabIconMeta} />
                    <div className="text-[11px] text-muted">不传默认取第 1 张素材</div>
                  </div>
                  <div>
                    <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                      便条文案 <span className="text-down">*</span> <span className="font-normal text-muted text-[11.5px]">建议 ≤ {TAB_TEXT_SOFT_MAX} 字</span>
                    </label>
                    <CountInput value={form.tabText} onChange={(v) => set({ tabText: v })} soft={TAB_TEXT_SOFT_MAX} hard={TAB_TEXT_HARD_MAX} placeholder="中秋礼金" />
                    {charLen(form.tabText) > TAB_TEXT_HARD_MAX ? (
                      <div className="mt-1 text-[12px] text-down">超过 {TAB_TEXT_HARD_MAX} 字，无法保存</div>
                    ) : charLen(form.tabText) > TAB_TEXT_SOFT_MAX ? (
                      <div className="mt-1 text-[11.5px]" style={{ color: '#b45309' }}>超过建议的 {TAB_TEXT_SOFT_MAX} 字，过长会被截断</div>
                    ) : null}
                  </div>
                </div>
              )}
            </>
          )}

          {caps.countdown === 'optional' && (
            <BehRow
              title={
                <>
                  <TimerIcon className="w-[14px] h-[14px]" />
                  倒计时关闭
                </>
              }
              tag={<Fixed>时长固定 3 秒，不可调</Fixed>}
              sub={
                <>
                  {pos === 'P1' ? 'P1 默认关，仅大型活动期间临时开启' : 'P7 默认开'}
                  <div className="flex flex-wrap gap-1 mt-[6px]">
                    {['0 秒起在 X 位置显示数字', '返回键全程可关', '结束后 X 放大 · 热区 ≥44×44dp', '倒计时中遮罩点击无效', '切后台暂停 · 回前台续计'].map((t) => (
                      <span key={t} className="text-[10.5px] bg-[#f1f5f9] text-[#475569] px-[7px] py-px rounded-[5px]">
                        {t}
                      </span>
                    ))}
                  </div>
                </>
              }
            >
              <Switch checked={form.countdown} onChange={(v) => set({ countdown: v })} />
            </BehRow>
          )}

          {caps.maskClosable === 'optional' && (
            <BehRow
              title="遮罩点击关闭"
              tag={forced ? <DisTag /> : undefined}
              dis={forced}
              sub={caps.countdown === 'optional' ? '开启倒计时时：倒计时期间无效，结束后按此配置生效' : '点击卡片以外的遮罩区域关闭'}
            >
              <Switch checked={form.maskClosable && !forced} disabled={forced} onChange={(v) => set({ maskClosable: v })} />
            </BehRow>
          )}

          <BehRow title="跳转方式" sub="不得自动跳转外部应用或商店，必须用户主动点击">
            <Seg<PopupOpenMode>
              value={form.openMode}
              onChange={(v) => set({ openMode: v })}
              options={[
                { value: 'webview', label: 'WebView 内' },
                { value: 'browser', label: '外部浏览器' },
                { value: 'store', label: '应用商店' },
              ]}
            />
          </BehRow>

          {pos === 'P3' && (
            <BehRow title="自动轮播间隔" sub="多张时自动切换（2–30 秒，默认 5 秒）">
              <div className="flex items-center gap-2">
                <input type="number" min={2} max={30} className="field-input !w-[84px] text-center font-mono" value={form.autoplaySeconds} onChange={(e) => set({ autoplaySeconds: e.target.value })} />
                <span className="text-[12.5px] text-ink-2">秒</span>
              </div>
            </BehRow>
          )}
          {pos === 'P1' && (
            <BehRow title="回前台触发间隔" sub="冷启动必触发；回前台且距上次进入后台 > N 分钟才再次触发（仍受每日 1 次频控）。5–1440 分钟，默认 30">
              <div className="flex items-center gap-2">
                <input type="number" min={5} max={1440} className="field-input !w-[84px] text-center font-mono" value={form.resumeGapMinutes} onChange={(e) => set({ resumeGapMinutes: e.target.value })} />
                <span className="text-[12.5px] text-ink-2">分钟</span>
              </div>
            </BehRow>
          )}
          {pos === 'P2' && (
            <BehRow title="红点角标" sub="浮标右上角红点">
              <Switch checked={form.badge} onChange={(v) => set({ badge: v })} />
            </BehRow>
          )}
        </Section>

        <PopupFrequency position={pos} closable={form.closable} />
        <PopupEffectNote />
      </fieldset>

      <div className="sticky top-0">
        <PopupPreview form={form} />
      </div>
    </div>
  );

  return (
    <DrawerShell
      open
      title={form?.id ? `编辑弹窗 · ${form.name || '未命名'}` : target === 'new' ? '新建弹窗' : '弹窗详情'}
      sub={
        form
          ? `${form.id ? `#${form.id}` : '保存后生成 ID'} · ${pos} ${caps.name}${detail && form.id ? ` · 最后修改 ${fmtLocalTime(detail.updatedAt)}` : ''}`
          : ''
      }
      onClose={onClose}
      body={body}
      foot={
        <>
          <span className="mr-auto flex items-center gap-[6px] text-[12px] text-muted">
            <ClockIcon className="w-[14px] h-[14px]" />
            保存后：冷启动必拉 / 回前台距上次拉取 &gt;5 分钟重拉；<b style={{ color: '#92400e' }}>最坏 = 用户下次回前台</b>
          </span>
          <Button onClick={onClose}>{canEdit ? '取消' : '关闭'}</Button>
          {canEdit && (
            <Button variant="primary" disabled={!form || save.isPending} onClick={() => void doSave()}>
              <SaveCheckIcon />
              {save.isPending ? '保存中…' : '保存'}
            </Button>
          )}
        </>
      }
    />
  );
}

function ClearBtn({ onClick }: { onClick: () => void }) {
  return (
    <button type="button" onClick={onClick} className="flex-none text-[12px] text-muted hover:text-down px-2 py-1">
      清除
    </button>
  );
}

function DrawerShell({
  open,
  title,
  sub,
  onClose,
  body,
  foot,
}: {
  open: boolean;
  title: string;
  sub: string;
  onClose: () => void;
  body: ReactNode;
  foot: ReactNode;
}) {
  return (
    <>
      <div
        onClick={onClose}
        className={cn('fixed inset-0 z-40 transition', open ? 'opacity-100 visible' : 'opacity-0 invisible')}
        style={{ background: 'rgba(15,23,42,.45)', backdropFilter: 'blur(2px)' }}
      />
      <aside
        className={cn(
          'fixed top-0 right-0 h-full w-[1140px] max-w-[96vw] z-50 flex flex-col bg-bg shadow-lg2 transition-transform duration-300',
          open ? 'translate-x-0' : 'translate-x-full',
        )}
        style={{ transitionTimingFunction: 'cubic-bezier(.4,0,.1,1)' }}
      >
        <div className="flex-none flex items-center gap-3 px-6 py-[18px] bg-panel border-b border-line">
          <div className="flex-1 min-w-0">
            <div className="text-[16px] font-bold truncate">{title}</div>
            <div className="text-[12px] text-muted mt-0.5 font-mono">{sub}</div>
          </div>
          <button onClick={onClose} className="grid place-items-center w-[38px] h-[38px] rounded-[10px] border border-line text-ink-2 hover:bg-bg" title="关闭">
            <CloseIcon className="w-[18px] h-[18px]" />
          </button>
        </div>
        <div className="overflow-auto px-6 py-5 flex-1">{body}</div>
        <div className="flex-none flex items-center justify-end gap-[10px] px-6 py-3 bg-panel border-t border-line">{foot}</div>
      </aside>
    </>
  );
}
