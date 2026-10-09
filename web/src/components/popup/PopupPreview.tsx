/**
 * 右侧手机实时预览（原型 .phone + 假 H5）。用真实上传的图片 URL，按位置画出形态：
 *  P1/P4/P6 居中卡片 + 右上关闭按钮（小圆圈 + 细 X）；P7 全屏；P2 悬浮球；P3 底部横幅；
 *  P5 顶部通栏；P8 便条。有便条态的位置可切「弹窗展示态 / 关闭后·便条态」。
 * 只做静态形态（不做原型里的倒计时 / 轮播 / 拖动动画）；多卡位置的圆点可点切换。
 */
import { useEffect, useState, type CSSProperties, type ReactNode } from 'react';
import { effectiveCards, type PopupForm } from '@/lib/popupForm';
import { POSITION_CAPS, isForcedMode } from '@/lib/popupMeta';
import { cn } from '@/lib/cn';
import { EyeIcon } from '../icons';
import { Seg } from './bits';

interface Theme {
  bg: string;
  card: string;
  ink: string;
  mut: string;
  acc: string;
  line: string;
  nav: string;
}
const THEMES: Record<string, Theme> = {
  gp: { bg: '#120c24', card: '#1f1638', ink: '#efeaff', mut: '#8f86b3', acc: '#8b5cf6', line: 'rgba(255,255,255,.07)', nav: '#170f2e' },
  ap: { bg: '#fff7ef', card: '#ffe9d5', ink: '#3b1d08', mut: '#b58a66', acc: '#f97316', line: 'rgba(0,0,0,.06)', nav: '#ffffff' },
  bp: { bg: '#f2f6ff', card: '#dfe9ff', ink: '#0b1f4d', mut: '#7d8fb8', acc: '#2563eb', line: 'rgba(0,0,0,.06)', nav: '#ffffff' },
};
const TILE_G = ['#f43f5e,#f59e0b', '#8b5cf6,#ec4899', '#0ea5e9,#22d3ee', '#10b981,#a3e635', '#f97316,#facc15', '#6366f1,#22d3ee', '#ef4444,#f97316', '#14b8a6,#3b82f6', '#a855f7,#6366f1'];
const H5_NAME: Record<string, string> = { ap: 'ArenaPlus', bp: 'BingoPlus', gp: 'GameZone' };

function FakeH5({ t, name }: { t: Theme; name: string }) {
  return (
    <div className="absolute inset-0 flex flex-col text-[10px]" style={{ color: t.ink }}>
      <div className="h-[26px] flex-none flex items-center justify-between px-5 text-[10.5px]">
        <b>9:41</b>
        <span className="opacity-70">▮▮▮ ◔ ▭</span>
      </div>
      <div className="h-10 flex-none flex items-center gap-[7px] px-3 text-[11.5px]">
        <span className="grid place-items-center w-6 h-6 rounded-[7px] text-white font-black" style={{ background: t.acc }}>{name[0]}</span>
        <b>{name}</b>
        <span className="ml-auto text-[10px] font-semibold" style={{ color: t.mut }}>₱ 1,250.00</span>
        <span className="rounded-xl px-[10px] py-1 text-white font-bold text-[10px]" style={{ background: t.acc }}>Deposit</span>
      </div>
      <div
        className="mx-3 mt-1 h-24 flex-none rounded-xl p-[14px] flex flex-col justify-end text-white"
        style={{ background: `linear-gradient(120deg,${t.acc},#ec4899 70%,#f59e0b)` }}
      >
        <small className="text-[8.5px] tracking-[.12em] opacity-85 font-bold">WEEKLY JACKPOT</small>
        <b className="text-[19px] font-black">₱ 2,480,000</b>
      </div>
      <div className="flex gap-[6px] px-3 pt-[10px] pb-1 flex-none">
        {['Hot', 'Tongits', 'Slots', 'Bingo', 'Live'].map((c, i) => (
          <i key={c} className="not-italic text-[9.5px] px-[9px] py-[3px] rounded-[10px] font-semibold" style={i === 0 ? { background: t.acc, color: '#fff' } : { background: t.card, color: t.mut }}>
            {c}
          </i>
        ))}
      </div>
      <div className="grid grid-cols-3 gap-[7px] px-3 py-2 flex-1 overflow-hidden content-start">
        {TILE_G.map((g) => (
          <i key={g} className="rounded-[10px] opacity-85" style={{ aspectRatio: '1/1.05', background: `linear-gradient(140deg,${g})` }} />
        ))}
      </div>
      <div className="h-[50px] flex-none flex justify-around items-center border-t" style={{ borderColor: t.line, background: t.nav }}>
        {['Home', 'Games', 'Promo', 'Wallet', 'Me'].map((n, i) => (
          <span key={n} className="flex flex-col items-center gap-0.5" style={{ color: i === 0 ? t.acc : t.mut }}>
            <span className="w-[14px] h-[14px] rounded-[4px] border-[1.5px] border-current" />
            <em className="not-italic text-[8.5px] font-semibold">{n}</em>
          </span>
        ))}
      </div>
      <div className="h-[14px] flex-none grid place-items-center" style={{ background: t.nav }}>
        <i className="w-[90px] h-[3.5px] rounded-sm opacity-35" style={{ background: t.ink }} />
      </div>
    </div>
  );
}

