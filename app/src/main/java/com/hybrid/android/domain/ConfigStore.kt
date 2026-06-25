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

    /** 编译期兜底 `assets/bootstrap.json`。仅「从未成功拉取过」时使用。 */
    fun readBootstrap(): BootstrapConfig?
}

/**
 * STEP1 的纯逻辑：三级取用。与网络/存储解耦，便于单测。
 *
 * 取用优先级（docs/admin/02-domain-failover.md STEP1）：
 *  1. 实时接口成功 → 用返回清单，并**立即写缓存**（覆盖旧值）。
 *  2. 接口失败 → 用本地缓存（上一次成功返回的配置）。
 *  3. 从未成功过 → 用编译期 `assets/bootstrap.json`。
 * 顺序一律以来源（console / 缓存 / bootstrap）给出的为准——主域名在队首，
 * 主域名优先由 [DomainResolver] 的 STEP2 保证（不再按 lastGood 重排，避免改主不生效）。
 */
class DomainListSelector(
    private val store: ConfigStore,
) {
    /**
     * @param fetch 实时拉取闭包（注入，便于测试）。返回 null 视为失败。
     * @return (域名清单, probePath)，顺序即来源给出的顺序（主在队首）。空清单表示彻底无可用来源。
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
        // 顺序以 console 配置为准（主在队首），不再按 lastGood 重排——
        // 否则旧的 lastGood 会顶掉新设的主域名，导致「后台改主域名不生效」。
        return Selection(config.domains, config.probePath)
    }

    data class Selection(val domains: List<String>, val probePath: String)
}
