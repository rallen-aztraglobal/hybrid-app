/**
 * 素材区（编辑抽屉分区 2）+ 通用图片上传位。
 *  - P1/P3/P7 最多 5 张（计数器、满 5 张禁用添加）、上移 / 下移 / 删除；其余位置固定 1 张。
 *  - 每个上传位标注 §1.1 建议尺寸（二倍图）；上传后显示后端返回的 key 文件名，
 *    并用返回的 width/height 比对：比例或尺寸不符给黄色提示，但不拦截保存。
 *  - 换图必须换 URL：每次上传后端都会生成新文件名（内容 hash + 时间戳），从不覆盖。
 */
import { useRef, useState, type ReactNode } from 'react';
import { popupApi } from '@/lib/api';
import { cn } from '@/lib/cn';
import { blankCard, type FormCard, type PopupForm } from '@/lib/popupForm';
import {
  MAX_CAROUSEL_CARDS,
  POSITION_CAPS,
  TAB_TEXT_HARD_MAX,
  TAB_TEXT_SOFT_MAX,
  cardImageSpec,
  charLen,
  checkImageSize,
  hasCardImage,
  imageKeyLabel,
  suggestedSizeText,
  type ImageSpec,
} from '@/lib/popupMeta';
import { toast } from '@/store/toastStore';
import { AlertIcon, ChevronDownIcon, ChevronUpIcon, ImageIcon, InfoIcon, PlusIcon, TrashIcon } from '../icons';
import { Button, Note } from '../ui';

const ACCEPT = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'];
const MAX_BYTES = 3 * 1024 * 1024;

type Meta = { key?: string; width: number; height: number };

/** 带计数器的输入框：soft 超出变黄、hard 超出变红。 */
export function CountInput({
  value,
  onChange,
  placeholder,
  soft,
  hard,
  mono,
  disabled,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  soft?: number;
  hard?: number;
  mono?: boolean;
  disabled?: boolean;
}) {
  const n = charLen(value);
  const limit = hard ?? soft;
  const color = hard != null && n > hard ? '#dc2626' : soft != null && n > soft ? '#b45309' : undefined;
  return (
    <div className="relative">
      <input
        className={cn('field-input', mono && 'font-mono text-[12.5px]', limit != null && '!pr-[52px]')}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
      />
      {limit != null && (
        <span
          className={cn('absolute right-[10px] top-1/2 -translate-y-1/2 text-[11px] font-mono', !color && 'text-muted', color && 'font-bold')}
          style={color ? { color } : undefined}
        >
          {n}/{soft != null && hard != null ? soft : limit}
        </span>
      )}
    </div>
  );
}

function SubWarn({ children }: { children: ReactNode }) {
  return (
    <div className="flex gap-[5px] items-start text-[11.5px] leading-[1.55] mt-[5px]" style={{ color: '#b45309' }}>
      <AlertIcon className="w-[13px] h-[13px] flex-none mt-[2px]" />
      <span>{children}</span>
    </div>
  );
}

