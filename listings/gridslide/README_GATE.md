# GridSlide — AB 面网关接入说明

本工程（原生 iOS / Swift / UIKit，**仅 iOS**，bundleId `com.fernvale.gridslide`）已接入
上架包 AB 面网关（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) /
[docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。启动即向渠道中台判定 A/B：
A 面进游戏本体（`MainTabBarController`），B 面按 `openMode` 内开
`WebContainerViewController` 或外开系统浏览器。判定失败一律 A 面。

接入方式与 [decktallypro](../decktallypro/README_GATE.md) /
[pocketledger](../pocketledger/README_GATE.md) 同构。

## ⚠️ 本包尚未编译过

**这份代码是在 Windows 上写的，那台机器没有 Xcode 也没有 Swift 工具链，
所以本工程从未被编译、从未运行、也没有任何截图。**

**接手第一件事：在 Mac 上跑通编译**（见文末「本地验证」）。

## 但算法层是真的验证过的

这一点与 pocketledger 不同，值得单独说。

`Core/Models/SlidePuzzle.swift` 里的规则内核 —— 合法性判定、整排滑动、打乱 ——
在写进本工程之前，先用**等价的 Dart 实现跑了 19 项测试并全部通过**
（Dart 在那台机器上有工具链，Swift 没有）。覆盖的点：

| 测的是什么 | 为什么值得测 |
| --- | --- |
| 复原态的构造与判定（3/4/5 三种尺寸） | 下标算错会让「差一格」被判成已完成 |
| `canMove` 的同行/同列/斜向/空格自身/越界 | 越界不返回 false 就会数组越界崩 |
| 相邻一格滑动、同行整排滑动、同列整排滑动 | 行与列只差一个步长（1 与 size），最容易只改对一半 |
| 非法点击不改动任何状态 | 静默改坏棋盘是最难查的一类 bug |
| 每一步都可逆（走一步再走回去 = 原状） | 这是打乱必定可解的前提 |
| 打乱后格子集合不变、同种子可复现 | 防止「凭空多出/丢失一个数字」 |
| **打乱产出的局面必定可解** | 见下 |

最后一条是重点。滑块拼图有**一半的随机排列是无解的** —— 玩家怎么推都推不回去。
测试里并排跑了两组各 200 次：朴素随机排列有一半以上按逆序数奇偶判据为不可解，
而本实现的 `shuffle` 产出的 200 个局面**全部可解**。做法是「从复原态开始随机走合法步」
而不是「随机排列后再修奇偶性」，后者边界多、极易写错。

`SlidePuzzle.swift` 是那份 Dart 实现的逐行转写，字段名与方法名刻意保持一致，便于对照。
**但要清楚：验证的是算法，不是这段 Swift。** 转写有没有笔误，仍要靠在 Mac 上编译并实跑确认。

## A 面是什么

数字滑块拼图。3×3 / 4×4 / 5×5 三种尺寸，点击与空格同行或同列的块即可滑动
（**支持整排滑动** —— 点隔着几格的块，中间的块一起挪，不必一格一格点）。
计步数与用时，各尺寸分别记录最少步数与最短用时。

三个标签页：Play / Records / Settings。纯离线，无广告、无内购、无账号、无排行榜。
成绩存在设备本地的 UserDefaults 里。

几个刻意的设计决定：

- **步数与时间各记各的纪录。** 有人追求最少步、有人追求最快，把两者绑在一起
  （只记「最少步那局的时间」）会让另一半玩家的努力看不见。
- **第一步落下才开始计时**，打开 App 放着不动不算进成绩。
- **计时按时间戳算，不是「每秒加一」。** 后者在 App 退到后台、timer 被系统停掉时会少算，
  回前台又接着加，最终显示的既不是真实耗时也不是游玩时长。`GameClock` 累计的是
  「确实在玩的那几段」之和，切后台、切标签页都会暂停。
- **归位的块换成强调色。** 这是这类拼图最有用的一条即时反馈。

## 目录

```
GridSlide/
  App/            AppDelegate（经 GateCoordinator 启动）+ MainTabBarController
  Gate/           AB 面网关（GateConfig / GateService / GateCoordinator /
                  LaunchGateViewController / WebContainerViewController）
  Push/           FCM token 注册（带 gateMode）
  Core/
    Models/       SlidePuzzle（规则内核，纯 Swift、无 UIKit 依赖）
    Services/     RecordStore（成绩）· SettingsStore · TrackingService（AF/Adjust）
    Theme/        AppTheme（深色配色）
    UI/           SharedViews（面板/统计块/表单行/分隔线）
    Extensions/   UIView+Layout（布局与提示条）
  MVP/            Play / Records / Settings 三个模块
  Assets.xcassets AppIcon（1024，24 位无 alpha）+ AccentColor
  Base.lproj/     LaunchScreen.storyboard（纯文字，不依赖图片资源）
```

游戏本体与网关**完全解耦**：`Core/Models`、`Core/Services/RecordStore.swift`、`MVP/`
里没有一处引用 `Gate/`。把 `AppDelegate` 里的 `GateCoordinator.start(in:)` 换成
`window.rootViewController = MainTabBarController()` 就是一个干净的单机游戏。

## 交付前的静态审查已做过一轮

代码写完后做过一次独立的静态审查（对照 decktallypro 与 pocketledger 逐项核）。
**没有发现会导致编译失败或必崩的问题**，但查出四个运行时缺陷，均已修复：

| 问题 | 表现 | 修法 |
| --- | --- | --- |
| 切标签页后**计时永久冻结** | `viewWillDisappear` 暂停了计时，`viewWillAppear` 却从没恢复；而 `boardViewDidMove` 只在 `moves == 0` 时启动计时，所以回来后继续点也救不回来。**冻结值还会被当成「最短用时」提交进纪录** | `viewWillAppear` 里按「牌局是否进行中」恢复计时 |
| 后台返回时给**不可见页面偷偷加时间** | 本页是 tab 根控制器、永不销毁，人在 Records 页时它照样收得到「回到前台」通知，于是重新开始走表 | 加 `isOnScreen` 标记，不在屏上直接 return |
| iPhone SE 上「**New game」按钮点不到** | 我原先的高度估算漏算了大标题导航栏（`prefersLargeTitles` 下恒占约 96pt）。而我为了「布局完全确定」刻意没给下边界 —— 恰恰是这个决定让小屏没有退让余地，按钮被顶到 tab bar 底下 | 本页关掉大标题（省 52pt）；棋盘宽度降为 `.defaultHigh` 并加「按钮不越过安全区底边」的约束，空间不够时压小棋盘而不是挤走按钮 |
| 改棋盘尺寸后**三处状态不同步** | 在 Settings 页改了尺寸，回 Play 页只刷新了 Best（读的是新尺寸的纪录），而分段控件和棋盘还是旧的；玩完还会用旧尺寸记账 | `viewWillAppear` 里比对存储值与当前棋盘，不一致就开新局 |

另修了四处小问题：棋盘左/上留白区被误判进第 0 行/列（`Int()` 向零取整，左右不对称）、
开新局的交叉淡入因 `layoutIfNeeded()` 位置写反而空转、`AttributeContainer.font` 改用
显式 `UIFont.` 避免双重类型推导、统计块标题在 320pt 宽屏上允许缩放避免截断。

审查同时逐项确认了：pbxproj 的 22 个对象 id 自洽无悬空引用、30 个顶层类型无冲突、
10 个 `#selector` 目标齐全、SF Symbol 与系统 API 全部满足 iOS 15.6、
`SlidePuzzle` 的转写**未发现笔误**（下标运算、循环终止、值语义逐行核过）。

## 已经规避的坑（来自 pocketledger 的审查）

同批次的 pocketledger 做过一轮静态审查，查出两个「第一次打开就死」的问题。
本包在写的时候就绕开了，这里记一笔免得日后改动时又踩回去：

- **单例的 `init` 里绝不发通知。** `RecordStore` / `SettingsStore` 的 `init` 只读不写、
  不 post 任何 Notification。原因：`static let shared` 的惰性初始化由 `swift_once` 保护，
  那把锁**不可重入** —— 在 init 里发通知，观察者回头访问 `shared` 就是同线程二次进入，
  libdispatch 直接崩。而且这种崩溃只在全新安装的第一次冷启动出现，极易被误判成偶发。
- **`FirebaseApp.configure()` 前先探测 plist 是否存在。** 不能用
  `if FirebaseApp.app() == nil` 来兜底 —— 未配置时它正好返回 nil，那样写等于把执行流
  **送进** `configure()`，而缺 `GoogleService-Info.plist` 时它抛的是 ObjC 异常，
  Swift catch 不住，直接 SIGABRT。

## 上线前必做

### 1. 后台建 listing 条目 —— 待做
Console → 上架包，新建一条：platform=`ios`、bundleId=`com.fernvale.gridslide`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

条目建好之前，服务端对本包的 bundleId 查不到 listing，判定恒为 A 面 —— 这是 fail-closed
的预期行为，不是故障。

### 2. 填 `Gate/GateConfig.swift`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。
  **这是网关 API，不是 B 面地址**。
- `appsFlyerDevKey`：已填（账号级 key，与其余上架包同一 AF 账号）。
- `appsFlyerAppleAppID`：**待填**，现在是 `TODO_APPSTORE_APP_ID`。要在 App Store Connect
  建好 App 条目后拿到那串数字 id。占位期间 AF 全链路 no-op。
- `adjustAppToken`：**已填 `4w8yd18jd0qo`** ✅。Adjust 里的 App 名为 **`GridSlide`**，
  iOS 平台已配 `com.fernvale.gridslide`（store `itunes` / `app_store`，
  `app_state: not_verified` —— ASC 条目还没建，数字 App ID 留空，与 ColorStack 的
  iOS 平台同样处理），reporting currency **PHP**、`no_eea_users: true`。
  这是本包**专属**的 token，与其余包互不相同（decktallypro `sn947o53ym80` /
  colorstack `bytg13h7yubk` / hexacolorsort `2yhxl7paa3ls` / pocketledger `zoavz0rdks1s`）——
  复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：**已填 `intubq`** ✅（取自 `GridSlide` 的 Events 页）。
  `GridSlide` 下已建齐与 DeckTallyPro 相同的 7 个事件：`AddToCart` / `CompleteRegistration` /
  `Login` / `OldRegPurchase` / **`OpenBLanding`** / `Purchase` / `TPFirstDeposit`（均非 unique）。
  其余 6 个是渠道壳 APK 的事件契约（ADR-0013），上架包本体只发 `OpenBLanding`，
  建齐只是为了与既有上架包保持同构。
- `adjustContentViewToken`：保持空串，与其余上架包一致。

### 3. SPM 包 —— 已在工程里声明好 ✅
`project.pbxproj` 里已声明三个 `XCRemoteSwiftPackageReference` 并挂进 Frameworks 阶段，
Xcode 打开工程会自动解析，**不需要手工 Add Package**：

- AppsFlyer：`https://github.com/AppsFlyerSDK/AppsFlyerFramework`（产品 `AppsFlyerLib`，7.0.0+）
- Adjust：`https://github.com/adjust/ios_sdk`（产品 `AdjustSdk`，锁 5.4.0）
- Firebase：`https://github.com/firebase/firebase-ios-sdk`（产品 `FirebaseMessaging`，12.16.0+）

> **别被代码里的 `#if canImport(...)` 误导。** 那层守卫的本意是「没加包也能编译」，
> 但本工程**已经把包接上了**，所以第一次构建时三个 `canImport` 全部为真、SDK 调用
> 会真的参与编译。当前不上报靠的是**另一层**：`appsFlyerAppleAppID` 与 `adjustAppToken`
> 仍是 `TODO_` 占位符，`GateConfig.isConfigured()` 把 SDK 拦在初始化之前。
>
> 实际影响：**第一次 `xcodebuild` 必须能联网解析这三个包**，否则会卡在 package
> resolution 上。那不是代码问题，但报错看起来像编译失败，别被带偏。

> **启用归因之前必须先补 ATT。** `Info.plist` 里已放 `NSUserTrackingUsageDescription`，
> 但工程里**没有**调用 `ATTrackingManager.requestTrackingAuthorization` —— 现在不会弹授权框，
> AF/Adjust 也拿不到 IDFA。当前状态自洽（SDK 本来就没启用，IDFA 问卷答 No）。
> 一旦填上真 token，就要同时：import `AppTrackingTransparency`、在合适时机
> （**不要在冷启动第一屏**，授权率极低）请求授权、并同步改 App Privacy 与 IDFA 答案。

### 4. 推送（FCM）—— 配置文件已就位 ✅，还差 Xcode 侧两步
`PushService.swift` 已写好。

- [x] `GridSlide/GoogleService-Info.plist` **已放入**。本包已在 Firebase 项目
  **`hybrid-listings-51660`** 下以 bundleId `com.fernvale.gridslide` 注册
  （`GOOGLE_APP_ID` = `1:609439342540:ios:068e2b62f6cd0639a98769`）。
  sync group 自动纳入 bundle，pbxproj 无需改动。
- [ ] Signing & Capabilities：加 **Push Notifications** + **Background Modes → Remote notifications**。
- [ ] APNs Auth Key（.p8）传到 Firebase 项目的 Cloud Messaging。

缺 plist 时整段推送 no-op（见上「已经规避的坑」），不影响网关与游戏本体。
现在文件在了，这条降级路径不会再走到，但 `PushService.start` 里那道
`Bundle.main.path(forResource: "GoogleService-Info", ...)` 的判断得留着 ——
改动或换 Firebase 项目时它仍是唯一的护栏。

### 5. 签名与团队 —— 待做
`DEVELOPMENT_TEAM` 是**空的**、`CODE_SIGN_STYLE = Automatic`。模拟器能直接跑
（`CODE_SIGNING_ALLOWED=NO`），但**出不了可上架的包**。

> 与 pocketledger 同样的待拍板项：本包若用与 decktallypro 相同的 Apple Developer 账号，
> App Store 商品页上的**卖家名称是公开的、且几个 App 会完全一致** —— 任何人都能看出
> 同属一家，与「各包使用互不相关的厂商命名空间」的初衷冲突。所以这里没有替你填 team id。

### 6. 应用图标 —— 已就位 ✅
`Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png`，1024×1024，**24 位无 alpha**
（iOS 图标带透明通道会被 App Store 拒收）。

图案是深色渐变底 + 3×3 网格，**右下角空一格**（滑块拼图之所以能玩全靠那个空格），
空格左边那块用强调色，读起来就是「它正要滑进空位」。纯几何、不依赖字体。

## 本地验证（**必须在 Mac 上做，尚未执行过**）

```bash
cd listings/gridslide

# 1. 先确认能编译（不签名，最快暴露语法与 API 问题）
xcodebuild -scheme GridSlide -project GridSlide.xcodeproj \
  -sdk iphonesimulator -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build

# 2. 跑起来
xcodebuild -scheme GridSlide -project GridSlide.xcodeproj \
  -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 16' build
```

跑通后按这条路径手验（覆盖了本包全部关键路径）：

1. 冷启动 → 加载页转圈 → 进 A 面（Play）。判定失败也必须进 A 面，不能白屏。
2. **删掉 App 重装再冷启动一次** —— 专门验单例初始化那条路径不崩。
3. 点一个与空格相邻的块 → 它滑进空位，步数 +1，计时开始走。
4. 点同一行上隔着两格的块 → **中间的块应一起滑过去**，且只算一步。
5. 点斜对角的块 → 应毫无反应（不是崩，也不是乱动）。
6. 把某块滑到它的正确位置 → 该块应变成强调色。
7. 真把一局解开（3×3 最快） → 结算浮层出现，步数/用时正确，首次通关应显示
   「New best moves and time」。
8. 切到 Records 页 → 3×3 那张卡显示刚才的成绩；另外两个尺寸仍是破折号。
9. **计时正确性（两个方向都要试，它们是相反的错法）**：
   - *不该多算*：开一局走两步 → 按 Home 键回桌面等 30 秒 → 回到 App，用时**不应**多出 30 秒。
     再切到 Records 页停 30 秒 → 期间 App 退后台再回前台 → 切回 Play 页，同样不应多算。
   - *不该冻住*：走两步 → 切到 Records 页 → 切回 Play 页 → **用时应继续往上走**。
     这一条专门验「切标签页后计时永久冻结」那个缺陷（它会让冻结值被当成最短用时记进纪录）。
10. Settings → 换棋盘尺寸 → 回 Play 页应是新尺寸的新局。
11. Settings → 关掉 Haptics → 走一步不应再震。
12. Settings → Reset all records → Records 页全部回到破折号。

## 仍未验证

- **这段 Swift 代码本身**。算法验过（见上），转写没验过；UI 层完全没验过。
- B 面两条路径（内开 / 外开）—— 服务端尚未建本包 listing 条目，正常判定恒为 A。
- 服务端真实判 B、推送、Adjust/AF 上报 —— 都还缺后台条目与配置。
- 真机（连模拟器都还没跑过）。
- 商店素材：截图与预览图**一张都没有**（要先能跑起来才能截）。

## 接入红线（已落实）

- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- B 面 url 还多一层 scheme + host 校验（与 pocketledger 同口径，decktallypro 尚未同步）：
  `URL(string: "somestring")` 会成功但没有 scheme 与 host，WKWebView 加载它只会白屏，
  那比回退 A 面糟得多。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端硬编码保证，客户端无从绕过。
