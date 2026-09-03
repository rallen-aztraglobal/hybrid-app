/**
 * 进程内 mock 设备数据（设备管理页，后端未就绪时的事实来源）。
 * 设备挂靠在 mockDb 的真实渠道清单上（跨 3 品牌若干渠道），随机生成 ~200 条：
 * 注册时间散布在近 60 天；gaid 恒非空（无 GAID 不上报），部分 adid 留空模拟「未归因」。
 *
 * 与真实后端语义对齐：GET /api/devices?applicationId=&from=&to=&page=&pageSize=，
 * pageSize 恒 50；翻页远超数据范围时抛出中文错误信息，模拟后端「翻页过深」400。
 */
import type { BrandCode, ChannelDevice, DeviceFilter, DeviceListResult } from '../types';
import { mockDb } from './db';

const DEVICE_MODELS = [
  'SM-G991B',
  'SM-A536E',
  'Pixel 7',
  'Pixel 8 Pro',
  'Redmi Note 12',
  'POCO X5',
  'realme 10',
  'vivo V27',
  'OPPO Reno8',
  'Infinix Hot 30',
  'Tecno Spark 10',
  'Nokia G42',
  'HUAWEI Mate 40',
  'HONOR X9',
  'motorola g84',
];

let seed = 20260817;
/** 简单可复现的伪随机数生成器（LCG），避免每次刷新页面数据集完全跳变影响演示。 */
function rand(): number {
  seed = (seed * 1103515245 + 12345) & 0x7fffffff;
  return seed / 0x7fffffff;
}

function randInt(min: number, max: number): number {
  return Math.floor(rand() * (max - min + 1)) + min;
}

function pick<T>(arr: T[]): T {
  return arr[randInt(0, arr.length - 1)];
}

function randHex(len: number): string {
  const chars = '0123456789abcdef';
  let out = '';
  for (let i = 0; i < len; i++) out += chars[randInt(0, chars.length - 1)];
  return out;
}

/** 形如 GAID/ADID/OAID 的伪 UUID（8-4-4-4-12）。 */
function randUuid(): string {
  return `${randHex(8)}-${randHex(4)}-${randHex(4)}-${randHex(4)}-${randHex(12)}`;
}

const DEVICE_COUNT = 200;

/** 从各品牌真实 mock 渠道清单里各挑若干个作为设备归属渠道（不必覆盖全部渠道）。 */
function pickTargetChannels() {
  const all = mockDb.listChannels().filter((c) => c.status !== 'archived');
  const byBrand: Record<BrandCode, typeof all> = { ap: [], bp: [], gp: [] };
  for (const c of all) byBrand[c.brandCode].push(c);
  const take = (list: typeof all, n: number) => list.slice(0, Math.min(n, list.length));
  return [...take(byBrand.ap, 6), ...take(byBrand.bp, 5), ...take(byBrand.gp, 5)];
}

function buildDevices(): ChannelDevice[] {
  const targetChannels = pickTargetChannels();
  if (targetChannels.length === 0) return [];
  const now = Date.now();
  const out: ChannelDevice[] = [];
  for (let i = 1; i <= DEVICE_COUNT; i++) {
    const ch = pick(targetChannels);
    const daysAgo = randInt(0, 59);
    const msAgo = daysAgo * 86_400_000 + randInt(0, 86_400_000 - 1);
    const createdAt = new Date(now - msAgo).toISOString();
    // 最后活跃时间：注册之后、当前之前的随机一点（约 1/3 设备注册后再没打开过 = 等于注册时间）。
    const updatedAt =
      rand() < 0.33 ? createdAt : new Date(now - randInt(0, Math.max(1, msAgo - 1))).toISOString();
    // 无有效 GAID 的设备整条不上报（客户端+服务端双向约定），故 mock 里 gaid 恒非空；
    // ~25% 未归因 → adid 为空。
    const gaid = randUuid();
    const adid = rand() < 0.25 ? undefined : randUuid();
    const oaid = undefined;
    out.push({
      id: i,
      deviceKey: randUuid(),
      deviceName: `${pick(DEVICE_MODELS)}-${randHex(4)}`,
      gaid,
      adid,
      oaid,
      applicationId: ch.applicationId,
      appName: ch.appName,
      palCode: ch.palCode,
      brandCode: ch.brandCode,
      createdAt,
      updatedAt,
    });
  }
  return out.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
}

const devices: ChannelDevice[] = buildDevices();

/** 深翻页保护缓冲页数（超过「最大可用页 + 缓冲」即视为异常翻页，模拟后端 400）。 */
const DEEP_PAGE_BUFFER = 50;

type MockDeviceFilter = Pick<
  DeviceFilter,
  'applicationIds' | 'device' | 'packageName' | 'from' | 'to' | 'activeFrom' | 'activeTo'
>;

/** 按 DeviceFilter 过滤（不分页），供导出（勾选/按筛选导出）复用。语义对齐后端 applyDeviceFilter：
 * 渠道多选 IN + 设备关键字（设备名/deviceKey/GAID/ADID 模糊）+ 包名模糊 + 注册/活跃时间范围，AND 叠加。 */
function filterDevices(filter: MockDeviceFilter): ChannelDevice[] {
  const deviceKw = (filter.device ?? '').trim().toLowerCase();
  const pkgKw = (filter.packageName ?? '').trim().toLowerCase();
  return devices.filter((d) => {
    if (filter.applicationIds?.length && !filter.applicationIds.includes(d.applicationId)) return false;
    if (deviceKw) {
      const hit = [d.deviceName, d.deviceKey, d.gaid ?? '', d.adid ?? '']
        .some((v) => v.toLowerCase().includes(deviceKw));
      if (!hit) return false;
    }
    if (pkgKw && !d.applicationId.toLowerCase().includes(pkgKw)) return false;
    const day = d.createdAt.slice(0, 10);
    if (filter.from && day < filter.from) return false;
    if (filter.to && day > filter.to) return false;
    const activeDay = (d.updatedAt ?? d.createdAt).slice(0, 10);
    if (filter.activeFrom && activeDay < filter.activeFrom) return false;
    if (filter.activeTo && activeDay > filter.activeTo) return false;
    return true;
  });
}

/** 分页查询（对齐后端 GET /api/devices 语义）；翻页过深抛出中文错误信息。 */
export function queryDevices(filter: DeviceFilter): DeviceListResult {
  const list = filterDevices(filter);
  const total = list.length;
  const pageSize = filter.pageSize || 50;
  const page = Math.max(1, filter.page || 1);
  const maxPage = Math.max(1, Math.ceil(total / pageSize));
  if (page > maxPage + DEEP_PAGE_BUFFER) {
    throw new Error('请求页码超出范围，请缩小筛选条件后重试');
  }
  const start = (page - 1) * pageSize;
  return { items: list.slice(start, start + pageSize), total };
}

export function devicesByIds(ids: number[]): ChannelDevice[] {
  const set = new Set(ids);
  return devices.filter((d) => set.has(d.id));
}

export function devicesByFilter(filter: MockDeviceFilter): ChannelDevice[] {
  return filterDevices(filter);
}
