# Release & Signing — GridSlide

> 本文是**操作手册**：从 Apple Developer 账号、证书、描述文件，到 App Store Connect
> 建条目、归档、上传、提交审核的完整步骤。
>
> ⚠️ **本文不包含、也不应包含任何密钥、口令、私钥、API Key 或 .p8 文件内容。**
> 边界见 `SECURITY_NOTES.md`。
>
> ⚠️ **下面每一步都还没执行过。** 本工程从未编译、从未运行，写代码的机器是 Windows，
> 没有 Xcode 也没有 Swift 工具链。整篇按「在一台装了 Xcode 的 Mac 上从零走一遍」写。

---

## 0. 前置：先让它编译过

签名之前，先确认代码本身是好的。不签名编译最快暴露语法与 API 问题：

```bash
cd listings/gridslide
xcodebuild -scheme GridSlide -project GridSlide.xcodeproj \
  -sdk iphonesimulator -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO build
```

跑不通就先修代码，别急着弄证书。手验路径（12 步）见 `README_GATE.md` 末节。

> 提醒一句本包特有的情况：`Core/Models/SlidePuzzle.swift` 的规则内核在写进工程之前
> 用等价的 Dart 实现跑过 19 项测试（含「打乱产出必定可解」），但**那验证的是算法，
> 不是这段 Swift**。逐行转写有没有笔误、UI 层对不对，只有这一步能告诉你。

---

## 1. Apple Developer 账号 —— **尚未确定，需要先拍板**

> **这是本包的第一个阻塞点，且是个业务决定，不是技术决定。**
>
> App Store 商品页上的 **Seller（卖家名称）是公开的**，取自 Apple Developer 账号的
> 法定名称。本包若与仓库里其余 iOS 上架包用**同一个** Apple Developer 账号，
> 几个 App 的 Seller 会**完全一致** —— 任何人点开商品页都能看出它们同属一家。
> 这与「各包使用互不相关的厂商命名空间（本包是 `fernvale`）」的初衷直接冲突。
>
> 两个选项：
>
> | | 复用现有账号 | 另开一个开发者账号 |
> | --- | --- | --- |
> | 成本 | 0 | 99 USD/年 + 一套主体资料 + 审核等待 |
> | Seller 公开一致 | **是** | 否 |
> | 上架速度 | 立刻 | 新账号首次上架通常更谨慎、更慢 |
>
> `project.pbxproj` 里的 `DEVELOPMENT_TEAM` 现在是**空字符串**，就是在等这个决定。
> 定下来之前，下面第 2 步往后都做不了。

确定后记下 **Team ID**（10 位字母数字，在 developer.apple.com → Membership 里）。
Team ID 本身不是机密（会出现在描述文件里），可以写进工程文件。

---

## 2. App ID（Identifier）

developer.apple.com → Certificates, Identifiers & Profiles → **Identifiers** → ＋

| 项 | 值 |
| --- | --- |
| Type | App IDs → App |
| Description | GridSlide |
| Bundle ID | **Explicit** → `com.fernvale.gridslide` |
| Capabilities | 勾 **Push Notifications** |

> - Bundle ID 必须是 **Explicit**，不能用通配符 —— 推送与 App Store 上架都要求。
> - 只勾 Push Notifications。本 App 不用 iCloud、不用 App Groups、不用 Game Center、
>   不用 Sign in with Apple，多勾的 capability 会让描述文件与 entitlements 对不上，
>   反而出签名错误。
> - Bundle ID 建好后**不能改**。拼错就只能换一个新的，并同步改
>   `project.pbxproj` 的 `PRODUCT_BUNDLE_IDENTIFIER`、`Gate/GateConfig.swift` 的
>   `bundleId`、Firebase 的 iOS App、Adjust 的 App、以及服务端 listing 条目。

---

## 3. 证书（Certificates）

需要两类，都在 Certificates, Identifiers & Profiles → **Certificates**：

