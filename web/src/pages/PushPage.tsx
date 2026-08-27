/**
 * 推送管理（07-push.md / 09-listing.md §6）—— Console 端推送活动编辑、触达预估、历史查看。
 *
 * 顶部 segmented 切换「渠道推送 / 上架包推送」两个并列面板：
 *  - 渠道推送（原有）：BrandTabs + 子 tab（编辑 / 历史），渠道多选直接复用
 *    PackPage L111–171 的全选/勾选/已选计数模式。
 *  - 上架包推送（新增）：无品牌概念，目标是「上架包」（ColorStack/DeckTallyPro）多选，
 *    按 android/ios 分组；服务端强制只投递最近判定为 B 面的设备（无 UI 选项可绕过）；
 *    后端仅创建/列表/发送 3 个端点、无编辑草稿接口，故创建后表单锁定，只能重新开始。
 * 表单校验：lib/pushValidation.ts（仿 ChannelDrawer 的 useState+validation 模式）。
 * 图片上传：pushApi.uploadImage（复用 multipart 管线，推送封面单图，两个面板共用）。
 * Feature gate：GET /api/push/status → enabled=false 时顶部挂提示条，发送按钮置灰。
 * 历史列表：仿 BuildsPage 的 section-card 行布局。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  useChannels,
  useCreateListingCampaign,
  useListingCampaigns,
  useListings,
  usePushAudience,
  usePushCampaigns,
  usePushStatus,
  useSavePushCampaign,
  useSchedulePushCampaign,
  useSendListingCampaign,
  useSendPushCampaign,
} from '@/hooks/queries';
import { useScopedBrands } from '@/hooks/useScopedBrands';
import { useUiStore } from '@/store/uiStore';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { iconInitials } from '@/lib/brands';
import { pushApi } from '@/lib/api';
import { validatePushBody, validatePushCampaign, validateDeeplinkPath, validatePushTitle, validateListingCampaign } from '@/lib/pushValidation';
import { PLATFORM_META, PLATFORM_ORDER } from '@/lib/listingMeta';
import type {
  BrandCode,
  Listing,
  ListingCampaign,
  ListingCampaignInput,
  ListingCampaignSendResult,
  ListingPlatform,
  PushCampaign,
  PushCampaignInput,
  PushSendResult,
} from '@/lib/types';
import { BrandTabs } from '@/components/BrandTabs';
import { AppIcon, Button, Note, SectionHeading } from '@/components/ui';
import { CalendarIcon, GridIcon, InfoIcon, LayersIcon, SendIcon, UploadIcon } from '@/components/icons';
import { cn } from '@/lib/cn';
import { timeAgo } from '@/lib/text';
import { BRAND_META } from '@/lib/brands';

/** 顶层切换：渠道推送（原有） / 上架包推送（新增）。 */
type PushMode = 'channel' | 'listing';

// ── 推送封面图上传（单图，直接内嵌到表单）─────────────────────────────────
function CoverUploader({
  value,
  onChange,
  canUpload = true,
}: {
  value: string | undefined;
  onChange: (url: string | undefined) => void;
  /** push:create（10-rbac.md：新建/编辑活动、上传图片）；false 时仅展示已有封面，隐藏上传/移除入口。 */
  canUpload?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  if (!canUpload) {
    return value ? (
      <img src={value} alt="封面" className="w-[120px] h-[72px] rounded-[10px] object-cover border border-line" />
    ) : (
      <span className="text-[11.5px] text-muted">当前账号无权限上传封面图</span>
    );
  }

  async function handleFile(file: File) {
    if (!file.type.startsWith('image/')) return;
    setUploading(true);
    try {
      const reader = new FileReader();
      const dataUrl = await new Promise<string>((res, rej) => {
        reader.onload = () => res(reader.result as string);
        reader.onerror = rej;
        reader.readAsDataURL(file);
      });
      const url = await pushApi.uploadImage(dataUrl);
      onChange(url);
    } catch {
      // 上传失败静默（mock 模式下 dataUrl 直接作为预览）
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="flex items-start gap-3">
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        className={cn(
          'relative w-[120px] h-[72px] rounded-[10px] border-2 border-dashed flex flex-col items-center justify-center gap-1 text-muted transition',
          'hover:border-brand hover:text-brand',
          uploading && 'opacity-60 pointer-events-none',
        )}
      >
        {value ? (
          <img src={value} alt="封面" className="absolute inset-0 w-full h-full object-cover rounded-[8px]" />
        ) : (
          <>
            <UploadIcon className="w-5 h-5" />
            <span className="text-[11px]">{uploading ? '上传中…' : '上传封面'}</span>
          </>
        )}
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) void handleFile(f);
          e.target.value = '';
        }}
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange(undefined)}
          className="text-[11.5px] text-down hover:underline mt-1"
        >
          移除封面
        </button>
      )}
      <span className="text-[11.5px] text-muted mt-1 leading-relaxed">
        推送通知封面图（选填）
        <br />
        建议 2:1 横版，≤ 1 MB
      </span>
    </div>
  );
}

