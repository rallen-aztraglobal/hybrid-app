# 12 · 马甲包弹窗模块：后台配置 + 客户端展示 + 埋点统计

> 需求定稿见 [docs/prd/马甲包弹窗模块_需求文档.md](../prd/马甲包弹窗模块_需求文档.md)（v1.3，下称 PRD）。
> 本文是**实现契约**：后端 / 前端 / Android 三方以此为准，字段名、端点、枚举逐字一致。
> 决策取舍见 [ADR-0019](../adr/0019-popup-module.md)。UI 原型见 [ui/popup.html](./ui/popup.html)。

- **马甲包 = 渠道 APK**（`app/` WebView 壳，ap/bp/gp 全部小渠道）。上架包（`listings/`）不在本期范围。
- 后台只管「投不投、投什么、投给谁」；**频控、倒计时时长、便条尺寸/位置、形态交互写死在客户端**（PRD §4.2、§6 可配置边界）。
- 配置改动不需要重新打包；客户端冷启动必拉、回前台距上次拉取 > 5 分钟重拉。

---

## 1. 位置码与能力矩阵

位置码直接用 PRD 编号字符串 `P1`…`P8`。客户端遇到未知位置码**忽略、不崩溃**（PRD §9）。

| 码 | 名称 | 遮罩类 | 卡片数 | 图片 | 文案（card.title） | 便条态 | 倒计时 | 可设强制模式 | 遮罩点击关闭 | 默认位置开关 |
|---|---|---|---|---|---|---|---|---|---|---|
| P1 | 启动弹窗 | ✅ | 1–5（轮播） | 必填（600×800px） | 可选 | 可选，默认开 | 可选，默认关 | ❌ | 可选 | **开** |
| P2 | 悬浮球 | ❌ | 1 | 必填（120×120px） | — | ❌ | ❌ | ❌ | — | **开** |
| P3 | 底部横幅 | ❌ | 1–5（自动轮播） | 可选（缩略图 80×80px） | **必填**（一行） | 可选，默认开 | ❌ | ❌ | — | 关 |
| P4 | 更新 / 公告 | ✅ | 1 | 可选（600×800px） | 无图时必填 | 非强制可选，默认开 | ❌ | ✅（=强制模式） | 可选（强制时恒否） | 关 |
| P5 | 顶部通栏 | ❌ | 1 | 不用 | **必填** | ❌ | ❌ | ❌ | — | 关 |
| P6 | 退出挽留 | ✅ | 1 | 必填（600×800px） | 可选 | ❌ | ❌ | ❌ | 可选 | 关 |
| P7 | 全屏插屏 | ✅ | 1–5（轮播） | 必填（720×1280px） | 可选 | 可选，默认开 | 可选，默认开 | ❌ | —（全屏无遮罩） | 关 |
| P8 | 便条（独立常驻入口） | ❌ | 1 | 必填（图标 40×40px） | **必填**，≤ 8 字符（建议 ≤ 5 字） | — | ❌ | ❌ | — | **开** |

### 1.1 显示尺寸与素材建议尺寸（二倍图）

素材一律按**二倍图**出：建议像素 = 客户端显示 dp × 2。后台上传位逐个标注；上传后尺寸不符**只提示、不拦截**（`upload-image` 返回 `width/height`，前端比对比例与建议值给黄色提示）。客户端按下表 dp 渲染（`centerCrop` 填满，不拉伸变形）。

| 素材 | 显示尺寸（dp） | 建议尺寸（px，二倍图） | 比例 |
|---|---|---|---|
| P1 / P4 / P6 卡片图 | 300×400（屏宽不足时按屏宽 84% 等比缩） | **600×800** | 3:4 |
| P7 全屏图 | 360×640（铺满屏幕，超出部分裁切） | **720×1280** | 9:16 |
| P2 悬浮球 | 60×60 | **120×120** | 1:1 |
| P3 横幅缩略图 | 40×40 | **80×80** | 1:1 |
| P8 便条图标 / 弹窗的「便条图标」 | 20×20（便条高 28） | **40×40** | 1:1 |
| P5 顶部通栏 | 纯文字，无图 | — | — |

