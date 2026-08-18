# 上架包模块（Listing）与 AB 面网关

> 决策见 [ADR-0014](../adr/0014-listing-ab-gate.md)。本文是实现契约：数据模型、API、客户端接入。

## 1. 是什么

「上架包」= 正式上架 Google Play / App Store 的独立合规应用，与 3 个品牌的渠道 APK（`channels/*.csv` + flavor）是**两条独立产线**。当前三个：

| 名称 | 技术栈 | 上架平台 | Bundle ID |
| --- | --- | --- | --- |
| ColorStack | Flutter | **仅 Android**（Google Play）| `com.vividnest.colorstack5821` |
| DeckTallyPro | 原生 iOS Swift | **仅 iOS**（App Store，id `6780248860`）| `com.deck.tallypro` |
| Hexa Color Sort | Flutter | **仅 Android**（Google Play）| `com.slatecove.hexasort4173` |

> 共 **3 个上架包**（Console 里建 3 条 listing）。ColorStack / Hexa Color Sort 只发 Android、
> DeckTallyPro 只发 iOS，故不存在 ColorStack 的 iOS 上架。推送 Firebase 项目 **`hybrid-listings-51660`**（旧 `hybrid-listings` 已删）。

每个上架包本体是干净小游戏（**A 面**）。开启「AB 面」后，命中放行规则的设备才访问配置的 web 链接（**B 面**，与品牌 ap/bp/gp 同一套域名）。

## 2. AB 面判定（核心）

判定**全在服务端**，客户端不内置任何 B 面地址。判定顺序，任一步判 A 即短路：

| # | 条件 | 结果 |
| --- | --- | --- |
| 1 | `gate_enabled=false`（总开关） | A |
| 2 | 国家 ∈ {CN, US}（硬编码，无视配置）| A |
| 3 | 命中 IP 黑名单 `ip_deny_cidrs` | A |
| 4 | GeoIP 解析不出国家 | A |
| 5 | 国家 ∉ 白名单 `countries`（必填）| A |
| 6 | `timezones` 非空且时区不在其中 | A |
| 7 | `ip_allow_cidrs` 非空且 IP 不在其中 | A |
| 8 | 以上全部通过 | **B** |

条件之间一律 **AND**。**默认安全**：任何不确定情形（库缺失、IP 取不到、包未知、判 B 但无域名）一律 A。

- `countries` **必填非空**；空 = 配置无效（判 A），不是「不限国家」。
- CN/US **不可加入白名单**（服务端 `model.ForcedACountries` 硬编码强制 A，前端也不给选项）。
- `timezones` 客户端上报、可伪造 → 只作叠加收紧，永不单独作准。

## 3. 数据模型（migration 000006）

- `listing_app`：`brand_id`（继承品牌域名）· `(platform,bundle_id)` 唯一 · `gate_enabled` 总开关 · `open_mode`（B 面打开方式：`internal` 内开=原生 WebView / `external` 外开=系统浏览器，默认 internal）· `af_dev_key/af_app_id` · `adjust_app_token/adjust_events`（复用 ADR-0013）· `use_brand_domains`。
- `listing_domain`：B 面域名覆盖，`position` 0 主 / 1..n 备（`use_brand_domains=false` 时生效）。
- `listing_gate`：`countries / timezones / ip_allow_cidrs / ip_deny_cidrs`（一对一）。
- `listing_gate_log`：判定流水 `ip/country/timezone/decision/reason`（排查用）。

## 4. API

### 公开（客户端 App）

```
POST /api/app/listing/gate                 不缓存（Cache-Control: no-store）
  req  { "platform":"android|ios", "bundleId":"...", "timezone":"Asia/Manila" }
  resp { "mode":"A" }  或  { "mode":"B", "url":"https://arenaplus.ph", "openMode":"internal|external" }
```

真实 IP 由服务端从可信代理链提取（`X-Forwarded-For`，见 [realip.go](../../server/internal/httpx/realip.go)），**客户端不传 IP**。响应**绝不含**判定原因/国家/命中规则。

`openMode` **仅 mode=B 时下发**（A 面响应不含），取自 `listing_app.open_mode`：`internal`=客户端原生 WebView 内开、`external`=客户端唤起系统浏览器外开；缺省/非法客户端一律按 `internal` 处理。它只决定「B 面 URL 怎么打开」，不影响 A/B 判定本身。

### 管理面（JWT，viewer 读 / operator 写）

```
GET    /api/listings?platform=&status=&q=
POST   /api/listings                        新建（gate_enabled 恒 false）
GET    /api/listings/:id
PUT    /api/listings/:id                     改；打开 gateEnabled 前强制校验国家白名单非空
DELETE /api/listings/:id
PUT    /api/listings/:id/domains             { inheritBrand, domains:[{position,url,enabled}] }
PUT    /api/listings/:id/gate                { countries, timezones, ipAllowCidrs, ipDenyCidrs }
POST   /api/listings/:id/gate/test           { ip, timezone } → 后台试算，返回 {mode,reason,country}
GET    /api/listings/:id/gate/logs?limit=    判定流水
```

`gate/test` 是后台自查，**返回原因**（与线上端点相反），供运营保存前用某 IP 预览判定。

**调用顺序约束**：打开总开关（`PUT /listings/:id {gateEnabled:true}`）前，网关规则必须已落库——
后端 `assertGateReadyToEnable` 会重新拉取 `Gate` 校验国家白名单非空。因此保存流程应为
**先 `PUT .../gate` 存规则，再 `PUT /listings/:id` 开开关**，不可合并成一次调用（否则命中 400）。
Console 前端已按此顺序提交（域名 → 网关规则 → 开关）。

