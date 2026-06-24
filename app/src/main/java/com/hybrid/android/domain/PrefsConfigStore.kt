package com.hybrid.android.domain

import android.content.Context
import android.util.Log
import androidx.core.content.edit

/**
 * [ConfigStore] 的 Android 实现：缓存落 SharedPreferences，编译期兜底读 `assets/bootstrap.json`。
 */
class PrefsConfigStore(private val context: Context) : ConfigStore {

    private val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    // bootstrap 只解析一次（assets 编译期固定，进程内不变）。
    private val bootstrap: BootstrapConfig? by lazy {
        runCatching {
            context.assets.open(BOOTSTRAP_ASSET).bufferedReader().use { it.readText() }
        }.mapCatching { BootstrapConfig.parse(it) }
            .onFailure { Log.w(TAG, "读取 assets/$BOOTSTRAP_ASSET 失败: ${it.message}") }
            .getOrNull()
    }

    override fun readCachedConfig(): DomainConfig? {
        val raw = prefs.getString(KEY_CACHED, null)?.takeIf { it.isNotBlank() } ?: return null
        return DomainConfig.parse(raw)
    }

    override fun writeCachedConfig(config: DomainConfig) {
        prefs.edit { putString(KEY_CACHED, config.toJson()) }
    }

    override fun readLastGood(): String? =
        prefs.getString(KEY_LAST_GOOD, null)?.takeIf { it.isNotBlank() }

    override fun writeLastGood(domain: String) {
        prefs.edit { putString(KEY_LAST_GOOD, domain) }
    }

    override fun readBootstrap(): BootstrapConfig? = bootstrap

    /** configUrl / appId / palcode / brand 取用：编译期不变量，从 bootstrap 取（不在缓存里）。 */
    fun configUrl(): String = bootstrap?.configUrl.orEmpty()
    fun bootstrapAppId(): String = bootstrap?.appId.orEmpty()
    fun bootstrapPalcode(): String = bootstrap?.palcode.orEmpty()
    fun brand(): String = bootstrap?.brand.orEmpty()

    companion object {
        private const val TAG = "DomainResolver"
        private const val PREFS = "domain_resolver"
        private const val KEY_CACHED = "cached_config"
        private const val KEY_LAST_GOOD = "last_good"
        private const val BOOTSTRAP_ASSET = "bootstrap.json"
    }
}