服务端保存时**按矩阵归一化**：不适用的行为字段强制回默认（如非 P1/P7 的 `countdown` 恒 false、非 P4 的 `closable` 恒 true、P4 强制模式下 `tabEnabled`/`maskClosable`/`countdown` 恒 false）。超出卡片上限直接 400「最多 N 张」。

> P8 有两种来源：① 其他弹窗关闭后**收起**出来的便条（由该弹窗的 `tabEnabled/tabIconUrl/tabText` 决定）；② 后台单独建的 P8 弹窗（独立常驻入口，单点直接跳转、长按当日关闭）。两者共用同一套便条 UI 与「同时最多 2 条」上限。

---

## 2. 数据模型（GORM，生产靠 AutoMigrate）

### 2.1 `popup` 弹窗

| 列 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | |
| name | varchar(64) | 运营内部名称，必填 |
| position | varchar(8) | `P1`…`P8` |
| enabled | bool | 单个弹窗开关 |
| priority | int | 0–999，越大越优先 |
| start_at / end_at | datetime NULL | 生效期，NULL = 不限；两者都有时 end > start |
| brand_codes | text(JSON 数组) | 定向品牌，空 = 全部 |
| app_ids | text(JSON 数组) | 定向渠道包 applicationId，空 = 全部（ADR-0009：以 applicationId 为身份） |
| min_version_code / max_version_code | int | 版本 `≥ min`、`< max`，0 = 不限；API 收发 `X.Y.Z`，存 `major*10000+minor*100+patch`（与 app/build.gradle 一致） |
| user_type | varchar(8) | `all` / `new` / `old` |
| countries | text(JSON 数组) | ISO-3166 二位大写国家码，空 = 全部 |
| tab_enabled | bool | 关闭后收起为便条 |
| tab_icon_url | varchar(512) | 便条图标，空 = 用第一张卡片图 |
| tab_text | varchar(32) | 便条文案；`tab_enabled` 时必填，≤ 8 字符 |
| countdown | bool | 倒计时关闭（仅 P1/P7），时长客户端写死 3 秒 |
| mask_closable | bool | 遮罩点击关闭 |
| closable | bool | 是否可关闭（仅 P4 可为 false = 强制模式：无 X、遮罩无效、返回键不关弹窗） |
| open_mode | varchar(16) | `webview` / `browser` / `store` |
| autoplay_seconds | int | P3 自动轮播间隔，2–30，默认 5 |
| resume_gap_minutes | int | P1 回前台触发间隔，5–1440，默认 30 |
| badge | bool | P2 红点角标 |
| created_by | varchar(64) | |
| created_at / updated_at / deleted_at | datetime | **软删**：统计仍能显示已删弹窗名称 |

### 2.2 `popup_card` 素材卡片

| 列 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | **卡片 ID，埋点按它记**；编辑时带 id 的卡片原地更新（ID 不变，统计连续），不带 id 的新建，缺席的删除 |
| popup_id | bigint index | |
| sort | int | 轮播顺序（保存时按数组下标重写 0..n-1） |
| image_url | varchar(512) | 必须 http(s)；后台上传产出的 URL |
| link_url | varchar(512) | 空 = 整图不可点。允许：`/path` 站内相对路径（仅 webview 模式，运行时拼当前域名，守 ADR-0002）、`http(s)://`、`market://`（store 模式） |
| button_text | varchar(32) | 空 = 整图可点 |
| title | varchar(64) | |
| description | varchar(255) | |

### 2.3 `popup_position` 位置开关

`code` varchar(8) PK、`enabled` bool、`updated_at`、`updated_by`。启动时 seed 8 行（P1/P2/P8 开，其余关），**已存在的不覆盖**。

### 2.4 `popup_stat_daily` 埋点日聚合

不落原始事件，入库即按天聚合累加（百万级事件/天下单表行数可控）。

