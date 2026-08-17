/**
 * 角色管理（10-rbac.md，需 role:manage 可见，现为独立菜单页 /roles）——
 * 表格化角色列表（名称/描述/人数/权限数/数据范围/操作）+ 新增/编辑抽屉（RoleDrawer）。
 * 宽屏一行放下全部列信息密度更高；窄屏收起次要列，折进名称单元格下面，不产生横向滚动。
 * 删除带确认；后端 409（有用户挂载）/ 400（内置角色）错误如实展示。
 * 页头/说明文案在 pages/RolesPage.tsx（页面级），本组件只负责列表主体 + 工具条。
 */
import { useState } from 'react';
import type { Role } from '@/lib/types';
import { useDeleteRole, usePermsCatalog, useRoles } from '@/hooks/queries';
import { Button } from './ui';
import { EditIcon, PlusIcon, TrashIcon } from './icons';
import { RoleDrawer } from './RoleDrawer';
import { cn } from '@/lib/cn';

/** 角色行的数据范围摘要文案，如「全部品牌 · 全部渠道」「AP/BP · 12 个渠道」。 */
function scopeSummary(role: Role): string {
  if (role.builtin) return '全部品牌 · 全部渠道';
  const brandPart = role.scope.allBrands
    ? '全部品牌'
    : role.scope.brands.length
      ? role.scope.brands.map((b) => b.toUpperCase()).join('/')
      : '未选品牌';
  const channelPart = role.scope.allChannels ? '全部渠道' : `${role.scope.channelIds.length} 个渠道`;
  return `${brandPart} · ${channelPart}`;
}

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
    <div>
      <div className="flex items-center justify-between mb-3">
        <span className="text-[12.5px] text-muted">共 {sorted.length} 个角色</span>
        <Button onClick={() => setTarget('new')}>
          <PlusIcon className="w-4 h-4" />
          新增角色
        </Button>
      </div>

      {isLoading ? (
        <div className="h-40 rounded-card bg-panel border border-line animate-pulse" />
      ) : sorted.length === 0 ? (
        <div className="section-card text-center text-muted py-[40px]">暂无角色</div>
      ) : (
        <div className="section-card !p-0 overflow-hidden">
          {/*
           * table-fixed + 显式列宽：原生 table-layout:auto 下，单元格宽度由内容撑开，
           * `truncate` 在没有约束宽度时不会真正生效（超长角色名会把整张表撑宽，导致横向滚动）。
           * 固定布局后每列宽度恒定，超出内容严格按 truncate 省略，不会撑破容器。
           */}
          <table className="w-full text-[13px] table-fixed">
            {/* 第一列给百分比而不是留空：留空时它会吃掉全部剩余宽度，一个几字符的角色名
                后面拖着一大片空白，正是「表格很空」的观感来源（与 UserManager 同一个坑）。 */}
            <colgroup>
              <col style={{ width: '28%' }} />
              <col className="hidden xl:table-column" style={{ width: 220 }} />
              <col style={{ width: 80 }} />
              <col className="hidden lg:table-column" style={{ width: 80 }} />
              <col className="hidden xl:table-column" style={{ width: 180 }} />
              <col style={{ width: 108 }} />
            </colgroup>
            <thead>
              <tr className="bg-panel-2 text-ink-2 text-[12px]">
                <th className="text-left font-semibold px-4 py-2.5">名称</th>
                <th className="text-left font-semibold px-4 py-2.5 hidden xl:table-cell">描述</th>
                <th className="text-left font-semibold px-4 py-2.5">人数</th>
                <th className="text-left font-semibold px-4 py-2.5 hidden lg:table-cell">权限数</th>
                <th className="text-left font-semibold px-4 py-2.5 hidden xl:table-cell">数据范围</th>
                <th className="text-right font-semibold px-4 py-2.5">操作</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((r) => (
                <tr key={r.id} className="border-t border-line-2 align-top">
                  <td className="px-4 py-3 min-w-0">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="font-semibold truncate min-w-0">{r.name}</span>
                      {r.builtin && (
                        <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#dbeafe] text-[#1d4ed8] flex-none">
                          内置
                        </span>
                      )}
                    </div>
                    {/* 窄屏（笔记本/侧边栏展开）把描述/权限数/数据范围折进第一列，不产生横向滚动。
                        注：断点按视口宽度算，248px 侧边栏会吃掉一截，这里断点比常规页面更保守。 */}
                    <div className="xl:hidden text-[11.5px] text-muted mt-0.5 truncate">{r.description || '（无描述）'}</div>
                    <div className="lg:hidden text-[11px] text-muted mt-0.5">{r.permCodes.length} 项权限</div>
                    <div className="xl:hidden text-[11px] text-muted mt-0.5 truncate">{scopeSummary(r)}</div>
                  </td>
                  <td className="hidden xl:table-cell px-4 py-3 text-ink-2 truncate" title={r.description || undefined}>
                    {r.description || '—'}
                  </td>
                  <td className="px-4 py-3 text-ink-2 whitespace-nowrap">{r.userCount} 人</td>
                  <td className="hidden lg:table-cell px-4 py-3 text-ink-2 whitespace-nowrap">{r.permCodes.length} 项</td>
                  <td className="hidden xl:table-cell px-4 py-3 text-ink-2 truncate" title={scopeSummary(r)}>
                    {scopeSummary(r)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-2">
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
                    {rowError?.id === r.id && <div className={cn('mt-1.5 text-[11.5px] text-down text-right')}>{rowError.message}</div>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <RoleDrawer target={target} catalog={catalog ?? []} onClose={() => setTarget(null)} />
    </div>
  );
}
