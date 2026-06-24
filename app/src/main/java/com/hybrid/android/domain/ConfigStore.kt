package com.hybrid.android.domain

/**
 * 域名清单的持久化与兜底来源。抽成接口让 STEP1 三级取用逻辑（[DomainListSelector]）
 * 可脱离 Android（SharedPreferences/assets）做 JVM 单测。
 */
interface ConfigStore {
    /** 读本地缓存（上一次成功拉取的配置）。无则 null。 */
    fun readCachedConfig(): DomainConfig?

    /** 写本地缓存（实时拉取成功后立即覆盖，使兜底=最近一次成功配置）。 */
    fun writeCachedConfig(config: DomainConfig)

    /** 上次真正加载成功的域名（lastGood），用于提到队首加速首屏。无则 null。 */
    fun readLastGood(): String?

    /** 记录 lastGood。 */
    fun writeLastGood(domain: String)

    /** 编译期兜底 `assets/bootstrap.json`。仅「从未成功拉取过」时使用。 */
    fun readBootstrap(): BootstrapConfig?
}

/**
 * STEP1 的纯逻辑：三级取用 + lastGood 提队首。与网络/存储解耦，便于单测。
 *
 * 取用优先级（docs/admin/02-domain-failover.md STEP1）：
 *  1. 实时接口成功 → 用返回清单，并**立即写缓存**（覆盖旧值）。
 *  2. 接口失败 → 用本地缓存（上一次成功返回的配置）。
 *  3. 从未成功过 → 用编译期 `assets/bootstrap.json`。
 * 最后把 lastGood 提到队首（不改变「主域名优先」语义，仅加速 failover）。
 */
class DomainListSelector(
    private val store: ConfigStore,
) {
    /**
     * @param fetch 实时拉取闭包（注入，便于测试）。返回 null 视为失败。
     * @return 已排好序（lastGood 提队首）的 (域名清单, probePath)。空清单表示彻底无可用来源。
     */
    fun select(fetch: () -> DomainConfig?): Selection {
        // ① 实时拉取
        val fresh = fetch()
        if (fresh != null && fresh.domains.isNotEmpty()) {
            store.writeCachedConfig(fresh)        // 成功即自更新兜底缓存
            return finalize(fresh)
        }
        // ② 本地缓存
        val cached = store.readCachedConfig()
        if (cached != null && cached.domains.isNotEmpty()) {
            return finalize(cached)
        }
        // ③ 编译期兜底
        val boot = store.readBootstrap()
        if (boot != null && boot.defaultDomains.isNotEmpty()) {
            return finalize(DomainConfig(boot.defaultDomains, boot.probePath))
        }
        return Selection(emptyList(), "/healthz")
    }

    private fun finalize(config: DomainConfig): Selection {
        val ordered = orderByLastGood(config.domains, store.readLastGood())
        return Selection(ordered, config.probePath)
    }

    data class Selection(val domains: List<String>, val probePath: String)

    companion object {
        /** 把 lastGood 提到队首；保持其余相对顺序（主域名优先语义不变，仅加速）。 */
        fun orderByLastGood(domains: List<String>, lastGood: String?): List<String> {
            if (lastGood.isNullOrBlank() || !domains.contains(lastGood)) return domains
            return listOf(lastGood) + domains.filter { it != lastGood }
        }
    }
}