| 用途 | 类型 |
| --- | --- |
| 上架构建签名 | **Apple Distribution** |
| 真机调试（可选） | **Apple Development** |

流程（**每一步都在自己机器上做，私钥不离开这台机器**）：

1. 钥匙串访问 → 证书助理 → **从证书颁发机构请求证书**，选「存储到磁盘」，
   生成 `.certSigningRequest`（CSR）。
2. 把 CSR 上传到 developer.apple.com，下载签好的 `.cer`。
3. 双击 `.cer` 导入钥匙串。此时钥匙串里是「证书 + 对应私钥」的一对。

> - **私钥只在生成 CSR 的那台机器上。** 换机器要从钥匙串导出 `.p12`（带口令）。
> - **`.p12` 与它的口令是机密，绝不进 git。** 见 `SECURITY_NOTES.md`。
> - Apple Distribution 证书一个团队最多 2 张（现行上限）。已经有两张时不要乱撤销 ——
>   撤销会让所有用它签的描述文件失效，包括别的 App 正在用的。**撤销前先确认没人在用。**

---

## 4. 描述文件（Provisioning Profile）

Certificates, Identifiers & Profiles → **Profiles** → ＋

| 项 | 值 |
| --- | --- |
| Type | **App Store Connect**（Distribution 下） |
| App ID | `com.fernvale.gridslide` |
| Certificate | 第 3 步的 Apple Distribution 证书 |
| Profile Name | 建议命名为 **`AppStore`** |

> 名字建议就叫 `AppStore`，因为第 5 步的手动签名配置里要按名字引用它
> （与仓库里其余上架包的做法一致）。

下载 `.mobileprovision` 并双击导入 Xcode。

---

## 5. Xcode 工程设置

### 5.1 填 team、切手动签名

`project.pbxproj` 现在是（Debug / Release 两份配置里都是）：

```
CODE_SIGN_STYLE = Automatic;
DEVELOPMENT_TEAM = "";
```

这样**模拟器能跑**（`CODE_SIGNING_ALLOWED=NO`），但**出不了可上架的包**。改成：

```
CODE_SIGN_STYLE = Manual;
DEVELOPMENT_TEAM = <你的 10 位 Team ID>;
CODE_SIGN_IDENTITY[sdk=iphoneos*] = "Apple Distribution";
PROVISIONING_PROFILE_SPECIFIER[sdk=iphoneos*] = AppStore;
```

> 建议在 Xcode 的 Signing & Capabilities 面板里改，让 Xcode 自己写回 pbxproj，
> 手改 pbxproj 容易漏掉 Debug/Release 其中一份配置（本工程确实是两份）。
>
> 为什么切 Manual：Automatic 会让 Xcode 在归档时自作主张地创建 / 更新描述文件，
> 多包共用一个账号时很容易互相踩。

### 5.2 勾 capability

Xcode → target GridSlide → **Signing & Capabilities** → ＋ Capability：

- **Push Notifications**
- **Background Modes** → 勾 **Remote notifications**

> `Info.plist` 里的 `UIBackgroundModes = remote-notification` 已经写好了，
> 但 capability 是另一件事（它写的是 entitlements 与 App ID），**两个都要有**。
>
> **不要**顺手勾 Game Center —— 本 App 没有排行榜也没有成就，勾了只会让
> entitlements 与描述文件对不上。

### 5.3 entitlements 检查

`GridSlide/GridSlide.entitlements` 现在是：

```xml
<key>aps-environment</key>
<string>development</string>
```

归档时 Xcode 通常会按 App Store 描述文件把它替换成 `production`，
但**要在导出的 `.ipa` 里实际确认一遍**，别假设（怎么确认见第 9 步）。

### 5.4 SPM 包

`project.pbxproj` 里已经声明好三个 `XCRemoteSwiftPackageReference`，
Xcode 打开工程会自动解析。若要手动加，File → Add Package Dependencies：

