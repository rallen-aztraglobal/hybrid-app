/**
 * 新增/编辑角色抽屉（10-rbac.md）—— 仿照 ChannelDrawer/ListingDrawer 的右侧滑出面板：
 * 名称 + 描述 + 按模块分组的权限勾选树（路由权限 / 按钮权限分区展示，支持模块级全选/全不选）。
 * builtin（超级管理员）角色打开后整体只读展示，不可编辑/保存。
 */
import { useEffect, useMemo, useState } from 'react';
import type { PermCatalogModule, Role, RoleInput } from '@/lib/types';
import { useSaveRole } from '@/hooks/queries';
import { cn } from '@/lib/cn';
import { Button, SectionHeading } from './ui';
import { CloseIcon, SaveCheckIcon } from './icons';
import { Field } from './FormField';

export function RoleDrawer({
  target,
  catalog,
  onClose,
}: {
  /** null=关闭；'new'=新增；Role=编辑（builtin 时只读展示） */
  target: 'new' | Role | null;
  catalog: PermCatalogModule[];
  onClose: () => void;
}) {
  const save = useSaveRole();
  const open = target !== null;
  const editing = target && target !== 'new' ? target : null;
  const readOnly = !!editing?.builtin;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [permCodes, setPermCodes] = useState<Set<string>>(new Set());
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (!target) return;
    setErrorMsg(null);
    if (target === 'new') {
      setName('');
      setDescription('');
      setPermCodes(new Set());
    } else {
      setName(target.name);
      setDescription(target.description);
      setPermCodes(new Set(target.permCodes));
    }
  }, [target]);

  function toggle(code: string) {
    if (readOnly) return;
    setPermCodes((prev) => {
      const next = new Set(prev);
      if (next.has(code)) next.delete(code);
      else next.add(code);
      return next;
    });
  }

  function toggleModule(mod: PermCatalogModule, checked: boolean) {
    if (readOnly) return;
    setPermCodes((prev) => {
      const next = new Set(prev);
      for (const p of mod.perms) {
        if (checked) next.add(p.code);
        else next.delete(p.code);
      }
      return next;
    });
  }

  async function submit() {
    if (readOnly) return;
    const trimmed = name.trim();
    if (!trimmed) {
      setErrorMsg('角色名称必填');
      return;
    }
    setErrorMsg(null);
    const input: RoleInput = { name: trimmed, description: description.trim(), permCodes: [...permCodes] };
    try {
      await save.mutateAsync({ id: editing ? editing.id : undefined, input });
      onClose();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : '保存失败');
    }
  }

  return (
    <>
      <div
        onClick={onClose}
        className={cn('fixed inset-0 z-40 transition', open ? 'opacity-100 visible' : 'opacity-0 invisible')}
        style={{ background: 'rgba(15,23,42,.45)', backdropFilter: 'blur(2px)' }}
      />
      <aside
        className={cn(
          'fixed top-0 right-0 h-full w-[520px] max-w-[94vw] z-50 flex flex-col bg-bg shadow-lg2 transition-transform duration-300',
          open ? 'translate-x-0' : 'translate-x-full',
        )}
        style={{ transitionTimingFunction: 'cubic-bezier(.4,0,.1,1)' }}
      >
        <div className="flex-none flex items-center gap-3 px-6 py-5 bg-panel border-b border-line">
          <div className="flex-1">
            <div className="text-[16px] font-bold">{editing ? `${readOnly ? '查看' : '编辑'}角色 · ${editing.name}` : '新增角色'}</div>
            {readOnly && <div className="text-[12px] text-muted mt-0.5">内置角色不可编辑/删除，仅供查看</div>}
          </div>
          <button
            onClick={onClose}
            className="grid place-items-center w-[38px] h-[38px] rounded-[10px] border border-line text-ink-2 hover:bg-bg"
          >
            <CloseIcon className="w-[18px] h-[18px]" />
          </button>
        </div>

        <div className="overflow-auto px-6 py-5 flex-1 flex flex-col gap-5">
          <div className="section-card">
            <SectionHeading num={1}>基本信息</SectionHeading>
            <Field label="角色名称" required>
              <input
                className="field-input"
                value={name}
                disabled={readOnly}
                onChange={(e) => setName(e.target.value)}
                placeholder="如：客服"
              />
            </Field>
            <Field label="描述" hint="选填">
              <input
                className="field-input"
                value={description}
                disabled={readOnly}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="该角色的职责说明"
              />
            </Field>
          </div>

          <div className="section-card">
            <SectionHeading num={2}>权限（按模块分组，路由权限 / 按钮权限）</SectionHeading>
            <div className="flex flex-col gap-4">
              {catalog.map((mod) => (
                <ModuleGroup
                  key={mod.module}
                  mod={mod}
                  selected={permCodes}
                  readOnly={readOnly}
                  onToggle={toggle}
                  onToggleModule={(checked) => toggleModule(mod, checked)}
                />
              ))}
              {catalog.length === 0 && <div className="text-[12.5px] text-muted">权限点清单加载中…</div>}
            </div>
          </div>
        </div>

        <div className="flex-none flex items-center justify-between gap-[10px] px-6 py-[14px] bg-panel border-t border-line">
          <div className="text-[12px] flex-1 min-w-0 truncate">{errorMsg && <span className="text-down">{errorMsg}</span>}</div>
          <div className="flex gap-[10px] flex-none">
            <Button onClick={onClose}>{readOnly ? '关闭' : '取消'}</Button>
            {!readOnly && (
              <Button variant="primary" onClick={submit} disabled={save.isPending}>
                <SaveCheckIcon />
                {save.isPending ? '保存中…' : '保存角色'}
              </Button>
            )}
          </div>
        </div>
      </aside>
    </>
  );
}