唯一键 `(stat_date, popup_id, card_id, card_index, application_id, app_version, event, dim)`，列 `count bigint`。
- `stat_date`：`YYYY-MM-DD`，按 `POPUP_TZ`（默认 `Asia/Manila`）由事件 `ts` 换算；`ts` 超出 `[now-7d, now+1h]` 时改用服务端接收时间。
- `card_id`/`card_index`：卡片级事件填，弹窗级事件为 0。`popup_slide` 的 `card_index` 填 `to`。
- `dim`：`popup_filtered` 填原因 `frequency|mutex|targeting|time`；`popup_close` 填方式 `button|mask|back`；`popup_impression`/`popup_click` 填展示来源 `auto|tab`；其余为空。
- 累加用 `INSERT … ON CONFLICT/ON DUPLICATE KEY UPDATE count = count + ?`（GORM `clause.OnConflict` 兼容 mysql/sqlite）。

### 2.5 `popup_event_batch` 上报幂等

`batch_id` varchar(64) 唯一、`created_at`。同一批次重试直接返回成功、不重复累加；cron 每天清理 7 天前的行。

---

## 3. 管理端 API（JWT；权限点见 §6）

统一 Envelope 响应（`httpx.OK/Fail`）。

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| GET | `/api/popups/positions` | `page:popups` | 8 个位置开关 `[{code,name,enabled,updatedAt,updatedBy}]` |
| PUT | `/api/popups/positions/:code` | `popup:edit` + **全量数据范围** | `{enabled}`；位置开关是全局的，品牌受限账号 403 |
| GET | `/api/popups` | `page:popups` | 列表，query：`position`、`brand`、`status`（`active|scheduled|ended|disabled`）、`keyword`；含 cards |
| GET | `/api/popups/:id` | `page:popups` | 详情 |
| POST | `/api/popups` | `popup:edit` | 新建 |
| PUT | `/api/popups/:id` | `popup:edit` | 全量更新（卡片按 id 合并，见 §2.2） |
| PUT | `/api/popups/:id/enabled` | `popup:edit` | `{enabled}` 行内快速开关 |
| DELETE | `/api/popups/:id` | `popup:edit` | 软删 |
| POST | `/api/popups/upload-image` | `popup:edit` | multipart `file`；png/jpeg/webp/gif，≤ 3MB；返回 `{url,key,width,height,size}` |
| GET | `/api/popups/stats` | `page:popups` | 统计，见 §5 |
| GET | `/api/popups/runtime-preview` | `page:popups` | `?appId=&country=`，返回与 App 端完全相同的 payload，供运营核对「这个包现在会拿到什么」 |

**弹窗 JSON（管理端收发）**

```json
{
  "id": 12, "name": "国庆首存", "position": "P1", "enabled": true, "priority": 100,
  "startAt": "2026-10-01T00:00:00+08:00", "endAt": null,
  "brandCodes": ["ap"], "appIds": [], "minVersion": "1.0.3", "maxVersion": "", "userType": "all", "countries": ["PH"],
  "tabEnabled": true, "tabIconUrl": "", "tabText": "Bonus",
  "countdown": false, "maskClosable": false, "closable": true, "openMode": "webview",
  "autoplaySeconds": 5, "resumeGapMinutes": 30, "badge": false,
  "cards": [{ "id": 34, "sort": 0, "imageUrl": "https://…/popup/images/3fa9c1e2-1760000000123.png",
              "linkUrl": "/promo/deposit", "buttonText": "Claim now", "title": "", "description": "" }],
  "status": "active", "positionEnabled": true,
  "createdBy": "allen", "createdAt": "…", "updatedAt": "…"
}
```

- `status` 服务端计算：`disabled`（弹窗关）> `ended`（end 已过）> `scheduled`（start 未到）> `active`。`positionEnabled` 回显所属位置开关，前端据此提示「位置已关，不会下发」。
- **换图必须换 URL**：上传对象 key 固定为 `popup/images/<sha256 前 12 位>-<unix 毫秒>.<ext>`，每次上传都是新文件名，**从不覆盖**（PRD §6）。
- **数据权限**（与推送活动同口径 ALL-match）：品牌受限账号新建/编辑时 `brandCodes` 必须非空且全在范围内；渠道受限账号还必须给出非空 `appIds` 且全在范围内（复用 `assertAppIDsInScope`）。列表只返回定向完全落在调用者范围内的弹窗（`brandCodes` 为空 = 全品牌，只有全量账号可见）。

