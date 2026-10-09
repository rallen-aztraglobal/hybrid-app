package com.hybrid.android.popup

import android.content.SharedPreferences

/**
 * 弹窗模块的极简键值存储抽象：频控 / 便条 / 可信时间 / 埋点队列都走它，
 * 线上用 SharedPreferences（[PrefsKv]），单测用 [MemoryKv]，纯逻辑不依赖 Android。
 */
interface KvStore {
    fun getString(key: String): String?
    fun putString(key: String, value: String?)
    fun getLong(key: String, default: Long = 0L): Long
    fun putLong(key: String, value: Long)
}

class MemoryKv : KvStore {
    private val map = HashMap<String, Any>()
    @Synchronized override fun getString(key: String): String? = map[key] as? String
    @Synchronized override fun putString(key: String, value: String?) {
        if (value == null) map.remove(key) else map[key] = value
    }
    @Synchronized override fun getLong(key: String, default: Long): Long = (map[key] as? Long) ?: default
    @Synchronized override fun putLong(key: String, value: Long) { map[key] = value }
}

class PrefsKv(private val prefs: SharedPreferences) : KvStore {
    override fun getString(key: String): String? = prefs.getString(key, null)
    override fun putString(key: String, value: String?) {
        prefs.edit().apply { if (value == null) remove(key) else putString(key, value) }.apply()
    }
    override fun getLong(key: String, default: Long): Long = prefs.getLong(key, default)
    override fun putLong(key: String, value: Long) { prefs.edit().putLong(key, value).apply() }
}
