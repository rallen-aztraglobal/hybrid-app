package com.hybrid.android.track

import android.content.Context
import android.content.SharedPreferences
import android.net.Uri
import android.util.Log
import androidx.core.content.edit
import com.adjust.sdk.Adjust
import com.adjust.sdk.AdjustDeeplink
import com.hybrid.android.BuildConfig

/**
 * BP 原始事件模式（[AdjustBootstrap.bpRawMode]）专用：customerId 缓存与绑定、H5 自定义 scheme
 * 事件解析（含注册）、Adjust 品牌短链归因、加载 URL 追加 appSource 参数。
 *
 * 行为参照 BP 团队自家壳（`H5ShellApplication` / `AdjustH5EventTracker` /
 * `AdjustDeepLinkTracker`，见「马甲包 Adjust 归因流程代码说明」），结构用我们自己的
 * （见 docs/admin/08-adjust.md / ADR-0013）。仅在 bpRawMode 为 true 时被调用。
 *
 * 所有对 Adjust SDK 的调用均 `runCatching` 包裹，异常只打日志，绝不崩进程。
 */
object BpRawAdjustTracker {

    private const val TAG = "HybridAdjustBpRaw"
    private const val PREFS_NAME = "adjust_bp_raw"
    private const val KEY_CUSTOMER_ID = "customer_id"
    private const val PARAM_CUSTOMER_ID = "customerId"
    private const val PARAM_APP_SOURCE = "appSource"

    /** deposit 单独特判固定映射到 ad_deposit；其余 action 走「原名 / ad_ 前缀名」两级查表。 */
    private const val DEPOSIT_ACTION = "deposit"
    private const val DEPOSIT_EVENT_NAME = "ad_deposit"
    private const val REGISTER_EVENT_NAME = "ad_registration"

    /** [parseAmount] 用到的常见货币符号（去掉即可，不影响数值本身）。 */
    private const val CURRENCY_SYMBOLS = "₱$€¥£"

    /**
     * [parseAmount] 清洗后必须完整匹配的数字格式：纯数字（"1000"）或美式千分位分组
     * （"1,000"，每组恰好 3 位），后面可选跟一个小数点+小数部分。不支持欧式「.」千分位 +
     * 「,」小数（如 "1.000,50"）等歧义格式，一律判非法返回 null。
     */
    private val AMOUNT_REGEX = Regex("^(\\d+|\\d{1,3}(?:,\\d{3})+)(\\.\\d+)?$")
    private val ALPHA3_PREFIX = Regex("^[A-Za-z]{3}")
    private val ALPHA3_SUFFIX = Regex("[A-Za-z]{3}$")

    @Volatile private var cachedCustomerId: String? = null

    private var prefs: SharedPreferences? = null

    /**
     * 冷启动/进程重建时调用一次：从 SharedPreferences 恢复缓存的 customerId。
     * 必须先于任何上报调用（尤其是 ad_app_opened）以及 [AdjustBootstrap.init] 构造 AdjustConfig
     * 之前调用，否则拿不到上次绑定的 ID 去设 externalDeviceId（见 [externalDeviceId] 的说明）。
     */
    fun init(context: Context) {
        val p = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs = p
        cachedCustomerId = p.getString(KEY_CUSTOMER_ID, null)
        Log.d(TAG, "初始化完成，恢复 customerId=$cachedCustomerId")
    }

    /**
     * 当前缓存的 customerId，供 [AdjustBootstrap.init] 在构造 AdjustConfig 时设为
     * `externalDeviceId`（见该处注释：Adjust SDK v5 的 externalDeviceId 只能在 initSdk 之前
     * 一次性设置，`setExternalDeviceIdInDelay` 只在开启 first session delay 时才生效，本工程
     * 未开该延迟，故不再调用它）。
     */
    val externalDeviceId: String? get() = cachedCustomerId