// ── extra key-value 编辑器 ─────────────────────────────────────────────────
function ExtraEditor({
  value,
  onChange,
}: {
  value: Record<string, string>;
  onChange: (v: Record<string, string>) => void;
}) {
  const entries = Object.entries(value);

  function setKey(oldKey: string, newKey: string) {
    const next: Record<string, string> = {};
    for (const [k, v] of Object.entries(value)) {
      next[k === oldKey ? newKey : k] = v;
    }
    onChange(next);
  }

  function setVal(key: string, val: string) {
    onChange({ ...value, [key]: val });
  }

  function remove(key: string) {
    const next = { ...value };
    delete next[key];
    onChange(next);
  }

  function add() {
    let n = 1;
    while (value[`key${n}`] !== undefined) n++;
    onChange({ ...value, [`key${n}`]: '' });
  }

  return (
    <div className="flex flex-col gap-1.5">
      {entries.map(([k, v]) => (
        <div key={k} className="flex gap-2">
          <input
            className="field-input mono flex-1 text-[12px]"
            placeholder="key"
            value={k}
            onChange={(e) => setKey(k, e.target.value)}
          />
          <input
            className="field-input mono flex-[2] text-[12px]"
            placeholder="value"
            value={v}
            onChange={(e) => setVal(k, e.target.value)}
          />
          <button
            type="button"
            onClick={() => remove(k)}
            className="text-[11px] text-down hover:underline px-1"
          >
            删
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={add}
        className="text-[11.5px] text-brand hover:underline self-start"
      >
        + 添加 extra 字段
      </button>
    </div>
  );
}

// ── 触达预估展示条 ──────────────────────────────────────────────────────────
function AudienceBadge({ appIds }: { appIds: string[] }) {
  const { data, isFetching } = usePushAudience(appIds);
  if (appIds.length === 0) return null;
  return (
    <div className="flex items-center gap-2 px-3 py-2 rounded-[10px] bg-[rgba(99,102,241,.06)] border border-[rgba(99,102,241,.18)] text-[12.5px]">
      <InfoIcon className="w-4 h-4 text-brand flex-none" />
      {isFetching ? (
        <span className="text-muted">预估触达设备数计算中…</span>
      ) : data ? (
        <span>
          预估触达 <strong className="text-ink">{data.totalDevices.toLocaleString()}</strong> 台设备（
          {appIds.length} 个渠道包）
        </span>
      ) : null}
    </div>
  );
}

// ── 推送活动状态徽标 ────────────────────────────────────────────────────────
function StatusPill({ status }: { status: PushCampaign['status'] }) {
  const map: Record<PushCampaign['status'], { cls: string; txt: string }> = {
    draft: { cls: 'text-[#475569] bg-[#e2e8f0]', txt: '草稿' },
    scheduled: { cls: 'text-[#92681a] bg-[#fef3c7]', txt: '定时中' },
    sending: { cls: 'text-[#1e40af] bg-[#dbeafe]', txt: '发送中' },
    done: { cls: 'text-[#15803d] bg-[#dcfce7]', txt: '已完成' },
    failed: { cls: 'text-[#b91c1c] bg-[#fee2e2]', txt: '失败' },
  };
  const { cls, txt } = map[status];
  return (
    <span className={cn('text-[11px] font-semibold px-[9px] py-[3px] rounded-full whitespace-nowrap', cls)}>
      {txt}
    </span>
  );
}

// ── 历史列表行 ─────────────────────────────────────────────────────────────
function CampaignRow({ campaign }: { campaign: PushCampaign }) {
  return (
    <div className="section-card">
      <div className="flex items-start gap-3 flex-wrap">
        <div
          className="grid place-items-center w-8 h-8 rounded-lg text-white font-bold text-[13px] flex-none"
          style={{ background: 'linear-gradient(135deg,#6366f1,#8b5cf6)' }}
        >
          推
        </div>
        <div className="min-w-0 flex-1">
          <div className="font-bold text-[14px] flex items-center gap-2 flex-wrap">
            <span>{campaign.name}</span>
            <StatusPill status={campaign.status} />
          </div>
          <div className="text-[12.5px] text-ink-2 mt-1 line-clamp-1">
            {campaign.title} — {campaign.body}
          </div>
          <div className="text-[11.5px] text-muted mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
            <span>{campaign.targetAppIds.length} 个渠道包</span>
            {campaign.sentAt && <span>发送于 {timeAgo(campaign.sentAt)}</span>}
            {campaign.scheduledAt && campaign.status === 'scheduled' && (
              <span>定时 {new Date(campaign.scheduledAt).toLocaleString('zh-CN', { hour12: false })}</span>
            )}
            {campaign.createdBy && <span>by {campaign.createdBy}</span>}
          </div>
        </div>
        {campaign.status === 'done' && (
          <div className="flex items-center gap-3 text-[12px] flex-none">
            <span className="text-[#15803d] font-semibold">
              ✓ {campaign.successCount.toLocaleString()} 成功
            </span>
            {campaign.failureCount > 0 && (
              <span className="text-down font-semibold">
                ✕ {campaign.failureCount.toLocaleString()} 失败
              </span>
            )}
          </div>
        )}
      </div>
      {campaign.deeplinkPath && (
        <div className="mt-2 text-[11.5px] font-mono text-muted bg-bg px-2 py-1 rounded-md inline-block">
          deeplink: {campaign.deeplinkPath}
        </div>
      )}
      {campaign.imageUrl && (
        <img
          src={campaign.imageUrl}
          alt="封面"
          className="mt-2 h-14 rounded-[8px] object-cover border border-line"
        />
      )}
    </div>
  );
}

// ── 渠道多选（直接复用 PackPage L111–171 的全选/勾选模式） ──────────────────
function ChannelPicker({
  appIds,
  onChangeAppIds,
  brand,
  accentColor,
  channels,
}: {
  appIds: Set<string>;
  onChangeAppIds: (next: Set<string>) => void;
  brand: BrandCode;
  accentColor: string;
  channels: { id: string; applicationId: string; appName: string; flavorName: string; status: string; iconMasterUrl?: string }[];
}) {
  function toggle(appId: string) {
    const next = new Set(appIds);
    if (next.has(appId)) next.delete(appId);
    else next.add(appId);
    onChangeAppIds(next);
  }

  function toggleAll() {
    if (appIds.size < channels.length) {
      onChangeAppIds(new Set(channels.map((c) => c.applicationId)));
    } else {
      onChangeAppIds(new Set());
    }
  }

  return (
    <div>
      <div className="flex items-center gap-3 mb-3">
        <strong className="text-[13px]">选择目标渠道包</strong>
        <span className="text-[12px] text-muted">
          已选 {appIds.size} / {channels.length}
        </span>
        <div className="ml-auto">
          <button className="chip" onClick={toggleAll}>
            全选 / 取消
          </button>
        </div>
      </div>
      <div className="flex flex-col gap-2 max-h-[360px] overflow-auto pr-1">
        {channels.map((c) => {
          const sel = appIds.has(c.applicationId);
          return (
            <div
              key={c.id}
              onClick={() => toggle(c.applicationId)}
              className={cn(
                'flex items-center gap-3 px-[13px] py-[11px] border rounded-[11px] bg-panel cursor-pointer transition',
                sel
                  ? 'border-brand bg-[rgba(99,102,241,.05)] shadow-[0_0_0_1px_var(--brand)_inset]'
                  : 'border-line hover:bg-panel-2 hover:border-[#dfe6f0]',
              )}
            >
              <span
                className={cn(
                  'grid place-items-center w-5 h-5 rounded-md border-[1.5px] flex-none transition',
                  sel ? 'bg-brand border-brand' : 'border-[#cbd5e1]',
                )}
              >
                {sel && (
                  <svg viewBox="0 0 24 24" className="w-[13px] h-[13px] text-white" fill="none" stroke="currentColor" strokeWidth={3}>
                    <path d="M20 6 9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </span>
              <AppIcon
                initials={iconInitials(c.appName, brand)}
                hex={accentColor}
                src={c.iconMasterUrl}
                size={34}
                radius={9}
              />
              <span className="min-w-0">
                <span className="block text-[13.5px] font-semibold truncate">{c.appName}</span>
                <span className="block text-[11px] text-muted font-mono">{c.applicationId}</span>
              </span>
              {c.status === 'disabled' && (
                <span className="ml-auto text-[10px] text-[#64748b] bg-[#f1f5f9] px-2 py-0.5 rounded-full">
                  已停用
                </span>
              )}
            </div>
          );
        })}
        {channels.length === 0 && (
          <div className="text-center text-muted py-10">该品牌暂无渠道</div>
        )}
      </div>
    </div>
  );
}

// ── 编辑面板 ───────────────────────────────────────────────────────────────
function EditPanel({
  pushEnabled,
  brand,
  accentColor,
}: {
  pushEnabled: boolean;
  brand: BrandCode;
  accentColor: string;
}) {
  const { data: channelsAll } = useChannels();
  const canCreate = useAuthStore((s) => s.hasPerm(PERM.PUSH_CREATE));
  const canSend = useAuthStore((s) => s.hasPerm(PERM.PUSH_SEND));

  const channels = useMemo(
    () => (channelsAll ?? []).filter((c) => c.brandCode === brand && c.status !== 'archived'),
    [channelsAll, brand],
  );

  // 表单状态（仿 ChannelDrawer）
  const [name, setName] = useState('');
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [imageUrl, setImageUrl] = useState<string | undefined>();
  const [deeplinkPath, setDeeplinkPath] = useState('');
  const [extra, setExtra] = useState<Record<string, string>>({});
  const [pickedAppIds, setPickedAppIds] = useState<Set<string>>(new Set());
  const [sendMode, setSendMode] = useState<'instant' | 'scheduled'>('instant');
  const [scheduledAt, setScheduledAt] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitErr, setSubmitErr] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  // dry-run 预览结果（点「预发测试」后展示，真发前可再确认）
  const [dryRunResult, setDryRunResult] = useState<PushSendResult | null>(null);

  const saveMutation = useSavePushCampaign();
  const sendMutation = useSendPushCampaign();
  const scheduleMutation = useSchedulePushCampaign();

  // 触达预估：选好渠道后拉取
  const selectedAppIds = useMemo(() => [...pickedAppIds], [pickedAppIds]);

  function buildInput(): PushCampaignInput {
    return {
      name: name.trim() || title.trim(),
      title: title.trim(),
      body: body.trim(),
      imageUrl: imageUrl || undefined,
      deeplinkPath: deeplinkPath.trim() || undefined,
      extraData: Object.keys(extra).length ? extra : undefined,
      targetAppIds: selectedAppIds,
    };
  }

  function validate(): boolean {
    const errs = validatePushCampaign(buildInput());
    const map: Record<string, string> = {};
    for (const e of errs) map[e.field] = e.message;
    setErrors(map);
    return errs.length === 0;
  }

  function resetForm() {
    setName(''); setTitle(''); setBody(''); setImageUrl(undefined);
    setDeeplinkPath(''); setExtra({}); setPickedAppIds(new Set());
    setSendMode('instant'); setScheduledAt('');
    setDryRunResult(null);
  }

  /** 预发测试（dry-run）：保存草稿后以 dryRun=true 调发送接口，展示 preview，不污染历史。 */
  async function handleDryRun() {
    if (!validate()) return;
    setSubmitErr(null);
    setSuccess(null);
    setDryRunResult(null);
    try {
      const saved = await saveMutation.mutateAsync({ input: buildInput() });
      const result = await sendMutation.mutateAsync({ id: saved.id, dryRun: true });
      // result.campaign 保持 draft 状态；展示 preview 供用户确认
      setDryRunResult(result);
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '预发测试失败');
    }
  }

  /** 真实发送：保存草稿 → 即时发/定时，使用 result.campaign.status 反映结果。 */
  async function handleSend() {
    if (!validate()) return;
    setSubmitErr(null);
    setSuccess(null);
    setDryRunResult(null);
    try {
      const saved = await saveMutation.mutateAsync({ input: buildInput() });
      if (sendMode === 'instant') {
        const result = await sendMutation.mutateAsync({ id: saved.id, dryRun: false });
        const st = result.campaign.status;
        setSuccess(st === 'done' ? '推送发送成功' : st === 'sending' ? '推送已触发，正在发送中' : '推送已提交');
      } else {
        if (!scheduledAt) { setErrors((p) => ({ ...p, scheduledAt: '请选择定时时间' })); return; }
        await scheduleMutation.mutateAsync({ id: saved.id, scheduledAt: new Date(scheduledAt).toISOString() });
        setSuccess(`已定时：${new Date(scheduledAt).toLocaleString('zh-CN', { hour12: false })}`);
      }
      resetForm();
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '操作失败');
    }
  }

  async function handleSaveDraft() {
    if (!validate()) return;
    setSubmitErr(null);
    setSuccess(null);
    try {
      await saveMutation.mutateAsync({ input: buildInput() });
      setSuccess('草稿已保存');
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '保存草稿失败');
    }
  }

  const isPending = saveMutation.isPending || sendMutation.isPending || scheduleMutation.isPending;
  const canFireSend = canSend && pickedAppIds.size > 0 && pushEnabled && !isPending;
  const canDryRun = canCreate && pickedAppIds.size > 0 && !isPending;

  return (
    <div className="grid gap-[18px] items-start" style={{ gridTemplateColumns: '1fr 360px' }}>
      {/* 左：内容表单 */}
      <div className="flex flex-col gap-[14px]">
        <div className="section-card">
          <SectionHeading num="1">推送内容</SectionHeading>

          {/* 活动名称 */}
          <div className="mb-[13px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
              活动名称 <span className="font-normal text-muted text-[11.5px]">（内部备注，不展示给用户）</span>
            </label>
            <input
              className="field-input"
              placeholder="如：618 大促活动"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          {/* 标题 */}
          <div className="mb-[13px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
              通知标题 <span className="text-down">*</span>{' '}
              <span className="font-normal text-muted text-[11.5px]">最多 50 字</span>
            </label>
            <input
              className={cn('field-input', errors.title && 'border-down')}
              placeholder="如：618 专属福利来袭！"
              value={title}
              maxLength={60}
              onChange={(e) => {
                setTitle(e.target.value);
                const err = validatePushTitle(e.target.value);
                setErrors((p) => ({ ...p, title: err ?? '' }));
              }}
            />
            <div className="flex justify-between mt-1">
              {errors.title ? <span className="text-[12px] text-down">{errors.title}</span> : <span />}
              <span className="text-[11px] text-muted">{title.length}/50</span>
            </div>
          </div>

          {/* 正文 */}
          <div className="mb-[13px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
              通知正文 <span className="text-down">*</span>{' '}
              <span className="font-normal text-muted text-[11.5px]">最多 500 字</span>
            </label>
            <textarea
              className={cn('field-input resize-none', errors.body && 'border-down')}
              rows={3}
              placeholder="如：今日限时：充值满 100 送 20，点击领取优惠券"
              value={body}
              maxLength={520}
              onChange={(e) => {
                setBody(e.target.value);
                const err = validatePushBody(e.target.value);
                setErrors((p) => ({ ...p, body: err ?? '' }));
              }}
            />
            <div className="flex justify-between mt-1">
              {errors.body ? <span className="text-[12px] text-down">{errors.body}</span> : <span />}
              <span className="text-[11px] text-muted">{body.length}/500</span>
            </div>
          </div>

          {/* 封面图 */}
          <div className="mb-[13px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">封面图（选填）</label>
            <CoverUploader value={imageUrl} onChange={setImageUrl} canUpload={canCreate} />
          </div>

          {/* Deeplink path */}
          <div className="mb-[13px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
              Deeplink Path <span className="font-normal text-muted text-[11.5px]">（选填，如 /promo/618，须以 / 开头）</span>
            </label>
            <input
              className={cn('field-input mono', errors.deeplinkPath && 'border-down')}
              placeholder="/promo/618"
              value={deeplinkPath}
              onChange={(e) => {
                setDeeplinkPath(e.target.value);
                const err = validateDeeplinkPath(e.target.value);
                setErrors((p) => ({ ...p, deeplinkPath: err ?? '' }));
              }}
            />
            {errors.deeplinkPath && (
              <div className="mt-1 text-[12px] text-down">{errors.deeplinkPath}</div>
            )}
          </div>

          {/* Extra data */}
          <div>
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
              Extra Data <span className="font-normal text-muted text-[11.5px]">（选填，KV 附加参数）</span>
            </label>
            <ExtraEditor value={extra} onChange={setExtra} />
          </div>
        </div>
      </div>

      {/* 右：渠道多选 + 发送配置 */}
      <div className="flex flex-col gap-[14px]">
        <div className="section-card">
          <SectionHeading num="2">目标渠道包</SectionHeading>
          {errors.targetAppIds && (
            <div className="mb-2 text-[12px] text-down">{errors.targetAppIds}</div>
          )}
          <ChannelPicker
            appIds={pickedAppIds}
            onChangeAppIds={setPickedAppIds}
            brand={brand}
            accentColor={accentColor}
            channels={channels}
          />
          {pickedAppIds.size > 0 && (
            <div className="mt-3">
              <AudienceBadge appIds={selectedAppIds} />
            </div>
          )}
        </div>

        <div className="section-card">
          <SectionHeading num="3">发送设置</SectionHeading>

          {/* 发送方式 */}
          <div className="flex gap-2 mb-[14px]">
            {(['instant', 'scheduled'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => setSendMode(mode)}
                className={cn(
                  'flex-1 px-3 py-2 rounded-[10px] border text-[13px] font-medium transition',
                  sendMode === mode
                    ? 'border-brand bg-[rgba(99,102,241,.07)] text-ink'
                    : 'border-line text-muted hover:border-[#dfe6f0]',
                )}
              >
                {mode === 'instant' ? '即时发送' : '定时发送'}
              </button>
            ))}
          </div>

          {sendMode === 'scheduled' && (
            <div className="mb-[14px]">
              <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                定时时间 <span className="text-down">*</span>
              </label>
              <div className="relative">
                <CalendarIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted pointer-events-none" />
                <input
                  type="datetime-local"
                  className={cn('field-input pl-9', errors.scheduledAt && 'border-down')}
                  value={scheduledAt}
                  onChange={(e) => {
                    setScheduledAt(e.target.value);
                    setErrors((p) => ({ ...p, scheduledAt: '' }));
                  }}
                  min={new Date().toISOString().slice(0, 16)}
                />
              </div>
              {errors.scheduledAt && (
                <div className="mt-1 text-[12px] text-down">{errors.scheduledAt}</div>
              )}
            </div>
          )}

          {/* dry-run 预览结果（点「预发测试」后展示，真发前确认用）*/}
          {dryRunResult?.preview && (
            <div className="mb-3 rounded-[10px] border border-[rgba(99,102,241,.25)] bg-[rgba(99,102,241,.06)] px-3 py-2.5 text-[12.5px]">
              <div className="font-semibold text-ink mb-1">
                预发测试结果 — 预计触达{' '}
                <strong className="text-brand">{dryRunResult.preview.totalDevices.toLocaleString()}</strong> 台设备
              </div>
              <div className="text-muted text-[11.5px] flex flex-col gap-0.5 max-h-[80px] overflow-auto">
                {Object.entries(dryRunResult.preview.byApp).map(([appId, n]) => (
                  <span key={appId} className="font-mono">
                    {appId.split('.').pop()} · {n.toLocaleString()} 台
                  </span>
                ))}
              </div>
              <div className="mt-1.5 text-[11px] text-muted">活动仍为草稿状态，未实际发送。确认无误后点「立即发送」。</div>
            </div>
          )}

          {/* 发送 & 保存草稿：按 push:send / push:create 分别显隐 */}
          <div className="flex flex-col gap-2">
            {canSend && (
              <Button
                variant="primary"
                className="w-full justify-center"
                disabled={!canFireSend}
                onClick={() => void handleSend()}
              >
                <SendIcon className="w-4 h-4" />
                {isPending
                  ? '处理中…'
                  : sendMode === 'instant'
                  ? `立即发送 (${pickedAppIds.size})`
                  : '确认定时'}
              </Button>
            )}
            {canCreate && (
              <>
                <Button
                  variant="ghost"
                  className="w-full justify-center"
                  disabled={!canDryRun}
                  onClick={() => void handleDryRun()}
                >
                  预发测试（dry-run）
                </Button>
                <Button
                  variant="ghost"
                  className="w-full justify-center"
                  disabled={isPending}
                  onClick={() => void handleSaveDraft()}
                >
                  保存草稿
                </Button>
              </>
            )}
          </div>

          {!pushEnabled && canSend && (
            <div className="mt-2 text-[11.5px] text-muted text-center">
              FCM 未配置，发送按钮已禁用；可编辑、选包、保存草稿、预发测试
            </div>
          )}
          {!canSend && !canCreate && (
            <div className="mt-2 text-[11.5px] text-muted text-center">
              当前账号无推送编辑/发送权限（push:create / push:send），仅可查看
            </div>
          )}
          {submitErr && <div className="mt-2 text-[12px] text-down">{submitErr}</div>}
          {success && (
            <div className="mt-2 text-[12px] text-[#15803d] bg-[#dcfce7] rounded-[8px] px-3 py-2">
              {success}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ── 历史面板 ───────────────────────────────────────────────────────────────
function HistoryPanel({ brand }: { brand: BrandCode }) {
  const { data: campaigns, isLoading } = usePushCampaigns(brand);

  if (isLoading) {
    return <div className="h-40 rounded-card bg-panel border border-line animate-pulse" />;
  }
  if (!campaigns?.length) {
    return <div className="text-center text-muted py-[60px]">暂无推送记录</div>;
  }
  return (
    <div className="flex flex-col gap-3">
      {campaigns.map((c) => (
        <CampaignRow key={c.id} campaign={c} />
      ))}
    </div>
  );
}

// =============================================================================
// 上架包推送（09-listing.md §6）—— 目标是「上架包」而非渠道，强制只投 B 面设备。
// =============================================================================

/** preview.byApp 的 key 形如 `listing:<id>`（`<id>` 即 Listing.id）；映射回名称+平台展示。 */
function listingLabelForKey(key: string, listings: Listing[]): string {
  const raw = key.includes(':') ? key.slice(key.indexOf(':') + 1) : key;
  const l = listings.find((x) => x.id === raw);
  return l ? `${l.name}（${PLATFORM_META[l.platform].label}）` : key;
}

// ── 上架包多选（按平台分组：Android / iOS，仿 ChannelPicker） ────────────────
function ListingPicker({
  listingIds,
  onChange,
  listings,
  disabled,
}: {
  listingIds: Set<string>;
  onChange: (next: Set<string>) => void;
  listings: Listing[];
  disabled?: boolean;
}) {
  function toggle(id: string) {
    if (disabled) return;
    const next = new Set(listingIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onChange(next);
  }

  const grouped = useMemo(() => {
    const g: Record<ListingPlatform, Listing[]> = { android: [], ios: [] };
    for (const l of listings) {
      if (l.status === 'archived') continue;
      g[l.platform].push(l);
    }
    return g;
  }, [listings]);

  const total = grouped.android.length + grouped.ios.length;
  if (total === 0) {
    return <div className="text-center text-muted py-10">暂无可选上架包，请先在「上架包」页新增</div>;
  }

  return (
    <div className={cn('flex flex-col gap-4', disabled && 'opacity-60 pointer-events-none')}>
      {PLATFORM_ORDER.map((p) => {
        const items = grouped[p];
        if (items.length === 0) return null;
        const meta = PLATFORM_META[p];
        return (
          <div key={p}>
            <div className="flex items-center gap-2 mb-2">
              <span className="w-[9px] h-[9px] rounded-full flex-none" style={{ background: meta.accentColor }} />
              <strong className="text-[12.5px]">{meta.label}</strong>
              <span className="text-[11px] text-muted">{items.length} 个上架包</span>
            </div>
            <div className="flex flex-col gap-2">
              {items.map((l) => {
                const sel = listingIds.has(l.id);
                const initials =
                  l.name.replace(/[^A-Za-z]/g, '').slice(0, 2).toUpperCase() || l.name.slice(0, 2).toUpperCase();
                return (
                  <div
                    key={l.id}
                    onClick={() => toggle(l.id)}
                    className={cn(
                      'flex items-center gap-3 px-[13px] py-[11px] border rounded-[11px] bg-panel cursor-pointer transition',
                      sel
                        ? 'border-brand bg-[rgba(99,102,241,.05)] shadow-[0_0_0_1px_var(--brand)_inset]'
                        : 'border-line hover:bg-panel-2 hover:border-[#dfe6f0]',
                    )}
                  >
                    <span
                      className={cn(
                        'grid place-items-center w-5 h-5 rounded-md border-[1.5px] flex-none transition',
                        sel ? 'bg-brand border-brand' : 'border-[#cbd5e1]',
                      )}
                    >
                      {sel && (
                        <svg
                          viewBox="0 0 24 24"
                          className="w-[13px] h-[13px] text-white"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth={3}
                        >
                          <path d="M20 6 9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
                        </svg>
                      )}
                    </span>
                    <AppIcon initials={initials} hex={meta.accentColor} size={34} radius={9} />
                    <span className="min-w-0">
                      <span className="block text-[13.5px] font-semibold truncate">{l.name}</span>
                      <span className="block text-[11px] text-muted font-mono">{l.bundleId}</span>
                    </span>
                    {!l.gateEnabled && (
                      <span className="ml-auto text-[10px] text-[#64748b] bg-[#f1f5f9] px-2 py-0.5 rounded-full whitespace-nowrap">
                        AB 面未开 · 0 台
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── 上架包推送历史行（仿 CampaignRow，复用 StatusPill） ──────────────────────
function ListingCampaignRow({ campaign, listings }: { campaign: ListingCampaign; listings: Listing[] }) {
  const targetNames = campaign.listingIds.length
    ? campaign.listingIds.map((id) => listings.find((l) => l.id === id)?.name ?? `#${id}`).join('、')
    : '（目标上架包信息缺失）';
  return (
    <div className="section-card">
      <div className="flex items-start gap-3 flex-wrap">
        <div
          className="grid place-items-center w-8 h-8 rounded-lg text-white font-bold text-[13px] flex-none"
          style={{ background: 'linear-gradient(135deg,#16a34a,#0ea5e9)' }}
        >
          架
        </div>
        <div className="min-w-0 flex-1">
          <div className="font-bold text-[14px] flex items-center gap-2 flex-wrap">
            <span>{campaign.name}</span>
            <StatusPill status={campaign.status} />
          </div>
          <div className="text-[12.5px] text-ink-2 mt-1 line-clamp-1">
            {campaign.title} — {campaign.body}
          </div>
          <div className="text-[11.5px] text-muted mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
            <span>目标：{targetNames}</span>
            {campaign.sentAt && <span>发送于 {timeAgo(campaign.sentAt)}</span>}
            {campaign.createdBy && <span>by {campaign.createdBy}</span>}
          </div>
        </div>
        {(campaign.status === 'done' || campaign.status === 'sending') && (
          <div className="flex items-center gap-3 text-[12px] flex-none">
            <span className="text-[#15803d] font-semibold">✓ {campaign.successCount.toLocaleString()} 成功</span>
            {campaign.failureCount > 0 && (
              <span className="text-down font-semibold">✕ {campaign.failureCount.toLocaleString()} 失败</span>
            )}
          </div>
        )}
      </div>
      {campaign.deeplinkPath && (
        <div className="mt-2 text-[11.5px] font-mono text-muted bg-bg px-2 py-1 rounded-md inline-block">
          deeplink: {campaign.deeplinkPath}
        </div>
      )}
      {campaign.imageUrl && (
        <img src={campaign.imageUrl} alt="封面" className="mt-2 h-14 rounded-[8px] object-cover border border-line" />
      )}
    </div>
  );
}

// ── 上架包推送：编辑面板 ──────────────────────────────────────────────────────
function ListingEditPanel({ pushEnabled, listings }: { pushEnabled: boolean; listings: Listing[] }) {
  const canCreate = useAuthStore((s) => s.hasPerm(PERM.PUSH_CREATE));
  const canSend = useAuthStore((s) => s.hasPerm(PERM.PUSH_SEND));
  const [name, setName] = useState('');
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [imageUrl, setImageUrl] = useState<string | undefined>();
  const [deeplinkPath, setDeeplinkPath] = useState('');
  const [extra, setExtra] = useState<Record<string, string>>({});
  const [pickedListingIds, setPickedListingIds] = useState<Set<string>>(new Set());
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitErr, setSubmitErr] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  // 已创建的草稿活动 id：09-listing.md §6 只有创建/列表/发送三个端点，没有编辑草稿的
  // 接口——一旦创建成功即锁定表单，避免「看起来能改但改了不会生效」的错觉。
  const [draftId, setDraftId] = useState<string | null>(null);
  const [dryRunResult, setDryRunResult] = useState<ListingCampaignSendResult | null>(null);

  const createMutation = useCreateListingCampaign();
  const sendMutation = useSendListingCampaign();

  const selectedListingIds = useMemo(() => [...pickedListingIds], [pickedListingIds]);
  const locked = draftId !== null;

  function buildInput(): ListingCampaignInput {
    return {
      name: name.trim() || title.trim(),
      title: title.trim(),
      body: body.trim(),
      imageUrl: imageUrl || undefined,
      deeplinkPath: deeplinkPath.trim() || undefined,
      extraData: Object.keys(extra).length ? extra : undefined,
      listingIds: selectedListingIds,
    };
  }

  function validate(): boolean {
    const errs = validateListingCampaign(buildInput());
    const map: Record<string, string> = {};
    for (const e of errs) map[e.field] = e.message;
    setErrors(map);
    return errs.length === 0;
  }

  function resetForm() {
    setName('');
    setTitle('');
    setBody('');
    setImageUrl(undefined);
    setDeeplinkPath('');
    setExtra({});
    setPickedListingIds(new Set());
    setDraftId(null);
    setDryRunResult(null);
    setErrors({});
    setSubmitErr(null);
    setSuccess(null);
  }

  /** 创建草稿（幂等）：已创建过就复用同一 id——后端没有编辑草稿的接口。 */
  async function ensureDraft(): Promise<string> {
    if (draftId) return draftId;
    const created = await createMutation.mutateAsync(buildInput());
    setDraftId(created.id);
    return created.id;
  }

  /** 试算受众（dry-run）：首次点击时顺带创建草稿并锁定表单。 */
  async function handleDryRun() {
    if (!locked && !validate()) return;
    setSubmitErr(null);
    setSuccess(null);
    try {
      const id = await ensureDraft();
      const result = await sendMutation.mutateAsync({ id, dryRun: true });
      setDryRunResult(result);
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '试算受众失败');
    }
  }

  /** 正式发送：要求已看过一次预演结果（发送前必看 B 面受众数），避免误发给真实用户。 */
  async function handleSend() {
    if (!draftId || !dryRunResult) return;
    setSubmitErr(null);
    setSuccess(null);
    try {
      await sendMutation.mutateAsync({ id: draftId, dryRun: false });
      setSuccess('推送已触发，正在发送中（仅投递最近判定为 B 面的设备）');
      resetForm();
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '发送失败');
    }
  }

  const isPending = createMutation.isPending || sendMutation.isPending;
  const canDryRun = canCreate && pickedListingIds.size > 0 && !isPending;
  const canFireSend = canSend && locked && dryRunResult != null && pushEnabled && !isPending;

  return (
    <div className="flex flex-col gap-[14px]">
      <Note>
        <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
        <div>
          上架包推送<b>仅投递给最近判定为 B 面的设备</b>（服务端硬过滤{' '}
          <span className="mono">last_gate_mode=&apos;B&apos;</span>，A 面/审核设备绝不会收到，无 UI 开关可绕过）。
        </div>
      </Note>

      <div className="grid gap-[18px] items-start" style={{ gridTemplateColumns: '1fr 360px' }}>
        {/* 左：内容表单（草稿创建后用 fieldset 整体锁定，原生禁用所有内部控件） */}
        <div className="flex flex-col gap-[14px]">
          <div className={cn('section-card', locked && 'opacity-75')}>
            <SectionHeading num="1">推送内容</SectionHeading>
            <fieldset disabled={locked} className="contents">
              <div className="mb-[13px]">
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                  活动名称 <span className="font-normal text-muted text-[11.5px]">（内部备注，不展示给用户）</span>
                </label>
                <input
                  className="field-input"
                  placeholder="如：ColorStack 新关卡推广"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>

              <div className="mb-[13px]">
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                  通知标题 <span className="text-down">*</span>{' '}
                  <span className="font-normal text-muted text-[11.5px]">最多 50 字</span>
                </label>
                <input
                  className={cn('field-input', errors.title && 'border-down')}
                  placeholder="如：新关卡上线，立即体验！"
                  value={title}
                  maxLength={60}
                  onChange={(e) => {
                    setTitle(e.target.value);
                    const err = validatePushTitle(e.target.value);
                    setErrors((p) => ({ ...p, title: err ?? '' }));
                  }}
                />
                <div className="flex justify-between mt-1">
                  {errors.title ? <span className="text-[12px] text-down">{errors.title}</span> : <span />}
                  <span className="text-[11px] text-muted">{title.length}/50</span>
                </div>
              </div>

              <div className="mb-[13px]">
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                  通知正文 <span className="text-down">*</span>{' '}
                  <span className="font-normal text-muted text-[11.5px]">最多 500 字</span>
                </label>
                <textarea
                  className={cn('field-input resize-none', errors.body && 'border-down')}
                  rows={3}
                  placeholder="如：20 个全新关卡已上线，点击立即体验"
                  value={body}
                  maxLength={520}
                  onChange={(e) => {
                    setBody(e.target.value);
                    const err = validatePushBody(e.target.value);
                    setErrors((p) => ({ ...p, body: err ?? '' }));
                  }}
                />
                <div className="flex justify-between mt-1">
                  {errors.body ? <span className="text-[12px] text-down">{errors.body}</span> : <span />}
                  <span className="text-[11px] text-muted">{body.length}/500</span>
                </div>
              </div>

              <div className="mb-[13px]">
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">封面图（选填）</label>
                <CoverUploader value={imageUrl} onChange={setImageUrl} canUpload={canCreate} />
              </div>

              <div className="mb-[13px]">
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                  Deeplink Path{' '}
                  <span className="font-normal text-muted text-[11.5px]">
                    （选填，如 /promo/new-levels，须以 / 开头）
                  </span>
                </label>
                <input
                  className={cn('field-input mono', errors.deeplinkPath && 'border-down')}
                  placeholder="/promo/new-levels"
                  value={deeplinkPath}
                  onChange={(e) => {
                    setDeeplinkPath(e.target.value);
                    const err = validateDeeplinkPath(e.target.value);
                    setErrors((p) => ({ ...p, deeplinkPath: err ?? '' }));
                  }}
                />
                {errors.deeplinkPath && <div className="mt-1 text-[12px] text-down">{errors.deeplinkPath}</div>}
              </div>

              <div>
                <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                  Extra Data <span className="font-normal text-muted text-[11.5px]">（选填，KV 附加参数）</span>
                </label>
                <ExtraEditor value={extra} onChange={setExtra} />
              </div>
            </fieldset>
          </div>
        </div>

        {/* 右：上架包多选 + 试算/发送 */}
        <div className="flex flex-col gap-[14px]">
          <div className="section-card">
            <SectionHeading num="2">目标上架包</SectionHeading>
            {errors.listingIds && <div className="mb-2 text-[12px] text-down">{errors.listingIds}</div>}
            <ListingPicker
              listingIds={pickedListingIds}
              onChange={setPickedListingIds}
              listings={listings}
              disabled={locked}
            />
          </div>

          <div className="section-card">
            <SectionHeading num="3">试算 & 发送</SectionHeading>

            {dryRunResult?.preview && (
              <div className="mb-3 rounded-[10px] border border-[rgba(99,102,241,.25)] bg-[rgba(99,102,241,.06)] px-3 py-2.5 text-[12.5px]">
                <div className="font-semibold text-ink mb-1">
                  B 面受众预估 — 共{' '}
                  <strong className="text-brand">{dryRunResult.preview.totalDevices.toLocaleString()}</strong> 台设备
                </div>
                <div className="text-muted text-[11.5px] flex flex-col gap-0.5 max-h-[100px] overflow-auto">
                  {Object.entries(dryRunResult.preview.byApp).map(([key, n]) => (
                    <span key={key}>
                      {listingLabelForKey(key, listings)} · {n.toLocaleString()} 台
                    </span>
                  ))}
                </div>
                <div className="mt-1.5 text-[11px] text-muted">
                  确认无误后点击「正式发送」；数字有变化可点「重新预演」刷新。
                </div>
              </div>
            )}

            <div className="flex flex-col gap-2">
              {!locked ? (
                canCreate ? (
                  <Button
                    variant="ghost"
                    className="w-full justify-center"
                    disabled={!canDryRun}
                    onClick={() => void handleDryRun()}
                  >
                    {isPending ? '创建中…' : '试算受众（创建草稿并预演）'}
                  </Button>
                ) : (
                  <div className="text-[11.5px] text-muted text-center py-2">
                    当前账号无推送草稿创建权限（push:create）
                  </div>
                )
              ) : (
                <>
                  {canSend && (
                    <Button
                      variant="primary"
                      className="w-full justify-center"
                      disabled={!canFireSend}
                      onClick={() => void handleSend()}
                    >
                      <SendIcon className="w-4 h-4" />
                      {isPending ? '处理中…' : '正式发送（仅投 B 面设备）'}
                    </Button>
                  )}
                  {canCreate && (
                    <Button
                      variant="ghost"
                      className="w-full justify-center"
                      disabled={isPending}
                      onClick={() => void handleDryRun()}
                    >
                      重新预演
                    </Button>
                  )}
                  <Button variant="ghost" className="w-full justify-center" disabled={isPending} onClick={resetForm}>
                    重新开始（放弃当前草稿）
                  </Button>
                </>
              )}
            </div>

            {locked && (
              <div className="mt-2 text-[11.5px] text-muted text-center">
                草稿已创建（活动 #{draftId}），内容与目标包已锁定不可再改；如需调整请「重新开始」。
              </div>
            )}
            {!pushEnabled && (
              <div className="mt-2 text-[11.5px] text-muted text-center">
                FCM 未配置，正式发送已禁用；试算受众仍可正常使用。
              </div>
            )}
            {submitErr && <div className="mt-2 text-[12px] text-down">{submitErr}</div>}
            {success && (
              <div className="mt-2 text-[12px] text-[#15803d] bg-[#dcfce7] rounded-[8px] px-3 py-2">{success}</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── 上架包推送：历史面板 ──────────────────────────────────────────────────────
function ListingHistoryPanel({ listings }: { listings: Listing[] }) {
  const { data: campaigns, isLoading } = useListingCampaigns();

  if (isLoading) {
    return <div className="h-40 rounded-card bg-panel border border-line animate-pulse" />;
  }
  if (!campaigns?.length) {
    return <div className="text-center text-muted py-[60px]">暂无上架包推送记录</div>;
  }
  return (
    <div className="flex flex-col gap-3">
      {campaigns.map((c) => (
        <ListingCampaignRow key={c.id} campaign={c} listings={listings} />
      ))}
    </div>
  );
}

// ── 上架包推送：主面板（子 tab 编辑/历史，仿渠道推送） ────────────────────────
function ListingPushPanel({ pushEnabled }: { pushEnabled: boolean }) {
  const { data: listingsAll } = useListings();
  const listings = useMemo(() => (listingsAll ?? []).filter((l) => l.status !== 'archived'), [listingsAll]);
  const [subTab, setSubTab] = useState<'edit' | 'history'>('edit');

  return (
    <section>
      {!pushEnabled && (
        <div className="flex items-center gap-2 px-4 py-2.5 rounded-[10px] mb-4 text-[12.5px] font-medium bg-[#fef3c7] text-[#92681a] border border-[#fde68a]">
          <InfoIcon className="w-4 h-4 flex-none" />
          FCM 未配置（后端 PUSH_ENABLED=false），正式发送已禁用 —— 编辑、选包、试算受众与历史查看仍可正常使用。
        </div>
      )}

      <div className="flex gap-1 mb-5 border-b border-line">
        {(
          [
            ['edit', '编辑推送'],
            ['history', '发送历史'],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setSubTab(key)}
            className={cn(
              'px-4 py-2.5 text-[13.5px] font-medium border-b-2 -mb-px transition',
              subTab === key ? 'border-brand text-brand' : 'border-transparent text-muted hover:text-ink',
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {subTab === 'edit' ? (
        <ListingEditPanel pushEnabled={pushEnabled} listings={listings} />
      ) : (
        <ListingHistoryPanel listings={listings} />
      )}
    </section>
  );
}

// ── 主页面 ─────────────────────────────────────────────────────────────────
export function PushPage() {
  const { brands, brand } = useScopedBrands();
  const { data: pushStatus } = usePushStatus();

  const currentBrand = useUiStore((s) => s.currentBrand);
  const setCurrentBrand = useUiStore((s) => s.setCurrentBrand);

  const [mode, setMode] = useState<PushMode>('channel');
  const [subTab, setSubTab] = useState<'edit' | 'history'>('edit');

  // 切品牌时重置到编辑 tab
  useEffect(() => {
    setSubTab('edit');
  }, [currentBrand]);

  const accentColor = brand?.accentColor ?? BRAND_META[currentBrand]?.accentColor ?? '#6366f1';

  const pushEnabled = pushStatus?.enabled ?? false;
  const brandEnabled = pushStatus?.brands[currentBrand] ?? false;
  const showFcmWarning = !pushEnabled || !brandEnabled;

  return (
    <section>
      {/* 顶层切换：渠道推送（原有） / 上架包推送（新增） */}
      <div className="inline-flex p-1 mb-5 rounded-[12px] bg-panel-2 border border-line gap-1">
        {(
          [
            ['channel', '渠道推送', GridIcon],
            ['listing', '上架包推送', LayersIcon],
          ] as const
        ).map(([key, label, Icon]) => (
          <button
            key={key}
            onClick={() => setMode(key)}
            className={cn(
              'flex items-center gap-1.5 px-4 py-[9px] rounded-[9px] text-[13px] font-semibold transition',
              mode === key ? 'bg-panel text-ink shadow-sm2' : 'text-muted hover:text-ink',
            )}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      {mode === 'channel' ? (
        <>
          {/* FCM feature gate 提示条 */}
          {showFcmWarning && (
            <div className="flex items-center gap-2 px-4 py-2.5 rounded-[10px] mb-4 text-[12.5px] font-medium bg-[#fef3c7] text-[#92681a] border border-[#fde68a]">
              <InfoIcon className="w-4 h-4 flex-none" />
              FCM 未配置，发送已禁用 —— 编辑、选包、保存草稿与历史查看仍可正常使用。
              {!pushEnabled ? '（后端 PUSH_ENABLED=false）' : `（${currentBrand.toUpperCase()} 品牌未启用 FCM）`}
            </div>
          )}

          {brands && <BrandTabs brands={brands} current={currentBrand} onSelect={setCurrentBrand} />}

          {/* 子 tab 切换 */}
          <div className="flex gap-1 mb-5 border-b border-line">
            {([['edit', '编辑推送'], ['history', '发送历史']] as const).map(([key, label]) => (
              <button
                key={key}
                onClick={() => setSubTab(key)}
                className={cn(
                  'px-4 py-2.5 text-[13.5px] font-medium border-b-2 -mb-px transition',
                  subTab === key
                    ? 'border-brand text-brand'
                    : 'border-transparent text-muted hover:text-ink',
                )}
              >
                {label}
              </button>
            ))}
          </div>

          {subTab === 'edit' ? (
            <EditPanel pushEnabled={pushEnabled} brand={currentBrand} accentColor={accentColor} />
          ) : (
            <HistoryPanel brand={currentBrand} />
          )}
        </>
      ) : (
        <ListingPushPanel pushEnabled={pushEnabled} />
      )}
    </section>
  );
}
