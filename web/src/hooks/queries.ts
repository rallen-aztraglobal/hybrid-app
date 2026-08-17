import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  brandApi,
  buildApi,
  channelApi,
  deviceApi,
  listingApi,
  listingCampaignApi,
  permsApi,
  pushApi,
  rolesApi,
  storeApi,
  usersApi,
} from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import type {
  AdminUserInput,
  BrandCode,
  BuildJobRequest,
  ChannelInput,
  DeviceFilter,
  DomainEntry,
  ListingCampaignInput,
  ListingCampaignSendResult,
  ListingInput,
  PushCampaignInput,
  PushSendResult,
  RoleInput,
  StoreInput,
  StoreUpdateInput,
} from '@/lib/types';

/**
 * 服务端状态层（TanStack Query）。组件只消费这些 hook，不直接碰 api 模块，
 * 便于缓存、失效、乐观更新统一管理。
 */

export const qk = {
  brands: ['brands'] as const,
  brandDomains: (code: BrandCode) => ['brands', code, 'domains'] as const,
  channels: ['channels'] as const,
  channel: (id: string) => ['channels', id] as const,
  builds: (brand?: BrandCode) => ['builds', brand ?? 'all'] as const,
  buildLogs: (jobId: string) => ['builds', 'logs', jobId] as const,
  pushStatus: ['push', 'status'] as const,
  pushCampaigns: (brand?: BrandCode) => ['push', 'campaigns', brand ?? 'all'] as const,
  pushCampaign: (id: string) => ['push', 'campaigns', id] as const,
  pushAudience: (appIds: string[]) => ['push', 'audience', ...appIds.slice().sort()] as const,
  stores: ['stores'] as const,
  listings: ['listings'] as const,
  listingGateLogs: (id: string) => ['listings', id, 'gateLogs'] as const,
  listingCampaigns: ['push', 'listing-campaigns'] as const,
  permsCatalog: ['perms', 'catalog'] as const,
  roles: ['roles'] as const,
  users: ['users'] as const,
  devices: (filter: DeviceFilter) =>
    ['devices', filter.applicationId ?? '', filter.from ?? '', filter.to ?? '', filter.page, filter.pageSize] as const,
};

export function useBrands() {
  return useQuery({ queryKey: qk.brands, queryFn: brandApi.list });
}

export function useChannels() {
  return useQuery({ queryKey: qk.channels, queryFn: channelApi.list });
}

export function useChannel(id: string | null) {
  return useQuery({
    queryKey: id ? qk.channel(id) : ['channels', 'none'],
    queryFn: () => channelApi.get(id!),
    enabled: !!id,
  });
}

export function useBrandDomains(code: BrandCode) {
  return useQuery({ queryKey: qk.brandDomains(code), queryFn: () => brandApi.getDomains(code) });
}

export function useBuildJobs(brand?: BrandCode) {
  return useQuery({ queryKey: qk.builds(brand), queryFn: () => buildApi.listJobs(brand) });
}

/** 应用商店清单（含 disabled）。渠道表单只取 enabled 项，设置页展示全部。 */
export function useStores() {
  return useQuery({ queryKey: qk.stores, queryFn: storeApi.list });
}

// ---------- mutations ----------

export function useSaveChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: ChannelInput }) =>
      id ? channelApi.update(id, input) : channelApi.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.channels });
      void qc.invalidateQueries({ queryKey: qk.brands });
    },
  });
}

export function useArchiveChannel() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => channelApi.archive(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.channels });
      void qc.invalidateQueries({ queryKey: qk.brands });
    },
  });
}

export function useSaveBrandDomains(code: BrandCode) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (domains: DomainEntry[]) => brandApi.updateDomains(code, domains),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.brandDomains(code) });
      void qc.invalidateQueries({ queryKey: qk.brands });
    },
  });
}

// ---------- 应用商店 mutations ----------

export function useCreateStore() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: StoreInput) => storeApi.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.stores });
    },
  });
}

export function useUpdateStore() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: StoreUpdateInput }) => storeApi.update(id, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.stores });
    },
  });
}

