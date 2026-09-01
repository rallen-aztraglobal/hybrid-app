# CalcPad — AB 面网关接入说明

本工程（Flutter，**仅 Android**，包名 `com.northglade.calcpad5170`）已接入上架包 AB 面网关
（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。
启动即向渠道中台判定 A/B：A 面进计算器本体（`CalculatorScreen`，原代码零改动），
B 面按 `openMode` 内开 `WebScreen` 或外开系统浏览器。判定失败一律 A 面。

接入方式与 [colorstack](../colorstack/README_GATE.md) / [hexacolorsort](../hexacolorsort/README_GATE.md)
基本同构，A 面入口不同（本包是 `CalculatorScreen`，hexa 是 `SplashScreen`，colorstack 是 `HomeScreen`），
另有一处刻意的修正见下。

## 与 hexacolorsort 的唯一差异：亮度判定去掉了一个 `/255`

`web_screen.dart` 里决定系统栏图标明暗的那段，原先写的是：

```dart
final l = (0.299 * _chrome.r + 0.587 * _chrome.g + 0.114 * _chrome.b) / 255.0;
return l > 0.6;
```

但 `Color.r/g/b` 在 **Flutter 3.27 之后已经是 0..1 的 double**（不再是 0..255 的 int），
加权和本身就落在 0..1，再除 255 会把结果压到千分位 —— **任何颜色都恒判"深色"**，
于是永远配浅色图标。

当前填充色是 `#1C1D27`（`gp` 深色站），两种算法都得"深色"，所以这个错误在本包与
hexacolorsort 上都表现不出来。但只要本包改挂 `ap`/`bp` 这类浅色站、
`GateConfig.bSideChromeColor` 换成白色，除 255 的版本就会继续给白底配白图标，
**状态栏图标直接隐形**。

现在是：

```dart
static bool get _chromeIsLight =>
    0.299 * _chrome.r + 0.587 * _chrome.g + 0.114 * _chrome.b > 0.6;
```

行为在当前配色下**完全不变**（`#1C1D27` 的加权亮度 ≈ 0.117，两种写法都 < 0.6）。
tilefit 上的同一处也是这么写的。**hexacolorsort 与 colorstack 尚未同步这个修正**
（colorstack 的 `web_screen.dart` 里没有这段亮度判定逻辑，故不受影响；
hexacolorsort 在另一个分支上，需要单独改）。

## 新增文件
- `lib/gate/gate_config.dart` — 编译期配置（**Adjust token 待填**，见下）
- `lib/gate/gate_service.dart` — 判定请求（`dart:io` HttpClient，无第三方依赖）
- `lib/gate/gate_screen.dart` — 启动闸：判定 → 决定 A/B（App 入口）
- `lib/gate/web_screen.dart` — B 面 WebView（webview_flutter）
- `lib/push/push_service.dart` — FCM token 注册（带 gateMode）
- `lib/tracking/tracking_service.dart` — AF/Adjust 上报
- `test/gate/gate_result_test.dart` — 锁住「只有 mode=B 且 url 非空才算 B 面」等安全不变量

## 改动的既有文件
- `lib/main.dart` — `home:` 由 `CalculatorScreen` 改为 `GateScreen`；启动时多一步
  `PushService.instance.initFirebase()`（不 await，失败静默降级）
- `pubspec.yaml` — 新增依赖 `webview_flutter` · `flutter_timezone` · `url_launcher` ·
  `firebase_core` · `firebase_messaging` · `appsflyer_sdk` · `adjust_sdk`；
  dev 侧加 `flutter_launcher_icons`
- `test/widget_test.dart` — 原来 4 处 `pumpWidget(const CalculatorApp())` 会连带跑起网关、
  在测试环境里发真实网络请求。改为直接挂 `CalculatorScreen`（新增 `_wrap()` 辅助），
  测的还是同一棵计算器 widget 树，只是绕开启动闸
- `android/app/src/main/AndroidManifest.xml` — 补 `INTERNET` 权限（原仅 debug/profile 有，
  release 缺）；补 `<queries>` 里的 https VIEW 意图（Android 11+ 外开需要）；
  `android:label` 改为 `CalcPad`
- `android/app/build.gradle.kts` — 条件应用 google-services 插件；release 走
  `key.properties` 正式签名；**显式声明 `androidx.appcompat:appcompat:1.7.1`**（原因见下）
- `android/settings.gradle.kts` — 声明 `com.google.gms.google-services` 插件版本

计算器本体（`lib/logic/` · `lib/models/` · `lib/screens/` · `lib/theme/` · `lib/widgets/`）
**一行未改**。

