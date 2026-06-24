import { Navigate, Route, Routes } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { AppShell } from '@/components/AppShell';
import { LoginPage } from '@/pages/LoginPage';
import { ChannelsPage } from '@/pages/ChannelsPage';
import { DomainsPage } from '@/pages/DomainsPage';
import { PackPage } from '@/pages/PackPage';
import { BuildsPage } from '@/pages/BuildsPage';
import { SettingsPage } from '@/pages/SettingsPage';

/**
 * 路由：未登录强制跳 /login；登录后进入 AppShell（侧栏+顶栏）下的各业务页。
 * 路由结构对齐原型侧边栏：渠道管理 / 域名配置 / 打包中心 / 构建记录 / 系统设置。
 */
export default function App() {
  const user = useAuthStore((s) => s.user);

  if (!user) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<Navigate to="/channels" replace />} />
      <Route element={<AppShell />}>
        <Route path="/channels" element={<ChannelsPage />} />
        <Route path="/domains" element={<DomainsPage />} />
        <Route path="/pack" element={<PackPage />} />
        <Route path="/builds" element={<BuildsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<Navigate to="/channels" replace />} />
      </Route>
    </Routes>
  );
}
