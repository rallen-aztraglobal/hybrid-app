package com.hybrid.android.popup

/**
 * 按弹窗粒度选用就绪版本（契约 §7.1）：
 *  - 新配置里某弹窗图片全部就绪 → 用新版本；
 *  - 未就绪且「在用配置」里有该弹窗且其图片仍就绪 → 沿用旧版本；
 *  - 都没有 → 本次不参与；
 *  - 新配置里已不存在的弹窗 → 立即下线（不会出现在结果里）。
 */
object ReadinessMerge {
    fun merge(fresh: List<Popup>, previous: List<Popup>, isReady: (String) -> Boolean): List<Popup> {
        val prevById = previous.associateBy { it.id }
        val out = ArrayList<Popup>()
        for (p in fresh) {
            if (p.assetUrls().all(isReady)) {
                out.add(p)
            } else {
                val old = prevById[p.id]
                if (old != null && old.assetUrls().all(isReady)) out.add(old)
            }
        }
        return out
    }

    /** 磁盘上需要保留的素材 URL：新配置全部 + 在用版本全部。 */
    fun referencedUrls(fresh: List<Popup>, effective: List<Popup>): Set<String> {
        val s = HashSet<String>()
        fresh.forEach { s.addAll(it.assetUrls()) }
        effective.forEach { s.addAll(it.assetUrls()) }
        return s
    }
}
