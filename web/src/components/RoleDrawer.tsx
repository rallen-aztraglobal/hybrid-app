/**
 * 新增/编辑角色抽屉（10-rbac.md）—— 仿照 ChannelDrawer/ListingDrawer 的右侧滑出面板：
 *  1) 名称 + 描述；
 *  2) 按模块分组的权限勾选树（路由权限 / 按钮权限分区展示，支持模块级全选/全不选）；
 *  3) 数据范围（scope）：品牌范围 + 渠道范围，见「数据权限」一节 —— 与权限点正交，
 *     权限点回答「能不能做」，scope 回答「对哪些数据做」。
 * builtin（超级管理员）角色打开后整体只读展示，不可编辑/保存，数据范围恒展示「全部品牌 · 全部渠道」。
 */
import { useEffect, useMemo, useState } from 'react';
import type { BrandCode, Channel, PermCatalogModule, Role, RoleInput, RoleScope } from '@/lib/types';
import { FULL_ROLE_SCOPE } from '@/lib/types';
import { useBrands, useChannels, useSaveRole } from '@/hooks/queries';
import { BRAND_META, BRAND_ORDER } from '@/lib/brands';
import { cn } from '@/lib/cn';
import { Button, SectionHeading } from './ui';
import { CloseIcon, SaveCheckIcon, SearchIcon } from './icons';
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
  const { data: brands } = useBrands();
  const { data: channels } = useChannels();
  const open = target !== null;
  const editing = target && target !== 'new' ? target : null;
  const readOnly = !!editing?.builtin;

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [permCodes, setPermCodes] = useState<Set<string>>(new Set());
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  // ---- 数据范围（scope）本地态 ----
  const [scopeAllBrands, setScopeAllBrands] = useState(true);
  const [scopeBrands, setScopeBrands] = useState<Set<BrandCode>>(new Set());
  const [scopeAllChannels, setScopeAllChannels] = useState(true);
  const [scopeChannelIds, setScopeChannelIds] = useState<Set<string>>(new Set());
  const [channelSearch, setChannelSearch] = useState('');

  useEffect(() => {
    if (!target) return;
    setErrorMsg(null);
    setChannelSearch('');
    if (target === 'new') {
      setName('');
      setDescription('');
      setPermCodes(new Set());
      setScopeAllBrands(true);
      setScopeBrands(new Set());
      setScopeAllChannels(true);
      setScopeChannelIds(new Set());
    } else {
      setName(target.name);
      setDescription(target.description);
      setPermCodes(new Set(target.permCodes));
      const scope = target.scope ?? FULL_ROLE_SCOPE;
      setScopeAllBrands(scope.allBrands);
      setScopeBrands(new Set(scope.brands));
      setScopeAllChannels(scope.allChannels);
      setScopeChannelIds(new Set(scope.channelIds));
    }
  }, [target]);

  // 品牌范围收窄时，自动移除已不在允许品牌内的已选渠道（与后端 EffectiveScope 求交语义一致）。
  // 关键：channels 未加载完成前直接跳过——否则编辑态回显的 scopeChannelIds 会在渠道列表
  // 还没拉到时被误判成「全部不在允许品牌内」而被整体清空（找不到 = 当作不允许，是错的）。
  useEffect(() => {
    if (scopeAllBrands || !channels) return;
    setScopeChannelIds((prev) => {
      const next = new Set<string>();
      for (const id of prev) {
        const ch = channels.find((c) => c.id === id);
        if (ch && scopeBrands.has(ch.brandCode)) next.add(id);
      }
      return next.size === prev.size ? prev : next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scopeAllBrands, scopeBrands, channels]);

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

  function toggleBrand(code: BrandCode) {
    if (readOnly) return;
    setScopeBrands((prev) => {
      const next = new Set(prev);
      if (next.has(code)) next.delete(code);
      else next.add(code);
      return next;
    });
  }

  const allowedBrandsForChannelPicker = scopeAllBrands ? BRAND_ORDER : BRAND_ORDER.filter((b) => scopeBrands.has(b));

  async function submit() {
    if (readOnly) return;
    const trimmed = name.trim();
    if (!trimmed) {
      setErrorMsg('角色名称必填');
      return;
    }
    if (!scopeAllBrands && scopeBrands.size === 0) {
      setErrorMsg('指定品牌范围时，至少要勾选一个品牌（或改选「全部品牌」）');
      return;
    }
    if (!scopeAllChannels && scopeChannelIds.size === 0) {
      setErrorMsg('指定渠道范围时，至少要勾选一个渠道（或改选「范围内全部渠道」）');
      return;
    }
    setErrorMsg(null);
    const scope: RoleScope = {
      allBrands: scopeAllBrands,
      brands: scopeAllBrands ? [] : [...scopeBrands],
      allChannels: scopeAllChannels,
      channelIds: scopeAllChannels ? [] : [...scopeChannelIds],
    };
    const input: RoleInput = { name: trimmed, description: description.trim(), permCodes: [...permCodes], scope };
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
          'fixed top-0 right-0 h-full w-[560px] max-w-[94vw] z-50 flex flex-col bg-bg shadow-lg2 transition-transform duration-300',
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

          <div className="section-card">
            <SectionHeading num={3}>
              数据范围 <span className="font-normal text-muted text-[11.5px]">· 限定该角色能操作哪些品牌/渠道的数据</span>
            </SectionHeading>

            {readOnly ? (
              <div className="text-[12.5px] text-ink-2">
                <b>全部品牌 · 全部渠道</b>
                <div className="text-[11.5px] text-muted mt-1">超级管理员不受数据范围限制。</div>
              </div>
            ) : (
              <>
                {/* 品牌范围 */}
                <div className="mb-4">
                  <div className="mb-1.5 text-[12.5px] font-semibold text-ink-2">品牌范围（大渠道）</div>
                  <div className="flex gap-2 mb-2">
                    <button type="button" onClick={() => setScopeAllBrands(true)} className={cn('chip', scopeAllBrands && 'chip-on')}>
                      全部品牌（含以后新增）
                    </button>
                    <button type="button" onClick={() => setScopeAllBrands(false)} className={cn('chip', !scopeAllBrands && 'chip-on')}>
                      指定品牌
                    </button>
                  </div>
                  {!scopeAllBrands && (
                    <div className="flex flex-wrap gap-[6px]">
                      {(brands ?? BRAND_ORDER.map((code) => ({ code, name: BRAND_META[code].name }))).map((b) => (
                        <button
                          key={b.code}
                          type="button"
                          onClick={() => toggleBrand(b.code)}
                          className={cn('chip', scopeBrands.has(b.code) && 'chip-on')}
                        >
                          {b.name}（{b.code}）
                        </button>
                      ))}
                    </div>
                  )}
                  <div className="mt-1.5 text-[11px] text-muted">
                    {scopeAllBrands
                      ? '「全部品牌」是动态范围：以后新增的品牌会自动纳入，无需回来改这个角色。'
                      : '仅下方勾选的品牌生效；以后新增的品牌不会自动加入这个角色，需要手动回来补选。'}
                  </div>
                </div>

                {/* 渠道范围 */}
                <div>
                  <div className="mb-1.5 text-[12.5px] font-semibold text-ink-2">渠道范围（小渠道包）</div>
                  <div className="flex gap-2 mb-2">
                    <button type="button" onClick={() => setScopeAllChannels(true)} className={cn('chip', scopeAllChannels && 'chip-on')}>
                      范围内全部渠道（含以后新增）
                    </button>
                    <button type="button" onClick={() => setScopeAllChannels(false)} className={cn('chip', !scopeAllChannels && 'chip-on')}>
                      指定渠道
                    </button>
                  </div>
                  {!scopeAllChannels && (
                    <ChannelScopePicker
                      channels={channels ?? []}
                      allowedBrands={allowedBrandsForChannelPicker}
                      selected={scopeChannelIds}
                      onChange={setScopeChannelIds}
                      search={channelSearch}
                      onSearchChange={setChannelSearch}
                    />
                  )}
                  <div className="mt-1.5 text-[11px] text-muted">
                    {scopeAllChannels
                      ? '「范围内全部渠道」同样是动态范围：品牌范围内以后新增的渠道会自动纳入。'
                      : '仅下方勾选的渠道生效；以后新增的渠道不会自动加入，需要手动回来补选。收窄品牌范围会自动移除不在新品牌范围内的已选渠道。'}
                  </div>
                </div>
              </>
            )}
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

/**
 * 渠道范围选择器 —— 按品牌分组（仅展示 allowedBrands 内的品牌）+ 搜索框 + 「全选本品牌」，
 * 避免一次铺开几十上百个 checkbox（84 个渠道量级）。可滚动容器限高，不撑爆抽屉。
 */
function ChannelScopePicker({
  channels,
  allowedBrands,
  selected,
  onChange,
  search,
  onSearchChange,
}: {
  channels: Channel[];
  allowedBrands: BrandCode[];
  selected: Set<string>;
  onChange: (next: Set<string>) => void;
  search: string;
  onSearchChange: (v: string) => void;
}) {
  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onChange(next);
  }

  function selectAllInBrand(brandChannels: Channel[], allSelected: boolean) {
    const next = new Set(selected);
    for (const c of brandChannels) {
      if (allSelected) next.delete(c.id);
      else next.add(c.id);
    }
    onChange(next);
  }

  const q = search.trim().toLowerCase();
  const groups = allowedBrands.map((brand) => {
    const all = channels.filter((c) => c.brandCode === brand && c.status !== 'archived');
    const filtered = q
      ? all.filter((c) => [c.flavorName, c.applicationId, c.palCode, c.appName].some((f) => f.toLowerCase().includes(q)))
      : all;
    return { brand, all, filtered };
  });
  const nonEmptyGroups = groups.filter((g) => g.all.length > 0);

  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted pointer-events-none" />
          <input
            className="field-input pl-9"
            placeholder="搜索 flavor / 应用名 / 包名 / PAL_CODE…"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
        <span className="text-[11.5px] text-muted whitespace-nowrap flex-none">已选 {selected.size} 个</span>
      </div>

      {nonEmptyGroups.length === 0 ? (
        <div className="text-[12px] text-muted py-4 text-center rounded-[10px] border border-line bg-panel-2">
          {allowedBrands.length === 0 ? '请先在上方选择品牌范围' : '所选品牌下暂无渠道'}
        </div>
      ) : (
        <div className="flex flex-col gap-3 max-h-[320px] overflow-y-auto pr-1 rounded-[10px] border border-line p-3">
          {nonEmptyGroups.map(({ brand, all, filtered }) => {
            const brandSelectedCount = all.filter((c) => selected.has(c.id)).length;
            const allSelected = all.length > 0 && brandSelectedCount === all.length;
            return (
              <div key={brand}>
                <div className="flex items-center gap-2 mb-1.5">
                  <span
                    className="text-[11px] font-bold px-2 py-0.5 rounded-full flex-none"
                    style={{ background: `${BRAND_META[brand].accentColor}1a`, color: BRAND_META[brand].accentColor }}
                  >
                    {BRAND_META[brand].name}
                  </span>
                  <span className="text-[11px] text-muted">
                    {brandSelectedCount}/{all.length}
                  </span>
                  <button
                    type="button"
                    onClick={() => selectAllInBrand(all, allSelected)}
                    className="ml-auto text-[11px] text-brand hover:underline flex-none"
                  >
                    {allSelected ? '取消全选本品牌' : '全选本品牌'}
                  </button>
                </div>
                {filtered.length === 0 ? (
                  <div className="text-[11.5px] text-muted pl-1 pb-1">无匹配渠道</div>
                ) : (
                  <div className="grid grid-cols-2 gap-x-3 gap-y-1">
                    {filtered.map((c) => (
                      <label key={c.id} className="flex items-center gap-1.5 text-[12px] cursor-pointer select-none py-0.5 min-w-0">
                        <input
                          type="checkbox"
                          checked={selected.has(c.id)}
                          onChange={() => toggle(c.id)}
                          className="w-[13px] h-[13px] accent-[var(--brand)] flex-none"
                        />
                        <span className="truncate" title={`${c.appName} · ${c.applicationId}`}>
                          {c.flavorName}
                        </span>
                      </label>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
