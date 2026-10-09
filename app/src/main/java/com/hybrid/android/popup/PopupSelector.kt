package com.hybrid.android.popup

/** 过滤原因（契约 §7.3，与埋点 reason 逐字一致）。 */
object FilterReason {
    const val TARGETING = "targeting"
    const val TIME = "time"
    const val FREQUENCY = "frequency"
    const val MUTEX = "mutex"
}

data class SelectionContext(val versionCode: Int, val isNewUser: Boolean, val nowMs: Long)

data class Filtered(val popup: Popup, val reason: String)

/**
 * @property overlay 本轮胜出的遮罩类（最多一个）
 * @property others 非遮罩类，每个位置至多一个
 * @property filtered 被拦截的弹窗及原因（调用方逐条记 popup_filtered）
 */
data class Selection(val overlay: Popup?, val others: List<Popup>, val filtered: List<Filtered>)

/** 候选筛选与互斥排序（契约 §7.3 / §7.4），纯函数。 */
object PopupSelector {

    /** 版本 ≥ min、< max（0=不限）；userType all/new/old。 */
    fun targetingOk(p: Popup, ctx: SelectionContext): Boolean {
        if (p.minVersionCode > 0 && ctx.versionCode < p.minVersionCode) return false
        if (p.maxVersionCode > 0 && ctx.versionCode >= p.maxVersionCode) return false
        return when (p.userType) {
            "new" -> ctx.isNewUser
            "old" -> !ctx.isNewUser
            else -> true
        }
    }

    fun timeOk(p: Popup, nowMs: Long): Boolean {
        if (p.startAt > 0 && nowMs < p.startAt) return false
        if (p.endAt > 0 && nowMs >= p.endAt) return false
        return true
    }

    /** 遮罩互斥排序：强制 P4 最先 → priority 降序 → 位置 P4>P7>P1>P6 → id 升序。 */
    val overlayComparator: Comparator<Popup> =
        compareByDescending<Popup> { it.forced }
            .thenByDescending { it.priority }
            .thenBy { it.position.rank }
            .thenBy { it.id }

    /** 同位置非遮罩排序：priority 降序 → id 升序。 */
    val sameSlotComparator: Comparator<Popup> =
        compareByDescending<Popup> { it.priority }.thenBy { it.id }

    fun select(
        popups: List<Popup>,
        positions: Set<PopupPosition>,
        ctx: SelectionContext,
        frequencyAllows: (Popup) -> Boolean,
    ): Selection {
        val filtered = ArrayList<Filtered>()
        val passOverlay = ArrayList<Popup>()
        val passOthers = ArrayList<Popup>()
        // 过滤顺序：targeting → time → frequency（互斥最后）
        for (p in popups.sortedWith(compareBy({ it.position.ordinal }, { it.id }))) {
            if (p.position !in positions) continue
            when {
                !targetingOk(p, ctx) -> filtered.add(Filtered(p, FilterReason.TARGETING))
                !timeOk(p, ctx.nowMs) -> filtered.add(Filtered(p, FilterReason.TIME))
                !frequencyAllows(p) -> filtered.add(Filtered(p, FilterReason.FREQUENCY))
                p.position.overlay -> passOverlay.add(p)
                else -> passOthers.add(p)
            }
        }
        var winner: Popup? = null
        if (passOverlay.isNotEmpty()) {
            val sorted = passOverlay.sortedWith(overlayComparator)
            winner = sorted.first()
            sorted.drop(1).forEach { filtered.add(Filtered(it, FilterReason.MUTEX)) }
        }
        val others = ArrayList<Popup>()
        passOthers.groupBy { it.position }.forEach { (_, list) ->
            val sorted = list.sortedWith(sameSlotComparator)
            others.add(sorted.first())
            sorted.drop(1).forEach { filtered.add(Filtered(it, FilterReason.MUTEX)) }
        }
        return Selection(winner, others.sortedBy { it.position.ordinal }, filtered)
    }
}
