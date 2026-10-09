import { create } from 'zustand';

/**
 * 轻量全局 toast（原型 .toast：底部居中深色条）。项目此前没有统一 toast，
 * 弹窗模块需要「后端 400 的 message 直接 toast」「403 提示仅全量数据权限」，故新增。
 */
export type ToastKind = 'ok' | 'err';
export interface ToastItem {
  id: number;
  kind: ToastKind;
  message: string;
}

interface ToastState {
  items: ToastItem[];
  push: (message: string, kind?: ToastKind) => void;
  dismiss: (id: number) => void;
}

let seq = 0;

export const useToastStore = create<ToastState>((set, get) => ({
  items: [],
  push: (message, kind = 'ok') => {
    seq += 1;
    const id = seq;
    set({ items: [...get().items.slice(-2), { id, kind, message }] });
    // 错误停留更久，便于读完后端返回的校验文案
    window.setTimeout(() => get().dismiss(id), kind === 'err' ? 6000 : 3500);
  },
  dismiss: (id) => set({ items: get().items.filter((t) => t.id !== id) }),
}));

export const toast = {
  ok: (m: string) => useToastStore.getState().push(m, 'ok'),
  err: (m: string) => useToastStore.getState().push(m, 'err'),
};
