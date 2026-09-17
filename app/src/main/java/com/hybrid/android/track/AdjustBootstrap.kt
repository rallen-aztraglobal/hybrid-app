package com.hybrid.android.track

import android.content.Context
import android.util.Log
import com.adjust.sdk.Adjust
import com.adjust.sdk.AdjustConfig
import com.adjust.sdk.AdjustEvent
import com.adjust.sdk.LogLevel
import com.adjust.sdk.oaid.AdjustOaid
import com.appsflyer.AppsFlyerLib
import com.appsflyer.attribution.AppsFlyerRequestListener
import com.hybrid.android.BuildConfig
import org.json.JSONObject

/**
 * Adjust 归因功能门控（feature gate，见 docs/admin/08-adjust.md §4.3 / ADR-0013）。
 *
 * 与 [com.hybrid.android.push.PushBootstrap] 同构：未在后台给该渠道包绑定 Adjust App Token 时，
 * [enabled] 返回 false，[init]/[trackEvent] 全部 no-op（不初始化 SDK、不发任何事件、无网络流量）。
 *
 * 只需在 Console 给渠道绑定 App Token + 事件 CSV 后重打包，[enabled] 自动变为 true，零改码。
 */
object AdjustBootstrap {

    private const val TAG = "HybridAdjust"

    /** 是否已在后台绑定 Adjust（App Token 非空）。空 → 全程 no-op。 */
    val enabled: Boolean get() = BuildConfig.ADJUST_APP_TOKEN.isNotBlank()

    /**
     * BP 原始事件模式（编译期开关，见 docs/admin/08-adjust.md「BP 原始事件」）。
     * 关闭时（默认）：[init]/[trackEvent] 走本文件原有逻辑，一字不差。
     * 开启时：[init] 额外初始化 [BpRawAdjustTracker] 并上报 ad_app_opened；
     * [trackEvent] 的逻辑事件（拦截接口判定出的 6 个）全部丢弃，不再分发到 Adjust——本模式的
     * 事件全部由 H5 通过 adjusth5event:// 触发（注册也是），见 [BpRawAdjustTracker]。
     */
    val bpRawMode: Boolean get() = enabled && BuildConfig.ADJUST_BP_RAW_EVENTS

    /**
     * App 内部事件名 → Adjust 事件 name 的固定适配表（见 docs/admin/08-adjust.md §4.5）。
     * 只有这两条名字对不齐，其余（Purchase/OldRegPurchase/TPFirstDeposit/AddToCart/Install）同名直接命中。
     * 这是 App 内一次性代码常量，不是每渠道要填的数据——后台只需 App Token + 事件 CSV。
     */
    private val LOGICAL_TO_ADJUST_NAME = mapOf(
        "af_login" to "Login",
        "af_complete_registration" to "CompleteRegistration",
    )

    /** Adjust 事件 name → token（编译期从 ADJUST_EVENT_MAP 懒解析）。 */
    private val eventTokens: Map<String, String> by lazy { parseEventMap() }

    @Volatile
    private var initialized = false

    /** bpRawMode 下 AF 同步上报所需的 Context（init 时记下 applicationContext）。 */
    @Volatile
    private var appContext: Context? = null

    /**
     * 在 WebViewActivity.onCreate 尽早调用。未绑定 App Token 时 no-op；重复调用幂等。
     * 环境沿用现有 ENABLE_TEST_EVENTS 开关：测试包 → SANDBOX，生产包 → PRODUCTION（不新增开关）。
     */
    fun init(context: Context) {
        if (!enabled) {
            Log.d(TAG, "未绑定 Adjust App Token，跳过初始化")
            return
        }
        if (initialized) return

        // OAID 采集开关：华为设备无 GMS → 拿不到 GAID，OAID 是 Adjust 唯一可用的广告标识；
        // 不开的话华为包（applicationId 以 .hw 结尾）的事件即使发出去也没有设备标识、无法归因。
        // 必须在 initSdk 之前调用（AdjustOaid.isOaidToBeRead 默认 false）；非华为设备上插件内部
        // 探测不到 HMS / MSA SDK 即跳过，不影响初始化，故这里无条件调用、不按包做分支。
        // AppGallery 安装来源由 adjust-android-huawei-referrer 插件默认开启，无需在此调用。
        AdjustOaid.readOaid(context.applicationContext)

        // BP 原始事件模式：先恢复 customerId 缓存，才能在下面构造 AdjustConfig 时把它设为
        // externalDeviceId——已用字节码核实 adjust-android 5.4.1 的 setExternalDeviceIdInDelay
        // 只在开启 first session delay 期间才写入，否则静默 return；本工程不开该延迟（未登录也要
        // 立刻报 ad_app_opened），过去在 updateCustomerId 里调它从未生效。externalDeviceId 只能
        // initSdk 之前设置一次：本次登录新绑定的 ID 要到下次冷启动（读到新缓存）才会生效。
        if (bpRawMode) {
            appContext = context.applicationContext
            BpRawAdjustTracker.init(context)
        }

        val environment = if (BuildConfig.ENABLE_TEST_EVENTS) {
            AdjustConfig.ENVIRONMENT_SANDBOX
        } else {
            AdjustConfig.ENVIRONMENT_PRODUCTION
        }
        val config = AdjustConfig(context.applicationContext, BuildConfig.ADJUST_APP_TOKEN, environment)
        // 测试包（SANDBOX）把 SDK 日志开到 VERBOSE，logcat -s Adjust 能看到每个请求与服务端回包，
        // 便于排查「事件发了但面板没有」；生产包保持 SDK 默认（几乎静默）。
        if (BuildConfig.ENABLE_TEST_EVENTS) config.setLogLevel(LogLevel.VERBOSE)
        if (bpRawMode) {
            BpRawAdjustTracker.externalDeviceId?.takeIf { it.isNotBlank() }?.let {
                config.setExternalDeviceId(it)
            }
        }
        Adjust.initSdk(config)
        initialized = true
        Log.d(TAG, "Adjust 初始化完成，environment=$environment")

        if (bpRawMode) {
            // 未登录也报，每次冷启动都报（对齐参考实现 H5ShellApplication 的行为）。
            trackRaw("ad_app_opened", BpRawAdjustTracker.withCustomerId(emptyMap()))
        }
    }

