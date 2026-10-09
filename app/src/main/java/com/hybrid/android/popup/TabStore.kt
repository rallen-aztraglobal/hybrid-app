package com.hybrid.android.popup

import org.json.JSONArray
import org.json.JSONObject

/**
 * 便条持久化条目（契约 §7.6）。P1/P4/P7 便条当日有效（[expireDay]），P3 便条 24h（[expireAt]）。
 * 独立 P8 不在此持久化——它由配置驱动，每次冷启动重新评估。
 */
data class TabEntry(val popupId: Long, val createdAt: Long, val expireDay: Long, val expireAt: Long) {
    fun isValid(today: Long, now: Long): Boolean =
        (expireDay < 0 || today <= expireDay) && (expireAt <= 0 || now < expireAt)
}

class TabStore(private val kv: KvStore, private val clock: TrustedClock, private val maxTabs: Int = MAX_TABS) {

    @Synchronized fun load(): List<TabEntry> {
        val today = clock.dayKey()
        val now = clock.now()
        return readAll().filter { it.isValid(today, now) }
    }

    /** 新增（同 id 则刷新为最新）；超过上限替换最旧。返回被挤掉的条目。 */
    @Synchronized fun add(p: Popup): List<TabEntry> {
        val now = clock.now()
        val entry = if (p.position == PopupPosition.P3)
            TabEntry(p.id, now, -1L, now + FrequencyPolicy.TAB_P3_TTL_MS)
        else TabEntry(p.id, now, clock.dayKey(), 0L)
        val list = ArrayList(load().filter { it.popupId != p.id })
        list.add(entry)
        val evicted = ArrayList<TabEntry>()
        while (list.size > maxTabs) evicted.add(list.removeAt(0))
        writeAll(list)
        return evicted
    }

    @Synchronized fun remove(popupId: Long) = writeAll(readAll().filter { it.popupId != popupId })

    @Synchronized fun contains(popupId: Long) = load().any { it.popupId == popupId }

    private fun readAll(): List<TabEntry> = runCatching {
        val a = JSONArray(kv.getString(KEY) ?: "[]")
        (0 until a.length()).mapNotNull { i ->
            a.optJSONObject(i)?.let {
                TabEntry(it.optLong("id"), it.optLong("c"), it.optLong("d", -1L), it.optLong("e", 0L))
            }
        }
    }.getOrDefault(emptyList())

    private fun writeAll(list: List<TabEntry>) {
        val a = JSONArray()
        list.forEach {
            a.put(JSONObject().put("id", it.popupId).put("c", it.createdAt).put("d", it.expireDay).put("e", it.expireAt))
        }
        kv.putString(KEY, a.toString())
    }

    companion object {
        const val MAX_TABS = 2
        private const val KEY = "tabs"
    }
}
