package com.hybrid.android.brand

import android.content.Context
import android.webkit.WebView

/**
 * 品牌策略回调宿主：由 [com.hybrid.android.WebViewActivity] 实现，
 * 让各品牌策略无需直接依赖 Activity 即可操作 WebView、发送事件、读写状态。
 */
interface BrandHost {
    /** Activity Context（AppsFlyer SDK 调用、startActivity、SharedPreferences 等） */
    val context: Context

    /** 当前品牌的站点域名（BuildConfig.DOMAIN） */
    val domain: String

    /** WebView 最近一次访问的 URL */
    val currentPath: String?

    /** 承载页面的 WebView */
    val webView: WebView

    /** 累加 AppsFlyer 事件参数（如 mobileNo / customerId） */
    fun putEventValue(key: String, value: Any)

    /**
     * 站点加载 URL 的统一「装饰」钩子：所有会 `loadUrl` 到站点的地方（首屏加载、运行中容灾重试、
     * 策略内部强刷页面等）都应经过这里，而不是各自裸调 `webView.loadUrl(url)`。
     * 默认恒等（原样返回），由 [com.hybrid.android.WebViewActivity] 按需覆写
     * （如 BP 原始事件模式下追加 `appSource` 参数，见 [com.hybrid.android.track.BpRawAdjustTracker]），
     * 这样新增「加载入口」时不会漏掉这类全局装饰。
     */
    fun decorateLoadUrl(url: String): String = url

    /** 发送一个 AppsFlyer 事件（携带已累加的事件参数） */
    fun sendAFEvent(eventName: String)

    /** 显示 Toast */
    fun showToast(text: String)
}
