import type { AdminUser, AdminUserInput, AuthUser, PermCatalogModule, Role, RoleInput } from '../types';
import { ALL_PERM_CODES, PERM, PERM_CATALOG } from '../permissions';

/**
 * mock 层用普通 Error 而非 api.ts 的 ApiError（避免 api.ts ↔ mock/rbac.ts 循环导入）。
 * UI 侧统一按 `err.message` 原样展示（10-rbac.md 要求「如实提示」，无需区分状态码）。
 */

/**
 * 进程内 mock RBAC 数据库（后端未就绪时的事实来源），遵循 10-rbac.md 契约：
 *  - 种子：超级管理员（builtin）+ 一个示例普通角色「运营」+ 两个示例用户（admin / operator）；
 *  - 校验规则与文档一致：builtin 不可改/删、删除挂人角色 409、最后一个超管保护、不能删自己等。
 *
 * mock 登录用 token 形如 `mock-token-<username>-<timestamp>`，authApi.me() 的 mock 分支据此
 * 反解出 username 再查回用户（仅 mock 场景；真实后端用 JWT 内的 sub）。
 */

interface MockRole {
  id: string;
  name: string;
  description: string;
  builtin: boolean;
  permCodes: string[];
}

interface MockUser {
  id: string;
  username: string;
  password: string;
  roleId: string;
  createdAt: string;
}

const OPERATOR_PERM_CODES = ALL_PERM_CODES.filter(
  (c) => c !== PERM.STORE_MANAGE && c !== PERM.USER_MANAGE && c !== PERM.ROLE_MANAGE,
);

let roles: MockRole[] = [
  { id: '1', name: '超级管理员', description: '内置角色，拥有全部权限，不可编辑或删除', builtin: true, permCodes: ALL_PERM_CODES },
  { id: '2', name: '运营', description: '日常运营：渠道 / 域名 / 打包 / 推送 / 上架包全部操作，不含系统设置管理', builtin: false, permCodes: OPERATOR_PERM_CODES },
];
let roleSeq = 3;

let users: MockUser[] = [
  { id: '1', username: 'admin', password: 'admin12345', roleId: '1', createdAt: '2026-01-01T00:00:00Z' },
  { id: '2', username: 'operator', password: 'operator123', roleId: '2', createdAt: '2026-02-01T00:00:00Z' },
];
let userSeq = 3;

function userCountOf(roleId: string): number {
  return users.filter((u) => u.roleId === roleId).length;
}

function toRole(r: MockRole): Role {
  return { id: r.id, name: r.name, description: r.description, builtin: r.builtin, permCodes: [...r.permCodes], userCount: userCountOf(r.id) };
}

function toAdminUser(u: MockUser): AdminUser {
  const role = roles.find((r) => r.id === u.roleId);
  return {
    id: u.id,
    username: u.username,
    roleId: u.roleId,
    roleName: role?.name ?? '（角色已删除）',
    builtinRole: role?.builtin ?? false,
    createdAt: u.createdAt,
  };
}

/** 是否为「最后一个超级管理员」：role.builtin 且当前只有这一个用户挂在该角色下。 */
function isLastSuperAdmin(u: MockUser): boolean {
  const role = roles.find((r) => r.id === u.roleId);
  if (!role?.builtin) return false;
  return userCountOf(u.roleId) <= 1;
}

/** 从 mock token（`mock-token-<username>-<ts>`）反解 username；非 mock token 返回 null。 */
function usernameFromMockToken(token: string | null): string | null {
  if (!token || !token.startsWith('mock-token-')) return null;
  const rest = token.slice('mock-token-'.length);
  const m = rest.match(/^(.*)-\d+$/);
  return m ? m[1] : null;
}

