package com.hybrid.android.popup

/**
 * 频控（契约 §7.3，常量写死）。按设备记（SharedPreferences），日维度用 [TrustedClock.dayKey]。
 * 「本会话」状态存内存（进程存活期）。
 */
class FrequencyPolicy(private val kv: KvStore, private val clock: TrustedClock) {

    /** 自动展示是否允许（不含 targeting/time）。 */
    fun allows(p: Popup): Boolean {
        if (isDismissedToday(p)) return false
        val today = clock.dayKey()
        return when (p.position) {
            PopupPosition.P1, PopupPosition.P7 -> kv.getLong(autoDayKey(p.position), NONE) != today
            PopupPosition.P4 -> p.forced || kv.getLong(autoDayKey(p.position), NONE) != today
            PopupPosition.P2 -> p.id !in sessionClosed
            PopupPosition.P3 -> {
                val closedAt = kv.getLong(K_P3_CLOSED_AT, NONE)
                closedAt == NONE || clock.now() - closedAt >= TAB_P3_TTL_MS
            }
            PopupPosition.P5 -> kv.getLong(K_P5_CLOSED_DAY, NONE) != today
            PopupPosition.P6 -> sessionP6Shown.not() && kv.getLong(autoDayKey(p.position), NONE) != today
            PopupPosition.P8 -> true // 仅受 isDismissedToday 约束
        }
    }

    /** 真实渲染成功时写入（契约：不是点击/关闭后写）。便条重开不调用。 */
    fun onRendered(p: Popup) {
        when (p.position) {
            PopupPosition.P1, PopupPosition.P7, PopupPosition.P4 ->
                kv.putLong(autoDayKey(p.position), clock.dayKey())
            PopupPosition.P6 -> {
                sessionP6Shown = true
                kv.putLong(autoDayKey(p.position), clock.dayKey())
            }
            else -> Unit
        }
    }

    /** 用户关闭（按钮 / 遮罩 / 返回）时写入的位置级状态。 */
    fun onClosed(p: Popup) {
        when (p.position) {
            PopupPosition.P2 -> sessionClosed.add(p.id)
            PopupPosition.P3 -> kv.putLong(K_P3_CLOSED_AT, clock.now())
            PopupPosition.P5 -> kv.putLong(K_P5_CLOSED_DAY, clock.dayKey())
            else -> Unit
        }
    }

    /** 便条长按：当日（P3 同样到次日）不再出现该便条与该弹窗。 */
    fun dismissToday(p: Popup) = kv.putLong(K_DISMISS + p.id, clock.dayKey())

    fun isDismissedToday(p: Popup): Boolean = kv.getLong(K_DISMISS + p.id, NONE) == clock.dayKey()

    companion object {
        const val TAB_P3_TTL_MS = 24 * 3_600_000L
        private const val NONE = Long.MIN_VALUE
        private fun autoDayKey(pos: PopupPosition) = "fq_auto_day_${pos.code}"
        private const val K_P3_CLOSED_AT = "fq_p3_closed_at"
        private const val K_P5_CLOSED_DAY = "fq_p5_closed_day"
        private const val K_DISMISS = "fq_dismiss_"

        /** 进程级「本会话」状态（Activity 重建不丢，进程重启清零）。 */
        private val sessionClosed = HashSet<Long>()
        @Volatile private var sessionP6Shown = false

        /** 新会话（新启动，非配置变更重建）：清空「本会话」级状态。 */
        fun resetSession() { sessionClosed.clear(); sessionP6Shown = false }
        internal fun resetSessionForTest() = resetSession()
    }
}
