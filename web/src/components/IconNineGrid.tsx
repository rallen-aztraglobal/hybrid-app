/**
 * Xcode 式图标九宫格 —— 复刻原型「App 图标」区（ADR-0005 / 03 §4）：
 * 1) 大 dropzone 拖入/点击上传主图 → 非方形进裁剪 → 1024² 主图；
 * 2) 三行槽位（方形 / 圆形 / 自适应前景），各 density 标注像素与 dpi，自动填入主图预览；
 * 3) 任意单槽支持「单独覆盖」该档（高级精修），覆盖图独立于主图。
 *
 * 客户端预览用 canvas 即时渲染（圆形遮罩 / 安全区留边）；最终多密度产物由后端 imaging fan-out。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  DENSITY_ORDER,
  FOREGROUND_PX,
  FOREGROUND_SAFE_RATIO,
  MASTER_ICON_MIN_PX,
  SQUARE_PX,
} from '@/lib/brands';
import type { IconVariant } from '@/lib/types';
import { loadImageFile, renderVariantPreview } from '@/lib/icon';
import { cn } from '@/lib/cn';
import { UploadIcon } from './icons';
import { IconCropper } from './IconCropper';

export interface IconState {
  /** 裁剪后的 1024² 主图（dataURL）；空 = 未上传 */
  master: string | null;
  /** 单槽覆盖：key = `${variant}:${dpi}` → dataURL */
  overrides: Record<string, string>;
}

export function emptyIconState(master: string | null = null): IconState {
  return { master, overrides: {} };
}

const VARIANTS: { variant: IconVariant; label: string; file: string; px: Record<string, number> }[] = [
  { variant: 'square', label: '方形 ic_launcher', file: 'ic_launcher.png', px: SQUARE_PX },
  { variant: 'round', label: '圆形 ic_launcher_round', file: 'ic_launcher_round.png', px: SQUARE_PX },
  {
    variant: 'foreground',
    label: '自适应前景 ic_launcher_foreground',
    file: 'ic_launcher_foreground.png',
    px: FOREGROUND_PX,
  },
];

export function IconNineGrid({
  value,
  onChange,
  accentHex,
}: {
  value: IconState;
  onChange: (next: IconState) => void;
  accentHex: string;
}) {
  const [cropSrc, setCropSrc] = useState<string | null>(null);
  const [warn, setWarn] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  async function handleFile(file: File) {
    setWarn(null);
    try {
      const img = await loadImageFile(file);
      if (img.tooSmall) {
        setWarn(`主图偏小（${img.width}×${img.height}），建议 ≥ ${MASTER_ICON_MIN_PX}² 以免下采样模糊`);
      }
      if (!img.isSquare) {
        // 非方形 → 进入裁剪
        setCropSrc(img.dataUrl);
      } else {
        // 方形：仍走裁剪器让用户确认/微调，导出标准 1024²
        setCropSrc(img.dataUrl);
      }
    } catch (e) {
      setWarn(e instanceof Error ? e.message : '图片读取失败');
    }
  }

  return (
    <div>
      <div
        onClick={() => fileRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragOver(false);
          const f = e.dataTransfer.files?.[0];
          if (f) void handleFile(f);
        }}
        className={cn(
          'rounded-[12px] p-[22px] text-center cursor-pointer transition border-2 border-dashed',
          dragOver ? 'border-brand bg-[rgba(99,102,241,.06)]' : 'border-[#c7d2e6]',
        )}
        style={
          dragOver
            ? undefined
            : {
                background:
                  'repeating-linear-gradient(45deg,#fbfcfe,#fbfcfe 12px,#f6f8fc 12px,#f6f8fc 24px)',
              }
        }
      >
        <div
          className="grid place-items-center w-[46px] h-[46px] rounded-[13px] mx-auto mb-[10px] text-white"
          style={{
            background: 'linear-gradient(135deg,var(--brand),#8b5cf6)',
            boxShadow: '0 8px 20px -8px rgba(99,102,241,.7)',
          }}
        >
          <UploadIcon className="w-[22px] h-[22px]" />
        </div>
        <div className="font-bold text-[13.5px]">
          {value.master ? '✓ 已生成全部尺寸（5 档 × 方形/圆形/自适应）' : '拖入主图 1024 × 1024 或点击上传'}
        </div>
        <div className="text-[12px] text-muted mt-[3px]">
          支持 PNG / JPG，自动方形裁剪并下采样到各 density
        </div>
        <input
          ref={fileRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          hidden
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void handleFile(f);
            e.target.value = '';
          }}
        />
      </div>

      {warn && <div className="mt-2 text-[12px] text-[#92681a] bg-[#fef3c7] rounded-lg px-3 py-2">{warn}</div>}

      {VARIANTS.map((v) => (
        <div key={v.variant}>
          <div className="flex items-center gap-[7px] text-[11.5px] font-bold text-ink-2 mt-[14px] mb-[9px]">
            {v.label}
            <span className="flex-1 h-px bg-line-2" />
          </div>
          <div className="flex gap-[10px] flex-wrap">
            {DENSITY_ORDER.map((dpi) => (
              <IconSlot
                key={`${v.variant}:${dpi}`}
                variant={v.variant}
                dpi={dpi}
                px={v.px[dpi]}
                master={value.master}
                accentHex={accentHex}
                override={value.overrides[`${v.variant}:${dpi}`]}
                onOverride={(dataUrl) =>
                  onChange({
                    ...value,
                    overrides: { ...value.overrides, [`${v.variant}:${dpi}`]: dataUrl },
                  })
                }
                onClearOverride={() => {
                  const next = { ...value.overrides };
                  delete next[`${v.variant}:${dpi}`];
                  onChange({ ...value, overrides: next });
                }}
              />
            ))}
          </div>
        </div>
      ))}

      {cropSrc && (
        <IconCropper
          srcDataUrl={cropSrc}
          onCancel={() => setCropSrc(null)}
          onDone={(square) => {
            setCropSrc(null);
            onChange({ ...value, master: square });
          }}
        />
      )}
    </div>
  );
}

