/**
 * 新增/编辑渠道抽屉 —— 复刻原型抽屉，落地为可用表单：
 *  1) 基本信息（名称/flavor/PAL_CODE/包名）+ **实时唯一性 & 格式校验**（CLAUDE.md #5）；
 *  2) Xcode 式图标九宫格（IconNineGrid）；
 *  3) splash 上传；
 *  4) 域名配置：「继承大渠道」开关 + 关闭后渠道级覆盖（ADR-0006）。
 *
 * 唯一性是硬约束：applicationId / pal_code / (brand,flavor) 重复时禁止保存并红字提示——
 * 这正是用来拦住 ap01035 / gzmarket062 这类脏数据的闸门。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { deriveApplicationId, type BrandMeta } from '@/lib/brands';
import type { Channel, ChannelInput, DomainEntry } from '@/lib/types';
import { useBrands, useChannels, useSaveChannel, useStores } from '@/hooks/queries';
import { useUiStore } from '@/store/uiStore';
import { validateChannel, composeFlavor, type FieldError } from '@/lib/validation';
import { loadImageFile, urlToDataUrl } from '@/lib/icon';
import { cn } from '@/lib/cn';
import { Button, Note, SectionHeading, Switch } from './ui';
import { CloseIcon, InfoIcon, SaveCheckIcon } from './icons';
import { IconNineGrid, emptyIconState, type IconState } from './IconNineGrid';
import { DomainEditor } from './DomainEditor';

type Err = FieldError;

const EMPTY_DOMAINS: DomainEntry[] = [{ position: 0, url: '', enabled: true, health: 'unconfigured' }];

export function ChannelDrawer({ brandMeta }: { brandMeta: BrandMeta }) {
  const target = useUiStore((s) => s.drawerTarget);
  const close = useUiStore((s) => s.closeDrawer);
  const { data: channels } = useChannels();
  const { data: brands } = useBrands();
  const { data: stores } = useStores();
  const save = useSaveChannel();

  // 表单可选商店：仅 enabled，按 sort 升序（停用的商店不应再被选用于新渠道）。
  const storeOptions = useMemo(
    () => (stores ?? []).filter((s) => s.status === 'enabled').slice().sort((a, b) => a.sort - b.sort),
    [stores],
  );

  // 后端下发的本品牌视图（含真实域名 / 包前缀）。
  const brandView = brands?.find((b) => b.code === brandMeta.code);

  // ADR-0009：品牌包前缀（优先后端下发，回落 BRAND_META）。applicationId 据此派生。
  const packagePrefix = brandView?.packagePrefix ?? brandMeta.packagePrefix;

  // 继承大渠道时展示的域名：优先后端**真实**配置；后端未加载到才回落静态兜底常量。
  // （修复：原先恒用 brandMeta.fallbackDomains，导致后台只配 1 个也固定显示兜底的 2 个。）
  const inheritedDomains = useMemo<DomainEntry[]>(() => {
    const real = (brandView?.domains ?? []).filter((d) => d.url.trim());
    if (real.length) return real.map((d) => ({ ...d }));
    return brandMeta.fallbackDomains.map((url, i) => ({
      position: i,
      url,
      enabled: true,
      health: 'unknown' as const,
    }));
  }, [brandView, brandMeta.fallbackDomains]);

  const editing = target && target !== 'new' ? target.id : null;
  const editChannel = useMemo(
    () => (editing ? channels?.find((c) => c.id === editing) : undefined),
    [editing, channels],
  );

  const [form, setForm] = useState<ChannelInput>(blankForm(brandMeta.code));
  // 用户在表单里实际填写的「基础 flavor」（不含商店后缀）；form.flavorName 是合成值。
  const [baseFlavor, setBaseFlavor] = useState('');
  // 选中的商店 code（''  = 无/默认）。提交时据此在 stores 中查回 storeId。
  const [storeCode, setStoreCode] = useState('');
  const [icon, setIcon] = useState<IconState>(emptyIconState());
  const [splash, setSplash] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);
  // 「从已有渠道复制」反馈：复制源名 + 图片抓取中标记
  const [copiedFrom, setCopiedFrom] = useState<string | null>(null);
  const [copyingAssets, setCopyingAssets] = useState(false);

  // 打开/切换目标时初始化表单
  useEffect(() => {
    if (!target) return;
    setTouched(false);
    setCopiedFrom(null);
    setCopyingAssets(false);
    if (editChannel) {
      // 编辑态：从存量 flavorName 反解出 base + 商店 code（base = flavorName 去掉尾部 `_`+code）。
      const editStoreCode = editChannel.store?.code ?? '';
      const derivedBase = editStoreCode ? stripStoreSuffix(editChannel.flavorName, editStoreCode) : editChannel.flavorName;
      setBaseFlavor(derivedBase);
      setStoreCode(editStoreCode);
      setForm({
        brandCode: editChannel.brandCode,
        flavorName: editChannel.flavorName,
        applicationId: editChannel.applicationId,
        palCode: editChannel.palCode,
        appName: editChannel.appName,
        useBrandDomains: editChannel.useBrandDomains,
        domains: editChannel.domains ?? EMPTY_DOMAINS,
        status: editChannel.status,
        remark: editChannel.remark,
        storeId: editChannel.storeId ?? null,
      });
      setIcon(emptyIconState(editChannel.iconMasterUrl ?? null));
      setSplash(editChannel.splashUrl ?? null);
    } else {
      setBaseFlavor('');
      setStoreCode('');
      setForm(blankForm(brandMeta.code));
      setIcon(emptyIconState());
      setSplash(null);
    }
  }, [target, editChannel, brandMeta.code]);

  const open = target !== null;

  // 实时校验（含唯一性）。仅在用户触碰过后展示，避免初始就标红。
  // baseFlavor 传给格式校验（避免合成 flavor 的下划线被误判非法）；
  // form.flavorName 此时已是合成值，供唯一性/包名校验。
  const errors: Err[] = useMemo(() => {
    if (!open) return [];
    return validateChannel(form, channels ?? [], editing ?? undefined, baseFlavor);
  }, [open, form, channels, editing, baseFlavor]);

  const errOf = (field: Err['field']): string | undefined =>
    touched ? errors.find((e) => e.field === field)?.message : undefined;

  const canSave = errors.length === 0;

  // 可复制的源渠道：本品牌、非归档，按 flavor 排序（新增模式才用）。
  const copySources = useMemo<Channel[]>(
    () =>
      (channels ?? [])
        .filter((c) => c.brandCode === brandMeta.code && c.status !== 'archived')
        .sort((a, b) => a.flavorName.localeCompare(b.flavorName)),
    [channels, brandMeta.code],
  );

  function set<K extends keyof ChannelInput>(key: K, value: ChannelInput[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  /**
   * 基础 flavor 变更时：与当前选中商店合成 flavor，同步派生只读 applicationId
   * （ADR-0009：不手填；商店后缀方案：合成 flavor 里的下划线在派生包名时转点号）。
   */
  function setFlavor(nextBase: string) {
    setBaseFlavor(nextBase);
    const composed = composeFlavor(nextBase, storeCode);
    setForm((f) => ({
      ...f,
      flavorName: composed,
      applicationId: deriveApplicationId(f.brandCode, composed, packagePrefix),
    }));
  }

  /** 应用商店变更时：重新合成 flavor + 派生 applicationId。 */
  function setStore(nextCode: string) {
    setStoreCode(nextCode);
    const composed = composeFlavor(baseFlavor, nextCode);
    const store = storeOptions.find((s) => s.code === nextCode);
    setForm((f) => ({
      ...f,
      flavorName: composed,
      applicationId: deriveApplicationId(f.brandCode, composed, packagePrefix),
      storeId: store ? store.id : null,
    }));
  }

  /**
   * 从已有渠道复制（仅新增模式）：搬过来「可复用」项——应用名 / 域名配置（含继承开关）/
   * 状态 / 备注 / 图标 / 启动图；但**留空唯一字段**（flavor、派生 applicationId、PAL_CODE），
   * 强制用户填新值，从源头避免复制出重复脏数据（ADR-0009 唯一性护栏）。
   * 图标/splash 必须抓成 dataURL 才能随新渠道真正上传（见 urlToDataUrl 注释）。
   */
  function copyFrom(src: Channel) {
    setBaseFlavor('');
    setStoreCode('');
    setForm({
      brandCode: brandMeta.code,
      flavorName: '',
      applicationId: '',
      palCode: '',
      appName: src.appName,
      useBrandDomains: src.useBrandDomains,
      domains: (src.domains ?? EMPTY_DOMAINS).map((d) => ({ ...d, health: 'unconfigured' })),
      status: src.status === 'archived' ? 'enabled' : src.status,
      remark: src.remark,
      storeId: null,
    });
    setTouched(false);
    setCopiedFrom(`${src.flavorName}（${src.appName}）`);

    const iconUrl = src.iconMasterUrl;
    const splashUrl = src.splashUrl;
    setIcon(emptyIconState());
    setSplash(null);
    if (!iconUrl && !splashUrl) return;

    // 图标/splash 各抓一次源图转 dataURL；失败则仅作预览（提交时不上传，需用户重选）。
    setCopyingAssets(true);
    const jobs: Promise<unknown>[] = [];
    if (iconUrl) {
      jobs.push(
        urlToDataUrl(iconUrl)
          .then((d) => setIcon(emptyIconState(d)))
          .catch(() => setIcon(emptyIconState(iconUrl))),
      );
    }
    if (splashUrl) {
      jobs.push(
        urlToDataUrl(splashUrl)
          .then((d) => setSplash(d))
          .catch(() => setSplash(splashUrl)),
      );
    }
    void Promise.allSettled(jobs).finally(() => setCopyingAssets(false));
  }

  async function onSplashFile(file: File) {
    try {
      const img = await loadImageFile(file);
      setSplash(img.dataUrl);
    } catch {
      /* ignore */
    }
  }

  async function submit() {
    setTouched(true);
    if (!canSave) return;
    const payload: ChannelInput = {
      ...form,
      flavorName: form.flavorName.trim(),
      applicationId: form.applicationId.trim(),
      palCode: form.palCode.trim(),
      appName: form.appName.trim(),
      iconMasterDataUrl: icon.master ?? undefined,
      splashDataUrl: splash ?? undefined,
      domains: form.useBrandDomains ? undefined : form.domains,
    };
    await save.mutateAsync({ id: editing ?? undefined, input: payload });
    close();
  }

  return (
    <>
      <div
        onClick={close}
        className={cn(
          'fixed inset-0 z-40 transition',
          open ? 'opacity-100 visible' : 'opacity-0 invisible',
        )}
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
              {editing ? `编辑渠道 · ${editChannel?.appName ?? ''}` : '新增小渠道'}
            </div>
            <div className="text-[12px] text-muted mt-0.5">
              大渠道：{brandMeta.name}（{brandMeta.code}）
              {brandMeta.hmsEnabled && ' · 含 HMS/OAID'}
            </div>
          </div>
          <button
            onClick={close}
            className="grid place-items-center w-[38px] h-[38px] rounded-[10px] border border-line text-ink-2 hover:bg-bg"
          >
            <CloseIcon className="w-[18px] h-[18px]" />
          </button>
        </div>

        {/* 体 */}
        <div className="overflow-auto px-6 py-5 flex-1 flex flex-col gap-5">
          {/* 0. 从已有渠道复制（仅新增模式）—— 选一个同品牌包，搬来可复用项再改改 */}
          {!editing && copySources.length > 0 && (
            <div className="flex items-center gap-[10px] p-[11px_13px] bg-panel-2 border border-line rounded-[10px]">
              <div className="flex-none text-[12.5px] font-semibold text-ink-2 whitespace-nowrap">
                从已有渠道复制
              </div>
              <select
                className="field-input flex-1"
                value=""
                onChange={(e) => {
                  const src = copySources.find((c) => c.id === e.target.value);
                  if (src) copyFrom(src);
                }}
              >
                <option value="">选择一个包，自动带入名称/域名/图标…</option>
                {copySources.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.flavorName} · {c.appName}
                  </option>
                ))}
              </select>
            </div>
          )}
          {!editing && copiedFrom && (
            <Note>
              <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
              <div>
                已从「{copiedFrom}」复制可复用项{copyingAssets && '（图标/启动图抓取中…）'}。
                请填写本渠道的 <b>Flavor 名</b> 与 <b>PAL_CODE</b>（唯一字段不会被复制），其余按需修改。
              </div>
            </Note>
          )}

          {/* 1. 基本信息 */}
          <div className="section-card">
            <SectionHeading num={1}>基本信息</SectionHeading>
            <Field label="应用名称" required hint="桌面图标下显示" error={errOf('appName')}>
              <input
                className="field-input"
                placeholder="例如 ArenaPlus:USA Basketball Live"
                value={form.appName}
                onChange={(e) => set('appName', e.target.value)}
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="基础 Flavor" required hint="不含商店后缀，选商店后自动拼接" error={errOf('flavorName')}>
                <input
                  className="field-input mono"
                  placeholder="ap01060"
                  value={baseFlavor}
                  disabled={!!editing}
                  onChange={(e) => setFlavor(e.target.value)}
                />
              </Field>
              <Field label="PAL_CODE" required hint="URL 参数，跨品牌可重复" error={errOf('palCode')}>
                <input
                  className="field-input mono"
                  placeholder="1053259…"
                  value={form.palCode}
                  onChange={(e) => set('palCode', e.target.value)}
                />
              </Field>
            </div>
            <Field
              label="应用商店"
              hint="选中后自动把后缀拼进包名，如 _hw；不选则为默认包名"
            >
              <select
                className="field-input"
                value={storeCode}
                disabled={!!editing}
                onChange={(e) => setStore(e.target.value)}
              >
                <option value="">无（默认）</option>
                {storeOptions.map((s) => (
                  <option key={s.id} value={s.code}>
                    {s.name}（{s.code}）
                  </option>
                ))}
              </select>
              {editing && (
                <div className="mt-1 text-[11.5px] text-muted">
                  已发布渠道不可更换商店（会改变 flavor/包名，等同换包）
                </div>
              )}
            </Field>
            {/* ADR-0009：applicationId 由「品牌包前缀 + flavor」派生、只读展示，不手填、不查重歧义。
                商店后缀方案：合成 flavor（含 `_`）派生包名时下划线转点号。 */}
            <Field
              label="包名 applicationId"
              hint="自动派生：品牌包前缀 + 合成 flavor（只读）"
            >
              <div className="flex items-center gap-2">
                <input
                  className="field-input mono bg-[#f1f5f9] text-ink-2 cursor-not-allowed flex-1"
                  value={form.applicationId || `${packagePrefix}.<flavor>`}
                  readOnly
                  tabIndex={-1}
                  title="由品牌包前缀 + flavor 自动派生，不可手填（ADR-0009）"
                />
                <span className="text-[11px] text-muted font-mono whitespace-nowrap">
                  合成 flavor：<b className="text-ink-2">{form.flavorName || '…'}</b>
                </span>
              </div>
              {errOf('applicationId') && (
                <div className="mt-1 text-[12px] text-down">{errOf('applicationId')}</div>
              )}
            </Field>
          </div>

          {/* 2. 图标九宫格 */}
          <div className="section-card">
            <SectionHeading num={2}>
              App 图标{' '}
              <span className="font-normal text-muted text-[11.5px]">· 拖入一张主图自动生成全部尺寸</span>
            </SectionHeading>
            <IconNineGrid value={icon} onChange={setIcon} accentHex={brandMeta.accentColor} />
          </div>

          {/* 3. splash */}
          <div className="section-card">
            <SectionHeading num={3}>启动图 splash</SectionHeading>
            <SplashDrop value={splash} onFile={onSplashFile} onClear={() => setSplash(null)} />
          </div>

          {/* 4. 域名配置 */}
          <div className="section-card">
            <SectionHeading num={4}>
              域名配置 <span className="font-normal text-muted text-[11.5px]">· 1 主 + 最多 3 备用</span>
            </SectionHeading>
            <div className="flex items-center gap-[10px] p-[11px_13px] bg-panel-2 border border-line rounded-[10px] mb-[14px]">
              <Switch
                checked={form.useBrandDomains}
                onChange={(v) => set('useBrandDomains', v)}
              />
              <div className="flex-1">
                <div className="text-[13px] font-semibold">继承大渠道默认域名</div>
                <div className="text-[11.5px] text-muted">关闭后可为本渠道单独配置（覆盖品牌默认）</div>
              </div>
            </div>

            {form.useBrandDomains ? (
              <DomainEditor domains={inheritedDomains} onChange={() => {}} disabled />
            ) : (
              <>
                <DomainEditor
                  domains={form.domains ?? EMPTY_DOMAINS}
                  onChange={(next) => set('domains', next)}
                />
                {errOf('domains') && (
                  <div className="text-[12px] text-down mb-2">{errOf('domains')}</div>
                )}
              </>
            )}

            <Note>
              <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
              <div>
                APK 启动按 <b>主 → 备用</b> 顺序探测，命中后加载{' '}
                <span className="mono">https://域名/?palcode=…</span>。只有<b>确认域名故障</b>才切换；本机断网只提示「网络异常」<b>不乱换</b>。域名可在后台随时热更，已安装包下次启动生效，无需重新打包。
              </div>
            </Note>
          </div>
        </div>

        {/* 脚 */}
        <div className="flex-none flex justify-end gap-[10px] px-6 py-[14px] bg-panel border-t border-line">
          {touched && !canSave && (
            <span className="mr-auto self-center text-[12px] text-down">
              有 {errors.length} 处校验未通过
            </span>
          )}
          <Button onClick={close}>取消</Button>
          <Button variant="primary" onClick={submit} disabled={save.isPending || (touched && !canSave)}>
            <SaveCheckIcon />
            {save.isPending ? '保存中…' : '保存渠道'}
          </Button>
        </div>
      </aside>
    </>
  );
}

