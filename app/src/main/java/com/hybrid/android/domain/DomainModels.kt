package com.hybrid.android.domain

import org.json.JSONArray
import org.json.JSONObject

/**
 * 域名解析的最终结果，对应 docs/admin/02-domain-failover.md 状态机的三个出口。
 */
sealed class ResolveResult {
    /** STEP2 命中：拿到可加载的完整 URL（已拼接 palcode）。 */
    data class Loadable(val url: String, val domain: String) : ResolveResult()

    /** STEP3 判定 B：设备能上公网但我方域名/服务全挂 → 「服务暂时不可用」。 */
    object ServiceDown : ResolveResult()

    /** STEP0 / STEP3 判定 A：本机没有真实公网 → 「网络异常」。绝不归咎域名。 */
    object NoNetwork : ResolveResult()
}

/**
 * 单个域名探测的归类，对应 02 文档「3. 错误类型与动作映射」。
 * 除 [HIT] 外都意味着「试下一个」；全部非命中后由 STEP3 中立探针裁决文案。
 */
enum class ProbeError {
    /** HTTP 2xx/3xx 且证书覆盖目标域名 —— 唯一的命中（证书=反劫持的密码学保证）。 */
    HIT,

    /** DNS 解析失败 / UnknownHostException —— 疑似域名问题（也可能没网，最终由 STEP3 区分）。 */
    DNS_FAIL,

    /** 连接 / 读取超时 —— 疑似域名问题。 */
    TIMEOUT,

    /** 连接被拒 ConnectException —— 域名问题（服务没起）。 */
    CONN_REFUSED,

    /** TLS 握手失败或证书域名不符 —— 强信号：疑似被中间人劫持。 */
    TLS_FAIL,

    /** HTTP 5xx —— 服务异常。 */
    HTTP_5XX,

    /** 证书不覆盖目标域名 —— 关键：可达但不是我们（DNS 污染 / 中间人劫持到了别的服务器）。 */
    HIJACK,

    /** 其它（4xx 如地区封锁 403、解析异常等）—— 该域名当前不可服务，试下一个。 */
    OTHER,
}

/**
 * 运行时配置端点 `GET ${configUrl}?palcode=` 的响应。
 * 形如 docs/admin/01-architecture.md §5.6：
 * `{ "palcode": "...", "domains": [...], "probePath": "/healthz", "configVersion": 42, "ttlSeconds": 600 }`
 */
data class DomainConfig(
    val domains: List<String>,
    val probePath: String,
    val configVersion: Int = 0,
) {
    /** 序列化进本地缓存（连同 probePath/version 一起持久化，保证缓存自洽）。 */
    fun toJson(): String = JSONObject().apply {
        put("domains", JSONArray(domains))
        put("probePath", probePath)
        put("configVersion", configVersion)
    }.toString()

    companion object {
        /** 解析运行时接口响应；缺字段给出安全默认值。失败返回 null（调用方降级到缓存/兜底）。 */
        fun parse(body: String, fallbackProbePath: String = "/healthz"): DomainConfig? = runCatching {
            val json = JSONObject(body)
            val arr = json.optJSONArray("domains") ?: return null
            val domains = (0 until arr.length())
                .mapNotNull { arr.optString(it, null)?.trim()?.takeIf(String::isNotEmpty) }
                .map { it.trimEnd('/') }
                .distinct()
            if (domains.isEmpty()) return null
            // APK 端再防一道：线上一律 https，过滤非 https（即便后台校验也兜一层，见 01 §5.7）。
            val httpsOnly = domains.filter { it.startsWith("https://", ignoreCase = true) }
            if (httpsOnly.isEmpty()) return null
            DomainConfig(
                domains = httpsOnly.take(MAX_DOMAINS),
                probePath = json.optString("probePath", fallbackProbePath).ifBlank { fallbackProbePath },
                configVersion = json.optInt("configVersion", 0),
            )
        }.getOrNull()

        /** 主域名 + 最多 3 个备用 = 4。 */
        const val MAX_DOMAINS = 4
    }
}

/**
 * 编译期兜底清单 `assets/bootstrap.json`（见 ADR-0002 / 02 文档 STEP1 ③）。
 * 仅在「从未成功拉取过运行时配置」时使用；成功一次后由本地缓存接管兜底。
 */
data class BootstrapConfig(
    val brand: String,
    /** 编译期烧录的 applicationId（ADR-0009：域名解析键）。运行时优先用 [com.hybrid.android.BuildConfig.APPLICATION_ID]。 */
    val appId: String,
    /** 编译期烧录的 PAL_CODE（仅用于加载 URL `/?palcode=`，非身份/解析键）。 */
    val palcode: String,
    val configUrl: String,
    val probePath: String,
    val defaultDomains: List<String>,
) {
    companion object {
        fun parse(body: String): BootstrapConfig? = runCatching {
            val json = JSONObject(body)
            val arr = json.optJSONArray("defaultDomains") ?: JSONArray()
            val domains = (0 until arr.length())
                .mapNotNull { arr.optString(it, null)?.trim()?.takeIf(String::isNotEmpty) }
                .map { it.trimEnd('/') }
            BootstrapConfig(
                brand = json.optString("brand", ""),
                appId = json.optString("appId", ""),
                palcode = json.optString("palcode", ""),
                configUrl = json.optString("configUrl", "").trimEnd('/'),
                probePath = json.optString("probePath", "/healthz").ifBlank { "/healthz" },
                defaultDomains = domains.take(DomainConfig.MAX_DOMAINS),
            )
        }.getOrNull()
    }
}
