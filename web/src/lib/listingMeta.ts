import type { ListingPlatform, ListingTech } from './types';

/**
 * 上架包模块的静态元数据与常量（09-listing.md / ADR-0014）。
 * 平台配色仅作前端展示区分，不承载业务含义。
 */

export interface PlatformMeta {
  code: ListingPlatform;
  label: string;
  accentColor: string;
}

export const PLATFORM_META: Record<ListingPlatform, PlatformMeta> = {
  android: { code: 'android', label: 'Android', accentColor: '#16a34a' },
  ios: { code: 'ios', label: 'iOS', accentColor: '#334155' },
};

export const PLATFORM_ORDER: ListingPlatform[] = ['android', 'ios'];

export const TECH_LABEL: Record<ListingTech, string> = {
  flutter: 'Flutter',
  native_ios: '原生 iOS',
  native_android: '原生 Android',
};

/**
 * 平台可选技术栈（对齐后端 normalizeTech 约束）：
 * native_ios 仅 ios 可选，native_android 仅 android 可选，flutter 两端皆可。
 */
export const TECH_OPTIONS_BY_PLATFORM: Record<ListingPlatform, { value: ListingTech; label: string }[]> = {
  android: [
    { value: 'flutter', label: TECH_LABEL.flutter },
    { value: 'native_android', label: TECH_LABEL.native_android },
  ],
  ios: [
    { value: 'flutter', label: TECH_LABEL.flutter },
    { value: 'native_ios', label: TECH_LABEL.native_ios },
  ],
};

/** 硬编码强制走 A 面的国家（ADR-0014）：绝不出现在可选项里，手输也会被前端拦截+后端 400。 */
export const FORCED_A_COUNTRIES = ['CN', 'US'];

/** 常用国家白名单可选项（ISO-3166-1 alpha-2），刻意排除 CN/US。 */
export const COUNTRY_OPTIONS: { code: string; name: string }[] = [
  { code: 'PH', name: '菲律宾' },
  { code: 'ID', name: '印尼' },
  { code: 'TH', name: '泰国' },
  { code: 'VN', name: '越南' },
  { code: 'MY', name: '马来西亚' },
  { code: 'SG', name: '新加坡' },
  { code: 'IN', name: '印度' },
  { code: 'BR', name: '巴西' },
  { code: 'MX', name: '墨西哥' },
  { code: 'KH', name: '柬埔寨' },
  { code: 'MM', name: '缅甸' },
  { code: 'BD', name: '孟加拉' },
  { code: 'PK', name: '巴基斯坦' },
  { code: 'NG', name: '尼日利亚' },
  { code: 'EG', name: '埃及' },
  { code: 'JP', name: '日本' },
  { code: 'KR', name: '韩国' },
  { code: 'TW', name: '台湾' },
  { code: 'HK', name: '香港' },
  { code: 'GB', name: '英国' },
  { code: 'DE', name: '德国' },
  { code: 'FR', name: '法国' },
  { code: 'CA', name: '加拿大' },
  { code: 'AU', name: '澳大利亚' },
];

/** 国家码 → 展示文案（未收录的码原样大写展示）。 */
export function countryLabel(code: string): string {
  const upper = code.toUpperCase();
  const found = COUNTRY_OPTIONS.find((c) => c.code === upper);
  return found ? `${found.name} ${found.code}` : upper;
}

/** 常用 IANA 时区可选项（覆盖东南亚主力市场 + 少量对照参考）。 */
export const TIMEZONE_OPTIONS: string[] = [
  'Asia/Manila',
  'Asia/Jakarta',
  'Asia/Bangkok',
  'Asia/Ho_Chi_Minh',
  'Asia/Kuala_Lumpur',
  'Asia/Singapore',
  'Asia/Kolkata',
  'Asia/Dhaka',
  'Asia/Karachi',
  'Asia/Yangon',
  'Asia/Phnom_Penh',
  'Asia/Hong_Kong',
  'Asia/Taipei',
  'Asia/Tokyo',
  'Asia/Seoul',
  'America/Sao_Paulo',
  'America/Mexico_City',
  'Africa/Lagos',
  'Africa/Cairo',
  'Europe/London',
  'Europe/Berlin',
  'Europe/Paris',
  'America/Toronto',
  'Australia/Sydney',
];

/** 多行文本域 → 去空白 / 去空行的字符串数组（IP 名单解析通用）。 */
export function parseLines(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
}
