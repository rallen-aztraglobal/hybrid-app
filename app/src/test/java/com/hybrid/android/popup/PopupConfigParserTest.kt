package com.hybrid.android.popup

import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class PopupConfigParserTest {
    private val json = """
    {"appId":"com.arenaplus.ap01001","configVersion":"abc","serverTime":1760000000000,"tzOffsetMinutes":480,
     "popups":[
      {"id":12,"position":"P1","priority":100,"startAt":1759248000000,"endAt":0,"minVersionCode":10003,"maxVersionCode":0,
       "userType":"all","openMode":"webview","closable":true,"maskClosable":true,"countdown":true,"tabEnabled":true,
       "tabIconUrl":"https://t.png","tabText":"Bonus","autoplaySeconds":5,"resumeGapMinutes":30,"badge":false,
       "cards":[{"id":34,"imageUrl":"https://a.png","linkUrl":"/promo","buttonText":"Claim","title":"","description":null}]},
      {"id":13,"position":"P99","cards":[{"id":1,"imageUrl":"x"}]},
      {"id":14,"position":"P4","closable":false,"maskClosable":true,"tabEnabled":true,"countdown":true,"cards":[{"id":2,"title":"Hi"}]},
      {"id":15,"position":"P2","cards":[]}
     ]}"""

    @Test fun parsesFieldsAndIgnoresUnknownPositionAndEmptyCards() {
        val cfg = PopupConfigParser.parse(json)!!
        assertEquals("abc", cfg.configVersion)
        assertEquals(480, cfg.tzOffsetMinutes)
        assertEquals(listOf(12L, 14L), cfg.popups.map { it.id })
        val p = cfg.popups[0]
        assertEquals(PopupPosition.P1, p.position)
        assertTrue(p.countdown && p.maskClosable && p.tabEnabled)
        assertEquals("", p.cards[0].description) // JSON null 不能变成 "null"
        assertEquals(listOf("https://a.png", "https://t.png"), p.assetUrls())
    }

    @Test fun forcedP4IsNormalized() {
        val p = PopupConfigParser.parse(json)!!.popups[1]
        assertTrue(p.forced)
        assertFalse(p.maskClosable); assertFalse(p.tabEnabled); assertFalse(p.countdown)
        assertTrue(p.assetUrls().isEmpty()) // 无图
    }

    @Test fun invalidJsonReturnsNull() {
        assertNull(PopupConfigParser.parse("<html>"))
        assertNull(PopupConfigParser.parse(null))
        assertNull(PopupConfigParser.parse(""))
    }

    @Test fun roundTripThroughToJson() {
        val cfg = PopupConfigParser.parse(json)!!
        val again = PopupConfigParser.parse(cfg.toJson())!!
        assertEquals(cfg.popups, again.popups)
    }

    @Test fun missingPopupsArrayIsEmptyList() {
        assertEquals(0, PopupConfigParser.parse("""{"appId":"a","configVersion":"v"}""")!!.popups.size)
    }
}
