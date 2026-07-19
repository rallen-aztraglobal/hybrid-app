# DeckTallyPro — AB 面网关接入说明

本工程已接入上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：A 面进游戏本体（`MainTabBarController`，原代码零改动），B 面进 `WebContainerViewController` 加载服务端下发的 web。判定失败一律 A 面。

## 新增文件（均在 file-system-synchronized group 下，自动纳入编译）
- `DeckTallyPro/Gate/GateConfig.swift` — 编译期配置（**需运维填写**）
- `DeckTallyPro/Gate/GateService.swift` — URLSession 判定请求（系统框架，无第三方依赖）
- `DeckTallyPro/Gate/GateCoordinator.swift` — 启动编排：判定 → 切根控制器
- `DeckTallyPro/Gate/LaunchGateViewController.swift` — 判定期加载页
- `DeckTallyPro/Gate/WebContainerViewController.swift` — B 面 WKWebView 容器
- `DeckTallyPro/Core/Services/TrackingService.swift` — AF/Adjust 上报
- `DeckTallyPro/App/AppDelegate.swift` — 改为经 `GateCoordinator.start(in:)` 启动

## 上线前必做

### 1. 填 `GateConfig.swift`
- `apiBases`：**已预置为与现有 APK 相同的基址** `https://api.fortunegems-jackpot.online`。基址迁移时改这里，可加多个候选抗封。**这是网关 API，不是 B 面地址**。
- `appsFlyerDevKey` / `appsFlyerAppleAppID`、`adjustAppToken`：AF/Adjust 的 key（占位 `TODO_*` 时对应 SDK 不启用）。

### 2. 加 AppsFlyer / Adjust 的 SPM 包（一次性）
`TrackingService.swift` 用 `#if canImport(...)` 包裹 SDK 调用：**未加包也能编译（走 no-op）**，加了自动启用。
Xcode → File → Add Package Dependencies：
- AppsFlyer：`https://github.com/AppsFlyerSDK/AppsFlyerFramework-Strict`（产品 `AppsFlyerLib`）
- Adjust：`https://github.com/adjust/ios_sdk`（产品 `Adjust`）

### 3. Info.plist 补充（AF/Adjust 归因需要）
- `NSUserTrackingUsageDescription`（ATT 弹窗文案）
- `NSAdvertisingAttributionReportEndpoint`（如用 SKAdNetwork，按 AF 文档）

### 4. 推送（FCM，已接入代码）
`DeckTallyPro/Push/PushService.swift` 已写好：配置 Firebase、注册 APNs、取 FCM token 并带 `gateMode`
调 `POST /api/app/listing/register-token`。服务端**只向 gateMode=B 的设备**发上架包推送。
`GoogleService-Info.plist` 已放在 `DeckTallyPro/`（sync group 自动纳入 bundle）。Firebase 项目 `hybrid-listings`。

Firebase SDK 同样用 `#if canImport(...)` 包裹，**未加包也能编译**（推送 no-op）。**一次性做**：
1. File → Add Package Dependencies：`https://github.com/firebase/firebase-ios-sdk`，选产品 **FirebaseMessaging**。
2. Signing & Capabilities：加 **Push Notifications** + **Background Modes → Remote notifications**。
3. APNs Auth Key（.p8）传到 Firebase 项目 `hybrid-listings` 的 Cloud Messaging（一次覆盖两个 iOS App）。

## 本地验证
```bash
xcodebuild -scheme DeckTallyPro -project DeckTallyPro.xcodeproj \
  -sdk iphonesimulator -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
```
