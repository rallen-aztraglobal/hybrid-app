# App Review Notes — GridSlide

> 本文两部分：
> 1. **提交时粘进 App Store Connect → App Review Information → Notes 的英文正文**
>    （代码块里的内容，可原样粘贴）
> 2. **提交前自查清单**（中文，内部用，**不要**粘进去）

---

## 一、粘进 App Review Information 的正文

App Store Connect 的 Notes 字段有 4000 字符上限，下面这段约 3 200 字符。

```
GridSlide is an offline number slide puzzle. Everything it does happens on the
device, and no account is involved.

NO ACCOUNT IS NEEDED
There is no sign-up, no login and no server-side account, so no demo credentials are
required. The app opens straight into a shuffled 4x4 board, so every feature can be
reached immediately on a fresh install.

SUGGESTED WALKTHROUGH (about two minutes)
1. Play tab: the board is already shuffled. Tap a numbered tile that sits next to the
   empty square - it slides in, the move counter goes to 1 and the clock starts. The
   clock deliberately does not start until the first move.
2. Tap a tile in the same row as the empty square but two or three places away. Every
   tile between it and the gap slides along with it, in one gesture, and the move
   counter goes up by one, not by three. This whole-row slide is the main thing that
   makes the game comfortable to play.
3. Tap a tile that is neither in the same row nor the same column as the empty square.
   Nothing happens; that move is not legal.
4. Slide any tile onto the square where it belongs. It turns green to show it is in
   place.
5. Switch the board to 3x3 with the control at the top and solve it - it takes well
   under a minute. A summary appears with the moves and time taken, and a line saying
   which record was set.
6. Records tab: the 3x3 card now shows a best-moves figure, a best-time figure and a
   solve count. The 4x4 and 5x5 cards still show dashes. Best moves and best time are
   tracked independently, so a careful run and a fast run are both recorded.
7. Settings tab: default board size, a haptics switch, a link to the privacy policy,
   and "Reset all records", which clears the Records tab after a confirmation.

NO ADS, NO PURCHASES, NO SIGN-IN, NO CONNECTION NEEDED
There are no advertising placements and no ad SDK with a display surface. There are
no in-app purchases and no subscriptions - nothing in the app is paid for. There is
no sign-in of any kind, including no third-party or social login. There is no
leaderboard, no chat, no user profile and no user-generated content. The game is
fully playable with the device in airplane mode; nothing in it is gated on a network
response.

There is also no gambling or simulated gambling content of any kind: no wheels, no
slots, no cards, no wagers, no virtual currency and no prizes. Randomness is used in
exactly one place - shuffling the board at the start of a round - and every move
after that is decided by the player.

WHERE THE DATA GOES
The best moves, best times and solve counts are stored in the app's own storage on the
device and are never uploaded. Nothing about how the user plays leaves the device.

The app includes the AppsFlyer and Adjust measurement SDKs for install attribution,
and Firebase Cloud Messaging for notifications. These receive device identifiers and
app open/session events only; they are never given scores or gameplay data. Our own
server receives, on launch, the platform name, the application identifier and the
device time zone. All of this is described in the privacy policy and declared in App
Privacy.

The app is iPhone-only, portrait-only and dark-themed, and requires iOS 15.6 or later.
```

> 写作口径：
> - 只讲 App 自身的功能与数据流向，**不出现任何内部字眼**（AB 面、B 面、网关、渠道、
>   落地页），也不提及其他上架包。
> - 「无广告 / 无内购 / 无第三方登录 / 完全离线可玩」那一段是**主动写的**。
>   一个免费的休闲游戏，审核员默认会去找变现点；先把「没有变现点、也不需要联网」
>   说死，能省一个来回。
> - 「无博彩」那一段同样是主动写的。Games 类目下的免费休闲游戏是模拟博彩内容的
>   高发区，而本 App 唯一用到随机数的地方（开局打乱）恰好是容易被误读的点，
>   顺手说清楚。口径与 `APP_STORE_LISTING.md` 年龄分级那一节完全一致。
> - **演示路径要能一步步照着点出来**：审核员照着走一遍，正好覆盖整排滑动、
>   非法点击、归位高亮、双纪录、清空成绩这几条。第 5 步特意让他解 3×3
>   而不是 4×4 —— 3×3 一分钟内能解开，4×4 不一定。
> - 演示账号留空（Sign-in required = **No**）。