    /**
     * 发一个逻辑事件（与 AppsFlyer 同源分发，见 WebViewActivity.sendAFEvent）。
     * 未绑定 App Token 时 no-op。bpRawMode 下逻辑事件全部丢弃（本模式事件由 H5 触发，见类注释）；
     * AppsFlyer 那条分发不受影响，仍由 WebViewActivity.sendAFEvent 照发。
     */
    fun trackEvent(logicalName: String, params: Map<String, Any> = emptyMap()) {
        if (!enabled) return
        if (bpRawMode) {
            Log.d(TAG, "BP 原始事件模式下丢弃逻辑事件: $logicalName")
            return
        }
        val adjustName = LOGICAL_TO_ADJUST_NAME[logicalName] ?: logicalName
        val token = eventTokens[adjustName] ?: run {
            Log.d(TAG, "事件 $logicalName → $adjustName 未配 token，跳过")
            return
        }
        val event = AdjustEvent(token)
        params.forEach { (key, value) -> event.addCallbackParameter(key, value.toString()) }
        Adjust.trackEvent(event)
        Log.d(TAG, "已发送 Adjust 事件: $logicalName → $adjustName")
    }

    /**
     * 上报一个「原始」事件（BP 原始事件模式专用，不经 LOGICAL_TO_ADJUST_NAME 适配，事件名即后台
     * CSV 里的 name，如 ad_deposit / ad_web_login）。供 [BpRawAdjustTracker] 复用，所有调用均已由
     * [enabled] 门控。
     *
     * 同一事件**同名同参数再给 AppsFlyer 一份**（需求：本分支的事件同步上报 AF）：
     * params 原样作 AF 事件参数，有收入时追加 af_revenue / af_currency。
     * Adjust 侧没配对应 token 时只跳过 Adjust，AF 照发。
     *
     * @param params  callback 参数（已含补齐的 customerId）
     * @param revenue 收入金额与币种，null = 不设收入
     * @param dedupId Adjust 去重 ID（订单号），null = 不设
     */
    internal fun trackRaw(
        adjustName: String,
        params: Map<String, String> = emptyMap(),
        revenue: Pair<Double, String>? = null,
        dedupId: String? = null,
    ) {
        // 写死的 ad_app_opened / ad_deeplink_opened / ad_registration / ad_deposit 与 CSV 里配置的
        // key 大小写可能不一致；先忽略大小写归一到表里的真实 key 再查 token，否则 CSV 大小写
        // 一旦对不上就只发 AF、Adjust 侧静默丢事件。
        val token = findEventName(adjustName)?.let { eventTokens[it] }
        if (token == null) {
            Log.d(TAG, "原始事件 $adjustName 未配 Adjust token，跳过 Adjust 侧")
        } else {
            val event = AdjustEvent(token)
            params.forEach { (key, value) -> event.addCallbackParameter(key, value) }
            revenue?.let { (amount, currency) -> event.setRevenue(amount, currency) }
            dedupId?.let { event.setDeduplicationId(it) }
            runCatching { Adjust.trackEvent(event) }
                .onFailure { Log.w(TAG, "上报 Adjust 原始事件 $adjustName 失败: ${it.message}") }
                .onSuccess { Log.d(TAG, "已发送 Adjust 原始事件: $adjustName") }
        }
        trackRawToAppsFlyer(adjustName, params, revenue)
    }

    /** bpRawMode 事件同步给 AppsFlyer：同名事件，参数同 Adjust callback，收入映射 af_revenue/af_currency。 */
    private fun trackRawToAppsFlyer(name: String, params: Map<String, String>, revenue: Pair<Double, String>?) {
        val ctx = appContext ?: run {
            Log.w(TAG, "AF 同步上报 $name 跳过：尚未 init")
            return
        }
        val afParams = HashMap<String, Any>(params)
        revenue?.let { (amount, currency) ->
            afParams["af_revenue"] = amount
            afParams["af_currency"] = currency
        }
        runCatching {
            AppsFlyerLib.getInstance().logEvent(ctx, name, afParams, object : AppsFlyerRequestListener {
                override fun onSuccess() {
                    Log.d(TAG, "AF 同步上报成功: $name")
                }

                override fun onError(errorCode: Int, message: String) {
                    Log.w(TAG, "AF 同步上报失败: $name code=$errorCode $message")
                }
            })
        }.onFailure { Log.w(TAG, "AF 同步上报 $name 异常: ${it.message}") }
    }

    /**
     * 忽略大小写查 ADJUST_EVENT_MAP 里是否配了该事件 name，返回表里实际的 key（保留原始大小写）。
     * 供 [BpRawAdjustTracker] 解析 H5 action → 事件名时使用；未配置返回 null。
     */
    internal fun findEventName(name: String): String? =
        eventTokens.keys.firstOrNull { it.equals(name, ignoreCase = true) }

    /** 解析 BuildConfig.ADJUST_EVENT_MAP（JSON 字符串）为 name→token 表；解析失败回落空表。 */
    private fun parseEventMap(): Map<String, String> = runCatching {
        val json = JSONObject(BuildConfig.ADJUST_EVENT_MAP)
        buildMap { json.keys().forEach { key -> put(key, json.getString(key)) } }
    }.getOrDefault(emptyMap())
}
