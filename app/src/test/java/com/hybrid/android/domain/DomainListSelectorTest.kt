package com.hybrid.android.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * STEP1 三级取用（实时拉取 → 缓存 → 编译期兜底）+ lastGood 提队首 的 JVM 单测。
 * 用内存版 [ConfigStore] 替身，脱离 Android 存储；但 toJson/parse 往返依赖 org.json，
 * 故用 Robolectric 提供真实 org.json 实现。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class DomainListSelectorTest {

    /** 内存版 ConfigStore，记录写入便于断言「成功即更新缓存」。 */
    private class FakeStore(
        var cached: DomainConfig? = null,
        var bootstrap: BootstrapConfig? = null,
    ) : ConfigStore {
        var writeCount = 0
        override fun readCachedConfig() = cached
        override fun writeCachedConfig(config: DomainConfig) { cached = config; writeCount++ }
        override fun readBootstrap() = bootstrap
    }

    /** 兜底配置构造助手：带 appId（ADR-0009），避免到处写位置参数。 */
    private fun boot(
        domains: List<String>,
        probePath: String = "/healthz",
    ) = BootstrapConfig(
        brand = "ap",
        appId = "com.arenaplus.ap01018",
        palcode = "p",
        configUrl = "https://cfg",
        probePath = probePath,
        defaultDomains = domains,
    )

    @Test
    fun `实时拉取成功 使用返回清单 并立即写缓存`() {
        val store = FakeStore(
            cached = DomainConfig(listOf("https://old.com"), "/healthz"),
            bootstrap = boot(listOf("https://boot.com")),
        )
        val selector = DomainListSelector(store)
        val fresh = DomainConfig(listOf("https://fresh1.com", "https://fresh2.com"), "/hz")

        val sel = selector.select { fresh }

        assertEquals(listOf("https://fresh1.com", "https://fresh2.com"), sel.domains)
        assertEquals("/hz", sel.probePath)
        // 成功即自更新兜底缓存（覆盖旧值）
        assertEquals(1, store.writeCount)
        assertEquals(fresh.domains, store.cached?.domains)
    }

    @Test
    fun `实时拉取失败 降级到本地缓存 不写缓存`() {
        val store = FakeStore(
            cached = DomainConfig(listOf("https://cached1.com", "https://cached2.com"), "/cz"),
            bootstrap = boot(listOf("https://boot.com")),
        )
        val selector = DomainListSelector(store)

        val sel = selector.select { null } // 拉取失败

        assertEquals(listOf("https://cached1.com", "https://cached2.com"), sel.domains)
        assertEquals("/cz", sel.probePath)
        assertEquals(0, store.writeCount) // 失败不写
    }

    @Test
    fun `从未成功过 用编译期兜底`() {
        val store = FakeStore(
            cached = null,
            bootstrap = boot(listOf("https://boot1.com", "https://boot2.com"), probePath = "/bz"),
        )
        val selector = DomainListSelector(store)

        val sel = selector.select { null }

        assertEquals(listOf("https://boot1.com", "https://boot2.com"), sel.domains)
        assertEquals("/bz", sel.probePath)
    }

    @Test
    fun `三级全空 返回空清单`() {
        val store = FakeStore(cached = null, bootstrap = null)
        val sel = DomainListSelector(store).select { null }
        assertTrue(sel.domains.isEmpty())
    }

    @Test
    fun `顺序以来源为准 主在队首 不做重排`() {
        val store = FakeStore(
            cached = DomainConfig(listOf("https://main.com", "https://b1.com", "https://b2.com"), "/healthz"),
        )
        val sel = DomainListSelector(store).select { null }
        // 不再按 lastGood 重排：主域名 main 保持在队首，原序不变。
        assertEquals(listOf("https://main.com", "https://b1.com", "https://b2.com"), sel.domains)
    }

    @Test
    fun `空 domains 的实时配置 视为失败 落到缓存`() {
        val store = FakeStore(cached = DomainConfig(listOf("https://c.com"), "/healthz"))
        // parse 不会产出空 domains 的 DomainConfig，但防御性：select 对空 domains 的 fresh 也降级
        val sel = DomainListSelector(store).select { DomainConfig(emptyList(), "/x") }
        assertEquals(listOf("https://c.com"), sel.domains)
        assertEquals(0, store.writeCount)
    }

    @Test
    fun `缓存往返 toJson 与 parse 自洽`() {
        val cfg = DomainConfig(listOf("https://a.com", "https://b.com"), "/healthz", 42)
        val back = DomainConfig.parse(cfg.toJson())
        assertEquals(cfg.domains, back?.domains)
        assertEquals(cfg.probePath, back?.probePath)
        assertEquals(42, back?.configVersion)
    }

    @Test
    fun `bootstrap 解析空体返回 null`() {
        assertNull(BootstrapConfig.parse("not json"))
    }
}
