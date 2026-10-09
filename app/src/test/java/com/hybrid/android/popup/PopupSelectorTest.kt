package com.hybrid.android.popup

import com.hybrid.android.popup.PopupPosition.*
import org.junit.Assert.*
import org.junit.Test

class PopupSelectorTest {
    private val all = setOf(P1, P2, P3, P4, P5, P7, P8)
    private fun sel(list: List<Popup>, ctx: SelectionContext = T.ctx, pos: Set<PopupPosition> = all, freq: (Popup) -> Boolean = { true }) =
        PopupSelector.select(list, pos, ctx, freq)

    @Test fun versionRangeIsMinInclusiveMaxExclusive() {
        val p = T.popup(1, minV = 10003, maxV = 10100)
        assertTrue(PopupSelector.targetingOk(p, T.ctx.copy(versionCode = 10003)))
        assertFalse(PopupSelector.targetingOk(p, T.ctx.copy(versionCode = 10002)))
        assertFalse(PopupSelector.targetingOk(p, T.ctx.copy(versionCode = 10100)))
        assertTrue(PopupSelector.targetingOk(T.popup(2), T.ctx.copy(versionCode = 1)))
    }

    @Test fun userTypeNewOld() {
        val n = T.popup(1, user = "new"); val o = T.popup(2, user = "old")
        val newUser = T.ctx.copy(isNewUser = true)
        assertTrue(PopupSelector.targetingOk(n, newUser)); assertFalse(PopupSelector.targetingOk(o, newUser))
        assertFalse(PopupSelector.targetingOk(n, T.ctx)); assertTrue(PopupSelector.targetingOk(o, T.ctx))
    }

    @Test fun reasonsAndOrder() {
        val r = sel(
            listOf(
                T.popup(1, minV = 99999),              // targeting
                T.popup(2, start = 5_000),             // time (未到)
                T.popup(3, end = 1_000),               // time (已过, end 含边界即过期)
                T.popup(4),                            // frequency
                T.popup(5),                            // 胜出
            ),
            freq = { it.id != 4L },
        )
        assertEquals(5L, r.overlay?.id)
        assertEquals(
            mapOf(1L to "targeting", 2L to "time", 3L to "time", 4L to "frequency"),
            r.filtered.associate { it.popup.id to it.reason }
        )
    }

    @Test fun targetingBeatsTimeBeatsFrequency() {
        val p = T.popup(1, minV = 99999, start = 5_000)
        assertEquals("targeting", sel(listOf(p), freq = { false }).filtered.single().reason)
        val q = T.popup(2, start = 5_000)
        assertEquals("time", sel(listOf(q), freq = { false }).filtered.single().reason)
    }

    @Test fun forcedP4AlwaysWinsThenPriorityThenPositionThenId() {
        val forced = T.popup(10, P4, priority = 1, closable = false)
        val hi = T.popup(11, P1, priority = 999)
        assertEquals(10L, sel(listOf(hi, forced)).overlay?.id)

        val a = T.popup(1, P1, priority = 50); val b = T.popup(2, P7, priority = 80)
        assertEquals(2L, sel(listOf(a, b)).overlay?.id)

        // 平手：P4 > P7 > P1
        val p1 = T.popup(1, P1, priority = 5); val p7 = T.popup(2, P7, priority = 5); val p4 = T.popup(3, P4, priority = 5)
        val r = sel(listOf(p1, p7, p4))
        assertEquals(3L, r.overlay?.id)
        assertEquals(setOf(1L, 2L), r.filtered.filter { it.reason == "mutex" }.map { it.popup.id }.toSet())

        // 再平手：id 升序
        assertEquals(7L, sel(listOf(T.popup(9, P1, priority = 5), T.popup(7, P1, priority = 5))).overlay?.id)
    }

    @Test fun nonOverlayOnePerPositionRestMutex() {
        val r = sel(listOf(
            T.popup(1, P2, priority = 1), T.popup(2, P2, priority = 9),
            T.popup(3, P5), T.popup(4, P8), T.popup(5, P8, priority = 3),
        ))
        assertNull(r.overlay)
        assertEquals(listOf(2L, 3L, 5L), r.others.map { it.id })
        assertEquals(setOf(1L, 4L), r.filtered.map { it.popup.id }.toSet())
        assertTrue(r.filtered.all { it.reason == "mutex" })
    }

    @Test fun emptyConfigRecordsNothing() {
        val r = sel(emptyList())
        assertNull(r.overlay); assertTrue(r.others.isEmpty()); assertTrue(r.filtered.isEmpty())
    }

    @Test fun positionsOutsideScopeAreIgnoredSilently() {
        val r = sel(listOf(T.popup(1, P6), T.popup(2, P1)), pos = setOf(P1))
        assertEquals(2L, r.overlay?.id)
        assertTrue(r.filtered.isEmpty())
    }
}
