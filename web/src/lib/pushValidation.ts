/**
 * 推送活动表单校验（07-push.md §Console）。
 *
 * 规则：
 *  - 标题：必填，最多 50 字
 *  - 正文：必填，最多 500 字
 *  - targetAppIds：至少选 1 个渠道包
 *  - deeplinkPath：若填，须以 / 开头，且不含协议/域名
 */

import type { ListingCampaignInput, PushCampaignInput } from './types';

export interface PushFieldError {
  field: keyof PushCampaignInput | 'targetAppIds';
  message: string;
}

/** 单字段即时校验，返回错误信息（null = 通过）。 */
export function validatePushTitle(v: string): string | null {
  const s = v.trim();
  if (!s) return '标题必填';
  if (s.length > 50) return `标题不超过 50 字（当前 ${s.length} 字）`;
  return null;
}

export function validatePushBody(v: string): string | null {
  const s = v.trim();
  if (!s) return '正文必填';
  if (s.length > 500) return `正文不超过 500 字（当前 ${s.length} 字）`;
  return null;
}

/**
 * deeplink path 校验：
 *  - 若为空，跳过（选填）
 *  - 须以 / 开头
 *  - 不含协议（http/https）或域名（含 . 的 host）
 */
export function validateDeeplinkPath(v: string): string | null {
  const s = v.trim();
  if (!s) return null; // 选填
  if (!s.startsWith('/')) return 'deeplink path 须以 / 开头（如 /promo/618）';
  // 排除含协议的情况
  if (/^https?:\/\//i.test(s)) return '请填相对 path，不含协议（如 /promo/618）';
  // 排除含 host（有 . 的段，如 example.com/path）
  const afterSlash = s.slice(1).split('/')[0];
  if (afterSlash.includes('.')) return '请填相对 path，不含域名（如 /promo/618）';
  return null;
}

/** 综合校验，返回所有错误。 */
export function validatePushCampaign(
  input: PushCampaignInput,
): PushFieldError[] {
  const errors: PushFieldError[] = [];

  const titleErr = validatePushTitle(input.title);
  if (titleErr) errors.push({ field: 'title', message: titleErr });

  const bodyErr = validatePushBody(input.body);
  if (bodyErr) errors.push({ field: 'body', message: bodyErr });

  if (!input.targetAppIds.length) {
    errors.push({ field: 'targetAppIds', message: '请至少选择 1 个渠道包' });
  }

  if (input.deeplinkPath) {
    const dlErr = validateDeeplinkPath(input.deeplinkPath);
    if (dlErr) errors.push({ field: 'deeplinkPath', message: dlErr });
  }

  return errors;
}

/**
 * 上架包推送活动校验（09-listing.md §6）—— 标题/正文/deeplink 复用同一套规则，
 * 目标字段从 targetAppIds 换成 listingIds；受众（强制只投 B 面设备）由后端硬控，前端无需校验。
 */
export interface ListingCampaignFieldError {
  field: keyof ListingCampaignInput | 'listingIds';
  message: string;
}

export function validateListingCampaign(input: ListingCampaignInput): ListingCampaignFieldError[] {
  const errors: ListingCampaignFieldError[] = [];

  const titleErr = validatePushTitle(input.title);
  if (titleErr) errors.push({ field: 'title', message: titleErr });

  const bodyErr = validatePushBody(input.body);
  if (bodyErr) errors.push({ field: 'body', message: bodyErr });

  if (!input.listingIds.length) {
    errors.push({ field: 'listingIds', message: '请至少选择 1 个目标上架包' });
  }

  if (input.deeplinkPath) {
    const dlErr = validateDeeplinkPath(input.deeplinkPath);
    if (dlErr) errors.push({ field: 'deeplinkPath', message: dlErr });
  }

  return errors;
}
