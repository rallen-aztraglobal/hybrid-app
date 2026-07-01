import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { brandApi, buildApi, channelApi, pushApi, storeApi } from '@/lib/api';
import type {
  BrandCode,
  BuildJobRequest,
  ChannelInput,
  DomainEntry,
  PushCampaignInput,
  PushSendResult,
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
