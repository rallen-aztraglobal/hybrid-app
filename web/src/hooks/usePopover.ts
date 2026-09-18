/**
 * 自绘弹层（DatePicker / TimePicker / DateTimePicker）通用开合逻辑：
 *  - 点击外部 / Esc 关闭；
 *  - 打开时按视口剩余空间自动翻转：右侧放不下就右对齐，下方放不下且上方放得下就向上弹。
 *    （推送页右栏只有 360px，日期+时间弹层 ~410px 宽，不翻转会被裁/撑出横向滚动。）
 */
import { useEffect, useLayoutEffect, useRef, useState } from 'react';

export interface PopoverPlacement {
  alignRight: boolean;
  up: boolean;
}

export function usePopover() {
  const [open, setOpen] = useState(false);
  const [placement, setPlacement] = useState<PopoverPlacement>({ alignRight: false, up: false });
  const wrapRef = useRef<HTMLDivElement>(null);
  const popRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // 首帧按默认位置（左对齐、向下）渲染后立即测量，paint 前修正，无闪烁。
  useLayoutEffect(() => {
    if (!open || !wrapRef.current || !popRef.current) return;
    const anchor = wrapRef.current.getBoundingClientRect();
    const pop = popRef.current.getBoundingClientRect();
    const gap = 8;
    const alignRight = anchor.left + pop.width > window.innerWidth - gap && anchor.right - pop.width >= gap;
    const up = anchor.bottom + pop.height + gap > window.innerHeight && anchor.top - pop.height - gap >= 0;
    setPlacement({ alignRight, up });
  }, [open]);

  /** 弹层定位 class（配合外层 relative 容器）。 */
  const popClass = [
    'absolute z-50',
    placement.alignRight ? 'right-0' : 'left-0',
    placement.up ? 'bottom-full mb-1' : 'top-full mt-1',
  ].join(' ');

  return { open, setOpen, wrapRef, popRef, popClass };
}
