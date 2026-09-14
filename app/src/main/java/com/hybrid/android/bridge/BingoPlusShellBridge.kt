package com.hybrid.android.bridge

import android.webkit.JavascriptInterface

/**
 * BP 原始事件模式专用的 JS 接口，注入名固定为 `BingoPlusShell`（与 BP 团队自家壳同名，
 * 见「马甲包 Adjust 归因流程代码说明」§4 / docs/admin/08-adjust.md §11.3）。
 *
 * H5 只会调用 `BingoPlusShell.openExternal(url)` 这一个方法：url 是 `adjusth5event://...`
 * 时由原生消费上报 Adjust 事件；否则作为外链交给系统打开。H5 靠加载 URL 上的
 * `appSource=<applicationId>` 参数识别「在壳内」，不靠探测本对象。
 *
 * 仅在 [com.hybrid.android.track.AdjustBootstrap.bpRawMode] 为 true 时注入；关闭时不存在此对象。
 * JavascriptInterface 方法运行在 WebView 的 JS 桥线程，UI 操作由回调方自行切主线程。
 */
class BingoPlusShellBridge(private val onOpenExternal: (String) -> Unit) {

    @JavascriptInterface
    fun openExternal(url: String?) {
        if (url.isNullOrBlank()) return
        onOpenExternal(url)
    }
}
