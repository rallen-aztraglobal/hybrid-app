package com.hybrid.android.track

import android.content.Context
import android.net.Uri
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * BP 原始事件模式（[BpRawAdjustTracker]）的 JVM 单测：金额/币种解析、action→事件名解析、
 * H5 事件消费行为（register/updatecustomerid）、appSource 参数追加。
 *
 * 纯逻辑抽成可测函数，不依赖 Adjust SDK 真实初始化（测试用 flavor 的 ADJUST_APP_TOKEN 为空，
 * AdjustBootstrap.trackRaw 内部因查不到 token 会静默跳过，不会触碰真实 Adjust 网络请求）。
 *
 * 用 Robolectric 运行：依赖 android.net.Uri / SharedPreferences 真实实现。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class BpRawAdjustTrackerTest {

    // ---------------------------- 金额 / 币种解析 ----------------------------

    @Test
    fun `金额解析 千分位逗号`() {
        assertEquals(1000.50, BpRawAdjustTracker.parseAmount("1,000.50")!!, 1e-9)
    }

    @Test
    fun `金额解析 货币符号前缀`() {
        assertEquals(1000.0, BpRawAdjustTracker.parseAmount("₱1000")!!, 1e-9)
    }

    @Test
    fun `金额解析 尾随货币字母与空格`() {
        assertEquals(100.5, BpRawAdjustTracker.parseAmount("100.5 PHP")!!, 1e-9)
    }

    @Test
    fun `金额解析 前缀货币字母与空格`() {
        assertEquals(100.0, BpRawAdjustTracker.parseAmount("PHP 100")!!, 1e-9)
    }

    @Test
    fun `金额解析 null 或非法或负数返回 null`() {
        assertNull(BpRawAdjustTracker.parseAmount(null))
        assertNull(BpRawAdjustTracker.parseAmount(""))
        assertNull(BpRawAdjustTracker.parseAmount("abc"))
        assertNull(BpRawAdjustTracker.parseAmount("-100"))
    }

    @Test
    fun `金额解析 科学计数法不被误判`() {
        // 曾经的坑：filter 数字/./- 会把 1e3 清成 13，误报成收入 13。
        assertNull(BpRawAdjustTracker.parseAmount("1e3"))
    }

    @Test
    fun `金额解析 欧式千分位小数格式拒绝`() {
        assertNull(BpRawAdjustTracker.parseAmount("1.000,50"))
    }

    @Test
    fun `金额解析 尾随符号拒绝`() {
        assertNull(BpRawAdjustTracker.parseAmount("100-"))
    }

    @Test
    fun `币种解析 合法三字母转大写`() {
        assertEquals("PHP", BpRawAdjustTracker.parseCurrency("php"))
        assertEquals("USD", BpRawAdjustTracker.parseCurrency(" usd "))
    }

    @Test
    fun `币种解析 缺失或不合法回落 PHP`() {
        assertEquals("PHP", BpRawAdjustTracker.parseCurrency(null))
        assertEquals("PHP", BpRawAdjustTracker.parseCurrency(""))
        assertEquals("PHP", BpRawAdjustTracker.parseCurrency("US"))
        assertEquals("PHP", BpRawAdjustTracker.parseCurrency("USDT1"))
    }

    // ---------------------------- action → 事件名解析 ----------------------------

    @Test
    fun `deposit 固定映射 ad_deposit`() {
        assertEquals("ad_deposit", BpRawAdjustTracker.resolveEventName("deposit") { null })
    }

    // lookup 契约与 AdjustBootstrap.findEventName 一致：命中返回事件表里的 name（原始大小写），
    // 不是 token；这里用一个「已配置事件 name 集合」模拟事件表，忽略大小写匹配。
    private fun fakeLookup(vararg configuredNames: String): (String) -> String? = { candidate ->
        configuredNames.firstOrNull { it.equals(candidate, ignoreCase = true) }
    }

    @Test
    fun `原名命中事件表`() {
        assertEquals("web_login", BpRawAdjustTracker.resolveEventName("web_login", fakeLookup("web_login")))
    }

    @Test
    fun `原名未命中回退 ad_ 前缀`() {
        assertEquals(
            "ad_web_login",
            BpRawAdjustTracker.resolveEventName("web_login", fakeLookup("ad_web_login"))
        )
    }

    @Test
    fun `两级都未命中返回 null`() {
        assertNull(BpRawAdjustTracker.resolveEventName("nonexistent") { null })
    }

    @Test
    fun `空白 action 返回 null`() {
        assertNull(BpRawAdjustTracker.resolveEventName("") { "x" })
    }

    // ---------------------------- H5 事件消费行为 ----------------------------

    @Test
    fun `非 adjusth5event scheme 不消费`() {
        assertFalse(BpRawAdjustTracker.handleH5Url(Uri.parse("https://example.com/deposit")))
    }

    @Test
    fun `updatecustomerid 消费且不区分大小写`() {
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse("adjustH5event://updateCustomerId?customerId=42")))
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse("ADJUSTH5EVENT://updatecustomerid?customerId=43")))
    }

    @Test
    fun `updatecustomerid 持久化到 SharedPreferences`() {
        val context = RuntimeEnvironment.getApplication()
        BpRawAdjustTracker.init(context)
        BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://updatecustomerid?customerId=abc123"))
        val prefs = context.getSharedPreferences("adjust_bp_raw", Context.MODE_PRIVATE)
        assertEquals("abc123", prefs.getString("customer_id", null))
    }

    @Test
    fun `updatecustomerid 空值清除缓存`() {
        val context = RuntimeEnvironment.getApplication()
        BpRawAdjustTracker.init(context)
        BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://updatecustomerid?customerId=abc123"))
        BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://updatecustomerid"))
        val prefs = context.getSharedPreferences("adjust_bp_raw", Context.MODE_PRIVATE)
        assertNull(prefs.getString("customer_id", null))
    }

    @Test
    fun `register 消费并绑定 customerId`() {
        val context = RuntimeEnvironment.getApplication()
        BpRawAdjustTracker.init(context)
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://register?customerId=99&mobileNo=123")))
        val prefs = context.getSharedPreferences("adjust_bp_raw", Context.MODE_PRIVATE)
        assertEquals("99", prefs.getString("customer_id", null))
    }

    @Test
    fun `register 不带 customerId 时先置空缓存`() {
        val context = RuntimeEnvironment.getApplication()
        BpRawAdjustTracker.init(context)
        BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://updatecustomerid?customerId=old"))
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://register?mobileNo=123")))
        val prefs = context.getSharedPreferences("adjust_bp_raw", Context.MODE_PRIVATE)
        assertNull(prefs.getString("customer_id", null))
    }

    @Test
    fun `未知 action 仍消费`() {
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://someUnknownAction?x=1")))
    }

    @Test
    fun `deposit 事件即使未配 token 也消费`() {
        val url = "adjusth5event://deposit?amount=1%2C000.50&currency=php&orderId=X1&customerId=7"
        assertTrue(BpRawAdjustTracker.handleH5Url(Uri.parse(url)))
    }

    // ---------------------------- appSource 追加 ----------------------------

    @Test
    fun `appSource 追加到已有 query 之后`() {
        assertEquals(
            "https://a.com/?palcode=1&appSource=com.x.y",
            BpRawAdjustTracker.appendAppSource("https://a.com/?palcode=1", "com.x.y")
        )
    }

    @Test
    fun `appSource 无 query 时用问号追加`() {
        assertEquals("https://a.com/promo?appSource=com.x.y", BpRawAdjustTracker.appendAppSource("https://a.com/promo", "com.x.y"))
    }

    @Test
    fun `appSource 已存在则不重复`() {
        val url = "https://a.com/?appSource=com.x.y&palcode=1"
        assertEquals(url, BpRawAdjustTracker.appendAppSource(url, "com.x.y"))
    }

    @Test
    fun `withCustomerId 已有则不覆盖 缺失则补缓存值`() {
        val context = RuntimeEnvironment.getApplication()
        BpRawAdjustTracker.init(context)
        BpRawAdjustTracker.handleH5Url(Uri.parse("adjusth5event://updatecustomerid?customerId=cached"))
        assertEquals(mapOf("customerId" to "u1"), BpRawAdjustTracker.withCustomerId(mapOf("customerId" to "u1")))
        assertEquals(mapOf("a" to "1", "customerId" to "cached"), BpRawAdjustTracker.withCustomerId(mapOf("a" to "1")))
    }
}
