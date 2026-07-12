package com.hybrid.android.track

import android.content.Context
import android.util.Log
import com.adjust.sdk.Adjust
import com.adjust.sdk.AdjustConfig
import com.adjust.sdk.AdjustEvent
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
        val environment = if (BuildConfig.ENABLE_TEST_EVENTS) {
            AdjustConfig.ENVIRONMENT_SANDBOX
        } else {
            AdjustConfig.ENVIRONMENT_PRODUCTION
        }
        val config = AdjustConfig(context.applicationContext, BuildConfig.ADJUST_APP_TOKEN, environment)
        Adjust.initSdk(config)
        initialized = true
        Log.d(TAG, "Adjust 初始化完成，environment=$environment")
    }

    /**
     * 发一个逻辑事件（与 AppsFlyer 同源分发，见 WebViewActivity.sendAFEvent）。
     * 未绑定 App Token，或该事件在后台未配 token，均 no-op。
     */
    fun trackEvent(logicalName: String, params: Map<String, Any> = emptyMap()) {
        if (!enabled) return
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

    /** 解析 BuildConfig.ADJUST_EVENT_MAP（JSON 字符串）为 name→token 表；解析失败回落空表。 */
    private fun parseEventMap(): Map<String, String> = runCatching {
        val json = JSONObject(BuildConfig.ADJUST_EVENT_MAP)
        buildMap { json.keys().forEach { key -> put(key, json.getString(key)) } }
    }.getOrDefault(emptyMap())
}