---

## 4. App 端 API（公开，无鉴权）

URL 由 `bootstrap.json` 的 `configUrl`（`…/api/app/config`）同源派生：把末段 `/config` 换成 `/popups`、`/popups/events`（与 `DeviceInfoRegistrar` 同规则）。

### 4.1 拉配置 `GET /api/app/popups?appId=<applicationId>`

`Cache-Control: no-store`（地区定向按请求 IP，不可被 CDN 共享缓存）。未知 / 已归档 appId → 404（客户端按拉取失败处理，用缓存）。

```json
{
  "appId": "com.arenaplus.ap01001",
  "configVersion": "9f2c4e1ab37d0c55",
  "serverTime": 1760000000000,
  "tzOffsetMinutes": 480,
  "popups": [{
    "id": 12, "position": "P1", "priority": 100,
    "startAt": 1759248000000, "endAt": 0,
    "minVersionCode": 10003, "maxVersionCode": 0, "userType": "all",
    "openMode": "webview", "closable": true, "maskClosable": false, "countdown": false,
    "tabEnabled": true, "tabIconUrl": "https://…", "tabText": "Bonus",
    "autoplaySeconds": 5, "resumeGapMinutes": 30, "badge": false,
    "cards": [{ "id": 34, "imageUrl": "https://…", "linkUrl": "/promo/deposit",
                "buttonText": "Claim now", "title": "", "description": "" }]
  }]
}
```

- 服务端过滤：位置开关开 ∧ 弹窗开 ∧ 未删 ∧ （`endAt` 未到）∧ 品牌/包定向命中 ∧ 地区命中（GeoIP 判不出国家时，设了地区定向的弹窗**不命中**）。**未到 `startAt` 的照常下发**（客户端提前预热、到点按时间过滤）。
- 版本 / 新老用户 / 生效时间由**客户端**过滤（客户端才知道自身状态，且缓存配置也要能正确判时间）。
- `startAt/endAt` 为毫秒时间戳，0 = 不限；`tabIconUrl` 为空时服务端已回填第一张卡片图。
- `configVersion` = 对 `popups` 数组 JSON 取 sha256 前 16 位 hex（不含 serverTime），任何下发内容变化都会变。
- `serverTime` + `tzOffsetMinutes` 供客户端算「每日」（PRD §11 #3）。

### 4.2 上报埋点 `POST /api/app/popups/events`

```json
{
  "batchId": "6f1c…uuid", "appId": "com.arenaplus.ap01001", "palcode": "ap01001",
  "appVersion": "1.0.3", "versionCode": 10003,
  "events": [
    { "event": "popup_impression", "popupId": 12, "position": "P1", "cardId": 34, "cardIndex": 0,
      "display": "auto", "ts": 1760000001234 },
    { "event": "popup_filtered", "popupId": 13, "position": "P7", "reason": "mutex", "ts": 1760000000900 },
    { "event": "popup_close", "popupId": 12, "position": "P1", "method": "back", "ts": 1760000005000 },
    { "event": "popup_slide", "popupId": 12, "position": "P1", "from": 0, "to": 1, "ts": 1760000003000 }
  ]
}
```

- 每批 ≤ 100 条；`appId` 必须是存在的渠道（否则 400）；未知 `event` 名**静默丢弃**（向前兼容）；`batchId` 重复 → 直接 200（幂等，见 §2.5）。
- 事件名（PRD §7.1，一个不多一个不少）：`popup_trigger` `popup_filtered` `popup_load_fail` `popup_impression` `popup_click` `popup_close` `popup_collapse` `tab_impression` `tab_click` `tab_dismiss` `popup_slide`。
- palcode / 包名 / App 版本在批次级携带，时间戳在事件级（PRD「所有事件需带」）。

---

## 5. 统计口径（`GET /api/popups/stats`）

Query：`from`、`to`（`YYYY-MM-DD`，含，默认近 7 天，跨度 ≤ 92 天）、`brand`、`appId`、`popupId`、`position`。响应按弹窗一行：

