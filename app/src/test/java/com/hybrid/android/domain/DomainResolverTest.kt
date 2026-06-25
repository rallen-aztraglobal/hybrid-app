package com.hybrid.android.domain

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * 整条状态机（STEP0~3）的 JVM 单测，覆盖 brief 要求的三场景：
 *  - 断网（STEP0 闸门）→ NoNetwork，且绝不碰任何域名。
 *  - 被封但有网（域名全挂 + 中立探针通）→ ServiceDown(B)，且上报告警。
 *  - 全挂且本机无网（域名全挂 + 中立探针不通）→ NoNetwork(A)，不归咎域名。
 * 另测 STEP2「最先命中」与「不乱换」关键不变量。
 *
 * 用 Robolectric 运行：DomainResolver 内部调 android.util.Log，纯 JVM 桩会抛 not-mocked，
 * Robolectric 提供真实 Log 实现。状态机其余依赖（探测/存储/网络闸门）全用注入替身，不碰真实网络。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class DomainResolverTest {

    private fun store(
        cached: DomainConfig? = DomainConfig(
            listOf("https://main.com", "https://b1.com", "https://b2.com"), "/healthz"
        ),
        bootstrap: BootstrapConfig? = BootstrapConfig(
            brand = "ap",
            appId = "com.arenaplus.ap01018",
            palcode = "palX",
            configUrl = "https://cfg.example/api/app/config",
            probePath = "/healthz",
            defaultDomains = listOf("https://boot.com"),
        ),
    ) = object : ConfigStore {
        var cachedV = cached
        override fun readCachedConfig() = cachedV
        override fun writeCachedConfig(config: DomainConfig) { cachedV = config }
        override fun readBootstrap() = bootstrap
    }

    /** 可编排的探测替身。 */
    private open class FakeProber(
        val hitDomain: String? = null,
        val neutralUp: Boolean = false,
        val runtime: DomainConfig? = null,
    ) : DomainProber(brand = "ap") {
        var reported = false
        var fetchCalled = false
        var fetchedAppId: String? = null
        var reportedAppId: String? = null
        var probedDomains = mutableListOf<String>()
        override fun fetchRuntimeConfig(configUrl: String, appId: String, fallbackProbePath: String): DomainConfig? {
            fetchCalled = true
            fetchedAppId = appId
            return runtime
        }
        override fun probe(domain: String, probePath: String): ProbeError {
            probedDomains += domain
            return if (domain == hitDomain) ProbeError.HIT else ProbeError.TIMEOUT
        }
        override fun neutralInternetReachable() = neutralUp
        override fun reportAllDomainsDown(configUrl: String, appId: String, domains: List<String>) {
            reported = true
            reportedAppId = appId
        }
    }

    private fun resolver(
        gate: Boolean,
        prober: DomainProber,
        store: ConfigStore = store(),
        appId: String = "com.arenaplus.ap01018",
        palCode: String = "palX",
    ) = DomainResolver(
        appId = appId,
        palCode = palCode,
        store = store,
        prober = prober,
        networkGate = { gate },
        configUrl = "https://cfg.example/api/app/config",
        bootstrapAppId = "com.arenaplus.ap01018",
    )

    @Test
    fun `STEP0 断网直接 NoNetwork 且不碰域名`() = runTest {
        val prober = FakeProber()
        val r = resolver(gate = false, prober = prober).resolve()
        assertEquals(ResolveResult.NoNetwork, r)
        // 闸门挡住：完全不拉取、不探测（这是「不乱换」的第一道保险）
        assertTrue(!prober.fetchCalled)
        assertTrue(prober.probedDomains.isEmpty())
    }

    @Test
    fun `STEP2 主域名命中即加载 拼接 palcode`() = runTest {
        val prober = FakeProber(hitDomain = "https://main.com")
        val r = resolver(gate = true, prober = prober).resolve()
        assertTrue(r is ResolveResult.Loadable)
        r as ResolveResult.Loadable
        assertEquals("https://main.com", r.domain)
        assertEquals("https://main.com/?palcode=palX", r.url)
    }

    @Test
    fun `ADR-0009 拉配置用 appId 加载 URL 仍用 palcode`() = runTest {
        val prober = FakeProber(hitDomain = "https://main.com")
        val r = resolver(
            gate = true, prober = prober,
            appId = "com.arenaplus.ap01018", palCode = "palX",
        ).resolve()
        // 解析键 = applicationId（appId），不是 palcode
        assertEquals("com.arenaplus.ap01018", prober.fetchedAppId)
        // 加载 URL 仍用 PAL_CODE 拼 /?palcode=
        assertEquals("https://main.com/?palcode=palX", (r as ResolveResult.Loadable).url)
    }

    @Test
    fun `STEP2 主域名挂 备用命中`() = runTest {
        val prober = FakeProber(hitDomain = "https://b2.com")
        val st = store()
        val r = resolver(gate = true, prober = prober, store = st).resolve()
        assertTrue(r is ResolveResult.Loadable)
        assertEquals("https://b2.com", (r as ResolveResult.Loadable).domain)
        // 主域名 main 先探（未命中）→ 备用并发 failover 命中 b2
        assertTrue(prober.probedDomains.contains("https://main.com"))
    }

    @Test
    fun `被封但有网 全挂且中立探针通 判 ServiceDown 并上报`() = runTest {
        val prober = FakeProber(hitDomain = null, neutralUp = true)
        val r = resolver(gate = true, prober = prober).resolve()
        assertEquals(ResolveResult.ServiceDown, r)
        assertTrue("应上报告警", prober.reported)
        // ADR-0009：告警以 appId 标识渠道
        assertEquals("com.arenaplus.ap01018", prober.reportedAppId)
    }

    @Test
    fun `全挂且本机无网 中立探针不通 判 NoNetwork 不上报`() = runTest {
        val prober = FakeProber(hitDomain = null, neutralUp = false)
        val r = resolver(gate = true, prober = prober).resolve()
        assertEquals(ResolveResult.NoNetwork, r)
        assertTrue("不应归咎域名上报", !prober.reported)
    }

    @Test
    fun `实时拉取成功 用新清单并命中新域名`() = runTest {
        val fresh = DomainConfig(listOf("https://fresh.com"), "/healthz")
        val prober = FakeProber(hitDomain = "https://fresh.com", runtime = fresh)
        val st = store()
        val r = resolver(gate = true, prober = prober, store = st).resolve()
        assertTrue(r is ResolveResult.Loadable)
        assertEquals("https://fresh.com", (r as ResolveResult.Loadable).domain)
        // 成功拉取即写缓存（兜底自更新）
        assertEquals(listOf("https://fresh.com"), st.readCachedConfig()?.domains)
    }

    @Test
    fun `STEP1 无任何来源 走中立探针裁决`() = runTest {
        val prober = FakeProber(neutralUp = true)
        val emptyStore = store(cached = null, bootstrap = null)
        val r = resolver(gate = true, prober = prober, store = emptyStore).resolve()
        // 无候选域名，但中立探针通 → 视为服务问题
        assertEquals(ResolveResult.ServiceDown, r)
    }
}
