import type {
  Brand,
  BrandCode,
  BuildArtifact,
  BuildJob,
  BuildJobRequest,
  BuildLogChunk,
  Channel,
  DomainEntry,
  DomainHealth,
} from '../types';
import { BRAND_META, BRAND_ORDER } from '../brands';
import { parseChannelsCsv } from '../csv';
import { countChannels } from '../validation';
import { cap } from '../text';
import { AP_CSV } from './channels.ap.csv';
import { BP_CSV } from './channels.bp.csv';
import { GP_CSV } from './channels.gp.csv';

/**
 * 进程内 mock 数据库（后端未就绪时的事实来源），从内联 CSV 还原。
 * 真实部署时由 Go 后端 + MySQL 取代；API 客户端在 fetch 失败时回退到此。
 */

const CSV_BY_BRAND: Record<BrandCode, string> = { ap: AP_CSV, bp: BP_CSV, gp: GP_CSV };

// 演示用：把每个品牌靠后的少数渠道标记为停用，让筛选器有内容可筛。
const DISABLED_FLAVORS = new Set([
  'johnc1010991',
  'johnc1010992',
  'bpom3442',
  'bpom3438',
  'gzmkt028',
  'gzmarket001',
]);

// 演示用：这些 flavor 有「最近一次成功构建」的 APK（渠道卡片显示「下载最新包」）。
const SEEDED_LATEST_APK: Record<string, string> = {
  ap01018: '/apks/ap/ap01018/1.0.3/app-ap01018-release.apk',
  ap01034: '/apks/ap/ap01034/1.0.3/app-ap01034-release.apk',
  ap01040: '/apks/ap/ap01040/1.0.3/app-ap01040-release.apk',
};

function buildChannels(): Channel[] {
  const all: Channel[] = [];
  for (const brand of BRAND_ORDER) {
    const list = parseChannelsCsv(CSV_BY_BRAND[brand], brand);
    for (const c of list) {
      if (DISABLED_FLAVORS.has(c.flavorName)) c.status = 'disabled';
      if (SEEDED_LATEST_APK[c.flavorName]) c.latestApkUrl = SEEDED_LATEST_APK[c.flavorName];
    }
    all.push(...list);
  }
  return all;
}

let channels: Channel[] = buildChannels();

// 品牌级默认域名（mock）。真实清单由后端 brand_domain 下发。
const HEALTH_PRESET: Record<BrandCode, DomainHealth[]> = {
  ap: ['ok', 'ok', 'warn'],
  bp: ['ok', 'down'],
  gp: ['ok', 'ok'],
};

function brandDomains(brand: BrandCode): DomainEntry[] {
  return BRAND_META[brand].fallbackDomains.map((url, i) => ({
    position: i,
    url,
    enabled: true,
    health: HEALTH_PRESET[brand]?.[i] ?? 'unknown',
    latencyMs: i === 0 ? 120 : 240 + i * 90,
  }));
}

const brandDomainStore: Record<BrandCode, DomainEntry[]> = {
  ap: brandDomains('ap'),
  bp: brandDomains('bp'),
  gp: brandDomains('gp'),
};

/** 模拟 nginx 静态产物地址（ADR-0008：/apks/<brand>/<flavor>/<versionName>/...）。 */
function mockApkUrl(brand: BrandCode, flavor: string, versionName: string): string {
  return `/apks/${brand}/${flavor}/${versionName}/app-${flavor}-release.apk`;
}

function mockArtifacts(brand: BrandCode, flavors: string[], versionName: string): BuildArtifact[] {
  return flavors.map((f) => ({
    flavor: f,
    versionName,
    apkUrl: mockApkUrl(brand, f, versionName),
    sizeBytes: 18_000_000 + Math.floor(Math.random() * 6_000_000),
    builtAt: new Date().toISOString(),
  }));
}

let buildJobs: BuildJob[] = [
  {
    id: '128',
    jobName: 'ap-1.0.3-20260623-1402',
    brandCode: 'ap',
    flavors: ['ap01018', 'ap01034', 'ap01040'],
    versionName: '1.0.3',
    testEvents: false,
    status: 'success',
    operator: 'Daly',
    artifacts: mockArtifacts('ap', ['ap01018', 'ap01034', 'ap01040'], '1.0.3'),
    logExcerpt: 'BUILD SUCCESSFUL in 3m 12s',
    startedAt: '2026-06-23T14:02:00Z',
    finishedAt: '2026-06-23T14:05:12Z',
  },
  {
    id: '127',
    jobName: 'bp-1.0.3-20260623-1130',
    brandCode: 'bp',
    flavors: ['bpom3410'],
    versionName: '1.0.3',
    testEvents: true,
    status: 'failed',
    operator: 'Daly',
    artifacts: [],
    logExcerpt: 'Execution failed for task :app:assembleBpom3410Release — keystore not found',
    startedAt: '2026-06-23T11:30:00Z',
    finishedAt: '2026-06-23T11:31:40Z',
  },
];

let recordSeq = 200;

/** 默认任务名：<品牌code>-<versionName>-<YYYYMMDD-HHmm>（ADR-0008）。 */
function defaultJobName(brand: BrandCode, versionName: string, when = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0');
  const stamp = `${when.getFullYear()}${p(when.getMonth() + 1)}${p(when.getDate())}-${p(when.getHours())}${p(when.getMinutes())}`;
  return `${brand}-${versionName || '0.0.0'}-${stamp}`;
}

