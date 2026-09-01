# App Store Connect — App Privacy（隐私营养标签）逐项答案

> 本文是 App Store Connect → 你的 App → **App Privacy** 表单的逐项答案，照填即可。
>
> ⚠️ **这是 Apple 的表单，不是 Google Play 的 Data safety。** 两者的分组方式完全不同：
> Apple 先问「是否用于追踪（Data Used to Track You）」，再把其余数据按
> **Data Linked to You / Data Not Linked to You** 分两栏，每一项还要选
> **Purposes（用途）**。不要照抄仓库里 Android 包的 `DATA_SAFETY.md` 字段名。
>
> 依据是本包**源码的实际内容**（`Gate/GateConfig.swift`、`Gate/GateService.swift`、
> `Push/PushService.swift`、`Core/Services/TrackingService.swift`、
> `Core/Services/LedgerStore.swift`、`Core/Services/UserSettingsStore.swift`），
> 不是照抄其余上架包。**改依赖或改这些文件后必须回来同步。**

---

## 先说清当前实现状态（内部口径，不进表单）

> 申报口径与当前构建的实际行为**不一致**，这是有意为之，理由见下。

| SDK | 是否打进包 | 当前是否真的上报 |
| --- | --- | --- |
| AppsFlyer | `project.pbxproj` 已声明 SPM 包，代码用 `#if canImport(AppsFlyerLib)` 守卫 | **否** —— `GateConfig.appsFlyerAppleAppID` 仍是 `TODO_APPSTORE_APP_ID`，`isConfigured()` 把它拦在 `initialize` 之前 |
| Adjust | 同上（`AdjustSdk`） | **否** —— `GateConfig.adjustAppToken` 仍是 `TODO_ADJUST_APP_TOKEN`，同样被 `isConfigured()` 拦下 |
| Firebase Cloud Messaging | 同上（`FirebaseMessaging`） | **看情况** —— `GoogleService-Info.plist` 尚未放入，缺文件时 `PushService.start` 整段 no-op；plist 一放进来就会取 token 并上报 |
| 启动配置请求 | 无 SDK，系统 `URLSession` | **是** —— `GateService.evaluate` 每次启动都会发 |

**申报按「上线后会启用」的口径写**，即 AppsFlyer / Adjust / FCM 三者都按会收集申报。
理由与其余上架包一致：SDK 已经打进包里，填上 key 就开始上报；
**申报宽于实现是安全方向，反过来（实现宽于申报）才是下架风险。**

一个例外见文末「ATT 与 Data Used to Track You」—— 那一项如果走「暂不启用归因」路线，
必须把申报一并收窄，不能只改一半。

---

## 第一个问题

> **Do you or your third-party partners collect data from this app?**

**Yes**（Yes, we collect data from this app）。

> 不能答 No：即便账本本身完全不出设备，启动时的配置请求、FCM token 注册、
> 以及上线后启用的两个归因 SDK 都会把数据发出去。

---

## Data Used to Track You（用于追踪你的数据）

> Apple 对 "tracking" 的定义：把本 App 收集的用户或设备数据，与**其他公司**的
> App、网站或线下数据关联，用于定向广告或广告投放效果衡量；或把数据交给数据经纪商。
> AppsFlyer / Adjust 的归因正落在这个定义里。

**答案：Yes**（前提是走 `APP_STORE_LISTING.md` 里的「路线 A」，见文末）。

| 类目 | 具体数据类型 | 说明 |
| --- | --- | --- |
| **Identifiers** | Device ID | IDFA（获得 ATT 授权后）与 IDFV，由 AppsFlyer / Adjust 采集 |
| **Usage Data** | Product Interaction | app 打开与会话事件、归因自定义事件（`af_content_view` / `OpenBLanding`） |
| **Usage Data** | Advertising Data | 归因结果本身（这次安装来自哪一次广告曝光） |

> 勾了 Data Used to Track You 之后，Apple 会在商品页上显示
> "Data Used to Track You" 一栏，并且**要求 App 实现 ATT 授权请求**。
> 本包目前**没有实现 ATT 请求**，见文末，提交前必须处理。

---

## Data Linked to You（与你的身份关联的数据）

**答案：无 —— 一项都不勾。**