- AppsFlyer：`https://github.com/AppsFlyerSDK/AppsFlyerFramework`（产品 `AppsFlyerLib`，7.0.0+）
- Adjust：`https://github.com/adjust/ios_sdk`（产品 `AdjustSdk`，锁 5.4.0）
- Firebase：`https://github.com/firebase/firebase-ios-sdk`（产品 `FirebaseMessaging`，12.16.0+）

三者的调用都用 `#if canImport` 守卫，**未加包也能编译（走 no-op）**。
**即便加了包，本包当前也不会有任何归因上报** —— 两个 key 还是占位符，
`GateConfig.isConfigured()` 会把 SDK 拦在初始化之前（见第 11 节）。

---

## 6. App Store Connect 建 App 条目

appstoreconnect.apple.com → **我的 App** → ＋ → 新建 App

| 项 | 值 |
| --- | --- |
| 平台 | **iOS**（只勾这一个） |
| 名称 | `GridSlide`（见 `APP_STORE_LISTING.md`） |
| 主要语言 | English (U.S.) |
| 套装 ID | 选第 2 步建的 `com.fernvale.gridslide` |
| SKU | 内部标识，随便取一个不重复的，例如 `gridslide-ios-001`；不公开 |
| 用户访问权限 | 完全访问 |

> **建完之后立刻做一件事**：在 App 信息页记下 **Apple ID**（一串数字，形如 6780248860）。
> 这就是 `Gate/GateConfig.swift` 里 `appsFlyerAppleAppID` 要填的值 ——
> **填数字本身，不带 `id` 前缀**。填完 AppsFlyer 才会真正初始化。

---

## 7. 填商品页与合规信息

按各自的文档填：

| 内容 | 文档 |
| --- | --- |
| 名称 / 副标题 / 宣传文本 / 描述 / 关键词 / **类目（Games › Puzzle）** / URL | `APP_STORE_LISTING.md` |
| 年龄分级问卷（含三项博彩相关的 None/No） | `APP_STORE_LISTING.md` 的「年龄分级」一节 |
| App Privacy（隐私营养标签） | `APP_PRIVACY.md` |
| App Review Information（备注 + 两分钟演示路径） | `APP_REVIEW_NOTES.md` |
| 截图 | `STORE_ASSETS.md`（**目前一张都没有**） |
| 价格 | Free；App 内无购买项目 |
| Game Center | **不开启**（无排行榜、无成就） |

---

## 8. 推送（APNs Auth Key）

1. developer.apple.com → Keys → ＋，勾 **Apple Push Notifications service (APNs)**，
   下载 `.p8`。
2. 把 `.p8` 上传到 **Firebase 项目 `hybrid-listings-51660`** →
   项目设置 → Cloud Messaging → APNs Authentication Key，填上 Key ID 与 Team ID。
3. 在同一个 Firebase 项目下以 bundleId `com.fernvale.gridslide` 注册一个 iOS App，
   下载 `GoogleService-Info.plist` 放到 `GridSlide/` 下
   （file-system-synchronized group 会自动纳入 bundle）。

> ⚠️ **`.p8` 只能下载一次，且是机密。** 丢了只能撤销重建。
> **绝不进 git、绝不贴进聊天或工单。** 见 `SECURITY_NOTES.md`。
> 一个 APNs Key 可以覆盖同一个开发者账号下的所有 App，所以如果最终复用现有账号，
> 很可能已经有一把可用的了 —— **先问，别急着新建**（每个账号的 Key 数量有上限）。

---

## 9. 归档、导出、上传

```bash
cd listings/gridslide

# 归档
xcodebuild -scheme GridSlide -project GridSlide.xcodeproj \
  -sdk iphoneos -configuration Release \
  -archivePath build/GridSlide.xcarchive archive

# 导出 .ipa（需要一份 ExportOptions.plist，见下）
xcodebuild -exportArchive \
  -archivePath build/GridSlide.xcarchive \
  -exportOptionsPlist ExportOptions.plist \
  -exportPath build/export
```