```json
{ "tz": "Asia/Manila", "tzOffsetMinutes": 480, "from": "2026-10-03", "to": "2026-10-09",
  "popups": [{
  "popupId": 12, "name": "国庆首存", "position": "P1", "deleted": false,
  "trigger": 1000, "displays": 820, "displaysAll": 900, "impressions": 1310, "clicks": 96, "closes": 640, "loadFails": 12,
  "collapses": 600, "tabImpressions": 610, "tabClicks": 75, "tabDismisses": 40, "slides": 530,
  "filtered": { "frequency": 2100, "mutex": 30, "targeting": 0, "time": 0 },
  "closeByMethod": { "button": 500, "mask": 20, "back": 120 },
  "rates": { "showRate": 0.82, "ctr": 0.073, "closeRate": 0.78, "loadFailRate": 0.012,
             "carouselDepth": 1.6, "recoveryRate": 0.125, "tabAbandonRate": 0.066, "totalCtr": 0.13 },
  "cards": [{ "cardId": 34, "cardIndex": 0, "imageUrl": "…", "impressions": 820, "clicks": 60, "ctr": 0.073 }],
  "daily": [{ "date": "2026-10-01", "trigger": 150, "displays": 120, "impressions": 190, "clicks": 14 }]
}]}
```

| 指标 | 公式 | 说明 |
|---|---|---|
| displays（展示次数） | Σ impression where card_index=0 ∧ dim=`auto` | 自动弹出且真实渲染的次数 |
| 展示率 showRate | displays / trigger | 低 → 工程问题（预加载失败 / 频控） |
| 点击率 ctr | clicks / impressions | 卡片级曝光为分母（PRD §3：卡片级曝光是前提） |
| 关闭率 closeRate | closes / displaysAll（= Σ impression where card_index=0，含便条重开） | 分母用「展示次数」（含便条重开），避免轮播卡片曝光把分母撑大 |
| 加载失败率 | loadFails / trigger | |
| 轮播深度 | impressions / displaysAll | 平均每次展示看到几张卡 |
| 找回率 recoveryRate | tabClicks / collapses | |
| 便条弃用率 | tabDismisses / tabImpressions | |
| 总点击率 totalCtr | (clicks + tabClicks) / impressions | |
| 拦截分布 | filtered 按 reason | |

分母为 0 时比率返回 0（前端按对应分母计数为 0 显示「—」）。数据权限：受限账号只统计其范围内 applicationId 的行；因此受限账号能在统计/运行时预览里看到「投放到自己包上的全品牌弹窗」的名称与卡片，这是预期行为，不算越权（列表仍按 ALL-match 只显示定向完全落在范围内的弹窗）。库里不存在的弹窗（未知 popupId）不出现在统计里。

---

## 6. 权限点（RBAC，追加到 [10-rbac.md](./10-rbac.md) 与 `server/internal/perm`）

| 模块 | code | kind | 说明 |
|---|---|---|---|
| 弹窗管理 | `page:popups` | route | 弹窗列表/详情/数据看板/运行时预览 |
| 弹窗管理 | `popup:edit` | button | 新建/编辑/删除/开关/上传素材/位置开关 |

---

## 7. 客户端规则（Android，`com.hybrid.android.popup`）

### 7.1 拉取、缓存、预热（PRD §6 生效机制）

- **冷启动必拉**；`onStart` 回前台时距上次**成功**拉取 > 5 分钟重拉。拉取不阻塞 WebView（IO 协程）。
- 拉取失败 → 用本地缓存配置；无缓存 → 本次不展示任何弹窗、不记任何事件。
- **素材预热**：拿到新配置立即后台下载全部图片（卡片图 + 便条图标）。图片磁盘缓存 key = 完整 URL 的 sha256，URL 变了自然重下。
- **按弹窗粒度切换**：新配置里某弹窗的图片全部就绪 → 用新版本；未就绪且旧配置里有该弹窗的就绪版本 → 沿用旧版本；都没有 → 本次不参与。**新配置里已不存在的弹窗立即下线**（含其便条）——保证后台关开关能「立即停止」，同时满足「未预热完成沿用上一份」。
- `configVersion` 变化时清理磁盘上不再被任何在用配置引用的素材文件。

