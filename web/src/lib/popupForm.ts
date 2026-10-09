/**
 * 弹窗编辑表单的状态模型、校验与提交整形（docs/admin/12-popup.md §1/§2/§3）。
 * 表单里数值输入用字符串（编辑过程中可为空），提交前统一转换；时间用本地 'YYYY-MM-DDTHH:mm'，
 * 提交时转成带偏移的 RFC3339（契约示例 2026-10-01T00:00:00+08:00）。
 */
import type { BrandCode, Popup, PopupInput, PopupOpenMode, PopupPositionCode, PopupUserType } from './types';
import { pad2, parseLocalDateTime, toLocalDateTime } from './dateTime';
import {
  DEFAULT_AUTOPLAY_SECONDS,
  DEFAULT_RESUME_GAP_MINUTES,
  MAX_CAROUSEL_CARDS,
  POSITION_CAPS,
  TAB_TEXT_HARD_MAX,
  TAB_TEXT_SOFT_MAX,
  behaviorDefaults,
  charLen,
  checkImageSize,
  cardImageSpec,
  codeToVersion,
  hasCardImage,
  isForcedMode,
  isValidVersion,
  IMAGE_SPECS,
  normalizeBehavior,
  normalizeCountry,
  validateLink,
  versionToCode,
} from './popupMeta';

export interface FormCard {
  /** 仅前端 React key。 */
  uid: string;
  /** 服务端卡片 ID；新卡片无。 */
  id?: number;
  imageUrl: string;
  linkUrl: string;
  buttonText: string;
  title: string;
  description: string;
  /** 本次会话上传返回的元信息（尺寸比对用）；服务端载入的卡片无。 */
  meta?: { key?: string; width: number; height: number };
}

export interface PopupForm {
  id?: number;
  name: string;
  position: PopupPositionCode;
  enabled: boolean;
  priority: string;
  /** 本地 'YYYY-MM-DDTHH:mm'，'' = 不限。 */
  startAt: string;
  endAt: string;
  brandCodes: BrandCode[];
  appIds: string[];
  minVersion: string;
  maxVersion: string;
  userType: PopupUserType;
  countries: string[];
  tabEnabled: boolean;
  tabIconUrl: string;
  tabIconMeta?: { key?: string; width: number; height: number };
  tabText: string;
  countdown: boolean;
  maskClosable: boolean;
  closable: boolean;
  openMode: PopupOpenMode;
  autoplaySeconds: string;
  resumeGapMinutes: string;
  badge: boolean;
  cards: FormCard[];
}

let uidSeq = 0;
export function newUid(): string {
  uidSeq += 1;
  return `c${Date.now().toString(36)}${uidSeq}`;
}

export function blankCard(): FormCard {
  return { uid: newUid(), imageUrl: '', linkUrl: '', buttonText: '', title: '', description: '' };
}

export function blankForm(position: PopupPositionCode = 'P1'): PopupForm {
  const d = behaviorDefaults(position);
  return {
    name: '',
    position,
    enabled: true,
    priority: '50',
    startAt: '',
    endAt: '',
    brandCodes: [],
    appIds: [],
    minVersion: '',
    maxVersion: '',
    userType: 'all',
    countries: [],
    tabEnabled: d.tabEnabled,
    tabIconUrl: '',
    tabText: '',
    countdown: d.countdown,
    maskClosable: d.maskClosable,
    closable: d.closable,
    openMode: 'webview',
    autoplaySeconds: String(d.autoplaySeconds),
    resumeGapMinutes: String(d.resumeGapMinutes),
    badge: d.badge,
    cards: [blankCard()],
  };
}

/** 切位置：保留名称/定向/生效期等通用字段，行为字段回到新位置默认值，卡片裁到上限。 */
export function switchPosition(form: PopupForm, position: PopupPositionCode): PopupForm {
  const d = behaviorDefaults(position);
  const max = POSITION_CAPS[position].maxCards;
  const cards = form.cards.slice(0, max);
  return {
    ...form,
    position,
    tabEnabled: d.tabEnabled,
    countdown: d.countdown,
    maskClosable: d.maskClosable,
    closable: d.closable,
    badge: d.badge,
    autoplaySeconds: String(d.autoplaySeconds),
    resumeGapMinutes: String(d.resumeGapMinutes),
    cards: cards.length ? cards : [blankCard()],
  };
}

export function formFromPopup(p: Popup): PopupForm {
  return {
    id: p.id,
    name: p.name,
    position: p.position,
    enabled: p.enabled,
    priority: String(p.priority),
    startAt: rfc3339ToLocal(p.startAt),
    endAt: rfc3339ToLocal(p.endAt),
    brandCodes: [...p.brandCodes],
    appIds: [...p.appIds],
    minVersion: p.minVersion || '',
    maxVersion: p.maxVersion || '',
    userType: p.userType,
    countries: [...p.countries],
    tabEnabled: p.tabEnabled,
    tabIconUrl: p.tabIconUrl || '',
    tabText: p.tabText || '',
    countdown: p.countdown,
    maskClosable: p.maskClosable,
    closable: p.closable,
    openMode: p.openMode,
    autoplaySeconds: String(p.autoplaySeconds || DEFAULT_AUTOPLAY_SECONDS),
    resumeGapMinutes: String(p.resumeGapMinutes || DEFAULT_RESUME_GAP_MINUTES),
    badge: p.badge,
    cards: p.cards.length
      ? [...p.cards]
          .sort((a, b) => a.sort - b.sort)
          .map((c) => ({
            uid: newUid(),
            id: c.id,
            imageUrl: c.imageUrl,
            linkUrl: c.linkUrl,
            buttonText: c.buttonText,
            title: c.title,
            description: c.description,
          }))
      : [blankCard()],
  };
}

