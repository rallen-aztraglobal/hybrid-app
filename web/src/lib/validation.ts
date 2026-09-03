import type { Channel, ChannelInput, DomainEntry, BrandCode } from './types';

/**
 * 校验层 —— 落实 CLAUDE.md 硬护栏 #5 与 ADR-0009（身份唯一性）+ 01-architecture.md §5.7（域名规则）。
 *
 * 前端校验是「即时提示」，后端仍是最终权威（DB 唯一约束 + 保存时探测）。
 * ADR-0009 更正：唯一性以 **applicationId（= 派生）与 (brand, flavor)** 为准；
 * **PAL_CODE 不再全局唯一**（跨品牌可重复），故移除 palCode 全局查重。
 * applicationId 由 `<品牌包前缀>.<flavor>` 派生（表单只读），从构造上保证唯一且与 flavor 一致。
 */

export interface FieldError {
  field: keyof ChannelInput | 'domains';
  message: string;
}

const FLAVOR_RE = /^[a-z][a-z0-9]*$/; // Gradle flavor：小写字母开头，字母数字
const APPID_RE = /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$/; // 反域名包名，至少两段
const PAL_RE = /^\d{6,}$/; // 现网 pal_code 都是长数字串
const SIGNING_KEY_RE = /^[a-z][a-z0-9_-]{0,31}$/; // 签名 key ID：小写字母开头，字母数字/下划线/连字符，≤32 位

/**
 * 合成 flavor：基础 flavor + 可选商店后缀（下划线连接）。
 * 无商店（storeCode 为空）时行为不变，等于基础 flavor。
 */
export function composeFlavor(baseFlavor: string, storeCode?: string | null): string {
  const base = baseFlavor.trim();
  const store = (storeCode ?? '').trim();
  return store ? `${base}_${store}` : base;
}

/**
 * 唯一性校验：在「除自己以外」的现有渠道里查重。
 * ADR-0009：以 applicationId（派生）与 (brand, flavor) 为准；PAL_CODE 不查重。
 * 注意：此处的 input.flavorName 须已是「合成 flavor」（含商店后缀），
 * 以便与既有渠道的 flavorName 直接比对——同 base + 同 store 才算重复，不同 store 允许并存。
 * @param existing 当前全量渠道（含其它品牌——applicationId 全局唯一）
 * @param selfId   编辑场景下排除自身；新增传 undefined
 */
export function checkUniqueness(
  input: Pick<ChannelInput, 'flavorName' | 'applicationId' | 'brandCode'>,
  existing: Channel[],
  selfId?: string,
): FieldError[] {
  const errors: FieldError[] = [];
  const others = existing.filter((c) => c.id !== selfId);

  const flavor = input.flavorName.trim();
  const appId = input.applicationId.trim();

  // applicationId 全局唯一（派生值；冲突即说明 (brand,flavor) 撞了或脏数据）。
  if (appId && others.some((c) => c.applicationId.toLowerCase() === appId.toLowerCase())) {
    const dup = others.find((c) => c.applicationId.toLowerCase() === appId.toLowerCase())!;
    errors.push({
      field: 'flavorName',
      message: `派生包名 ${appId} 已被渠道「${dup.flavorName}」占用`,
    });
  }

  // (brand, flavor) 唯一
  if (
    flavor &&
    others.some((c) => c.brandCode === input.brandCode && c.flavorName.toLowerCase() === flavor.toLowerCase())
  ) {
    errors.push({
      field: 'flavorName',
      message: `该品牌下已存在 flavor「${flavor}」`,
    });
  }

  return errors;
}

/**
 * 字段格式校验（必填 + 形态）。
 * @param baseFlavor 用户在表单里实际填写的「基础 flavor」（不含商店后缀）；
 *   缺省时回落用 input.flavorName（无商店场景下二者相同，向后兼容）。
 *   格式校验必须对**基础** flavor 做，而非合成 flavor——合成值含下划线，
 *   会被 FLAVOR_RE（不允许下划线）误判为不合法。
 */