function IconSlot({
  variant,
  dpi,
  px,
  master,
  accentHex,
  override,
  onOverride,
  onClearOverride,
}: {
  variant: IconVariant;
  dpi: string;
  px: number;
  master: string | null;
  accentHex: string;
  override?: string;
  onOverride: (dataUrl: string) => void;
  onClearOverride: () => void;
}) {
  const round = variant === 'round';
  const fileRef = useRef<HTMLInputElement>(null);
  const [preview, setPreview] = useState<string | null>(null);

  // 主图变化时即时渲染该槽预览（圆形/安全区由 variant 决定）
  useEffect(() => {
    let alive = true;
    if (override) {
      setPreview(override);
      return;
    }
    if (!master) {
      setPreview(null);
      return;
    }
    void renderVariantPreview(master, variant, 74, FOREGROUND_SAFE_RATIO).then((url) => {
      if (alive) setPreview(url);
    });
    return () => {
      alive = false;
    };
  }, [master, override, variant]);

  const hasContent = !!preview;
  const boxStyle = useMemo(
    () => ({ borderRadius: round ? '50%' : 12 }) as React.CSSProperties,
    [round],
  );

  return (
    <div className="w-[74px] text-center group/slot">
      <div
        onClick={() => fileRef.current?.click()}
        title="点击单独覆盖该档"
        className={cn(
          'relative grid place-items-center w-[74px] h-[74px] overflow-hidden cursor-pointer transition',
          hasContent ? 'border-[1.5px] border-solid border-transparent' : 'border-[1.5px] border-dashed border-[#cdd7e6] bg-panel-2',
        )}
        style={boxStyle}
      >
        {hasContent ? (
          <img src={preview!} alt={`${variant}-${dpi}`} className="w-full h-full object-contain" />
        ) : (
          <span className="text-muted text-[11px]">{px}</span>
        )}
        {override && (
          <span
            className="absolute top-0.5 right-0.5 text-[9px] px-1 rounded bg-brand text-white"
            title="已单独覆盖，点击右下角×还原"
          >
            改
          </span>
        )}
        {/* 覆盖输入 */}
        <input
          ref={fileRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          hidden
          onChange={async (e) => {
            const f = e.target.files?.[0];
            e.target.value = '';
            if (!f) return;
            try {
              const img = await loadImageFile(f);
              onOverride(img.dataUrl);
            } catch {
              /* ignore */
            }
          }}
        />
        {accentHex && null /* accentHex 预留：未来空槽用品牌色占位 */}
      </div>
      <div className="text-[11px] font-bold text-ink-2 mt-[6px]">{px}px</div>
      <div className="text-[10px] text-muted">
        {dpi}
        {override && (
          <button
            onClick={onClearOverride}
            className="ml-1 text-brand hover:underline"
            title="还原为主图生成"
          >
            还原
          </button>
        )}
      </div>
    </div>
  );
}
