/**
 * 新增/编辑上架包抽屉 —— 仿照 ChannelDrawer 的分区结构与提交模式：
 *  1) 基本信息（品牌/平台/包名/代号/参考渠道包/状态/备注）；
 *  2) B 面域名：直接复用 DomainEditor（与品牌小渠道包同一套域名，ADR-0006 继承语义）；
 *  3) AB 面网关：ListingGateSection（重点，见该组件注释）；
 *  4) 归因：AF 两个字段 + 复用 AdjustSection。
 *
 * 提交模式与 Channel 不同：单个「保存」按钮触发 create/update，api 层负责按正确顺序
 * （基本信息 → 域名 → 网关规则 → 总开关）串行提交多个后端端点（见 lib/api.ts）。
 * 保存成功后**不自动关闭**——AB 面网关配置天然是「保存→试算→调整→再试算→开启」的迭代过程，
 * 强行关闭会打断这个流程；新建保存成功后改为原地切换进「编辑」态（试算/判定流水需要已有 id）。
 */
import { useEffect, useMemo, useState } from 'react';
import type {
  BrandCode,
  DomainEntry,
  ListingInput,
  ListingOpenMode,
  ListingPlatform,
  ListingStatus,
} from '@/lib/types';
import { BRAND_META, BRAND_ORDER } from '@/lib/brands';
import { FORCED_A_COUNTRIES, parseLines } from '@/lib/listingMeta';
import { validateDomains } from '@/lib/validation';
import { useBrands, useListings, useSaveListing } from '@/hooks/queries';
import { cn } from '@/lib/cn';
import { Button, Select, SectionHeading, Switch } from './ui';
import { CloseIcon, SaveCheckIcon } from './icons';
import { Field } from './FormField';
import { DomainEditor } from './DomainEditor';
import { AdjustSection } from './AdjustSection';
import { ListingGateSection } from './ListingGateSection';

interface ListingFormState {
  brandCode: BrandCode;
  platform: ListingPlatform;
  bundleId: string;
  name: string;
  displayName: string;
  palCode: string;
  storeUrl: string;
  status: ListingStatus;
  openMode: ListingOpenMode;
  remark: string;
  afDevKey: string;
  afAppId: string;
  adjustAppToken: string;
  adjustEvents: Record<string, string>;
  useBrandDomains: boolean;
  domains: DomainEntry[];
  gateEnabled: boolean;
  countries: string[];
  timezoneDeny: string[];
  ipAllowText: string;
  ipDenyText: string;
}

const EMPTY_DOMAINS: DomainEntry[] = [{ position: 0, url: '', enabled: true, health: 'unconfigured' }];

const STATUS_OPTIONS: { value: ListingStatus; label: string }[] = [
  { value: 'enabled', label: '启用中' },
  { value: 'disabled', label: '已停用' },
  { value: 'archived', label: '已归档' },
];

