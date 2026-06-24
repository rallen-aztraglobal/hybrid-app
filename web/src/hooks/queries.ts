import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { brandApi, buildApi, channelApi } from '@/lib/api';
import type { BrandCode, BuildJobRequest, ChannelInput, DomainEntry } from '@/lib/types';

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
