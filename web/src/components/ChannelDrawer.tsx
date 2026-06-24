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
import type { ChannelInput, DomainEntry } from '@/lib/types';
import { useBrands, useChannels, useSaveChannel } from '@/hooks/queries';
import { useUiStore } from '@/store/uiStore';
import { validateChannel, type FieldError } from '@/lib/validation';
import { loadImageFile } from '@/lib/icon';
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
  const save = useSaveChannel();

  // ADR-0009：品牌包前缀（优先后端下发，回落 BRAND_META）。applicationId 据此派生。
  const packagePrefix =
    brands?.find((b) => b.code === brandMeta.code)?.packagePrefix ?? brandMeta.packagePrefix;

  const editing = target && target !== 'new' ? target.id : null;
  const editChannel = useMemo(
    () => (editing ? channels?.find((c) => c.id === editing) : undefined),
    [editing, channels],
  );

  const [form, setForm] = useState<ChannelInput>(blankForm(brandMeta.code));
  const [icon, setIcon] = useState<IconState>(emptyIconState());
  const [splash, setSplash] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);

  // 打开/切换目标时初始化表单
  useEffect(() => {
    if (!target) return;
    setTouched(false);
    if (editChannel) {
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
      });
      setIcon(emptyIconState(editChannel.iconMasterUrl ?? null));
      setSplash(editChannel.splashUrl ?? null);
    } else {
      setForm(blankForm(brandMeta.code));
      setIcon(emptyIconState());
      setSplash(null);
    }
  }, [target, editChannel, brandMeta.code]);

  const open = target !== null;

  // 实时校验（含唯一性）。仅在用户触碰过后展示，避免初始就标红。
  const errors: Err[] = useMemo(() => {
    if (!open) return [];
    return validateChannel(form, channels ?? [], editing ?? undefined);
  }, [open, form, channels, editing]);

  const errOf = (field: Err['field']): string | undefined =>
    touched ? errors.find((e) => e.field === field)?.message : undefined;

  const canSave = errors.length === 0;

  function set<K extends keyof ChannelInput>(key: K, value: ChannelInput[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  /** flavor 变更时，同步派生只读 applicationId（ADR-0009：不手填）。 */
  function setFlavor(flavor: string) {
    setForm((f) => ({
      ...f,
      flavorName: flavor,
      applicationId: deriveApplicationId(f.brandCode, flavor, packagePrefix),
    }));
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
              <Field label="Flavor 名" required error={errOf('flavorName')}>
                <input
                  className="field-input mono"
                  placeholder="ap01060"
                  value={form.flavorName}
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
            {/* ADR-0009：applicationId 由「品牌包前缀 + flavor」派生、只读展示，不手填、不查重歧义。 */}
            <Field
              label="包名 applicationId"
              hint="自动派生：品牌包前缀 + flavor（只读）"
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
                  {packagePrefix}.<b className="text-ink-2">{form.flavorName || '…'}</b>
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
              <DomainEditor
                domains={brandMeta.fallbackDomains.map((url, i) => ({
                  position: i,
                  url,
                  enabled: true,
                  health: 'unknown',
                }))}
                onChange={() => {}}
                disabled
              />
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
  };
}
