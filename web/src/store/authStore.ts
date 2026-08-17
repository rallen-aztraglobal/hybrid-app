import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthUser } from '@/lib/types';
import { ApiError, authApi, setToken } from '@/lib/api';

/**
 * 登录态（本地状态，Zustand + persist）。
 * token 同步写入 api 模块共用的 localStorage key，供请求头携带。
 *
 * RBAC（10-rbac.md）：user.perms 是按钮/路由权限判断的唯一依据（hasPerm），
 * 启动时由 App.tsx 调 refreshMe() 拉 GET /api/auth/me 刷新 role+perms
 * （角色被改后无需重登即可生效）。
 *
 * refreshMe 失败区分两类（评审「应修1」）：
 *  - 401（账号/token 已失效，如账号被删）：直接登出，交回 App.tsx 走 /login，
 *    不让用户顶着旧 perms 卡在原地——真实 401 走 api.ts request() 的全局处理已经会
 *    清 token/localStorage，这里的 logout() 是在 store 层面同步这一状态，确保
 *    `user` 立刻变 null、触发 App 重新渲染成登录态（不依赖整页刷新）。
 *  - 其它失败（网络错误 / 端点缺失 404 / 5xx / 混合部署下 mock 兜底解不出真实 JWT 对应的
 *    用户等）：保留旧 perms，但把错误记到 meError，供 UI（App.tsx 的 NoAccessPage）
 *    展示提示 + 提供手动重试，避免用户永久静默卡死在空态却毫无线索。
 */
interface AuthState {
  user: AuthUser | null;
  /** 最近一次 refreshMe 非 401 失败的原因；成功或尚未失败过时为 null（不持久化）。 */
  meError: string | null;
  setUser: (user: AuthUser) => void;
  logout: () => void;
  /** 是否拥有某权限点 code；未登录恒为 false。 */
  hasPerm: (code: string) => boolean;
  refreshMe: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      meError: null,
      setUser: (user) => {
        setToken(user.token);
        set({ user, meError: null });
      },
      logout: () => {
        setToken(null);
        set({ user: null, meError: null });
      },
      hasPerm: (code) => get().user?.perms.includes(code) ?? false,
      refreshMe: async () => {
        const current = get().user;
        if (!current) return;
        try {
          const me = await authApi.me();
          // token/refreshToken/username 不变，只合并最新 role+perms。
          set({ user: { ...current, role: me.role, perms: me.perms }, meError: null });
        } catch (err) {
          if (err instanceof ApiError && err.status === 401) {
            // 账号/token 已失效（如账号被后台删除）：登出而非静默保留旧 perms，
            // 避免用户顶着过期权限停留在界面上直到踩到下一个 401 才被踢。
            get().logout();
            return;
          }
          // 网络/端点缺失/5xx/混合部署下 mock 兜底无法识别真实 JWT 等：不是「登录态失效」，
          // 保留旧 perms（不能因为一次探活失败就把人锁死在无权限空态），但记录错误可见、可重试。
          set({ meError: err instanceof Error ? err.message : '刷新权限失败' });
        }
      },
    }),
    {
      name: 'hybrid:auth',
      version: 2,
      // 只持久化 user：meError 是一次运行期的临时提示，不应该跨会话残留。
      partialize: (state) => ({ user: state.user }),
      // 旧版本持久化数据里 user.role 是 'admin'|'operator'|'viewer' 字符串、无 perms 字段。
      // 安全迁移：role 归一成占位对象、perms 缺失时当空数组，等 refreshMe() 首次拉取后自愈。
      migrate: (persisted) => {
        const state = persisted as { user?: Record<string, unknown> } | undefined;
        const u = state?.user;
        if (!u) return { user: null };
        const roleIsLegacyString = typeof u.role === 'string';
        if (roleIsLegacyString || !Array.isArray(u.perms)) {
          return {
            user: {
              username: u.username,
              token: u.token,
              refreshToken: u.refreshToken,
              role:
                u.role && typeof u.role === 'object'
                  ? u.role
                  : { id: '', name: typeof u.role === 'string' ? u.role : '', builtin: false },
              perms: Array.isArray(u.perms) ? u.perms : [],
            } as AuthUser,
          };
        }
        return { user: u as unknown as AuthUser };
      },
      // 持久化时把 token 同步回 api 层（刷新页面后请求仍带 token）
      onRehydrateStorage: () => (state) => {
        if (state?.user?.token) setToken(state.user.token);
      },
    },
  ),
);
