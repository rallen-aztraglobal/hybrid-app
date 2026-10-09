package com.hybrid.android.popup

/** 配置变更重建时需要原样恢复的 UI 状态（不记 trigger / 不写频控 / 不重复记曝光）。 */
data class OverlaySnap(val popup: Popup, val display: String, val page: Int, val countdownElapsedMs: Long, val impressed: Set<Long>)
data class NonOverlaySnap(val popup: Popup, val display: String, val impressed: Set<Long>)
data class TabSnap(val popup: Popup, val independent: Boolean)
/** 已选中并记过 trigger、但素材加载 / 等待 RESUMED 中尚未渲染的弹窗：重建后继续加载渲染，不重复记 trigger。 */
data class PendingSnap(val popup: Popup, val display: String)
data class SessionSnapshot(
    val overlay: OverlaySnap?,
    val nonOverlay: List<NonOverlaySnap>,
    val tabs: List<TabSnap>,
    val pending: List<PendingSnap> = emptyList(),
)

/**
 * 进程级「会话」状态。区分两种 Activity 创建：
 *  - 新启动（savedInstanceState == null，或进程被杀后恢复、没有重建快照）= 新会话：重置会话级状态，照常冷启动评估；
 *  - 配置变更重建（旋转 / 深色模式…，上一个实例在 onDestroy 里留下了快照）：保留状态，只恢复 UI。
 */
object PopupSession {
    @Volatile var coldDone = false
    @Volatile var p6Evaluated = false
    /** 上次进入后台的 elapsedRealtime；放在会话里，后台期间发生重建（深色模式 / 字体）回前台仍可判定 P1 间隔。0 = 无。 */
    @Volatile var lastStopElapsed = 0L
    private val tabImpressedSet = HashSet<Long>()
    @Volatile private var snapshot: SessionSnapshot? = null

    fun tabImpressedAdd(id: Long): Boolean = synchronized(tabImpressedSet) { tabImpressedSet.add(id) }
    fun tabImpressedRemove(id: Long) { synchronized(tabImpressedSet) { tabImpressedSet.remove(id) } }

    /** Activity 重建前（onDestroy 且 isChangingConfigurations）留下快照。 */
    fun saveForRecreate(s: SessionSnapshot) { snapshot = s }

    /** onCreate 调用。返回非 null = 配置变更重建，需恢复该快照；null = 新会话（已重置）。 */
    fun onActivityCreate(hasSavedState: Boolean): SessionSnapshot? {
        val snap = snapshot
        snapshot = null
        if (hasSavedState && snap != null) return snap
        reset()
        return null
    }

    fun reset() {
        coldDone = false
        p6Evaluated = false
        lastStopElapsed = 0L
        synchronized(tabImpressedSet) { tabImpressedSet.clear() }
        snapshot = null
        FrequencyPolicy.resetSession()
    }
}

/** P6 关闭后的退出步数（产品决定：PRD「第二次返回直接退出」）。 */
object P6Exit {
    enum class Action { EXIT_NOW, EXIT_ON_NEXT_BACK }

    /** 返回键关闭 → 直接退出 App；X / 遮罩关闭 → 下一次顶层返回直接退出（不再出「Press back again」）。 */
    fun afterClose(method: String): Action =
        if (method == "back") Action.EXIT_NOW else Action.EXIT_ON_NEXT_BACK
}

/**
 * 渲染前复核（冷启动等待 / 等待 RESUMED 期间配置、时间、频控可能已变）：
 * 弹窗须仍在在用配置里、未被当日长按关闭；[enforceRules] 时还要重新满足 targeting / time / 频控。
 * 不满足则放弃渲染（不记 load_fail）。便条重开与重建恢复不强制规则（只要求仍在配置里）。
 */
object RenderGuard {
    fun eligible(
        popup: Popup,
        effective: PopupConfig?,
        ctx: SelectionContext,
        enforceRules: Boolean,
        frequencyAllows: (Popup) -> Boolean,
        isDismissed: (Popup) -> Boolean,
    ): Boolean {
        val cur = effective?.popups?.firstOrNull { it.id == popup.id } ?: return false
        if (isDismissed(cur)) return false
        if (!enforceRules) return true
        return PopupSelector.targetingOk(cur, ctx) && PopupSelector.timeOk(cur, ctx.nowMs) && frequencyAllows(cur)
    }
}

/** P6 用 X / 遮罩关闭后「紧接着的那次顶层返回直接退出」：发生导航或进入后台即解除。 */
class ExitGate {
    @Volatile var armed = false
        private set
    fun arm() { armed = true }
    fun disarm() { armed = false }
}
