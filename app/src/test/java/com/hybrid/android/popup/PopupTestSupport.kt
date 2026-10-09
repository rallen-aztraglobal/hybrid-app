package com.hybrid.android.popup

/** 单测公共夹具。 */
object T {
    fun card(id: Long = 1, img: String = "https://cdn/x$id.png", link: String = "/p", btn: String = "Go") =
        PopupCard(id, img, link, btn, "", "")

    fun popup(
        id: Long, pos: PopupPosition = PopupPosition.P1, priority: Int = 0,
        start: Long = 0, end: Long = 0, minV: Int = 0, maxV: Int = 0, user: String = "all",
        closable: Boolean = true, tab: Boolean = false, cards: List<PopupCard> = listOf(card(id)),
        tabIcon: String = "", countdown: Boolean = false,
    ) = Popup(
        id = id, position = pos, priority = priority, startAt = start, endAt = end,
        minVersionCode = minV, maxVersionCode = maxV, userType = user, openMode = "webview",
        closable = closable, maskClosable = false, countdown = countdown, tabEnabled = tab,
        tabIconUrl = tabIcon, tabText = "Tab", autoplaySeconds = 5, resumeGapMinutes = 30,
        badge = false, cards = cards,
    )

    class FakeEnv(var wall: Long = 1_000_000_000_000L, var elapsed: Long = 100_000L, var boot: Long = 5L) : ClockEnv {
        override fun wallMs() = wall
        override fun elapsedMs() = elapsed
        override fun bootCount() = boot
    }

    val ctx = SelectionContext(versionCode = 10003, isNewUser = false, nowMs = 1_000L)
}
