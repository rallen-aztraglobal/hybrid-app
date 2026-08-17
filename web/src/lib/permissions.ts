/**
 * 权限点 code 常量 —— 与 docs/admin/10-rbac.md 「权限点清单」表逐字一致。
 * 前端所有路由 / 按钮权限判断统一引用这里的常量，避免散落的裸字符串打错、
 * 与后端契约漂移。PERM_CATALOG 同时充当 mock 的 catalog 数据源（真实后端
 * 就绪前 GET /api/perms/catalog 回退到这里）。
 */
import type { PermCatalogModule } from './types';

export const PERM = {
  PAGE_CHANNELS: 'page:channels',
  CHANNEL_CREATE: 'channel:create',
  CHANNEL_EDIT: 'channel:edit',
  CHANNEL_ARCHIVE: 'channel:archive',

  PAGE_DOMAINS: 'page:domains',
  DOMAIN_EDIT: 'domain:edit',

  PAGE_PACK: 'page:pack',
  BUILD_SUBMIT: 'build:submit',

  PAGE_BUILDS: 'page:builds',

  PAGE_PUSH: 'page:push',
  PUSH_CREATE: 'push:create',
  PUSH_SEND: 'push:send',
  PUSH_CONFIG: 'push:config',

  PAGE_LISTINGS: 'page:listings',
  LISTING_EDIT: 'listing:edit',
  LISTING_GATE: 'listing:gate',

  PAGE_SETTINGS: 'page:settings',
  STORE_MANAGE: 'store:manage',
  USER_MANAGE: 'user:manage',
  ROLE_MANAGE: 'role:manage',
} as const;

export type PermCode = (typeof PERM)[keyof typeof PERM];

/** 权限点清单（module 分组），与 10-rbac.md 表格逐字一致（code / kind 不可与后端漂移）。 */
export const PERM_CATALOG: PermCatalogModule[] = [
  {
    module: 'channels',
    label: '渠道管理',
    perms: [
      { code: PERM.PAGE_CHANNELS, label: '渠道列表/详情/res.zip/最新包下载', kind: 'route' },
      { code: PERM.CHANNEL_CREATE, label: '新增渠道', kind: 'button' },
      { code: PERM.CHANNEL_EDIT, label: '编辑渠道、上传图标/启动图', kind: 'button' },
      { code: PERM.CHANNEL_ARCHIVE, label: '归档（删除）渠道', kind: 'button' },
    ],
  },
  {
    module: 'domains',
    label: '域名管理',
    perms: [
      { code: PERM.PAGE_DOMAINS, label: '品牌/渠道域名查看', kind: 'route' },
      { code: PERM.DOMAIN_EDIT, label: '修改品牌域名、渠道域名', kind: 'button' },
    ],
  },
  {
    module: 'pack',
    label: '打包中心',
    perms: [
      { code: PERM.PAGE_PACK, label: '打包提交页（manifest/当前版本查询）', kind: 'route' },
      { code: PERM.BUILD_SUBMIT, label: '提交打包任务', kind: 'button' },
    ],
  },
  {
    module: 'builds',
    label: '构建记录',
    perms: [{ code: PERM.PAGE_BUILDS, label: '构建记录/日志查看', kind: 'route' }],
  },
  {
    module: 'push',
    label: '推送管理',
    perms: [
      { code: PERM.PAGE_PUSH, label: '推送状态/活动/受众查看（含上架包推送活动）', kind: 'route' },
      { code: PERM.PUSH_CREATE, label: '新建/编辑活动、上传图片', kind: 'button' },
      { code: PERM.PUSH_SEND, label: '发送/定时发送（含上架包活动发送）', kind: 'button' },
      { code: PERM.PUSH_CONFIG, label: '上传 google-services.json', kind: 'button' },
    ],
  },
  {
    module: 'listings',
    label: '上架包',
    perms: [
      { code: PERM.PAGE_LISTINGS, label: '上架包列表/详情/判定流水查看', kind: 'route' },
      { code: PERM.LISTING_EDIT, label: '新建/编辑/删除上架包、改域名', kind: 'button' },
      { code: PERM.LISTING_GATE, label: 'AB 面网关配置与试算', kind: 'button' },
    ],
  },
  {
    module: 'settings',
    label: '系统设置',
    perms: [
      { code: PERM.PAGE_SETTINGS, label: '设置页（商店/运行时预览等查看）', kind: 'route' },
      { code: PERM.STORE_MANAGE, label: '商店管理写操作', kind: 'button' },
      { code: PERM.USER_MANAGE, label: '用户管理（增删改/重置密码）', kind: 'button' },
      { code: PERM.ROLE_MANAGE, label: '角色管理（增删改）', kind: 'button' },
    ],
  },
];

/** 全部权限点 code（超级管理员角色即拥有这个完整集合）。 */
export const ALL_PERM_CODES: string[] = PERM_CATALOG.flatMap((m) => m.perms.map((p) => p.code));

/** 路由权限 → 路径映射顺序（侧边栏/路由守卫按此顺序过滤 + 决定「第一个有权限的页面」）。 */
export const ROUTE_PERM_ORDER: { path: string; perm: string }[] = [
  { path: '/channels', perm: PERM.PAGE_CHANNELS },
  { path: '/listings', perm: PERM.PAGE_LISTINGS },
  { path: '/domains', perm: PERM.PAGE_DOMAINS },
  { path: '/push', perm: PERM.PAGE_PUSH },
  { path: '/pack', perm: PERM.PAGE_PACK },
  { path: '/builds', perm: PERM.PAGE_BUILDS },
  { path: '/settings', perm: PERM.PAGE_SETTINGS },
];
