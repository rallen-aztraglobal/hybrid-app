package com.hybrid.android.popup

import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.*
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.io.File

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class AssetAndRepoTest {
    @get:Rule val tmp = TemporaryFolder()

    // 测试里把「以 IMG 开头」视为可解码图片，避免依赖 Robolectric 的 BitmapFactory 行为。
    private val validator: (File) -> Boolean = { it.readText().startsWith("IMG") }

    @Test fun htmlResponseNeverEntersCache() = runBlocking {
        MockWebServer().use { srv ->
            srv.enqueue(MockResponse().setBody("<html>blocked</html>"))
            val cache = AssetCache(tmp.newFolder(), OkHttpClient(), validator)
            val url = srv.url("/a.png").toString()
            assertFalse(cache.ensure(url, 3_000))
            assertFalse(cache.isReady(url))
            assertFalse(cache.fileFor(url).exists())
        }
    }

    @Test fun validDownloadIsReadyAndBadLegacyFileIsDeleted() = runBlocking {
        MockWebServer().use { srv ->
            srv.enqueue(MockResponse().setBody("IMGDATA"))
            val dir = tmp.newFolder()
            val cache = AssetCache(dir, OkHttpClient(), validator)
            val url = srv.url("/ok.png").toString()
            assertTrue(cache.ensure(url, 3_000)); assertTrue(cache.isReady(url))
            // 磁盘上历史遗留的坏文件：新进程（未校验过）首次 isReady 时被识别并删除
            val bad = "https://cdn/bad.png"
            cache.fileFor(bad).writeText("<html>")
            val fresh = AssetCache(dir, OkHttpClient(), validator)
            assertFalse(fresh.isReady(bad)); assertFalse(cache.fileFor(bad).exists())
            // invalidate：解码失败后删除，下次重新下载
            cache.invalidate(url); assertFalse(cache.fileFor(url).exists())
        }
    }

    @Test fun cleanupKeepsReferencedAndActiveTmp() {
        val dir = tmp.newFolder()
        val cache = AssetCache(dir, OkHttpClient(), validator)
        val keep = "https://cdn/keep.png"; val drop = "https://cdn/drop.png"
        cache.fileFor(keep).writeText("IMG"); cache.fileFor(drop).writeText("IMG")
        cache.cleanup(setOf(keep))
        assertTrue(cache.fileFor(keep).exists()); assertFalse(cache.fileFor(drop).exists())
    }

    private fun repo(responses: ArrayDeque<String?>): PopupRepository {
        val kv = MemoryKv()
        val clock = TrustedClock(kv, T.FakeEnv())
        val cache = AssetCache(tmp.newFolder(), OkHttpClient(), validator)
        return PopupRepository(tmp.newFolder(), clock, cache, { responses.removeFirst() }, { 0L })
    }

    private fun cfg(vararg ids: Int) = """{"appId":"a","configVersion":"v${ids.joinToString("")}","serverTime":1760000000000,"tzOffsetMinutes":0,
      "popups":[${ids.joinToString(",") { """{"id":$it,"position":"P5","cards":[{"id":$it,"title":"t"}]}""" }}]}"""

    @Test fun refreshPublishesStateAndDropsVanishedPopupsImmediately() = runBlocking {
        val r = repo(ArrayDeque(listOf(cfg(1, 2), cfg(2))))
        assertNull(r.state.value)
        assertTrue(r.refresh())
        assertEquals(listOf(1L, 2L), r.state.value!!.popups.map { it.id })
        assertTrue(r.refresh())
        // 新配置里消失的弹窗立即下线；订阅方（Manager）据此对账便条 / 视图
        assertEquals(listOf(2L), r.state.first { it?.configVersion == "v2" }!!.popups.map { it.id })
        assertEquals(listOf(2L), r.effective!!.popups.map { it.id })
    }

    @Test fun failedFetchKeepsEffective() = runBlocking {
        val r = repo(ArrayDeque(listOf(cfg(1), null)))
        r.refresh()
        assertFalse(r.refresh())
        assertEquals(listOf(1L), r.effective!!.popups.map { it.id })
    }
}
