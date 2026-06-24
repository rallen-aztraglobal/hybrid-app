import type { BrandCode, IconVariant } from './types';

/**
 * 大渠道（品牌）静态元数据。
 *
 * 事实来源：app/build.gradle 的 `brandConfig`（scheme / hms / domain）
 * 与 docs/admin/ui/index.html 的品牌色。运行时域名清单不在此硬编码——
 * 域名一律走后端下发（ADR-0002），这里仅保留品牌身份与默认主域名作为
 * 展示兜底（mock 模式用）。
 */
export interface BrandMeta {
  code: BrandCode;
  name: string;
  scheme: string;
  hmsEnabled: boolean;
  accentColor: string;
  /** CSS 变量名，配合 index.css 令牌 */
  accentVar: string;
  /** 仅作展示兜底的默认域名（真实清单由后端下发） */
  fallbackDomains: string[];
  /**
   * 品牌包前缀（ADR-0009）：applicationId = packagePrefix + '.' + flavor。
   * ap→com.arenaplus / bp→com.bingoplus / gp→com.gamezone（与 app/build.gradle 一致）。
   * 后端 BrandView 未下发 packagePrefix 时用此兜底。
   */
  packagePrefix: string;
}

export const BRAND_META: Record<BrandCode, BrandMeta> = {
  ap: {
    code: 'ap',
    name: 'ArenaPlus',
    scheme: 'gzone',
    hmsEnabled: false,
    accentColor: '#f97316',
    accentVar: 'var(--ap)',
    fallbackDomains: ['https://arenaplus.ph', 'https://ap-mirror.net', 'https://ap-cdn.app'],
    packagePrefix: 'com.arenaplus',
  },
  bp: {
    code: 'bp',
    name: 'BingoPlus',
    scheme: 'bingo',
    hmsEnabled: true,
    accentColor: '#2563eb',
    accentVar: 'var(--bp)',
    fallbackDomains: ['https://www.bingoplus.com', 'https://bingo-mirror.net'],
    packagePrefix: 'com.bingoplus',
  },
  gp: {
    code: 'gp',
    name: 'GameZone',
    scheme: 'gzone',
    hmsEnabled: false,
    accentColor: '#7c3aed',
    accentVar: 'var(--gp)',
    fallbackDomains: ['https://gzone.ph', 'https://gz-backup.app'],
    packagePrefix: 'com.gamezone',
  },
};

export const BRAND_ORDER: BrandCode[] = ['ap', 'bp', 'gp'];

export function isBrandCode(s: string): s is BrandCode {
  return s === 'ap' || s === 'bp' || s === 'gp';
}

/**
 * 派生 applicationId（ADR-0009 的核心规则）：`<品牌包前缀>.<flavor>`。
 * 表单只读展示、提交时附带；从构造上保证「applicationId 后缀 == flavor」、全局唯一。
 * @param prefix 优先用后端下发的 brand.packagePrefix；缺省回落到 BRAND_META。
 */
export function deriveApplicationId(brand: BrandCode, flavor: string, prefix?: string): string {
  const base = (prefix && prefix.trim()) || BRAND_META[brand].packagePrefix;
  const f = flavor.trim();
  return f ? `${base}.${f}` : '';
}

/** 由应用名提炼 2 字母图标占位（与原型一致：取字母、大写、截 2 位）。 */
export function iconInitials(name: string, brand: BrandCode): string {
  const letters = name.replace(/[^A-Za-z]/g, '').slice(0, 2).toUpperCase();
  return letters || BRAND_META[brand].name[0];
}

/** 卡片/图标的品牌渐变（复刻原型 iconGrad）。 */
export function iconGradient(hex: string): string {
  return `linear-gradient(135deg, ${hex}, ${hex}cc 60%, #ffffff22)`;
}

/**
 * Android 图标密度矩阵（见 03-build-and-icon-pipeline.md §2 / ADR-0005）。
 * 方形 & 圆形共用 48/72/96/144/192；自适应前景用 108/162/216/324/432。
 */
export const DENSITY_ORDER = ['mdpi', 'hdpi', 'xhdpi', 'xxhdpi', 'xxxhdpi'] as const;

export const SQUARE_PX: Record<(typeof DENSITY_ORDER)[number], number> = {
  mdpi: 48,
  hdpi: 72,
  xhdpi: 96,
  xxhdpi: 144,
  xxxhdpi: 192,
};

export const FOREGROUND_PX: Record<(typeof DENSITY_ORDER)[number], number> = {
  mdpi: 108,
  hdpi: 162,
  xhdpi: 216,
  xxhdpi: 324,
  xxxhdpi: 432,
};

export function pxForVariant(variant: IconVariant, dpi: (typeof DENSITY_ORDER)[number]): number {
  return variant === 'foreground' ? FOREGROUND_PX[dpi] : SQUARE_PX[dpi];
}

/** 自适应图标安全区比例（前景内容居中于内 ~66%）。 */
export const FOREGROUND_SAFE_RATIO = 0.66;

/** 主图标准尺寸（与 App Store 一致，留足下采样空间）。 */
export const MASTER_ICON_PX = 1024;
/** 主图最小可接受边长，低于此前端拦截提示放大模糊。 */
export const MASTER_ICON_MIN_PX = 512;
