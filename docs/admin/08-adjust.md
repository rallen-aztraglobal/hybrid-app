# 08 · Adjust 归因集成

> 需求：给部分渠道包接入 Adjust 归因。**只有在 Console 给渠道包绑定了 Adjust App Token 的包，打包时才集成并发事件；未绑定的包不集成、不发任何 Adjust 事件。** Adjust 账号**无 Automation**（不能程序化建 app/事件），只能面板手工建好后导出。决策见 [ADR-0013](../adr/0013-adjust-attribution.md)。

## 一句话方案

```
Console 渠道编辑页 · Adjust 区块
   填 App Token  +  上传事件 CSV(token,name,unique)
        │
        ▼
   Go 后端: channel.adjust_app_token / adjust_events(可空=未绑定)
        │
        ▼  CLI pull/build 渲染(只写有 token 的渠道)
   app/adjust-tokens.json  { applicationId: { appToken, events{name:token} } }
        │
        ▼  build.gradle 自包含旁路块(applicationVariants.all, 按 appId 注入)
   BuildConfig.ADJUST_APP_TOKEN / ADJUST_EVENT_MAP   (未绑定→空)
        │
        ▼  运行时
   AdjustBootstrap: token 空 → 全程 no-op(不 init/不发事件)
                    token 非空 → Adjust.initSdk + sendAFEvent 同源 fan-out
```

**核心心智：与 FCM（[07](./07-push.md) / [ADR-0012](../adr/0012-push-fcm.md)）完全同构的 feature gate**——依赖恒在 classpath，靠独立数据文件 + 按 flavor 探测决定「集成/跳过」，运行时探不到就 no-op，零改码激活。

## 1. 与 FCM 的对照（照抄成熟模式）

| 维度 | FCM（已上线） | Adjust（本方案） |
| --- | --- | --- |
| 开关数据源 | `app/google-services.json` | `app/adjust-tokens.json`（CLI 渲染） |
| 绑定键 | applicationId → client | **applicationId → appToken**（ADR-0009） |
| flavor 粒度开关 | `applicationVariants.all` 禁 `processXxxGoogleServices` | `applicationVariants.all` 注入 `ADJUST_APP_TOKEN` |
| 依赖 | firebase-messaging 无条件在 classpath | adjust-android 无条件在 classpath |
| 运行时门控 | `PushBootstrap.guard()` 探 `FirebaseApp` | `AdjustBootstrap.enabled`（token 是否为空） |
| 未配置行为 | onNewToken/onMessage no-op | init/trackEvent 全 no-op |
| 激活方式 | 丢 json + 重打包，零改码 | 绑 token + 重打包，零改码 |

## 2. 数据模型（复用现有 channel 表，加 2 个可空字段）

```sql
ALTER TABLE channel
  ADD COLUMN adjust_app_token VARCHAR(64) NULL,   -- 空/NULL = 未绑定 = 不集成
  ADD COLUMN adjust_events    JSON        NULL;    -- {"Login":"wzb3fb","Purchase":"gyuu2f",...}
```

- `adjust_app_token`：Adjust 面板「App Token」，一个 app 一个。空即该渠道不启用 Adjust。
- `adjust_events`：由上传的 CSV 解析成 `{name: token}`（`unique` 列丢弃，那是面板侧去重设置，SDK 调用不需要）。

## 3. 渲染产物 `app/adjust-tokens.json`

CLI `pull`/`build` 时，把**有 App Token 的渠道**渲染成（键=applicationId，守 ADR-0009）：

```json
{
  "com.gamezone.gpgzmkk042": {
    "appToken": "abc123xyz",
    "events": {
      "AddToCart": "jrilpe",
      "CompleteRegistration": "ny18sp",
      "Login": "wzb3fb",
      "OldRegPurchase": "4y3tr5",
      "Purchase": "gyuu2f",
      "TPFirstDeposit": "975l72"
    }
  }
}
```

- 未绑定渠道**不写入**（或写空 appToken）→ 该 flavor 运行时休眠。
- 入 `.gitignore`（同 `google-services.json`，由 CLI 渲染，不进 git）。App Token 非机密（随 APK 分发），故可入 DB/API/CLI。

## 4. App 层改动（android-kotlin，只动 `app/`）

### 4.1 依赖（无条件加入，同 firebase-messaging）

`gradle/libs.versions.toml` + `app/build.gradle`：

```groovy
implementation libs.adjust.android      // com.adjust.sdk:adjust-android:5.x
implementation libs.installreferrer     // com.android.installreferrer:2.2（Play 安装归因）
// GAID 已有：libs.gms.play.services.ads.identifier ✅（Adjust 采集广告 ID 复用）
```

