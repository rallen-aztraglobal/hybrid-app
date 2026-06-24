import { MASTER_ICON_PX, MASTER_ICON_MIN_PX } from './brands';

/**
 * 前端图标处理工具（ADR-0005：前端裁剪、后端 fan-out）。
 * 这里只负责「读图 → 校验尺寸 → 把 react-easy-crop 的裁剪区域导出为 1024² 主图」。
 * 多密度 fan-out（圆形遮罩 / 安全区留边）由后端 imaging 完成。
 */

export interface CropArea {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface LoadedImage {
  dataUrl: string;
  width: number;
  height: number;
  /** 是否方形（非方形需进入裁剪） */
  isSquare: boolean;
  /** 是否过小（< 512，放大会模糊） */
  tooSmall: boolean;
}

/** 读取 File 为 dataURL 并解出尺寸 + 校验。 */
export function loadImageFile(file: File): Promise<LoadedImage> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error('读取文件失败'));
    reader.onload = () => {
      const dataUrl = reader.result as string;
      const img = new Image();
      img.onerror = () => reject(new Error('不是有效的图片'));
      img.onload = () => {
        resolve({
          dataUrl,
          width: img.width,
          height: img.height,
          isSquare: Math.abs(img.width - img.height) <= 1,
          tooSmall: Math.min(img.width, img.height) < MASTER_ICON_MIN_PX,
        });
      };
      img.src = dataUrl;
    };
    reader.readAsDataURL(file);
  });
}

/**
 * 按裁剪区域把源图导出为正方形主图（默认 1024²）。
 * croppedAreaPixels 来自 react-easy-crop 的 onCropComplete。
 */
export function exportCroppedSquare(
  srcDataUrl: string,
  croppedAreaPixels: CropArea,
  size = MASTER_ICON_PX,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onerror = () => reject(new Error('裁剪失败：图片加载错误'));
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = size;
      canvas.height = size;
      const ctx = canvas.getContext('2d');
      if (!ctx) return reject(new Error('Canvas 2D 不可用'));
      ctx.imageSmoothingQuality = 'high';
      ctx.drawImage(
        img,
        croppedAreaPixels.x,
        croppedAreaPixels.y,
        croppedAreaPixels.width,
        croppedAreaPixels.height,
        0,
        0,
        size,
        size,
      );
      resolve(canvas.toDataURL('image/png'));
    };
    img.src = srcDataUrl;
  });
}

/**
 * 客户端预览用：把方形主图按 variant 渲染成预览 dataURL。
 * - square：原样缩放
 * - round：圆形遮罩
 * - foreground：安全区内缩放居中（透明留边）
 * 仅用于九宫格所见即所得预览；最终产物以后端 fan-out 为准。
 */
export function renderVariantPreview(
  squareDataUrl: string,
  variant: 'square' | 'round' | 'foreground',
  px: number,
  safeRatio = 0.66,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onerror = () => reject(new Error('预览渲染失败'));
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = px;
      canvas.height = px;
      const ctx = canvas.getContext('2d');
      if (!ctx) return reject(new Error('Canvas 2D 不可用'));
      ctx.imageSmoothingQuality = 'high';

      if (variant === 'round') {
        ctx.save();
        ctx.beginPath();
        ctx.arc(px / 2, px / 2, px / 2, 0, Math.PI * 2);
        ctx.closePath();
        ctx.clip();
        ctx.drawImage(img, 0, 0, px, px);
        ctx.restore();
      } else if (variant === 'foreground') {
        const inner = Math.round(px * safeRatio);
        const off = Math.round((px - inner) / 2);
        ctx.clearRect(0, 0, px, px);
        ctx.drawImage(img, off, off, inner, inner);
      } else {
        ctx.drawImage(img, 0, 0, px, px);
      }
      resolve(canvas.toDataURL('image/png'));
    };
    img.src = squareDataUrl;
  });
}