export function useDeleteStore() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => storeApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.stores });
    },
  });
}

export function useSubmitBuildJob() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (req: BuildJobRequest) => buildApi.submitJob(req),
    onSuccess: (job) => {
      void qc.invalidateQueries({ queryKey: qk.builds(job.brandCode) });
      void qc.invalidateQueries({ queryKey: qk.builds() });
    },
  });
}

// =========================================================================
// 推送管理查询 hooks（07-push.md）
// =========================================================================

export function usePushStatus() {
  return useQuery({ queryKey: qk.pushStatus, queryFn: pushApi.getStatus, staleTime: 60_000 });
}

export function usePushCampaigns(brand?: BrandCode) {
  return useQuery({
    queryKey: qk.pushCampaigns(brand),
    queryFn: () => pushApi.listCampaigns(brand),
  });
}

export function usePushCampaign(id: string | null) {
  return useQuery({
    queryKey: id ? qk.pushCampaign(id) : ['push', 'none'],
    queryFn: () => pushApi.getCampaign(id!),
    enabled: !!id,
  });
}

export function usePushAudience(appIds: string[]) {
  return useQuery({
    queryKey: qk.pushAudience(appIds),
    queryFn: () => pushApi.getAudience(appIds),
    enabled: appIds.length > 0,
    staleTime: 30_000,
  });
}

export function useSavePushCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: PushCampaignInput }) =>
      id ? pushApi.updateCampaign(id, input) : pushApi.createCampaign(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['push', 'campaigns'] });
    },
  });
}

export function useSendPushCampaign() {
  const qc = useQueryClient();
  return useMutation<PushSendResult, Error, { id: string; dryRun?: boolean }>({
    mutationFn: ({ id, dryRun }) => pushApi.sendCampaign(id, dryRun),
    onSuccess: (result) => {
      // dry-run 时 campaign 状态未变、历史列表不应出现新完成记录，不做 invalidate。
      // 真发时刷新列表 + 详情，让状态更新可见。
      if (!result.dryRun) {
        void qc.invalidateQueries({ queryKey: ['push', 'campaigns'] });
        void qc.invalidateQueries({ queryKey: qk.pushCampaign(result.campaign.id) });
      }
    },
  });
}

// =========================================================================
// 上架包管理查询 hooks（09-listing.md / ADR-0014）
// =========================================================================

/** 全量上架包（含品牌/域名/网关）；数据量小，页面自行按平台/状态/关键词过滤，不做服务端分页。 */
export function useListings() {
  return useQuery({ queryKey: qk.listings, queryFn: listingApi.list });
}

export function useSaveListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: ListingInput }) =>
      id ? listingApi.update(id, input) : listingApi.create(input),
    // 用 onSettled（而非 onSuccess）：保存是多步提交（基本信息→域名→网关规则→总开关，
    // 见 lib/api.ts applyListingSideEffects），后面某一步失败时前面的步骤已经落库，
    // 缓存也要刷新以反映「部分保存成功」的真实后端状态，不能让 UI 停留在旧快照上。
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: qk.listings });
    },
  });
}

export function useDeleteListing() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => listingApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.listings });
    },
  });
}

/** 网关试算（后台自查工具）：每次调用都是独立一次性动作，用 mutation 而非 query。 */
export function useTestListingGate() {
  return useMutation({
    mutationFn: ({ id, ip, timezone }: { id: string; ip: string; timezone: string }) =>
      listingApi.testGate(id, ip, timezone),
  });
}

/** 判定流水，惰性加载（enabled 由调用方控制，避免抽屉一打开就多发一个请求）。 */
export function useListingGateLogs(id: string | null, enabled: boolean) {
  return useQuery({
    queryKey: id ? qk.listingGateLogs(id) : ['listings', 'none', 'gateLogs'],
    queryFn: () => listingApi.gateLogs(id!, 100),
    enabled: !!id && enabled,
  });
}

