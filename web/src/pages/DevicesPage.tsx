/**
 * 设备管理页（/devices）—— 渠道包内 Android 设备注册流水查看 + CSV 导出。
 * 筛选栏（渠道多选 + 设备/包名关键字 + 注册时间范围）→ 查询后重置到第 1 页；表格支持勾选
 * （跨页保留已选，全选语义=当前页全选）；导出分两种：勾选导出（POST /devices/export，
 * 前端拦截 >1000）与按当前筛选导出全部（GET /devices/export-token → 原生 <a href> 下载）。
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { useChannels, useDevices } from '@/hooks/queries';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { deviceApi } from '@/lib/api';
import type { ChannelDevice, DeviceFilter } from '@/lib/types';
import { Button } from '@/components/ui';
import { MultiSearchSelect } from '@/components/MultiSearchSelect';
import type { SearchSelectOption } from '@/components/SearchSelect';
import { DatePicker } from '@/components/DatePicker';
import { DownloadIcon, SearchIcon } from '@/components/icons';
import { cn } from '@/lib/cn';

const PAGE_SIZE = 50;
const MAX_SELECTED_EXPORT = 1000;

function formatDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString('zh-CN', { hour12: false });
}

/** 表头全选复选框（三态：全不选/部分选/当前页全选），仿 RoleDrawer 的 ref indeterminate 写法。 */
function HeaderCheckbox({
  checked,
  indeterminate,
  onChange,
  disabled,
}: {
  checked: boolean;
  indeterminate: boolean;
  onChange: () => void;
  disabled?: boolean;
}) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = indeterminate;
  }, [indeterminate]);
  return (
    <input
      ref={ref}
      type="checkbox"
      checked={checked}
      disabled={disabled}
      onChange={onChange}
      className="w-[15px] h-[15px] accent-[var(--brand)]"
    />
  );
}

function MonoCell({ value }: { value?: string }) {
  if (!value) return <span className="text-muted">—</span>;
  return (
    <span className="mono text-[12px] text-ink-2 block max-w-[200px] truncate" title={value}>
      {value}
    </span>
  );
}

