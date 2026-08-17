/**
 * 用户管理（10-rbac.md，需 user:manage 可见）—— 用户列表（用户名/角色/创建时间）
 * + 新增（用户名/密码/确认密码/角色下拉）、改角色、重置密码、删除（带确认）。
 * 「最后一个超管」「不能删自己」等后端 400/409 错误如实展示（不在前端翻译措辞）。
 */
import { useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import {
  useCreateUser,
  useDeleteUser,
  useResetUserPassword,
  useRoles,
  useUpdateUserRole,
  useUsers,
} from '@/hooks/queries';
import { Button, Select } from './ui';
import { PlusIcon, TrashIcon } from './icons';
import { cn } from '@/lib/cn';

export function UserManager() {
  const currentUsername = useAuthStore((s) => s.user?.username);
  const { data: users, isLoading } = useUsers();
  const { data: roles } = useRoles();
  const createUser = useCreateUser();
  const updateRole = useUpdateUserRole();
  const resetPassword = useResetUserPassword();
  const deleteUser = useDeleteUser();

  const roleOptions = (roles ?? []).map((r) => ({ value: r.id, label: r.name }));

  const [adding, setAdding] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newPasswordConfirm, setNewPasswordConfirm] = useState('');
  const [newRoleId, setNewRoleId] = useState('');
  const [addError, setAddError] = useState<string | null>(null);

  const [resettingId, setResettingId] = useState<string | null>(null);
  const [resetPwd, setResetPwd] = useState('');
  const [resetPwdConfirm, setResetPwdConfirm] = useState('');

  const [rowError, setRowError] = useState<{ id: string; message: string } | null>(null);

  async function submitAdd() {
    setAddError(null);
    const username = newUsername.trim();
    if (!username) {
      setAddError('用户名必填');
      return;
    }
    if (!newPassword || newPassword.length < 6) {
      setAddError('密码至少 6 位');
      return;
    }
    if (newPassword !== newPasswordConfirm) {
      setAddError('两次输入的密码不一致');
      return;
    }
    if (!newRoleId) {
      setAddError('请选择角色');
      return;
    }
    try {
      await createUser.mutateAsync({ username, password: newPassword, roleId: newRoleId });
      setAdding(false);
      setNewUsername('');
      setNewPassword('');
      setNewPasswordConfirm('');
      setNewRoleId('');
    } catch (err) {
      setAddError(err instanceof Error ? err.message : '新增失败');
    }
  }

  async function changeRole(userId: string, roleId: string) {
    setRowError(null);
    try {
      await updateRole.mutateAsync({ id: userId, roleId });
    } catch (err) {
      setRowError({ id: userId, message: err instanceof Error ? err.message : '修改角色失败' });
    }
  }

  function startReset(userId: string) {
    setResettingId(userId);
    setResetPwd('');
    setResetPwdConfirm('');
    setRowError(null);
  }

  async function submitReset(userId: string) {
    setRowError(null);
    if (!resetPwd || resetPwd.length < 6) {
      setRowError({ id: userId, message: '密码至少 6 位' });
      return;
    }
    if (resetPwd !== resetPwdConfirm) {
      setRowError({ id: userId, message: '两次输入的密码不一致' });
      return;
    }
    try {
      await resetPassword.mutateAsync({ id: userId, password: resetPwd });
      setResettingId(null);
    } catch (err) {
      setRowError({ id: userId, message: err instanceof Error ? err.message : '重置密码失败' });
    }
  }

  async function remove(userId: string, username: string) {
    setRowError(null);
    if (!confirm(`确认删除用户「${username}」？`)) return;
    try {
      await deleteUser.mutateAsync(userId);
    } catch (err) {
      setRowError({ id: userId, message: err instanceof Error ? err.message : '删除失败' });
    }
  }

  return (
    <div className="section-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-[13px] font-bold">用户管理</h3>
        <Button onClick={() => setAdding((v) => !v)}>
          <PlusIcon className="w-4 h-4" />
          新增用户
        </Button>
      </div>
      <p className="text-[12px] text-muted mb-3">
        每个用户挂一个角色，权限完全由角色决定。系统会拒绝删除/改角色「最后一个超级管理员」、拒绝删除自己。
      </p>

      {adding && (
        <div className="flex flex-wrap items-end gap-2 p-3 mb-3 rounded-[10px] border border-line bg-panel-2">
          <div className="flex-1 min-w-[120px]">
            <label className="block text-[11.5px] text-muted mb-1">用户名</label>
            <input className="field-input" value={newUsername} onChange={(e) => setNewUsername(e.target.value)} />
          </div>
          <div className="flex-1 min-w-[120px]">
            <label className="block text-[11.5px] text-muted mb-1">密码</label>
            <input
              type="password"
              className="field-input"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
          </div>
          <div className="flex-1 min-w-[120px]">
            <label className="block text-[11.5px] text-muted mb-1">确认密码</label>
            <input
              type="password"
              className="field-input"
              value={newPasswordConfirm}
              onChange={(e) => setNewPasswordConfirm(e.target.value)}
            />
          </div>
          <div className="flex-1 min-w-[140px]">
            <label className="block text-[11.5px] text-muted mb-1">角色</label>
            <Select value={newRoleId} onChange={setNewRoleId} options={roleOptions} placeholder="选择角色…" />
          </div>
          <Button variant="primary" onClick={submitAdd} disabled={createUser.isPending}>
            {createUser.isPending ? '保存中…' : '确定'}
          </Button>
          <Button
            onClick={() => {
              setAdding(false);
              setAddError(null);
            }}
          >
            取消
          </Button>
          {addError && <div className="w-full text-[12px] text-down">{addError}</div>}
        </div>
      )}

      {isLoading && <div className="text-[12.5px] text-muted py-2">加载中…</div>}
      {!isLoading && (users ?? []).length === 0 && <div className="text-[12.5px] text-muted py-2">暂无用户</div>}

      <div className="flex flex-col gap-2">
        {(users ?? []).map((u) => (
          <div key={u.id} className="rounded-[10px] border border-line p-3">
            <div className="flex flex-wrap items-center gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-[13px] font-semibold truncate">{u.username}</span>
                  {u.username === currentUsername && (
                    <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#f1f5f9] text-[#64748b] flex-none">
                      当前登录
                    </span>
                  )}
                  {u.builtinRole && (
                    <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#dbeafe] text-[#1d4ed8] flex-none">
                      超级管理员
                    </span>
                  )}
                </div>
                <div className="text-[11px] text-muted mt-0.5">
                  创建于 {u.createdAt ? new Date(u.createdAt).toLocaleString('zh-CN', { hour12: false }) : '—'}
                </div>
              </div>
              <div className="w-[160px] flex-none">
                {roleOptions.length > 0 ? (
                  <Select value={u.roleId} onChange={(v) => void changeRole(u.id, v)} options={roleOptions} />
                ) : (
                  <span className="text-[12.5px] text-ink-2">{u.roleName}</span>
                )}
              </div>
              <Button onClick={() => startReset(u.id)}>重置密码</Button>
              <button
                title="删除"
                onClick={() => void remove(u.id, u.username)}
                className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 hover:text-down hover:border-[#fecaca] hover:bg-[#fef2f2] transition flex-none"
              >
                <TrashIcon className="w-[15px] h-[15px]" />
              </button>
            </div>

            {resettingId === u.id && (
              <div className="mt-3 flex flex-wrap items-end gap-2 p-3 rounded-[10px] border border-line bg-panel-2">
                <div className="flex-1 min-w-[120px]">
                  <label className="block text-[11.5px] text-muted mb-1">新密码</label>
                  <input
                    type="password"
                    className="field-input"
                    value={resetPwd}
                    onChange={(e) => setResetPwd(e.target.value)}
                  />
                </div>
                <div className="flex-1 min-w-[120px]">
                  <label className="block text-[11.5px] text-muted mb-1">确认新密码</label>
                  <input
                    type="password"
                    className="field-input"
                    value={resetPwdConfirm}
                    onChange={(e) => setResetPwdConfirm(e.target.value)}
                  />
                </div>
                <Button variant="primary" onClick={() => void submitReset(u.id)} disabled={resetPassword.isPending}>
                  {resetPassword.isPending ? '保存中…' : '确定'}
                </Button>
                <Button onClick={() => setResettingId(null)}>取消</Button>
              </div>
            )}
            {rowError?.id === u.id && <div className={cn('mt-2 text-[12px] text-down')}>{rowError.message}</div>}
          </div>
        ))}
      </div>
    </div>
  );
}
