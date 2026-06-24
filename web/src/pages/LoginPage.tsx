/**
 * 登录页 —— 沿用品牌视觉（深色渐变 + 玻璃卡片）。
 * 调真实 authApi.login（POST /api/auth/login，拿 accessToken 写入登录态）。
 * 后端默认 bootstrap 账号 admin / admin12345（server seed）。后端不可达时回退演示模式。
 */
import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Button } from '@/components/ui';

export function LoginPage() {
  const navigate = useNavigate();
  const setUser = useAuthStore((s) => s.setUser);
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const user = await authApi.login(username.trim(), password);
      setUser(user);
      navigate('/channels', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="min-h-screen grid place-items-center px-4"
      style={{ background: 'linear-gradient(135deg,#0c1426 0%,#111c33 55%,#1a1240 100%)' }}
    >
      <div className="w-full max-w-[400px]">
        <div className="flex items-center gap-3 justify-center mb-6">
          <div
            className="grid place-items-center w-11 h-11 rounded-[12px] text-white font-extrabold text-lg"
            style={{
              background: 'linear-gradient(135deg,#6366f1,#8b5cf6 55%,#ec4899)',
              boxShadow: '0 6px 18px -6px rgba(139,92,246,.7)',
            }}
          >
            渠
          </div>
          <div className="text-white">
            <div className="font-bold text-lg leading-tight">渠道中台</div>
            <div className="text-[12px] text-[#8b97b3]">Hybrid Channel Console</div>
          </div>
        </div>

        <form
          onSubmit={onSubmit}
          className="rounded-[16px] p-6 backdrop-blur-xl"
          style={{
            background: 'rgba(255,255,255,.06)',
            border: '1px solid rgba(255,255,255,.1)',
            boxShadow: '0 24px 60px -20px rgba(0,0,0,.6)',
          }}
        >
          <h1 className="text-white text-[18px] font-bold mb-1">登录</h1>
          <p className="text-[#8b97b3] text-[12.5px] mb-5">使用后台账号登录以管理渠道、域名与打包</p>

          <label className="block text-[12.5px] font-semibold text-[#cbd5e1] mb-1.5">账号</label>
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            className="w-full rounded-[10px] px-3 py-[10px] text-[13px] text-white outline-none mb-4 transition focus:border-brand"
            style={{ background: 'rgba(255,255,255,.06)', border: '1px solid rgba(255,255,255,.12)' }}
            placeholder="admin"
          />

          <label className="block text-[12.5px] font-semibold text-[#cbd5e1] mb-1.5">密码</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            className="w-full rounded-[10px] px-3 py-[10px] text-[13px] text-white outline-none mb-5 transition focus:border-brand"
            style={{ background: 'rgba(255,255,255,.06)', border: '1px solid rgba(255,255,255,.12)' }}
            placeholder="••••••••"
          />

          {error && (
            <div className="mb-4 text-[12.5px] text-[#fca5a5] bg-[#7f1d1d]/30 border border-[#b91c1c]/40 rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <Button type="submit" variant="primary" disabled={loading} className="w-full justify-center">
            {loading ? '登录中…' : '登录'}
          </Button>

          <p className="text-[11.5px] text-[#6b7793] mt-4 text-center">
            真实后端默认账号 <span className="mono">admin / admin12345</span>。后端不可达时回退演示模式（任意非空账号）。
          </p>
        </form>
      </div>
    </div>
  );
}