    /**
     * 更新绑定的 customerId：trim 后持久化（空 → 清缓存，对应退出登录）。
     * 注意：这**只影响本地缓存**，不会立即绑定到 Adjust 侧的 externalDeviceId——SDK v5 的
     * externalDeviceId 只能在 initSdk 之前设置一次，本次登录绑定的新值要等下一次冷启动
     * （[AdjustBootstrap.init] 读到新缓存）才会随 initSdk 生效；每个事件 callback 里的
     * customerId 参数（见 [withCustomerId]）不受此限制，立即生效。
     */
    fun updateCustomerId(id: String?) {
        persistCustomerId(id?.trim().orEmpty())
    }

    private fun persistCustomerId(id: String) {
        cachedCustomerId = id.ifEmpty { null }
        prefs?.edit {
            if (id.isEmpty()) remove(KEY_CUSTOMER_ID) else putString(KEY_CUSTOMER_ID, id)
        }
    }

    /** params 里已有非空 customerId 原样返回；否则有缓存就补一个 customerId 参数。 */
    fun withCustomerId(params: Map<String, String>): Map<String, String> {
        if (!paramIgnoreCase(params, PARAM_CUSTOMER_ID).isNullOrEmpty()) return params
        val cached = cachedCustomerId?.takeIf { it.isNotEmpty() } ?: return params
        return params + (PARAM_CUSTOMER_ID to cached)
    }

    /**
     * 给要加载的站点 URL 追加 `appSource=<applicationId>`（已有则不重复）。H5 据此识别「在壳内」，
     * 改走 BingoPlusShell.openExternal / adjusth5event 上报路径。解析失败原样返回。
     */
    fun appendAppSource(url: String, appSource: String = BuildConfig.APPLICATION_ID): String {
        val uri = runCatching { Uri.parse(url) }.getOrNull() ?: return url
        val has = runCatching { uri.queryParameterNames.any { it.equals(PARAM_APP_SOURCE, ignoreCase = true) } }
            .getOrDefault(false)
        if (has) return url
        return uri.buildUpon().appendQueryParameter(PARAM_APP_SOURCE, appSource).build().toString()
    }

    /**
     * 处理 H5 自定义 scheme 事件（`adjusth5event://<action>?...`），三处入口
     * （shouldOverrideUrlLoading / JS openExternal / window.open）共用。
     * @return true 表示已消费，调用方不得再加载该 URL；false 表示不是本模式管的 scheme。
     */
    fun handleH5Url(uri: Uri): Boolean {
        if (!"adjusth5event".equals(uri.scheme, ignoreCase = true)) return false
        val action = uri.host?.lowercase().orEmpty()
        val query = parseQuery(uri)

        if (action == "updatecustomerid") {
            updateCustomerId(paramIgnoreCase(query, PARAM_CUSTOMER_ID))
            return true
        }
        if (action == "register") {
            // 注册：先绑定 customerId 再上报，顺序不能对调；customerId 为空也先置空（对齐参考实现）。
            updateCustomerId(paramIgnoreCase(query, PARAM_CUSTOMER_ID))
            report(REGISTER_EVENT_NAME, query)
            return true
        }

        val adjustName = resolveEventName(action)
        if (adjustName == null) {
            Log.d(TAG, "未知 H5 事件 action=$action，仅消费不上报")
            return true
        }
        // 其他事件带了非空 customerId 顺手绑定；不带则不动缓存。
        paramIgnoreCase(query, PARAM_CUSTOMER_ID)?.let { updateCustomerId(it) }
        report(adjustName, query)
        return true
    }

    /** 统一上报：query 全部作参数、补 customerId、amount/currency 设收入、orderId 设去重 ID。 */
    private fun report(adjustName: String, query: Map<String, String>) {
        val revenue = parseAmount(paramIgnoreCase(query, "amount"))
            ?.let { it to parseCurrency(paramIgnoreCase(query, "currency")) }
        val dedupId = paramIgnoreCase(query, "orderId")?.trim()?.takeIf { it.isNotEmpty() }
        AdjustBootstrap.trackRaw(adjustName, withCustomerId(query), revenue, dedupId)
    }