### 7.2 时间与「每日」（PRD §11 #3）

- 可信时间 `trustedNow`：拉取成功时记录 `(serverTime, elapsedRealtime, bootCount)`；同一次开机内 `trustedNow = serverTime + (elapsedRealtime - 记录值)`，不受用户改系统时间影响。
- 重启后尚未拉到新配置：用 `本地时间 + 上次记录的 (serverTime - 本地时间) 偏差`；同次开机内若检测到本地墙钟相对 `elapsedRealtime` 跳变 > 2 小时，视为作弊、**不重置每日计数**（沿用上一个 dayKey）。
- `dayKey = floor((trustedNow + tzOffsetMinutes·60000) / 86400000)`。

### 7.3 频控常量（写死，PRD §4.2；按设备记，§11 #2）

| 位置 | 自动展示条件 | 关闭后 |
|---|---|---|
| P1 | 当日未自动展示过（每日 1 次） | 便条态开 → 收起为便条（当日有效）；当日不再自动弹 |
| P2 | 本会话未关闭 | 本会话（进程存活期）不再出现 |
| P3 | 不在「关闭后 24h」窗口内 | 便条态开 → 便条（24h 有效）；24h 内不再自动出现 |
| P4 普通 | 当日未展示过 | 便条态开 → 便条（当日有效） |
| P4 强制 | 每次冷启动 | 不可关闭（无 X、遮罩无效、返回键不关弹窗） |
| P5 | 当日未关闭 | 当日不再出现 |
| P6 | 当日未展示过 ∧ 本会话未展示过 | 直接消失 |
| P7 | 当日未自动展示过 | 同 P1 |
| P8 | 当日未被长按关闭 | — |

- 频控在**真实渲染成功时**写入（比「点击/关闭后写」更稳：展示中被杀进程也不会当日重复弹）。轮播按整个弹窗计一次。
- 便条单点重新展开弹窗**不重复计频控**、不记 trigger、**不重跑倒计时**（用户已看过该素材）；该次展示照常记 impression/click，`display = "tab"`。
- 拦截原因四种都记（PRD §7.1 为准；§5 流程图里定向/时间分支没画「记 filtered」，以 §7.1 的事件定义覆盖）。
- 便条被长按关闭 → 当日不再出现该便条**和该弹窗**（P3 同样到次日）。
- 被拦截时记 `popup_filtered`，reason：版本/新老用户不匹配 = `targeting`，不在生效期 = `time`，频控 = `frequency`，被更高优先级占用 = `mutex`。配置为空时什么都不记。

### 7.4 触发时机与互斥（PRD §4.1 / §4.3）

- **遮罩类（P1/P4/P6/P7）同一时刻只允许一个。** 冷启动在 WebView **首屏加载完成后**（splash 已退、错误页未显示）统一评估 P4/P7/P1 候选：强制 P4 永远最先；其余按 `priority` 降序，平手按位置 P4 > P7 > P1，再平手按 id 升序；只展示第一个，其余记 `mutex`。同位置多个弹窗同理。
- 回前台（非冷启动）且距上次进入后台 > 该 P1 的 `resumeGapMinutes` → 只评估 P1。
- P6：顶层页（WebView 不能后退）按返回键时，若有合格 P6 → 展示并消费这次返回；否则走原「再按一次退出」。P6 关闭后再按返回 → 正常退出流程。
- 非遮罩类（P2/P3/P5/P8）首屏完成后评估，每个位置取优先级最高的一个，其余记 `mutex`；可与遮罩类共存，层级在遮罩之下。
- 遮罩类弹窗显示/隐藏时通知 H5 暂停自身弹窗：
  `window.__HYBRID_NATIVE_POPUP__ = true|false; window.dispatchEvent(new CustomEvent('hybrid:native-popup', {detail:{visible:true|false, position:'P1'}}))`。
- 展示流程：记 `popup_trigger` → 素材就绪检查（未在磁盘缓存则现下，**超时 3 秒**）→ 失败记 `popup_load_fail`、不展示、不计曝光 → 渲染 → 卡片停留 ≥ 1 秒记 `popup_impression`（同一次展示同一卡片只记一次）。