export const mockRbacDb = {
  catalog(): PermCatalogModule[] {
    return PERM_CATALOG;
  },

  listRoles(): Role[] {
    return roles.map(toRole);
  },

  createRole(input: RoleInput): Role {
    const name = input.name.trim();
    if (!name) throw new Error('角色名称必填');
    if (roles.some((r) => r.name === name)) throw new Error(`角色名称「${name}」已存在`);
    const invalid = input.permCodes.filter((c) => !ALL_PERM_CODES.includes(c));
    if (invalid.length) throw new Error(`权限点不存在：${invalid.join(', ')}`);
    const role: MockRole = { id: String(roleSeq++), name, description: input.description?.trim() ?? '', builtin: false, permCodes: [...new Set(input.permCodes)] };
    roles = [...roles, role];
    return toRole(role);
  },

  updateRole(id: string, input: RoleInput): Role {
    const role = roles.find((r) => r.id === id);
    if (!role) throw new Error('角色不存在');
    if (role.builtin) throw new Error('内置角色不可编辑');
    const name = input.name.trim();
    if (!name) throw new Error('角色名称必填');
    if (roles.some((r) => r.id !== id && r.name === name)) throw new Error(`角色名称「${name}」已存在`);
    const invalid = input.permCodes.filter((c) => !ALL_PERM_CODES.includes(c));
    if (invalid.length) throw new Error(`权限点不存在：${invalid.join(', ')}`);
    role.name = name;
    role.description = input.description?.trim() ?? '';
    role.permCodes = [...new Set(input.permCodes)];
    return toRole(role);
  },

  removeRole(id: string): void {
    const role = roles.find((r) => r.id === id);
    if (!role) throw new Error('角色不存在');
    if (role.builtin) throw new Error('内置角色不可删除');
    if (userCountOf(id) > 0) throw new Error('该角色下还有用户，无法删除，请先将用户改到其他角色');
    roles = roles.filter((r) => r.id !== id);
  },

  listUsers(): AdminUser[] {
    return users.map(toAdminUser);
  },

  createUser(input: AdminUserInput): AdminUser {
    const username = input.username.trim();
    if (!username) throw new Error('用户名必填');
    if (username.length > 64) throw new Error('用户名不能超过 64 字节');
    if (users.some((u) => u.username.toLowerCase() === username.toLowerCase())) throw new Error(`用户名「${username}」已存在`);
    if (!input.password || input.password.length < 6) throw new Error('密码至少 6 位');
    const role = roles.find((r) => r.id === input.roleId);
    if (!role) throw new Error('角色不存在');
    const user: MockUser = { id: String(userSeq++), username, password: input.password, roleId: role.id, createdAt: new Date().toISOString() };
    users = [...users, user];
    return toAdminUser(user);
  },

  updateUserRole(id: string, roleId: string): AdminUser {
    const user = users.find((u) => u.id === id);
    if (!user) throw new Error('用户不存在');
    const newRole = roles.find((r) => r.id === roleId);
    if (!newRole) throw new Error('角色不存在');
    if (user.roleId !== roleId && isLastSuperAdmin(user)) {
      throw new Error('不能修改最后一个超级管理员的角色');
    }
    user.roleId = roleId;
    return toAdminUser(user);
  },

  resetPassword(id: string, password: string): void {
    const user = users.find((u) => u.id === id);
    if (!user) throw new Error('用户不存在');
    if (!password || password.length < 6) throw new Error('密码至少 6 位');
    user.password = password;
  },

  removeUser(id: string, currentUsername: string | null): void {
    const user = users.find((u) => u.id === id);
    if (!user) throw new Error('用户不存在');
    if (currentUsername && user.username === currentUsername) throw new Error('不能删除自己');
    if (isLastSuperAdmin(user)) throw new Error('不能删除最后一个超级管理员');
    users = users.filter((u) => u.id !== id);
  },

  /** 登录：仅接受种子账号（admin/admin12345、operator/operator123），与真实后端 bootstrap 账号一致。 */
  login(username: string, password: string): AuthUser {
    const uname = username.trim();
    if (!uname || !password) throw new Error('请输入账号密码');
    const user = users.find((u) => u.username === uname);
    if (!user || user.password !== password) throw new Error('账号或密码错误');
    const role = roles.find((r) => r.id === user.roleId);
    const perms = role ? (role.builtin ? ALL_PERM_CODES : role.permCodes) : [];
    return {
      username: user.username,
      role: role ? { id: role.id, name: role.name, builtin: role.builtin } : { id: '', name: '未分配角色', builtin: false },
      perms,
      token: `mock-token-${user.username}-${Date.now()}`,
    };
  },

  /** GET /api/auth/me 的 mock：从 mock token 反解 username 重新查权限（角色被改后刷新生效）。 */
  me(token: string | null): { role: AuthUser['role']; perms: string[] } {
    const uname = usernameFromMockToken(token);
    const user = uname ? users.find((u) => u.username === uname) : undefined;
    if (!user) throw new Error('未登录或登录态已失效');
    const role = roles.find((r) => r.id === user.roleId);
    const perms = role ? (role.builtin ? ALL_PERM_CODES : role.permCodes) : [];
    return { role: role ? { id: role.id, name: role.name, builtin: role.builtin } : { id: '', name: '未分配角色', builtin: false }, perms };
  },
};
