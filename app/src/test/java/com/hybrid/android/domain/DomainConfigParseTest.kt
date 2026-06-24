package com.hybrid.android.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * 运行时配置解析的 JVM 单测。重点是 APK 端的防御性过滤：
 * 线上一律 https（即便后台已校验，APK 再兜一道）、去重、去尾斜杠、最多 4 个。
 *
 * 用 Robolectric 运行：[DomainConfig.parse] / [BootstrapConfig.parse] 依赖 org.json，
 * 纯 JVM 桩会抛 NPE，Robolectric 提供真实 org.json 实现。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class DomainConfigParseTest {

    @Test
    fun `解析标准响应`() {
        val body = """
            {"palcode":"123","domains":["https://arenaplus.ph","https://arenaplus-cdn.com","https://ap-backup.net"],
             "probePath":"/healthz","configVersion":42,"ttlSeconds":600}
        """.trimIndent()
        val cfg = DomainConfig.parse(body)!!
        assertEquals(
            listOf("https://arenaplus.ph", "https://arenaplus-cdn.com", "https://ap-backup.net"),
            cfg.domains
        )
        assertEquals("/healthz", cfg.probePath)
        assertEquals(42, cfg.configVersion)
    }

    @Test
    fun `过滤非 https 域名`() {
        val body = """{"domains":["http://insecure.com","https://secure.com"]}"""
        val cfg = DomainConfig.parse(body)!!
        assertEquals(listOf("https://secure.com"), cfg.domains)
    }

    @Test
    fun `全为非 https 返回 null`() {
        assertNull(DomainConfig.parse("""{"domains":["http://a.com","http://b.com"]}"""))
    }

    @Test
    fun `去重并去掉尾部斜杠`() {
        val body = """{"domains":["https://a.com/","https://a.com","https://b.com/"]}"""
        val cfg = DomainConfig.parse(body)!!
        assertEquals(listOf("https://a.com", "https://b.com"), cfg.domains)
    }

    @Test
    fun `最多保留 4 个域名`() {
        val body = """{"domains":["https://1.com","https://2.com","https://3.com","https://4.com","https://5.com"]}"""
        val cfg = DomainConfig.parse(body)!!
        assertEquals(4, cfg.domains.size)
        assertEquals("https://1.com", cfg.domains.first())
    }

    @Test
    fun `缺 probePath 用默认`() {
        val cfg = DomainConfig.parse("""{"domains":["https://a.com"]}""")!!
        assertEquals("/healthz", cfg.probePath)
    }

    @Test
    fun `domains 缺失或空 返回 null`() {
        assertNull(DomainConfig.parse("""{"probePath":"/x"}"""))
        assertNull(DomainConfig.parse("""{"domains":[]}"""))
    }

    @Test
    fun `非法 JSON 返回 null`() {
        assertNull(DomainConfig.parse("garbage"))
        assertNull(DomainConfig.parse(""))
    }

    @Test
    fun `bootstrap 解析完整`() {
        val body = """
            {"schemaVersion":1,"brand":"ap","appId":"com.arenaplus.ap01018","palcode":"123",
             "configUrl":"https://cfg.example/api/app/config/",
             "probePath":"/healthz","defaultDomains":["https://arenaplus.ph","https://b.com"]}
        """.trimIndent()
        val boot = BootstrapConfig.parse(body)!!
        assertEquals("ap", boot.brand)
        // ADR-0009：bootstrap 带 appId（applicationId，域名解析键）
        assertEquals("com.arenaplus.ap01018", boot.appId)
        assertEquals("123", boot.palcode) // palcode 仅用于加载 URL
        assertEquals("https://cfg.example/api/app/config", boot.configUrl) // 去尾斜杠
        assertEquals(listOf("https://arenaplus.ph", "https://b.com"), boot.defaultDomains)
    }

    @Test
    fun `bootstrap 缺 appId 时为空串`() {
        // 向后兼容：旧 bootstrap 无 appId 字段 → 运行时回落 BuildConfig.APPLICATION_ID（见 DomainResolver.invoke）
        val boot = BootstrapConfig.parse(
            """{"brand":"ap","palcode":"1","configUrl":"https://c","defaultDomains":["https://a.com"]}"""
        )!!
        assertEquals("", boot.appId)
    }

    @Test
    fun `中立端点清单包含 gstatic 与 cloudflare`() {
        val urls = DomainProber.NEUTRAL_ENDPOINTS.map { it.first }
        assertTrue(urls.any { it.contains("gstatic.com") })
        assertTrue(urls.any { it.contains("cloudflare.com") })
    }
}
