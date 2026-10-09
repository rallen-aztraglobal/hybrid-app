package com.hybrid.android.popup

import org.json.JSONArray
import org.json.JSONObject

/** 位置码（契约 §1）。[overlay]=遮罩类，同一时刻只允许一个；[rank]=遮罩互斥平手时 P4 > P7 > P1 > P6。 */
enum class PopupPosition(val code: String, val overlay: Boolean, val rank: Int) {
    P1("P1", true, 2),
    P2("P2", false, 9),
    P3("P3", false, 9),
    P4("P4", true, 0),
    P5("P5", false, 9),
    P6("P6", true, 3),
    P7("P7", true, 1),
    P8("P8", false, 9);

    companion object {
        /** 未知位置码返回 null（契约 §1：忽略、不崩）。 */
        fun fromCode(code: String?): PopupPosition? = values().firstOrNull { it.code == code }
    }
}

data class PopupCard(
    val id: Long,
    val imageUrl: String,
    val linkUrl: String,
    val buttonText: String,
    val title: String,
    val description: String,
)

data class Popup(
    val id: Long,
    val position: PopupPosition,
    val priority: Int,
    val startAt: Long,
    val endAt: Long,
    val minVersionCode: Int,
    val maxVersionCode: Int,
    val userType: String,
    val openMode: String,
    val closable: Boolean,
    val maskClosable: Boolean,
    val countdown: Boolean,
    val tabEnabled: Boolean,
    val tabIconUrl: String,
    val tabText: String,
    val autoplaySeconds: Int,
    val resumeGapMinutes: Int,
    val badge: Boolean,
    val cards: List<PopupCard>,
    /** 原始 JSON，用于把「按弹窗粒度选中的版本」整体落盘。 */
    val raw: String = "",
) {
    /** P4 强制模式：无 X、遮罩无效、返回键不关弹窗。 */
    val forced: Boolean get() = position == PopupPosition.P4 && !closable

    /** 便条图标：tabIconUrl，空则第一张卡片图。 */
    val tabIcon: String get() = tabIconUrl.ifBlank { cards.firstOrNull()?.imageUrl.orEmpty() }

    /** 需要预热 / 就绪才可参与的全部图片 URL（卡片图 + 便条图标）。 */
    fun assetUrls(): List<String> {
        val urls = LinkedHashSet<String>()
        cards.forEach { if (it.imageUrl.isNotBlank()) urls.add(it.imageUrl) }
        if ((tabEnabled || position == PopupPosition.P8) && tabIcon.isNotBlank()) urls.add(tabIcon)
        return urls.toList()
    }

    /** 当前位置是否支持收起为便条（契约 §1：P1/P3/P4 非强制/P7）。 */
    val supportsTab: Boolean
        get() = tabEnabled && !forced &&
            (position == PopupPosition.P1 || position == PopupPosition.P3 ||
                position == PopupPosition.P4 || position == PopupPosition.P7)
}

data class PopupConfig(
    val appId: String,
    val configVersion: String,
    val serverTime: Long,
    val tzOffsetMinutes: Int,
    val popups: List<Popup>,
) {
    /** 序列化成与 §4.1 同形的 JSON（popups 用各自 raw），便于缓存与再解析。 */
    fun toJson(): String = JSONObject().apply {
        put("appId", appId)
        put("configVersion", configVersion)
        put("serverTime", serverTime)
        put("tzOffsetMinutes", tzOffsetMinutes)
        put("popups", JSONArray().also { arr -> popups.forEach { p -> runCatching { arr.put(JSONObject(p.raw)) } } })
    }.toString()
}

object PopupConfigParser {
    /** 解析契约 §4.1 响应；非法 JSON 返回 null；未知位置码 / 无卡片的弹窗被忽略。 */
    fun parse(text: String?): PopupConfig? {
        if (text.isNullOrBlank()) return null
        return try {
            val o = JSONObject(text)
            val arr = o.optJSONArray("popups") ?: JSONArray()
            val list = ArrayList<Popup>()
            for (i in 0 until arr.length()) {
                val po = arr.optJSONObject(i) ?: continue
                runCatching { parsePopup(po) }.getOrNull()?.let { list.add(it) }
            }
            PopupConfig(
                appId = o.s("appId"),
                configVersion = o.s("configVersion"),
                serverTime = o.optLong("serverTime", 0L),
                tzOffsetMinutes = o.optInt("tzOffsetMinutes", 0),
                popups = list,
            )
        } catch (_: Exception) {
            null
        }
    }

    private fun parsePopup(o: JSONObject): Popup? {
        val pos = PopupPosition.fromCode(o.s("position")) ?: return null
        val cardsArr = o.optJSONArray("cards") ?: return null
        val cards = ArrayList<PopupCard>()
        for (i in 0 until cardsArr.length()) {
            val c = cardsArr.optJSONObject(i) ?: continue
            cards.add(
                PopupCard(
                    id = c.optLong("id"),
                    imageUrl = c.s("imageUrl"),
                    linkUrl = c.s("linkUrl"),
                    buttonText = c.s("buttonText"),
                    title = c.s("title"),
                    description = c.s("description"),
                )
            )
        }
        if (cards.isEmpty()) return null
        val closableRaw = o.optBoolean("closable", true)
        val forced = pos == PopupPosition.P4 && !closableRaw
        return Popup(
            id = o.optLong("id"),
            position = pos,
            priority = o.optInt("priority", 0),
            startAt = o.optLong("startAt", 0L),
            endAt = o.optLong("endAt", 0L),
            minVersionCode = o.optInt("minVersionCode", 0),
            maxVersionCode = o.optInt("maxVersionCode", 0),
            userType = o.s("userType").ifBlank { "all" },
            openMode = o.s("openMode").ifBlank { "webview" },
            // 客户端防御性归一化（服务端已按矩阵归一化过一次）。
            closable = if (pos == PopupPosition.P4) closableRaw else true,
            maskClosable = !forced && o.optBoolean("maskClosable", false),
            countdown = !forced && (pos == PopupPosition.P1 || pos == PopupPosition.P7) && o.optBoolean("countdown", false),
            tabEnabled = !forced && o.optBoolean("tabEnabled", false),
            tabIconUrl = o.s("tabIconUrl"),
            tabText = o.s("tabText"),
            autoplaySeconds = o.optInt("autoplaySeconds", 5).coerceIn(2, 30),
            resumeGapMinutes = o.optInt("resumeGapMinutes", 30).coerceIn(5, 1440),
            badge = o.optBoolean("badge", false),
            cards = cards,
            raw = o.toString(),
        )
    }
}

/** null 安全取字符串（org.json 的 optString 对 JSON null 会返回 "null"）。 */
internal fun JSONObject.s(key: String): String = if (isNull(key)) "" else optString(key, "").trim()
