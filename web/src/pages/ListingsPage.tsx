/**
 * 上架包管理页 —— 仿照 ChannelsPage 的列表+抽屉模式：
 * Android / iOS 两个平台 Tab + 状态筛选 chip + 本地搜索 + 新增 + 卡片网格。
 * 数据量小（当前仅 ColorStack/DeckTallyPro 等个位数条目），一次性拉全量、纯前端过滤，
 * 不做服务端分页/防抖。
 */
import { useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { qk, useDeleteListing, useListings } from '@/hooks/queries';
import { PLATFORM_ORDER } from '@/lib/listingMeta';
import type { Listing, ListingPlatform, ListingStatus } from '@/lib/types';
import { PlatformTabs } from '@/components/PlatformTabs';
import { ListingCard } from '@/components/ListingCard';
import { ListingDrawer } from '@/components/ListingDrawer';
import { Button } from '@/components/ui';
import { PlusIcon, RefreshIcon, SearchIcon } from '@/components/icons';
import { cn } from '@/lib/cn';

type Filter = 'all' | 'enabled' | 'disabled';

const FILTERS: { key: Filter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'enabled', label: '启用中' },
  { key: 'disabled', label: '已停用' },
];

type DrawerTarget = 'new' | { id: string } | null;

export function ListingsPage() {
  const { data: listings, isLoading, isFetching } = useListings();
  const del = useDeleteListing();
  const qc = useQueryClient();

  const [platform, setPlatform] = useState<ListingPlatform>('android');
  const [activeFilter, setActiveFilter] = useState<Filter>('all');
  const [search, setSearch] = useState('');
  const [drawerTarget, setDrawerTarget] = useState<DrawerTarget>(null);

  const counts = useMemo(() => {
    const c: Record<ListingPlatform, number> = { android: 0, ios: 0 };
    for (const l of listings ?? []) {
      if (l.status !== 'archived') c[l.platform] += 1;
    }
    return c;
  }, [listings]);

  const visible = useMemo(() => {
    let list = (listings ?? []).filter((l) => l.platform === platform && l.status !== 'archived');
    if (activeFilter !== 'all') {
      const want: ListingStatus = activeFilter === 'enabled' ? 'enabled' : 'disabled';
      list = list.filter((l) => l.status === want);
    }
    const q = search.trim().toLowerCase();
    if (q) {
      list = list.filter((l) => [l.name, l.displayName, l.bundleId].some((f) => f.toLowerCase().includes(q)));
    }
    return list;
  }, [listings, platform, activeFilter, search]);

  function handleDelete(l: Listing) {
    del.mutate(l.id, {
      onError: (err) => {
        alert(err instanceof Error ? err.message : '删除失败');
      },
    });
  }

  return (
    <section>
      <PlatformTabs counts={counts} current={platform} onSelect={setPlatform} />

      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <div className="flex gap-2 items-center">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              onClick={() => setActiveFilter(f.key)}
              className={cn('chip', activeFilter === f.key && 'chip-on')}
            >
              {f.label}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-2 bg-panel border border-line rounded-[10px] px-3 py-2 w-[240px] text-muted">
          <SearchIcon className="w-4 h-4" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索名称 / 包名…"
            className="border-none bg-transparent outline-none w-full text-ink text-[13px] placeholder:text-muted"
          />
        </div>
        <div className="ml-auto flex gap-[10px]">
          <Button
            onClick={() => void qc.invalidateQueries({ queryKey: qk.listings })}
            disabled={isFetching}
          >
            <RefreshIcon />
            {isFetching ? '刷新中…' : '刷新'}
          </Button>
          <Button variant="primary" onClick={() => setDrawerTarget('new')}>
            <PlusIcon />
            新增上架包
          </Button>
        </div>
      </div>

      {isLoading ? (
        <GridSkeleton />
      ) : visible.length === 0 ? (
        <div className="text-center text-muted py-[60px]">
          {search.trim() ? '没有匹配搜索的上架包' : '该筛选下暂无上架包'}
        </div>
      ) : (
        <div
          className="grid gap-[14px]"
          style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(290px,1fr))' }}
        >
          {visible.map((l: Listing) => (
            <ListingCard
              key={l.id}
              listing={l}
              onEdit={() => setDrawerTarget({ id: l.id })}
              onDelete={() => handleDelete(l)}
            />
          ))}
        </div>
      )}

      <ListingDrawer
        target={drawerTarget}
        defaultPlatform={platform}
        onClose={() => setDrawerTarget(null)}
        onCreated={(id) => setDrawerTarget({ id })}
      />
    </section>
  );
}

function GridSkeleton() {
  return (
    <div className="grid gap-[14px]" style={{ gridTemplateColumns: 'repeat(auto-fill,minmax(290px,1fr))' }}>
      {Array.from({ length: PLATFORM_ORDER.length * 2 }).map((_, i) => (
        <div key={i} className="h-[176px] rounded-card bg-panel border border-line animate-pulse" />
      ))}
    </div>
  );
}