理由：本 App **没有账号、没有登录、不收集姓名 / 邮箱 / 电话 / 地址**，
不存在任何可以把数据挂到「一个人」身上的标识。所有出设备的数据都只与
**这台设备的安装**相关联，因此全部落在下一栏。

> 这一栏空着是本包的实际情况，不是偷懒。如果以后加了账号体系、
> 或者把用户填的任何个人信息随请求发出去，就必须重新分栏。

---

## Data Not Linked to You（不与你的身份关联的数据）

| 类目 | 具体数据类型 | Purposes（用途） | 来源 |
| --- | --- | --- | --- |
| **Identifiers** | Device ID | Analytics · Developer's Advertising or Marketing · App Functionality | AppsFlyer / Adjust 的设备标识；Firebase Cloud Messaging 的注册 token（用于把推送投递到这台设备） |
| **Usage Data** | Product Interaction | Analytics · Developer's Advertising or Marketing | app 打开与会话事件、归因自定义事件 |
| **Usage Data** | Advertising Data | Developer's Advertising or Marketing | 归因结果 |
| **Diagnostics** | Other Diagnostic Data | App Functionality · Analytics | 随 token 注册上报的**操作系统版本**（`PushService.report` 的 payload） |
| **Other Data** | Other Data Types | App Functionality | 启动配置请求里的**设备时区名**（IANA 格式，如 `Asia/Manila`）与应用标识 |

> 逐项取证：
>
> - **Device ID**：`Core/Services/TrackingService.swift` 初始化 AppsFlyer / Adjust；
>   `Push/PushService.swift` 取 FCM token 后 `POST /api/app/listing/register-token`。
> - **Other Diagnostic Data**：`PushService.report` 的 payload 里有一个键名叫
>   `"model"`、值却是 `UIDevice.current.systemVersion` 的字段 —— **实际发出去的是
>   操作系统版本，不是设备型号**。键名与内容不符（这是个应当修掉的小 bug，见
>   `APP_REVIEW_NOTES.md` 的自查清单），但**申报要按实际发出的内容写**，所以这里
>   申报的是操作系统版本。修键名时不要顺手改成真发型号，那会扩大申报范围。
> - **Other Data Types**：`Gate/GateService.swift:evaluate` 的 payload 只有三个字段：
>   `platform` / `bundleId` / `timezone`。前两个不是用户数据，时区是。
>
> **明确不勾的**（App 均不收集）：Contact Info（Name / Email / Phone / Address）·
> Health & Fitness · Financial Info · Location（见下）· Sensitive Info · Contacts ·
> User Content（Photos, Videos, Audio, Gameplay Content, Customer Support, Emails or
> Text Messages, Other User Content）· Browsing History · Search History ·
> Purchases · Payment Info · Credit Info · Diagnostics → Crash Data / Performance Data。

---

## 待定项：Financial Info

**不申报。** 明确记一下理由，免得下次有人纠结：

Apple 的 **Financial Info** 指 Payment Info（支付卡号等）、Credit Info（信用评分）、
Other Financial Info（薪资、收入、资产、负债等**收集上来的**财务信息）。

用户在 PocketLedger 里输入的金额确实是财务信息，但它**从不离开设备** ——
存在 App 沙盒 Application Support 目录下的 `pocketledger.json`
（`Core/Services/LedgerStore.swift`），不上传、不同步、不进任何请求体、
不作为归因事件参数，卸载即随 App 数据一并删除。

**Apple 的 App Privacy 只要求申报「被收集（collect）」的数据**，其定义是
「把数据从设备上传出去（transmit off the device）并留存超过处理该次请求所必需的时间」。
纯本地数据不在申报范围内 → **这一栏不勾任何东西。**

> 但**隐私政策里必须如实写出来**（已写，见 `PRIVACY_POLICY.md` 的
> "Stored only on your device"）。商品页上说「什么都不收集」而 App 里明明在记钱，
> 是没必要给自己留的解释成本。
>
> ⚠️ **红线**：一旦有任何代码把账本内容（金额、账户名、备注、CSV）发到服务器或第三方，
> 这一栏就必须改成 **Financial Info → Other Financial Info**，且很可能变成
> **Data Linked to You**。改 `LedgerStore` 或新增任何上传路径时，回来重看这一段。
>
> CSV 导出（`SettingsModule.exportCSV`）不改变这个结论：文件写到 App 自己的临时目录，
> 由**用户主动**通过系统分享面板决定发给谁，我们既不接收也不经手。