// ---------------------------------------------------------------------------
// 时间格式
// ---------------------------------------------------------------------------

/** 本地 'YYYY-MM-DDTHH:mm' → 带偏移 RFC3339；空串 → null。 */
export function localToRFC3339(local: string): string | null {
  const d = parseLocalDateTime(local);
  if (!d) return null;
  const off = -d.getTimezoneOffset();
  const sign = off >= 0 ? '+' : '-';
  const hh = pad2(Math.floor(Math.abs(off) / 60));
  const mm = pad2(Math.abs(off) % 60);
  return `${local}:00${sign}${hh}:${mm}`;
}

/** RFC3339（任意时区）→ 浏览器本地 'YYYY-MM-DDTHH:mm'；空/非法 → ''。 */
export function rfc3339ToLocal(s: string | null | undefined): string {
  if (!s) return '';
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? '' : toLocalDateTime(d);
}

// ---------------------------------------------------------------------------
// 校验
// ---------------------------------------------------------------------------

export interface ScopeFlags {
  /** 品牌范围为全量（allBrands）。受限账号 brandCodes 必须非空。 */
  allBrands: boolean;
  /** 渠道范围为全量（allChannels）。受限账号 appIds 必须非空。 */
  allChannels: boolean;
}

export interface ValidationResult {
  /** 阻断保存的错误。 */
  errors: string[];
  /** 黄色提示，不阻断。 */
  warnings: string[];
}

function toInt(s: string): number | null {
  const v = s.trim();
  if (!/^-?\d+$/.test(v)) return null;
  return Number(v);
}

/** 当前位置实际生效的卡片（不支持轮播的位置只取第 1 张）。 */
export function effectiveCards(form: PopupForm): FormCard[] {
  return form.cards.slice(0, POSITION_CAPS[form.position].maxCards);
}

export function validatePopupForm(
  form: PopupForm,
  scope: ScopeFlags = { allBrands: true, allChannels: true },
): ValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];
  const caps = POSITION_CAPS[form.position];
  const forced = isForcedMode(form.position, form.closable);

  const name = form.name.trim();
  if (!name) errors.push('请填写弹窗名称');
  else if (charLen(name) > 64) errors.push('弹窗名称不能超过 64 个字符');

  const prio = toInt(form.priority);
  if (prio === null || prio < 0 || prio > 999) errors.push('优先级需为 0–999 的整数');

  const start = parseLocalDateTime(form.startAt);
  const end = parseLocalDateTime(form.endAt);
  if (start && end && end.getTime() <= start.getTime()) errors.push('生效结束时间需晚于开始时间');

  if (!scope.allBrands && form.brandCodes.length === 0) errors.push('当前账号数据范围受限，必须选择定向品牌');
  if (!scope.allChannels && form.appIds.length === 0) errors.push('当前账号数据范围受限，必须选择定向渠道包');

  if (!isValidVersion(form.minVersion)) errors.push('版本下限格式需为 X.Y.Z（各段 0–99）');
  if (!isValidVersion(form.maxVersion)) errors.push('版本上限格式需为 X.Y.Z（各段 0–99）');
  const minC = versionToCode(form.minVersion);
  const maxC = versionToCode(form.maxVersion);
  if (minC !== null && maxC !== null && minC >= maxC) errors.push('版本下限（≥）需小于版本上限（<）');

  for (const c of form.countries) {
    if (!normalizeCountry(c)) errors.push(`国家码 ${c} 不是合法的 ISO 二位大写码`);
  }

  // 卡片
  const cards = effectiveCards(form);
  if (form.cards.length > caps.maxCards) errors.push(`${form.position} 最多 ${caps.maxCards} 张`);
  if (cards.length === 0) errors.push('至少需要 1 张素材卡片');
  if (caps.maxCards > 1 && cards.length > MAX_CAROUSEL_CARDS) errors.push(`最多 ${MAX_CAROUSEL_CARDS} 张`);

  const spec = cardImageSpec(form.position);
  cards.forEach((c, i) => {
    const tag = caps.maxCards > 1 ? `卡片 ${i + 1}` : '素材';
    if (hasCardImage(form.position)) {
      if (caps.image === 'required' && !c.imageUrl) errors.push(`${tag}：请上传图片`);
      if (c.imageUrl && spec && c.meta) {
        const w = checkImageSize(spec, c.meta.width, c.meta.height);
        if (w) warnings.push(`${tag}：${w}`);
      }
    }
    if (caps.title === 'ifNoImage' && !c.imageUrl && !c.title.trim()) {
      errors.push(`${tag}：无图时必须填写标题`);
    }
    if (caps.title === 'required' && !c.title.trim()) {
      errors.push(`${tag}：${form.position === 'P8' ? '便条文案' : '文案'}必填`);
    }
    if (form.position === 'P8') {
      const n = charLen(c.title);
      if (n > TAB_TEXT_HARD_MAX) errors.push(`便条文案不能超过 ${TAB_TEXT_HARD_MAX} 个字符`);
      else if (n > TAB_TEXT_SOFT_MAX) warnings.push(`便条文案 ${n} 字，建议 ≤ ${TAB_TEXT_SOFT_MAX} 字，过长会被截断`);
    }
    if (charLen(c.title) > 64) errors.push(`${tag}：标题不能超过 64 个字符`);
    if (charLen(c.description) > 255) errors.push(`${tag}：描述不能超过 255 个字符`);
    if (charLen(c.buttonText) > 32) errors.push(`${tag}：按钮文案不能超过 32 个字符`);
    const le = validateLink(c.linkUrl, form.openMode);
    if (le) errors.push(`${tag}：${le}`);
  });

  // 便条态
  const tabOn = caps.tab === 'optional' && form.tabEnabled && !forced;
  if (tabOn) {
    const t = form.tabText.trim();
    const n = charLen(t);
    if (!t) errors.push('开启便条态后，便条文案必填');
    else if (n > TAB_TEXT_HARD_MAX) errors.push(`便条文案不能超过 ${TAB_TEXT_HARD_MAX} 个字符`);
    else if (n > TAB_TEXT_SOFT_MAX) warnings.push(`便条文案 ${n} 字，建议 ≤ ${TAB_TEXT_SOFT_MAX} 字，过长会被截断`);
    if (form.tabIconUrl && form.tabIconMeta) {
      const w = checkImageSize(IMAGE_SPECS.icon, form.tabIconMeta.width, form.tabIconMeta.height);
      if (w) warnings.push(`便条图标：${w}`);
    }
  }

  if (form.position === 'P3') {
    const v = toInt(form.autoplaySeconds);
    if (v === null || v < 2 || v > 30) errors.push('P3 轮播间隔需为 2–30 秒');
  }
  if (form.position === 'P1') {
    const v = toInt(form.resumeGapMinutes);
    if (v === null || v < 5 || v > 1440) errors.push('P1 回前台间隔需为 5–1440 分钟');
  }

  return { errors, warnings };
}

