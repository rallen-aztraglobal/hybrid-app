# PocketLedger — AB 面网关接入说明

本工程（原生 iOS / Swift / UIKit，**仅 iOS**，bundleId `com.stillwater.pocketledger`）已接入
上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) /
[docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：
A 面进记账本体（`MainTabBarController`），B 面按 `openMode` 内开 `WebContainerViewController`
或外开系统浏览器。判定失败一律 A 面。

接入方式与 [decktallypro](../decktallypro/README_GATE.md) 同构，仅 A 面入口不同
（本包是记账主界面，decktallypro 是游戏主界面），另有一处刻意的加固见下。

## ⚠️ 本包尚未编译过

**这份代码是在 Windows 上写的，那台机器没有 Xcode 也没有 Swift 工具链，
所以本工程从未被编译、从未运行、也没有任何截图。**

已做的静态保证：

- 工程文件逐项对照 decktallypro 那份**已知可用**的 `project.pbxproj` 改写，
  26 个对象 id 的定义集与引用集完全重合，无悬空引用。
- **网关、推送、归因三层**（`Gate/` · `Push/` · `Core/Services/TrackingService.swift`）
  沿用 decktallypro 的写法与 API，只有 B 面 url 校验一处是有意加固的（见下）。
- **记账本体是全新写的**，用到了 decktallypro 里从未出现过的 API：
  `UIButton.Configuration`、`AttributedString`/`AttributeContainer`、
  `UITableView.sectionHeaderTopPadding`、`UIDatePicker.preferredDatePickerStyle`、
  以及整套 `Decimal` 运算。这些的最低版本都已逐个核对（均 ≤ iOS 15.0，部署目标 15.6），
  但**没有「在本仓库里已验证过」这层保险**，接手时值得知道。
- 26 个 SF Symbol 逐个核对可用性，全部是 iOS 13–14 时代的符号，没有 16+ 的。
- 40 个类型各定义一次；18 个 `#selector` 目标全部存在且带 `@objc`。

这些都替代不了一次真实编译。

**接手第一件事：在 Mac 上跑通编译**（见文末「本地验证」），把冒出来的问题修掉再谈其他。

### 交付前的静态审查已做过一轮，修掉了 6 个问题

代码写完后做过一次独立的静态审查（对照 decktallypro 逐项核），发现并已修复：

| 严重度 | 问题 | 修法 |
| --- | --- | --- |
| 必崩 | `LedgerStore.shared` 在自己的 `init` 里同步发 `.ledgerDidChange`，观察者回头访问该单例 → 同线程重入 `swift_once` → libdispatch 递归加锁崩溃。**只在全新安装的第一次冷启动出现**，崩过一次存档就落盘了、第二次不复现，极易被误判成偶发 | `persist(notify:)` 加开关，`init` 路径不发通知 |
| 必崩 | `PushService.start` 里 `if FirebaseApp.app() == nil { FirebaseApp.configure() }` 方向写反了 —— 未配置时 `app()` 正好返回 nil，等于把执行流**送进** `configure()`；而缺 `GoogleService-Info.plist` 时它抛的是 ObjC 异常，Swift catch 不住，直接 SIGABRT | 改为先探测 bundle 里有没有那份 plist，没有就整段跳过 |
| 高 | 账户页 `tableHeaderView` 高度写死 108，内部约束链实际需要约 117 → 首屏必然打断约束 | 改为 `systemLayoutSizeFitting` 自算高度，并做变更判定避免布局死循环 |
| 高 | 编辑既有记录时用 `"\(decimal)"` 回填（恒以 `.` 作小数点），而 `parseSigned` 先剥**本地**分组符 —— 在 de/es/it/id/pt-BR 等以 `.` 作分组符的地区，`12.50` 会被读成 `1250`。用户只是打开再保存，金额就放大一百倍 | 新增 `MoneyFormatter.editableText`，用关掉分组符的本地化十进制格式回填 |
| 高 | 滑动删除的 handler 里同步走到 `reloadData()`，此时该行滑动态还没收尾 | 先 `completion(true)`，删除挪到下一个 runloop |
| 高 | `NumberFormatter` 硬写 2 位小数，而支持的币种里 JPY/KRW/VND/IDR 是零小数位 | 不覆盖位数，交给 `.currency` 样式按币种取 |

另外修了两处稳健性：删除后 `dismiss` 挪到下一个 runloop（避免被消耗在 alert 上）、
设备币种取到后统一转大写再匹配（`Locale.Currency.identifier` 可能返回小写）。

## A 面是什么

PocketLedger 是一个记账 App。核心设定是**账户类型由用户自己选**：

- 新建账户时选 **Card / E-wallet / Cash / Bank account** 四选一，各自有图标、配色与一句说明。
- 每笔流水都挂在具体账户上；支出、收入之外还支持**账户间转账**（例如从银行卡充值到电子钱包）。
- 账户余额 = 期初余额 ± 其上全部流水。卡类账户余额可以为负（欠款），列表里标红。
- 统计口径：转账**不计入**收支，只挪动余额；净资产 = 全部账户余额之和。

四个标签页：Overview（净资产、本月收支、钱花在哪、最近几笔）/ Transactions（按天分组的流水）/
Accounts（账户与余额）/ Settings（币种、CSV 导出、清空数据、隐私政策）。

纯离线，无广告、无内购、无账号、无排行榜。全部数据存在 App 沙盒里的一个 JSON 文件中。

## 目录

```
PocketLedger/
  App/            AppDelegate（经 GateCoordinator 启动）+ MainTabBarController
  Gate/           AB 面网关（GateConfig / GateService / GateCoordinator /
                  LaunchGateViewController / WebContainerViewController）
  Push/           FCM token 注册（带 gateMode）
  Core/
    Models/       LedgerModels（账户/分类/流水）+ LedgerMath（余额与统计，纯函数）
    Services/     LedgerStore（JSON 落盘 + CRUD）· MoneyFormatter · UserSettingsStore ·
                  TrackingService（AF/Adjust）
    Theme/        AppTheme（浅色配色）
    UI/           SharedViews（卡片/徽章/统计块/空状态/表单行）+ LedgerObservingViewController
    Extensions/   UIView+Layout（布局与提示条）
  MVP/            Overview / Transactions / Accounts / Settings 四个模块
  Assets.xcassets AppIcon（1024，24 位无 alpha）+ AccentColor
  Base.lproj/     LaunchScreen.storyboard（纯文字，不依赖图片资源）
```

记账本体与网关**完全解耦**：`Core/Models`、`Core/Services/LedgerStore`、`MVP/` 里没有一处
引用 `Gate/`。把 `AppDelegate` 里的 `GateCoordinator.start(in:)` 换成
`window.rootViewController = MainTabBarController()` 就是一个干净的单机记账 App。

## 与 decktallypro 的唯一差异：B 面 url 多了一层校验

`GateService.parse` 里，decktallypro 只要求 `URL(string:)` 解析成功且字符串非空。
问题是 `URL(string: "somestring")` **会成功** —— 得到一个只有 path、没有 scheme 与 host 的
相对 URL，`WKWebView` 加载它只会白屏。用户看到一个坏掉的 App，比回退 A 面糟得多。

本包加了一层：

```swift
private static func usableURL(_ raw: String) -> URL? {
    guard !raw.isEmpty, let url = URL(string: raw) else { return nil }
    guard let scheme = url.scheme?.lowercased(), scheme == "http" || scheme == "https" else { return nil }
    guard let host = url.host, !host.isEmpty else { return nil }
    return url
}
```

挡不住的畸形值宁可判 A。这与 colorstack / calcpad / tilefit 的 Dart 侧口径一致
（那边的 `_isUsableUrl` 就是同一套判断）。**decktallypro 尚未同步这个加固。**

## 上线前必做

### 1. 后台建 listing 条目 —— 待做
Console → 上架包，新建一条：platform=`ios`、bundleId=`com.stillwater.pocketledger`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

条目建好之前，服务端对本包的 bundleId 查不到 listing，判定恒为 A 面 —— 这是 fail-closed
的预期行为，不是故障。

### 2. 填 `Gate/GateConfig.swift`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。
  **这是网关 API，不是 B 面地址**；基址迁移时改这里，可加多个候选抗封。
- `appsFlyerDevKey`：已填 `fXoKsKQwxPCRdhD8CD8q6F`（账号级 key，与其余上架包同一 AF 账号）。
- `appsFlyerAppleAppID`：**待填，现在是 `TODO_APPSTORE_APP_ID`**。要在 App Store Connect
  建好 App 条目后拿到那串数字 id（形如 6780248860，不带 `id` 前缀）。占位期间 AF 全链路 no-op。
- `adjustAppToken`：**Adjust 后台的 App 条目已建好，但 token 待回填**（现在仍是
  `TODO_ADJUST_APP_TOKEN`）。Adjust 里的 App 名为 **`PocketLedger`**，iOS 平台已配
  `com.stillwater.pocketledger`（store `itunes` / `app_store`，`app_state: not_verified` ——
  ASC 条目还没建，数字 App ID 留空，与 ColorStack 的 iOS 平台同样处理），
  reporting currency **PHP**、`no_eea_users: true`。
  取值方式：Adjust Suite → AppView → `PocketLedger` → App information，复制 12 位 App Token。
  **不可复用其他包的 token**（decktallypro `sn947o53ym80` / colorstack `bytg13h7yubk` /
  hexacolorsort `2yhxl7paa3ls`），复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：**event 已建好，token 待回填**（现在仍是空串）。
  `PocketLedger` 下已建齐与 DeckTallyPro 相同的 7 个事件：`AddToCart` /
  `CompleteRegistration` / `Login` / `OldRegPurchase` / **`OpenBLanding`** / `Purchase` /
  `TPFirstDeposit`（均非 unique）。
  取值方式：该 App → Events 页，复制 `OpenBLanding` 那一行的 6 位 token。
  留空时 `TrackingService.trackOpenBLanding()` 只发 AppsFlyer、跳过 Adjust —— 不崩，
  但 Adjust 侧看不到外开事件。
- `adjustContentViewToken`：保持空串。与其余上架包一致，本包不发这个事件。

### 3. 加 AppsFlyer / Adjust / Firebase 的 SPM 包（一次性）
三个 SDK 的调用都用 `#if canImport(...)` 包裹：**未加包也能编译（走 no-op）**，加了自动启用。
`project.pbxproj` 里已经声明好三个 `XCRemoteSwiftPackageReference`，Xcode 打开工程后会自动解析；
若要手动添加，File → Add Package Dependencies：

- AppsFlyer：`https://github.com/AppsFlyerSDK/AppsFlyerFramework`（产品 `AppsFlyerLib`，7.0.0+）
- Adjust：`https://github.com/adjust/ios_sdk`（产品 `AdjustSdk`，锁 5.4.0）
- Firebase：`https://github.com/firebase/firebase-ios-sdk`（产品 `FirebaseMessaging`，12.16.0+）

注意：**即便加了包，本包当前也不会有任何上报** —— `appsFlyerAppleAppID` 与 `adjustAppToken`
都还是占位符，`isConfigured()` 会把两个 SDK 都拦在初始化之前。这是预期行为。

> **启用归因之前必须先补 ATT。** `Info.plist` 里已经放了
> `NSUserTrackingUsageDescription`（ATT 弹窗文案），但工程里**没有**调用
> `ATTrackingManager.requestTrackingAuthorization` —— 也就是说现在这个包不会弹授权框，
> AF/Adjust 也拿不到 IDFA。这在当前状态下是自洽的（两个 SDK 本来就没启用，
> App Store Connect 的 IDFA 问题答 No）。
>
> 但一旦填上真 token 启用归因，就得同时做三件事：import `AppTrackingTransparency`、
> 在合适时机（**不要在冷启动第一屏**，那时授权率极低）调 `requestTrackingAuthorization`、
> 并把 App Privacy 与 IDFA 问卷答案同步改掉。少做任何一件都可能踩 5.1.2。
> 详见 `APP_PRIVACY.md` 与 `APP_STORE_LISTING.md` 里交叉引用的那两条路线。

### 4. 推送（FCM）—— 配置文件已就位 ✅，还差 Xcode 侧两步
`PushService.swift` 已写好（配置 Firebase、注册 APNs、取 FCM token 并带 `gateMode` 调
`POST /api/app/listing/register-token`）。服务端**只向 gateMode=B 的设备**发上架包推送。

- [x] `PocketLedger/GoogleService-Info.plist` **已放入**。本包已在 Firebase 项目
  **`hybrid-listings-51660`** 下以 bundleId `com.stillwater.pocketledger` 注册
  （`GOOGLE_APP_ID` = `1:609439342540:ios:13df0abd70c61efaa98769`）。
  sync group 会自动把它纳入 bundle，pbxproj 无需改动。
- [ ] Signing & Capabilities：加 **Push Notifications** + **Background Modes → Remote
  notifications**（`Info.plist` 里的 `UIBackgroundModes` 已经写好，capability 仍要在 Xcode 里勾）。
- [ ] APNs Auth Key（.p8）传到 Firebase 项目的 Cloud Messaging。

缺 plist 时的降级：`PushService.start` 会先 `guard Bundle.main.path(forResource:
"GoogleService-Info", ofType: "plist") != nil`，找不到就整段 no-op。
**不能**改回用 `FirebaseApp.app() == nil` 来判断 —— 未配置时它正好返回 nil，那样写等于把
执行流送进 `configure()`，而缺 plist 时它抛的是 ObjC 异常，Swift catch 不住、直接 SIGABRT。
现在文件在了，这条降级路径不会再走到，但判断得留着（改动/换项目时它仍是唯一的护栏）。

### 5. 签名与团队 —— 待做
`project.pbxproj` 里 `DEVELOPMENT_TEAM` 是**空的**、`CODE_SIGN_STYLE = Automatic`。
这样模拟器能直接跑（`CODE_SIGNING_ALLOWED=NO`），但**出不了可上架的包**。

上架前要填真实 team、并按 decktallypro 的做法切到 `CODE_SIGN_STYLE = Manual` +
`PROVISIONING_PROFILE_SPECIFIER[sdk=iphoneos*] = AppStore`。

> **一个需要运营先拍板的事**：本包若用与 decktallypro 相同的 Apple Developer 账号，
> App Store 商品页上的**卖家名称（Seller）是公开的、且两个 App 会完全一致** —— 任何人
> 点开都能看出这两个 App 同属一家。这与「各包使用互不相关的厂商命名空间
> （`deck` / `stillwater` / …）」的初衷直接冲突。所以这里没有替你填 team id：
> 要么接受这层公开关联，要么另开一个开发者账号。

### 6. 应用图标 —— 已就位 ✅
`Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png`，1024×1024，**24 位无 alpha**
（iOS 图标带透明通道会被 App Store 拒收）。

图案是品牌蓝渐变底 + 三根递增的白色圆角柱 + 一条基线，纯几何、不依赖字体渲染。
生成脚本没有进仓库（那是个一次性的 Dart 脚本，而本包是纯 Swift 工程，塞个 Dart 包进来不合适），
设计参数记在 `STORE_ASSETS.md` 里，需要改时照着重画即可。

## 本地验证（**必须在 Mac 上做，尚未执行过**）

```bash
cd listings/pocketledger

# 1. 先确认能编译（不签名，最快暴露语法与 API 问题）
xcodebuild -scheme PocketLedger -project PocketLedger.xcodeproj \
  -sdk iphonesimulator -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build

# 2. 跑起来看 A 面
open -a Simulator
xcodebuild -scheme PocketLedger -project PocketLedger.xcodeproj \
  -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 16' build
```

跑通后按这条路径手验一遍（覆盖了本包全部关键路径）：

1. 冷启动 → 应看到加载页转圈 → 进 A 面（Overview）。判定失败也必须进 A 面，不能白屏。
2. Accounts → ＋ → 建一个 **Card** 账户、期初余额用 `+−` 设成负数 → 列表里余额应标红。
3. 再建一个 **E-wallet** 账户。
4. Transactions → ＋ → 记一笔支出 → Overview 的净资产与「本月支出」应同步变化。
5. 记一笔 **Transfer**（卡 → 电子钱包）→ 两个账户余额一增一减，**净资产不变**，
   且这笔不计入本月支出。
6. Settings → 换币种 → 全部界面金额符号应立刻跟着变。
7. Settings → Export as CSV → 分享面板出现、导出的 .csv 能打开。
8. Settings → Erase all data → 回到只有一个 Cash 账户的初始状态。
9. 杀掉进程重进 → 数据仍在（落盘生效）。
10. **删掉 App 重装再冷启动一次**。这一步专门验上面表里那条「只在全新安装的第一次冷启动
    出现」的崩溃已经修好 —— 装着旧数据跑是验不出来的。
11. **一次金额精度往返**：记一笔 `0.10`、再记一笔 `0.20`，看账户余额是不是**正好** `0.30`；
    然后杀进程重进再看一次。账本用 `Decimal` 存，但要经 `JSONEncoder`/`JSONDecoder` 落盘，
    这条往返在本包上没有验证过。若发现有一分钱的偏差，把 `LedgerStore.Snapshot` 里的金额
    改成以字符串编码即可根治。
12. **换一个以 `.` 作分组分隔符的地区**（模拟器 Settings → General → Language & Region
    改成 Germany 或 Indonesia），打开一笔已有记录再直接保存，确认金额没变 ——
    这是上面表里那条「放大一百倍」的回归验证点。

## 仍未验证

- **一切**。本包从未编译、从未运行。上面「本地验证」整节都是待做。
- B 面两条路径（内开 / 外开）—— 服务端尚未建本包 listing 条目，正常判定恒为 A。
- 服务端真实判 B、推送、Adjust/AF 上报 —— 都还缺后台条目与配置。
- 商店素材：截图与预览图**一张都没有**（要先能跑起来才能截），见 `STORE_ASSETS.md`。

## 接入红线（已落实）

- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端硬编码保证，客户端无从绕过。
