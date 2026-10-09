package com.hybrid.android.popup

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.json.JSONArray
import org.json.JSONObject

/** 批次级公共字段（契约 §4.2）。 */
data class BatchMeta(
    val appId: String,
    val palcode: String,
    val appVersion: String,
    val versionCode: Int,
)

/**
 * 埋点上报器：入队 → 10s 防抖 / onStop / 冷启动 / 网络恢复触发 [flush]；
 * 2xx 出队，400 视为批次本身非法丢弃（避免队头永久阻塞），其余失败指数退避重试。
 *
 * @param send 发送函数，返回 HTTP 状态码；网络异常返回 -1。
 */
class PopupReporter(
    private val queue: EventQueue,
    private val scope: CoroutineScope,
    private val meta: BatchMeta,
    private val send: suspend (body: String) -> Int,
    private val debounceMs: Long = 10_000L,
    private val nowMs: () -> Long = System::currentTimeMillis,
) {
    private val mutex = Mutex()
    private var debounceJob: Job? = null
    private var retryJob: Job? = null
    @Volatile var failures = 0
        private set
    /** 退避期内（失败后）不允许防抖触发的 flush 提前重试；显式触发（onStop / 网络恢复）不受限。 */
    @Volatile private var nextAllowedAt = 0L

    fun record(event: JSONObject) {
        queue.add(event)
        scheduleDebounced()
    }

    fun scheduleDebounced() {
        debounceJob?.cancel()
        debounceJob = scope.launch {
            delay(debounceMs)
            if (failures > 0 && nowMs() < nextAllowedAt) return@launch // 不绕过退避
            // 另起独立 Job：之后新事件只会重置防抖计时（cancel 的是这个已结束的 Job），不会取消进行中的发送
            scope.launch { flush() }
        }
    }

    /** 立即触发一次（onStop / 冷启动 / 网络恢复）。 */
    fun flushAsync() { scope.launch { flush() } }

    suspend fun flush() {
        retryJob?.cancel()
        doFlush()
    }

    private suspend fun doFlush() {
        mutex.withLock {
            while (true) {
                val batch = queue.nextBatch() ?: break
                val code = try {
                    send(buildBody(batch))
                } catch (e: CancellationException) {
                    throw e // 取消不是失败，不计 failures
                } catch (_: Exception) { -1 }
                when {
                    code in 200..299 -> { queue.ack(batch.id); failures = 0; nextAllowedAt = 0L }
                    isPermanent(code) -> { queue.ack(batch.id) } // 批次本身非法（400/413/422…），重试无意义
                    else -> { // 含 404（后端未部署时保留事件）、408/429、5xx、网络错误
                        failures++
                        val d = Backoff.delayMs(failures)
                        nextAllowedAt = nowMs() + d
                        scheduleRetry(d)
                        break
                    }
                }
            }
        }
    }

    /** 明确的永久性客户端错误；404 / 408 / 429 例外（可恢复）。 */
    private fun isPermanent(code: Int) = code in 400..499 && code != 404 && code != 408 && code != 429

    private fun scheduleRetry(ms: Long) {
        retryJob?.cancel()
        retryJob = scope.launch {
            delay(ms)
            doFlush()
        }
    }

    internal fun buildBody(batch: EventBatch): String = JSONObject().apply {
        put("batchId", batch.id)
        put("appId", meta.appId)
        put("palcode", meta.palcode)
        put("appVersion", meta.appVersion)
        put("versionCode", meta.versionCode)
        put("events", JSONArray().also { a -> batch.events.forEach { a.put(it) } })
    }.toString()
}

/** 事件构造（字段名严格按契约 §4.2）。 */
object PopupEvents {
    private fun base(name: String, p: Popup, ts: Long) = JSONObject()
        .put("event", name).put("popupId", p.id).put("position", p.position.code).put("ts", ts)

    fun trigger(p: Popup, ts: Long) = base("popup_trigger", p, ts)
    fun filtered(p: Popup, reason: String, ts: Long) = base("popup_filtered", p, ts).put("reason", reason)
    fun loadFail(p: Popup, ts: Long) = base("popup_load_fail", p, ts)
    fun impression(p: Popup, card: PopupCard, index: Int, display: String, ts: Long) =
        base("popup_impression", p, ts).put("cardId", card.id).put("cardIndex", index).put("display", display)
    fun click(p: Popup, card: PopupCard, index: Int, display: String, ts: Long) =
        base("popup_click", p, ts).put("cardId", card.id).put("cardIndex", index).put("display", display)
    fun close(p: Popup, method: String, ts: Long) = base("popup_close", p, ts).put("method", method)
    fun collapse(p: Popup, ts: Long) = base("popup_collapse", p, ts)
    fun tabImpression(p: Popup, ts: Long) = base("tab_impression", p, ts)
    fun tabClick(p: Popup, ts: Long) = base("tab_click", p, ts)
    fun tabDismiss(p: Popup, ts: Long) = base("tab_dismiss", p, ts)
    fun slide(p: Popup, from: Int, to: Int, ts: Long) = base("popup_slide", p, ts).put("from", from).put("to", to)
}