## 二、App Review Information 表单其余字段

| 字段 | 填什么 |
| --- | --- |
| Sign-in required | **No**（无账号体系，无需演示账号） |
| First name / Last name | 填实际能接审核电话/邮件的人 |
| Phone number | 同上，必须真实可达 |
| Email address | `TODO_SUPPORT_EMAIL`（与商品页支持邮箱可以是同一个，但**都不要复用其他上架包的**） |
| Notes | 上面那段 |
| Attachment | 不需要 |

---

## 三、提交前自查清单

### A. 阻塞项 —— 没做完就不要提交

- [ ] **在 Mac 上编译通过。** 本工程**从未编译过**（写代码的机器是 Windows，没有 Xcode
      与 Swift 工具链）。命令见 `README_GATE.md` 末节。**这是第一件事**，编译不过谈别的没意义。
      注意：`Core/Models/SlidePuzzle.swift` 的**算法**在 Dart 侧跑过 19 项测试，
      但**这段 Swift 是逐行转写，转写本身没验过**，UI 层则完全没验过。
- [ ] **在模拟器上跑通 `README_GATE.md` 的 12 步手验路径**，尤其：
      第 4 步（整排滑动只算一步）、第 5 步（斜向点击无反应）、
      第 9 步（切后台 30 秒回来用时不多算）、第 2 步（删掉重装再冷启动不崩）。
- [ ] **在真机上跑一次。** 模拟器不覆盖推送、震动反馈、真实签名与 IDFA 行为
      （震动是本包的一个设置项，模拟器上根本感受不到）。
- [ ] **截图产出。** 目前**一张都没有**，App Store 至少要 6.9" 一档。见 `STORE_ASSETS.md`。
- [ ] **确定 Apple Developer team。** `project.pbxproj` 里 `DEVELOPMENT_TEAM` 现在是空的，
      出不了可上架的包。附带一个需要运营先拍板的问题（Seller 名称公开一致），
      见 `README_GATE.md` §5 与 `RELEASE_SIGNING.md`。
- [ ] **隐私政策页托管完成、公网可访问**，URL 填进 App Store Connect。
      托管后必须从外部（换 IP 或无痕）真拉一次 —— 政策 URL 挂登录墙会被直接驳回。
- [ ] **Support URL 与支持邮箱定下来**，且邮箱自己发一封测试邮件确认收得到。
- [ ] **ATT 路线拍板**（补 ATT 请求 + 填 key，或维持现状并收窄申报）。
      三方打架状态不能提交，见 `APP_STORE_LISTING.md` 的「IDFA / ATT」与 `APP_PRIVACY.md` 末节。
- [ ] **`SettingsStore.privacyPolicyURLString` 与 `supportEmail` 换成真值。**
      不换的话设置页点「Privacy policy」只会弹一句
      "The privacy policy link is not set up yet." —— 审核员大概率会点这一行
      （演示路径第 7 步就写着 Settings 页有这一行）。
      **改这两个常量需要重新出构建版本，别留到最后。**

### B. 提交前一定要核对的

- [ ] **`aps-environment` 要是 `production`。** `GridSlide/GridSlide.entitlements`
      现在是 `development`。Xcode 归档时通常会按 provisioning profile 自动替换成
      production，但**要在导出的 `.ipa` 里实际确认一遍**（解包看 embedded.mobileprovision
      与 entitlements），别假设。
- [ ] **Push Notifications capability 在 Xcode 里勾了。** `Info.plist` 里的
      `UIBackgroundModes = remote-notification` 已经写好，但 capability 是另一件事。