---

## 待定项：Location（Coarse Location）—— **必须在提交前定死**

App **不申请任何定位权限**（`Info.plist` 里没有 `NSLocationWhenInUseUsageDescription`
之类的键，代码里也没有 CoreLocation），读不到 GPS。

但启动配置请求会到达我们自己的服务器，服务器**必然能看到请求的来源 IP**
（任何 HTTP 请求都如此），而 IP 可以推出国家级的粗略位置。Apple 对
「由 IP 推得的粗略位置」是否需要申报为 Coarse Location，一直有解释空间。

- **保守口径**：申报 `Location → Coarse Location`，Data Not Linked to You，
  Purposes = App Functionality（内容写「服务端从请求来源 IP 得到的国家级粗略位置」）。
- **宽松口径**：不申报，理由是 App 自身不采集位置、IP 只是网络传输的必然产物。

> **这是与其余上架包共用的待定项，几个包必须取同一口径**，不要一个申报一个不申报 ——
> 口径不一致本身就是个会被注意到的信号。上架前定下来，把结论写死在这里，别留着。

---

## 其余三个总体问题

| 问题 | 答案 | 说明 |
| --- | --- | --- |
| 数据传输是否加密？ | **Yes** | 全部请求走 HTTPS（`GateConfig.apiBases` 只有 `https://` 基址；FCM 与两个归因 SDK 自身也是 HTTPS） |
| 是否提供删除数据的途径？ | **Yes** | 隐私政策里的联系邮箱可请求删除；本地账本用设置页的 "Erase all data" 或卸载即可清除 |
| 是否所有收集的数据都属于可选披露豁免？ | **No** | 本包不适用任何豁免（豁免仅限于「用户主动提交且明确可见、不用于追踪、不与身份关联」等窄情形） |

---

## ATT 与 Data Used to Track You —— 提交前必须拍板

**现状**：`Info.plist` 里有 `NSUserTrackingUsageDescription`，
但全工程 grep 不到 `ATTrackingManager` / `AppTrackingTransparency`
（`Gate/GateCoordinator.swift` 只调了 `TrackingService.shared.start()`）。
也就是说 **App 目前不会弹 ATT 授权框**。

`APP_STORE_LISTING.md` 的「IDFA / ATT」一节给了两条路线，**本文的申报必须跟着走同一条**：

| | 路线 A（补上 ATT） | 路线 B（暂不启用归因） |
| --- | --- | --- |
| 代码 | 加 `ATTrackingManager.requestTrackingAuthorization` | 保持不变，`TODO_*` 占位符不填 |
| Data Used to Track You | **Yes**（按上文那三行勾） | **No**（整栏清空） |
| Data Not Linked to You | 按上表全填 | 去掉 AppsFlyer / Adjust 带来的 Advertising Data 与相应的 Analytics 用途；FCM token 与配置请求的部分保留 |
| 上传构建版本时的 IDFA 问卷 | Yes + 勾 ATT 确认项 | No |

> **不要出现「plist 有 ATT 文案 + 申报 Yes + 代码不弹框」这种三方打架的状态** ——
> 声明使用 IDFA 却不实现 ATT，是 Guideline 5.1.2 的直接驳回项。
>
> 走路线 B 上线、之后再补归因，是完全可以的：填 key + 加 ATT + 更新本表 + 更新隐私政策，
> 一起随下一个版本提交即可。**先上线再悄悄打开归因而不更新申报，才是问题。**

---

## 申报与实现必须一致

本文与 `PRIVACY_POLICY.md` 是按源码实际内容写的。以下任一变动后**必须回来同步这两份**：

- 增删任何第三方 SDK
- 改 `Gate/GateService.swift` 或 `Push/PushService.swift` 的请求 payload 字段
- 让账本数据以任何形式离开设备
- 填上 `appsFlyerAppleAppID` / `adjustAppToken`（这会把「申报了但没启用」变成真的启用）
- 加 ATT 请求

申报与实现不符是应用被下架的常见原因，不是理论风险。