/** 图片 / 占位。 */
function Img({ src, className, style, label = '待上传' }: { src?: string; className?: string; style?: CSSProperties; label?: string }) {
  return src ? (
    <img src={src} alt="" className={cn('object-cover', className)} style={style} draggable={false} />
  ) : (
    <div
      className={cn('grid place-items-center text-[10px] font-semibold text-white/80', className)}
      style={{ background: 'linear-gradient(160deg,#475569,#94a3b8)', ...style }}
    >
      {label}
    </div>
  );
}

/** 关闭按钮：小圆圈 + 细 X（原型最终版）；倒计时期间在同一位置显示数字。 */
function XBtn({ countdown, style }: { countdown?: boolean; style?: CSSProperties }) {
  return (
    <div
      className="absolute grid place-items-center w-7 h-7 rounded-full text-white z-[4]"
      style={{
        border: `1.5px solid ${countdown ? 'rgba(255,255,255,.4)' : 'rgba(255,255,255,.85)'}`,
        background: 'rgba(0,0,0,.28)',
        transform: countdown ? 'scale(.8)' : undefined,
        ...style,
      }}
    >
      {countdown ? (
        <b className="text-[13px] font-extrabold">3</b>
      ) : (
        <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      )}
    </div>
  );
}

function Dots({ n, i, onPick, className }: { n: number; i: number; onPick: (k: number) => void; className?: string }) {
  if (n <= 1) return null;
  return (
    <div className={cn('flex justify-center gap-[5px]', className)}>
      {Array.from({ length: n }, (_, k) => (
        <button
          key={k}
          type="button"
          onClick={() => onPick(k)}
          className="h-[6px] rounded-[3px] transition-all"
          style={{ width: k === i ? 16 : 6, background: k === i ? '#fff' : 'rgba(255,255,255,.4)' }}
        />
      ))}
    </div>
  );
}

function Cta({ text, style }: { text: string; style?: CSSProperties }) {
  if (!text) return null;
  return (
    <span
      className="absolute left-1/2 -translate-x-1/2 whitespace-nowrap font-extrabold text-[14px] px-7 py-[10px] rounded-[30px] z-[3]"
      style={{
        background: 'linear-gradient(180deg,#ffe28a,#f5b800)',
        color: '#5b3400',
        boxShadow: '0 6px 14px -4px rgba(0,0,0,.45), inset 0 -2px 0 rgba(0,0,0,.12)',
        ...style,
      }}
    >
      {text}
    </span>
  );
}

/** 便条：右侧贴边扁条，图标 + 文案。 */
function TabNote({ icon, text }: { icon?: string; text: string }) {
  return (
    <div
      className="absolute right-0 h-[23px] w-[84px] flex items-center gap-[5px] pl-[3px] pr-2 text-white text-[10.5px] font-bold overflow-hidden whitespace-nowrap z-[6]"
      style={{
        bottom: '25%',
        borderRadius: '12px 0 0 12px',
        background: 'rgba(15,23,42,.72)',
        backdropFilter: 'blur(4px)',
        opacity: 0.94,
        boxShadow: '-2px 4px 12px -4px rgba(0,0,0,.45)',
      }}
    >
      <Img src={icon} className="w-[17px] h-[17px] rounded-full flex-none" label="" />
      <span className="truncate">{text || '便条文案'}</span>
    </div>
  );
}

