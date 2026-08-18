# Hexa Color Sort — AB 面网关接入说明

本工程（Flutter，**仅 Android**，包名 `com.hexacolorsort.hexa_color_sort`）已接入上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：A 面进游戏本体（`SplashScreen` → `HomeScreen`，原代码零改动），B 面按 `openMode` 内开 `WebScreen` 或外开系统浏览器。判定失败一律 A 面。

接入方式与 [colorstack](../colorstack/README_GATE.md) 完全同构，仅 A 面入口不同（本包的游戏入口是 `SplashScreen`，colorstack 是 `HomeScreen`）。

## 新增文件
- `lib/gate/gate_config.dart` — 编译期配置（**需运维填写 Adjust token**）
- `lib/gate/gate_service.dart` — 判定请求（`dart:io` HttpClient，无第三方依赖）
- `lib/gate/gate_screen.dart` — 启动闸：判定 → 决定 A/B（App 入口）
- `lib/gate/web_screen.dart` — B 面 WebView（webview_flutter）
- `lib/push/push_service.dart` — FCM token 注册（带 gateMode）
- `lib/tracking/tracking_service.dart` — AF/Adjust 上报
- `test/gate/gate_result_test.dart` — 锁住「只有 mode=B 且 url 非空才算 B 面」等安全不变量

## 改动的既有文件
- `lib/app.dart` — `home:` 由 `SplashScreen` 改为 `GateScreen`
- `lib/main.dart` — 启动时多一步 `PushService.instance.initFirebase()`（失败静默降级）
- `pubspec.yaml` — 新增依赖 `webview_flutter` · `flutter_timezone` · `url_launcher` · `firebase_core` · `firebase_messaging` · `appsflyer_sdk` · `adjust_sdk`
- `android/app/src/main/AndroidManifest.xml` — 补 `INTERNET` 权限（原仅 debug/profile 有，release 缺）；补 `<queries>` 里的 https VIEW 意图（Android 11+ 外开需要）
- `android/app/build.gradle.kts` — 条件应用 google-services 插件；release 走 `key.properties` 正式签名
- `android/settings.gradle.kts` — 声明 `com.google.gms.google-services` 插件版本

游戏本体（`lib/core/` · `lib/game/` · `lib/screens/` · `lib/widgets/`）**一行未改**。

## 上线前必做

### 1. 后台建 listing 条目
Console → 上架包，新建一条：platform=`android`、bundleId=`com.hexacolorsort.hexa_color_sort`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

### 2. 填 `lib/gate/gate_config.dart`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。**这是网关 API，不是 B 面地址**；基址迁移时改这里，可加多个候选抗封。
- `appsFlyerDevKey`：已填（账号级 key，与 colorstack / decktallypro 同一 AF 账号）。`appsFlyerAppId` 在 Android 侧即包名，无需改。
- `adjustAppToken`：**当前是占位 `TODO_ADJUST_APP_TOKEN`，Adjust 全链路 no-op**。需运营在 Adjust 后台为本 App 建条目后填入 —— 不可复用 colorstack(`bytg13h7yubk`) / decktallypro(`sn947o53ym80`) 的 token，复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：Adjust 后台建 event `OpenBLanding` 后填入（留空则外开只发 AF 事件）。

### 3. FCM（推送）
本包的 `google-services.json` **尚未就位**。需在 Firebase 项目 **`hybrid-listings-51660`** 里
以包名 `com.hexacolorsort.hexa_color_sort` 注册 Android App，下载 `google-services.json` 放到
`android/app/google-services.json`（该文件非机密，随包分发，可进 git，与 colorstack 同口径）。

在此之前构建不会失败：`android/app/build.gradle.kts` 只在该文件存在时才 apply google-services 插件，
缺文件时 `Firebase.initializeApp()` 失败被 try/catch 吞掉 → 推送 no-op，**不影响网关与 App 本体**。

> 首次带 google-services.json 出包时留意 AGP 9.0.1 与 google-services 4.4.2 的兼容（本仓库其余
> Flutter 上架包用的是 AGP 8.11.1）。若 Gradle 报插件不兼容，把 `settings.gradle.kts` 里的
> google-services 版本抬到最新即可。

### 4. 签名
`android/key.properties`（照 `key.properties.example` 填）+ keystore 放到 `android/` 下。
两者都在 `.gitignore` 里，**不进 git**。没有 key.properties 时 release 退回 debug 签名，仅供本地跑通。

## 本地验证
```bash
flutter pub get
flutter analyze          # 应 No issues found
flutter test             # 31 passed（28 原有 + 3 网关）
flutter build bundle     # 全量 Dart 编译（不经 Gradle）
```

上述四步已在 Flutter 3.44.4 / Dart 3.12.2 下全部跑通。

## 接入红线（已落实）
- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端 `repo.ActiveListingTokensBMode` 硬编码保证，客户端无从绕过。
