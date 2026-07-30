import type { AuthUser } from './types';

/**
 * 角色判断 —— 唯一权威来源是后端（403 才是真正的拒绝）；这里只用于前端 UX
 * （隐藏导航/按钮），避免同一个判断散落在多个组件里各写一遍。
 */
export function isAdmin(user: AuthUser | null | undefined): boolean {
  return user?.role === 'admin';
}
