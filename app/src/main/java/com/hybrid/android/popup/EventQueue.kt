package com.hybrid.android.popup

import org.json.JSONArray
import org.json.JSONObject
import java.util.UUID
import java.util.concurrent.Executor
import java.util.concurrent.atomic.AtomicBoolean

/** 一个待发送 / 已固化的批次。 */
data class EventBatch(val id: String, val events: List<JSONObject>)

/**
 * 埋点持久化队列（契约 §7.9）：
 *  - 上限 500 条（含在途批次），超出丢最旧；
 *  - 每批 ≤ 100；发出前先把 batchId + 事件固化为「在途批次」，失败重试沿用同一批（服务端幂等）；
 *  - 2xx 后 [ack] 才出队。
 *
 * 队列与在途批次分开存储（线上是两个独立的 SharedPreferences 文件）、分开解析，一处损坏不连累另一处。
 * 入队的序列化与落盘由 [persistExecutor] 异步完成（主线程只做内存追加）；
 * 在途批次的固化在 [nextBatch]（上报协程，IO 线程）里同步写，保证「先持久化再发送」。
 */
class EventQueue(
    private val queueKv: KvStore,
    private val inflightKv: KvStore = queueKv,
    private val maxSize: Int = MAX,
    private val batchSize: Int = BATCH,
    private val persistExecutor: Executor = Executor { it.run() },
) {
    private val queue = ArrayList<JSONObject>()
    private var inflight: EventBatch? = null
    private var version = 0L
    private var written = 0L
    private val writeLock = Any()
    private val persistPending = AtomicBoolean(false)

    init {
        runCatching {
            queueKv.getString(K_QUEUE)?.let { s ->
                val a = JSONArray(s)
                for (i in 0 until a.length()) a.optJSONObject(i)?.let { queue.add(it) }
            }
        }
        runCatching {
            inflightKv.getString(K_INFLIGHT)?.let { s ->
                val o = JSONObject(s)
                val a = o.getJSONArray("events")
                val list = ArrayList<JSONObject>()
                for (i in 0 until a.length()) a.optJSONObject(i)?.let { list.add(it) }
                inflight = EventBatch(o.getString("id"), list)
            }
        }
    }

    @Synchronized fun size(): Int = queue.size + (inflight?.events?.size ?: 0)

    fun add(event: JSONObject) {
        synchronized(this) {
            queue.add(event)
            val overflow = size() - maxSize
            if (overflow > 0) repeat(minOf(overflow, queue.size)) { queue.removeAt(0) }
            version++
        }
        schedulePersist()
    }

    /** 取下一批：有在途批次则原样返回（同 batchId），否则从队首切一批并先（同步）持久化。 */
    fun nextBatch(): EventBatch? {
        val batch: EventBatch
        synchronized(this) {
            inflight?.let { return it }
            if (queue.isEmpty()) return null
            val take = queue.subList(0, minOf(batchSize, queue.size))
            batch = EventBatch(UUID.randomUUID().toString(), ArrayList(take))
            take.clear()
            inflight = batch
            version++
        }
        persistInflight(batch)
        persistQueueNow()
        return batch
    }

    fun ack(batchId: String) {
        val done = synchronized(this) {
            if (inflight?.id == batchId) { inflight = null; true } else false
        }
        if (done) persistInflight(null)
    }

    private fun schedulePersist() {
        if (persistPending.compareAndSet(false, true)) {
            persistExecutor.execute { persistPending.set(false); persistQueueNow() }
        }
    }

    private fun persistQueueNow() {
        val snapshot: List<JSONObject>
        val v: Long
        synchronized(this) { snapshot = ArrayList(queue); v = version }
        val str = JSONArray().also { a -> snapshot.forEach { a.put(it) } }.toString()
        synchronized(writeLock) {
            if (v >= written) { written = v; queueKv.putString(K_QUEUE, str) }
        }
    }

    private fun persistInflight(b: EventBatch?) {
        val str = b?.let {
            JSONObject().put("id", it.id).put("events", JSONArray().also { a -> it.events.forEach { e -> a.put(e) } }).toString()
        }
        synchronized(writeLock) { inflightKv.putString(K_INFLIGHT, str) }
    }

    companion object {
        const val MAX = 500
        const val BATCH = 100
        private const val K_QUEUE = "ev_queue"
        private const val K_INFLIGHT = "ev_inflight"
    }
}

object Backoff {
    const val CAP_MS = 5 * 60_000L
    /** 第 n 次连续失败后的等待：2s、4s、8s … 上限 5 分钟。 */
    fun delayMs(failures: Int): Long {
        if (failures <= 0) return 0L
        val shift = minOf(failures, 20)
        return minOf(CAP_MS, 1_000L shl shift)
    }
}
