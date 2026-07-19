# ColorStack — AB 面网关接入说明

本工程（Flutter，Android + iOS 同包名 `com.vividnest.colorstack5821`）已接入上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：A 面进游戏本体（`HomeScreen`，原代码零改动），B 面进 `WebScreen` 加载服务端下发的 web。判定失败一律 A 面。

## 新增文件
- `lib/gate/gate_config.dart` — 编译期配置（**需运维填写**）
- `lib/gate/gate_service.dart` — 判定请求（`dart:io` HttpClient，无第三方依赖）
- `lib/gate/gate_screen.dart` — 启动闸：判定 → 决定 A/B（App 入口）
- `lib/gate/web_screen.dart` — B 面 WebView（webview_flutter）
- `lib/tracking/tracking_service.dart` — AF/Adjust 上报
- `lib/app.dart` — 入口改为 `GateScreen`

## 新增依赖（已写入 pubspec.yaml）
`webview_flutter` · `flutter_timezone` · `appsflyer_sdk` · `adjust_sdk`

Android 已在主 `AndroidManifest.xml` 补 `INTERNET` 权限（原仅 debug/profile 有，release 缺）。

## 上线前必做
填 `lib/gate/gate_config.dart`：
- `apiBases`：**已预置为与现有 APK 相同的基址** `https://api.fortunegems-jackpot.online`（APK bootstrap.json 的 configUrl 同源）。基址迁移时改这里，可加多个候选抗封。**这是网关 API，不是 B 面地址**。
- `appsFlyerDevKey` / `appsFlyerAppId`、`adjustAppToken`（两端不同的 Adjust app，token 各异）。占位 `TODO_*` 时对应 SDK 不启用（no-op）。
- iOS 还需在 `ios/Runner/Info.plist` 补 `NSUserTrackingUsageDescription`（AF/Adjust 的 ATT）。

## 推送（FCM，已接入）
已加 `firebase_core` + `firebase_messaging`；`lib/push/push_service.dart` 在启动初始化 Firebase、
判定后取 FCM token 并带 `gateMode` 调 `POST /api/app/listing/register-token`。服务端**只向
gateMode=B 的设备**发上架包推送。Firebase 项目 `hybrid-listings`。

已就位：`android/app/google-services.json`、`ios/Runner/GoogleService-Info.plist`、
Android 的 `com.google.gms.google-services` 插件（settings + app gradle）。

**iOS 还需在 Xcode 一次性做**（native，无法脚本化）：
1. 把 `ios/Runner/GoogleService-Info.plist` 拖进 Runner target（勾选 Copy if needed + Runner 成员）。
2. Runner → Signing & Capabilities：加 **Push Notifications**；加 **Background Modes → Remote notifications**。
3. APNs Auth Key（.p8）传到 Firebase 项目 `hybrid-listings` 的 Cloud Messaging（一次覆盖两个 iOS App）。

> 未做上述 iOS 步骤时，`Firebase.initializeApp()` 会失败并被 try/catch 吞掉 → 推送 no-op，不影响 App 与网关。

## 本地验证
```bash
fvm use 3.44.6   # 项目要求 Dart ^3.10.8
flutter pub get
flutter analyze lib/          # 应 No issues found
flutter build bundle          # 全量 Dart 编译（不经 Gradle）
```
