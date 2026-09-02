# Security Notes — GridSlide

> 本文是本包的内部安全备忘：哪些东西绝不能进仓库、哪些是非机密的、上架前对产物要做哪些自查。
> **面向内部，不要粘进 App Store Connect。**
>
> iOS 与 Android 的机密清单不一样：这里没有 keystore，取而代之的是
> **证书私钥（.p12）、描述文件、APNs .p8、App Store Connect API Key**。
> 不要照抄仓库里 Android 包的 `SECURITY_NOTES.md`。

---

## 一、绝不进 git 的东西

| 文件 / 内容 | 为什么 |
| --- | --- |
| `*.p12` / `*.cer` 对应的**私钥**（从钥匙串导出的分发证书） | 拿到它 + 一份描述文件就能以你的身份签名分发 App |
| `.p12` 的导出口令 | 同上；口令和文件分开也不行，都不进仓库 |
| **APNs Auth Key（`.p8`）** | 能给你所有 App 的所有设备发推送。**只能下载一次**，丢了只能撤销重建 |
| **App Store Connect API Key（`.p8`）+ Key ID + Issuer ID** | 能上传构建版本、改商品页、读销售数据 |
| `*.mobileprovision` | 本身不算高危（里面是公钥与授权列表），但含 Team ID、设备 UDID 列表与账号结构，**没必要进仓库** |
| `ExportOptions.plist` | 含 Team ID。不是机密，但同上，没必要 |
| Apple ID 账号口令 / 双重认证恢复密钥 | 不用解释 |
| 后台管理账号口令、Firebase service account 私钥、**Adjust 个人 API token** | 都是账号级凭据，与本包用的 App Token 完全不同 |

> **Adjust 的两种 token 别搞混：**
> - **App Token**（12 位，`GateConfig.adjustAppToken`）—— 随包分发，**非机密**。
>   本包现在还是占位符 `TODO_ADJUST_APP_TOKEN`。
> - **个人 API token** —— 能读整个账号的数据，**是机密**，不要写进代码、不要贴到聊天或工单里。

**当前状态**：以上东西本包一个都还没有（团队都还没定），所以仓库里现在是干净的。
但这也意味着**还没人验证过 `.gitignore` 会不会挡住它们** —— 第一次拿到证书前，
先确认仓库根的 `.gitignore` 覆盖了 `*.p12` / `*.p8` / `*.mobileprovision` / `*.certSigningRequest`。

提交前的机械检查（在仓库根跑）：

```bash
git status --porcelain | grep -iE '\.(p12|p8|cer|certSigningRequest|mobileprovision|keystore|jks)$'
```

有输出就是要出事了，**先处理再 commit**。已经 commit 进去的话，撤销证书 / 撤销 key
比清 git 历史更要紧 —— 历史清干净了，泄露出去的那份仍然有效。

---

## 二、非机密、可以进仓库的

- **`GoogleService-Info.plist`**（**尚未放入**）—— 随 IPA 分发，解包即可得，非机密。
  Firebase 控制台按 iOS App 单独下发这份文件，**天然只含本 App 的节点**，
  不像 Android 的 `google-services.json` 那样会把同项目下其余应用的包名一起带出来。
  但仍要**打开看一眼**再放进来，确认里面的 `BUNDLE_ID` 是 `com.fernvale.gridslide`
  且没有别的包名。
  缺文件时的降级：`PushService.start` 的第一行就是
  `guard Bundle.main.path(forResource: "GoogleService-Info", ofType: "plist") != nil else { return }`，
  配置缺失时推送整段 no-op，不影响 App 本体。
  （**注意这里不能改成用 `FirebaseApp.app() == nil` 判断** —— 未配置时它正好返回 nil，
  那样写等于把执行流送进 `configure()`，而它抛的是 Swift catch 不住的 ObjC 异常。）
- **`Gate/GateConfig.swift` 里的 AppsFlyer devKey**（`fXoKsKQwxPCRdhD8CD8q6F`，账号级）
  与 **Adjust App Token**（当前仍是占位符）—— 同样随包分发，非机密。
- **Team ID**（10 位）—— 会出现在描述文件与 `.ipa` 里，非机密。
- **App Store Connect 的 Apple ID**（App 的数字 id）—— 公开信息，商品页 URL 里就有。

---

## 三、客户端不含线上落地页地址

`Gate/GateConfig.swift` 是全工程唯一出现域名的地方，里面只有服务端 API 基址
（`apiBases`）。其余地址由服务端在响应里下发，**不落任何编译期常量**。

上架前的自查 —— **对 `.ipa` 产物做，不是对源码**：

```bash
cd build/export
unzip -o GridSlide.ipa -d unpacked >/dev/null
strings unpacked/Payload/GridSlide.app/GridSlide \
  | grep -oE '[a-z0-9.-]+\.(com|net|online|io|app|dev)' | sort -u
```

应当只看到 `GateConfig.apiBases` 里那个 API 域名，以及 Apple / Firebase /
AppsFlyer / Adjust 自带的文档与端点域名。**出现任何品牌站点域名就是漏了。**

顺便扫一遍不该出现的 token：

```bash
strings unpacked/Payload/GridSlide.app/GridSlide \
  | grep -E 'sn947o53ym80|bytg13h7yubk|2yhxl7paa3ls'
```

> 这几个是**其他上架包的 Adjust App Token**（见 `GateConfig.swift` 的注释）。
> 本包的二进制里出现任何一个，都说明有人复制粘贴的时候没改干净 ——
> 这不仅会把归因数据算到别的 App 上，也是一条把两个包关联起来的硬证据。
> **必须没有任何输出。**

