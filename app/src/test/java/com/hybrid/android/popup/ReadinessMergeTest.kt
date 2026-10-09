package com.hybrid.android.popup

import org.junit.Assert.*
import org.junit.Test

class ReadinessMergeTest {
    private fun p(id: Long, url: String, tab: Boolean = false, icon: String = "") =
        T.popup(id, cards = listOf(T.card(id, url)), tab = tab, tabIcon = icon)

    @Test fun newReadyVersionWins() {
        val old = p(1, "old"); val fresh = p(1, "new")
        val out = ReadinessMerge.merge(listOf(fresh), listOf(old)) { true }
        assertEquals("new", out.single().cards[0].imageUrl)
    }

    @Test fun unreadyNewFallsBackToReadyOld() {
        val old = p(1, "old"); val fresh = p(1, "new")
        val out = ReadinessMerge.merge(listOf(fresh), listOf(old)) { it == "old" }
        assertEquals("old", out.single().cards[0].imageUrl)
    }

    @Test fun unreadyNewWithoutOldIsExcluded() {
        assertTrue(ReadinessMerge.merge(listOf(p(1, "new")), emptyList()) { false }.isEmpty())
        // 旧版本图片也没了 → 同样不参与
        assertTrue(ReadinessMerge.merge(listOf(p(1, "new")), listOf(p(1, "old"))) { false }.isEmpty())
    }

    @Test fun granularityIsPerPopup() {
        val fresh = listOf(p(1, "n1"), p(2, "n2"))
        val prev = listOf(p(1, "o1"), p(2, "o2"))
        val out = ReadinessMerge.merge(fresh, prev) { it == "n1" || it == "o2" }
        assertEquals(listOf("n1", "o2"), out.map { it.cards[0].imageUrl })
    }

    @Test fun popupMissingFromFreshGoesOfflineImmediately() {
        val out = ReadinessMerge.merge(listOf(p(2, "n2")), listOf(p(1, "o1"), p(2, "o2"))) { true }
        assertEquals(listOf(2L), out.map { it.id }) // 含其便条一并下线（上层据此 reconcile）
    }

    @Test fun tabIconCountsTowardReadiness() {
        val fresh = p(1, "img", tab = true, icon = "icon")
        assertTrue(ReadinessMerge.merge(listOf(fresh), emptyList()) { it == "img" }.isEmpty())
        assertEquals(1, ReadinessMerge.merge(listOf(fresh), emptyList()) { true }.size)
    }

    @Test fun referencedUrlsKeepFreshAndEffective() {
        val s = ReadinessMerge.referencedUrls(listOf(p(1, "n1")), listOf(p(1, "o1")))
        assertEquals(setOf("n1", "o1"), s)
    }
}