export function useSchedulePushCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, scheduledAt }: { id: string; scheduledAt: string }) =>
      pushApi.scheduleCampaign(id, scheduledAt),
    onSuccess: (campaign) => {
      void qc.invalidateQueries({ queryKey: ['push', 'campaigns'] });
      void qc.invalidateQueries({ queryKey: qk.pushCampaign(campaign.id) });
    },
  });
}

// =========================================================================
// 上架包推送查询 hooks（09-listing.md §6）
// =========================================================================

/** 上架包推送活动列表（kind=listing，与渠道推送历史分开展示）。 */
export function useListingCampaigns() {
  return useQuery({ queryKey: qk.listingCampaigns, queryFn: listingCampaignApi.listCampaigns });
}

/** 创建上架包推送活动草稿。后端无编辑端点，创建成功后内容视为锁定。 */
export function useCreateListingCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ListingCampaignInput) => listingCampaignApi.createCampaign(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.listingCampaigns });
    },
  });
}

/** 发送（dry-run 预览 / 真发）。统一用 onSettled 失效列表缓存，保证 status/统计数字与后端一致。 */
export function useSendListingCampaign() {
  const qc = useQueryClient();
  return useMutation<ListingCampaignSendResult, Error, { id: string; dryRun: boolean }>({
    mutationFn: ({ id, dryRun }) => listingCampaignApi.sendCampaign(id, dryRun),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: qk.listingCampaigns });
    },
  });
}

// =========================================================================
// RBAC：权限点清单 + 角色 + 用户（10-rbac.md）
// =========================================================================

/** 权限点清单（登录即可），渲染角色管理的勾选树；变化极少，长缓存。 */
export function usePermsCatalog() {
  return useQuery({ queryKey: qk.permsCatalog, queryFn: permsApi.catalog, staleTime: 5 * 60_000 });
}

/** 角色列表（需 role:manage，仅在角色管理分区渲染时才会被组件挂载触发请求）。 */
export function useRoles() {
  return useQuery({ queryKey: qk.roles, queryFn: rolesApi.list });
}

export function useSaveRole() {
  const qc = useQueryClient();
  const refreshMe = useAuthStore((s) => s.refreshMe);
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: RoleInput }) => (id ? rolesApi.update(id, input) : rolesApi.create(input)),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.roles });
      // 编辑的可能正是当前登录用户所属的角色（权限即时收紧/放宽）：立即拉最新 perms 自愈
      // 侧边栏/按钮显隐，不必等下一次 401 才被踢、或手动刷新页面。
      void refreshMe();
    },
  });
}

export function useDeleteRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => rolesApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.roles });
    },
  });
}

/** 用户列表（需 user:manage）。 */
export function useUsers() {
  return useQuery({ queryKey: qk.users, queryFn: usersApi.list });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: AdminUserInput) => usersApi.create(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.users });
      // 新用户挂到某角色下，该角色 userCount 变化。
      void qc.invalidateQueries({ queryKey: qk.roles });
    },
  });
}

export function useUpdateUserRole() {
  const qc = useQueryClient();
  const refreshMe = useAuthStore((s) => s.refreshMe);
  return useMutation({
    mutationFn: ({ id, roleId }: { id: string; roleId: string }) => usersApi.updateRole(id, roleId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.users });
      void qc.invalidateQueries({ queryKey: qk.roles });
      // 可能改的正是当前登录用户自己的角色：立即拉最新 perms 自愈界面，不必等下一次 401。
      void refreshMe();
    },
  });
}

export function useResetUserPassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) => usersApi.resetPassword(id, password),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => usersApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.users });
      void qc.invalidateQueries({ queryKey: qk.roles });
    },
  });
}

// =========================================================================
// 设备管理（GET /api/devices）
// =========================================================================

/** 设备列表（分页 + 筛选）；keepPreviousData 让翻页时旧数据先留在屏幕上，不闪白。 */
export function useDevices(filter: DeviceFilter) {
  return useQuery({
    queryKey: qk.devices(filter),
    queryFn: () => deviceApi.listDevices(filter),
    placeholderData: keepPreviousData,
  });
}
