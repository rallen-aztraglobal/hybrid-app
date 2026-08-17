import { useEffect, useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { ROUTE_PERM_ORDER } from '@/lib/permissions';
import { AppShell } from '@/components/AppShell';
import { LoginPage } from '@/pages/LoginPage';
import { ChannelsPage } from '@/pages/ChannelsPage';
import { ListingsPage } from '@/pages/ListingsPage';
import { DomainsPage } from '@/pages/DomainsPage';
import { PackPage } from '@/pages/PackPage';
import { BuildsPage } from '@/pages/BuildsPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { PushPage } from '@/pages/PushPage';
import { DevicesPage } from '@/pages/DevicesPage';
import { RolesPage } from '@/pages/RolesPage';
import { UsersPage } from '@/pages/UsersPage';
import { Button } from '@/components/ui';

/** 路由路径 → 页面组件（与 ROUTE_PERM_ORDER 的 path 一一对应）。 */
const PAGES: Record<string, JSX.Element> = {
  '/channels': <ChannelsPage />,
  '/listings': <ListingsPage />,
  '/domains': <DomainsPage />,
  '/push': <PushPage />,
  '/devices': <DevicesPage />,
  '/pack': <PackPage />,
  '/builds': <BuildsPage />,
  '/settings': <SettingsPage />,
  '/roles': <RolesPage />,
  '/users': <UsersPage />,
};

/**
 * 未分配任何 page:* 权限时的友好空态。两种成因：
 *  (1) 真的没有任何页面权限（联系管理员分配角色）；
 *  (2) refreshMe() 还没成功回填 perms（旧会话迁移 / 混合部署下 /auth/me 探活失败等）——
 *      这种情况下不该让用户永久卡在这里：展示上次失败原因 + 提供手动重试。
 * 注意：本组件在路由里**不绑定固定路径**（App() 的 `*` 通配符按当前 fallbackPath 动态决定
 * 渲染 Navigate 还是本组件），所以一旦 refreshMe 后 perms 非空，下一次渲染会被
 * 通配符路由自动重定向到第一个有权限的页面，不需要额外的命令式 navigate。
 */
function NoAccessPage() {
  const meError = useAuthStore((s) => s.meError);
  const refreshMe = useAuthStore((s) => s.refreshMe);
  const [retrying, setRetrying] = useState(false);

  async function retry() {
    setRetrying(true);
    try {
      await refreshMe();
    } finally {
      setRetrying(false);
    }
  }

  return (
    <div className="grid place-items-center h-[60vh] text-center px-6">
      <div className="max-w-[420px]">
        <div className="text-[15px] font-bold text-ink mb-2">当前账号暂无可用页面</div>
        <div className="text-[13px] text-muted">未分配任何页面权限，请联系管理员在「系统 · 角色管理」中调整。</div>
        {meError && (
          <div className="mt-3 text-[12.5px] text-down">
            权限信息刷新失败：{meError}（已保留上次已知权限，可能并非真的无权限）
          </div>
        )}
        <Button className="mt-4" onClick={() => void retry()} disabled={retrying}>
          {retrying ? '重试中…' : '重新获取权限'}
        </Button>
      </div>
    </div>
  );
}

/**
 * 路由：未登录强制跳 /login；登录后按 RBAC perms 过滤路由（10-rbac.md）：
 *  - 只渲染 hasPerm(page:*) 通过的路由；
 *  - 未匹配 / 无权限路径统一跳到「第一个有权限的页面」；
 *  - 全无页面权限（或 perms 尚未刷新出来）时落到 AppShell 内的友好空态，
 *    且该空态是**动态求值**的（见 fallbackPath 用法），refreshMe() 回填 perms 后
 *    下一次渲染会自动带用户离开空态，不需要用户手动点侧边栏。
 */
export default function App() {
  const user = useAuthStore((s) => s.user);
  const hasPerm = useAuthStore((s) => s.hasPerm);
  const refreshMe = useAuthStore((s) => s.refreshMe);

  // 已登录时启动刷新一次 role+perms（角色被后台改过也无需重登即可生效）。
  useEffect(() => {
    if (user) void refreshMe();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [!!user]);

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    );
  }

  const allowed = ROUTE_PERM_ORDER.filter((r) => hasPerm(r.perm));
  // null = 当前没有任何页面权限。刻意不给「无权限空态」注册固定路径（如 /no-access）——
  // 只让通配符 `*` 按当前 allowed/fallbackPath **每次渲染动态判定**要不要重定向。
  // 这样即便浏览器地址栏之前被带到过某个空态落地页，refreshMe() 之后 fallbackPath
  // 变为非空时，下一次渲染通配符会自动改成 <Navigate>，用户随之自动跳走
  // （评审「应修1(a)」：不需要额外监听「是否停在无权限页」的旁路 effect）。
  const fallbackPath = allowed[0]?.path ?? null;

  return (
    <Routes>
      <Route path="/login" element={<Navigate to={fallbackPath ?? '/no-access'} replace />} />
      <Route element={<AppShell />}>
        {allowed.map((r) => (
          <Route key={r.path} path={r.path} element={PAGES[r.path]} />
        ))}
        <Route path="*" element={fallbackPath ? <Navigate to={fallbackPath} replace /> : <NoAccessPage />} />
      </Route>
    </Routes>
  );
}
