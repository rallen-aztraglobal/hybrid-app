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

    /** 发送一个 AppsFlyer 事件（携带已累加的事件参数） */
    fun sendAFEvent(eventName: String)

    /** 显示 Toast */
    fun showToast(text: String)
}
