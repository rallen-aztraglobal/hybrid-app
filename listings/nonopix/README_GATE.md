# NonoPix — AB 面网关接入说明

本工程（Flutter，**仅 Android**，包名 `com.oakmere.nonopix3927`）已接入上架包 AB 面网关。
口径对齐 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) 与
[docs/admin/09-listing.md](../../docs/admin/09-listing.md)。

## 一句话

启动即向服务端请求一次判定：**命中才进 B 面**（服务端下发的 web），否则进 A 面
（游戏本体）。判定期间显示与游戏同底色的加载页，避免白屏。

## 三条不能破的原则

**① 客户端不内置任何 B 面地址。**
`lib/gate/gate_config.dart` 里只有**网关 API 的基址** —— 那是一个返回 A/B 的接口。
B 面 URL 由服务端在判定为 B 时下发。审核方静态扫描 APK 也扫不到线上域名。

**② fail-closed。**
判定进行中、请求超时、网络不通、服务端返回异常、结果非 B —— **一律进 A 面**。
`GateService` 里没有任何一条分支能在「没拿到明确的 B」时进 B。
超时设 6 秒，宁短勿长：启动路径上不该为一个可失败即回退的判定长时间卡着。

**③ 游戏本体对网关零感知。**
`lib/logic` · `lib/screens` · `lib/widgets` · `lib/storage` 里**没有一处** import 到
`lib/gate/`。依赖是单向的：`gate_screen.dart` 会 import 本体的
`screens/game_screen.dart`（判定为 A 时要挂它）和 `theme/app_colors.dart`
（加载页用本体的底色，切过去时背景不跳变），本体不回指。
这条靠 grep 就能查。注意要匹配 **import 行**而不是 `gate/` 三个字 ——
本体里的注释就写着「没有一处 import 到 `lib/gate/`」，按后者搜会命中那句注释、
得到一个假警报：

```bash
grep -rn "^import .*gate/" lib/logic lib/screens lib/widgets lib/storage \
  && echo '违规！' || echo 'ok'
```

## 判定是怎么做的

请求 `POST /api/app/listing/gate`，体里只有三样：

```json
{ "platform": "android", "bundleId": "com.oakmere.nonopix3927", "timezone": "Asia/Manila" }
```

国家由**服务端从请求的真实 IP** 判定，不由客户端上报 —— 客户端报的国家可以被改。
时区由客户端给（`flutter_timezone`），和 IP 国家一起作为判定输入。

响应只有三个字段：`mode`（`"A"` / `"B"`）、`url`、`openMode`。

- `openMode = internal` → 用应用内 WebView 打开（`lib/gate/web_screen.dart`）
- `openMode = external` → 唤起系统浏览器打开，App 本体仍展示 A 面。
  浏览器打不开时静默降级，仍停在 A 面。

**CN / US 硬编码强制 A 面**，这一条在服务端。

## 服务端要建的条目

Console → 上架包，新建一条：platform=`android`、bundleId=`com.oakmere.nonopix3927`，
配好 palCode 与品牌。

**本包挂 `ap`**（ArenaPlus 浅色站）。六个新包按 2/2/2 分到 ap / bp / gp。

**打开方式：外开（系统浏览器）** —— B 面不在 App 内渲染，因此 `bSideChromeColor`
（本包 `0xFFFFFFFF`）目前用不上。它仍按品牌取值，是因为打开方式由服务端下发，
运营在 Console 里改成内开就立刻生效，客户端不重新打包。

外开还有一个连带结果：**`OpenBLanding` 这个 Adjust 事件只有外开才会触发**
（`gate_screen.dart`，且要浏览器确实唤起成功才发）。改成内开的话它永远不发。

## 编译期烧录的配置

全在 `lib/gate/gate_config.dart`：

| 字段 | 现状 |
| --- | --- |
| `bundleId` | ✅ `com.oakmere.nonopix3927` |
| `apiBases` | ✅ 与其余上架包同一个渠道中台基址，可再加候选抗封 |
| `appsFlyerDevKey` | ✅ 账号级 key，与其余包共用（AF 按账号发，不按 App 发） |
| `adjustAppToken` | ✅ **`fpny286clibk`** —— 若改回 `TODO_` 前缀，`TrackingService` 会跳过 Adjust 初始化，全链路 no-op（不崩，只是不上报） |
| `adjustOpenBLandingToken` | ✅ `ur46ap`（Adjust 后台 event `OpenBLanding`）—— 与上面成对生效，缺一个就整条 no-op |
| `bSideChromeColor` | ✅ `0xFFFFFFFF`（浅色站）。**若改挂 gp 深色站，要改成 `0xFF1C1D27` 并重新打包** |

Adjust 后台已给本包**单独建了 App**（名 `NonoPix`，Android 平台 `com.oakmere.nonopix3927`，
store google play，reporting currency PHP，勾了「用户群仅在欧洲经济区之外」）——
逐项与 ColorStack 一致。token **不与任何其他包共用**：复用会把本包的安装与会话
归到别的 App 上。

## Firebase

`android/app/google-services.json` **已放入** ✅。本包已在 Firebase 项目
**`hybrid-listings-51660`**（project_number `609439342540`）下以包名 `com.oakmere.nonopix3927` 注册
（`mobilesdk_app_id` = `1:609439342540:android:e673caa6f5a2e2d5a98769`），**不加 SHA 证书指纹** —— 与 ColorStack 一致。
该文件随 APK 分发、反编译即可得，非机密，进 git。

**已按规矩裁剪成只含本包一个 client。** 控制台下载的原始文件会把该项目下**所有**
Android App 都列进来（本次下载里有 5 条），理由见 `SECURITY_NOTES.md`。
google-services 插件只按 applicationId 取匹配的那条，多余条目会被忽略。

> `android/app/build.gradle.kts` 里这个插件是**按文件存在与否条件应用**的 ——
> 文件在了插件才真正生效，所以加/删这个文件会改变构建路径，改动后要重新构建一次。
> 放入前跑的是降级路径：`Firebase.initializeApp()` 失败被吞掉 → 推送 no-op，
> 不影响网关与本体。

## 启动路径上的一个坑

`main()` 里触发 Firebase 初始化但**不 await**：

```dart
PushService.instance.initFirebase();   // 故意不 await
runApp(const NonoPixApp());
```

启动路径上不放任何可能卡住的原生调用 —— `runApp` 之前一挂就是纯黑屏、App 完全打不开。
需要 token 时 `PushService` 内部会带超时地等它收尾。

## 怎么验

### 判定回退（最重要的一条）

服务端不可达时必须进 A 面，而不是白屏或崩溃：

```bash
adb shell svc wifi disable && adb shell svc data disable
adb shell am force-stop com.oakmere.nonopix3927
adb shell monkey -p com.oakmere.nonopix3927 -c android.intent.category.LAUNCHER 1
# 等约 6 秒（超时）后应当进入游戏界面
adb shell svc wifi enable
```

### 包里没有 B 面地址

```bash
unzip -p build/app/outputs/flutter-apk/app-release.apk \
  assets/flutter_assets/kernel_blob.bin 2>/dev/null | strings \
  | grep -Ei 'https?://' | sort -u
# 只应看到网关 API 基址与各 SDK 自己的域名，不应有任何落地页地址
```

### 本体与网关的隔离

见上面那条 grep。

## 界面测试要注意

给游戏本体写 widget 测试时，直接挂 `GameScreen` 而**不是** `NonoPixApp` ——
后者的 `home` 是启动闸，会真的发网络请求，测试会卡满超时再回退。