export function ListingDrawer({
  target,
  defaultPlatform,
  onClose,
  onCreated,
}: {
  target: 'new' | { id: string } | null;
  defaultPlatform: ListingPlatform;
  onClose: () => void;
  /** 新建保存成功后回调，携带新 id；父组件据此把 target 切到编辑态，抽屉保持打开。 */
  onCreated: (id: string) => void;
}) {
  const { data: listings } = useListings();
  const { data: brands } = useBrands();
  const save = useSaveListing();

  const editing = target && target !== 'new' ? target.id : null;
  const editListing = useMemo(
    () => (editing ? listings?.find((l) => l.id === editing) : undefined),
    [editing, listings],
  );

  const [form, setForm] = useState<ListingFormState>(blankForm(defaultPlatform));
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [warnMsg, setWarnMsg] = useState<string | null>(null);
  const [savedFlash, setSavedFlash] = useState(false);

  const open = target !== null;

  // 打开/切换目标时初始化表单：编辑态从存量数据回显，新建态清空。
  useEffect(() => {
    if (!target) return;
    setErrorMsg(null);
    setWarnMsg(null);
    if (editListing) {
      setForm({
        brandCode: editListing.brandCode,
        platform: editListing.platform,
        bundleId: editListing.bundleId,
        name: editListing.name,
        displayName: editListing.displayName,
        palCode: editListing.palCode ?? '',
        storeUrl: editListing.storeUrl,
        status: editListing.status,
        openMode: editListing.openMode ?? 'internal',
        remark: editListing.remark,
        afDevKey: editListing.afDevKey,
        afAppId: editListing.afAppId,
        adjustAppToken: editListing.adjustAppToken ?? '',
        adjustEvents: editListing.adjustEvents ?? {},
        useBrandDomains: editListing.useBrandDomains,
        domains: editListing.domains ?? EMPTY_DOMAINS,
        gateEnabled: editListing.gateEnabled,
        countries: editListing.gate?.countries ?? [],
        timezoneDeny: editListing.gate?.timezoneDeny ?? [],
        ipAllowText: (editListing.gate?.ipAllowCidrs ?? []).join('\n'),
        ipDenyText: (editListing.gate?.ipDenyCidrs ?? []).join('\n'),
      });
    } else if (target === 'new') {
      setForm(blankForm(defaultPlatform));
    }
  }, [target, editListing, defaultPlatform]);

  function set<K extends keyof ListingFormState>(key: K, value: ListingFormState[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  // 继承品牌时展示的域名：优先后端真实配置，缺省回落静态兜底（与 ChannelDrawer 同一套逻辑）。
  const selectedBrand = brands?.find((b) => b.code === form.brandCode);
  const inheritedDomains = useMemo<DomainEntry[]>(() => {
    const real = (selectedBrand?.domains ?? []).filter((d) => d.url.trim());
    if (real.length) return real.map((d) => ({ ...d }));
    return BRAND_META[form.brandCode].fallbackDomains.map((url, i) => ({
      position: i,
      url,
      enabled: true,
      health: 'unknown' as const,
    }));
  }, [selectedBrand, form.brandCode]);

  const domainError = useMemo(() => {
    if (form.useBrandDomains) return null;
    return validateDomains(form.domains)[0] ?? null;
  }, [form.useBrandDomains, form.domains]);

  async function submit() {
    setWarnMsg(null);
    const errors = validate(form, !!editing);
    if (errors.length > 0) {
      setErrorMsg(errors[0]);
      return;
    }
    setErrorMsg(null);

    // 提交前兜底：即便选择器已排除 CN/US，仍防手输/粘贴带入（后端也会 400，这里提前过滤+提示）。
    const rawCountries = form.countries.map((c) => c.trim().toUpperCase()).filter(Boolean);
    const filteredCountries = rawCountries.filter((c) => !FORCED_A_COUNTRIES.includes(c));
    const droppedForced = filteredCountries.length !== rawCountries.length;

    const input: ListingInput = {
      brandCode: form.brandCode,
      platform: form.platform,
      bundleId: form.bundleId.trim(),
      name: form.name.trim(),
      displayName: form.displayName.trim(),
      palCode: form.palCode.trim(),
      storeUrl: form.storeUrl.trim(),
      status: form.status,
      openMode: form.openMode,
      remark: form.remark.trim(),
      afDevKey: form.afDevKey.trim(),
      afAppId: form.afAppId.trim(),
      adjustAppToken: form.adjustAppToken.trim(),
      adjustEvents: form.adjustEvents,
      useBrandDomains: form.useBrandDomains,
      domains: form.useBrandDomains ? undefined : form.domains,
      gateEnabled: form.gateEnabled,
      gate: {
        countries: filteredCountries,
        timezoneDeny: form.timezoneDeny,
        ipAllowCidrs: parseLines(form.ipAllowText),
        ipDenyCidrs: parseLines(form.ipDenyText),
      },
    };

    try {
      const saved = await save.mutateAsync({ id: editing ?? undefined, input });
      setSavedFlash(true);
      setTimeout(() => setSavedFlash(false), 2200);
      if (droppedForced) {
        setWarnMsg('已自动移除国家白名单中的 CN/US（这两个国家强制走 A 面，不可加入白名单）');
      }
      if (!editing) onCreated(saved.id);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : '保存失败');
    }
  }

  return (
    <>
      <div
        onClick={onClose}
        className={cn('fixed inset-0 z-40 transition', open ? 'opacity-100 visible' : 'opacity-0 invisible')}
        style={{ background: 'rgba(15,23,42,.45)', backdropFilter: 'blur(2px)' }}
      />
      <aside
        className={cn(
          'fixed top-0 right-0 h-full w-[560px] max-w-[94vw] z-50 flex flex-col bg-bg shadow-lg2 transition-transform duration-300',
          open ? 'translate-x-0' : 'translate-x-full',
        )}
        style={{ transitionTimingFunction: 'cubic-bezier(.4,0,.1,1)' }}
      >
        {/* 头 */}
        <div className="flex-none flex items-center gap-3 px-6 py-5 bg-panel border-b border-line">
          <div className="flex-1">
            <div className="text-[16px] font-bold">
              {editing ? `编辑上架包 · ${editListing?.name ?? ''}` : '新增上架包'}
            </div>
            <div className="text-[12px] text-muted mt-0.5">
              平台：{form.platform === 'android' ? 'Android' : 'iOS'}
              {editListing && ` · ${editListing.bundleId}`}
            </div>
          </div>
          <button
            onClick={onClose}
            className="grid place-items-center w-[38px] h-[38px] rounded-[10px] border border-line text-ink-2 hover:bg-bg"
          >
            <CloseIcon className="w-[18px] h-[18px]" />
          </button>
        </div>

        {/* 体 */}
        <div className="overflow-auto px-6 py-5 flex-1 flex flex-col gap-5">
          {/* 1. 基本信息 */}
          <div className="section-card">
            <SectionHeading num={1}>基本信息</SectionHeading>
            <div className="grid grid-cols-2 gap-3">
              <Field label="所属品牌" required>
                <Select
                  value={form.brandCode}
                  onChange={(v) => set('brandCode', v as BrandCode)}
                  options={BRAND_ORDER.map((code) => ({ value: code, label: `${BRAND_META[code].name}（${code}）` }))}
                />
              </Field>
              <Field label="平台" required hint="创建后不可更改">
                <Select
                  value={form.platform}
                  disabled={!!editing}
                  onChange={(v) => set('platform', v as ListingPlatform)}
                  options={[
                    { value: 'android', label: 'Android' },
                    { value: 'ios', label: 'iOS' },
                  ]}
                />
              </Field>
            </div>
            <Field label="包名 Bundle ID" required>
              <input
                className="field-input mono"
                placeholder="com.example.app"
                value={form.bundleId}
                disabled={!!editing}
                onChange={(e) => set('bundleId', e.target.value)}
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="内部代号" required hint="如 ColorStack">
                <input className="field-input" value={form.name} onChange={(e) => set('name', e.target.value)} />
              </Field>
              <Field label="商店展示名" hint="选填">
                <input
                  className="field-input"
                  value={form.displayName}
                  onChange={(e) => set('displayName', e.target.value)}
                />
              </Field>
            </div>
            <Field label="参考渠道包 PAL_CODE" required hint="手填参考渠道包的 palcode（纯数字），拼 B 面 /?palcode=">
              <input
                className="field-input mono"
                placeholder="如 1053259"
                inputMode="numeric"
                value={form.palCode}
                onChange={(e) => set('palCode', e.target.value)}
              />
            </Field>
            {editing && (
              <Field label="状态">
                <Select value={form.status} onChange={(v) => set('status', v as ListingStatus)} options={STATUS_OPTIONS} />
              </Field>
            )}
            <Field label="商店详情页链接" hint="选填">
              <input
                className="field-input mono"
                placeholder="https://play.google.com/store/apps/details?id=…"
                value={form.storeUrl}
                onChange={(e) => set('storeUrl', e.target.value)}
              />
            </Field>
            <Field label="备注" hint="选填">
              <input className="field-input" value={form.remark} onChange={(e) => set('remark', e.target.value)} />
            </Field>
          </div>

          {/* 2. B 面域名 */}
          <div className="section-card">
            <SectionHeading num={2}>
              B 面域名 <span className="font-normal text-muted text-[11.5px]">· 与品牌小渠道包同一套域名</span>
            </SectionHeading>
            <div className="flex items-center gap-[10px] p-[11px_13px] bg-panel-2 border border-line rounded-[10px] mb-[14px]">
              <Switch checked={form.useBrandDomains} onChange={(v) => set('useBrandDomains', v)} />
              <div className="flex-1">
                <div className="text-[13px] font-semibold">继承品牌默认域名</div>
                <div className="text-[11.5px] text-muted">关闭后可为本上架包单独配置（覆盖品牌默认）</div>
              </div>
            </div>
            {form.useBrandDomains ? (
              <DomainEditor domains={inheritedDomains} onChange={() => {}} disabled />
            ) : (
              <>
                <DomainEditor domains={form.domains} onChange={(next) => set('domains', next)} />
                {domainError && <div className="text-[12px] text-down mb-2">{domainError}</div>}
              </>
            )}
          </div>

          {/* 3. AB 面网关（重点） */}
          <div className="section-card">
            <SectionHeading num={3}>
              AB 面网关 <span className="font-normal text-muted text-[11.5px]">· 按 IP 国家判定放行 B 面（ADR-0014）</span>
            </SectionHeading>
            <Field label="打开方式" hint="仅在判定进入 B 面时生效；A 面始终展示 App 本体">
              <Select
                value={form.openMode}
                onChange={(v) => set('openMode', v as ListingOpenMode)}
                options={[
                  { value: 'internal', label: '内开（App 内 WebView 打开 B 面）' },
                  { value: 'external', label: '外开（系统浏览器打开 B 面）' },
                ]}
              />
            </Field>
            <ListingGateSection
              listingId={editing}
              gateEnabled={form.gateEnabled}
              onGateEnabledChange={(v) => set('gateEnabled', v)}
              draft={{
                countries: form.countries,
                timezoneDeny: form.timezoneDeny,
                ipAllowText: form.ipAllowText,
                ipDenyText: form.ipDenyText,
              }}
              onDraftChange={(next) =>
                setForm((f) => ({
                  ...f,
                  countries: next.countries,
                  timezoneDeny: next.timezoneDeny,
                  ipAllowText: next.ipAllowText,
                  ipDenyText: next.ipDenyText,
                }))
              }
            />
          </div>

          {/* 4. 归因 */}
          <div className="section-card">
            <SectionHeading num={4}>归因</SectionHeading>
            <div className="grid grid-cols-2 gap-3 mb-4">
              <Field label="AppsFlyer Dev Key" hint="账号级；留空则不启用 AF">
                <input
                  className="field-input mono"
                  value={form.afDevKey}
                  onChange={(e) => set('afDevKey', e.target.value)}
                  placeholder="AF Dev Key"
                />
              </Field>
              <Field label="AppsFlyer App ID" hint="iOS 形如 id123456789，Android 为包名">
                <input
                  className="field-input mono"
                  value={form.afAppId}
                  onChange={(e) => set('afAppId', e.target.value)}
                  placeholder="id123456789 / 包名"
                />
              </Field>
            </div>
            <AdjustSection
              appToken={form.adjustAppToken}
              onAppTokenChange={(v) => set('adjustAppToken', v)}
              events={form.adjustEvents}
              onEventsChange={(v) => set('adjustEvents', v)}
            />
          </div>
        </div>

        {/* 脚 */}
        <div className="flex-none flex items-center justify-between gap-[10px] px-6 py-[14px] bg-panel border-t border-line">
          <div className="text-[12px] flex-1 min-w-0 truncate">
            {errorMsg ? (
              <span className="text-down">{errorMsg}</span>
            ) : warnMsg ? (
              <span className="text-[#92681a]">{warnMsg}</span>
            ) : savedFlash ? (
              <span className="text-[#15803d]">已保存</span>
            ) : null}
          </div>
          <div className="flex gap-[10px] flex-none">
            <Button onClick={onClose}>关闭</Button>
            <Button variant="primary" onClick={submit} disabled={save.isPending}>
              <SaveCheckIcon />
              {save.isPending ? '保存中…' : editing ? '保存' : '创建并继续配置'}
            </Button>
          </div>
        </div>
      </aside>
    </>
  );
}

function blankForm(platform: ListingPlatform): ListingFormState {
  return {
    brandCode: 'ap',
    platform,
    bundleId: '',
    name: '',
    displayName: '',
    palCode: '',
    storeUrl: '',
    status: 'enabled',
    openMode: 'internal',
    remark: '',
    afDevKey: '',
    afAppId: '',
    adjustAppToken: '',
    adjustEvents: {},
    useBrandDomains: true,
    domains: EMPTY_DOMAINS,
    gateEnabled: false,
    countries: [],
    timezoneDeny: [],
    ipAllowText: '',
    ipDenyText: '',
  };
}

/** 轻量表单校验：非空必填 + 域名格式（未继承时）+ 网关自洽（开启则国家白名单非空）。 */
function validate(form: ListingFormState, editing: boolean): string[] {
  const errors: string[] = [];
  if (!form.name.trim()) errors.push('内部代号必填');
  if (!editing && !form.bundleId.trim()) errors.push('包名 Bundle ID 必填');
  const pal = form.palCode.trim();
  if (!pal) errors.push('参考渠道包 PAL_CODE 必填');
  else if (!/^\d+$/.test(pal)) errors.push('参考渠道包 PAL_CODE 应为纯数字串');
  if (!form.useBrandDomains) {
    const de = validateDomains(form.domains);
    if (de.length) errors.push(de[0]);
  }
  if (form.gateEnabled && form.countries.filter((c) => c.trim()).length === 0) {
    errors.push('开启 AB 面前必须配置国家白名单');
  }
  return errors;
}
