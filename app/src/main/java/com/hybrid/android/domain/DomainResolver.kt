package com.hybrid.android.domain

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.util.Log
import com.hybrid.android.BuildConfig
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/** STEP0 本机网络闸门，抽成接口以便测试替身。 */
fun interface NetworkGate {
    /** 是否存在「已验证可上网」的网络。 */
    fun hasValidatedNetwork(): Boolean
}

/**
 * 域名容灾状态机。严格实现 docs/admin/02-domain-failover.md 的 STEP0~3，核心约束「不乱换」：
 * 只有确认是域名故障才切换；本机网络问题（STEP0 闸门 + STEP3 中立探针双保险）只提示不换。
 *
 * - 编译期不变量（[configUrl] / [appId] / [palCode] / [brand]）来自 `assets/bootstrap.json`（ADR-0002），
 *   域名清单一律运行时下发，绝不编译期硬编码。
 * - 网络细节委托 [DomainProber]，三级取用委托 [DomainListSelector]，STEP0 委托 [NetworkGate]，
 *   本类只编排流程；三者皆可注入替身，使整条状态机可在纯 JVM 单测。
 *
 * **ADR-0009 身份约定**：
 *  - [appId]（= `BuildConfig.APPLICATION_ID`）是**域名解析键**：拉运行时配置 `GET ${configUrl}?appId=` 与告警上报都用它（全局唯一、无歧义）。
 *  - [palCode]（= `BuildConfig.PAL_CODE`）**仅用于命中后的加载 URL** `/?palcode=`，跨品牌可重复、**不作身份/解析键**。
 */
class DomainResolver(
    private val appId: String,
    private val palCode: String,
    private val store: ConfigStore,
    private val prober: DomainProber,
    private val networkGate: NetworkGate,
    private val configUrl: String,
    private val bootstrapAppId: String,
) {
    private val tag = "DomainResolver"
    private val selector = DomainListSelector(store)

    /** 运行一次完整状态机。STEP1~3 在 IO 线程，STEP0 极轻量。 */
    suspend fun resolve(): ResolveResult = withContext(Dispatchers.IO) {
        // STEP0：本机网络闸门——无「已验证可上网」网络则直接判 A，绝不碰任何域名。
        if (!networkGate.hasValidatedNetwork()) {
            Log.d(tag, "STEP0 无已验证网络 → NoNetwork（不碰域名）")
            return@withContext ResolveResult.NoNetwork
        }

        // STEP1：三级取用 + lastGood 提队首。实时拉取成功即自更新缓存。
        // ADR-0009：解析键用 appId（applicationId），不再用 palcode。
        val appIdForFetch = appId.ifEmpty { bootstrapAppId }
        val fallbackProbePath = store.readBootstrap()?.probePath ?: "/healthz"
        val selection = selector.select {
            prober.fetchRuntimeConfig(configUrl, appIdForFetch, fallbackProbePath)
        }
        if (selection.domains.isEmpty()) {
            // 既无运行时配置、无缓存、也无兜底 —— 走中立探针裁决文案（极端情况）。
            Log.w(tag, "STEP1 无任何候选域名来源")
            return@withContext verdictWhenExhausted(emptyList())
        }
        Log.d(tag, "STEP1 候选域名(${selection.domains.size}): ${selection.domains}")

        // STEP2：并发探测，取「最先命中且校验通过」者。
        val hit = probeAllPickFirst(selection.domains, selection.probePath)
        if (hit != null) {
            store.writeLastGood(hit)
            val url = "$hit/?palcode=$palCode"
            Log.d(tag, "STEP2 命中 $hit → 加载 $url")
            return@withContext ResolveResult.Loadable(url, hit)
        }

        // STEP3：清单耗尽 → 中立探针裁决 A / B。
        verdictWhenExhausted(selection.domains)
    }

    /** STEP3 裁决：中立探针通=域名问题(B，上报告警)；不通=本机问题(A)。 */
    private fun verdictWhenExhausted(domains: List<String>): ResolveResult {
        return if (prober.neutralInternetReachable()) {
            Log.w(tag, "STEP3 中立探针通、我方全挂 → ServiceDown(B)，上报告警")
            // 上报本身容错，失败不影响 UI。以 appId 标识渠道（ADR-0009）。
            prober.reportAllDomainsDown(configUrl, appId.ifEmpty { bootstrapAppId }, domains)
            ResolveResult.ServiceDown
        } else {
            Log.d(tag, "STEP3 中立探针也不通 → NoNetwork(A)，不归咎域名")
            ResolveResult.NoNetwork
        }
    }

    /**
     * STEP2 并发探测：同时探测全部域名，返回**最先命中且校验通过**者；全不命中返回 null。
     * 各域名探测把结果投进 channel，主循环取最先到达的 HIT，并保留「主域名优先」语义——
     * 即 lastGood/主域名已在 STEP1 提到队首，谁先命中谁胜出，加速首屏。
     */
    private suspend fun probeAllPickFirst(domains: List<String>, probePath: String): String? =
        coroutineScope {
            val results = Channel<Pair<String, ProbeError>>(capacity = domains.size)
            val jobs = domains.map { domain ->
                launch {
                    val r = runCatching { prober.probe(domain, probePath) }
                        .getOrDefault(ProbeError.OTHER)
                    results.trySend(domain to r)
                }
            }
            try {
                var seen = 0
                while (seen < domains.size) {
                    val (domain, result) = results.receive()
                    seen++
                    if (result == ProbeError.HIT) return@coroutineScope domain
                }
                null
            } catch (e: CancellationException) {
                throw e
            } finally {
                jobs.forEach { it.cancel() }
                results.close()
            }
        }

    companion object {
        /**
         * Android 生产入口：从 `assets/bootstrap.json` 取编译期不变量，组装真实依赖。
         * 域名清单仍一律运行时下发（[DomainProber.fetchRuntimeConfig]），bootstrap 只提供 configUrl/appId/palcode/兜底。
         *
         * ADR-0009：解析键用 [BuildConfig.APPLICATION_ID]（appId），加载 URL 仍用 [BuildConfig.PAL_CODE]（palcode）。
         */
        operator fun invoke(
            context: Context,
            appId: String = BuildConfig.APPLICATION_ID,
            palCode: String = BuildConfig.PAL_CODE,
        ): DomainResolver {
            val store = PrefsConfigStore(context)
            val brand = store.brand().ifEmpty { BuildConfig.BRAND }
            return DomainResolver(
                appId = appId,
                palCode = palCode,
                store = store,
                prober = DomainProber(brand),
                networkGate = AndroidNetworkGate(context),
                configUrl = store.configUrl(),
                bootstrapAppId = store.bootstrapAppId().ifEmpty { BuildConfig.APPLICATION_ID },
            )
        }
    }
}

/** STEP0 的 Android 实现：ConnectivityManager 判断「已验证可上网」（INTERNET + VALIDATED）。 */
class AndroidNetworkGate(private val context: Context) : NetworkGate {
    override fun hasValidatedNetwork(): Boolean = runCatching {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
            ?: return false
        val net = cm.activeNetwork ?: return false
        val cap = cm.getNetworkCapabilities(net) ?: return false
        cap.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
            cap.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }.getOrDefault(false)
}