> 为何不像 HMS/OAID 那样按 flavor 物理排除？`src/main` 共享代码要引用 `com.adjust.sdk.*`，某些 flavor 无此依赖会**编译不过**。FCM 也因此选「依赖恒在、运行时休眠」。SDK 小，休眠即等价「不集成」（零初始化、零流量）。见 [ADR-0013 备选](../adr/0013-adjust-attribution.md)。

### 4.2 build.gradle 门控块（紧跟现有 FCM 块之后，复刻其结构）

```groovy
// Adjust 按 flavor 自动开关（对齐上面 FCM 的 feature gate）：
//   adjust-tokens.json 里该 applicationId 有非空 appToken → 注入 BuildConfig，运行时真集成、发事件；
//   不在（未在 Console 绑定 Adjust）→ token 留空，运行时 AdjustBootstrap 探测为空后全程 no-op。
//   依赖恒在 classpath（同 firebase-messaging），仅靠 token 是否为空决定「集成/跳过」，零改码激活。
if (file("adjust-tokens.json").exists()) {
    def adjustCfg = new groovy.json.JsonSlurper().parse(file("adjust-tokens.json"))
    android.applicationVariants.all { variant ->
        def entry = adjustCfg[variant.applicationId]
        if (entry?.appToken) {
            def events = groovy.json.JsonOutput.toJson(entry.events ?: [:])
                    .replace('\\', '\\\\').replace('"', '\\"')
            variant.buildConfigField "String", "ADJUST_APP_TOKEN", "\"${entry.appToken}\""
            variant.buildConfigField "String", "ADJUST_EVENT_MAP", "\"${events}\""
        }
    }
}
```

`defaultConfig` 里给兜底默认，保证字段对所有 flavor 都存在（编译安全；变体级会覆盖默认级）：

```groovy
buildConfigField "String", "ADJUST_APP_TOKEN", "\"\""
buildConfigField "String", "ADJUST_EVENT_MAP", "\"{}\""
```

> 这是一个自包含旁路块，**没动 `loadChannels`/`productFlavors` 一行**——与 FCM 块同样的护栏姿势（ADR-0004）。

### 4.3 新增 `app/src/main/java/com/hybrid/android/track/AdjustBootstrap.kt`（镜像 `push/PushBootstrap.kt`）

```kotlin
object AdjustBootstrap {
    private const val TAG = "HybridAdjust"

    /** 是否已在后台绑定 Adjust（App Token 非空）。空 → 全程 no-op。 */
    val enabled: Boolean get() = BuildConfig.ADJUST_APP_TOKEN.isNotBlank()

    /** App 内部事件名 → Adjust 事件 name 的固定适配表（见 §4.5）。 */
    private val LOGICAL_TO_ADJUST_NAME = mapOf(
        "af_login" to "Login",
        "af_complete_registration" to "CompleteRegistration",
        // 其余同名：Purchase / OldRegPurchase / TPFirstDeposit / AddToCart / Install
    )

    /** Adjust 事件 name → token（编译期从 ADJUST_EVENT_MAP 解析）。 */
    private val eventTokens: Map<String, String> by lazy { parseEventMap() }

    @Volatile private var initialized = false

    /** 在 WebViewActivity.onCreate 尽早调用。未绑定 → no-op；幂等。 */
    fun init(context: Context) {
        if (!enabled) { Log.d(TAG, "未绑定 Adjust App Token，跳过初始化"); return }
        if (initialized) return
        val env = if (BuildConfig.ENABLE_TEST_EVENTS) AdjustConfig.ENVIRONMENT_SANDBOX
                  else AdjustConfig.ENVIRONMENT_PRODUCTION
        Adjust.initSdk(AdjustConfig(context.applicationContext, BuildConfig.ADJUST_APP_TOKEN, env))
        initialized = true
    }

    /** 发一个逻辑事件。未绑定 / 该事件未配 token → no-op。 */
    fun trackEvent(logicalName: String, params: Map<String, Any> = emptyMap()) {
        if (!enabled) return
        val adjustName = LOGICAL_TO_ADJUST_NAME[logicalName] ?: logicalName
        val token = eventTokens[adjustName] ?: run {
            Log.d(TAG, "事件 $logicalName → $adjustName 未配 token，跳过"); return
        }
        Adjust.trackEvent(AdjustEvent(token).apply {
            params.forEach { (k, v) -> addCallbackParameter(k, v.toString()) }
        })
    }

    private fun parseEventMap(): Map<String, String> = runCatching {
        val json = JSONObject(BuildConfig.ADJUST_EVENT_MAP)
        buildMap { json.keys().forEach { put(it, json.getString(it)) } }
    }.getOrDefault(emptyMap())
}
```

