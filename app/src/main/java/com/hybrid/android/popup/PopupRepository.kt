package com.hybrid.android.popup

import android.util.Log
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.withPermit
import java.io.File

/**
 * 配置仓库（契约 §7.1）：拉取 → 立即按「弹窗粒度」选用就绪版本 → 后台预热素材 → 再选一次 → 清理无引用素材。
 *
 *  - [effective] 是对外唯一的「在用配置」，落盘 effective.json；冷启动先读它（= 失败用缓存）。
 *  - 从未成功拉取且无缓存 → effective == null → 调用方不展示、不记事件。
 */
class PopupRepository(
    private val dir: File,
    private val clock: TrustedClock,
    private val cache: AssetCache,
    private val fetch: suspend () -> String?,
    private val elapsedMs: () -> Long,
) {
    @Volatile var effective: PopupConfig? = null
        private set

    private val _state = MutableStateFlow<PopupConfig?>(null)
    /** 在用配置的变更流（每次 publish 后发射），Manager 订阅后立即对账（下线已消失的弹窗 / 便条）。 */
    val state: StateFlow<PopupConfig?> = _state.asStateFlow()

    @Volatile private var lastSuccessElapsed: Long = NEVER
    private val mutex = Mutex()

    init {
        dir.mkdirs()
        effective = runCatching { PopupConfigParser.parse(File(dir, EFFECTIVE).takeIf { it.isFile }?.readText()) }.getOrNull()
        _state.value = effective
    }

    /** 回前台：距上次成功拉取 > 5 分钟（或本进程还没成功拉过）才重拉。 */
    fun needsForegroundRefresh(): Boolean =
        lastSuccessElapsed == NEVER || elapsedMs() - lastSuccessElapsed > REFRESH_INTERVAL_MS

    /** 拉取一次；失败返回 false，在用配置保持不变。 */
    suspend fun refresh(): Boolean = mutex.withLock {
        val text = try {
            fetch()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) { null }
        val cfg = PopupConfigParser.parse(text)
        if (cfg == null) {
            Log.w(TAG, "弹窗配置拉取失败，沿用缓存 effective=${effective?.configVersion}")
            return@withLock false
        }
        clock.recordAnchor(cfg.serverTime, cfg.tzOffsetMinutes)
        lastSuccessElapsed = elapsedMs()

        // 先立即选用一次（已就绪的新版本 / 沿用旧版本 / 下线已不存在的），不等下载。
        publish(cfg, ReadinessMerge.merge(cfg.popups, effective?.popups.orEmpty(), cache::isReady))

        val urls = cfg.popups.flatMap { it.assetUrls() }.toSet().filterNot(cache::isReady)
        if (urls.isNotEmpty()) prewarm(urls)

        val finalPopups = ReadinessMerge.merge(cfg.popups, effective?.popups.orEmpty(), cache::isReady)
        publish(cfg, finalPopups)
        cache.cleanup(ReadinessMerge.referencedUrls(cfg.popups, finalPopups))
        true
    }

    private suspend fun prewarm(urls: List<String>) = coroutineScope {
        val sem = Semaphore(3)
        urls.map { u -> async { sem.withPermit { cache.ensure(u, PREWARM_TIMEOUT_MS) } } }.awaitAll()
    }

    private fun publish(cfg: PopupConfig, popups: List<Popup>) {
        val eff = cfg.copy(popups = popups)
        effective = eff
        _state.value = eff
        runCatching {
            val tmp = File(dir, "$EFFECTIVE.tmp")
            tmp.writeText(eff.toJson())
            if (!tmp.renameTo(File(dir, EFFECTIVE))) { File(dir, EFFECTIVE).writeText(eff.toJson()); tmp.delete() }
        }
    }

    companion object {
        private const val TAG = "HybridPopup"
        private const val EFFECTIVE = "effective.json"
        private const val NEVER = Long.MIN_VALUE
        const val REFRESH_INTERVAL_MS = 5 * 60_000L
        const val PREWARM_TIMEOUT_MS = 20_000L
    }
}
