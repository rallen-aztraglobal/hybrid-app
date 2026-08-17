/**
 * 用户管理页（/users）—— 从「系统设置」拆出的独立菜单页（用户明确要求菜单粒度）。
 * 路由/菜单可见性守卫用 user:manage（10-rbac.md：该 code 现已从「系统设置」模块下的
 * button 权限改为独立「用户管理」模块的 route 权限，字符串本身不变）。
 * 页面本身只负责标题 + 说明文案，具体列表/操作逻辑仍在 <UserManager /> 里。
 */
import { UserManager } from '@/components/UserManager';

export function UsersPage() {
  return (
    // 不再用窄 max-w 卡片布局（宽屏会留大片空白）：UserManager 内部改为表格，
    // 天然随容器宽度自适应，这里只加一个宽屏下的合理上限，避免超宽显示器上表格行过长。
    <section className="flex flex-col gap-3 max-w-[1040px]">
      {/* 不再重复页面标题：AppShell 顶栏已经渲染「用户管理」+ 面包屑，这里再来一个 h2 是同屏两遍。
          只留一句说明；宽度收到 1040 —— 四列表格再宽就是把几个短字段拉散，不是信息密度。 */}
      <p className="text-[12.5px] text-muted">
        每个用户挂一个角色，权限完全由角色决定；系统会拒绝删除/改角色「最后一个超级管理员」、拒绝删除自己。
      </p>
      <UserManager />
    </section>
  );
}
