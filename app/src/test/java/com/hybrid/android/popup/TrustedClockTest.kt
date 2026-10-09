package com.hybrid.android.popup

import org.junit.Assert.*
import org.junit.Test

class TrustedClockTest {
    private val DAY = TrustedClock.DAY_MS
    private val server = 1_760_000_000_000L // 某天中间

    private fun make(env: T.FakeEnv, kv: KvStore = MemoryKv()) = TrustedClock(kv, env).also { it.recordAnchor(server, 480) }

    @Test fun sameBootUsesElapsedNotWallClock() {
        val env = T.FakeEnv(); val c = make(env)
        env.elapsed += 60_000
        env.wall += 10 * DAY // 用户把系统时间调到 10 天后
        assertEquals(server + 60_000, c.now())
    }

    @Test fun dayKeyUsesTzOffset() {
        val env = T.FakeEnv(); val c = make(env)
        assertEquals(Math.floorDiv(server + 480 * 60_000L, DAY), c.dayKey())
    }

    @Test fun afterRebootUsesWallPlusServerOffsetAndIgnoresLaterJumps() {
        val kv = MemoryKv()
        val env = T.FakeEnv(wall = 1_000_000_000_000L, elapsed = 100_000L, boot = 5)
        val c = make(env, kv)
        // 重启：boot+1，elapsed 清零；服务器与本机墙钟偏差 = server - wall(记录时)
        val offset = server - 1_000_000_000_000L
        val env2 = T.FakeEnv(wall = 1_000_000_000_000L + 3 * 3_600_000L, elapsed = 5_000L, boot = 6)
        val c2 = TrustedClock(kv, env2)
        assertEquals(env2.wall + offset, c2.now())
        val k0 = c2.dayKey()
        // 同次开机内墙钟向前跳 5 天（>2h）：不应改变 dayKey
        env2.wall += 5 * DAY; env2.elapsed += 1_000
        assertEquals(k0, c2.dayKey())
        // 向后跳 3 天：也不回退
        env2.wall -= 8 * DAY; env2.elapsed += 1_000
        assertEquals(k0, c2.dayKey())
        // 真实流逝一天（elapsed）才换日
        env2.elapsed += DAY
        assertEquals(k0 + 1, c2.dayKey())
        assertNotNull(c)
    }

    @Test fun newAnchorAfterRebootSwitchesBackToElapsedMode() {
        val kv = MemoryKv()
        make(T.FakeEnv(boot = 5), kv)
        val env2 = T.FakeEnv(boot = 6, elapsed = 1_000)
        val c2 = TrustedClock(kv, env2)
        c2.now() // 固化基线
        c2.recordAnchor(server + 7 * DAY, 480)
        env2.elapsed += 500
        assertEquals(server + 7 * DAY + 500, c2.now())
    }

    @Test fun dayKeyMonotonicWithinBootButServerAnchorCanCorrect() {
        val env = T.FakeEnv(); val kv = MemoryKv(); val c = make(env, kv)
        val k = c.dayKey()
        // 同一次开机内，本地推算不会回退
        assertEquals(k, c.dayKey())
        // 拿到服务端锚点时允许校正回来（服务端时间权威）
        c.recordAnchor(server - 3 * DAY, 480)
        assertEquals(k - 3, c.dayKey())
    }

    @Test fun monotonicGuardOnlyWithinSameBoot() {
        val kv = MemoryKv()
        val env = T.FakeEnv(boot = 5); val c = make(env, kv)
        val k = c.dayKey()
        // 重启后基线由墙钟推出：墙钟比之前小很多 → 不再被上次 dayKey 钉住
        val env2 = T.FakeEnv(wall = env.wall - 5 * DAY, elapsed = 1_000, boot = 6)
        assertTrue(TrustedClock(kv, env2).dayKey() < k)
    }

    @Test fun unreadableBootCountTreatedAsReboot() {
        val env = T.FakeEnv(boot = 0); val c = make(env, MemoryKv())
        val offset = server - env.wall
        env.wall += 1_000; env.elapsed += 1_000
        // boot=0 不能据 elapsed 判「同次开机」：走墙钟基线（首次取用固化）而不是 server+elapsed差
        assertEquals(env.wall + offset, c.now())
        val t0 = c.now()
        env.wall += 10 * DAY; env.elapsed += 500 // 之后墙钟跳变不影响
        assertEquals(t0 + 500, c.now())
    }

    @Test fun noAnchor() {
        val c = TrustedClock(MemoryKv(), T.FakeEnv())
        assertFalse(c.hasAnchor())
    }
}
