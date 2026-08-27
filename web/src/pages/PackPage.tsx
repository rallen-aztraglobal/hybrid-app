/**
 * 打包中心（ADR-0008）—— 复刻原型「打包中心」视图并落地真实打包链路：
 * 品牌 Tab + 小渠道多选 + versionName(X.Y.Z) + 任务名(默认可改) + 测试事件开关
 *   → 触发 POST /api/build/jobs（useSubmitBuildJob）
 *   → 轮询 GET /api/build/jobs/:id/logs 展示实时进度/日志（useBuildLogStream）。
 * 完成后引导去「构建记录」页下载 APK。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useChannels, useSubmitBuildJob } from '@/hooks/queries';
import { useScopedBrands } from '@/hooks/useScopedBrands';
import { useBuildLogStream } from '@/hooks/useBuildLogStream';
import { useUiStore } from '@/store/uiStore';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { iconInitials } from '@/lib/brands';
import { defaultJobName, validateVersionName } from '@/lib/validation';
import type { Channel } from '@/lib/types';
import { BrandTabs } from '@/components/BrandTabs';
import { AppIcon, Button, SectionHeading, Switch } from '@/components/ui';
import { PlayIcon } from '@/components/icons';
import { cn } from '@/lib/cn';

const DEFAULT_VERSION = '1.0.0';

export function PackPage() {
  const navigate = useNavigate();
  const { brands, brand } = useScopedBrands();
  const { data: channels } = useChannels();
  const submit = useSubmitBuildJob();
  const log = useBuildLogStream();
  const canSubmit = useAuthStore((s) => s.hasPerm(PERM.BUILD_SUBMIT));

  const currentBrand = useUiStore((s) => s.currentBrand);
  const setCurrentBrand = useUiStore((s) => s.setCurrentBrand);

  const list = useMemo<Channel[]>(
    () => (channels ?? []).filter((c) => c.brandCode === currentBrand && c.status !== 'archived'),
    [channels, currentBrand],
  );

  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [testEvents, setTestEvents] = useState(false);
  const [versionName, setVersionName] = useState(DEFAULT_VERSION);
  // 任务名：用户改过就别再被默认值覆盖。
  const [jobName, setJobName] = useState('');
  const [jobNameTouched, setJobNameTouched] = useState(false);
  const [submitErr, setSubmitErr] = useState<string | null>(null);

  const versionErr = validateVersionName(versionName);
  const computedDefaultName = defaultJobName(currentBrand, versionErr ? DEFAULT_VERSION : versionName);

  // 切品牌：清空选择 + 重置日志，任务名回到默认。
  useEffect(() => {
    setPicked(new Set());
    setJobNameTouched(false);
    setSubmitErr(null);
    log.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentBrand]);

  // 未手改任务名时，随品牌/版本联动默认名。
  useEffect(() => {
    if (!jobNameTouched) setJobName(computedDefaultName);
  }, [computedDefaultName, jobNameTouched]);

  function toggle(id: string) {
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleAll() {
    setPicked((prev) => (prev.size < list.length ? new Set(list.map((c) => c.id)) : new Set()));
  }

  const selected = list.filter((c) => picked.has(c.id));
  const canRun = canSubmit && selected.length > 0 && !versionErr && !submit.isPending && !log.streaming;

  async function run() {
    setSubmitErr(null);
    if (!selected.length) {
      setSubmitErr('请先勾选至少一个小渠道');
      return;
    }
    if (versionErr) {
      setSubmitErr(versionErr);
      return;
    }
    log.reset();
    try {
      const job = await submit.mutateAsync({
        brandCode: currentBrand,
        flavors: selected.map((c) => c.flavorName),
        versionName: versionName.trim(),
        jobName: jobName.trim() || undefined,
        testEvents,
      });
      // 任务已入队，开始拉日志流。
      log.start(job.id);
    } catch (err) {
      setSubmitErr(err instanceof Error ? err.message : '触发打包失败');
    }
  }

  return (
    <section>
      {brands && <BrandTabs brands={brands} current={currentBrand} onSelect={setCurrentBrand} />}

      <div className="grid gap-[18px] items-start" style={{ gridTemplateColumns: '1fr 380px' }}>
        {/* 左：渠道多选 */}
        <div>
          <div className="flex items-center gap-3 mb-3">
            <strong className="text-[13px]">选择要打包的小渠道</strong>
            <span className="text-[12px] text-muted">
              已选 {picked.size} / {list.length}
            </span>
            <div className="ml-auto">
              <button className="chip" onClick={toggleAll}>
                全选 / 取消
              </button>
            </div>
          </div>
          <div className="flex flex-col gap-2 max-h-[560px] overflow-auto pr-1">
            {list.map((c) => {
              const sel = picked.has(c.id);
              return (
                <div
                  key={c.id}
                  onClick={() => toggle(c.id)}
                  className={cn(
                    'flex items-center gap-3 px-[13px] py-[11px] border rounded-[11px] bg-panel cursor-pointer transition',
                    sel
                      ? 'border-brand bg-[rgba(99,102,241,.05)] shadow-[0_0_0_1px_var(--brand)_inset]'
                      : 'border-line hover:bg-panel-2 hover:border-[#dfe6f0]',
                  )}
                >
                  <span
                    className={cn(
                      'grid place-items-center w-5 h-5 rounded-md border-[1.5px] flex-none transition',
                      sel ? 'bg-brand border-brand' : 'border-[#cbd5e1]',
                    )}
                  >
                    {sel && (
                      <svg viewBox="0 0 24 24" className="w-[13px] h-[13px] text-white" fill="none" stroke="currentColor" strokeWidth={3}>
                        <path d="M20 6 9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    )}
                  </span>
                  <AppIcon
                    initials={iconInitials(c.appName, c.brandCode)}
                    hex={brand?.accentColor ?? '#6366f1'}
                    src={c.iconMasterUrl}
                    size={34}
                    radius={9}
                  />
                  <span className="min-w-0">
                    <span className="block text-[13.5px] font-semibold truncate">{c.appName}</span>
                    <span className="block text-[11px] text-muted font-mono">{c.flavorName}</span>
                  </span>
                  {c.status === 'disabled' && (
                    <span className="ml-auto text-[10px] text-[#64748b] bg-[#f1f5f9] px-2 py-0.5 rounded-full">
                      已停用
                    </span>
                  )}
                </div>
              );
            })}
            {list.length === 0 && <div className="text-center text-muted py-10">该品牌暂无渠道</div>}
          </div>
        </div>

        {/* 右：配置 + 终端 */}
        <div className="sticky top-0">
          <div className="section-card">
            <SectionHeading num="⚙">打包配置</SectionHeading>

            {/* 版本号 */}
            <div className="mb-[13px]">
              <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                版本号 versionName <span className="text-down">*</span>{' '}
                <span className="font-normal text-muted text-[11.5px]">X.Y.Z</span>
              </label>
              <input
                className="field-input mono"
                value={versionName}
                placeholder="1.0.3"
                onChange={(e) => setVersionName(e.target.value)}
              />
              {versionErr && <div className="mt-1 text-[12px] text-down">{versionErr}</div>}
            </div>

            {/* 任务名 */}
            <div className="mb-[13px]">
              <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">
                任务名{' '}
                <span className="font-normal text-muted text-[11.5px]">默认 品牌-版本-时间，可改</span>
              </label>
              <input
                className="field-input mono"
                value={jobName}
                placeholder={computedDefaultName}
                onChange={(e) => {
                  setJobName(e.target.value);
                  setJobNameTouched(true);
                }}
              />
              {jobNameTouched && (
                <button
                  className="mt-1 text-[11.5px] text-brand hover:underline"
                  onClick={() => {
                    setJobNameTouched(false);
                    setJobName(computedDefaultName);
                  }}
                >
                  恢复默认任务名
                </button>
              )}
            </div>

            {/* 测试事件 */}
            <div className="flex items-center gap-[10px] p-[11px_13px] bg-panel-2 border border-line rounded-[10px] mb-[14px]">
              <Switch checked={testEvents} onChange={setTestEvents} />
              <div>
                <div className="text-[13px] font-semibold">测试事件</div>
                <div className="text-[11.5px] text-muted">首次安装一次性发送全部 AppsFlyer + Adjust 事件（Adjust 走 Sandbox，见测试控制台；仅对已绑定 Adjust 的渠道生效）</div>
              </div>
            </div>

            <div className="mb-[13px]">
              <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">构建类型</label>
              <input className="field-input" value="Release" readOnly tabIndex={-1} />
            </div>

            {canSubmit ? (
              <Button variant="primary" className="w-full justify-center mt-1" disabled={!canRun} onClick={run}>
                <PlayIcon />
                {submit.isPending ? '提交中…' : log.streaming ? '打包中…' : `开始打包 (${picked.size})`}
              </Button>
            ) : (
              <div className="mt-1 text-center text-[12px] text-muted py-2">当前账号无打包权限（build:submit）</div>
            )}
            {submitErr && <div className="mt-2 text-[12px] text-down">{submitErr}</div>}
          </div>

          <Terminal
            lines={log.lines}
            brand={currentBrand}
            streaming={log.streaming}
            done={!log.streaming && log.jobId != null}
            status={log.status}
            onGotoBuilds={() => navigate('/builds')}
          />
        </div>
      </div>
    </section>
  );
}

