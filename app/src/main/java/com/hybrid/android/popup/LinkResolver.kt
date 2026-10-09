package com.hybrid.android.popup

/** 点击后的动作（契约 §7.7）。 */
sealed class LinkAction {
    /**
     * 主 WebView 加载（最终 URL 尚未经 decorateLoadUrl）。
     * [external]=true 表示 H5 给的绝对 http(s) 地址：需先过 BrandStrategy.shouldOverrideUrl（与 H5 内链接一致）。
     */
    data class LoadInWebView(val url: String, val external: Boolean = false) : LinkAction()
    data class OpenBrowser(val url: String) : LinkAction()
    /** 商店：优先 [url]（market:// 或 https），失败回落 [fallbackUrl]（浏览器）。 */
    data class OpenStore(val url: String, val fallbackUrl: String?) : LinkAction()
}

object LinkResolver {

    private fun isHttp(l: String) = l.startsWith("http://", true) || l.startsWith("https://", true)
    private fun isRelative(l: String) = l.startsWith("/") && !l.startsWith("//")
    private fun isMarket(l: String) = l.startsWith("market://", true)

    /** 链接对该 openMode 是否可点；空或非法 → false（整图不可点，仅可关闭）。 */
    fun isClickable(openMode: String, link: String): Boolean = resolve(openMode, link, "https://x", "") != null

    fun resolve(openMode: String, link: String, domain: String, palcode: String): LinkAction? {
        val l = link.trim()
        if (l.isEmpty()) return null
        return when (openMode) {
            "browser" -> if (isHttp(l)) LinkAction.OpenBrowser(l) else null
            "store" -> when {
                isMarket(l) -> LinkAction.OpenStore(l, marketToHttps(l))
                isHttp(l) -> LinkAction.OpenStore(l, l)
                else -> null
            }
            else -> when { // webview（含未知取值按 webview 处理）
                isRelative(l) -> LinkAction.LoadInWebView(appendPalcode(domain.trimEnd('/') + l, palcode))
                isHttp(l) -> LinkAction.LoadInWebView(l, external = true)
                else -> null
            }
        }
    }

    /** 同推送 deeplink：追加 palcode（path 已有 query 时用 &，已带 palcode 不重复）。 */
    fun appendPalcode(url: String, palcode: String): String {
        if (palcode.isBlank() || Regex("[?&]palcode=").containsMatchIn(url)) return url
        val hashIdx = url.indexOf('#')
        val base = if (hashIdx >= 0) url.substring(0, hashIdx) else url
        val frag = if (hashIdx >= 0) url.substring(hashIdx) else ""
        val sep = if (base.contains('?')) "&" else "?"
        return "$base${sep}palcode=$palcode$frag"
    }

    private fun marketToHttps(l: String): String? {
        val rest = l.substring("market://".length)
        return if (rest.isBlank()) null else "https://play.google.com/store/apps/$rest"
    }
}
