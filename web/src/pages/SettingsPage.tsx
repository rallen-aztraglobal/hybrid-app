/**
 * 系统设置 —— 在原型占位基础上做成可用页：
 *  - 账号与权限说明（admin/operator/viewer，RBAC）；
 *  - 探针端点 / CDN 快照配置（展示当前值）；
 *  - **运行时配置预览工具**：输入 PAL_CODE，按三级取用（实时→缓存→兜底，ADR-0002）
 *    解析该渠道当前会拿到的域名，并标注命中来源，便于运营核对「域名热更是否生效」；
 *  - **安全红线提示**：keystore / 签名口令永不进后台（CLAUDE.md #4）。
 */
import { useMemo, useState } from 'react';
import { resolveAppConfig, type ResolveResult } from '@/lib/runtimeConfig';
import { useAuthStore } from '@/store/authStore';
import { useChannels, useCreateStore, useDeleteStore, useStores, useUpdateStore } from '@/hooks/queries';
import { BRAND_META } from '@/lib/brands';
import type { Store } from '@/lib/types';
import { ApiError } from '@/lib/api';
import { Button, Note, Select, Switch } from '@/components/ui';
import { EditIcon, InfoIcon, PlusIcon, ShieldIcon, TrashIcon } from '@/components/icons';
import { cn } from '@/lib/cn';

