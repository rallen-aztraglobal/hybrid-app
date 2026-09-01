# PocketLedger

原生 iOS 记账 App（Swift + UIKit），**仅 iOS**，作为上架包接入了 AB 面网关。

- bundleId `com.stillwater.pocketledger`
- 商店名 `PocketLedger`
- A 面 = 记账本体（`MVP/` 四个模块），B 面 = 服务端下发的 web

> ⚠️ **本工程从未编译过。** 代码是在一台没有 Xcode 的 Windows 机器上写的。
> 接手第一件事是在 Mac 上跑通编译，详见 `README_GATE.md` 的「本地验证」。

## 这个 App 做什么

记账，核心设定是**账户类型由用户自己选**：

- 建账户时四选一 —— **Card / E-wallet / Cash / Bank account**，各有图标、配色与说明。
  卡和电子钱包分开记，是因为这两种钱的用法与余额语义本来就不同。
- 每笔流水挂在具体账户上，支持**支出 / 收入 / 账户间转账**（如银行卡充值到电子钱包）。
- 账户余额 = 期初余额 ± 其上全部流水；卡类余额可为负（欠款），列表里标红。
- 转账不计入收支统计，只挪动余额；净资产 = 全部账户余额之和。

四个标签页：Overview / Transactions / Accounts / Settings。
纯离线，无广告、无内购、无账号。数据存在设备本地的一个 JSON 文件里，可导出 CSV。

## 从哪份文档开始看

| 想做的事 | 看这份 |
| --- | --- |
| 搞懂网关怎么接的、还差什么没配、**怎么在 Mac 上跑起来** | **`README_GATE.md`**（先看这份） |
| 证书、描述文件、App Store Connect | `RELEASE_SIGNING.md` |
| 填 App Privacy（隐私营养标签） | `APP_PRIVACY.md` |
| 上传商店文案 | `APP_STORE_LISTING.md` |
| 提交给审核员的备注与自查 | `APP_REVIEW_NOTES.md` |
| 截图 / 预览图规格与现状 | `STORE_ASSETS.md` |
| 隐私政策正文 | `PRIVACY_POLICY.md`（可托管版：`store/privacy-policy.html`） |
| 密钥与机密的边界 | `SECURITY_NOTES.md` |

## 当前状态

**已完成**：全部 Swift 源码、Xcode 工程（照 decktallypro 那份已知可用的 pbxproj 改写，
用 Xcode 16 的文件系统同步组，加文件不必改工程）、应用图标（1024，24 位无 alpha）、
LaunchScreen、网关/推送/归因三层接入。

**尚未完成**：

- [ ] **在 Mac 上编译并跑通**（从未编译过 —— 这是第一优先级）
- [ ] App Store Connect 建 App 条目，拿到数字 App ID 填进 `GateConfig.appsFlyerAppleAppID`
- [ ] Adjust 后台建 App，填 `adjustAppToken` 与 `adjustOpenBLandingToken`
- [ ] Firebase 注册本包，放入 `PocketLedger/GoogleService-Info.plist`
- [ ] 定 Apple Developer team 并配好签名（现在 `DEVELOPMENT_TEAM` 是空的，出不了上架包）
- [ ] 渠道中台建 listing 条目（建好之前判定恒为 A 面，属预期的 fail-closed）
- [ ] 托管隐私政策、定支持邮箱（都**不要**复用其他上架包的）
- [ ] 截图与预览图（要先能跑起来）

## 目录

```
PocketLedger/
  App/            AppDelegate + MainTabBarController
  Gate/           AB 面网关（与 decktallypro 同构，仅 B 面 url 多一层校验）
  Push/           FCM token 注册（带 gateMode）
  Core/
    Models/       账户/分类/流水 + LedgerMath（余额与统计，纯函数、无 UIKit 依赖）
    Services/     LedgerStore（JSON 落盘）· MoneyFormatter · UserSettingsStore · TrackingService
    Theme/        AppTheme（浅色配色）
    UI/           共用视图 + 会跟着账本自动刷新的页面基类
    Extensions/   布局与提示条
  MVP/            Overview / Transactions / Accounts / Settings
  Assets.xcassets AppIcon + AccentColor
store/            可托管的隐私政策页（截图待补）
```

记账本体与网关**完全解耦**：`Core/Models`、`Core/Services/LedgerStore`、`MVP/` 里没有一处
引用 `Gate/`。
