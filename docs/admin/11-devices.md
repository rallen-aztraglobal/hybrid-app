# 11 · 设备管理:GAID/ADID 上报与受众导出

> 渠道 APK 启动时上报设备广告标识(GAID / Adjust ADID),Console「设备管理」页
> 按渠道 + 注册时间筛选、多选/全选、导出一份带全部列的 CSV,供运营上传 Meta / TikTok
> 做设备 ID 受众(Custom Audience)投放。**无有效 GAID(无 GMS/用户 opt-out 全 0)的设备
> 整条不上报、服务端也拒收**——GAID 是受众上传的唯一有效标识,没有它这条数据毫无用处。
> 决策取舍见 [ADR-0015](../adr/0015-device-registry.md)。

## 为什么要哈希列(调研结论)

Meta 与 TikTok 都支持上传「设备广告 ID 名单」创建自定义受众/相似受众:

- **Meta(Facebook)**:官方要求 MAID(GAID/IDFA)**不哈希**、保留连字符、**全小写**直接上传。
  SHA256 只用于邮箱/手机号等其它身份字段。→ 用导出 CSV 的 `gaid` 列。
- **TikTok**:GAID 支持 原文 / MD5 / SHA256 三种口径,哈希前需统一大小写(我们统一小写)。
  → 用导出 CSV 的 `gaid_sha256` 列(`hex(sha256(小写 gaid))`)。

SHA256 **导出时流式计算,不落库**——库里只存小写 GAID 原文,口径变化(比如以后要 MD5)不用洗库。

## 上报协议(APK 端,公开无鉴权)

```
POST /api/app/device/register
{
  "appId":      BuildConfig.APPLICATION_ID,   // 身份键,服务端校验渠道存在否则 400
  "palcode":    BuildConfig.PAL_CODE,          // 仅冗余展示,不作身份键(ADR-0009)
  "deviceKey":  "<安装 UUID>",                 // 客户端持久化,一次安装一个,upsert 唯一键
  "deviceName": "samsung SM-A515F",            // Build.MANUFACTURER + Build.MODEL
  "gaid":       "384…d",                       // 小写;必填——为空/全 0 时客户端整条不发,服务端 400 拒收
  "adid":       "acf…"                         // Adjust ADID,首启可能为空,二次启动补报
}
```

- 客户端实现:`app/src/main/java/com/hybrid/android/track/DeviceInfoRegistrar.kt`,
  触发点在 WebViewActivity onCreate(与 FCM token 上报同批,IO 协程,零启动影响)。
- **节流**:payload 哈希 + 24h 存 SharedPreferences,内容不变且未超 24h 不重发;
  HTTP 2xx 才记账,失败下次启动自动重试;adid 参与哈希,归因完成后自动补报。
- **写入量级**:百万日活 × 24h 节流 ≈ 日均十几 QPS、峰值百余 QPS,单机 MySQL 直扛,无需队列。
- 服务端 upsert 语义:按 `device_key` 合并——adid/device_name **来值非空才覆盖**,
  渠道归属字段(application_id/brand/palcode/app_name 快照)覆盖为最新,`created_at`(注册时间)
  永不覆盖。无有效 GAID(空/全 0)的上报直接 400 拒收,不落库。

## Console 端(RBAC 权限点见 10-rbac.md)

- `GET /api/devices?applicationId=&from=&to=&page=&pageSize=`(`page:devices`):
  服务端分页,pageSize 默认 50 钳 200,offset 钳 10 万(超出 400,提示缩小筛选或用导出);
  from/to 为 `YYYY-MM-DD`,语义 `[from, to+1d)`;排序 `created_at DESC, id DESC`。
- 导出两条通道(均需 `device:export`),**默认导出美化 XLSX**,原始 CSV 通道保留:
  - **勾选导出**:`POST /api/devices/export` 传 `{ids(≤1000), format?}`,`format` 缺省/`"xlsx"`
    → 美化 Excel,`"csv"` → 原始 CSV;前端 fetch→Blob 下载;
  - **按筛选全量导出**(可达百万行):先 `GET /api/devices/export-token` 拿 5 分钟
    scope=`device_export` 的短 token,再 `<a href="/api/devices/export.xlsx?token=…&筛选参数">`
    浏览器原生下载(`export.csv` 同 token 可用);服务端 `Rows()` 游标流式读,内存可控。

## 导出格式

**XLSX(默认,Console 两个导出按钮都用它)**:中文表头加粗白字深蓝底、冻结首行、
预设列宽、每 100 万行自动分 sheet(Excel 单 sheet 上限 1,048,576 行)。列:

```
设备名称 | GAID(Meta 直接上传) | GAID SHA256(TikTok) | Adjust ADID | 应用名 | PAL_CODE | 包名(applicationId) | 品牌 | 注册时间
```

**CSV(原始通道,给直接上传平台/脚本处理的场景)**:英文表头 + UTF-8 BOM:

```
device_name,gaid,gaid_sha256,adid,oaid,app_name,palcode,application_id,brand,created_at
```

- 运营自行拆列:Meta 上传取 GAID 原文列(不哈希),TikTok 取 SHA256 列。
  库内所有设备均带有效 GAID(无 GAID 不上报),两列恒非空,可直接拆列上传。
- CSV 的 `oaid` 为保留列(当前客户端不采集,恒空);XLSX 已不含该列。
- 注册时间格式 `2006-01-02 15:04:05`(服务器时区)。

## 上线运维

- 表 `channel_device` 由生产 AutoMigrate 自动建(migrations/000011 仅留档)。
- **已存在的角色不会自动获得新权限点**:上线后需管理员在「角色管理」给相关角色勾上
  `page:devices` / `device:export`。