- 环境用现有 `ENABLE_TEST_EVENTS` 开关：测试包 → SANDBOX，生产包 → PRODUCTION（复用，不新增开关）。
- init 位置：`WebViewActivity.onCreate`（单 Activity 应用，与现有 AppsFlyer 一致，不新增 Application 类）。Adjust v5 自动注册生命周期回调，会话追踪无需手写 onResume/onPause。

### 4.4 事件复用现有 seam（改动极小）

[WebViewActivity.kt](../../app/src/main/java/com/hybrid/android/WebViewActivity.kt)：

```kotlin
// onCreate 内，strategy.initTracking(this) 旁：
AdjustBootstrap.init(this)   // 未绑定则 no-op

// sendAFEvent 开头加一行 fan-out，原 AppsFlyer 逻辑一字不动：
override fun sendAFEvent(eventName: String) {
    AdjustBootstrap.trackEvent(eventName, eventValues)   // 与 AppsFlyer 同源事件；未绑定 no-op
    AppsFlyerLib.getInstance().logEvent(/* ...原逻辑不动... */)
}
```

`StandardStrategy` / `BpStrategy` **完全不用改**——它们现有的 `host.sendAFEvent("Purchase")` 等会自动同时打到 AppsFlyer + Adjust。

### 4.5 关键接缝：事件名适配表

App 现在 `sendAFEvent()` 实际派发的字符串，与 Adjust 事件 CSV 的 `name` 有 2 处对不齐（AppsFlyer 常量）：

| App 内部派发字符串 | Adjust CSV name | 处理 |
| --- | --- | --- |
| `AddToCart` / `Purchase` / `OldRegPurchase` / `TPFirstDeposit` | 同名 | 直接命中 |
| `af_login`（=`AFInAppEventType.LOGIN`） | `Login` | 适配表转换 |
| `af_complete_registration` | `CompleteRegistration` | 适配表转换 |
| `Install` | （CSV 无） | Adjust SDK 自动归因 install，忽略 |

适配表是 **App 内一次性代码常量**（§4.3 `LOGICAL_TO_ADJUST_NAME`），**不是每渠道要填的数据**。因此后台永远只有两件事：填 App Token、传 CSV。CLI 只把 CSV 原样解析成 `{name: token}`，不感知 App 内部命名——干净分层。

> 若将来在 Adjust 新建了 App 未知的事件 name，它不会被触发（App 只发这 7 个逻辑事件），无副作用。

### 4.6 ProGuard

`minifyEnabled true` 已开，`app/proguard-rules.pro` 补：

```proguard
-keep class com.adjust.sdk.** { *; }
-keep class com.google.android.gms.common.ConnectionResult { int SUCCESS; }
-keep class com.google.android.gms.ads.identifier.AdvertisingIdClient { *; }
-keep class com.google.android.gms.ads.identifier.AdvertisingIdClient$Info { *; }
-keep public class com.android.installreferrer.** { *; }
```

## 5. CLI 层（cli-go，只动 `cli/`）

`pull`/`build` 渲染 `app/adjust-tokens.json`——与现在渲染 `google-services.json` / `bootstrap.json` 同一处逻辑：

1. 拉后台配置，取每个渠道的 `adjust_app_token` + `adjust_events`。
2. 只对有 token 的渠道写入 `{applicationId: {appToken, events}}`。
3. 把文件写到 `app/adjust-tokens.json`；确保 `.gitignore` 含该文件。

上传 CSV 的解析（后台侧，非 CLI）：`token,name,unique` → `{name: token}`，`unique` 丢弃。

## 6. Web 控制台（frontend-react，只动 `web/`）

渠道编辑页加「**Adjust**」区块，两个控件即可（对齐「有这两个就够」）：

- **App Token** 输入框（空 = 不启用 Adjust）。
- **事件列表**：上传 Adjust 导出的 CSV（`token,name,unique`），前端解析预览成表格（name / token），保存为 `adjust_events`。可显示友好名，但存的 key 用 CSV 的 `name`（`Login` / `CompleteRegistration` / ...）。

未填 App Token 的渠道 = 未绑定 = 打包时跳过 Adjust。

## 7. 「无 Automation」的影响