## 5. 客户端接入（已落地在本仓库）

三个 App 源码已收进仓库并完成接入，逻辑同构：启动请求 gate → A 面走应用原有首页（一行不改）→ B 面按 `openMode` 打开：`internal` 推全屏原生 WebView（默认，与渠道壳 App 一致）；`external` 唤起系统浏览器打开、App 本体退回展示 A 面（既送用户去 B 面，又让 App 看起来仍是干净游戏；外开失败静默降级停在 A 面）。AF/Adjust **A/B 均初始化**；进 B 面发 AF 标准事件 `af_content_view`（内开/外开都发）。

- Flutter：外开用 `url_launcher`（`LaunchMode.externalApplication`），内开仍用 `webview_flutter`；分流在 `lib/gate/gate_screen.dart`，`GateResult.openMode` 由 `gate_service.dart` 解析。
- iOS：外开用 `UIApplication.open`，内开仍用 `WKWebView`（`WebContainerViewController`）；分流在 `DeckTallyPro/Gate/GateCoordinator.swift`，`GateOpenMode` 由 `GateService.swift` 解析。

| App | 位置 | 入口改造 | 验证 |
| --- | --- | --- | --- |
| ColorStack（Flutter） | [listings/colorstack/](../../listings/colorstack/) | `lib/app.dart` → `GateScreen` | `flutter analyze` 通过 + Dart 全量编译通过 |
| DeckTallyPro（iOS） | [listings/decktallypro/](../../listings/decktallypro/) | `AppDelegate` → `GateCoordinator` | `xcodebuild ... BUILD SUCCEEDED` |
| Hexa Color Sort（Flutter） | [listings/hexacolorsort/](../../listings/hexacolorsort/) | `lib/app.dart` → `GateScreen`（A 面回落到原有 `SplashScreen`）| `flutter analyze` 通过 + `flutter test` 31 passed + Android 35 模拟器实跑 4 条路径（A 面 debug/release、B 面内开/外开）|

- Flutter：`lib/gate/`（config/service/screen/web）+ `lib/tracking/`，判定用 `dart:io` HttpClient（无第三方依赖），B 面用 `webview_flutter`，时区用 `flutter_timezone` 取 IANA 名。
- iOS：`DeckTallyPro/Gate/`（工程用 Xcode 16 file-system-synchronized group，新文件自动纳入编译）+ `Core/Services/TrackingService.swift`，判定用 `URLSession`、B 面用 `WKWebView`（均系统框架）。AF/Adjust 用 `#if canImport(...)` 包裹，未加 SPM 包也能编译（no-op），加了自动启用。

各自的运维接入步骤（填 API 基址、AF/Adjust key、iOS 加 SPM 包）见 [colorstack/README_GATE.md](../../listings/colorstack/README_GATE.md)、[decktallypro/README_GATE.md](../../listings/decktallypro/README_GATE.md)、[hexacolorsort/README_GATE.md](../../listings/hexacolorsort/README_GATE.md)。

**接入红线（已落实）**：客户端不存任何 B 面 URL 常量（只存网关 API 基址）；不解析/不判 IP（服务端做）；判定失败/超时/非 B 一律 A 面。

## 6. 推送（已实现，详见 [07-push.md](07-push.md)）

- `push_device_token` / `push_campaign_target` / `push_record` 加 `listing_id`；`push_campaign` 加 `kind`（channel/listing）。migration 000007。
- 客户端拿到 gate 判定后调 `POST /api/app/listing/register-token` 上报 `{platform,bundleId,deviceToken,gateMode}`，服务端记 `last_gate_mode`。
- 上架包推送**强制只推 `last_gate_mode='B'` 的活跃设备**：发送侧 `repo.ActiveListingTokensBMode` 里硬编码该过滤，**无参数可绕过**、无 UI 选项 —— 绝不把 B 面内容推给 A 面（可能是审核员）设备。
- iOS 走 **FCM 中转**（APNs `.p8` 传到 Firebase），Android/iOS 都经独立的 `listings` Firebase 项目（路由键 `fcmRouteKeyListings`）。复用渠道推送的发送内核（`dispatchFCMJobs`）、状态机、失效 token 下线逻辑。
- 未配 `FIREBASE_SA_LISTINGS` 时上架包推送整体 `Skipped`（不算失败），流程仍可跑通，配好私钥即真发。

**管理面 API**：
```
POST /api/push/listing-campaigns            创建（kind=listing，body 含 listingIds）
POST /api/push/listing-campaigns/:id/send   发送（body {dryRun}；dryRun 预览 B 面受众数）
```
**新增 env**：`FIREBASE_SA_LISTINGS`（私钥路径/内容）、`FIREBASE_PROJECT_LISTINGS`（项目 ID）。

## 7. 运维

- GeoIP：DB-IP country-lite 构建期烤进镜像（[Dockerfile.api](../../deploy/Dockerfile.api)），`GEOIP_REFRESH_ENABLE=true` 时 cron 每月 3 号 04:00 自动更新。库缺失/损坏 → 全判 A，不阻断启动。
- 相关 env：`GEOIP_PATH` · `GEOIP_REFRESH_ENABLE` · `TRUSTED_PROXY_CIDRS`（留空用内置私有网段）· `GATE_LOG_ENABLE`。
- 署名：DB-IP CC-BY 4.0，Console 设置页页脚声明「IP 地理数据来自 DB-IP」。
