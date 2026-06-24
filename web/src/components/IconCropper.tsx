/**
 * 图标方形裁剪器 —— react-easy-crop（ADR-0005「前端裁剪」）。
 * 用户拖入任意尺寸主图 → 1:1 裁剪框 + 缩放 → 导出 1024² 主图（dataURL），上传后端 fan-out。
 */
import { useCallback, useState } from 'react';
import Cropper, { type Area } from 'react-easy-crop';
import { exportCroppedSquare } from '@/lib/icon';
import { Button } from './ui';
import { CloseIcon } from './icons';

export function IconCropper({
  srcDataUrl,
  onCancel,
  onDone,
}: {
  srcDataUrl: string;
  onCancel: () => void;
  onDone: (square1024: string) => void;
}) {
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [area, setArea] = useState<Area | null>(null);
  const [busy, setBusy] = useState(false);

  const onComplete = useCallback((_: Area, areaPixels: Area) => {
    setArea(areaPixels);
  }, []);

  async function confirm() {
    if (!area) return;
    setBusy(true);
    try {
      const out = await exportCroppedSquare(srcDataUrl, area, 1024);
      onDone(out);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-[60] grid place-items-center p-4" style={{ background: 'rgba(15,23,42,.6)' }}>
      <div className="w-full max-w-[460px] bg-panel rounded-[16px] overflow-hidden shadow-lg2">
        <div className="flex items-center px-5 py-4 border-b border-line">
          <div className="flex-1">
            <div className="text-[15px] font-bold">裁剪主图</div>
            <div className="text-[12px] text-muted mt-0.5">拖动定位、滚轮/滑块缩放，导出 1024 × 1024</div>
          </div>
          <button
            onClick={onCancel}
            className="grid place-items-center w-9 h-9 rounded-[10px] border border-line text-ink-2 hover:bg-bg"
          >
            <CloseIcon className="w-[18px] h-[18px]" />
          </button>
        </div>

        <div className="relative h-[320px] bg-[#0c1426]">
          <Cropper
            image={srcDataUrl}
            crop={crop}
            zoom={zoom}
            aspect={1}
            cropShape="rect"
            showGrid
            onCropChange={setCrop}
            onZoomChange={setZoom}
            onCropComplete={onComplete}
          />
        </div>

        <div className="px-5 py-4">
          <label className="block text-[12px] font-semibold text-ink-2 mb-2">缩放</label>
          <input
            type="range"
            min={1}
            max={3}
            step={0.01}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
            className="w-full accent-brand"
          />
        </div>

        <div className="flex justify-end gap-[10px] px-5 py-4 border-t border-line bg-panel">
          <Button onClick={onCancel}>取消</Button>
          <Button variant="primary" onClick={confirm} disabled={busy || !area}>
            {busy ? '生成中…' : '确认裁剪'}
          </Button>
        </div>
      </div>
    </div>
  );
}
