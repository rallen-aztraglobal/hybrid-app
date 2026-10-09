package com.hybrid.android.popup

import com.hybrid.android.popup.PopupPosition.*
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class FrequencyPolicyTest {
    private lateinit var env: T.FakeEnv
    private lateinit var clock: TrustedClock
    private lateinit var fq: FrequencyPolicy
    private val kv = MemoryKv()
    private val DAY = TrustedClock.DAY_MS

    @Before fun setup() {
        FrequencyPolicy.resetSessionForTest()
        env = T.FakeEnv()
        clock = TrustedClock(kv, env)
        clock.recordAnchor(1_760_000_000_000L, 0)
        fq = FrequencyPolicy(kv, clock)
    }

    private fun nextDay() { env.elapsed += DAY }

    @Test fun p1P7P4DailyOnceWrittenOnRender() {
        for (pos in listOf(P1, P7, P4)) {
            val p = T.popup(1, pos)
            assertTrue(fq.allows(p)); fq.onRendered(p); assertFalse("$pos 当日不再", fq.allows(p))
        }
        nextDay()
        assertTrue(fq.allows(T.popup(1, P1)))
    }

    @Test fun forcedP4EveryColdStart() {
        val p = T.popup(1, P4, closable = false)
        fq.onRendered(p)
        assertTrue(fq.allows(p))
    }

    @Test fun p2SessionOnly() {
        val p = T.popup(1, P2)
        assertTrue(fq.allows(p)); fq.onClosed(p); assertFalse(fq.allows(p))
        FrequencyPolicy.resetSessionForTest() // 模拟新进程
        assertTrue(fq.allows(p))
    }

    @Test fun p3Window24h() {
        val p = T.popup(1, P3)
        fq.onClosed(p)
        assertFalse(fq.allows(p))
        env.elapsed += 23 * 3_600_000L; assertFalse(fq.allows(p))
        env.elapsed += 2 * 3_600_000L; assertTrue(fq.allows(p))
    }

    @Test fun p5ClosedDay() {
        val p = T.popup(1, P5)
        assertTrue(fq.allows(p)); fq.onClosed(p); assertFalse(fq.allows(p))
        nextDay(); assertTrue(fq.allows(p))
    }

    @Test fun p6DayAndSession() {
        val p = T.popup(1, P6)
        assertTrue(fq.allows(p)); fq.onRendered(p); assertFalse(fq.allows(p))
        nextDay()
        assertFalse("同会话已展示过", fq.allows(p))
        FrequencyPolicy.resetSessionForTest()
        assertTrue(fq.allows(p))
    }

    @Test fun tabDismissBlocksPopupAndTabForToday() {
        val p = T.popup(1, P1)
        fq.dismissToday(p)
        assertTrue(fq.isDismissedToday(p)); assertFalse(fq.allows(p))
        nextDay()
        assertFalse(fq.isDismissedToday(p)); assertTrue(fq.allows(p))
    }

    @Test fun p8OnlyDismissal() {
        val p = T.popup(1, P8)
        assertTrue(fq.allows(p)); fq.dismissToday(p); assertFalse(fq.allows(p))
    }
}
