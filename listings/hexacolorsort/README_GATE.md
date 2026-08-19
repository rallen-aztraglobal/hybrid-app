# Hexa Color Sort — AB 面网关接入说明

本工程（Flutter，**仅 Android**，包名 `com.slatecove.hexasort4173`）已接入上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：A 面进游戏本体（`SplashScreen` → `HomeScreen`，原代码零改动），B 面按 `openMode` 内开 `WebScreen` 或外开系统浏览器。判定失败一律 A 面。

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
Console → 上架包，新建一条：platform=`android`、bundleId=`com.slatecove.hexasort4173`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

### 2. 填 `lib/gate/gate_config.dart`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。**这是网关 API，不是 B 面地址**；基址迁移时改这里，可加多个候选抗封。
- `appsFlyerDevKey`：已填（账号级 key，与 colorstack / decktallypro 同一 AF 账号）。`appsFlyerAppId` 在 Android 侧即包名，无需改。
- `adjustAppToken`：**当前是占位 `TODO_ADJUST_APP_TOKEN`，Adjust 全链路 no-op**。需运营在 Adjust 后台为本 App 建条目后填入 —— 不可复用 colorstack(`bytg13h7yubk`) / decktallypro(`sn947o53ym80`) 的 token，复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：Adjust 后台建 event `OpenBLanding` 后填入（留空则外开只发 AF 事件）。

### 3. FCM（推送）—— 已就位 ✅
`android/app/google-services.json` 已放入：Firebase 项目 **`hybrid-listings-51660`**
（project_number `609439342540`）下以包名 `com.slatecove.hexasort4173` 注册的 Android App，
`mobilesdk_app_id` = `1:609439342540:android:c211607d0498572ea98769`。
该文件非机密（随 APK 分发），进 git，与 colorstack 同口径。

**注意：仓库里这份是手工裁剪过的 —— 只保留本包一个 client 条目。**
Firebase 控制台下载的原始文件会把该项目下**所有** Android App 都列进来，包括
`com.vividnest.colorstack5821`。带着它出包，等于在 Hexa 的 APK 里明写 ColorStack 的包名，
任何人解压 APK 就能看出两个上架包同属一家 —— 这正是三个包各用不相关厂商命名空间
（`vividnest` / `deck` / `slatecove`）想避免的事。google-services 插件只按 applicationId
取匹配的那条 client，多余条目会被忽略，故裁掉不影响功能（已实测：构建通过、Firebase 初始化成功）。

> 以后重新下载这个文件，记得再裁一次，别把 colorstack 的条目带回来。

> 关联性的实话：裁剪只去掉了「明写对方包名」。两个包用的是同一个 Firebase 项目，
> 因此 `project_number` 与 `api_key` 仍然相同 —— 有人同时解压两个 APK 逐字段比对，
> 依然能看出同属一个项目。要彻底切断就得每个上架包一个独立 Firebase 项目，
> 但服务端目前是单一路由键（`fcmRouteKeyListings` 是常量、一个 service account 管整个项目），
> 那是架构级改动，不在本包范围内。

> AGP 兼容已实测：本包 AGP 9.0.1 + google-services 4.4.2 构建通过
> （其余 Flutter 上架包用的是 AGP 8.11.1）。

### 4. 签名
`android/key.properties`（照 `key.properties.example` 填）+ keystore 放到 `android/` 下。
两者都在 `.gitignore` 里，**不进 git**。没有 key.properties 时 release 退回 debug 签名，仅供本地跑通。

### 5. 应用图标（尚未换）
`android/app/src/main/res/mipmap-*/ic_launcher.png` 目前与 Flutter 模板默认图标 **md5 完全一致**
（即蓝色 Flutter logo），本包也没有配 `flutter_launcher_icons`。上架包用默认图标既不像成品、
也是审核风险。照 colorstack 的做法办：根目录放一张方形图（它叫 `icones.png`），
pubspec 加 `flutter_launcher_icons` 配置后跑 `dart run flutter_launcher_icons`。

## 本地验证
```bash
flutter pub get
flutter analyze          # 应 No issues found
flutter test             # 31 passed（28 原有 + 3 网关）
flutter build bundle     # 全量 Dart 编译（不经 Gradle）
flutter build apk --release   # 走 R8 混淆，上架用这条
```

已在 Flutter 3.44.4 / Dart 3.12.2 + AGP 9.0.1 下跑通，并在 Android 35 模拟器
（Pixel 6 / x86_64，**必须 `-gpu host`**）上实跑验过五条路径：

| 路径 | 包类型 | 结果 |
| --- | --- | --- |
| A 面 | debug | 判定 → 加载页 → Splash → 首页；点 Play 进游戏、移动方块生效 |
| A 面 | **release（R8）** | 同上，且无 ClassNotFound / NoSuchMethod / NoClassDefFound —— 归因与 Firebase 的反射未被 R8 剪坏 |
| B 面内开 | debug | 全屏 WebView 加载成功、白底状态栏 + 深色图标、无 App 外壳 |
| B 面外开 | debug | 系统 Chrome 被唤起（证明 manifest 的 https VIEW query 生效），**App 本体退回 A 面显示游戏** |
| 推送客户端半程 | debug | Firebase 初始化成功、Installations 注册状态 REGISTERED、FCM token 已签发（sender 609439342540 与项目一致）→ 代码随即带 gateMode 发 register-token |

B 面两条都要临时把判定结果写死才能验（服务端尚未建本包 listing 条目，正常判定恒为 A）；
验完均已还原，`lib/` 已扫过无 TEMP / example.com 残留，工作区与 HEAD 一致。

**缺 google-services.json 时的降级路径也实测过**（在该文件放入之前的那一版 release 包）：
日志出现 `Default FirebaseApp failed to initialize because no default options were found` 之后
App 照常进 A 面、不崩 —— 即推送 no-op、不影响网关与游戏。文件现已就位，此路径仅作为兜底记录。

> 模拟器注意：用 `-gpu swiftshader_indirect`（软件 GL）跑 release 会出现单帧 500 秒以上、
> 画面看起来是白屏。那是模拟器 GPU 问题，不是 App 卡死（`app_time_stats` 能看到帧在出）。
> 一定要用 `-gpu host`。

## 仍未验证
- 服务端真实判 B（需先在 Console 建 listing 条目）。
- 推送**服务端侧**：客户端半程已验通（见上表），但服务端是否收下并记入
  `push_device_token.last_gate_mode`、以及 Console 发推送能否到达设备，都需要 Console
  建好 listing 条目后才能验。
- Adjust 上报（token 仍是占位 `TODO_ADJUST_APP_TOKEN`，目前 no-op）。
- 真机（只在 Android 35 模拟器上跑过）。

## 接入红线（已落实）
- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端 `repo.ActiveListingTokensBMode` 硬编码保证，客户端无从绕过。