Adjust Automation 用于**程序化建 app / 建事件 / 拿 token**。没有它 → 运营在 Adjust 面板手工：建 app 拿 App Token、建各事件、导出事件 CSV → 贴进控制台。本方案**纯客户端 SDK 埋点，不依赖任何 Adjust 自动化 API**，故无 Automation 不影响落地，仅绑定环节是人工录入（这正是把录入做进 Web 控制台的原因）。每个 Adjust app 的事件 token 各不相同，故需逐渠道上传 CSV，属固有代价。

## 8. 验收 / 自测

- **未绑定包**：`assembleDebug` 不放 `adjust-tokens.json` → `ADJUST_APP_TOKEN=""` → 冷启动日志见「未绑定，跳过初始化」，抓包无 Adjust 请求。
- **绑定包**：放一份含测试 appToken + CSV 的 `adjust-tokens.json`，`-PtestEvents=true` 打 SANDBOX 包 → Adjust 面板 Testing Console 见 install/session + 触发登录/充值后见对应事件。
- **AppsFlyer 回归**：确认原有 6 类事件在 AppsFlyer 侧不受影响（fan-out 只增不改）。

## 9. 落地清单（分目录，多代理各动其目录）

| 目录 | 事项 |
| --- | --- |
| `app/` | 加依赖 + build.gradle 门控块 + `AdjustBootstrap.kt` + `sendAFEvent` fan-out + `AdjustBootstrap.init` + ProGuard |
| `cli/` | `pull/build` 渲染 `adjust-tokens.json` + `.gitignore` |
| `server/` | `channel` 加 `adjust_app_token`/`adjust_events`（migration）；CRUD + CLI 配置接口带出这两字段；CSV 解析 |
| `web/` | 渠道编辑页「Adjust」区块：App Token + 上传 CSV |

---

## 10. 华为商店包（`_hw`）：设备标识与安装来源

> 补于 2026-07-30。现象：`_hw` 包的 Adjust / AppsFlyer 事件在后台看不到。

### 10.1 为什么华为包特殊

华为设备**没有 GMS**，拿不到 GAID。而 `.hw` 包恰恰主要装在华为设备上——两个 SDK 都没有任何广告标识可用，事件即使发出去也落不到具体设备/用户身上，后台表现就是「事件丢了」。华为设备上唯一可用的广告标识是 **OAID**，安装来源也只能走 **AppGallery referrer**（Play Install Referrer 取不到）。

关键前提：**商店维度是跨品牌的**（[ADR-0009](../adr/0009-channel-identity-appid.md) 商店后缀）。`_hw` 可能挂在任何品牌下，所以不能只靠 `brandConfig[brand].hms` 这个品牌级开关——那是 BP 的历史遗留，只覆盖 BP。

### 10.2 依赖（4 个，注入范围不同）

| 依赖 | 作用 | 注入范围 |
| --- | --- | --- |
| `com.adjust.sdk:adjust-android-oaid` | Adjust 读 OAID | **全部 flavor**（无条件） |
| `com.adjust.sdk:adjust-android-huawei-referrer` | AppGallery 安装来源 | **全部 flavor**（无条件） |
| `com.appsflyer:oaid` | AppsFlyer 读 OAID（内含 MSA 移动安全联盟 SDK） | BP 品牌 + 所有 `_hw` |
| `com.huawei.hms:ads-identifier` | HMS 侧 OAID provider | 同上 |

两个 Adjust 插件的版本**必须与 `adjust-android` 主 SDK 同版本号**（用同一个 `version.ref = "adjustAndroid"`）。

为何前两个无条件、后两个按 flavor：Adjust 插件各十几 KB，且 `AdjustBootstrap`（`src/main` 共享代码）要直接引用 `AdjustOaid`——按 flavor 物理排除会让没有该依赖的 flavor **编译不过**，与 §4.1 是同一个理由。`com.appsflyer:oaid` 体积大得多，而 AF 侧的调用点 `setCollectOaid(true)` 属于主 SDK，未注入 OAID 库时调用是安全的 no-op，因此能按 flavor 排除、不必给全部 90 个包铺开。

### 10.3 代码开关（两个要显式开，一个默认开）

| 开关 | 默认 | 要求 |
| --- | --- | --- |
| `AdjustOaid.readOaid(context)` | `isOaidToBeRead = false`，**默认关** | 必须在 `Adjust.initSdk` **之前**调用 |
| `AppsFlyerLib.setCollectOaid(true)` | 默认关 | 必须在 `AppsFlyerLib.init` **之前**调用，**每个 `BrandStrategy` 实现都要有** |
| `AdjustHuaweiReferrer` | `shouldReadHuaweiReferrer = true`，**默认开** | 加依赖即生效，不需要调用 |