### 为什么要显式写 appcompat
`appsflyer_sdk` 依赖 `androidx.appcompat:appcompat:1.0.0`，它带进来的
`vectordrawable:1.0.0` 与 `vectordrawable-animated:1.0.0` 在各自的 AndroidManifest 里
声明了**同一个** namespace（`androidx.vectordrawable`）。AGP 9 的 manifest merger 校验
namespace 唯一，于是 `:app:processReleaseMainManifest` 直接失败：

```
Namespace 'androidx.vectordrawable' is used in multiple modules and/or libraries:
androidx.vectordrawable:vectordrawable-animated:1.0.0, androidx.vectordrawable:vectordrawable:1.0.0
```

hexacolorsort 没踩到这个坑，纯属巧合：它多一个 `shared_preferences` 依赖，
其 Android 侧带 `androidx.preference:1.2.1`，顺手把 appcompat 抬到了 1.1.0，
而 1.1.0 之后 vectordrawable 拆分了 namespace。本包没有 `shared_preferences`，
appcompat 就停在 1.0.0 上。所以这里不依赖那种巧合，直接在 `android/app/build.gradle.kts`
里写死 `implementation("androidx.appcompat:appcompat:1.7.1")`。

## 上线前必做

### 1. 后台建 listing 条目 —— 待做
Console → 上架包，新建一条：platform=`android`、bundleId=`com.northglade.calcpad5170`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

条目建好之前，服务端对本包的 bundleId 查不到 listing，判定恒为 A 面 —— 这是 fail-closed
的预期行为，不是故障。

### 2. 填 `lib/gate/gate_config.dart`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。
  **这是网关 API，不是 B 面地址**；基址迁移时改这里，可加多个候选抗封。
- `appsFlyerDevKey`：已填 `fXoKsKQwxPCRdhD8CD8q6F`（账号级 key，与 colorstack /
  decktallypro / hexacolorsort 同一 AF 账号）。`appsFlyerAppId` 在 Android 侧即包名，无需改。
- `adjustAppToken`：**待填，现在是 `TODO_ADJUST_APP_TOKEN`**。需要在 Adjust 后台为本包
  新建一个 App（Platform 选 Android、Store 选 Google Play、bundleId 填
  `com.northglade.calcpad5170`、reporting currency 与现有 app 对齐用 **PHP**），
  拿到 12 位 App Token 填进来。
  **不可复用其他包的 token**（colorstack `bytg13h7yubk` / decktallypro `sn947o53ym80` /
  hexacolorsort `2yhxl7paa3ls`），复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：**待填，现在是空串**。在上面那个新 App 下建一个名为
  `OpenBLanding` 的 event（非 unique event，每次外开成功都要计一次），把 6 位 event token
  填进来。留空时 `TrackingService.onOpenBLanding()` 只发 AppsFlyer、跳过 Adjust —— 不崩，
  但 Adjust 侧看不到外开事件。
- `adjustContentViewToken`：保持空串。与 colorstack 一致，本包不发这个事件。

### 3. FCM（推送）—— 待做
`android/app/google-services.json` **还没放**。需要在 Firebase 项目
**`hybrid-listings-51660`**（project_number `609439342540`）下以包名
`com.northglade.calcpad5170` 注册一个 Android App，下载 `google-services.json` 放到
`android/app/` 下。该文件非机密（随 APK 分发），进 git，与其余上架包同口径。

**放进来之前记得裁剪 —— 只保留本包一个 client 条目。**
Firebase 控制台下载的原始文件会把该项目下**所有** Android App 都列进来，包括
`com.vividnest.colorstack5821` 和 `com.slatecove.hexasort4173`。带着它出包，等于在
CalcPad 的 APK 里明写另外两个上架包的包名，任何人解压 APK 就能看出这几个包同属一家 ——
这正是各包使用互不相关的厂商命名空间（`vividnest` / `deck` / `slatecove` / `northglade`）
想避免的事。google-services 插件只按 applicationId 取匹配的那条 client，多余条目会被忽略。

缺这个文件时的降级路径已在 hexacolorsort 上实测过：日志出现
`Default FirebaseApp failed to initialize because no default options were found`
之后 App 照常进 A 面、不崩 —— 即推送 no-op、不影响网关与计算器。本包当前就跑在这条
降级路径上（`android/app/build.gradle.kts` 里 google-services 插件是条件应用的）。

### 4. 签名 —— 待做
本包目前**没有** release keystore。`android/key.properties`（照 `key.properties.example` 填）
+ keystore 放到 `android/` 下。两者都在 `.gitignore` 里，**不进 git**。
没有 key.properties 时 release 退回 debug 签名，仅供本地跑通 —— **Play 拒收 debug 签名的包**。
生成步骤见 `RELEASE_SIGNING.md`。

