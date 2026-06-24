package com.hybrid.android.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.io.IOException
import java.io.InterruptedIOException
import java.net.ConnectException
import java.net.SocketTimeoutException
import java.net.UnknownHostException
import javax.net.ssl.SSLException
import javax.net.ssl.SSLPeerUnverifiedException
// 注：证书匹配规则用 namesCoverHost 纯逻辑测，无需生成真实 X509 证书。

/**
 * STEP2 错误分类（docs/admin/02-domain-failover.md「3. 错误类型与动作映射」）的 JVM 单测。
 * 校验异常 → ProbeError 的映射正确，保证容灾「试下一个 vs 命中」的判定可靠。
 *
 * 用 Robolectric 运行：bodyMatchesSite 依赖 org.json，纯 JVM 桩会抛 NPE。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ProbeClassificationTest {

    @Test
    fun `DNS 解析失败 归类 DNS_FAIL`() {
        assertEquals(ProbeError.DNS_FAIL, DomainProber.classify(UnknownHostException("no such host")))
    }

    @Test
    fun `连接超时 归类 TIMEOUT`() {
        assertEquals(ProbeError.TIMEOUT, DomainProber.classify(SocketTimeoutException("timeout")))
    }

    @Test
    fun `连接被拒 归类 CONN_REFUSED`() {
        assertEquals(ProbeError.CONN_REFUSED, DomainProber.classify(ConnectException("refused")))
    }

    @Test
    fun `TLS 证书不符 归类 TLS_FAIL`() {
        assertEquals(ProbeError.TLS_FAIL, DomainProber.classify(SSLPeerUnverifiedException("bad cert")))
        assertEquals(ProbeError.TLS_FAIL, DomainProber.classify(SSLException("handshake")))
    }

    @Test
    fun `OkHttp callTimeout 抛 InterruptedIOException timeout 归类 TIMEOUT`() {
        assertEquals(ProbeError.TIMEOUT, DomainProber.classify(InterruptedIOException("timeout")))
    }

    @Test
    fun `其它 IOException 归类 OTHER`() {
        assertEquals(ProbeError.OTHER, DomainProber.classify(IOException("broken pipe")))
    }

    @Test
    fun `host 解析`() {
        assertEquals("arenaplus.ph", DomainProber.hostOf("https://arenaplus.ph"))
        assertEquals("arenaplus.ph", DomainProber.hostOf("arenaplus.ph"))
        assertEquals("a.example.com", DomainProber.hostOf("https://a.example.com/healthz"))
    }

    @Test
    fun `业务特征校验 ok true 且 brand 匹配才命中`() {
        assertTrue(DomainProber.bodyMatchesSite("""{"ok":true,"brand":"ap","v":1}""", "ap"))
        assertTrue(DomainProber.bodyMatchesSite("""{"ok":true,"brand":"AP"}""", "ap")) // 不区分大小写
        // ok=false → 不命中
        assertFalse(DomainProber.bodyMatchesSite("""{"ok":false,"brand":"ap"}""", "ap"))
        // brand 不符（劫持到别家站点）→ 不命中
        assertFalse(DomainProber.bodyMatchesSite("""{"ok":true,"brand":"gp"}""", "ap"))
        // 门户页/广告页返回的非 JSON 200 → 不命中（关键：可达≠是我们）
        assertFalse(DomainProber.bodyMatchesSite("<html>portal login</html>", "ap"))
        assertFalse(DomainProber.bodyMatchesSite(null, "ap"))
        assertFalse(DomainProber.bodyMatchesSite("", "ap"))
        // brand 字段缺失则只校验 ok（向后兼容）
        assertTrue(DomainProber.bodyMatchesSite("""{"ok":true}""", "ap"))
    }

    @Test
    fun `证书名称覆盖 host 的匹配规则`() {
        val names = listOf("arenaplus.ph", "*.arenaplus.ph")
        assertTrue(DomainProber.namesCoverHost(names, "arenaplus.ph"))
        assertTrue(DomainProber.namesCoverHost(names, "www.arenaplus.ph")) // *.arenaplus.ph
        assertFalse(DomainProber.namesCoverHost(names, "evil.com"))
        assertFalse(DomainProber.namesCoverHost(names, "a.b.arenaplus.ph")) // 通配符只覆盖一层
        // 大小写不敏感
        assertTrue(DomainProber.namesCoverHost(listOf("ArenaPlus.PH"), "arenaplus.ph"))
    }
}