### 7.5 关闭、倒计时、返回键（PRD §2 倒计时 6 条 + §8）

- 倒计时（P1/P7 且 `countdown`）：从第 0 秒起在关闭按钮位置显示数字 3→2→1；3 秒后变为正常尺寸 X（热区 ≥ 44×44dp）；倒计时期间遮罩点击无效、结束后按 `maskClosable`；`onPause` 暂停、`onResume` 续计；时长常量 3 秒。
- **返回键全程可关闭**任何遮罩类弹窗（`method = back`），不受倒计时限制；**唯一例外 P4 强制模式**（PRD §8 第 2 条，2026-10-09 用户再次确认）：返回键不关弹窗，交给原「再按一次退出」流程（连按两次返回仍可退出 App，用户不会被困住）。
- 无倒计时时 X 即时可见，热区 ≥ 44×44dp。除 P4 强制外所有弹窗必须可关闭。
- **P6 退出挽留**（PRD「第二次返回直接退出」）：P6 显示时按返回 = 记 `popup_close`(back) 并**直接退出 App**；用 X / 遮罩关闭 P6 后，下一次顶层返回直接退出，不再出「Press back again to exit」提示。
- 点击 CTA / 整图 → 记 `popup_click` → 关闭弹窗（**不**收起为便条、不记 `popup_close`）→ 跳转。
- 关闭（按钮/遮罩/返回）→ 记 `popup_close` → 便条态开则记 `popup_collapse` 并生成便条。

### 7.6 便条（P8，PRD §2 P8）

- 贴屏幕右边缘，锚定距底部约 25% 屏高（加底部安全区）；高 28dp；展开态 = 图标 + 文案，宽 90–110dp，半透明；静止 3 秒后收缩为仅图标（约 32dp 宽）、不透明度再降；用户触摸页面或 WebView 滚动停止后恢复展开态并重新计时。
- **同时最多 2 条**，上下堆叠；第 3 条出现时替换最旧的一条。便条状态持久化，冷启动后在有效期内恢复。
- **单点** → `tab_click` → 重新展示对应弹窗（独立 P8：直接按链接跳转）。**长按** → 震动 + 提示 "Hidden for today" → `tab_dismiss` → 当日不再出现。单点永远不会关掉便条。
- 便条渲染时记 `tab_impression`（同一便条实例每个进程只记一次）。

### 7.7 跳转（`openMode`）

- `webview`：`/path` 相对路径 → `当前解析出的域名 + path`，并追加 `palcode`（同推送 deeplink 规则），经 `decorateLoadUrl` 后主 WebView 加载；`http(s)://` 绝对地址同样主 WebView 加载。
- `browser`：`ACTION_VIEW` 系统浏览器。`store`：`ACTION_VIEW` 打开 `market://` 或商店 https 链接，失败回落浏览器。
- 链接为空或非法 → 整图不可点，仅可关闭。**不得自动跳转**，一律用户点击触发。

### 7.8 新老用户（PRD §11 #7）

进程启动时检查本地「首次启动记录」：不存在 → 本进程生命周期内为**新用户**并写入记录；存在 → 老用户。

### 7.9 埋点可靠性（PRD §7.3）

- 事件先落本地持久化队列（上限 500 条，超出丢最旧），再批量上报（每批 ≤ 100）。
- 触发上报：入队后 10 秒防抖、进入后台（`onStop`）、冷启动、网络恢复。
- 批次发出前先持久化 `batchId`，失败重试沿用同一 `batchId`（服务端幂等）；2xx 才出队；失败指数退避（上限 5 分钟）。
- 断网启动不崩溃，事件进缓存队列，恢复网络后补传。

---

## 8. PRD §11 决策项落地对照