function Terminal({
  lines,
  brand,
  streaming,
  done,
  status,
  onGotoBuilds,
}: {
  lines: string[];
  brand: string;
  streaming: boolean;
  done: boolean;
  status: string | null;
  onGotoBuilds: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.scrollTo({ top: ref.current.scrollHeight });
  }, [lines]);

  return (
    <div className="mt-[14px]">
      <div
        ref={ref}
        className="rounded-[14px] p-4 font-mono text-[12px] shadow-md2 min-h-[160px] max-h-[320px] overflow-auto leading-[1.7]"
        style={{ background: '#0c1426', color: '#94e2b5' }}
      >
        {lines.length === 0 ? (
          <>
            <span className="text-[#64748b]"># hybrid-pack release --brand {brand}</span>
            <br />
            {streaming ? (
              '排队中…'
            ) : (
              <span className="text-[#64748b]">尚未发起打包 — 勾选渠道后点上方「开始打包」</span>
            )}
          </>
        ) : (
          lines.map((l, i) => <TermLine key={i} text={l} />)
        )}
        {streaming && <span className="inline-block w-2 h-3 ml-0.5 bg-[#94e2b5] animate-pulse align-middle" />}
      </div>
      {done && (
        <div
          className={cn(
            'mt-2 flex items-center gap-2 text-[12px] rounded-[10px] px-3 py-2',
            status === 'failed' ? 'bg-[#fee2e2] text-[#b91c1c]' : 'bg-[#dcfce7] text-[#15803d]',
          )}
        >
          {status === 'failed' ? '构建失败，查看日志排查' : '构建完成'}
          {status !== 'failed' && (
            <button className="ml-auto text-brand font-semibold hover:underline" onClick={onGotoBuilds}>
              去构建记录下载 APK →
            </button>
          )}
        </div>
      )}
    </div>
  );
}

function TermLine({ text }: { text: string }) {
  let cls = '';
  if (text.startsWith('#')) cls = 'text-[#64748b]';
  else if (text.startsWith('→')) cls = 'text-[#60a5fa]';
  else if (text.trimStart().startsWith('✓') || text.startsWith('✓')) cls = 'text-[#4ade80]';
  else if (/失败|error|failed/i.test(text)) cls = 'text-[#f87171]';
  return <div className={cls}>{text}</div>;
}
