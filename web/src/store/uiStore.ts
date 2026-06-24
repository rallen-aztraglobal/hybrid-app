import { create } from 'zustand';
import type { BrandCode } from '@/lib/types';

/**
 * 跨页面 UI 本地状态：当前选中的大渠道（渠道管理 / 打包中心共享）、
 * 顶栏全局搜索词、渠道抽屉开合与编辑目标。
 */
interface UiState {
  /** 当前大渠道 Tab（渠道管理 & 打包中心共享选择） */
  currentBrand: BrandCode;
  setCurrentBrand: (b: BrandCode) => void;

  /** 顶栏全局搜索（驱动渠道管理过滤） */
  search: string;
  setSearch: (s: string) => void;

  /** 渠道抽屉：null = 关闭；'new' = 新增；string = 编辑该 channel id */
  drawerTarget: 'new' | { id: string } | null;
  openCreate: () => void;
  openEdit: (id: string) => void;
  closeDrawer: () => void;
}

export const useUiStore = create<UiState>((set) => ({
  currentBrand: 'ap',
  setCurrentBrand: (b) => set({ currentBrand: b }),

  search: '',
  setSearch: (s) => set({ search: s }),

  drawerTarget: null,
  openCreate: () => set({ drawerTarget: 'new' }),
  openEdit: (id) => set({ drawerTarget: { id } }),
  closeDrawer: () => set({ drawerTarget: null }),
}));
