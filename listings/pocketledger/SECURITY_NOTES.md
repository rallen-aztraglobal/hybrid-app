# Security Notes — PocketLedger

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
  但仍要**打开看一眼**再放进来，确认里面的 `BUNDLE_ID` 是 `com.stillwater.pocketledger`
  且没有别的包名。
  缺文件时的降级：`PushService.start` 对 `FirebaseApp.configure()` 做了
  `FirebaseApp.app() == nil` 的前置判断，配置缺失时推送整段 no-op，不影响 App 本体。
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
unzip -o PocketLedger.ipa -d unpacked >/dev/null
strings unpacked/Payload/PocketLedger.app/PocketLedger \
  | grep -oE '[a-z0-9.-]+\.(com|net|online|io|app|dev)' | sort -u
```

应当只看到 `GateConfig.apiBases` 里那个 API 域名，以及 Apple / Firebase /
AppsFlyer / Adjust 自带的文档与端点域名。**出现任何品牌站点域名就是漏了。**

顺便扫一遍不该出现的 token：

```bash
strings unpacked/Payload/PocketLedger.app/PocketLedger \
  | grep -E 'sn947o53ym80|bytg13h7yubk|2yhxl7paa3ls'
```

> 这几个是**其他上架包的 Adjust App Token**。本包的二进制里出现任何一个，
> 都说明有人复制粘贴的时候没改干净 —— 这不仅会把归因数据算到别的 App 上，
> 也是一条把两个包关联起来的硬证据。**必须没有任何输出。**

⚠️ **上面两条命令都还没跑过** —— 本包从未编译，还没有 `.ipa` 可扫。
第一次归档成功后**立刻补跑一遍**。

---

## 四、本地存储里有什么

`Core/Services/LedgerStore.swift` 把整个账本写成一个 JSON 文件：

- 位置：App 沙盒的 **Application Support** 目录下，文件名 `pocketledger.json`
- 内容：账户（名字、类型、期初余额、颜色、创建时间）、分类、全部流水
  （金额、日期、账户、分类、备注）
- 写法：`Data.write(to:options:.atomic)` —— 原子写，中途崩溃不会留下截断的账本

另外 `Core/Services/UserSettingsStore.swift` 在 UserDefaults 里存一个键
（`pkl.currencyCode`，值是三字母币种代码）。

**没有加密**，这是有意的：

- iOS 的沙盒机制保证其他 App 读不到这个文件；
- 文件默认受数据保护（`NSFileProtectionCompleteUntilFirstUserAuthentication`），
  设备锁屏加密已经覆盖了「设备丢失」这一威胁；
- 加密要么把密钥也存在同一台设备上（等于没加），要么要求用户设口令
  （一个离线记账 App 强上口令，多数人会直接卸载）。

> ⚠️ **但这也意味着：设备被解锁后取得物理访问权的人，通过备份或越狱是能读到账本的。**
> 如果以后要提升这一档，正确做法是接 **Keychain + 生物识别解锁**，
> 而不是自己发明一套加密。
>
> **不要往这两个 store 里加任何会离开设备的字段而不回来同步文档** —— 一旦账本内容
> 以任何形式进入请求体、归因事件或日志，`APP_PRIVACY.md` 的申报口径就变了
> （Financial Info 那一栏必须勾上，且很可能变成 Data Linked to You）。

CSV 导出（`MVP/Settings/SettingsModule.swift`）写到 **App 自己的临时目录**，
交给系统分享面板，由用户决定发给谁。我们既不接收也不经手。
> 小提醒：临时目录里那份 `pocketledger-export.csv` **不会自动清理**（同名覆盖，
> 系统在磁盘吃紧时才回收）。不是安全问题（仍在沙盒内），但值得知道。

---

## 五、申报与实现必须一致

`APP_PRIVACY.md` 与 `PRIVACY_POLICY.md` 是按**源码的实际内容**写的，取证点：

| 声明的行为 | 源码位置 |
| --- | --- |
| 账本只存本地、不上传 | `Core/Services/LedgerStore.swift` |
| 启动配置请求只带 platform / bundleId / timezone | `Gate/GateService.swift:evaluate` |
| FCM token + 系统版本上报 | `Push/PushService.swift:report` |
| AppsFlyer / Adjust 初始化与事件 | `Core/Services/TrackingService.swift` |
| 无定位权限 | `PocketLedger/Info.plist`（只有三个键，没有任何 `NSLocation*`） |
| 出口合规 = 无非豁免加密 | `Info.plist` 的 `ITSAppUsesNonExemptEncryption = false` |

**以下任一变动后必须回来同步 `APP_PRIVACY.md` 与 `PRIVACY_POLICY.md`（两份都要）：**

- 增删任何第三方 SDK
- 改 `GateService` / `PushService` 的请求 payload 字段
- 让账本数据以任何形式离开设备
- 填上 `appsFlyerAppleAppID` / `adjustAppToken`（这会把「申报了但没启用」变成真的启用）
- 加或去掉 ATT 请求

申报与实现不符是应用被下架的常见原因，不是理论风险。

---

## 六、两个已知的、与安全相关的待办

1. **ATT 状态不自洽。** `Info.plist` 里有 `NSUserTrackingUsageDescription`，
   但全工程 grep 不到 `ATTrackingManager` / `AppTrackingTransparency`。
   声明使用 IDFA 却不实现 ATT 是 Guideline 5.1.2 的直接驳回项。
   两条路线见 `APP_STORE_LISTING.md` 的「IDFA / ATT」一节，**必须挑一条并保持一致**。
2. **`aps-environment` 还是 `development`**（`PocketLedger/PocketLedger.entitlements`）。
   归档时通常会被自动替换成 `production`，但**要在 `.ipa` 里实际确认**，
   命令见 `RELEASE_SIGNING.md` 第 9 步。带着 development 上架的话推送在生产环境收不到。

---

## 七、当前验证状态

**一切都未验证。** 本工程从未编译、从未运行、没有 `.ipa`、没有证书、没有 team。
本文里所有 `unzip` / `strings` / `codesign` 的自查命令都**还没跑过**，
第一次归档成功后必须补做。这不是「大概没问题」，是「还没看过」。