    /**
     * Adjust 品牌短链归因：http/https 且 host 忽略大小写等于 [BuildConfig.ADJUST_DEEP_LINK_HOST]
     * （非空）才处理。命中后走 [Adjust.processDeeplink] + 上报 ad_deeplink_opened。
     * @return true 表示是短链；调用方不应再 loadUrl 这条短链，照常加载默认首页。
     */
    fun handleDeepLink(context: Context, uri: Uri): Boolean {
        val configuredHost = BuildConfig.ADJUST_DEEP_LINK_HOST
        if (configuredHost.isBlank()) return false
        val scheme = uri.scheme
        if (!("http".equals(scheme, ignoreCase = true) || "https".equals(scheme, ignoreCase = true))) {
            return false
        }
        if (!configuredHost.equals(uri.host, ignoreCase = true)) return false

        runCatching { Adjust.processDeeplink(AdjustDeeplink(uri), context.applicationContext) }
            .onFailure { Log.w(TAG, "processDeeplink 失败: ${it.message}") }
        AdjustBootstrap.trackRaw("ad_deeplink_opened", withCustomerId(emptyMap()))
        return true
    }

    /**
     * action → Adjust 事件 name：`deposit` 固定映射 `ad_deposit`；否则先按 action 原名在
     * ADJUST_EVENT_MAP 里找（忽略大小写），找不到再找 `ad_` + action；都没有则 null。
     * [lookup] 默认接 [AdjustBootstrap.findEventName]（真实 ADJUST_EVENT_MAP），测试可注入假表。
     */
    internal fun resolveEventName(
        action: String,
        lookup: (String) -> String? = AdjustBootstrap::findEventName,
    ): String? {
        if (action.isBlank()) return null
        if (action == DEPOSIT_ACTION) return DEPOSIT_EVENT_NAME
        return lookup(action) ?: lookup("ad_$action")
    }

    /**
     * 解析金额字符串（收紧版，避免如 `1e3` 被当成 `13` 这类误判）：
     * trim → 去掉空白、常见货币符号（₱ $ € ¥ £）与前后缀的 3 个字母币种码（如 `PHP100` / `100 PHP`）
     * → 必须完整匹配 [AMOUNT_REGEX]（纯数字，或美式千分位分组 + 可选小数）才去掉分组逗号、
     * `toDoubleOrNull`；科学计数法（`1e3`）、欧式 `.`千分位+`,`小数（`1.000,50`）、
     * 带正负号后缀（`100-`）、非数字（`abc`）等一律返回 null，不设收入。
     */
    internal fun parseAmount(raw: String?): Double? {
        if (raw == null) return null
        var cleaned = raw.trim()
        if (cleaned.isEmpty()) return null
        cleaned = cleaned.filterNot { it.isWhitespace() || it in CURRENCY_SYMBOLS }
        cleaned = cleaned.replace(ALPHA3_PREFIX, "").replace(ALPHA3_SUFFIX, "")
        if (cleaned.isEmpty() || !AMOUNT_REGEX.matches(cleaned)) return null
        return cleaned.replace(",", "").toDoubleOrNull()
    }

    /** 解析币种：trim 转大写，须是 3 个字母；缺失或不合法回落 PHP。 */
    internal fun parseCurrency(raw: String?): String {
        val upper = raw?.trim()?.uppercase().orEmpty()
        return if (upper.length == 3 && upper.all { it in 'A'..'Z' }) upper else "PHP"
    }

    /** 解析 query 为 Map：键忽略大小写取值时用 [paramIgnoreCase]；空白值视为无（不进 map）。 */
    private fun parseQuery(uri: Uri): Map<String, String> = runCatching {
        uri.queryParameterNames
            .associateWith { uri.getQueryParameter(it) }
            .filterValues { !it.isNullOrBlank() }
            .mapValues { (_, v) -> v!! }
    }.getOrDefault(emptyMap())

    /** 忽略大小写在 query map 中查某个参数。 */
    private fun paramIgnoreCase(query: Map<String, String>, key: String): String? =
        query.entries.firstOrNull { it.key.equals(key, ignoreCase = true) }?.value
}