⚠️ **上面两条命令都还没跑过** —— 本包从未编译，还没有 `.ipa` 可扫。
第一次归档成功后**立刻补跑一遍**。

---

## 四、本地存储里有什么

本包存在设备上的东西非常少，全在 `UserDefaults` 里，一共三个键：

| 键 | 内容 | 出处 |
| --- | --- | --- |
| `gsl.records` | 一段 JSON：`{"3": {...}, "4": {...}, "5": {...}}`，每个尺寸记 `bestMoves` / `bestSeconds` / `completions` | `Core/Services/RecordStore.swift` |
| `gsl.boardSize` | 默认棋盘尺寸（3 / 4 / 5） | `Core/Services/SettingsStore.swift` |
| `gsl.hapticsEnabled` | 震动反馈开关 | 同上 |

**没有任何个人信息、没有账号、没有可识别到人的东西**，就是三个数字加一个开关。
没有加密，也不需要加密：

- iOS 的沙盒机制保证其他 App 读不到；
- 内容本身不敏感 —— 泄露出去最多让人知道某台设备解 4×4 用了多少步；
- 给「最少步数」加密没有任何威胁模型上的意义，只会增加出错面。

> ⚠️ **红线**：不要往这两个 store 里加任何会离开设备的字段而不回来同步文档。
> 一旦成绩以任何形式进入请求体、归因事件参数或日志，`APP_PRIVACY.md` 的申报口径就变了
> （**User Content → Gameplay Content** 那一项必须勾上，且很可能变成 Data Linked to You）。
> 现在那一项是明确不勾的，理由写在 `APP_PRIVACY.md` 的「待定项：Gameplay Content」一节。

另外注意 `RecordStore` / `SettingsStore` 的 **`init` 里只读不写、不 post 任何 Notification**
（`persist()` 才发 `.recordsDidChange`）。这不是风格问题：`static let shared` 的惰性初始化
由 `swift_once` 保护，那把锁**不可重入** —— 在 init 里发通知，观察者回头访问 `shared`
就是同线程二次进入，libdispatch 直接崩，而且**只在全新安装的第一次冷启动出现**。
改这两个类时别把这条破坏掉。

---

## 五、申报与实现必须一致

`APP_PRIVACY.md` 与 `PRIVACY_POLICY.md` 是按**源码的实际内容**写的，取证点：

| 声明的行为 | 源码位置 |
| --- | --- |
| 成绩只存本地、不上传 | `Core/Services/RecordStore.swift`（UserDefaults `gsl.records`） |
| 设置只存本地 | `Core/Services/SettingsStore.swift` |
| 启动配置请求只带 platform / bundleId / timezone | `Gate/GateService.swift:evaluate` |
| FCM token + 系统版本上报 | `Push/PushService.swift:report` |
| AppsFlyer / Adjust 初始化与事件（当前被占位符拦下） | `Core/Services/TrackingService.swift` |
| 无定位权限 | `GridSlide/Info.plist`（只有三个键，没有任何 `NSLocation*`） |
| 无内购 / 无广告 SDK | 全工程没有 StoreKit，也没有任何广告 SDK |
| 出口合规 = 无非豁免加密 | `Info.plist` 的 `ITSAppUsesNonExemptEncryption = false` |

**以下任一变动后必须回来同步 `APP_PRIVACY.md` 与 `PRIVACY_POLICY.md`（两份都要）：**

- 增删任何第三方 SDK
- 改 `GateService` / `PushService` 的请求 payload 字段
- 让成绩或任何游戏数据以任何形式离开设备（云存档、排行榜、分享）
- 填上 `appsFlyerAppleAppID` / `adjustAppToken`（这会把「申报了但没启用」变成真的启用）
- 加或去掉 ATT 请求

申报与实现不符是应用被下架的常见原因，不是理论风险。

---

## 六、三个已知的、与合规/安全相关的待办

1. **ATT 状态不自洽（对外口径层面）。** `Info.plist` 里有
   `NSUserTrackingUsageDescription`，但全工程没有 `ATTrackingManager` /
   `AppTrackingTransparency`。当前构建本身是自洽的（两个归因 key 还是占位符，
   SDK 根本不初始化，也就用不到 IDFA），但**一旦按「上线后会启用」申报 IDFA
   却不实现 ATT，就是 Guideline 5.1.2 的直接驳回项**。
   两条路线见 `APP_STORE_LISTING.md` 的「IDFA / ATT」一节，**必须挑一条并保持一致**。
2. **`aps-environment` 还是 `development`**（`GridSlide/GridSlide.entitlements`）。
   归档时通常会被自动替换成 `production`，但**要在 `.ipa` 里实际确认**，
   命令见 `RELEASE_SIGNING.md` 第 9 步。带着 development 上架的话推送在生产环境收不到。
3. **`PushService.report` 的 `"model"` 字段发的是系统版本不是机型。**
   不是安全漏洞，但它决定了 `APP_PRIVACY.md` 里申报的是「操作系统版本」。
   修的时候**只改键名，不要顺手改成真发机型** —— 那会扩大申报范围。

---

## 七、当前验证状态

**一切都未验证。** 本工程从未编译、从未运行、没有 `.ipa`、没有证书、没有 team。
本文里所有 `unzip` / `strings` / `codesign` / `sips` 的自查命令都**还没跑过**，
第一次归档成功后必须补做。这不是「大概没问题」，是「还没看过」。

> 唯一有实测支撑的是 `Core/Models/SlidePuzzle.swift` 的**算法**：
> 它在写进本工程前用等价的 Dart 实现跑过 19 项测试（含「打乱产出必定可解」，
> 见 `README_GATE.md`）。但那验证的是算法，**不是这段 Swift 转写，也不是任何 UI 代码**。
