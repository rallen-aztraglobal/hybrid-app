import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthUser } from '@/lib/types';
import { setToken } from '@/lib/api';

/**
 * 登录态（本地状态，Zustand + persist）。
 * token 同步写入 api 模块共用的 localStorage key，供请求头携带。
 */
interface AuthState {
  user: AuthUser | null;
  setUser: (user: AuthUser) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      setUser: (user) => {
        setToken(user.token);
        set({ user });
      },
      logout: () => {
        setToken(null);
        set({ user: null });
      },
    }),
    {
      name: 'hybrid:auth',
      // 持久化时把 token 同步回 api 层（刷新页面后请求仍带 token）
      onRehydrateStorage: () => (state) => {
        if (state?.user?.token) setToken(state.user.token);
      },
    },
  ),
);
