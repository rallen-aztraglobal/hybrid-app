package com.hybrid.android.popup

import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

class SessionAndP6Test {
    private val p = T.popup(1)
    private fun snap() = SessionSnapshot(OverlaySnap(p, "auto", 1, 1500, setOf(34L)), emptyList(), listOf(TabSnap(p, false)))

    @Before fun clean() { PopupSession.reset() }

    @Test fun freshLaunchResetsSessionState() {
        // 上一个会话：冷启动评估过、P6 评估过、记过便条曝光、P2/P6 会话级状态
        PopupSession.coldDone = true; PopupSession.p6Evaluated = true
        PopupSession.tabImpressedAdd(7)
        FrequencyPolicy(MemoryKv(), TrustedClock(MemoryKv(), T.FakeEnv())).onClosed(T.popup(9, PopupPosition.P2))
        // 用户 finish() 后进程仍在，再次从桌面打开：savedInstanceState == null
        assertNull(PopupSession.onActivityCreate(hasSavedState = false))
        assertFalse(PopupSession.coldDone); assertFalse(PopupSession.p6Evaluated)
        assertTrue(PopupSession.tabImpressedAdd(7)) // 已清空，可再次记 tab_impression
        val kv = MemoryKv(); val c = TrustedClock(kv, T.FakeEnv()).also { it.recordAnchor(1_760_000_000_000L, 0) }
        assertTrue(FrequencyPolicy(kv, c).allows(T.popup(9, PopupPosition.P2))) // P2 sessionClosed 已清
    }

    @Test fun configChangeRecreateKeepsStateAndReturnsSnapshot() {
        PopupSession.coldDone = true; PopupSession.p6Evaluated = true
        PopupSession.saveForRecreate(snap())
        val s = PopupSession.onActivityCreate(hasSavedState = true)!!
        assertEquals(1, s.overlay!!.page); assertEquals(1500, s.overlay!!.countdownElapsedMs)
        assertTrue(PopupSession.coldDone); assertTrue(PopupSession.p6Evaluated) // 不重新冷启动评估
        // 快照一次性
        assertNull(PopupSession.onActivityCreate(hasSavedState = true))
    }

    @Test fun processDeathRestoreWithoutSnapshotIsNewSession() {
        PopupSession.coldDone = true
        assertNull(PopupSession.onActivityCreate(hasSavedState = true))
        assertFalse(PopupSession.coldDone)
    }

    @Test fun staleSnapshotIgnoredOnFreshLaunch() {
        PopupSession.saveForRecreate(snap())
        assertNull(PopupSession.onActivityCreate(hasSavedState = false))
        assertNull(PopupSession.onActivityCreate(hasSavedState = true)) // 旧快照已被丢弃
    }

    @Test fun p6BackExitsImmediatelyButButtonOrMaskExitsOnNextBack() {
        assertEquals(P6Exit.Action.EXIT_NOW, P6Exit.afterClose("back"))
        assertEquals(P6Exit.Action.EXIT_ON_NEXT_BACK, P6Exit.afterClose("button"))
        assertEquals(P6Exit.Action.EXIT_ON_NEXT_BACK, P6Exit.afterClose("mask"))
    }

    @Test fun coldEvaluationInterruptedByRecreateIsRerunByNewInstance() {
        // 冷启动评估等待中（coldDone 尚未置位）被旋转重建：快照里带着加载中的弹窗，新实例仍会重跑评估
        PopupSession.saveForRecreate(SessionSnapshot(null, emptyList(), emptyList(), pending = listOf(PendingSnap(p, "auto"))))
        val s = PopupSession.onActivityCreate(hasSavedState = true)!!
        assertEquals(listOf(1L), s.pending.map { it.popup.id }) // 加载中的弹窗不丢，且无需重复记 trigger
        assertFalse(PopupSession.coldDone)                      // → 新实例 onPageFinished 会重新评估
    }

    @Test fun lastStopSurvivesBackgroundRecreateButNotFreshLaunch() {
        PopupSession.lastStopElapsed = 12_345L
        PopupSession.saveForRecreate(snap())
        PopupSession.onActivityCreate(hasSavedState = true)
        assertEquals(12_345L, PopupSession.lastStopElapsed) // 后台期间重建：回前台仍能判定 P1 间隔
        PopupSession.onActivityCreate(hasSavedState = false)
        assertEquals(0L, PopupSession.lastStopElapsed)
    }

    @Test fun exitGateArmedUntilNavigationOrStop() {
        val g = ExitGate()
        assertFalse(g.armed)
        g.arm(); assertTrue(g.armed)
        g.disarm(); assertFalse(g.armed) // onPageStarted / onStop
    }

    @Test fun renderGuardRechecksBeforeRender() {
        val cfg = PopupConfig("a", "v", 0, 0, listOf(T.popup(1), T.popup(2, end = 500)))
        val ctx = T.ctx // nowMs=1000
        val ok = { _: Popup -> true }; val no = { _: Popup -> false }
        assertTrue(RenderGuard.eligible(T.popup(1), cfg, ctx, true, ok, no))
        assertFalse("已从配置消失", RenderGuard.eligible(T.popup(9), cfg, ctx, true, ok, no))
        assertFalse("生效期已过", RenderGuard.eligible(T.popup(2), cfg, ctx, true, ok, no))
        assertFalse("频控已不允许", RenderGuard.eligible(T.popup(1), cfg, ctx, true, no, no))
        assertFalse("当日已长按关闭", RenderGuard.eligible(T.popup(1), cfg, ctx, true, ok, { true }))
        // 便条重开 / 重建恢复：不强制规则，只要求仍在配置里
        assertTrue(RenderGuard.eligible(T.popup(1), cfg, ctx, false, no, no))
        assertFalse(RenderGuard.eligible(T.popup(9), cfg, ctx, false, ok, no))
        assertFalse(RenderGuard.eligible(T.popup(1), null, ctx, false, ok, no))
    }
}