export function DevicesPage() {
  const { data: channels } = useChannels();
  const canExport = useAuthStore((s) => s.hasPerm(PERM.DEVICE_EXPORT));

  // 筛选栏「待提交」输入态：点「查询」后才生效为真正发请求的 filter（并重置 page=1）。
  const [pendingAppIds, setPendingAppIds] = useState<string[]>([]);
  const [pendingDeviceKw, setPendingDeviceKw] = useState('');
  const [pendingPkgKw, setPendingPkgKw] = useState('');
  const [pendingFrom, setPendingFrom] = useState('');
  const [pendingTo, setPendingTo] = useState('');
  const [pendingActiveFrom, setPendingActiveFrom] = useState('');
  const [pendingActiveTo, setPendingActiveTo] = useState('');
  const [filter, setFilter] = useState<DeviceFilter>({ page: 1, pageSize: PAGE_SIZE });

  const { data, isLoading, isFetching, error } = useDevices(filter);
  const items = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // 多选：id 集合，跨页保留；筛选条件变化后旧集合语义不再适用，随查询提交一并清空。
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [exportErr, setExportErr] = useState<string | null>(null);
  const [exportingSelected, setExportingSelected] = useState(false);
  const [exportingAll, setExportingAll] = useState(false);
  const [pageJump, setPageJump] = useState(String(filter.page));

  useEffect(() => {
    setPageJump(String(filter.page));
  }, [filter.page]);

  // 多选语义下不再需要「全部渠道」空选项：什么都不勾 = 不限。
  const channelOptions: SearchSelectOption[] = useMemo(() => {
    const opts: SearchSelectOption[] = [];
    for (const c of channels ?? []) {
      if (c.status === 'archived') continue;
      opts.push({ value: c.applicationId, label: c.appName, sub: `${c.palCode} · ${c.applicationId}` });
    }
    return opts;
  }, [channels]);

  function handleQuery() {
    setFilter({
      applicationIds: pendingAppIds.length > 0 ? pendingAppIds : undefined,
      device: pendingDeviceKw.trim() || undefined,
      packageName: pendingPkgKw.trim() || undefined,
      from: pendingFrom || undefined,
      to: pendingTo || undefined,
      activeFrom: pendingActiveFrom || undefined,
      activeTo: pendingActiveTo || undefined,
      page: 1,
      pageSize: PAGE_SIZE,
    });
    setSelected(new Set());
    setExportErr(null);
  }

  function goPage(page: number) {
    const clamped = Math.min(Math.max(1, page), totalPages);
    if (clamped === filter.page) return;
    setFilter((f) => ({ ...f, page: clamped }));
  }

  function toggleOne(id: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  const pageIds = useMemo(() => items.map((d) => d.id), [items]);
  const pageAllSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id));
  const pageSomeSelected = !pageAllSelected && pageIds.some((id) => selected.has(id));

  function toggleSelectAllPage() {
    setSelected((prev) => {
      const next = new Set(prev);
      if (pageAllSelected) {
        for (const id of pageIds) next.delete(id);
      } else {
        for (const id of pageIds) next.add(id);
      }
      return next;
    });
  }

  async function handleExportSelected() {
    if (selected.size === 0) return;
    if (selected.size > MAX_SELECTED_EXPORT) {
      setExportErr(`最多支持导出 ${MAX_SELECTED_EXPORT} 台，当前已选 ${selected.size} 台，请减少勾选`);
      return;
    }
    setExportErr(null);
    setExportingSelected(true);
    try {
      await deviceApi.exportDevicesSelected([...selected]);
    } catch (err) {
      setExportErr(err instanceof Error ? err.message : '导出失败');
    } finally {
      setExportingSelected(false);
    }
  }

  async function handleExportAll() {
    setExportErr(null);
    setExportingAll(true);
    try {
      await deviceApi.exportDevicesByFilter({
        applicationIds: filter.applicationIds,
        device: filter.device,
        packageName: filter.packageName,
        from: filter.from,
        to: filter.to,
        activeFrom: filter.activeFrom,
        activeTo: filter.activeTo,
      });
    } catch (err) {
      setExportErr(err instanceof Error ? err.message : '导出失败');
    } finally {
      setExportingAll(false);
    }
  }

  return (
    <section>
      {/* 筛选栏 */}
      <div className="section-card mb-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-[260px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">渠道（可多选）</label>
            <MultiSearchSelect
              values={pendingAppIds}
              onChange={setPendingAppIds}
              options={channelOptions}
              placeholder="全部渠道"
              searchPlaceholder="搜索应用名 / PAL_CODE / 包名…"
            />
          </div>
          <div className="w-[190px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">设备</label>
            <input
              className="field-input"
              value={pendingDeviceKw}
              onChange={(e) => setPendingDeviceKw(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleQuery();
              }}
              placeholder="设备名 / GAID / ADID"
            />
          </div>
          <div className="w-[190px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">包名</label>
            <input
              className="field-input"
              value={pendingPkgKw}
              onChange={(e) => setPendingPkgKw(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleQuery();
              }}
              placeholder="applicationId，支持模糊"
            />
          </div>
          <div className="w-[170px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">注册开始日期</label>
            <DatePicker value={pendingFrom} onChange={setPendingFrom} max={pendingTo || undefined} placeholder="不限" />
          </div>
          <div className="w-[170px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">注册结束日期</label>
            <DatePicker value={pendingTo} onChange={setPendingTo} min={pendingFrom || undefined} placeholder="不限" />
          </div>
          <div className="w-[170px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">活跃开始日期</label>
            <DatePicker
              value={pendingActiveFrom}
              onChange={setPendingActiveFrom}
              max={pendingActiveTo || undefined}
              placeholder="不限"
            />
          </div>
          <div className="w-[170px]">
            <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">活跃结束日期</label>
            <DatePicker
              value={pendingActiveTo}
              onChange={setPendingActiveTo}
              min={pendingActiveFrom || undefined}
              placeholder="不限"
            />
          </div>
          <Button variant="primary" onClick={handleQuery}>
            <SearchIcon className="w-4 h-4" />
            查询
          </Button>

          {canExport && (
            <div className="ml-auto flex items-center gap-2">
              <Button onClick={() => void handleExportSelected()} disabled={selected.size === 0 || exportingSelected}>
                <DownloadIcon className="w-4 h-4" />
                {exportingSelected ? '导出中…' : `导出勾选 (${selected.size})`}
              </Button>
              <Button onClick={() => void handleExportAll()} disabled={exportingAll}>
                <DownloadIcon className="w-4 h-4" />
                {exportingAll ? '导出中…' : '导出全部（按当前筛选）'}
              </Button>
            </div>
          )}
        </div>
        {exportErr && <div className="mt-2 text-[12px] text-down">{exportErr}</div>}
      </div>

      {/* 表格 */}
      {isLoading ? (
        <div className="h-40 rounded-card bg-panel border border-line animate-pulse" />
      ) : error ? (
        <div className="section-card text-center text-down py-10">
          {error instanceof Error ? error.message : '加载失败'}
        </div>
      ) : items.length === 0 ? (
        <div className="section-card text-center text-muted py-[60px]">暂无设备记录</div>
      ) : (
        <div className={cn('section-card !p-0 overflow-hidden', isFetching && 'opacity-70')}>
          <div className="overflow-x-auto">
            <table className="w-full text-[12.5px]">
              <thead>
                <tr className="bg-panel-2 text-ink-2">
                  <th className="w-10 px-3 py-2">
                    <HeaderCheckbox checked={pageAllSelected} indeterminate={pageSomeSelected} onChange={toggleSelectAllPage} />
                  </th>
                  <th className="text-left font-semibold px-3 py-2">设备名称</th>
                  <th className="text-left font-semibold px-3 py-2">GAID</th>
                  <th className="text-left font-semibold px-3 py-2">ADID</th>
                  <th className="text-left font-semibold px-3 py-2">应用名</th>
                  <th className="text-left font-semibold px-3 py-2">PAL_CODE</th>
                  <th className="text-left font-semibold px-3 py-2">注册时间</th>
                  <th className="text-left font-semibold px-3 py-2">最后活跃</th>
                </tr>
              </thead>
              <tbody>
                {items.map((d: ChannelDevice) => (
                  <tr key={d.id} className="border-t border-line-2">
                    <td className="px-3 py-[9px]">
                      <input
                        type="checkbox"
                        checked={selected.has(d.id)}
                        onChange={() => toggleOne(d.id)}
                        className="w-[15px] h-[15px] accent-[var(--brand)]"
                      />
                    </td>
                    <td className="px-3 py-[9px] max-w-[160px] truncate" title={d.deviceName}>
                      {d.deviceName}
                    </td>
                    <td className="px-3 py-[9px]">
                      <MonoCell value={d.gaid} />
                    </td>
                    <td className="px-3 py-[9px]">
                      <MonoCell value={d.adid} />
                    </td>
                    <td className="px-3 py-[9px] max-w-[160px] truncate" title={d.appName}>
                      {d.appName}
                    </td>
                    <td className="px-3 py-[9px] mono text-ink-2">{d.palCode}</td>
                    <td className="px-3 py-[9px] text-ink-2 whitespace-nowrap">{formatDateTime(d.createdAt)}</td>
                    <td className="px-3 py-[9px] text-ink-2 whitespace-nowrap">
                      {d.updatedAt ? formatDateTime(d.updatedAt) : <span className="text-muted">—</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* 分页脚 */}
          <div className="flex items-center gap-3 flex-wrap px-4 py-3 border-t border-line text-[12.5px]">
            <span className="text-muted">共 {total.toLocaleString()} 台</span>
            <div className="ml-auto flex items-center gap-2">
              <Button onClick={() => goPage(filter.page - 1)} disabled={filter.page <= 1 || isFetching}>
                上一页
              </Button>
              <span className="flex items-center gap-1.5 text-ink-2">
                第
                <input
                  className="field-input w-14 text-center px-1 py-1"
                  value={pageJump}
                  onChange={(e) => setPageJump(e.target.value.replace(/[^0-9]/g, ''))}
                  onBlur={() => goPage(Number(pageJump) || 1)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') goPage(Number(pageJump) || 1);
                  }}
                />
                / {totalPages} 页
              </span>
              <Button onClick={() => goPage(filter.page + 1)} disabled={filter.page >= totalPages || isFetching}>
                下一页
              </Button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
