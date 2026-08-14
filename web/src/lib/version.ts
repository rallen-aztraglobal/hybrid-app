/**
 * 版本号（X.Y.Z）解析与数值比较 —— 打包中心「当前版本」展示与前端预校验用。
 * 与后端 server/internal/service/build.go 的 parseSemver/compareSemver 保持同一语义：
 * 三段均为非负整数，按数值比较而非字符串字典序比较（否则 "1.10.0" 会被误判小于 "1.2.0"）。
 * 后端仍是最终权威——这里只用于「当前版本」展示与提交前的即时提示，不做强制拦截。
 */

const VERSION_TUPLE_RE = /^(\d+)\.(\d+)\.(\d+)$/;

export type VersionTuple = readonly [number, number, number];

/** 解析形如 X.Y.Z 的版本号；不合法（含历史脏数据）返回 null，绝不抛异常。 */
export function parseVersion(v: string): VersionTuple | null {
  const m = VERSION_TUPLE_RE.exec(v.trim());
  if (!m) return null;
  return [Number(m[1]), Number(m[2]), Number(m[3])];
}

/** 返回 a 相对 b 的大小：-1 表示 a<b，0 表示相等，1 表示 a>b（逐段数值比较）。 */
export function compareVersions(a: VersionTuple, b: VersionTuple): number {
  for (let i = 0; i < 3; i++) {
    if (a[i] !== b[i]) return a[i] < b[i] ? -1 : 1;
  }
  return 0;
}

/**
 * 从一批版本号字符串里取语义版本最高的一个（而非数组里的最后/最新一项）。
 * 忽略无法解析的非法版本号；全部非法或数组为空时返回 null。
 */
export function highestVersion(versionNames: readonly string[]): string | null {
  let best: string | null = null;
  let bestTuple: VersionTuple | null = null;
  for (const name of versionNames) {
    const tuple = parseVersion(name);
    if (!tuple) continue;
    if (!bestTuple || compareVersions(tuple, bestTuple) > 0) {
      best = name;
      bestTuple = tuple;
    }
  }
  return best;
}

/**
 * 提交前的即时提示：新版本是否低于当前版本。currentVersion 为 null（暂无成功构建）时永远放行。
 * 非法格式的 newVersion 不在这里报错——交给既有的 validateVersionName 处理格式校验。
 */
export function isVersionLowerThanCurrent(newVersion: string, currentVersion: string | null): boolean {
  if (!currentVersion) return false;
  const newTuple = parseVersion(newVersion);
  const curTuple = parseVersion(currentVersion);
  if (!newTuple || !curTuple) return false;
  return compareVersions(newTuple, curTuple) < 0;
}
