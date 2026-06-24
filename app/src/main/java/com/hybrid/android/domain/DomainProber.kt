package com.hybrid.android.domain

import android.util.Log
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Request
import org.json.JSONObject
import java.io.IOException
import java.net.ConnectException
import java.net.SocketTimeoutException
import java.net.UnknownHostException
import java.security.cert.X509Certificate
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLPeerUnverifiedException
import javax.net.ssl.SSLException

/**
 * 域名探测器：封装 STEP1 拉取、STEP2 探测+校验、STEP3 中立探针、告警上报的网络细节。
 * 网络实现集中于此，[DomainResolver] 只编排状态机，便于替身测试。
 *
 * 设计要点（对应 docs/admin/02-domain-failover.md）：
 *  - 探测「可达 ≠ 是我们」：校验 ① 证书 CN/SAN 含目标域名（防 DNS 污染到劫持服务器）
 *    ② 业务特征 JSON `{"ok":true,"brand":"<brand>"}`（防 ISP 劫持假 200 / 门户页）。
 *  - 不跟随重定向（劫持常用 302 跳广告页）。
 *  - 错误分类是纯函数 [classify]，可脱离 Android 做 JVM 单测。
 */
open class DomainProber(
    private val brand: String,
    perDomainTimeoutMs: Long = 2500,
    neutralTimeoutMs: Long = 2000,
    fetchTimeoutMs: Long = 1200,
) {
    private val tag = "DomainResolver"

    // 探测/中立/拉取各自的 client：超时预算不同（见 02 文档 §5）。统一关掉自动重定向。
    private val probeClient: OkHttpClient = baseClient(perDomainTimeoutMs)
    private val neutralClient: OkHttpClient = baseClient(neutralTimeoutMs)
    private val fetchClient: OkHttpClient = baseClient(fetchTimeoutMs)

    private fun baseClient(timeoutMs: Long): OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(timeoutMs, TimeUnit.MILLISECONDS)
        .readTimeout(timeoutMs, TimeUnit.MILLISECONDS)
        .callTimeout(timeoutMs + 800, TimeUnit.MILLISECONDS)
        .followRedirects(false)            // 劫持常用 302 跳广告页
        .followSslRedirects(false)
        .retryOnConnectionFailure(false)
        .build()

    /**
     * STEP1：实时拉取运行时配置。短超时（默认 1.2s），失败返回 null。
     * `GET ${configUrl}?appId=<applicationId>`（ADR-0009：解析键改为 applicationId，
     * 因 PAL_CODE 跨品牌可重复、不唯一；palcode 仅用于 STEP2 命中后的加载 URL）。
     */
    open fun fetchRuntimeConfig(configUrl: String, appId: String, fallbackProbePath: String): DomainConfig? {
        if (configUrl.isBlank()) return null
        val url = buildUrl(configUrl, appId)
        return runCatching {
            val req = Request.Builder().url(url).header("Accept", "application/json").get().build()
            fetchClient.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return null
                val body = resp.body?.string() ?: return null
                DomainConfig.parse(body, fallbackProbePath)
            }
        }.getOrElse {
            Log.w(tag, "STEP1 拉取运行时配置失败，降级到缓存/兜底: ${it.javaClass.simpleName}")
            null
        }
    }

    /**
     * STEP2：探测单个域名并校验「确实是我们的站点」。返回 [ProbeError]，[ProbeError.HIT] 即命中。
     */
    open fun probe(domain: String, probePath: String): ProbeError {
        val url = domain.trimEnd('/') + ensureLeadingSlash(probePath)
        return try {
            val req = Request.Builder().url(url).header("Accept", "application/json").get().build()
            probeClient.newCall(req).execute().use { resp ->
                val code = resp.code
                when {
                    code in 500..599 -> ProbeError.HTTP_5XX
                    code != 200 -> ProbeError.OTHER
                    !certMatchesDomain(resp.handshake?.peerCertificates, domain) -> ProbeError.HIJACK
                    !bodyMatchesSite(resp.body?.string()) -> ProbeError.HIJACK
                    else -> ProbeError.HIT
                }
            }
        } catch (t: Throwable) {
            classify(t).also { Log.d(tag, "STEP2 探测 $domain → $it") }
        }
    }

    /**
     * STEP3：中立连通性探针。多端点取「任一成功即有公网」，互为备份避免单点被封。
     * @return true=设备能上公网（→ 域名问题 B）；false=本机没网（→ 本机问题 A）。
     */
    open fun neutralInternetReachable(): Boolean {
        for ((url, expect) in NEUTRAL_ENDPOINTS) {
            val ok = runCatching {
                val req = Request.Builder().url(url).get().build()
                neutralClient.newCall(req).execute().use { resp ->
                    if (expect == EXPECT_SUCCESS_BODY) {
                        resp.isSuccessful && (resp.body?.string()?.contains("Success") == true)
                    } else {
                        resp.code == expect
                    }
                }
            }.getOrDefault(false)
            if (ok) {
                Log.d(tag, "STEP3 中立探针命中: $url → 设备有公网")
                return true
            }
        }
        Log.d(tag, "STEP3 中立探针全失败 → 判定本机无网")
        return false
    }

    /**
     * 全部备用域名耗尽且判定为域名问题（B）时上报后台告警。
     * 本身必须容错：失败只记日志，绝不影响 UI（见 02 文档 STEP3）。
     * 上报以 appId 标识渠道（ADR-0009：身份/解析键统一用 applicationId）。
     */
    open fun reportAllDomainsDown(configUrl: String, appId: String, domains: List<String>) {
        runCatching {
            if (configUrl.isBlank()) return
            val reportUrl = deriveReportUrl(configUrl)
            val payload = JSONObject().apply {
                put("appId", appId)
                put("brand", brand)
                put("event", "all_domains_down")
                put("domains", domains.joinToString(","))
                put("ts", System.currentTimeMillis())
            }.toString()
            val req = Request.Builder()
                .url(reportUrl)
                .post(payload.toRequestBody(JSON_MEDIA))
                .build()
            neutralClient.newCall(req).execute().use { /* 忽略响应 */ }
        }.onFailure { Log.w(tag, "告警上报失败（已忽略，不影响 UI）: ${it.javaClass.simpleName}") }
    }

    /** 业务特征校验：`{"ok":true,"brand":"<brand>"}`，brand 必须与本包一致。 */
    private fun bodyMatchesSite(body: String?): Boolean = bodyMatchesSite(body, brand)

    /** 证书域名校验：证书链首张的 CN/SAN 是否覆盖目标 host（防 DNS 污染到劫持服务器）。 */
    private fun certMatchesDomain(certs: List<java.security.cert.Certificate>?, domain: String): Boolean {
        val host = hostOf(domain) ?: return false
        val x509 = certs?.firstOrNull() as? X509Certificate ?: return false
        return certCoversHost(x509, host)
    }

    companion object {
        private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()
        private const val EXPECT_SUCCESS_BODY = -1

        /** 中立端点（互为备份）。captive.apple.com 期望响应体含 "Success"。 */
        val NEUTRAL_ENDPOINTS: List<Pair<String, Int>> = listOf(
            "https://www.gstatic.com/generate_204" to 204,
            "https://cp.cloudflare.com/generate_204" to 204,
            "https://captive.apple.com" to EXPECT_SUCCESS_BODY,
        )

        /**
         * 纯函数：把网络异常映射到 [ProbeError]，对应 02 文档「3. 错误类型与动作映射」。
         * 抽成 companion 静态函数以便 JVM 单测（不依赖 Android/OkHttp 实例）。
         */
        fun classify(t: Throwable): ProbeError = when (t) {
            is UnknownHostException -> ProbeError.DNS_FAIL
            is SocketTimeoutException -> ProbeError.TIMEOUT
            is ConnectException -> ProbeError.CONN_REFUSED
            is SSLPeerUnverifiedException -> ProbeError.TLS_FAIL
            is SSLException -> ProbeError.TLS_FAIL
            is IOException -> {
                // OkHttp 的 callTimeout 抛 InterruptedIOException("timeout")，归到超时。
                if (t.message?.contains("timeout", ignoreCase = true) == true) ProbeError.TIMEOUT
                else ProbeError.OTHER
            }
            else -> ProbeError.OTHER
        }

        /** 证书 CN/SAN 是否覆盖 host（支持通配符 *.example.com）。 */
        fun certCoversHost(cert: X509Certificate, host: String): Boolean {
            val names = mutableListOf<String>()
            // SAN（type 2 = dNSName）
            runCatching {
                cert.subjectAlternativeNames?.forEach { entry ->
                    val type = entry.getOrNull(0) as? Int
                    val value = entry.getOrNull(1) as? String
                    if (type == 2 && value != null) names += value
                }
            }
            // CN 兜底
            runCatching {
                val dn = cert.subjectX500Principal.name
                Regex("CN=([^,]+)").find(dn)?.groupValues?.getOrNull(1)?.let { names += it }
            }
            return namesCoverHost(names, host)
        }

        /** 证书名称列表是否覆盖 host。抽出（纯逻辑）以便 JVM 单测通配符/精确匹配规则。 */
        fun namesCoverHost(names: List<String>, host: String): Boolean =
            names.any { hostMatches(it.trim().lowercase(), host.lowercase()) }

        /**
         * 业务特征校验（纯函数，便于单测）：`{"ok":true,"brand":"<brand>"}`。
         * brand 为空（后端未回该字段）则只校验 ok（向后兼容）；非空则必须不区分大小写匹配。
         * 这是「可达 ≠ 是我们」的关键防线：劫持/门户页即便返回 200 也过不了校验。
         */
        fun bodyMatchesSite(body: String?, brand: String): Boolean {
            if (body.isNullOrBlank()) return false
            return runCatching {
                val json = JSONObject(body)
                val ok = json.optBoolean("ok", false)
                val b = json.optString("brand", "")
                ok && (b.isEmpty() || b.equals(brand, ignoreCase = true))
            }.getOrDefault(false)
        }

        private fun hostMatches(pattern: String, host: String): Boolean {
            if (pattern == host) return true
            if (pattern.startsWith("*.")) {
                val suffix = pattern.substring(1) // ".example.com"
                // *.example.com 匹配 a.example.com，但不匹配 example.com 或 a.b.example.com
                if (!host.endsWith(suffix)) return false
                val head = host.dropLast(suffix.length)
                return head.isNotEmpty() && !head.contains('.')
            }
            return false
        }

        fun hostOf(domain: String): String? = runCatching {
            (if (domain.contains("://")) domain else "https://$domain").toHttpUrl().host
        }.getOrNull()

        /** 配置端点拼接解析键：ADR-0009 用 `appId`（applicationId）而非 palcode。 */
        private fun buildUrl(configUrl: String, appId: String): String {
            val base = configUrl.trimEnd('/')
            val sep = if (base.contains('?')) "&" else "?"
            return "$base${sep}appId=$appId"
        }

        /** 把 configUrl 的最后一段换成 /api/app/report（与配置端点同基座，最大化可达）。 */
        private fun deriveReportUrl(configUrl: String): String {
            val base = configUrl.substringBefore('?').trimEnd('/')
            return if (base.endsWith("/config")) base.removeSuffix("/config") + "/report"
            else "$base/report"
        }

        private fun ensureLeadingSlash(path: String): String =
            if (path.startsWith("/")) path else "/$path"
    }
}
