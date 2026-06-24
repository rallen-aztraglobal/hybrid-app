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
import { useChannels } from '@/hooks/queries';
import { BRAND_META } from '@/lib/brands';
import { Button, Note } from '@/components/ui';
import { InfoIcon, ShieldIcon } from '@/components/icons';
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
    </section>
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
        <select
          className="field-input mono flex-1"
          value={channelId}
          onChange={(e) => {
            setChannelId(e.target.value);
            setResult(null);
          }}
        >
          <option value="">选择渠道（applicationId）…</option>
          {options.map((c) => (
            <option key={c.id} value={c.id}>
              {c.applicationId} · {BRAND_META[c.brandCode].name} · {c.appName}
            </option>
          ))}
        </select>
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