| # | 默认取值 | 落地 |
|---|---|---|
| 1 | 首批开 P1/P2/P8，其余代码实现但默认关 | `popup_position` seed |
| 2 | 频控按设备 | 客户端 SharedPreferences（无登录态） |
| 3 | 每日以服务端时间为准，跳变 > 2h 不重置 | §7.2 |
| 4 | 便条最多 2 条，超出替换最旧 | §7.6 |
| 5 | P1 默认不开倒计时 | 表单默认值 |
| 6 | P7 默认开倒计时 | 表单默认值 |
| 7 | 新老用户按本地首次启动记录 | §7.8 |
| 8 | 图片走 CDN + 客户端磁盘缓存 | 对象存储公开 URL + §7.1 |
| 9 | P3 默认关，启用前由 H5 让出高度 | 位置开关默认关 |
| 10 | 埋点独立上报 | 现有回传链路是 AppsFlyer/Adjust 归因事件，不适合承载自有库存统计 → 走 §4.2 独立端点 |

---

## 9. 实现补充约定（2026-10-09 实现与评审时确定）

契约正文没写到、在实现与评审中定下来的细节，三端以此为准：

**服务端**
- 公开端点的生效弹窗列表与位置开关有 **15 秒进程内缓存**，后台写操作会立即失效本进程缓存；多实例部署时其它实例最多滞后 15 秒。所以 §7.1 的生效延迟要再加最多 15 秒。
- 埋点入库前按「弹窗 id → 卡片 id 集合」缓存（含软删，TTL 30 秒）校验：未知 `popupId` 的事件**丢弃**；`cardId` 不属于该弹窗时置 0（`card_index` 保留）；`reason/method/display` 不在枚举内的存 `other`；`card_index` 限 0–99。软删弹窗的迟到埋点照常入库。
- 累加按唯一键排序后 upsert，保证 MySQL 并发加锁顺序一致、不死锁。
- 保存：请求里带了不属于本弹窗的卡片 id → 当新建；同时填最低与最高版本时要求最低 < 最高；品牌须存在且支持渠道包，appIds 须是真实渠道。保存时若弹窗已被并发软删 → 404（不会复活）。
- 列表 `brand` 过滤：定向「全品牌」的弹窗也算命中该品牌。
- 统计响应顶层带 `tz`/`tzOffsetMinutes`/`from`/`to`，每行带 `displaysAll`；前端「今日 / 近 7 天」按该时区计算。
- `GET /api/channels` 的 any-of 增加 `page:popups`（弹窗定向、看板筛选、运行时预览要读渠道清单）。
- **存量库**的「运营 / 只读」角色不会自动获得新权限点（沿用历史惯例，避免覆盖管理员的手动裁剪）：上线后需在「角色管理」给运营勾 `page:popups` + `popup:edit`、给只读勾 `page:popups`。超管自动拥有。

**客户端**
- 频控**按位置**记（同位置多个弹窗共用当日额度，如同一天最多只自动弹一个 P1）；便条长按关闭按弹窗记。
- 新用户判定兼容升级：本地无首次启动记录但 `af_install.install_tracked` 已为 true（升级前装过）→ 视为老用户。
- 冷启动评估最多等 3 秒拉取 + 预热；超时用已就绪的缓存配置。配置刷新完成后立即对账：已下线弹窗的便条/非遮罩视图/正在显示的遮罩立即移除。
- 「会话」= 一次任务启动（新 Activity 且非配置变更重建）。配置变更重建（旋转、深色模式等）只恢复 UI（便条、P2/P3/P5、正在显示的遮罩及其倒计时剩余），不记 trigger、不写频控、不重复记曝光。
- 只在 Activity 处于 RESUMED 时渲染（如首启通知权限框在上面时等待），图片解码失败按加载失败处理（记 `popup_load_fail`、不展示、删除坏缓存）。
- P8 独立便条只记 `tab_impression/tab_click/tab_dismiss`；P3 只自动轮播、不支持手动滑动，不记 `popup_slide`。
- 埋点上报：400/413 丢弃该批次（批次本身非法，重试只会堵队头）；404/5xx/网络错误保留并指数退避重试（后端未部署时不丢事件）。
- webview 模式下的站外 http(s) 链接与 H5 内链接同样先交 `BrandStrategy.shouldOverrideUrl` 判定。
- 主框架加载出错、错误页或 splash 重新显示期间，整个弹窗层隐藏、不评估。
