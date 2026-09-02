# GridSlide

原生 iOS 数字滑块拼图（Swift + UIKit），**仅 iOS**，作为上架包接入了 AB 面网关。

- bundleId `com.fernvale.gridslide`
- 商店名 `GridSlide`
- A 面 = 游戏本体（`MVP/` 三个模块），B 面 = 服务端下发的 web

> ⚠️ **本工程从未编译过。** 代码是在一台没有 Xcode 的 Windows 机器上写的。
> 接手第一件事是在 Mac 上跑通编译，详见 `README_GATE.md` 的「本地验证」。
>
> 不过**规则内核的算法是验证过的** —— 写进工程前先用等价的 Dart 实现跑了 19 项测试，
> 含「打乱产出的局面必定可解」的穷举验证。详见 `README_GATE.md`。

## 这个游戏做什么

数字滑块拼图（十五数码那一类）。把打乱的数字块推回 1、2、3…… 的顺序。

- **3×3 / 4×4 / 5×5** 三种尺寸，随时切换。
- 点击与空格同行或同列的块即可滑动，**支持整排滑动** —— 点隔着几格的块，
  中间的块一起挪，不必一格一格点。整排只算一步。
- 归位的块会变成强调色，一眼看得出还差哪几块。
- 记步数与用时，**两者各记各的最好成绩**，按尺寸分别保存。

纯离线，无广告、无内购、无账号、无排行榜。成绩存在设备本地。

## 从哪份文档开始看

| 想做的事 | 看这份 |
| --- | --- |
| 搞懂网关怎么接的、算法验到什么程度、**怎么在 Mac 上跑起来** | **`README_GATE.md`**（先看这份） |
| 证书、描述文件、App Store Connect | `RELEASE_SIGNING.md` |
| 填 App Privacy（隐私营养标签） | `APP_PRIVACY.md` |
| 上传商店文案 | `APP_STORE_LISTING.md` |
| 提交给审核员的备注与自查 | `APP_REVIEW_NOTES.md` |
| 截图 / 预览图规格与现状 | `STORE_ASSETS.md` |
| 隐私政策正文 | `PRIVACY_POLICY.md`（可托管版：`store/privacy-policy.html`） |
| 密钥与机密的边界 | `SECURITY_NOTES.md` |

## 当前状态

**已完成**：全部 Swift 源码、Xcode 工程（照 decktallypro 那份已知可用的 pbxproj 改写，
用 Xcode 16 的文件系统同步组）、应用图标（1024，24 位无 alpha）、LaunchScreen、
网关/推送/归因三层接入。规则内核的算法已用 Dart 等价实现验证。

**尚未完成**：

- [ ] **在 Mac 上编译并跑通**（从未编译过 —— 第一优先级）
- [ ] App Store Connect 建 App 条目，拿数字 App ID 填进 `GateConfig.appsFlyerAppleAppID`
- [x] Adjust 后台建 App，填 `adjustAppToken` 与 `adjustOpenBLandingToken`
- [x] Firebase 注册本包，放入 `GridSlide/GoogleService-Info.plist`
- [ ] 定 Apple Developer team 并配好签名（现在 `DEVELOPMENT_TEAM` 是空的）
- [ ] 渠道中台建 listing 条目（建好之前判定恒为 A 面，属预期的 fail-closed）
- [ ] 托管隐私政策、定支持邮箱（都**不要**复用其他上架包的）
- [ ] 截图与预览图（要先能跑起来）
- [ ] 启用归因前先补 ATT

## 目录

```
GridSlide/
  App/            AppDelegate + MainTabBarController
  Gate/           AB 面网关（与 decktallypro / pocketledger 同构）
  Push/           FCM token 注册（带 gateMode）
  Core/
    Models/       SlidePuzzle（规则内核，纯 Swift、无 UIKit 依赖）
    Services/     RecordStore · SettingsStore · TrackingService
    Theme/        AppTheme（深色配色）
    UI/           共用视图
    Extensions/   布局与提示条
  MVP/            Play / Records / Settings
  Assets.xcassets AppIcon + AccentColor
store/            可托管的隐私政策页（截图待补）
```

游戏本体与网关**完全解耦**：`Core/Models`、`Core/Services/RecordStore.swift`、`MVP/`
里没有一处引用 `Gate/`。
