package com.hybrid.android.brand

import android.app.Activity
import android.net.Uri
import org.json.JSONObject

/**
 * 大渠道（品牌）策略插件。每个品牌一份实现，封装其差异化逻辑：
 * AppsFlyer 初始化、deeplink 处理、URL 跳转拦截、归因事件上报。
 *
 * 通过 [BrandStrategies.create] 按 BuildConfig.BRAND 选取实现。
 */
interface BrandStrategy {

    /**
     * AppsFlyer 初始化（品牌差异点：BP 采集 OAID，其它开启 debug log），
     * 并加载本品牌持久化的归因状态。在 Activity.onCreate 最早期调用。
     */
    fun initTracking(activity: Activity)

    /** Activity.onResume 回调，默认无操作。 */
    fun onResume(host: BrandHost) {}

    /** 系统 deeplink（onNewIntent 的 Intent.data），默认无操作。 */
    fun onDeepLinkIntent(uri: Uri, host: BrandHost) {}

    /**
     * WebView URL 跳转拦截（shouldOverrideUrlLoading）。
     * @return true 表示已自行处理（拦截），false 表示交给 WebView 继续加载。
     */
    fun shouldOverrideUrl(url: String, host: BrandHost): Boolean

    /**
     * 命中的 API 响应（已从拦截数据中解析出 response.body），按品牌上报归因事件。
     *
     * @param apiUrl   被拦截的接口 URL
     * @param fullJson 完整拦截数据原文（url/method/headers/body/response）
     * @param body     已解析的 response.body 对象
     */
    fun onApiResponse(apiUrl: String, fullJson: String, body: JSONObject, host: BrandHost)

    /**
     * 推送通知点击后回调（可选，默认空实现）。供各品牌定制点击归因（如 AppsFlyer 事件上报）。
     * 在 WebViewActivity 经 DomainResolver 拿到可用域名并拼出完整 URL 之后调用。
     *
     * @param path   推送携带的相对 deeplink path（如 `/promo/618`），不含域名（守 ADR-0002）
     * @param domain 当前 DomainResolver 解析出的可用域名（运行时值，非编译期常量）
     * @param host   宿主 Activity 的回调接口
     */
    fun onPushOpen(path: String, domain: String, host: BrandHost) {}
}