function ModuleGroup({
  mod,
  selected,
  readOnly,
  onToggle,
  onToggleModule,
}: {
  mod: PermCatalogModule;
  selected: Set<string>;
  readOnly: boolean;
  onToggle: (code: string) => void;
  onToggleModule: (checked: boolean) => void;
}) {
  const routePerms = useMemo(() => mod.perms.filter((p) => p.kind === 'route'), [mod.perms]);
  const buttonPerms = useMemo(() => mod.perms.filter((p) => p.kind === 'button'), [mod.perms]);
  const selectedCount = mod.perms.filter((p) => selected.has(p.code)).length;
  const allSelected = selectedCount === mod.perms.length && mod.perms.length > 0;
  const someSelected = selectedCount > 0 && !allSelected;

  return (
    <div className="rounded-[10px] border border-line p-3">
      <label className="flex items-center gap-2 mb-2 cursor-pointer select-none">
        <input
          type="checkbox"
          checked={allSelected}
          ref={(el) => {
            if (el) el.indeterminate = someSelected;
          }}
          disabled={readOnly}
          onChange={(e) => onToggleModule(e.target.checked)}
          className="w-[15px] h-[15px] accent-[var(--brand)]"
        />
        <span className="text-[13px] font-bold">{mod.label}</span>
        <span className="text-[11px] text-muted">
          {selectedCount}/{mod.perms.length}
        </span>
      </label>
      <div className="grid grid-cols-2 gap-3 pl-[23px]">
        <PermList title="路由权限" perms={routePerms} selected={selected} readOnly={readOnly} onToggle={onToggle} />
        <PermList title="按钮权限" perms={buttonPerms} selected={selected} readOnly={readOnly} onToggle={onToggle} />
      </div>
    </div>
  );
}

function PermList({
  title,
  perms,
  selected,
  readOnly,
  onToggle,
}: {
  title: string;
  perms: PermCatalogModule['perms'];
  selected: Set<string>;
  readOnly: boolean;
  onToggle: (code: string) => void;
}) {
  if (perms.length === 0) return <div />;
  return (
    <div>
      <div className="text-[11px] font-semibold text-muted mb-1">{title}</div>
      <div className="flex flex-col gap-1">
        {perms.map((p) => (
          <label key={p.code} className="flex items-start gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={selected.has(p.code)}
              disabled={readOnly}
              onChange={() => onToggle(p.code)}
              className="w-[14px] h-[14px] mt-0.5 accent-[var(--brand)]"
            />
            <span className="text-[12px] text-ink-2 leading-[1.4]">
              <span className="font-mono text-[11px] text-muted">{p.code}</span>
              <br />
              {p.label}
            </span>
          </label>
        ))}
      </div>
    </div>
  );
}
