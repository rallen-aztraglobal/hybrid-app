/**
 * 文本工具。
 * cap：首字母大写，复刻 package.sh / Gradle 的 task 名规则
 * （assemble<Cap(flavor)>Release，如 ap01018 → Ap01018）。
 */
export function cap(s: string): string {
  if (!s) return s;
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** 人类可读文件大小（用于 APK 产物体积展示）。 */
export function formatBytes(bytes?: number): string {
  if (bytes == null || bytes <= 0) return '';
  const units = ['B', 'KB', 'MB', 'GB'];
  let v = bytes;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

/** 从下载地址末段取文件名（用于下载按钮的 download 属性）。 */
export function apkFileName(url: string): string {
  const clean = url.split('?')[0].split('#')[0];
  const seg = clean.split('/').filter(Boolean).pop();
  return seg || 'app-release.apk';
}

/** 相对时间（粗粒度，用于构建记录展示）。 */
export function timeAgo(iso?: string): string {
  if (!iso) return '—';
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return iso;
  const diff = Date.now() - t;
  const min = Math.floor(diff / 60000);
  if (min < 1) return '刚刚';
  if (min < 60) return `${min} 分钟前`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr} 小时前`;
  const day = Math.floor(hr / 24);
  return `${day} 天前`;
}
