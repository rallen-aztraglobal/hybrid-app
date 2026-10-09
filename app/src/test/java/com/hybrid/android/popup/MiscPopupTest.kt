package com.hybrid.android.popup

import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class MiscPopupTest {

    @Test fun endpointsDerivedLikeDeviceRegistrar() {
        assertEquals("https://h/api/app/popups?appId=com.a.b", PopupEndpoints.configUrl("https://h/api/app/config", "com.a.b"))
        assertEquals("https://h/api/app/popups/events", PopupEndpoints.eventsUrl("https://h/api/app/config/"))
        assertEquals("https://h/x/popups/events", PopupEndpoints.eventsUrl("https://h/x/cfg"))
        assertNull(PopupEndpoints.eventsUrl(""))
    }

    @Test fun linkResolution() {
        val d = "https://site.com/"
        assertEquals(LinkAction.LoadInWebView("https://site.com/promo?palcode=ap01"), LinkResolver.resolve("webview", "/promo", d, "ap01"))
        assertEquals(LinkAction.LoadInWebView("https://site.com/p?a=1&palcode=ap01#x"), LinkResolver.resolve("webview", "/p?a=1#x", d, "ap01"))
        assertEquals(LinkAction.LoadInWebView("https://other.com/a", external = true), LinkResolver.resolve("webview", "https://other.com/a", d, "ap01"))
        assertEquals(LinkAction.OpenBrowser("https://other.com/a"), LinkResolver.resolve("browser", "https://other.com/a", d, "ap01"))
        assertNull(LinkResolver.resolve("browser", "/path", d, "ap01"))
        assertNull(LinkResolver.resolve("webview", "", d, "ap01"))
        assertNull(LinkResolver.resolve("webview", "javascript:alert(1)", d, "ap01"))
        assertNull(LinkResolver.resolve("webview", "//evil.com/x", d, "ap01"))
        assertEquals(
            LinkAction.OpenStore("market://details?id=com.x", "https://play.google.com/store/apps/details?id=com.x"),
            LinkResolver.resolve("store", "market://details?id=com.x", d, "ap01")
        )
        assertFalse(LinkResolver.isClickable("webview", "market://x"))
    }

    @Test fun sampleSizeKeepsAtLeastTarget() {
        assertEquals(1, AssetCache.sampleSize(600, 800, 600, 800))
        assertEquals(2, AssetCache.sampleSize(1200, 1600, 600, 800))
        assertEquals(4, AssetCache.sampleSize(2400, 3200, 600, 800))
        assertEquals(1, AssetCache.sampleSize(700, 900, 600, 800))
        assertEquals(1, AssetCache.sampleSize(100, 100, 0, 0))
    }

    @Test fun assetKeyIsSha256OfUrl() {
        assertEquals(64, AssetCache.sha256("https://a/b.png").length)
        assertNotEquals(AssetCache.sha256("https://a/b.png"), AssetCache.sha256("https://a/b2.png"))
    }

    @Test fun tabStoreCapReplacesOldestAndExpires() {
        val env = T.FakeEnv(); val kv = MemoryKv()
        val clock = TrustedClock(kv, env).also { it.recordAnchor(1_760_000_000_000L, 0) }
        val store = TabStore(kv, clock)
        store.add(T.popup(1)); store.add(T.popup(2))
        val evicted = store.add(T.popup(3))
        assertEquals(listOf(1L), evicted.map { it.popupId })
        assertEquals(listOf(2L, 3L), store.load().map { it.popupId })
        // P3 便条 24h 过期，P1 便条当日过期
        store.add(T.popup(4, PopupPosition.P3))
        env.elapsed += 25 * 3_600_000L
        assertTrue(store.load().none { it.popupId == 4L })
        env.elapsed += TrustedClock.DAY_MS
        assertTrue(store.load().isEmpty())
    }

    @Test fun supportsTabMatrix() {
        assertTrue(T.popup(1, PopupPosition.P1, tab = true).supportsTab)
        assertFalse(T.popup(1, PopupPosition.P1, tab = false).supportsTab)
        assertFalse(T.popup(1, PopupPosition.P4, closable = false, tab = true).supportsTab)
        assertFalse(T.popup(1, PopupPosition.P6, tab = true).supportsTab)
        assertFalse(T.popup(1, PopupPosition.P2, tab = true).supportsTab)
    }
}
