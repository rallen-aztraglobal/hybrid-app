package com.hybrid.android.popup

/** 时钟环境，便于单测注入。 */
interface ClockEnv {
    /** 本地墙钟（System.currentTimeMillis），用户可改。 */
    fun wallMs(): Long
    /** 开机以来单调时间（SystemClock.elapsedRealtime），用户改不了。 */
    fun elapsedMs(): Long
    /** 开机次数（Settings.Global.BOOT_COUNT），用来判断「是否同一次开机」。 */
    fun bootCount(): Long
}

/**
 * 可信时间与 dayKey（契约 §7.2）。
 *
 *  - 拉取成功时 [recordAnchor]：(serverTime, elapsedRealtime, bootCount, wall)。
 *  - 同一次开机：now = serverTime + (elapsed - anchor.elapsed)，与用户改系统时间无关。
 *  - 重启后尚未拉到新配置：以「开机后首次取用」为基线（wall + 上次记录的 serverTime-wall 偏差），
 *    之后同样按 elapsed 外推——于是同次开机内墙钟再怎么跳变（>2h 或更小）都不会让 dayKey 回退/重置。
 *  - dayKey 另外加单调保护：永不小于上一次算出的值。
 */
class TrustedClock(private val kv: KvStore, private val env: ClockEnv) {

    private val lock = Any()

    fun recordAnchor(serverTime: Long, tzOffsetMinutes: Int) = synchronized(lock) {
        if (serverTime <= 0L) return@synchronized
        kv.putLong(K_SERVER, serverTime)
        kv.putLong(K_ELAPSED, env.elapsedMs())
        kv.putLong(K_BOOT, env.bootCount())
        kv.putLong(K_WALL, env.wallMs())
        kv.putLong(K_TZ, tzOffsetMinutes.toLong())
        // 新锚点覆盖旧基线；并解除 dayKey 单调保护（服务端时间权威，可把 dayKey 校正回来）
        kv.putLong(K_BASE_BOOT, -1L)
        kv.putLong(K_LAST_DAY_BOOT, -1L)
    }

    /** 是否有过锚点（从未拉到过配置时不应展示任何东西）。 */
    fun hasAnchor(): Boolean = kv.getLong(K_SERVER, 0L) > 0L

    val tzOffsetMinutes: Int get() = kv.getLong(K_TZ, 0L).toInt()

    fun now(): Long = synchronized(lock) {
        val serverTime = kv.getLong(K_SERVER, 0L)
        if (serverTime <= 0L) return@synchronized env.wallMs() // 无锚点：退回墙钟（仅用于埋点 ts 等非关键用途）
        val anchorElapsed = kv.getLong(K_ELAPSED, 0L)
        val boot = env.bootCount()
        // BOOT_COUNT 读不到（0）时无法判定同次开机，一律按重启处理。
        if (boot != 0L && kv.getLong(K_BOOT, -2L) == boot && env.elapsedMs() >= anchorElapsed) {
            return@synchronized serverTime + (env.elapsedMs() - anchorElapsed)
        }
        // 重启后：基线（本次开机内首次取用时固化）
        val baseStale = boot == 0L && env.elapsedMs() < kv.getLong(K_BASE_ELAPSED, 0L) // 无 bootCount 时靠 elapsed 回退识别重启
        if (kv.getLong(K_BASE_BOOT, -1L) != boot || baseStale) {
            val offset = serverTime - kv.getLong(K_WALL, serverTime)
            kv.putLong(K_BASE_BOOT, boot)
            kv.putLong(K_BASE_TIME, env.wallMs() + offset)
            kv.putLong(K_BASE_ELAPSED, env.elapsedMs())
        }
        kv.getLong(K_BASE_TIME, 0L) + (env.elapsedMs() - kv.getLong(K_BASE_ELAPSED, 0L))
    }

    /** dayKey = floor((trustedNow + tz) / 1d)，单调不减。 */
    fun dayKey(): Long = synchronized(lock) {
        val computed = Math.floorDiv(now() + tzOffsetMinutes * 60_000L, DAY_MS)
        val boot = env.bootCount()
        val last = kv.getLong(K_LAST_DAY, Long.MIN_VALUE)
        // 单调保护只在同一次开机内生效（boot 读不到时不启用）
        val guard = boot != 0L && kv.getLong(K_LAST_DAY_BOOT, -1L) == boot
        val k = if (guard && last != Long.MIN_VALUE && computed < last) last else computed
        kv.putLong(K_LAST_DAY, k)
        kv.putLong(K_LAST_DAY_BOOT, boot)
        k
    }

    companion object {
        const val DAY_MS = 86_400_000L
        private const val K_SERVER = "clk_server"
        private const val K_ELAPSED = "clk_elapsed"
        private const val K_BOOT = "clk_boot"
        private const val K_WALL = "clk_wall"
        private const val K_TZ = "clk_tz"
        private const val K_BASE_BOOT = "clk_base_boot"
        private const val K_BASE_TIME = "clk_base_time"
        private const val K_BASE_ELAPSED = "clk_base_elapsed"
        private const val K_LAST_DAY = "clk_last_day"
        private const val K_LAST_DAY_BOOT = "clk_last_day_boot"
    }
}
