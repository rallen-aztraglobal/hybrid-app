/**
 * 角色管理（10-rbac.md，需 role:manage 可见）—— 角色列表（名称/描述/人数/builtin 徽标）
 * + 新增/编辑抽屉（RoleDrawer）。删除带确认；后端 409（有用户挂载）/ 400（内置角色）错误如实展示。
 */
import { useState } from 'react';
import type { Role } from '@/lib/types';
import { useDeleteRole, usePermsCatalog, useRoles } from '@/hooks/queries';
import { Button } from './ui';
import { EditIcon, PlusIcon, ShieldIcon, TrashIcon } from './icons';
import { RoleDrawer } from './RoleDrawer';
import { cn } from '@/lib/cn';

export function RoleManager() {
  const { data: roles, isLoading } = useRoles();
  const { data: catalog } = usePermsCatalog();
  const deleteRole = useDeleteRole();

  const [target, setTarget] = useState<'new' | Role | null>(null);
  const [rowError, setRowError] = useState<{ id: string; message: string } | null>(null);

  const sorted = (roles ?? []).slice().sort((a, b) => (b.builtin ? 1 : 0) - (a.builtin ? 1 : 0) || a.name.localeCompare(b.name));

  async function remove(role: Role) {
    setRowError(null);
    if (!confirm(`确认删除角色「${role.name}」？`)) return;
    try {
      await deleteRole.mutateAsync(role.id);
    } catch (err) {
      setRowError({ id: role.id, message: err instanceof Error ? err.message : '删除失败' });
    }
  }

  return (
    <div className="section-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="flex items-center gap-2 text-[13px] font-bold">
          <ShieldIcon className="w-[16px] h-[16px]" />
          角色管理
        </h3>
        <Button onClick={() => setTarget('new')}>
          <PlusIcon className="w-4 h-4" />
          新增角色
        </Button>
      </div>
      <p className="text-[12px] text-muted mb-3">
        角色是一组权限点的集合；超级管理员为内置角色，拥有全部权限、不可编辑或删除。删除角色前需先把挂在其下的用户改到其他角色。
      </p>

      {isLoading && <div className="text-[12.5px] text-muted py-2">加载中…</div>}

      <div className="flex flex-col gap-2">
        {sorted.map((r) => (
          <div key={r.id} className="rounded-[10px] border border-line p-3">
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-[13px] font-semibold truncate">{r.name}</span>
                  {r.builtin && (
                    <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#dbeafe] text-[#1d4ed8] flex-none">
                      内置
                    </span>
                  )}
                </div>
                <div className="text-[11.5px] text-muted mt-0.5 truncate">{r.description || '（无描述）'}</div>
              </div>
              <span className="text-[11px] text-ink-2 bg-panel-2 px-2 py-0.5 rounded-full flex-none">
                {r.userCount} 人 · {r.permCodes.length} 项权限
              </span>
              <button
                title={r.builtin ? '查看' : '编辑'}
                onClick={() => setTarget(r)}
                className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 hover:bg-bg hover:text-ink transition flex-none"
              >
                <EditIcon className="w-[15px] h-[15px]" />
              </button>
              {!r.builtin && (
                <button
                  title="删除"
                  onClick={() => remove(r)}
                  className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 hover:text-down hover:border-[#fecaca] hover:bg-[#fef2f2] transition flex-none"
                >
                  <TrashIcon className="w-[15px] h-[15px]" />
                </button>
              )}
            </div>
            {rowError?.id === r.id && <div className={cn('mt-2 text-[12px] text-down')}>{rowError.message}</div>}
          </div>
        ))}
        {!isLoading && sorted.length === 0 && <div className="text-[12.5px] text-muted py-2">暂无角色</div>}
      </div>

      <RoleDrawer target={target} catalog={catalog ?? []} onClose={() => setTarget(null)} />
    </div>
  );
}