/** mock 日志脚本（逐行模拟 hybrid-pack release，供日志流轮询消费）。 */
const jobLogScripts = new Map<string, { lines: string[]; createdAt: number; job: BuildJob }>();

function buildLogScript(job: BuildJob): string[] {
  const lines = [
    `# hybrid-pack release --brand ${job.brandCode} --version ${job.versionName}${job.testEvents ? ' --test-events' : ''}`,
    `# job: ${job.jobName}`,
    `→ 入队构建任务 #${job.id}`,
    `→ 从后台同步配置 (hybrid-pack pull)… ✓ ${job.flavors.length} 渠道`,
    `→ 渲染 channels/${job.brandCode}.csv + res + bootstrap.json… ✓`,
    ...job.flavors.map((f) => `→ gradlew assemble${cap(f)}Release`),
    ...job.flavors.map((f) => `  ✓ ${mockApkUrl(job.brandCode, f, job.versionName)}`),
    `✓ 完成 ${job.flavors.length} 个 APK · 已写构建记录 #${job.id}`,
  ];
  return lines;
}

export const mockDb = {
  listBrands(): Brand[] {
    return BRAND_ORDER.map((code) => {
      const meta = BRAND_META[code];
      return {
        code,
        name: meta.name,
        scheme: meta.scheme,
        hmsEnabled: meta.hmsEnabled,
        accentColor: meta.accentColor,
        channelCount: countChannels(channels, code),
        domains: brandDomainStore[code].map((d) => ({ ...d })),
      };
    });
  },

  getBrandDomains(code: BrandCode): DomainEntry[] {
    return brandDomainStore[code].map((d) => ({ ...d }));
  },

  setBrandDomains(code: BrandCode, domains: DomainEntry[]): DomainEntry[] {
    // 保存时模拟一次探测：把空 health 标成 ok（后端真实探测）。
    brandDomainStore[code] = domains
      .filter((d) => d.url.trim())
      .map((d) => ({ ...d, health: d.health ?? 'ok' }));
    return this.getBrandDomains(code);
  },

  listChannels(): Channel[] {
    return channels.map((c) => ({ ...c }));
  },

  getChannel(id: string): Channel | undefined {
    const c = channels.find((x) => x.id === id);
    return c ? { ...c } : undefined;
  },

  upsertChannel(input: Channel): Channel {
    const idx = channels.findIndex((c) => c.id === input.id);
    if (idx >= 0) {
      channels[idx] = { ...input, updatedAt: new Date().toISOString() };
      return { ...channels[idx] };
    }
    const created = { ...input, updatedAt: new Date().toISOString() };
    channels = [created, ...channels];
    return { ...created };
  },

  archiveChannel(id: string): void {
    const c = channels.find((x) => x.id === id);
    if (c) c.status = 'archived';
  },

  listBuildJobs(brand?: BrandCode): BuildJob[] {
    const list = brand ? buildJobs.filter((r) => r.brandCode === brand) : buildJobs;
    return list.map((r) => ({ ...r }));
  },

  /** 触发任务（ADR-0008）：入队一条 running 记录并准备日志脚本。 */
  createBuildJob(req: BuildJobRequest): BuildJob {
    recordSeq += 1;
    const id = String(recordSeq);
    const job: BuildJob = {
      id,
      jobName: req.jobName?.trim() || defaultJobName(req.brandCode, req.versionName),
      brandCode: req.brandCode,
      flavors: req.flavors,
      versionName: req.versionName,
      testEvents: req.testEvents,
      status: 'running',
      operator: 'Daly',
      artifacts: [],
      startedAt: new Date().toISOString(),
    };
    buildJobs = [job, ...buildJobs];
    jobLogScripts.set(id, { lines: buildLogScript(job), createdAt: Date.now(), job });
    return { ...job };
  },

  /**
   * 增量日志（模拟流）：按时间逐行「吐出」脚本，约每 450ms 一行。
   * 全部吐完后把任务标记 success 并补齐 artifacts，done=true 让前端停止轮询。
   */
  jobLogs(jobId: string, after: number): BuildLogChunk {
    const script = jobLogScripts.get(jobId);
    if (!script) {
      const existing = buildJobs.find((j) => j.id === jobId);
      return { cursor: after, lines: [], done: true, status: existing?.status ?? 'success' };
    }
    const elapsed = Date.now() - script.createdAt;
    const revealed = Math.min(script.lines.length, Math.floor(elapsed / 450) + 1);
    const lines = script.lines.slice(after, revealed);
    const done = revealed >= script.lines.length;
    if (done) {
      const job = buildJobs.find((j) => j.id === jobId);
      if (job && job.status === 'running') {
        job.status = 'success';
        job.artifacts = mockArtifacts(job.brandCode, job.flavors, job.versionName);
        job.logExcerpt = 'BUILD SUCCESSFUL';
        job.finishedAt = new Date().toISOString();
        // 回填渠道「最新包」地址，使渠道卡片「下载最新包」可用。
        for (const a of job.artifacts) {
          const ch = channels.find((c) => c.brandCode === job.brandCode && c.flavorName === a.flavor);
          if (ch) ch.latestApkUrl = a.apkUrl;
        }
      }
    }
    return { cursor: revealed, lines, done, status: done ? 'success' : 'running' };
  },
};
