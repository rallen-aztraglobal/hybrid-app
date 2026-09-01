# App Review Notes — PocketLedger

> 本文两部分：
> 1. **提交时粘进 App Store Connect → App Review Information → Notes 的英文正文**
>    （代码块里的内容，可原样粘贴）
> 2. **提交前自查清单**（中文，内部用，**不要**粘进去）

---

## 一、粘进 App Review Information 的正文

App Store Connect 的 Notes 字段有 4000 字符上限，下面这段约 1 600 字符。

```
PocketLedger is an offline personal money tracker. Everything it does happens on
the device.

NO ACCOUNT IS NEEDED
There is no sign-up, no login and no server-side account, so no demo credentials
are required. The app opens straight into the Overview tab with a starting "Cash"
account and 14 built-in categories already in place, so every feature can be
reached immediately on a fresh install.

SUGGESTED WALKTHROUGH (about two minutes)
1. Accounts tab, "+": create an account and choose its type - Card, E-wallet,
   Cash or Bank account. Card balances are allowed to go negative and are shown
   in red.
2. Accounts tab, "+" again: create a second account of a different type.
3. Transactions tab, "+": record an expense. The Overview tab's net balance and
   this month's figures update immediately.
4. Transactions tab, "+": record a Transfer between the two accounts. One balance
   goes down and the other goes up; the net balance is unchanged, because a
   transfer is not spending.
5. Settings: switch the currency (17 are available); every amount in the app
   changes with it. "Export as CSV" opens the standard share sheet with a .csv
   file. "Erase all data" clears the ledger after a confirmation.

THIS IS NOT A FINANCIAL SERVICE
The app does not connect to any bank, card issuer, wallet or payment provider. It
initiates no payments or transfers of real money, reads no messages or
notifications, and gives no investment or financial advice. Every figure in it is
one the user typed in by hand. "Transfer" means moving a number between two of the
user's own manually created accounts inside the app; no money moves anywhere.

WHERE THE DATA GOES
Accounts and entries are stored in a single file in the app's own container and
are never uploaded. The only way anything leaves the device is the CSV export,
which the user starts and sends wherever they choose.

The app includes the AppsFlyer and Adjust measurement SDKs for install
attribution, and Firebase Cloud Messaging for notifications. These receive device
identifiers and app open/session events only; they are never given the content of
the ledger. Our own server receives, on launch, the platform name, the application
identifier and the device time zone. All of this is described in the privacy
policy and declared in App Privacy.

There are no ads, no in-app purchases and no subscriptions. The app is
iPhone-only and portrait-only, and requires iOS 15.6 or later.
```

> 写作口径：
> - 只讲 App 自身的功能与数据流向，**不出现任何内部字眼**（AB 面、B 面、网关、渠道、
>   落地页），也不提及其他上架包。
> - 「不是金融服务」那一段是**主动写的**，不是等审核问。Finance 类目 + 出现
>   "Transfer" 字样，很容易被按 Guideline 3.2.1（金融服务需要资质）多问一轮；
>   先把「没有任何真实资金流动」说死，能省一个来回。
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
- [ ] **在模拟器上跑通 `README_GATE.md` 的 9 步手验路径**，尤其第 5 步（转账后净资产不变）
      与第 9 步（杀进程重进数据还在）。
- [ ] **在真机上跑一次。** 模拟器不覆盖推送、ATT 弹窗、真实签名与 IDFA 行为。
- [ ] **截图产出。** 目前**一张都没有**，App Store 至少要 6.9" 一档。见 `STORE_ASSETS.md`。
- [ ] **确定 Apple Developer team。** `project.pbxproj` 里 `DEVELOPMENT_TEAM` 现在是空的，
      出不了可上架的包。附带一个需要运营先拍板的问题（Seller 名称公开一致），
      见 `README_GATE.md` §5 与 `RELEASE_SIGNING.md`。
- [ ] **隐私政策页托管完成、公网可访问**，URL 填进 App Store Connect。
      托管后必须从外部（换 IP 或无痕）真拉一次 —— 政策 URL 挂登录墙会被直接驳回。
