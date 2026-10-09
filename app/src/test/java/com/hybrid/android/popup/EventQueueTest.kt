package com.hybrid.android.popup

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.runTest
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class EventQueueTest {
    private fun ev(i: Int) = JSONObject().put("event", "popup_trigger").put("popupId", i).put("ts", i.toLong())

    @Test fun capsAt500DroppingOldest() {
        val q = EventQueue(MemoryKv())
        repeat(520) { q.add(ev(it)) }
        assertEquals(500, q.size())
        val b = q.nextBatch()!!
        assertEquals(20, b.events.first().getInt("popupId")) // 最旧 20 条被丢
    }

    @Test fun batchesAtMost100() {
        val q = EventQueue(MemoryKv())
        repeat(250) { q.add(ev(it)) }
        val b1 = q.nextBatch()!!; assertEquals(100, b1.events.size); q.ack(b1.id)
        val b2 = q.nextBatch()!!; assertEquals(100, b2.events.size); q.ack(b2.id)
        val b3 = q.nextBatch()!!; assertEquals(50, b3.events.size); q.ack(b3.id)
        assertNull(q.nextBatch())
    }

    @Test fun batchIdReusedUntilAckedAndSurvivesRestart() {
        val kv = MemoryKv()
        val q = EventQueue(kv)
        repeat(3) { q.add(ev(it)) }
        val b = q.nextBatch()!!
        q.add(ev(99)) // 在途期间新事件不并入在途批次
        assertEquals(b.id, q.nextBatch()!!.id)
        assertEquals(3, q.nextBatch()!!.events.size)
        // 进程重启：从持久化恢复同一批次
        val q2 = EventQueue(kv)
        val r = q2.nextBatch()!!
        assertEquals(b.id, r.id); assertEquals(3, r.events.size)
        q2.ack(r.id)
        val next = q2.nextBatch()!!
        assertNotEquals(b.id, next.id); assertEquals(99, next.events.single().getInt("popupId"))
    }

    @Test fun backoffDoublesAndCapsAtFiveMinutes() {
        assertEquals(0L, Backoff.delayMs(0))
        assertEquals(2_000L, Backoff.delayMs(1))
        assertEquals(4_000L, Backoff.delayMs(2))
        assertEquals(300_000L, Backoff.delayMs(10))
        assertEquals(300_000L, Backoff.delayMs(500))
    }

    @Test fun corruptInflightDoesNotAffectQueueAndViceVersa() {
        val qkv = MemoryKv(); val ikv = MemoryKv()
        val q = EventQueue(qkv, ikv)
        repeat(3) { q.add(ev(it)) }
        q.nextBatch(); q.add(ev(50))
        // 在途存储损坏：队列仍可读
        ikv.putString("ev_inflight", "{bad json")
        val a = EventQueue(qkv, ikv)
        assertEquals(1, a.size())
        // 队列存储损坏：在途批次仍可读
        val ikv2 = MemoryKv(); val q2 = EventQueue(MemoryKv(), ikv2); q2.add(ev(1)); q2.nextBatch()
        val b = EventQueue(MemoryKv().also { it.putString("ev_queue", "[oops") }, ikv2)
        assertEquals(1, b.nextBatch()!!.events.size)
    }

    @Test fun enqueuePersistenceRunsOnExecutorNotCaller() {
        val kv = MemoryKv(); val tasks = ArrayList<Runnable>()
        val q = EventQueue(kv, persistExecutor = { tasks.add(it) })
        repeat(5) { q.add(ev(it)) }
        assertNull(kv.getString("ev_queue")) // 调用线程只做内存追加
        assertEquals(1, tasks.size)          // 多次入队合并为一次落盘
        tasks.forEach { it.run() }
        assertEquals(5, EventQueue(kv).size())
    }

    @Test fun reporterBackoffRules() = runTest {
        val now = 1_000_000L // 固定墙钟：退避期（2s）内
        val q = EventQueue(MemoryKv()); q.add(ev(1))
        val codes = ArrayDeque(listOf(404, 404, 200))
        val r = PopupReporter(q, TestScope(testScheduler), meta, { codes.removeFirst() }, debounceMs = 1_000L, nowMs = { now })
        r.flush()                                  // 404：保留重试并退避，不丢事件
        assertEquals(1, r.failures); assertEquals(1, q.size())
        r.scheduleDebounced()                      // 防抖 flush（1s）早于退避重试（2s）：不得绕过退避
        testScheduler.advanceTimeBy(1_001); testScheduler.runCurrent()
        assertEquals(1, r.failures); assertEquals(2, codes.size) // 没有再发请求
        testScheduler.advanceTimeBy(1_500); testScheduler.runCurrent() // 退避到点才重试
        assertEquals(2, r.failures); assertEquals(1, codes.size)
    }

    @Test fun newEventDuringInflightSendDoesNotCancelIt() = runTest {
        val q = EventQueue(MemoryKv()); q.add(ev(1))
        val gate = kotlinx.coroutines.CompletableDeferred<Unit>()
        val sent = ArrayList<String>()
        val r = PopupReporter(q, TestScope(testScheduler), meta, { body -> sent.add(body); gate.await(); 200 }, debounceMs = 1_000L)
        r.scheduleDebounced()
        testScheduler.advanceTimeBy(1_001); testScheduler.runCurrent() // 防抖结束，发送进行中（卡在 gate）
        r.record(ev(2))                                                // 新事件：只重置防抖，不得取消在途发送
        testScheduler.runCurrent()
        gate.complete(Unit); testScheduler.runCurrent()
        assertEquals(0, r.failures)
        assertTrue("首个批次已 2xx 出队", sent.isNotEmpty())
        testScheduler.advanceTimeBy(1_001); testScheduler.runCurrent()
        assertEquals(0, q.size()) // 两条事件都已送达
    }

    @Test fun reporterDrops413AndCancellationIsNotFailure() = runTest {
        val q = EventQueue(MemoryKv()); q.add(ev(1))
        val r = PopupReporter(q, TestScope(testScheduler), meta, { 413 })
        r.flush(); assertEquals(0, q.size()); assertEquals(0, r.failures)

        val q2 = EventQueue(MemoryKv()); q2.add(ev(2))
        val r2 = PopupReporter(q2, TestScope(testScheduler), meta, { throw kotlinx.coroutines.CancellationException("x") })
        try { r2.flush(); fail() } catch (_: kotlinx.coroutines.CancellationException) {}
        assertEquals(0, r2.failures); assertEquals(1, q2.size())
    }

    private val meta = BatchMeta("com.x.ap01", "ap01", "1.0.3", 10003)

    @Test fun reporterRetriesSameBatchIdAndOnlyDequeuesOn2xx() = runTest {
        val q = EventQueue(MemoryKv())
        repeat(2) { q.add(ev(it)) }
        val seen = ArrayList<String>()
        val codes = ArrayDeque(listOf(500, -1, 200))
        val r = PopupReporter(q, TestScope(testScheduler), meta, { body ->
            seen.add(JSONObject(body).getString("batchId")); codes.removeFirst()
        })
        r.flush(); assertEquals(1, r.failures); assertEquals(2, q.size())
        r.flush(); assertEquals(2, r.failures)
        r.flush(); assertEquals(0, r.failures); assertEquals(0, q.size())
        assertEquals(1, seen.toSet().size) // 三次尝试同一个 batchId
    }

    @Test fun reporterDrops400BatchAndContinues() = runTest {
        val q = EventQueue(MemoryKv(), batchSize = 1)
        repeat(2) { q.add(ev(it)) }
        val codes = ArrayDeque(listOf(400, 200))
        val r = PopupReporter(q, TestScope(testScheduler), meta, { codes.removeFirst() })
        r.flush()
        assertEquals(0, q.size())
    }

    @Test fun batchBodyCarriesMetaAndContractFields() = runTest {
        val q = EventQueue(MemoryKv()); q.add(ev(1))
        var body = ""
        PopupReporter(q, TestScope(testScheduler), meta, { body = it; 200 }).flush()
        val o = JSONObject(body)
        assertEquals("com.x.ap01", o.getString("appId")); assertEquals("ap01", o.getString("palcode"))
        assertEquals("1.0.3", o.getString("appVersion")); assertEquals(10003, o.getInt("versionCode"))
        assertEquals(1, o.getJSONArray("events").length())
    }

    @Test fun eventFieldNamesFollowContract() {
        val p = T.popup(12, PopupPosition.P1); val c = T.card(34)
        val imp = PopupEvents.impression(p, c, 0, "auto", 1L)
        assertEquals("popup_impression", imp.getString("event"))
        assertEquals(12, imp.getInt("popupId")); assertEquals("P1", imp.getString("position"))
        assertEquals(34, imp.getInt("cardId")); assertEquals(0, imp.getInt("cardIndex")); assertEquals("auto", imp.getString("display"))
        assertEquals("mutex", PopupEvents.filtered(p, "mutex", 1L).getString("reason"))
        assertEquals("back", PopupEvents.close(p, "back", 1L).getString("method"))
        val s = PopupEvents.slide(p, 0, 1, 1L)
        assertEquals(0, s.getInt("from")); assertEquals(1, s.getInt("to"))
    }
}
