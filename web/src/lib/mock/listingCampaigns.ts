import type { ListingCampaign, ListingCampaignInput, ListingCampaignSendResult } from '../types';
import { mockListingDb } from './listings';

/**
 * 进程内 mock 上架包推送活动（09-listing.md §6）。与 mock/listings.ts、mock/db.ts
 * 保持同一套约定：函数式、深拷贝返回、失败抛普通 Error（由 withFallback 消化）。
 *
 * 真实后端只有 3 个端点（创建 / 列表 / 发送），**没有编辑草稿的接口**——本 mock 与
 * 真实契约保持一致的语义：create 之后活动内容不可再改，只能创建一条新的。
 *
 * B 面受众：按目标上架包的 gateEnabled 状态决定——未开 AB 面网关的包恒为 0
 * （现实中也确实不会有设备被判定过 B），已开的包给一个基于 id 的确定性伪随机数，
 * 避免每次刷新数字乱跳。
 */

let campaigns: ListingCampaign[] = [
  {
    id: 'lc-1',
    kind: 'listing',
    name: 'ColorStack 新关卡推广',
    title: '20 个新关卡上线啦！',
    body: '全新拼图关卡现已开放，点击立即体验',
    imageUrl: undefined,
    deeplinkPath: '/promo/new-levels',
    extraData: { campaign_id: 'colorstack_new_levels' },
    listingIds: ['lst-1'],
    status: 'done',
    sentAt: '2026-07-10T08:00:00Z',
    totalDevices: 1280,
    successCount: 1265,
    failureCount: 15,
    createdBy: 'Daly',
    createdAt: '2026-07-09T10:00:00Z',
  },
];
let seq = 1;

/** B 面活跃设备数（演示用确定性伪随机，未开 AB 面网关的包恒 0）。 */
function bDeviceCountFor(listingId: string): number {
  const l = mockListingDb.get(listingId);
  if (!l || !l.gateEnabled) return 0;
  let h = 0;
  for (const ch of listingId) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return 200 + (h % 1800);
}

export const mockListingCampaignDb = {
  list(): ListingCampaign[] {
    return campaigns
      .map((c) => ({ ...c }))
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  },

  create(input: ListingCampaignInput): ListingCampaign {
    if (!input.listingIds.length) throw new Error('至少选择一个目标上架包');
    seq += 1;
    const c: ListingCampaign = {
      id: `lc-${seq}`,
      kind: 'listing',
      name: input.name.trim() || input.title.trim(),
      title: input.title.trim(),
      body: input.body.trim(),
      imageUrl: input.imageUrl || undefined,
      deeplinkPath: input.deeplinkPath || undefined,
      extraData: input.extraData && Object.keys(input.extraData).length ? input.extraData : undefined,
      listingIds: [...input.listingIds],
      status: 'draft',
      totalDevices: 0,
      successCount: 0,
      failureCount: 0,
      createdBy: 'Daly',
      createdAt: new Date().toISOString(),
    };
    campaigns = [c, ...campaigns];
    return { ...c };
  },

  send(id: string, dryRun: boolean): ListingCampaignSendResult {
    const idx = campaigns.findIndex((c) => c.id === id);
    if (idx < 0) throw new Error('活动不存在');
    const c = campaigns[idx];

    if (dryRun) {
      const byApp: Record<string, number> = {};
      let total = 0;
      for (const lid of c.listingIds) {
        const n = bDeviceCountFor(lid);
        byApp[`listing:${lid}`] = n;
        total += n;
      }
      return { dryRun: true, preview: { totalDevices: total, byApp } };
    }

    if (c.status === 'sending' || c.status === 'done') {
      throw new Error(`活动已处于 ${c.status} 状态，不可重复发送`);
    }
    const total = c.listingIds.reduce((s, lid) => s + bDeviceCountFor(lid), 0);
    campaigns[idx] = {
      ...c,
      status: 'done',
      sentAt: new Date().toISOString(),
      totalDevices: total,
      successCount: Math.floor(total * 0.97),
      failureCount: Math.ceil(total * 0.03),
    };
    return { dryRun: false };
  },
};
