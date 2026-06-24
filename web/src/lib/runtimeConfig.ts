import type { AppRuntimeConfig } from './types';

/**
 * 运行时配置三级取用（ADR-0002） —— Web 端镜像实现。
 *
 * 这是后台「按 PAL_CODE 预览某渠道实际会解析到哪些域名」的工具，逻辑刻意与
 * APK 侧 DomainResolver 对齐，便于运营核对：
 *   ① 实时拉取 GET /api/app/config?palcode=（成功即写本地缓存，覆盖旧值）
 *   ② 失败 → 用本地缓存（上一次成功返回）
 *   ③ 从未成功过 → 用编译期兜底清单（assets/bootstrap.json 等价物）
 *
 * 硬约束（CLAUDE.md #2/#3）：域名**绝不编译期硬编码**。这里唯一编译期固定的
 * 只有「配置服务地址」(configEndpoint) 与按 palcode 烧录的兜底域名（来自后端
 * 下发快照的离线副本），且兜底仅在「从未成功拉取」时启用。
 */

const CACHE_PREFIX = 'hybrid:appconfig:';
const cacheKey = (palcode: string) => `${CACHE_PREFIX}${palcode}`;

/** 取用结果，标注命中来源，便于 UI 显示「实时 / 缓存 / 兜底」。 */
export interface ResolveResult {
  source: 'remote' | 'cache' | 'bootstrap';
  config: AppRuntimeConfig | null;
  /** 若实时拉取失败，记录原因供 UI 区分「域名问题 vs 本机网络」 */
  remoteError?: string;
}

/** 读本地缓存（上一次成功的配置）。 */
export function readCache(palcode: string): AppRuntimeConfig | null {
  try {
    const raw = localStorage.getItem(cacheKey(palcode));
    return raw ? (JSON.parse(raw) as AppRuntimeConfig) : null;
  } catch {
    return null;
  }
}

/** 成功拉取后写缓存（覆盖旧值），使「兜底 = 最近一次成功配置」。 */
export function writeCache(cfg: AppRuntimeConfig): void {
  try {
    localStorage.setItem(cacheKey(cfg.palcode), JSON.stringify(cfg));
  } catch {
    /* 隐私模式 / 配额满：静默降级，不影响取用链路 */
  }
}

/**
 * 编译期兜底（仅首启 / 从未成功拉取过时使用）。
 * 注意：这里的 domains 是后端下发快照的**离线副本**占位，不是把线上域名焊死。
 * 真实 APK 由 CLI 写入每个 flavor 的 assets/bootstrap.json；Web 端用一份等价的
 * 通用兜底（不含任何业务真实域名硬编码假设——值由运行环境注入或留空）。
 */
function bootstrapFallback(palcode: string): AppRuntimeConfig {
  return {
    palcode,
    domains: [], // 不硬编码业务域名；首启无网且无缓存时 UI 提示「请联网获取配置」
    probePath: '/healthz',
    configVersion: 0,
    ttlSeconds: 600,
  };
}

/**
 * 三级取用。fetcher 注入便于测试 / mock；默认走真实 endpoint。
 * 短超时（默认 1.2s，落在 ADR-0002 的 0.8~1.5s 区间），首屏不被接口拖慢。
 */
export async function resolveAppConfig(
  palcode: string,
  opts?: {
    fetcher?: (palcode: string, signal: AbortSignal) => Promise<AppRuntimeConfig>;
    timeoutMs?: number;
  },
): Promise<ResolveResult> {
  const timeoutMs = opts?.timeoutMs ?? 1200;
  const fetcher = opts?.fetcher ?? defaultFetcher;

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const cfg = await fetcher(palcode, controller.signal);
    clearTimeout(timer);
    writeCache(cfg); // ② 成功即自更新缓存
    return { source: 'remote', config: cfg };
  } catch (err) {
    clearTimeout(timer);
    const cached = readCache(palcode); // ② 失败用缓存
    if (cached) {
      return { source: 'cache', config: cached, remoteError: errMessage(err) };
    }
    // ③ 从未成功过 → 编译期兜底
    return { source: 'bootstrap', config: bootstrapFallback(palcode), remoteError: errMessage(err) };
  }
}

async function defaultFetcher(palcode: string, signal: AbortSignal): Promise<AppRuntimeConfig> {
  const res = await fetch(`/api/app/config?palcode=${encodeURIComponent(palcode)}`, { signal });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const json = (await res.json()) as { data?: AppRuntimeConfig } | AppRuntimeConfig;
  // 兼容 {code,message,data} 信封与裸对象
  return 'data' in json && json.data ? json.data : (json as AppRuntimeConfig);
}

function errMessage(err: unknown): string {
  if (err instanceof DOMException && err.name === 'AbortError') return '请求超时';
  if (err instanceof Error) return err.message;
  return String(err);
}
