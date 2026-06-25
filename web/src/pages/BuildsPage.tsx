/**
 * 构建记录（ADR-0008）—— 列出每个打包任务（build job）：
 * 任务名 / 品牌 / 版本 / 渠道数 / 测试事件 / 状态 / 操作人 / 时间 + 日志摘要，
 * 并对每个成功产出的 APK 给**下载链接**（nginx 静态 /apks/...）。
 */
import { useState } from 'react';
import { useBuildJobs } from '@/hooks/queries';
import { BRAND_META, BRAND_ORDER, isBrandCode } from '@/lib/brands';
import type { BrandCode, BuildArtifact, BuildJob } from '@/lib/types';
import { apkFileName, formatBytes, timeAgo } from '@/lib/text';
import { ApkDownloadButton } from '@/components/ui';
import { DownloadIcon } from '@/components/icons';
import { cn } from '@/lib/cn';

/**
 * 顺序触发整批 APK 下载：为每个产物建临时 <a download> 并错峰点击。
 * 直连同源 /apks 静态地址、不缓冲进内存；浏览器首次会提示「允许下载多个文件」。
 */
function downloadAll(artifacts: BuildArtifact[]) {
  artifacts.forEach((a, i) => {
    setTimeout(() => {
      const link = document.createElement('a');
      link.href = a.apkUrl;
      link.download = apkFileName(a.apkUrl);
      link.rel = 'noopener';
      document.body.appendChild(link);
      link.click();
      link.remove();
    }, i * 350);
  });
}

function Chevron({ open }: { open: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={cn('w-4 h-4 transition-transform', open && 'rotate-90')}
      fill="none"
      stroke="currentColor"
      strokeWidth={2.5}
    >
      <path d="m9 6 6 6-6 6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function BuildsPage() {
  const [brand, setBrand] = useState<BrandCode | 'all'>('all');
  const { data: jobs, isLoading } = useBuildJobs(brand === 'all' ? undefined : brand);

  return (
    <section>
      <div className="flex gap-2 mb-4 flex-wrap">
        <button className={cn('chip', brand === 'all' && 'chip-on')} onClick={() => setBrand('all')}>
          全部
        </button>
        {BRAND_ORDER.map((code) => (
          <button
            key={code}
            className={cn('chip', brand === code && 'chip-on')}
            onClick={() => setBrand(code)}
          >
            {BRAND_META[code].name}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="h-40 rounded-card bg-panel border border-line animate-pulse" />
      ) : (jobs ?? []).length === 0 ? (
        <div className="text-center text-muted py-[60px]">暂无构建记录</div>
      ) : (
        <div className="flex flex-col gap-3">
          {(jobs ?? []).map((j) => (
            <JobRow key={j.id} job={j} />
          ))}
        </div>
      )}
    </section>
  );
}

function JobRow({ job }: { job: BuildJob }) {
  const meta = isBrandCode(job.brandCode) ? BRAND_META[job.brandCode] : undefined;
  const statusMap: Record<BuildJob['status'], { cls: string; txt: string }> = {
    queued: { cls: 'text-[#475569] bg-[#e2e8f0]', txt: '排队中' },
    running: { cls: 'text-[#92681a] bg-[#fef3c7]', txt: '构建中' },
    success: { cls: 'text-[#15803d] bg-[#dcfce7]', txt: '成功' },
    failed: { cls: 'text-[#b91c1c] bg-[#fee2e2]', txt: '失败' },
  };
  const st = statusMap[job.status];
  const artifacts = job.artifacts ?? [];
  const [open, setOpen] = useState(false);

  return (
    <div className="section-card">
      <div className="flex items-center gap-3 flex-wrap">
        <span
          className="grid place-items-center w-8 h-8 rounded-lg text-white font-bold text-[13px] flex-none"
          style={{ background: meta?.accentColor ?? '#94a3b8' }}
        >
          {meta?.name[0] ?? '?'}
        </span>
        <div className="min-w-0">
          <div className="font-bold text-[14px] flex items-center gap-2 flex-wrap">
            <span className="font-mono">{job.jobName || `#${job.id}`}</span>
            {job.versionName && (
              <span className="font-mono text-[11.5px] px-2 py-0.5 rounded-md bg-[#ede9fe] text-[#6d28d9]">
                v{job.versionName}
              </span>
            )}
            <span className="font-mono text-[12px] text-muted">{job.flavors.length} flavor</span>
          </div>
          <div className="text-[12px] text-muted mt-0.5">
            {meta?.name ?? job.brandCode} · {job.operator ?? '—'} · {timeAgo(job.startedAt)}
          </div>
        </div>
        <div className="ml-auto flex items-center gap-2">
          {job.testEvents && (
            <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-[#ede9fe] text-[#6d28d9]">
              测试事件
            </span>
          )}
          <span className={cn('text-[11px] font-semibold px-[9px] py-[3px] rounded-full', st.cls)}>
            {st.txt}
          </span>
        </div>
      </div>

      {/* 产物：折叠头（产物数 + 一键下载全部 + 展开箭头）+ 可展开的逐个下载列表 */}
      {artifacts.length > 0 ? (
        <div className="mt-3">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setOpen((v) => !v)}
              className="flex items-center gap-1.5 text-[12.5px] font-semibold text-ink-2 hover:text-ink transition"
            >
              <Chevron open={open} />
              {open ? '收起产物' : '展开产物'} · {artifacts.length} 个 APK
            </button>
            <button
              onClick={() => downloadAll(artifacts)}
              className="ml-auto btn btn-primary text-[12px]"
              title="顺序下载该任务全部 APK（浏览器会提示允许下载多个文件）"
            >
              <DownloadIcon className="w-4 h-4" />
              下载全部 ({artifacts.length})
            </button>
          </div>
          {open && (
            <div className="mt-2 flex flex-col gap-2">
              {artifacts.map((a) => (
                <div
                  key={a.flavor}
                  className="flex items-center gap-3 rounded-[10px] border border-line bg-panel-2 px-3 py-2"
                >
                  <span className="font-mono text-[12px] font-semibold text-ink-2">{a.flavor}</span>
                  <span className="font-mono text-[11px] text-muted truncate" title={apkFileName(a.apkUrl)}>
                    {apkFileName(a.apkUrl)}
                  </span>
                  {a.sizeBytes ? (
                    <span className="text-[11px] text-muted">{formatBytes(a.sizeBytes)}</span>
                  ) : null}
                  <ApkDownloadButton url={a.apkUrl} className="ml-auto" />
                </div>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {job.flavors.map((f) => (
            <span key={f} className="font-mono text-[11px] px-2 py-0.5 rounded-md bg-bg text-ink-2">
              {f}
            </span>
          ))}
        </div>
      )}

      {job.logExcerpt && (
        <pre
          className="mt-3 rounded-lg p-3 text-[11.5px] font-mono overflow-auto"
          style={{ background: '#0c1426', color: '#94e2b5' }}
        >
          {job.logExcerpt}
        </pre>
      )}
    </div>
  );
}
