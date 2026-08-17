/**
 * 角色管理页（/roles）—— 从「系统设置」拆出的独立菜单页（用户明确要求菜单粒度）。
 * 路由/菜单可见性守卫用 role:manage（10-rbac.md：该 code 现已从「系统设置」模块下的
 * button 权限改为独立「角色管理」模块的 route 权限，字符串本身不变）。
 * 页面本身只负责标题 + 说明文案，具体列表/抽屉逻辑仍在 <RoleManager /> 里。
 */
import { RoleManager } from '@/components/RoleManager';

export function RolesPage() {
  return (
    // 不再用窄 max-w 卡片布局（宽屏会留大片空白）：RoleManager 内部改为表格，
    // 天然随容器宽度自适应，这里只加一个宽屏下的合理上限，避免超宽显示器上表格行过长。
    <section className="flex flex-col gap-3 max-w-[1180px]">
      {/* 同 UsersPage：顶栏已有标题 + 面包屑，页面内不再重复 h2。 */}
      <p className="text-[12.5px] text-muted">
        角色是一组权限点的集合，新增/编辑角色时按模块勾选路由权限与按钮权限，并可限定该角色能操作的
        品牌/渠道数据范围；<b>超级管理员</b>为内置角色，拥有全部权限与数据范围，不可编辑或删除；
        删除角色前需先把挂在其下的用户改到其他角色。
      </p>
      <RoleManager />
    </section>
  );
}