/** 上传框：点击选文件 → 上传 → 回传结果。 */
export function ImageUpload({
  src,
  spec,
  canEdit,
  onUploaded,
  onNatural,
  width,
}: {
  src: string;
  spec: ImageSpec;
  canEdit: boolean;
  onUploaded: (r: { url: string; meta: Meta }) => void;
  onNatural?: (w: number, h: number) => void;
  /** 框宽(px)，高度按 spec 比例推算。 */
  width: number;
}) {
  const ref = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);

  async function pick(file: File | undefined) {
    if (!file) return;
    if (!ACCEPT.includes(file.type)) {
      toast.err('仅支持 png / jpeg / webp / gif 图片');
      return;
    }
    if (file.size > MAX_BYTES) {
      toast.err('图片不能超过 3MB');
      return;
    }
    setBusy(true);
    try {
      const r = await popupApi.uploadImage(file);
      onUploaded({ url: r.url, meta: { key: r.key, width: r.width, height: r.height } });
    } catch (err) {
      toast.err(err instanceof Error ? err.message : '上传失败');
    } finally {
      setBusy(false);
      if (ref.current) ref.current.value = '';
    }
  }

  return (
    <>
      <button
        type="button"
        disabled={!canEdit || busy}
        onClick={() => ref.current?.click()}
        className={cn(
          'group relative overflow-hidden rounded-[10px] grid place-items-center text-center text-muted text-[11px] transition flex-none',
          src ? 'border border-transparent shadow-sm2' : 'border-[1.5px] border-dashed border-[#c7d2e6] hover:border-brand hover:text-brand-ink',
          (!canEdit || busy) && 'cursor-not-allowed',
        )}
        style={{
          width,
          aspectRatio: `${spec.w}/${spec.h}`,
          background: src ? '#e2e8f0' : 'repeating-linear-gradient(45deg,#fbfcfe,#fbfcfe 10px,#f4f7fb 10px,#f4f7fb 20px)',
        }}
      >
        {src ? (
          <>
            <img
              src={src}
              alt=""
              className="absolute inset-0 w-full h-full object-cover"
              onLoad={(e) => onNatural?.(e.currentTarget.naturalWidth, e.currentTarget.naturalHeight)}
            />
            {canEdit && (
              <span className="absolute bottom-[6px] left-1/2 -translate-x-1/2 bg-[rgba(15,23,42,.78)] text-white text-[10.5px] px-2 py-0.5 rounded-[10px] opacity-0 group-hover:opacity-100 transition whitespace-nowrap">
                更换图片
              </span>
            )}
          </>
        ) : (
          <span className="flex flex-col items-center gap-[3px] p-[6px]">
            <ImageIcon className="w-5 h-5" />
            <b className="text-[12px] text-ink-2">{spec.w}×{spec.h}px</b>
            <span>{spec.ratio} · 二倍图</span>
            <span>{canEdit ? '点击上传' : '无上传权限'}</span>
          </span>
        )}
        {busy && <span className="absolute inset-0 grid place-items-center bg-white/70 text-[12px] font-semibold text-brand-ink">上传中…</span>}
      </button>
      <input ref={ref} type="file" accept={ACCEPT.join(',')} className="hidden" onChange={(e) => void pick(e.target.files?.[0])} />
    </>
  );
}

/** 上传位下方的信息：建议尺寸、文件名（后端 key）、换图必换 URL 说明、尺寸比对黄色提示。 */
export function UploadInfo({ spec, url, meta, natural }: { spec: ImageSpec; url: string; meta?: Meta; natural?: { w: number; h: number } | null }) {
  const size = meta ?? (natural ? { width: natural.w, height: natural.h } : undefined);
  const warn = url && size ? checkImageSize(spec, size.width, size.height) : null;
  const fname = meta?.key || imageKeyLabel(url);
  return (
    <>
      <div className="flex items-center gap-[5px] flex-wrap text-[11.5px] text-ink-2 mb-[6px]">
        <ImageIcon className="w-3 h-3" />
        {suggestedSizeText(spec)}
        <span className="text-muted text-[11px]">尺寸不符只提示、不拦截</span>
      </div>
      <div
        className={cn(
          'flex items-center gap-[6px] font-mono text-[11.5px] px-[9px] py-[6px] rounded-lg break-all mb-1',
          fname ? 'bg-[#eef2ff] text-brand-ink' : 'bg-[#f1f5f9] text-muted',
        )}
      >
        <ImageIcon className="w-[13px] h-[13px] flex-none" />
        <span>{fname || '未上传 · 上传后自动生成文件名'}</span>
      </div>
      <div className="text-[11px] text-muted mb-[6px]">
        {size && url ? `实际 ${size.width}×${size.height}px · ` : ''}上传即生成新文件名（内容 hash + 时间戳）· 不覆盖同名 · 换图必须换 URL
      </div>
      {warn && <SubWarn>{warn}</SubWarn>}
    </>
  );
}

function Lbl({ children, req, hint }: { children: ReactNode; req?: boolean; hint?: string }) {
  return (
    <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
      {children} {req && <span className="text-down">*</span>}{' '}
      {hint && <span className="font-normal text-muted text-[11.5px]">{hint}</span>}
    </label>
  );
}