`setCollectOaid` 原来只写在 `BpStrategy` 里（历史上 OAID 被当成 BP 的品牌差异），`StandardStrategy`（AP/GP）漏了——AP/GP 的华为包因此彻底没有设备标识。现在两个实现都要开，`BrandStrategy.initTracking` 的接口注释也已同步：OAID 采集是**设备差异**，不是品牌差异。

两处调用都是**无条件**的，不按包名分支：非华为设备上插件内部探测不到 HMS / MSA SDK 会静默跳过，不影响初始化。

### 10.4 Manifest（最容易漏的一处）

Huawei referrer 插件读的是 ContentProvider `content://com.huawei.appmarket.commondata/item/5`。工程 `targetSdk 36`，在 Android 11+ 的**包可见性**限制下必须显式声明，否则 `resolveContentProvider` 直接返回 `null`：

```xml
<uses-permission android:name="com.huawei.appmarket.service.commondata.permission.GET_COMMON_DATA"/>
<queries>
    <provider android:authorities="com.huawei.appmarket.commondata" />
</queries>
```

漏了不会报错、不会崩，只是**华为包的安装归因全部静默落成自然量**——排查时最容易被忽略。非华为设备上该 provider 不存在，声明本身无副作用。

### 10.5 ProGuard：必须补 `-dontwarn com.bun.miitmdid.**`

`-keep class com.adjust.sdk.** { *; }`（§4.6）已覆盖 `com.adjust.sdk.oaid` / `com.adjust.sdk.huawei` 两个子包，但**还要额外加一条**，否则 `minifyXxxReleaseWithR8` 会直接失败：

```proguard
-dontwarn com.bun.miitmdid.**
```

原委：本工程**不打包 MSA（移动安全联盟）SDK**——`com.appsflyer:oaid` 里只有 `com.appsflyer.oaid.*`，它靠反射调 MSA，所以 `# OAID` 段那一大串 `com.bun.miitmdid.**` keep 规则一直是对不存在的类生效（no-op），从没暴露过问题。而 Adjust 的 OAID 插件是**直接类引用**，且 AAR 里**没带 consumer proguard 规则**（AF 的 AAR 带了），于是 R8 报：

```
ERROR: R8: Missing class com.bun.miitmdid.core.MdidSdkHelper
       (referenced from: void com.adjust.sdk.oaid.AdjustOaid.readOaid(android.content.Context))
```

`-dontwarn` 在运行时是安全的（已核对插件字节码）：`readOaid` 里 `MdidSdkHelper.InitCert` 的调用点落在 `catch Throwable` 的异常表内，缺类时被吞掉并置 `isMsaSdkAvailable=false`；`Util` 取 OAID 前先判这个标志，`MsaSdkClient` 永远不会被加载。**华为 OAID 走的是另一条 `HmsSdkClient` 路径**，不经过 MSA，所以功能不受影响——这正是我们要的那条路。

### 10.6 后台侧（代码解决不了，必须人工核对）

1. **AppsFlyer 是按包名认 App 的。** `com.<brand>.<flavor>.hw` 是一个全新的 applicationId，AF 后台没有同名 App 的话，事件在 AF 服务端就被丢弃——**这一条与 OAID 完全无关，两件事都得对**。每个 `_hw` 包都要在 AF 后台建一个 App。dev key 全渠道共用、已烧进 `BuildConfig.AF_DEV_KEY`，所以 AF 侧不需要往 Console 填任何东西，但后台的 App 必须存在。
2. **Adjust 侧**照常走 [`adjust-sync`](../../.claude/skills/adjust-sync/SKILL.md)：建 app + 6 个事件 → token 回填 Console → **重新打包**（App Token 是编译期烧进 `app/adjust-tokens.json` 的，回填后不重打包不生效）。
3. Adjust 里华为包的商店字段目前统一填 `google`。不影响 SDK 事件，但既然 AppGallery referrer 已接上，改成 Huawei App Gallery 会让商店归因报表更准。

### 10.7 验收

- **华为真机**装 `_hw` 包，`adb logcat -s HybridAdjust Adjust AppsFlyer`：Adjust 请求里 `oaid` 字段非空；AF 的 deviceData 里能看到 `oaid`。Adjust Testing Console 见 install/session 与事件。
- **非华为机回归**：装任一存量包（非 `_hw`），确认 Adjust / AF 事件照常——OAID 读取失败必须是静默跳过，不能影响 SDK 初始化。
- **构建回归**：`_hw` 与非 `_hw` 各挑一个 flavor 跑 `assembleXxxRelease`（含 R8），确认 minify 阶段无 missing class 警告。