export function SettingsPage() {
  const user = useAuthStore((s) => s.user);

  return (
    <section className="flex flex-col gap-4 max-w-[760px]">
      {/* 账号与权限 */}
      <div className="section-card">
        <h3 className="text-[13px] font-bold mb-3">账号与权限（RBAC）</h3>
        <div className="text-[12.5px] text-ink-2 mb-3">
          当前登录：<b>{user?.username}</b>（{user?.role}）
        </div>
        <div className="grid grid-cols-3 gap-3">
          {[
            { role: 'admin', desc: '全部权限：账号、域名、渠道、打包' },
            { role: 'operator', desc: '渠道 CRUD、图标、触发打包' },
            { role: 'viewer', desc: '只读：查看渠道与构建记录' },
          ].map((r) => (
            <div
              key={r.role}
              className={cn(
                'rounded-[10px] border p-3',
                user?.role === r.role ? 'border-brand bg-[rgba(99,102,241,.05)]' : 'border-line bg-panel-2',
              )}
            >
              <div className="font-semibold text-[13px]">{r.role}</div>
              <div className="text-[11.5px] text-muted mt-1">{r.desc}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 探针 / CDN */}
      <div className="section-card">
        <h3 className="text-[13px] font-bold mb-3">探针端点 · CDN 快照</h3>
        <KeyVal k="业务特征探针 probePath" v="/healthz" hint="供 APK 校验「确实是我们的站点」（ADR-0003）" />
        <KeyVal
          k="运行时配置端点"
          v="GET /api/app/config?palcode="
          hint="部署在抗封 CDN / 对象存储，保存域名即生成静态快照（ADR-0002）"
        />
        <KeyVal
          k="中立连通性探针"
          v="gstatic / cloudflare generate_204"
          hint="裁决「域名故障 vs 本机网络」，绝不乱换（ADR-0003）"
        />
      </div>

      {/* 应用商店 */}
      <StoresManager />

      {/* 运行时配置预览 */}
      <RuntimePreview />

      {/* 安全红线 */}
      <div
        className="rounded-card p-[18px] border"
        style={{ borderColor: 'rgba(239,68,68,.3)', background: 'rgba(239,68,68,.04)' }}
      >
        <h3 className="flex items-center gap-2 text-[13px] font-bold mb-2 text-[#b91c1c]">
          <ShieldIcon className="w-[18px] h-[18px]" />
          安全红线
        </h3>
        <div className="text-[12.5px] text-ink-2 leading-[1.7]">
          <b>keystore 与签名口令永不进入后台 / 上传 / 任何回传路径</b>（CLAUDE.md #4）。签名密钥只留本地{' '}
          <span className="mono">local.properties</span> 指向的 keystore，由 CLI 在本机调用 gradlew 时使用。后台与本前端
          <b>不采集、不展示、不存储</b>任何签名材料。
        </div>
      </div>

      {/* 合规署名：上架包 AB 面网关（09-listing.md / ADR-0014）用 DB-IP 做 GeoIP 国家解析，
          免费库的许可要求在此处声明署名，不显眼但必须存在。 */}
      <div className="text-center text-[11px] text-muted pt-1 pb-2">
        IP 地理数据来自{' '}
        <a
          href="https://db-ip.com"
          target="_blank"
          rel="noopener noreferrer"
          className="hover:underline hover:text-ink-2"
        >
          DB-IP
        </a>
        （db-ip.com），CC BY 4.0
      </div>
    </section>
  );
}

const STORE_CODE_RE = /^[a-z][a-z0-9]*$/;

/**
 * 应用商店管理 —— 渠道包发布商店清单（华为 / 应用宝 / Google Play 等）。
 * 新增渠道时选中某商店，会把其 code 作为后缀拼入 flavor（下划线连接），
 * 派生包名随之带上点号后缀（见 lib/brands.ts deriveApplicationId）。
 * code 一经创建不可改（会影响既有渠道的 flavor/包名语义）；删除被引用的商店会被后端拒绝。
 */
function StoresManager() {
  const { data: stores, isLoading } = useStores();
  const createStore = useCreateStore();
  const updateStore = useUpdateStore();
  const deleteStore = useDeleteStore();

  const [adding, setAdding] = useState(false);
  const [newCode, setNewCode] = useState('');
  const [newName, setNewName] = useState('');
  const [newSort, setNewSort] = useState('0');
  const [addError, setAddError] = useState<string | null>(null);

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState('');
  const [editSort, setEditSort] = useState('0');

  const [rowError, setRowError] = useState<{ id: number; message: string } | null>(null);

  const sorted = useMemo(() => (stores ?? []).slice().sort((a, b) => a.sort - b.sort), [stores]);

  function startEdit(s: Store) {
    setEditingId(s.id);
    setEditName(s.name);
    setEditSort(String(s.sort));
    setRowError(null);
  }

  async function submitAdd() {
    setAddError(null);
    const code = newCode.trim();
    const name = newName.trim();
    if (!code || !STORE_CODE_RE.test(code)) {
      setAddError('商店代号须小写字母开头、仅含字母数字（如 hw）');
      return;
    }
    if (!name) {
      setAddError('商店名称必填');
      return;
    }
    if (sorted.some((s) => s.code.toLowerCase() === code.toLowerCase())) {
      setAddError(`商店代号「${code}」已存在`);
      return;
    }
    try {
      await createStore.mutateAsync({ code, name, sort: Number(newSort) || 0 });
      setAdding(false);
      setNewCode('');
      setNewName('');
      setNewSort('0');
    } catch (err) {
      setAddError(err instanceof Error ? err.message : '新增失败');
    }
  }

  async function submitEdit(id: number) {
    setRowError(null);
    const name = editName.trim();
    if (!name) {
      setRowError({ id, message: '商店名称必填' });
      return;
    }
    try {
      await updateStore.mutateAsync({ id, input: { name, sort: Number(editSort) || 0 } });
      setEditingId(null);
    } catch (err) {
      setRowError({ id, message: err instanceof Error ? err.message : '保存失败' });
    }
  }

  async function toggleStatus(s: Store) {
    setRowError(null);
    try {
      await updateStore.mutateAsync({
        id: s.id,
        input: { status: s.status === 'enabled' ? 'disabled' : 'enabled' },
      });
    } catch (err) {
      setRowError({ id: s.id, message: err instanceof Error ? err.message : '操作失败' });
    }
  }

  async function remove(s: Store) {
    setRowError(null);
    if (!confirm(`确认删除商店「${s.name}」？`)) return;
    try {
      await deleteStore.mutateAsync(s.id);
    } catch (err) {
      // 后端在商店被渠道引用时返回冲突（409），给运营友好提示而非原始错误信息。
      const conflict = err instanceof ApiError && err.status === 409;
      setRowError({
        id: s.id,
        message: conflict ? '该商店已被渠道使用，请改为停用' : err instanceof Error ? err.message : '删除失败',
      });
    }
  }

  return (
    <div className="section-card">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-[13px] font-bold">应用商店</h3>
        <Button onClick={() => setAdding((v) => !v)}>
          <PlusIcon className="w-4 h-4" />
          新增商店
        </Button>
      </div>
      <p className="text-[12px] text-muted mb-3">
        新增渠道时可选择发布到的应用商店；选中后其代号会作为后缀拼入 flavor（如 <span className="mono">_hw</span>），
        派生包名相应带上点号后缀。代号一经创建不可修改；被渠道引用的商店无法删除，请改为停用。
      </p>

      {adding && (
        <div className="flex flex-wrap items-end gap-2 p-3 mb-3 rounded-[10px] border border-line bg-panel-2">
          <div className="flex-1 min-w-[100px]">
            <label className="block text-[11.5px] text-muted mb-1">代号 code</label>
            <input
              className="field-input mono"
              placeholder="hw"
              value={newCode}
              onChange={(e) => setNewCode(e.target.value)}
            />
          </div>
          <div className="flex-1 min-w-[140px]">
            <label className="block text-[11.5px] text-muted mb-1">名称</label>
            <input
              className="field-input"
              placeholder="华为应用市场"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
            />
          </div>
          <div className="w-20">
            <label className="block text-[11.5px] text-muted mb-1">排序</label>
            <input
              className="field-input mono"
              value={newSort}
              onChange={(e) => setNewSort(e.target.value)}
            />
          </div>
          <Button variant="primary" onClick={submitAdd} disabled={createStore.isPending}>
            {createStore.isPending ? '保存中…' : '确定'}
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
      {!isLoading && sorted.length === 0 && (
        <div className="text-[12.5px] text-muted py-2">暂无应用商店，新增渠道时将不显示商店选项（视为默认包名）。</div>
      )}

      <div className="flex flex-col gap-2">
        {sorted.map((s) => (
          <div key={s.id} className="rounded-[10px] border border-line p-3">
            {editingId === s.id ? (
              <div className="flex flex-wrap items-end gap-2">
                <div className="text-[12px] text-muted mono px-2 py-1.5 rounded-md bg-panel-2 flex-none">
                  {s.code}
                </div>
                <div className="flex-1 min-w-[140px]">
                  <label className="block text-[11.5px] text-muted mb-1">名称</label>
                  <input className="field-input" value={editName} onChange={(e) => setEditName(e.target.value)} />
                </div>
                <div className="w-20">
                  <label className="block text-[11.5px] text-muted mb-1">排序</label>
                  <input className="field-input mono" value={editSort} onChange={(e) => setEditSort(e.target.value)} />
                </div>
                <Button variant="primary" onClick={() => submitEdit(s.id)} disabled={updateStore.isPending}>
                  保存
                </Button>
                <Button onClick={() => setEditingId(null)}>取消</Button>
              </div>
            ) : (
              <div className="flex items-center gap-3">
                <span className="text-[12px] font-mono px-2 py-0.5 rounded-md bg-panel-2 text-ink-2 flex-none">
                  {s.code}
                </span>
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-semibold truncate">{s.name}</div>
                  <div className="text-[11px] text-muted">排序 {s.sort}</div>
                </div>
                <span
                  className={cn(
                    'text-[11px] font-semibold px-2 py-0.5 rounded-full flex-none',
                    s.status === 'enabled' ? 'text-[#15803d] bg-[#dcfce7]' : 'text-[#64748b] bg-[#f1f5f9]',
                  )}
                >
                  {s.status === 'enabled' ? '启用' : '停用'}
                </span>
                <Switch checked={s.status === 'enabled'} onChange={() => toggleStatus(s)} />
                <button
                  title="编辑"
                  onClick={() => startEdit(s)}
                  className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 hover:bg-bg hover:text-ink transition flex-none"
                >
                  <EditIcon className="w-[15px] h-[15px]" />
                </button>
                <button
                  title="删除"
                  onClick={() => remove(s)}
                  className="grid place-items-center w-[30px] h-[30px] rounded-lg border border-line bg-panel text-ink-2 hover:text-down hover:border-[#fecaca] hover:bg-[#fef2f2] transition flex-none"
                >
                  <TrashIcon className="w-[15px] h-[15px]" />
                </button>
              </div>
            )}
            {rowError?.id === s.id && <div className="mt-2 text-[12px] text-down">{rowError.message}</div>}
          </div>
        ))}
      </div>
    </div>
  );
}

function RuntimePreview() {
  const { data: channels } = useChannels();
  const [channelId, setChannelId] = useState<string>('');
  const [result, setResult] = useState<ResolveResult | null>(null);
  const [busy, setBusy] = useState(false);

  // ADR-0009：下拉按渠道（展示 applicationId）选择，不再手输 palcode。
  const options = useMemo(
    () => (channels ?? []).filter((c) => c.status !== 'archived').slice().sort((a, b) => a.applicationId.localeCompare(b.applicationId)),
    [channels],
  );
  const selected = options.find((c) => c.id === channelId);

  async function preview() {
    if (!selected) return;
    setBusy(true);
    try {
      // 三级取用：实时拉取(成功即写缓存) → 缓存 → 编译期兜底。
      // 当前后端 /api/app/config 仍以 palcode 为参数键；以选中渠道的 palCode 调用，
      // UI 层已按 ADR-0009 用 applicationId 选渠道。后端切 appId 键后仅改 runtimeConfig fetcher。
      const r = await resolveAppConfig(selected.palCode.trim());
      setResult(r);
    } finally {
      setBusy(false);
    }
  }

  const sourceLabel: Record<ResolveResult['source'], string> = {
    remote: '① 实时拉取（已写本地缓存）',
    cache: '② 本地缓存（上次成功）',
    bootstrap: '③ 编译期兜底（从未成功拉取）',
  };

  return (
    <div className="section-card">
      <h3 className="text-[13px] font-bold mb-1">运行时配置预览（按渠道 applicationId 解析域名）</h3>
      <p className="text-[12px] text-muted mb-3">
        选择渠道（按 applicationId），模拟 APK 端 DomainResolver 三级取用，核对该渠道当前会拿到的域名清单与命中来源。
        域名不在前端硬编码——实时拉取失败才回落缓存 / 兜底（ADR-0002 / ADR-0009）。
      </p>
      <div className="flex gap-2 mb-3">
        <Select
          className="mono flex-1"
          value={channelId}
          placeholder="选择渠道（applicationId）…"
          onChange={(v) => {
            setChannelId(v);
            setResult(null);
          }}
          options={options.map((c) => ({
            value: c.id,
            label: `${c.applicationId} · ${BRAND_META[c.brandCode].name} · ${c.appName}`,
          }))}
        />
        <Button variant="primary" onClick={preview} disabled={busy || !selected}>
          {busy ? '解析中…' : '解析'}
        </Button>
      </div>
      {selected && (
        <div className="text-[11.5px] text-muted mb-3 font-mono">
          flavor <b className="text-ink-2">{selected.flavorName}</b> · palcode{' '}
          <b className="text-ink-2">{selected.palCode}</b>（仅 URL 参数，非解析键）
        </div>
      )}

      {result && (
        <div className="rounded-[10px] border border-line bg-panel-2 p-3">
          <div className="flex items-center gap-2 mb-2">
            <span
              className={cn(
                'text-[11px] font-semibold px-2 py-0.5 rounded-full',
                result.source === 'remote'
                  ? 'text-[#15803d] bg-[#dcfce7]'
                  : result.source === 'cache'
                    ? 'text-[#92681a] bg-[#fef3c7]'
                    : 'text-[#b91c1c] bg-[#fee2e2]',
              )}
            >
              {sourceLabel[result.source]}
            </span>
            {result.remoteError && (
              <span className="text-[11.5px] text-muted">实时拉取失败：{result.remoteError}</span>
            )}
          </div>
          {result.config && result.config.domains.length > 0 ? (
            <ol className="list-decimal list-inside text-[12.5px] font-mono text-ink-2 space-y-1">
              {result.config.domains.map((d, i) => (
                <li key={d}>
                  {i === 0 ? '主 ' : `备${i} `}
                  {d}
                </li>
              ))}
            </ol>
          ) : (
            <div className="text-[12.5px] text-muted">
              无可用域名（首启无网且无缓存）。APK 此时提示「请联网获取配置」，不白屏、不硬编码业务域名。
            </div>
          )}
        </div>
      )}

      <Note>
        <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
        <div>
          后端未就绪时此处会走「兜底」分支（无缓存）；接好 <span className="mono">/api/app/config</span> 后即可看到「实时拉取」并自动写入本地缓存。
        </div>
      </Note>
    </div>
  );
}

function KeyVal({ k, v, hint }: { k: string; v: string; hint?: string }) {
  return (
    <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 py-2 border-b border-line-2 last:border-0">
      <span className="text-[12.5px] text-muted w-[180px] flex-none">{k}</span>
      <span className="font-mono text-[12.5px] text-ink-2">{v}</span>
      {hint && <span className="text-[11.5px] text-muted w-full pl-[180px]">{hint}</span>}
    </div>
  );
}