export function PopupPreview({ form }: { form: PopupForm }) {
  const caps = POSITION_CAPS[form.position];
  const forced = isForcedMode(form.position, form.closable);
  const cards = effectiveCards(form);
  const canTab = caps.tab !== 'no' && form.tabEnabled && !forced;
  const [pv, setPv] = useState<'show' | 'tab'>('show');
  const [idx, setIdx] = useState(0);
  const i = Math.min(idx, Math.max(0, cards.length - 1));
  const c = cards[i] ?? cards[0];

  useEffect(() => {
    if (!canTab) setPv('show');
  }, [canTab]);

  const brand = form.brandCodes[0] ?? 'gp';
  const theme = THEMES[brand] ?? THEMES.gp;
  const tabIcon = form.tabIconUrl || cards[0]?.imageUrl;

  let overlay: ReactNode = null;
  if (pv === 'tab' && canTab) {
    overlay = <TabNote icon={tabIcon} text={form.tabText} />;
  } else {
    switch (form.position) {
      case 'P1':
      case 'P4':
      case 'P6': {
        const noImg = !c?.imageUrl;
        overlay = (
          <>
            <div className="absolute inset-0 z-10" style={{ background: 'rgba(5,8,20,.62)' }} />
            <div className="absolute left-1/2 top-1/2 w-[244px] z-[11]" style={{ transform: 'translate(-50%,-50%)' }}>
              {!forced && <XBtn countdown={form.position === 'P1' && form.countdown} style={{ right: 0, top: -38 }} />}
              <div
                className="relative w-full overflow-hidden"
                style={{ aspectRatio: '3/4', borderRadius: 16, boxShadow: '0 20px 40px -12px rgba(0,0,0,.6)', background: noImg ? '#fff' : undefined }}
              >
                {!noImg && <Img src={c?.imageUrl} className="w-full h-full" />}
                {(c?.title || c?.description) && (
                  <div
                    className={cn('absolute left-0 right-0 px-4 text-center', noImg ? 'top-1/2 -translate-y-1/2 text-ink' : 'bottom-14 text-white')}
                    style={noImg ? undefined : { textShadow: '0 1px 6px rgba(0,0,0,.6)' }}
                  >
                    {c.title && <div className="text-[15px] font-extrabold leading-tight">{c.title}</div>}
                    {c.description && <div className="text-[11px] mt-1 opacity-90">{c.description}</div>}
                  </div>
                )}
                <Cta text={c?.buttonText ?? ''} style={{ bottom: 14 }} />
              </div>
              <Dots n={cards.length} i={i} onPick={setIdx} className="mt-3" />
            </div>
            {forced && (
              <span
                className="absolute left-1/2 -translate-x-1/2 z-30 text-[9.5px] font-semibold whitespace-nowrap rounded-md px-[6px] py-0.5 pointer-events-none"
                style={{ top: 96, color: '#b91c1c', background: 'rgba(254,242,242,.97)', border: '1px dashed #f87171' }}
              >
                强制模式：无 X · 遮罩 / 返回键均不关
              </span>
            )}
          </>
        );
        break;
      }
      case 'P7':
        overlay = (
          <div className="absolute inset-0 z-[11] bg-black">
            <Img src={c?.imageUrl} className="w-full h-full" />
            <XBtn countdown={form.countdown} style={{ top: 34, right: 12 }} />
            <Cta text={c?.buttonText ?? ''} style={{ bottom: 62, fontSize: 13 }} />
            <Dots n={cards.length} i={i} onPick={setIdx} className="absolute bottom-9 left-0 right-0" />
          </div>
        );
        break;
      case 'P2':
        overlay = (
          <div className="absolute z-[5] w-[46px] h-[46px]" style={{ right: 6, top: '56%' }}>
            <Img src={c?.imageUrl} className="absolute inset-0 w-full h-full rounded-full border-2 border-white" style={{ boxShadow: '0 8px 18px -6px rgba(0,0,0,.55)' }} />
            {form.badge && (
              <i className="absolute -top-0.5 -right-0.5 grid place-items-center min-w-[16px] h-4 rounded-lg bg-[#ef4444] text-white text-[9px] font-extrabold border-[1.5px] border-white not-italic">1</i>
            )}
            <span className="absolute -left-[5px] -top-[5px] w-4 h-4 rounded-full grid place-items-center text-white" style={{ background: 'rgba(15,23,42,.75)' }}>
              <svg viewBox="0 0 24 24" width="8" height="8" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
            </span>
          </div>
        );
        break;
      case 'P3':
        overlay = (
          <div
            className="absolute left-[10px] right-[10px] h-[50px] rounded-[13px] z-[5] text-[#0f172a]"
            style={{ bottom: 76, background: 'rgba(255,255,255,.97)', boxShadow: '0 10px 24px -10px rgba(0,0,0,.5)' }}
          >
            <div className="absolute left-[6px] right-[34px] top-[6px] bottom-[6px] flex gap-[9px] items-center">
              <Img src={c?.imageUrl} className="w-[38px] h-[38px] rounded-[9px] flex-none" label="" />
              <span className="flex-1 min-w-0 text-[11px] font-bold truncate">{c?.title || '…'}</span>
            </div>
            <span className="absolute right-[6px] top-1/2 -translate-y-1/2 w-6 h-6 grid place-items-center text-[#94a3b8]">
              <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
            </span>
            {cards.length > 1 && (
              <span className="absolute bottom-1 left-1/2 -translate-x-1/2 flex gap-[3px]">
                {cards.map((_, k) => (
                  <button key={k} type="button" onClick={() => setIdx(k)} className="h-1 rounded-sm" style={{ width: k === i ? 10 : 4, background: k === i ? '#6366f1' : '#cbd5e1' }} />
                ))}
              </span>
            )}
          </div>
        );
        break;
      case 'P5':
        overlay = (
          <div
            className="absolute top-[26px] left-0 right-0 h-[30px] flex items-center gap-[7px] pl-[11px] pr-1 text-[10.5px] font-semibold z-[5]"
            style={{ background: '#fffbeb', color: '#92400e', borderBottom: '1px solid #fde68a' }}
          >
            <span className="flex-1 truncate">📢 {c?.title || '…'}</span>
            <span className="w-[26px] h-[26px] grid place-items-center">
              <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round"><path d="M18 6 6 18M6 6l12 12" /></svg>
            </span>
          </div>
        );
        break;
      case 'P8':
        overlay = <TabNote icon={c?.imageUrl} text={c?.title ?? ''} />;
        break;
    }
  }

  return (
    <div className="flex flex-col items-center gap-3">
      <div className="self-stretch flex items-center gap-2 text-[12.5px] font-bold text-ink-2">
        <EyeIcon className="w-[15px] h-[15px]" />
        实时预览 · {form.position} {caps.name}
        <span className="ml-auto inline-flex items-center gap-[5px] text-[11px] text-[#15803d] font-semibold">
          <i className="w-[7px] h-[7px] rounded-full bg-ok inline-block" />
          随表单更新
        </span>
      </div>
      <div
        className="relative flex-none w-[300px] h-[620px] rounded-[44px] p-[9px]"
        style={{ background: '#0b0f1a', boxShadow: '0 30px 60px -25px rgba(15,23,42,.55), inset 0 0 0 2px #2a3245' }}
      >
        <div className="relative w-full h-full rounded-[36px] overflow-hidden select-none" style={{ background: theme.bg }}>
          <FakeH5 t={theme} name={H5_NAME[brand] ?? 'GameZone'} />
          {overlay}
          <i className="absolute top-2 left-1/2 -ml-[5.5px] w-[11px] h-[11px] rounded-full z-40" style={{ background: '#05070d' }} />
        </div>
      </div>
      {canTab && (
        <Seg
          value={pv}
          onChange={setPv}
          options={[
            { value: 'show', label: '弹窗展示态' },
            { value: 'tab', label: '关闭后 · 便条态' },
          ]}
        />
      )}
      <div className="text-[11px] text-muted text-center leading-normal max-w-[300px]">
        示意形态，以客户端实际渲染为准；主题取「定向 › 品牌」首个。多卡位置可点圆点切换。倒计时 / 轮播动画未复刻。
      </div>
    </div>
  );
}
