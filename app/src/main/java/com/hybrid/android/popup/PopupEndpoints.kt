package com.hybrid.android.popup

/** 端点派生，规则同 DeviceInfoRegistrar：`…/api/app/config` → `…/api/app/popups`。 */
object PopupEndpoints {
    fun popupsBase(configUrl: String): String? {
        val u = configUrl.trim().trimEnd('/')
        if (u.isBlank()) return null
        return if (u.endsWith("/api/app/config")) u.removeSuffix("/config") + "/popups"
        else u.substringBeforeLast('/') + "/popups"
    }

    fun configUrl(configUrl: String, appId: String): String? =
        popupsBase(configUrl)?.let { "$it?appId=${java.net.URLEncoder.encode(appId, "UTF-8")}" }

    fun eventsUrl(configUrl: String): String? = popupsBase(configUrl)?.let { "$it/events" }
}