// ---------------------------------------------------------------------------
// 提交整形
// ---------------------------------------------------------------------------

/** 表单 → 提交 payload：先按能力矩阵归一化，再按契约格式化时间 / 版本 / 卡片 sort。 */
export function toPopupInput(form: PopupForm): PopupInput {
  const caps = POSITION_CAPS[form.position];
  const b = normalizeBehavior({
    position: form.position,
    tabEnabled: form.tabEnabled,
    countdown: form.countdown,
    maskClosable: form.maskClosable,
    closable: form.closable,
    badge: form.badge,
    autoplaySeconds: toInt(form.autoplaySeconds) ?? DEFAULT_AUTOPLAY_SECONDS,
    resumeGapMinutes: toInt(form.resumeGapMinutes) ?? DEFAULT_RESUME_GAP_MINUTES,
  });
  const cards = effectiveCards(form).map((c, i) => ({
    ...(c.id ? { id: c.id } : {}),
    sort: i,
    imageUrl: hasCardImage(form.position) ? c.imageUrl : '',
    linkUrl: c.linkUrl.trim(),
    buttonText: caps.cta ? c.buttonText.trim() : '',
    title: caps.title === 'none' ? '' : c.title.trim(),
    description: caps.cta ? c.description.trim() : '',
  }));
  const minC = versionToCode(form.minVersion);
  const maxC = versionToCode(form.maxVersion);
  return {
    name: form.name.trim(),
    position: form.position,
    enabled: form.enabled,
    priority: toInt(form.priority) ?? 0,
    startAt: localToRFC3339(form.startAt),
    endAt: localToRFC3339(form.endAt),
    brandCodes: [...form.brandCodes],
    appIds: [...form.appIds],
    // 规范化成 X.Y.Z（去空白、去前导零）
    minVersion: minC === null ? '' : codeToVersion(minC),
    maxVersion: maxC === null ? '' : codeToVersion(maxC),
    userType: form.userType,
    countries: Array.from(new Set(form.countries.map((c) => normalizeCountry(c)).filter((c): c is string => !!c))),
    tabEnabled: b.tabEnabled,
    tabIconUrl: b.tabEnabled ? form.tabIconUrl : '',
    tabText: b.tabEnabled ? form.tabText.trim() : '',
    countdown: b.countdown,
    maskClosable: b.maskClosable,
    closable: b.closable,
    openMode: form.openMode,
    autoplaySeconds: b.autoplaySeconds,
    resumeGapMinutes: b.resumeGapMinutes,
    badge: b.badge,
    cards,
  };
}