export function checkFormat(input: ChannelInput, baseFlavor?: string): FieldError[] {
  const errors: FieldError[] = [];

  if (!input.appName.trim()) {
    errors.push({ field: 'appName', message: '应用名称必填' });
  }

  const flavor = (baseFlavor ?? input.flavorName).trim();
  if (!flavor) {
    errors.push({ field: 'flavorName', message: 'Flavor 名必填' });
  } else if (!FLAVOR_RE.test(flavor)) {
    errors.push({ field: 'flavorName', message: 'Flavor 须小写字母开头、仅含字母数字（如 ap01060）' });
  }

  const appId = input.applicationId.trim();
  if (!appId) {
    errors.push({ field: 'applicationId', message: '包名 applicationId 必填' });
  } else if (!APPID_RE.test(appId)) {
    errors.push({ field: 'applicationId', message: '包名须为合法反域名（如 com.arenaplus.ap01060）' });
  }

  const pal = input.palCode.trim();
  if (!pal) {
    errors.push({ field: 'palCode', message: 'PAL_CODE 必填' });
  } else if (!PAL_RE.test(pal)) {
    errors.push({ field: 'palCode', message: 'PAL_CODE 应为纯数字串' });
  }

  // 签名 key：空 = 默认，非空须匹配 GET /api/signing-keys 下发的 id 格式。
  const signingKey = (input.signingKey ?? '').trim();
  if (signingKey && !SIGNING_KEY_RE.test(signingKey)) {
    errors.push({ field: 'signingKey', message: '签名 key ID 须小写字母开头，仅含字母数字/下划线/连字符（≤32 位）' });
  }

  return errors;
}

/**
 * 综合校验：格式 + 唯一性 + （未继承时）域名。返回去重后的错误列表。
 * @param baseFlavor 见 checkFormat；input.flavorName 此时应已是合成 flavor（供唯一性/包名校验）。
 */
export function validateChannel(
  input: ChannelInput,
  existing: Channel[],
  selfId?: string,
  baseFlavor?: string,
): FieldError[] {
  const errors = [...checkFormat(input, baseFlavor), ...checkUniqueness(input, existing, selfId)];
  if (!input.useBrandDomains) {
    const de = validateDomains(input.domains ?? []);
    if (de.length) {
      errors.push({ field: 'domains', message: de[0] });
    }
  }
  return errors;
}

/**
 * 域名校验（01-architecture.md §5.7）：
 * - 必须 https://
 * - 必须合法域名
 * - 主域名必填，备用 0~3，去重
 * 返回人类可读的错误信息数组（空 = 通过）。
 */
export function validateDomains(domains: DomainEntry[]): string[] {
  const errors: string[] = [];
  const filled = domains
    .filter((d) => d.url.trim().length > 0)
    .sort((a, b) => a.position - b.position);

  const main = domains.find((d) => d.position === 0);
  if (!main || !main.url.trim()) {
    errors.push('主域名必填');
  }

  const backups = filled.filter((d) => d.position > 0);
  if (backups.length > 3) {
    errors.push('备用域名最多 3 个');
  }

  for (const d of filled) {
    const v = validateDomainUrl(d.url.trim());
    if (v) {
      const tag = d.position === 0 ? '主域名' : `备用${d.position}`;
      errors.push(`${tag}：${v}`);
    }
  }

  // 去重
  const seen = new Set<string>();
  for (const d of filled) {
    const norm = d.url.trim().toLowerCase().replace(/\/+$/, '');
    if (seen.has(norm)) {
      errors.push(`域名重复：${d.url.trim()}`);
    }
    seen.add(norm);
  }

  return errors;
}

/** 单个域名 URL 形态校验。返回错误信息或 null。 */
export function validateDomainUrl(url: string): string | null {
  if (!url) return '不能为空';
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch {
    return '不是合法 URL';
  }
  if (parsed.protocol !== 'https:') {
    return '线上一律 https://';
  }
  // 合法域名：含点、无空格、非纯 IP（这里仅做轻量形态校验，后端做 DNS 解析）
  const host = parsed.hostname;
  if (!host.includes('.') || /\s/.test(host)) {
    return '不是合法域名';
  }
  return null;
}

/** 品牌下渠道计数（不含 archived），用于 Tab/侧栏徽标。 */
export function countChannels(channels: Channel[], brand: BrandCode): number {
  return channels.filter((c) => c.brandCode === brand && c.status !== 'archived').length;
}

/** versionName 校验（ADR-0008 打包中心）：必须 X.Y.Z（各段为非负整数）。 */
const VERSION_RE = /^\d+\.\d+\.\d+$/;
export function validateVersionName(v: string): string | null {
  const s = v.trim();
  if (!s) return '版本号必填';
  if (!VERSION_RE.test(s)) return '版本号须为 X.Y.Z（如 1.0.3）';
  return null;
}

/**
 * 默认任务名（ADR-0008）：`<品牌code>-<versionName>-<YYYYMMDD-HHmm>`。
 * 打包中心初始填入、可改；留空则后端用同规则生成。
 */
export function defaultJobName(brand: BrandCode, versionName: string, when = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0');
  const stamp = `${when.getFullYear()}${p(when.getMonth() + 1)}${p(when.getDate())}-${p(when.getHours())}${p(when.getMinutes())}`;
  return `${brand}-${versionName || '0.0.0'}-${stamp}`;
}
