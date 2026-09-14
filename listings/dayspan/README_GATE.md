# AB 面网关 —— DaySpan

本包接的是 ADR-0014 的上架包网关。**工具本体对它零感知** ——
`lib/logic/` 下没有一行代码知道 B 面的存在。

## 判定怎么走

```
启动 → GateScreen 向网关 API 发 { platform, bundleId, timezone }
     → 服务端按**请求真实 IP 的国家** + 时区判定
     → 回 { Mode: "A"|"B", URL, OpenMode }
     → A：展示工具本体（HomeScreen）
       B：按 OpenMode 内开（App 内 WebView）或外开（系统浏览器）
```

三条硬性质，缺一条这套东西就不该上架：

1. **客户端零内置 B 面地址。** 包里只有网关 API 的基址 —— 它只返回 A/B，
   泄露它不等于泄露 B 面。审核方静态扫描包也扫不到落地页域名。
2. **fail-closed。** 判定超时、网络失败、返回非法、JSON 解析失败 —— 一律进 A 面。
   宁可少放行，不可误放行。
3. **CN/US 硬编码强制 A 面。** 服务端 `model.ForcedACountries`，无视任何配置。
   这是最后一道闸，不做成可配置项。

## 服务端要建的条目

Console → 上架包，新建一条：platform=`android`、bundleId=`com.haymoor.dayspan3306`，
palCode 填 `1146761515996676096`、品牌选 `bp`。

**本包挂 `bp`**（BingoPlus 浅色站），PAL_CODE `1146761515996676096`（取自渠道 `bpom3407`）。
九个上架包按 3/3/3 分到 ap / bp / gp。

**打开方式：外开（系统浏览器）** —— B 面不在 App 内渲染，因此 `bSideChromeColor`
（本包 `0xFFFFFFFF`）当前用不上。它仍与品牌吻合（浅色主题配浅色站），
所以这次定品牌**一处常量都没改、也不需要重新打包**。

外开还有一个连带结果：**`OpenBLanding` 这个 Adjust 事件只有外开才会触发**
（`gate_screen.dart`，且要浏览器确实唤起成功才发）。改成内开的话它永远不发。

## 编译期烧录的配置

全在 `lib/gate/gate_config.dart`：

| 字段 | 现状 |
| --- | --- |
| `bundleId` | ✅ `com.haymoor.dayspan3306` |
| `apiBases` | ✅ 与其余上架包同一个渠道中台基址，可再加候选抗封 |
| `appsFlyerDevKey` | ✅ 账号级 key，与其余包共用（AF 按账号发，不按 App 发） |
| `adjustAppToken` | ✅ **`jmnqnzl3ury8`** —— 若改回 `TODO_` 前缀，`TrackingService` 会跳过 Adjust 初始化，全链路 no-op（不崩，只是不上报） |
| `adjustOpenBLandingToken` | ✅ `3e00vd`（Adjust 后台 event `OpenBLanding`）—— 与上面成对生效，缺一个就整条 no-op |
| `bSideChromeColor` | ✅ `0xFFFFFFFF`（挂 `bp`）。当前外开，这个值用不上，见上 |

Adjust 后台已给本包**单独建了 App**（名 `DaySpan`，Android 平台 `com.haymoor.dayspan3306`，
store Google Play，reporting currency PHP，勾了「用户群仅在欧洲经济区之外」，
App Links 关闭）—— 逐项与 ColorStack 一致。token **不与任何其他包共用**：
复用会把本包的安装与会话归到别的 App 上（已核验 24 个 token 跨包无一重复）。

`DaySpan` 下已建齐与 ColorStack 相同的 7 个事件：`AddToCart` / `CompleteRegistration` /
`Login` / `OldRegPurchase` / **`OpenBLanding`** / `Purchase` / `TPFirstDeposit`。
上架包本体只发 `OpenBLanding`，其余 6 个是渠道壳 APK 的事件契约（ADR-0013），
建齐只为保持同构。可直接传 Console 的事件 CSV 在 `store/adjust-events.csv`。

## 打开方式的两个连带结果

**`bSideChromeColor` 只在内开时才用得上。** 外开时 B 面交给系统浏览器，
`web_screen.dart` 根本不会被走到。仍按品牌取值，是因为打开方式由服务端下发、
Console 改一下就生效，客户端不重新打包 —— 那一刻它立刻生效。

**`OpenBLanding` 事件只有外开才触发**（`gate_screen.dart`，且要浏览器确实唤起成功
才发）。改成内开的话它永远不发。

## Firebase

`android/app/google-services.json` **已放入** ✅。本包已在 Firebase 项目
**`hybrid-listings-51660`**（project_number `609439342540`）下以包名 `com.haymoor.dayspan3306` 注册
（`mobilesdk_app_id` = `1:609439342540:android:394c6eabacde76e8a98769`），**不加 SHA 证书指纹** —— 与 ColorStack 一致。
该文件随 APK 分发、反编译即可得，非机密，进 git。

**已按规矩裁剪成只含本包一个 client。** 控制台下载的原始文件会把该项目下**所有**
Android App 都列进来（本次下载里有 13 条），理由见 `SECURITY_NOTES.md`。
google-services 插件只按 applicationId 取匹配的那条，多余条目会被忽略。

> `android/app/build.gradle.kts` 里这个插件是**按文件存在与否条件应用**的 ——
> 文件在了插件才真正生效，所以加/删这个文件会改变构建路径，改动后要重新构建一次。
> 已重新构建并核验：生成的 `google_app_id` 正是本包那条。

## 启动路径上的一个坑

`main()` 里触发 Firebase 与归因初始化但**不 await**：

```dart
PushService.instance.initFirebase();   // 故意不 await
TrackingService.instance.init();       // 同上
runApp(const DaySpanApp());
```

启动路径上不放任何可能卡住的原生调用 —— `runApp` 之前一挂就是纯黑屏、App 完全打不开。
需要结果时各自内部带超时地等。

## 怎么验

### 判定回退（最重要的一条）

服务端不可达时必须进 A 面，而不是白屏或崩溃：

```bash
adb shell svc wifi disable && adb shell svc data disable
adb shell am force-stop com.haymoor.dayspan3306
adb shell monkey -p com.haymoor.dayspan3306 -c android.intent.category.LAUNCHER 1
# 等约 6 秒（超时）后应当进入工具界面
adb shell svc wifi enable
```

### 包里没有 B 面地址

```bash
unzip -p build/app/outputs/flutter-apk/app-release.apk \
  assets/flutter_assets/kernel_blob.bin 2>/dev/null | strings \
  | grep -Ei 'https?://' | sort -u
# 只应看到网关 API 基址与各 SDK 自己的域名，不应有任何落地页地址
```