// ---------- 子组件 ----------
function Field({
  label,
  required,
  hint,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  hint?: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-[13px] last:mb-0">
      <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
        {label} {required && <span className="text-down">*</span>}{' '}
        {hint && <span className="font-normal text-muted text-[11.5px]">{hint}</span>}
      </label>
      {children}
      {error && <div className="mt-1 text-[12px] text-down">{error}</div>}
    </div>
  );
}

function SplashDrop({
  value,
  onFile,
  onClear,
}: {
  value: string | null;
  onFile: (f: File) => void;
  onClear: () => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <div
      onClick={() => ref.current?.click()}
      onDragOver={(e) => e.preventDefault()}
      onDrop={(e) => {
        e.preventDefault();
        const f = e.dataTransfer.files?.[0];
        if (f) onFile(f);
      }}
      className="rounded-[12px] p-4 text-center cursor-pointer border-2 border-dashed border-[#c7d2e6] hover:border-brand transition"
      style={{
        background: value
          ? undefined
          : 'repeating-linear-gradient(45deg,#fbfcfe,#fbfcfe 12px,#f6f8fc 12px,#f6f8fc 24px)',
      }}
    >
      {value ? (
        <div className="flex items-center gap-3 justify-center">
          <img src={value} alt="splash" className="h-16 rounded-md object-cover border border-line" />
          <div className="text-left">
            <div className="text-[13px] font-semibold">已选择启动图</div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onClear();
              }}
              className="text-[12px] text-down hover:underline"
            >
              移除
            </button>
          </div>
        </div>
      ) : (
        <>
          <div className="text-[13px] font-bold">拖入全屏启动图 splash_fullscreen</div>
          <div className="text-[12px] text-muted mt-0.5">全屏 CENTER_CROP，建议竖屏高清大图</div>
        </>
      )}
      <input
        ref={ref}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        hidden
        onChange={(e) => {
          const f = e.target.files?.[0];
          e.target.value = '';
          if (f) onFile(f);
        }}
      />
    </div>
  );
}

function blankForm(brandCode: BrandMeta['code']): ChannelInput {
  return {
    brandCode,
    flavorName: '',
    applicationId: '',
    palCode: '',
    appName: '',
    useBrandDomains: true,
    domains: EMPTY_DOMAINS,
    status: 'enabled',
    storeId: null,
  };
}

/** 反解基础 flavor：从存量合成 flavor 中去掉尾部 `_`+商店 code（编辑态回显用）。 */
function stripStoreSuffix(flavorName: string, storeCode: string): string {
  const suffix = `_${storeCode}`;
  return flavorName.endsWith(suffix) ? flavorName.slice(0, -suffix.length) : flavorName;
}