function CardEditor({
  form,
  card,
  index,
  total,
  canEdit,
  onChange,
  onMove,
  onRemove,
}: {
  form: PopupForm;
  card: FormCard;
  index: number;
  total: number;
  canEdit: boolean;
  onChange: (patch: Partial<FormCard>) => void;
  onMove: (dir: -1 | 1) => void;
  onRemove: () => void;
}) {
  const pos = form.position;
  const caps = POSITION_CAPS[pos];
  const spec = cardImageSpec(pos);
  const hasImg = hasCardImage(pos);
  const [natural, setNatural] = useState<{ w: number; h: number } | null>(null);
  const carousel = caps.maxCards > 1;
  const upW = spec && spec.w / spec.h < 0.7 ? 118 : 84;

  return (
    <div className="rounded-xl border border-line bg-panel-2 mb-3 overflow-hidden">
      <div className="flex items-center gap-2 pl-3 pr-[10px] py-[7px] border-b border-line-2 bg-white text-[12.5px] font-bold">
        <span className="grid place-items-center w-5 h-5 rounded-md bg-ink text-white text-[11px]">{index + 1}</span>
        {carousel ? `卡片 ${index + 1}` : '素材'}
        <span className="text-[11px] text-muted font-medium font-mono">{card.id ? `#${card.id}` : '新卡片'}</span>
        {carousel && total > 1 && canEdit && (
          <div className="ml-auto flex gap-1">
            <MiniBtn title="上移（轮播顺序）" disabled={index === 0} onClick={() => onMove(-1)}>
              <ChevronUpIcon className="w-[14px] h-[14px]" />
            </MiniBtn>
            <MiniBtn title="下移" disabled={index === total - 1} onClick={() => onMove(1)}>
              <ChevronDownIcon className="w-[14px] h-[14px]" />
            </MiniBtn>
            <MiniBtn title="删除卡片" danger onClick={onRemove}>
              <TrashIcon className="w-[13px] h-[13px]" />
            </MiniBtn>
          </div>
        )}
      </div>
      <div className={cn('grid gap-[14px] p-3', hasImg ? 'grid-cols-[auto_minmax(0,1fr)]' : 'grid-cols-1')}>
        {hasImg && spec && (
          <div>
            <ImageUpload
              src={card.imageUrl}
              spec={spec}
              canEdit={canEdit}
              width={upW}
              onNatural={(w, h) => setNatural({ w, h })}
              onUploaded={({ url, meta }) => {
                setNatural(null);
                onChange({ imageUrl: url, meta });
              }}
            />
          </div>
        )}
        <div className="min-w-0">
          {hasImg && spec && <UploadInfo spec={spec} url={card.imageUrl} meta={card.meta} natural={natural} />}

          {pos === 'P5' && (
            <div className="mb-[13px]">
              <Lbl req hint="纯文字单行，适合维护公告 / 系统提示，不放营销内容">通栏文案</Lbl>
              <CountInput value={card.title} onChange={(v) => onChange({ title: v })} placeholder="Scheduled maintenance Oct 15, 02:00–04:00 (GMT+8)" hard={64} disabled={!canEdit} />
            </div>
          )}
          {pos === 'P3' && (
            <div className="mb-[13px]">
              <Lbl req hint="缩略图右侧单行展示">一行文案</Lbl>
              <CountInput value={card.title} onChange={(v) => onChange({ title: v })} placeholder="Weekend rebate 10% · auto credited Monday" hard={64} disabled={!canEdit} />
            </div>
          )}
          {pos === 'P8' && (
            <div className="mb-[13px]">
              <Lbl req hint={`建议 ≤ ${TAB_TEXT_SOFT_MAX} 字，最多 ${TAB_TEXT_HARD_MAX} 字`}>便条文案</Lbl>
              <CountInput value={card.title} onChange={(v) => onChange({ title: v })} placeholder="签到领钱" soft={TAB_TEXT_SOFT_MAX} hard={TAB_TEXT_HARD_MAX} disabled={!canEdit} />
              {charLen(card.title) > TAB_TEXT_HARD_MAX ? (
                <div className="mt-1 text-[12px] text-down">超过 {TAB_TEXT_HARD_MAX} 字，无法保存</div>
              ) : charLen(card.title) > TAB_TEXT_SOFT_MAX ? (
                <SubWarn>超过建议的 {TAB_TEXT_SOFT_MAX} 字，便条宽度有限，过长会被截断</SubWarn>
              ) : null}
            </div>
          )}

          <div className="mb-[13px]">
            <Lbl hint="站内路径（/promo/618，仅 WebView 方式）运行时拼当前域名，不写死域名；也可 https:// 或 market://">跳转链接</Lbl>
            <CountInput mono value={card.linkUrl} onChange={(v) => onChange({ linkUrl: v })} placeholder="/promo/618 或 https://…" disabled={!canEdit} />
          </div>
          {caps.cta && (
            <>
              <div className="mb-[13px]">
                <Lbl hint="留空则整图可点">按钮文案</Lbl>
                <CountInput value={card.buttonText} onChange={(v) => onChange({ buttonText: v })} placeholder="Join now" hard={32} disabled={!canEdit} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Lbl req={pos === 'P4' && !card.imageUrl} hint={pos === 'P4' ? '无图时必填' : '可选'}>标题</Lbl>
                  <CountInput value={card.title} onChange={(v) => onChange({ title: v })} hard={64} disabled={!canEdit} />
                </div>
                <div>
                  <Lbl hint="可选">描述</Lbl>
                  <CountInput value={card.description} onChange={(v) => onChange({ description: v })} hard={255} disabled={!canEdit} />
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function MiniBtn({ children, title, danger, disabled, onClick }: { children: ReactNode; title: string; danger?: boolean; disabled?: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'grid place-items-center w-[26px] h-[26px] rounded-[7px] border border-line bg-panel text-ink-2 transition hover:bg-bg hover:text-ink disabled:opacity-40 disabled:cursor-not-allowed',
        danger && 'hover:!text-down hover:!border-[#fecaca] hover:!bg-[#fef2f2]',
      )}
    >
      {children}
    </button>
  );
}

export function PopupMaterial({
  form,
  setForm,
  canEdit,
}: {
  form: PopupForm;
  /** 函数式更新：异步上传回调里不能用渲染时的 form 快照，否则会回滚期间的修改。 */
  setForm: (updater: (f: PopupForm) => PopupForm) => void;
  canEdit: boolean;
}) {
  const caps = POSITION_CAPS[form.position];
  const carousel = caps.maxCards > 1;
  const shown = carousel ? form.cards : form.cards.slice(0, 1);
  const n = shown.length;
  const full = n >= MAX_CAROUSEL_CARDS;

  const patchCard = (uid: string, patch: Partial<FormCard>) =>
    setForm((f) => ({ ...f, cards: f.cards.map((c) => (c.uid === uid ? { ...c, ...patch } : c)) }));
  const move = (i: number, dir: -1 | 1) =>
    setForm((f) => {
      const cards = f.cards.slice();
      const j = i + dir;
      if (j < 0 || j >= cards.length) return f;
      [cards[i], cards[j]] = [cards[j], cards[i]];
      return { ...f, cards };
    });

  return (
    <div className="section-card">
      <h3 className="flex items-center gap-2 flex-wrap text-[13px] font-bold text-ink mb-[14px]">
        <span className="grid place-items-center w-5 h-5 rounded-md bg-brand text-white text-[11px] font-extrabold">2</span>
        素材
        {carousel ? (
          <>
            <span className={cn('font-mono text-[11.5px] font-extrabold px-2 py-px rounded-full', full ? 'bg-[#fee2e2] text-[#b91c1c]' : 'bg-[#eef2ff] text-brand-ink')}>
              {n}/{MAX_CAROUSEL_CARDS}
            </span>
            <span className="font-normal text-muted text-[11.5px]">支持轮播 · 上移 / 下移即轮播顺序 · 单张时隐藏圆点、禁用滑动</span>
          </>
        ) : (
          <span className="font-normal text-muted text-[11.5px]">· 该位置固定 1 张{form.position === 'P5' ? '（纯文字，无图）' : ''}</span>
        )}
      </h3>

      {!carousel && form.cards.length > 1 && (
        <SubWarn>当前位置不支持轮播，仅保留第 1 张；保存时丢弃其余 {form.cards.length - 1} 张</SubWarn>
      )}

      {shown.map((c, i) => (
        <CardEditor
          key={c.uid}
          form={form}
          card={c}
          index={i}
          total={n}
          canEdit={canEdit}
          onChange={(patch) => patchCard(c.uid, patch)}
          onMove={(dir) => move(i, dir)}
          onRemove={() => setForm((f) => ({ ...f, cards: f.cards.filter((x) => x.uid !== c.uid) }))}
        />
      ))}

      {carousel && canEdit && (
        <div className="flex items-center gap-[10px] flex-wrap">
          <Button className="!px-[10px] !py-[5px] text-[12px]" disabled={full} onClick={() => setForm((f) => ({ ...f, cards: [...f.cards, blankCard()] }))}>
            <PlusIcon />
            添加卡片
          </Button>
          <span className={cn('text-[11.5px]', full ? 'text-down font-semibold' : 'text-muted')}>最多 {MAX_CAROUSEL_CARDS} 张，超出不允许保存</span>
        </div>
      )}

      {form.position !== 'P5' && (
        <div className="mt-3">
          <Note>
            <InfoIcon className="w-[17px] h-[17px] text-brand flex-none mt-px" />
            <div>
              <b className="text-ink">换图必须换 URL：</b>
              上传即生成新文件名（内容 hash + 时间戳），不允许覆盖同名文件。客户端磁盘缓存以完整 URL 为 key，URL 变了才会重新下载；同 URL 换图会被 CDN + 客户端缓存挡住数天。
            </div>
          </Note>
        </div>
      )}
    </div>
  );
}