- [ ] **`GoogleService-Info.plist` 已放入**（Firebase 项目 `hybrid-listings-51660`，
      bundleId `com.fernvale.gridslide`）。缺文件时推送整段 no-op，**不会崩**，
      所以不算阻塞项，但推送就是不工作的。
- [ ] **App 图标**：`GridSlide/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png` 已就位，
      1024×1024、**24 位无 alpha**（带透明通道会被拒收）。归档后再确认一次没被工具链改回 32 位。
- [ ] **竖屏锁定**与 **iPhone only** 的设置没被误改
      （`INFOPLIST_KEY_UISupportedInterfaceOrientations = UIInterfaceOrientationPortrait`、
      `TARGETED_DEVICE_FAMILY = 1`、`SUPPORTS_MACCATALYST = NO`）。
- [ ] **`ITSAppUsesNonExemptEncryption = false`** 还在 `Info.plist` 里
      （在的话上传构建版本时不会再问出口合规）。
- [ ] **App Privacy 表单填完并与 `APP_PRIVACY.md` 一致**，尤其
      Location（Coarse Location）那个待定项已经拍板、几个包口径统一，
      以及 **Gameplay Content 确认不勾**（成绩纯本地，理由见 `APP_PRIVACY.md`）。
- [ ] **年龄分级问卷答完**：Simulated Gambling / Gambling / Contests 三项都是
      **None / No / None**，Unrestricted Web Access 是 No，Advertising 是 No。
- [ ] **Game Center 保持关闭**（本 App 无排行榜、无成就）。
- [ ] **删掉任何调试用的东西**：`GateConfig.adjustEnvironment` 应为 `"production"`
      （当前就是），不要带着 `"sandbox"` 上架。

### C. 已知的小问题（不阻塞上架，但值得顺手修）

- **`PushService.report` 的字段名与内容不符。** payload 里键名是 `"model"`，
  值却是 `UIDevice.current.systemVersion`（操作系统版本，不是设备型号）。
  服务端字段语义因此是错的。修的时候**只改键名，不要顺手改成真发设备型号** ——
  那会扩大申报范围，`APP_PRIVACY.md` 得跟着改。
- **`PushService.postToFirstAvailable` 的重试是递归的**，且只在「非 200」时试下一个基址；
  网络层直接报错时也会走进 completion 并递归。当前只有一个候选基址，影响有限，
  加候选之前值得看一眼。
- **`SlidePuzzle.shuffle` 在恰好打乱回复原态时会递归多走一轮。** 逻辑是对的
  （`steps + size` 递增，不会无限），但这条分支在 Swift 侧没实跑过，
  编译通过后开局多按几次 "New game" 确认没有卡顿或死循环。

### D. 最可能被问到的四件事，先备好答案

1. **「一个免费游戏靠什么赚钱？为什么没有广告和内购？」**
   本版本确实没有任何变现点：无广告位、无内购、无订阅。包里有归因 SDK（AppsFlyer /
   Adjust）用于衡量推广效果，这在 App Privacy 与隐私政策里都已申报。Notes 里已写。
2. **「开局是随机的，这算不算模拟赌博？」**
   不算。随机只用于生成初始局面（相当于洗牌），之后每一步都由玩家决定，
   没有下注、没有赔付、没有虚拟货币、没有奖品。年龄分级问卷里三项博彩相关问题
   全部答 None/No，理由见 `APP_STORE_LISTING.md`。
3. **「为什么一个离线拼图要联网 / 要推送权限？」**
   联网用于取应用配置与投递通知；游戏本体与成绩不参与任何请求，飞行模式下完全可玩。
   隐私政策与 App Privacy 里已如实申报。
4. **「隐私政策链接点不开」**
   → 就是自查清单 A 里那两条（政策页未托管 / `SettingsStore` 里还是占位符）。
   这是本包**最容易被挑**的一条，务必在提交前解决。
