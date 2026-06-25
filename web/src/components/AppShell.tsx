/**
 * 应用骨架 —— 复刻原型布局：248px 深色侧栏 + 顶栏（标题/面包屑/全局搜索/通知）+ 内容区。
 * 侧栏导航用 NavLink 高亮当前路由；底部 userchip 显示登录用户与角色。
 */
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { useBrands } from '@/hooks/queries';
import { useAuthStore } from '@/store/authStore';
import { useUiStore } from '@/store/uiStore';
import { cn } from '@/lib/cn';
import {
  BellIcon,
  ClockIcon,
  GlobeIcon,
  GridIcon,
  LogoutIcon,
  MegaphoneIcon,
  PackageIcon,
  SearchIcon,
  SettingsIcon,
} from './icons';

interface NavItem {
  to: string;
  label: string;
  icon: typeof GridIcon;
  group: string;
  badge?: number;
}

const META: Record<string, { title: string; crumb: string }> = {
  '/channels': { title: '渠道管理', crumb: '运营 / 按大渠道分组管理小渠道包' },
  '/domains': { title: '域名配置', crumb: '运营 / 主域名 + 备用域名 + 健康巡检' },
  '/push': { title: '推送管理', crumb: '运营 / FCM 推送活动编辑、渠道批量发送、历史查看' },
  '/pack': { title: '打包中心', crumb: '交付 / 拉取后台配置并跨平台打包' },
  '/builds': { title: '构建记录', crumb: '交付 / CLI 回传的打包历史' },
  '/settings': { title: '系统设置', crumb: '系统 / 账号权限与探针配置' },
};

export function AppShell() {
  const { pathname } = useLocation();
  const { data: brands } = useBrands();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const search = useUiStore((s) => s.search);
  const setSearch = useUiStore((s) => s.setSearch);

  const total = (brands ?? []).reduce((sum, b) => sum + b.channelCount, 0);

  const items: NavItem[] = [
    { to: '/channels', label: '渠道管理', icon: GridIcon, group: '运营', badge: total || undefined },
    { to: '/domains', label: '域名配置', icon: GlobeIcon, group: '运营' },
    { to: '/push', label: '推送管理', icon: MegaphoneIcon, group: '运营' },
    { to: '/pack', label: '打包中心', icon: PackageIcon, group: '交付' },
    { to: '/builds', label: '构建记录', icon: ClockIcon, group: '交付' },
    { to: '/settings', label: '系统设置', icon: SettingsIcon, group: '系统' },
  ];

  const meta = META[pathname] ?? META['/channels'];

  // 按 group 分组渲染
  const groups = items.reduce<Record<string, NavItem[]>>((acc, it) => {
    (acc[it.group] ??= []).push(it);
    return acc;
  }, {});

  return (
    <div className="grid h-screen overflow-hidden" style={{ gridTemplateColumns: '248px 1fr' }}>
      {/* ---------- 侧边栏 ---------- */}
      <aside
        className="relative flex flex-col px-[14px] py-[18px] text-[#cbd5e1]"
        style={{ background: 'linear-gradient(180deg,#0c1426 0%,#111c33 60%,#0e1830 100%)' }}
      >
        <div className="flex items-center gap-[11px] px-2 pb-[18px] pt-2">
          <div
            className="grid place-items-center w-[34px] h-[34px] rounded-[10px] text-white font-extrabold"
            style={{
              background: 'linear-gradient(135deg,#6366f1,#8b5cf6 55%,#ec4899)',
              boxShadow: '0 6px 18px -6px rgba(139,92,246,.7)',
            }}
          >
            渠
          </div>
          <div>
            <div className="font-bold text-white text-[15px] tracking-[.2px]">渠道中台</div>
            <div className="text-[11px] text-[#7c89a3]">Hybrid Channel Console</div>
          </div>
        </div>

        <nav className="flex flex-col gap-[3px] mt-[6px]">
          {Object.entries(groups).map(([group, list]) => (
            <div key={group}>
              <div className="text-[11px] uppercase tracking-[.7px] text-[#5f6c85] px-[10px] pt-[14px] pb-[6px]">
                {group}
              </div>
              {list.map((it) => (
                <NavLink
                  key={it.to}
                  to={it.to}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-[11px] w-full px-[11px] py-[9px] rounded-[10px] text-[13.5px] font-medium transition',
                      isActive
                        ? 'text-white'
                        : 'text-[#aab6cc] hover:bg-white/5 hover:text-[#e6ecf6]',
                    )
                  }
                  style={({ isActive }) =>
                    isActive
                      ? {
                          background:
                            'linear-gradient(135deg,rgba(99,102,241,.22),rgba(139,92,246,.14))',
                          boxShadow: 'inset 0 0 0 1px rgba(129,140,248,.3)',
                        }
                      : undefined
                  }
                >
                  <it.icon className="w-[18px] h-[18px] opacity-90 flex-none" />
                  {it.label}
                  {it.badge != null && (
                    <span className="ml-auto text-[11px] bg-white/10 px-[7px] py-px rounded-full text-[#c7d2e6]">
                      {it.badge}
                    </span>
                  )}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <div className="mt-auto px-2 pt-3 pb-1 border-t border-white/[.07]">
          <div className="flex items-center gap-[10px] p-2 rounded-[10px] hover:bg-white/5 transition group">
            <div
              className="grid place-items-center w-8 h-8 rounded-[9px] text-white font-bold text-[13px]"
              style={{ background: 'linear-gradient(135deg,#22d3ee,#3b82f6)' }}
            >
              {(user?.username[0] ?? 'D').toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <div className="text-[13px] text-[#e2e8f0] font-semibold truncate">{user?.username ?? 'Daly'}</div>
              <div className="text-[11px] text-[#7c89a3]">
                {roleLabel(user?.role)} · {user?.role ?? 'admin'}
              </div>
            </div>
            <button
              onClick={logout}
              title="退出登录"
              className="grid place-items-center w-7 h-7 rounded-lg text-[#7c89a3] hover:text-white hover:bg-white/10 transition"
            >
              <LogoutIcon className="w-4 h-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* ---------- 主区 ---------- */}
      <div className="flex flex-col overflow-hidden">
        <header
          className="h-[62px] flex-none flex items-center gap-4 px-[26px] border-b border-line"
          style={{ background: 'rgba(255,255,255,.8)', backdropFilter: 'blur(10px)' }}
        >
          <div>
            <h1 className="text-[17px] font-bold tracking-[.2px]">{meta.title}</h1>
            <div className="text-muted text-[12.5px] mt-px">{meta.crumb}</div>
          </div>
          <div className="ml-auto flex items-center gap-2 bg-bg border border-line rounded-[10px] px-3 py-2 w-[280px] text-muted">
            <SearchIcon className="w-4 h-4" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="搜索渠道 / 包名 / PAL_CODE…"
              className="border-none bg-transparent outline-none w-full text-ink text-[13px] placeholder:text-muted"
            />
          </div>
          <button
            className="grid place-items-center w-[38px] h-[38px] rounded-[10px] border border-line bg-panel text-ink-2 hover:bg-bg hover:text-ink transition"
            title="通知"
          >
            <BellIcon className="w-[18px] h-[18px]" />
          </button>
        </header>

        <main className="flex-1 overflow-auto px-[26px] pt-6 pb-10">
          <div className="animate-fade">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

function roleLabel(role?: string): string {
  return role === 'admin' ? '管理员' : role === 'viewer' ? '只读' : '运营';
}
