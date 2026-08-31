/**
 * 渠道管理页 —— 复刻原型「渠道管理」视图：
 * 3 大渠道 Tab + 状态筛选 chip + 同步状态 + 新增渠道 + 渠道卡片网格。
 * 顶栏全局搜索（uiStore.search）联动过滤 flavor/包名/PAL_CODE/应用名/线上版本号。
 */
import { useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { qk, useArchiveChannel, useChannels } from '@/hooks/queries';
import { useScopedBrands } from '@/hooks/useScopedBrands';
import { useUiStore } from '@/store/uiStore';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { BRAND_META } from '@/lib/brands';
import type { Channel, ChannelStatus } from '@/lib/types';
import { friendlyNotFoundMessage } from '@/lib/api';
import { BrandTabs } from '@/components/BrandTabs';
import { ChannelCard } from '@/components/ChannelCard';
import { ChannelDrawer } from '@/components/ChannelDrawer';
import { Button } from '@/components/ui';
import { DownloadIcon, PlusIcon, RefreshIcon } from '@/components/icons';
import { apkFileName } from '@/lib/text';
import { cn } from '@/lib/cn';

type Filter = 'all' | 'enabled' | 'disabled';

/** 顺序触发整批最新 APK 下载（同源 /apks 直连，不占内存；浏览器首次会提示允许多文件下载）。 */
function downloadAllApks(urls: string[]) {
  urls.forEach((url, i) => {
    setTimeout(() => {
      const link = document.createElement('a');
      link.href = url;
      link.download = apkFileName(url);
      link.rel = 'noopener';
      document.body.appendChild(link);
      link.click();
      link.remove();
    }, i * 350);
  });
}

const FILTERS: { key: Filter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'enabled', label: '启用中' },
  { key: 'disabled', label: '已停用' },
];

export function ChannelsPage() {
  const { brands, brand } = useScopedBrands();
  const { data: channels, isLoading, isFetching } = useChannels();
  const archive = useArchiveChannel();
  const qc = useQueryClient();
  const canCreate = useAuthStore((s) => s.hasPerm(PERM.CHANNEL_CREATE));

  const currentBrand = useUiStore((s) => s.currentBrand);
  const setCurrentBrand = useUiStore((s) => s.setCurrentBrand);
  const search = useUiStore((s) => s.search);
  const openCreate = useUiStore((s) => s.openCreate);
  const openEdit = useUiStore((s) => s.openEdit);

  // 本页本地筛选态（启用/停用），不污染全局 store
  const [activeFilter, setFilter] = useState<Filter>('all');

  const visible = useMemo(() => {
    let list = (channels ?? []).filter(
      (c) => c.brandCode === currentBrand && c.status !== 'archived',
    );
    if (activeFilter !== 'all') {
      const want: ChannelStatus = activeFilter === 'enabled' ? 'enabled' : 'disabled';
      list = list.filter((c) => c.status === want);
    }
    const q = search.trim().toLowerCase();
    if (q) {
      list = list.filter((c) =>
        [c.flavorName, c.applicationId, c.palCode, c.appName, c.liveVersion ?? ''].some((f) =>
          f.toLowerCase().includes(q),
        ),
      );
    }
    return list;
  }, [channels, currentBrand, activeFilter, search]);

  // 当前可见渠道里「有最新成功包」的下载地址（用于一键下载全部）。
  const apkUrls = useMemo(
    () => visible.map((c) => c.latestApkUrl).filter((u): u is string => !!u),
    [visible],
  );

  function handleArchive(c: Channel) {
    archive.mutate(c.id, {
      onError: (err) => {
        // 越界渠道后端故意返回 404（10-rbac.md「数据权限」）：不能照搬「不存在」类文案，
        // 更常见的原因是角色数据范围被收紧了，而不是数据真的被删。
        alert(friendlyNotFoundMessage(err, '归档失败'));
      },
    });
  }

  return (
    <section>
      {brands && (
        <BrandTabs brands={brands} current={currentBrand} onSelect={setCurrentBrand} />
      )}

      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <div className="flex gap-2 items-center">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setFilter(f.key)}
              className={cn('chip', activeFilter === f.key && 'chip-on')}
            >
              {f.label}
            </button>
          ))}
        </div>
        <div className="ml-auto flex gap-[10px]">
          {apkUrls.length > 0 && (
            <Button
              onClick={() => downloadAllApks(apkUrls)}
              title="下载当前列表中所有渠道的最新 APK（浏览器会提示允许下载多个文件）"
            >
              <DownloadIcon />
              下载全部最新包 ({apkUrls.length})
            </Button>
          )}
          <Button
            onClick={() => {
              void qc.invalidateQueries({ queryKey: qk.channels });
              void qc.invalidateQueries({ queryKey: qk.brands });
            }}
            disabled={isFetching}
          >
            <RefreshIcon />
            {isFetching ? '刷新中…' : '刷新'}
          </Button>
          {canCreate && (
            <Button variant="primary" onClick={openCreate}>
              <PlusIcon />
              新增渠道
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <GridSkeleton />
      ) : brands && brands.length === 0 ? (
        // 数据范围为空的角色（10-rbac.md）：直说原因，别留一片白板让人以为是坏了。
        <div className="text-center text-muted py-[60px]">
          你的角色没有被授予任何品牌数据范围，看不到渠道包。
          <br />
          请让管理员在「角色管理 → 编辑角色 → 数据范围」里配置品牌/渠道范围。
        </div>
      ) : visible.length === 0 ? (
        <div className="text-center text-muted py-[60px]">
          {search.trim() ? '没有匹配搜索的渠道' : '该筛选下暂无渠道'}
        </div>
      ) : (
        <div
          className="grid gap-[14px]"
          style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(290px,1fr))' }}
        >
          {brand &&
            visible.map((c: Channel) => (
              <ChannelCard
                key={c.id}
                channel={c}
                brand={brand}
                onEdit={() => openEdit(c.id)}
                onArchive={() => handleArchive(c)}
              />
            ))}
        </div>
      )}

      {/* 新增/编辑抽屉 */}
      <ChannelDrawer brandMeta={BRAND_META[currentBrand]} />
    </section>
  );
}

function GridSkeleton() {
  return (
    <div className="grid gap-[14px]" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(290px,1fr))' }}>
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="h-[176px] rounded-card bg-panel border border-line animate-pulse" />
      ))}
    </div>
  );
}