- [ ] **Support URL 与支持邮箱定下来**，且邮箱自己发一封测试邮件确认收得到。
- [ ] **ATT 路线拍板**（补 ATT 请求，或收窄申报）。三方打架状态不能提交，
      见 `APP_STORE_LISTING.md` 的「IDFA / ATT」与 `APP_PRIVACY.md` 末节。
- [ ] **`UserSettingsStore.privacyPolicyURLString` 与 `supportEmail` 换成真值。**
      不换的话设置页点「Privacy policy」只会弹一句
      "The privacy policy link is not set up yet." —— 审核员大概率会点这一行。
      **改这两个常量需要重新出构建版本，别留到最后。**

### B. 提交前一定要核对的

- [ ] **`aps-environment` 要是 `production`。** `PocketLedger/PocketLedger.entitlements`
      现在是 `development`。Xcode 归档时通常会按 provisioning profile 自动替换成
      production，但**要在导出的 `.ipa` 里实际确认一遍**（解包看 embedded.mobileprovision
      与 entitlements），别假设。
- [ ] **Push Notifications capability 在 Xcode 里勾了。** `Info.plist` 里的
      `UIBackgroundModes = remote-notification` 已经写好，但 capability 是另一件事。
- [ ] **`GoogleService-Info.plist` 已放入**（Firebase 项目 `hybrid-listings-51660`，
      bundleId `com.stillwater.pocketledger`）。缺文件时推送整段 no-op，**不会崩**，
      所以不算阻塞项，但推送就是不工作的。
- [ ] **App 图标**：`Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png` 已就位，
      1024×1024、**24 位无 alpha**（带透明通道会被拒收）。归档后再确认一次没被工具链改回 32 位。
- [ ] **竖屏锁定**与 **iPhone only** 的设置没被误改
      （`UISupportedInterfaceOrientations = Portrait`、`TARGETED_DEVICE_FAMILY = 1`）。
- [ ] **`ITSAppUsesNonExemptEncryption = false`** 还在 `Info.plist` 里
      （在的话上传构建版本时不会再问出口合规）。
- [ ] **App Privacy 表单填完并与 `APP_PRIVACY.md` 一致**，尤其
      Location（Coarse Location）那个待定项已经拍板、几个包口径统一。
- [ ] **年龄分级问卷答完**，Simulated Gambling 与 Unrestricted Web Access 都是 None / No。
- [ ] **删掉任何调试用的东西**：`GateConfig.adjustEnvironment` 应为 `"production"`
      （当前就是），不要带着 `"sandbox"` 上架。

### C. 已知的小问题（不阻塞上架，但值得顺手修）

- **`PushService.report` 的字段名与内容不符。** payload 里键名是 `"model"`，
  值却是 `UIDevice.current.systemVersion`（操作系统版本，不是设备型号）。
  服务端字段语义因此是错的。修的时候**只改键名，不要顺手改成真发设备型号** ——
  那会扩大申报范围，`APP_PRIVACY.md` 得跟着改。
- **`PushService.postToFirstAvailable` 的重试是递归的**，且只在「非 200」时试下一个基址；
  网络层直接报错（`error != nil`）时也会走进 completion 并递归。当前只有一个候选基址，
  影响有限，加候选之前值得看一眼。

### D. 最可能被问到的三件事，先备好答案

1. **「Transfer 是不是在转真钱？」**
   不是。是把一个数字从用户自己手工创建的一个账户挪到另一个账户，纯本地记账动作，
   不接任何支付通道。Notes 里已经写了。
2. **「为什么一个离线记账 App 要联网 / 要推送权限？」**
   联网用于取应用配置与投递通知；账本内容不参与任何请求。隐私政策与 App Privacy 里
   已如实申报。
3. **「隐私政策链接点不开」**
   → 就是自查清单 A 里那两条（政策页未托管 / `UserSettingsStore` 里还是占位符）。
   这是本包**最容易被挑**的一条，务必在提交前解决。
