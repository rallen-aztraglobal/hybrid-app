package com.hybrid.android.brand

import com.hybrid.android.BuildConfig

/** 按 BuildConfig.BRAND 选取对应的大渠道策略实现。 */
object BrandStrategies {
    fun create(): BrandStrategy = when (BuildConfig.BRAND) {
        "bp" -> BpStrategy()
        "ap", "gp" -> StandardStrategy()
        // 新增大渠道时必须在此登记，避免漏配后静默走标准逻辑导致归因错误
        else -> error("未登记的大渠道: ${BuildConfig.BRAND}")
    }
}