`ExportOptions.plist` 的内容（**这份文件不含机密，但也没必要进 git，因为它带 Team ID**）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>method</key><string>app-store-connect</string>
  <key>teamID</key><string>你的 Team ID</string>
  <key>signingStyle</key><string>manual</string>
  <key>provisioningProfiles</key>
  <dict>
    <key>com.fernvale.gridslide</key><string>AppStore</string>
  </dict>
  <key>uploadSymbols</key><true/>
</dict>
</plist>
```

**导出后、上传前，先验一遍产物**：

```bash
cd build/export
unzip -o GridSlide.ipa -d unpacked >/dev/null

# 1. entitlements 里的 aps-environment 必须是 production
codesign -d --entitlements :- unpacked/Payload/GridSlide.app 2>/dev/null | grep -A1 aps-environment

# 2. 签名身份必须是 Apple Distribution
codesign -dvvv unpacked/Payload/GridSlide.app 2>&1 | grep Authority

# 3. 图标必须无 alpha 通道
sips -g hasAlpha unpacked/Payload/GridSlide.app/AppIcon*.png
```

再顺手跑一遍 `SECURITY_NOTES.md` 第三节那两条 `strings` 自查（域名 / 他包 Adjust token）。

上传（二选一）：

- **Xcode Organizer** → Distribute App → App Store Connect（最省事）
- 命令行：
  ```bash
  xcrun altool --upload-app -f build/export/GridSlide.ipa -t ios \
    --apiKey <Key ID> --apiIssuer <Issuer ID>
  ```
  > App Store Connect API Key（`.p8`）**是机密**，用环境变量或钥匙串传，
  > 不要写死在脚本里、不要进 git。

---

## 10. 提交审核

1. 构建版本上传后要等 10 分钟到几小时才会出现在 App Store Connect 的版本里。
2. 选中构建版本 → 回答 **出口合规**。
   > `Info.plist` 里已有 `ITSAppUsesNonExemptEncryption = false`，
   > 所以**这一问通常不会再出现**。若出现了，答案是 No（只用系统标准 HTTPS，属豁免项）。
3. 回答 **IDFA** 问卷 —— 见 `APP_STORE_LISTING.md` 的「IDFA / ATT」一节，
   **必须与代码里是否实现 ATT 请求一致**。当前代码没有 ATT 请求、
   两个归因 key 也还是占位符，照现状提交的答案是 **No**。
4. 检查 App Privacy 已填完（`APP_PRIVACY.md`）。
5. 提交前对一遍 `APP_REVIEW_NOTES.md` 的自查清单。
6. 提交。

---

## 11. 上架后要回填的两个 key

上架流程走完后，有两处配置才拿得到值。填完**需要重新出一个构建版本**：

| 配置 | 值从哪来 |
| --- | --- |
| `GateConfig.appsFlyerAppleAppID` | 第 6 步记下的 App Store Connect **Apple ID**（纯数字），现在是 `TODO_APPSTORE_APP_ID` |
| `GateConfig.adjustAppToken` | Adjust 后台为本包**新建**的 App 的 12 位 App Token，现在是 `TODO_ADJUST_APP_TOKEN` |

> Adjust 建 App 时：Platform 选 iOS、Store 选 App Store、
> bundleId 填 `com.fernvale.gridslide`、reporting currency 与现有 app 对齐用 **PHP**
> （**建后不可改**）。另外还要建一个名为 `OpenBLanding` 的 event（非 unique）
> 拿它的 token 填 `adjustOpenBLandingToken`。
>
> ⚠️ **不可复用其他包的 Adjust token** —— 复用会把本包的安装与会话归到别的 App 上。
> 具体不能用哪些见 `Gate/GateConfig.swift` 的注释。
>
> 填上这两个之后，AppsFlyer 与 Adjust 才会真正初始化 —— 也就是说
> **`APP_PRIVACY.md` 里申报的那些数据才真的开始被收集**。填之前请确认申报口径已就位
> （尤其 ATT 那条路线已经拍板并实现）。