### 5. 应用图标 —— 已就位 ✅
原先是 Flutter 模板默认图标。现已换成自绘的 2×2 运算键，沿用 `AppColors` 的原色
（底 `#121212`、数字键 `#333333`、运算键 `#FF9F0A`）：

- `icon.png` — 1024×1024，legacy 图标源（`mipmap-*/ic_launcher.png`）
- `icon_foreground.png` / `icon_background.png` — 自适应图标两层（Android 8+）
- 由 `flutter_launcher_icons` 生成，配置在 pubspec.yaml
- 生成脚本的思路：四个圆角键位画 `+ − × =`，符号是纯几何图形拼的（不依赖字体渲染，
  各机型/各 DPI 下都一致），右下 `=` 用运算键橙色作为视觉重点

**为什么做了自适应图标**（colorstack 只有 legacy）：只给 legacy 图标时，Pixel 启动器会把它
缩小塞进一个自己生成的浅色圆底里，深色图标外面套一圈不搭的亮环。给了前景/背景两层后，
启动器用我们自己的 `#121212` 底铺满整个形状。`flutter_launcher_icons` 生成的
`mipmap-anydpi-v26/ic_launcher.xml` 会再给前景套一层 16% inset，所以前景源图的留白
按这层 inset 反算过（14.7%），使按键在圆形与方圆形两种遮罩下都填得满、四角又不被切到。

## 本地验证
```bash
flutter pub get
flutter analyze               # 应 No issues found
flutter test                  # 23 passed（20 原有 + 3 网关）
flutter build apk --release   # 47.7MB，本地验证用
flutter build appbundle --release   # 46.5MB，上架传这个
```

已在 Flutter 3.44.4 / Dart 3.12.2 + AGP 9.0.1 下跑通，并在 Android 35 模拟器
（Pixel 6 / x86_64，**必须 `-gpu host`**）上实跑验过：

| 路径 | 包类型 | 结果 |
| --- | --- | --- |
| A 面 | **release（R8）** | 判定 → 加载页 → 计算器；`1234 × 56 = 69104`、`45.6 ÷ 8 = 5.7` 均正确，无 ClassNotFound / NoSuchMethod / NoClassDefFound —— 归因与 Firebase 的反射未被 R8 剪坏 |
| 缺 google-services.json 的降级 | release | Firebase 初始化失败后照常进 A 面、不崩（本包当前状态） |
| AOT 产物无 B 面域名 | release AAB | 反解 `base/lib/arm64-v8a/libapp.so` 的字符串，出现的域名只有网关 API `api.fortunegems-jackpot.online` 以及 Flutter/Firebase 自带的文档与插件通道域名（`docs.flutter.dev` · `pub.dev` · `firebase.google.com` · `plugins.flutter.io` 等）—— 无任何品牌站点域名，也无 `example.com` / 临时调试残留 |
| AF devKey 已编进产物 | release AAB | `fXoKsKQwxPCRdhD8CD8q6F` 与 `com.northglade.calcpad5170` 在 snapshot 里各命中一次；Adjust token 位置目前是 `TODO_ADJUST_APP_TOKEN`（占位符如实编进去了，说明填上真 token 后同样会生效），且**没有**其他包的 token 混进来 |
| 权限清单 | release APK | `dumpsys package` 的 requested permissions 与 hexacolorsort 逐条相同，见 `SUBMISSION_NOTES.md` |

`store/` 下的四张商店截图就是这次 release 实跑截的（1080×2400 原图左右补边到 1200×2400，
理由见 `STORE_ASSETS.md`）。

> 模拟器注意：用 `-gpu swiftshader_indirect`（软件 GL）跑 release 会出现单帧 500 秒以上、
> 画面看起来是白屏。那是模拟器 GPU 问题，不是 App 卡死（`app_time_stats` 能看到帧在出）。
> 一定要用 `-gpu host`。

## 仍未验证
- **B 面两条路径（内开 / 外开）** —— 服务端尚未建本包 listing 条目，正常判定恒为 A。
  `lib/gate/web_screen.dart` 与 hexacolorsort 上已实测通过的那份**仅差亮度判定一处**
  （见下面「与 hexacolorsort 的唯一差异」），其余（edge-to-edge、chrome 底色填充、
  亮度自适应图标色、错误时停 loading）逐字节相同，但本包自身没跑过 B 面。
- 服务端真实判 B（需先在 Console 建 listing 条目）。
- 推送：`google-services.json` 未放，客户端半程都没跑起来。
- Adjust 上报：App Token 与 event token 都还是占位符。
- 真机（只在 Android 35 模拟器上跑过）。

## 接入红线（已落实）
- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端 `repo.ActiveListingTokensBMode` 硬编码保证，
  客户端无从绕过。
