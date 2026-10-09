package com.hybrid.android.popup

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.net.ConnectivityManager
import android.net.Network
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import android.view.View
import android.widget.FrameLayout
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import com.hybrid.android.BuildConfig
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.withResumed
import com.hybrid.android.brand.BrandHost
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull

/**
 * 弹窗编排（契约 §7.4）：评估 → 过滤 → 互斥 → 素材就绪 → 渲染 → 埋点 / 频控 / 便条 / 跳转。
 *
 * 层级（自下而上）：WebView < 非遮罩层（P2/P3/P5/便条）< 遮罩层（P1/P4/P6/P7）< errorView。
 * 全部在主线程协程（Activity 的 lifecycleScope）里运行，弹窗从不阻塞 WebView 加载。
 */
class PopupManager(
    private val activity: ComponentActivity,
    private val host: BrandHost,
    private val rootLayout: FrameLayout,
    errorView: View,
    private val isErrorShowing: () -> Boolean,
    /** P6 渲染失败时回落到原「再按一次退出」流程。 */
    private val performDefaultBack: () -> Unit,
    private val scope: CoroutineScope,
    /** Activity 是否为配置变更后的重建（savedInstanceState != null）。 */
    recreated: Boolean,
    /** P6 显示时按返回：记 close(back) 后直接退出 App。 */
    private val exitApp: () -> Unit,
    /** 用 X / 遮罩关闭 P6 后：下一次顶层返回直接退出（不再出「Press back again」）。 */
    private val markExitOnNextBack: () -> Unit,
) {
    private val rt = PopupRuntime.get(activity)
    private val density = activity.resources.displayMetrics.density

    private val nonOverlayLayer = FrameLayout(activity)
    private val overlayLayer = FrameLayout(activity)
    private var tabsView: TabStripView? = null

    // ---- 状态 ----
    private class ShowSession(val popup: Popup, val display: String, val view: View) {
        val impressed = HashSet<Long>()
    }

    /** 配置变更重建时的恢复信息。 */
    private class RestoreInfo(val page: Int, val countdownElapsedMs: Long, val impressed: Set<Long>)

    private class TabRef(var popup: Popup, val independent: Boolean)

    /** 非 null = 本次是配置变更重建，需要恢复 UI；null = 新会话（会话级状态已重置）。 */
    private val restoreSnapshot: SessionSnapshot? = PopupSession.onActivityCreate(recreated)

    private var insetTop = 0
    private var insetBottom = 0
    private var tabGen = 0

    private var overlay: ShowSession? = null
    private var overlayBusy = false // 已选中、素材加载 / 解码中，尚未渲染
    private val nonOverlay = HashMap<PopupPosition, ShowSession>()
    private val tabRefs = ArrayList<TabRef>()

    private var coldRunning = false
    private var layersVisible = true
    private var insetsReceived = false
    private class PendingShow(val popup: Popup, val display: String, val job: Job)
    /** 已选中 / 已记 trigger、素材加载或等待 RESUMED 中的弹窗（重建时写进快照）。 */
    private val pendingShows = HashMap<Long, PendingShow>()
    private var netCallback: ConnectivityManager.NetworkCallback? = null
    private val handler = Handler(Looper.getMainLooper())
    private val scrollIdle = Runnable { tabsView?.notifyActivity() }

    init {
        val idx = rootLayout.indexOfChild(errorView).let { if (it < 0) rootLayout.childCount else it }
        rootLayout.addView(nonOverlayLayer, idx, FrameLayout.LayoutParams(matchParent(), matchParent()))
        rootLayout.addView(overlayLayer, idx + 1, FrameLayout.LayoutParams(matchParent(), matchParent()))
        // 与 WebView 同口径：用分发到弹窗层的 insets（而非 getRootWindowInsets 的未消费值）
        ViewCompat.setOnApplyWindowInsetsListener(nonOverlayLayer) { _, w ->
            val i = w.getInsets(WindowInsetsCompat.Type.systemBars())
            insetsReceived = true
            if (i.top != insetTop || i.bottom != insetBottom) {
                insetTop = i.top; insetBottom = i.bottom
                onInsetsChanged()
            }
            w
        }
    }

    private fun onInsetsChanged() {
        (nonOverlay[PopupPosition.P3]?.view?.layoutParams as? FrameLayout.LayoutParams)?.let {
            it.bottomMargin = insetBottom + (8 * density).toInt(); nonOverlay[PopupPosition.P3]?.view?.requestLayout()
        }
        (nonOverlay[PopupPosition.P5]?.view?.layoutParams as? FrameLayout.LayoutParams)?.let {
            it.topMargin = insetTop; nonOverlay[PopupPosition.P5]?.view?.requestLayout()
        }
        tabsView?.insetBottom = insetBottom
    }

    /** splash / errorView 显示期间隐藏整个弹窗层，恢复后再显示。 */
    fun setLayersVisible(visible: Boolean) {
        if (layersVisible == visible) return
        layersVisible = visible
        val v = if (visible) View.VISIBLE else View.INVISIBLE
        nonOverlayLayer.visibility = v
        overlayLayer.visibility = v
        // 隐藏期间暂停倒计时 / 曝光停留 / 轮播，恢复可见（且 Activity 在前台）再续
        if (!visible) pauseViews()
        else if (activity.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)) resumeViews()
    }

    // ============================== 生命周期入口 ==============================

    /** Activity.onCreate 末尾：冷启动必拉配置 + 补传积压埋点 + 监听网络恢复。 */
    fun start() {
        rt.refreshAsync()
        rt.flushEvents()
        registerNetwork()
        // 配置刷新后立即对账：已消失的弹窗 / 便条马上下线
        scope.launch { rt.repo.state.collect { reconcileWithConfig() } }
        // 重建后新 WebView 还在加载、splash 可见：层保持隐藏，直到页面完成
        if (restoreSnapshot != null) setLayersVisible(false)
        restoreSnapshot?.let { scope.launch { restoreUi(it) } }
    }

    fun onDestroy() {
        if (activity.isChangingConfigurations) PopupSession.saveForRecreate(buildSnapshot())
        handler.removeCallbacksAndMessages(null)
        netCallback?.let { cb ->
            runCatching { (activity.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager)?.unregisterNetworkCallback(cb) }
        }
        netCallback = null
    }

    fun onStart() {
        // 冷启动由首屏 onPageFinished 评估；lastStop 放在会话里，后台期间重建回前台也能判定 P1 间隔
        val last = PopupSession.lastStopElapsed
        PopupSession.lastStopElapsed = 0L // 消费：前台旋转等重建不得再触发回前台评估
        if (last == 0L) return
        val gap = SystemClock.elapsedRealtime() - last
        val refresh = if (rt.repo.needsForegroundRefresh()) rt.refreshAsync() else rt.currentRefresh()
        scope.launch {
            refresh?.let { withTimeoutOrNull(2_000) { it.join() } }
            evaluateResume(gap)
        }
    }

    fun onStop() {
        if (!activity.isChangingConfigurations) PopupSession.lastStopElapsed = SystemClock.elapsedRealtime()
        rt.flushEvents()
    }

    fun onPause() = pauseViews()

    fun onResume() { if (layersVisible) resumeViews() }

    private fun pauseViews() {
        (overlay?.view as? HostLifecycleAware)?.onHostPause()
        nonOverlay.values.forEach { (it.view as? HostLifecycleAware)?.onHostPause() }
        tabsView?.onHostPause()
    }

    private fun resumeViews() {
        (overlay?.view as? HostLifecycleAware)?.onHostResume()
        nonOverlay.values.forEach { (it.view as? HostLifecycleAware)?.onHostResume() }
        tabsView?.onHostResume()
    }

    /** WebView 每次 onPageFinished：首次触发冷启动评估；遮罩可见时给新页面重发「原生弹窗」标志。 */
    fun onPageFinished(url: String?, mainFrameFailed: Boolean) {
        if (mainFrameFailed || (url != null && url.startsWith("chrome-error://"))) {
            setLayersVisible(false)
            return
        }
        setLayersVisible(true)
        overlay?.let { notifyH5(true, it.popup.position) }
        if (!PopupSession.coldDone && !coldRunning && !isErrorShowing()) {
            coldRunning = true
            scope.launch { try { evaluateCold() } finally { coldRunning = false } }
        }
    }

    fun onWebViewTouch() { tabsView?.notifyActivity() }

    fun onWebViewScroll() {
        handler.removeCallbacks(scrollIdle)
        handler.postDelayed(scrollIdle, 150) // 滚动停止 150ms 后才恢复展开
    }

    /** 返回键最前置：返回 true 表示已消费。 */
    fun onBackPressed(): Boolean {
        if (isErrorShowing() || !layersVisible) return false // 错误页 / 层隐藏（splash）时不去关看不见的遮罩
        overlay?.let {
            if (it.popup.forced) return false // P4 强制：交给原「再按一次退出」
            closeOverlay("back")
            return true
        }
        if (overlayBusy) return false
        if (host.webView.canGoBack() || PopupSession.p6Evaluated) return false
        val cfg = rt.repo.effective ?: return false   // 配置没到：不消耗本会话 P6 额度
        if (!rt.clock.hasAnchor()) return false
        PopupSession.p6Evaluated = true // 真正完成评估才置位；同一会话只评估一次，避免每次返回都记 filtered
        val sel = PopupSelector.select(cfg.popups, setOf(PopupPosition.P6), ctx(), rt.frequency::allows)
        recordFiltered(sel)
        val p6 = sel.overlay ?: return false
        launchShow(p6, "auto", recordTrigger = true) { performDefaultBack() }
        return true
    }

    // ============================== 重建恢复 ==============================

    private fun buildSnapshot() = SessionSnapshot(
        overlay = overlay?.let {
            val v = it.view as? OverlayPopupView
            OverlaySnap(it.popup, it.display, v?.currentPage() ?: 0, v?.countdownElapsedMs() ?: 0L, HashSet(it.impressed))
        },
        nonOverlay = nonOverlay.values.map { NonOverlaySnap(it.popup, it.display, HashSet(it.impressed)) },
        tabs = tabRefs.map { TabSnap(it.popup, it.independent) },
        pending = pendingShows.values.map { PendingSnap(it.popup, it.display) },
    )

    /** 只恢复 UI：不记 trigger、不写频控、已记过的曝光不重复记，倒计时 / 卡片页延续。 */
    private suspend fun restoreUi(snap: SessionSnapshot) {
        val byId = rt.repo.effective?.popups?.associateBy { it.id }.orEmpty()
        snap.tabs.forEach { t ->
            val p = byId[t.popup.id] ?: return@forEach
            if (tabRefs.none { it.popup.id == p.id }) tabRefs.add(TabRef(p, t.independent))
        }
        refreshTabs()
        val jobs = ArrayList<Pair<Popup, Pair<String, RestoreInfo>>>()
        snap.overlay?.let { o -> byId[o.popup.id]?.let { jobs.add(it to (o.display to RestoreInfo(o.page, o.countdownElapsedMs, o.impressed))) } }
        snap.nonOverlay.forEach { n -> byId[n.popup.id]?.let { jobs.add(it to (n.display to RestoreInfo(0, 0L, n.impressed))) } }
        // 加载中的弹窗：trigger 已记过，新实例继续加载渲染（不重复记 trigger）
        snap.pending.forEach { pd -> byId[pd.popup.id]?.let { launchShow(it, pd.display, recordTrigger = false) } }
        awaitLayerReady()
        for ((popup, info) in jobs) {
            if (!rt.cache.ensureAll(popup.assetUrls(), ASSET_TIMEOUT_MS)) continue
            val bitmaps = withContext(Dispatchers.IO) { decodeAll(popup) }
            if (popup.cards.any { it.imageUrl.isNotBlank() && bitmaps[it.imageUrl] == null }) continue
            activity.lifecycle.withResumed {
                if (!activity.isFinishing && !activity.isDestroyed) render(popup, info.first, bitmaps, info.second)
            }
        }
    }

    // ============================== 评估 ==============================

    private fun ctx() = SelectionContext(BuildConfig.VERSION_CODE, rt.isNewUser, rt.now())

    private suspend fun evaluateCold() {
        rt.currentRefresh()?.let { withTimeoutOrNull(3_000) { it.join() } }
        // 等待结束后才置位（此后到 launch 完成之间没有挂起点）：等待期间被重建的新实例会重跑评估
        PopupSession.coldDone = true
        val cfg = rt.repo.effective ?: return // 无缓存也没拉到：不展示、不记事件
        if (!rt.clock.hasAnchor()) return
        restoreTabs(cfg)
        val sel = PopupSelector.select(cfg.popups, COLD_POSITIONS, ctx(), rt.frequency::allows)
        recordFiltered(sel)
        sel.overlay?.let { launchShow(it, "auto", recordTrigger = true) }
        sel.others.forEach { p ->
            if (p.position == PopupPosition.P8) addIndependentTab(p)
            else launchShow(p, "auto", recordTrigger = true)
        }
    }

    private fun evaluateResume(gapMs: Long) {
        if (!PopupSession.coldDone || overlay != null || overlayBusy || isErrorShowing()) return
        val cfg = rt.repo.effective ?: return
        if (!rt.clock.hasAnchor()) return
        val candidates = cfg.popups.filter {
            it.position == PopupPosition.P1 && gapMs > it.resumeGapMinutes * 60_000L
        }
        if (candidates.isEmpty()) return
        val sel = PopupSelector.select(candidates, setOf(PopupPosition.P1), ctx(), rt.frequency::allows)
        recordFiltered(sel)
        sel.overlay?.let { launchShow(it, "auto", recordTrigger = true) }
    }

    private fun recordFiltered(sel: Selection) {
        sel.filtered.forEach { rt.track(PopupEvents.filtered(it.popup, it.reason, rt.now())) }
    }

    // ============================== 展示流程 ==============================

    /** trigger → 素材就绪（3s）→ 渲染。失败记 load_fail 且不展示。 */
    private fun launchShow(popup: Popup, display: String, recordTrigger: Boolean, onFail: (() -> Unit)? = null) {
        val isOverlay = popup.position.overlay
        if (isOverlay) {
            if (overlay != null || overlayBusy) return
            overlayBusy = true
        }
        val job = scope.launch(start = CoroutineStart.LAZY) {
            try {
                if (recordTrigger) rt.track(PopupEvents.trigger(popup, rt.now()))
                val ready = rt.cache.ensureAll(popup.assetUrls(), ASSET_TIMEOUT_MS)
                if (!ready) {
                    rt.track(PopupEvents.loadFail(popup, rt.now()))
                    onFail?.invoke()
                    return@launch
                }
                val bitmaps = withContext(Dispatchers.IO) { decodeAll(popup) }
                // 卡片图解码失败（坏文件 / OOM）：删缓存、记 load_fail、不渲染
                val bad = popup.cards.map { it.imageUrl }.filter { it.isNotBlank() && bitmaps[it] == null }
                if (bad.isNotEmpty()) {
                    withContext(Dispatchers.IO) { bad.forEach { rt.cache.invalidate(it) } }
                    rt.track(PopupEvents.loadFail(popup, rt.now()))
                    onFail?.invoke()
                    return@launch
                }
                awaitLayerReady()
                // 只在 RESUMED 渲染：权限框 / Home 之后不在后台渲染、不跑倒计时、不记曝光
                activity.lifecycle.withResumed {
                    if (activity.isFinishing || activity.isDestroyed) return@withResumed
                    // 等待期间配置 / 时间 / 频控可能已变：渲染前复核，不满足则放弃（不记 load_fail）
                    if (!RenderGuard.eligible(popup, rt.repo.effective, ctx(), display == "auto", rt.frequency::allows, rt.frequency::isDismissedToday)) return@withResumed
                    render(popup, display, bitmaps, null)
                }
            } finally {
                pendingShows.remove(popup.id)
                if (isOverlay) overlayBusy = false
            }
        }
        pendingShows[popup.id] = PendingShow(popup, display, job)
        job.start()
    }

    /** 弹窗层拿到 insets 并完成布局后再渲染（P7 关闭按钮不能落进状态栏）；最多等 1s。 */
    private suspend fun awaitLayerReady() {
        var waited = 0
        while (!(insetsReceived && nonOverlayLayer.width > 0) && waited < 1_000) { delay(16); waited += 16 }
    }

    private fun decodeAll(popup: Popup): Map<String, Bitmap?> {
        val sw = screenW()
        val sh = screenH()
        val sizes = HashMap<String, Pair<Int, Int>>()
        fun want(url: String, w: Int, h: Int) {
            if (url.isBlank()) return
            val old = sizes[url]
            sizes[url] = if (old == null) w to h else maxOf(old.first, w) to maxOf(old.second, h)
        }
        val (cw, ch) = when (popup.position) {
            PopupPosition.P7 -> sw to sh
            PopupPosition.P2 -> (60 * density).toInt() to (60 * density).toInt()
            PopupPosition.P3, PopupPosition.P8 -> (40 * density).toInt() to (40 * density).toInt()
            PopupPosition.P5 -> 1 to 1
            else -> minOf((300 * density).toInt(), (sw * 0.84f).toInt()) to (minOf((300 * density).toInt(), (sw * 0.84f).toInt()) * 4 / 3)
        }
        popup.cards.forEach { want(it.imageUrl, cw, ch) }
        return sizes.mapValues { (u, s) -> rt.cache.decode(u, s.first, s.second) }
    }

    private fun render(popup: Popup, display: String, bitmaps: Map<String, Bitmap?>, restore: RestoreInfo?) {
        val insets = insetTop to insetBottom
        val session: ShowSession
        when (popup.position) {
            PopupPosition.P1, PopupPosition.P4, PopupPosition.P6, PopupPosition.P7 -> {
                if (overlay != null) return
                val view = OverlayPopupView(
                    activity, popup, bitmaps,
                    runCountdown = popup.countdown && display == "auto",
                    insetTop = insets.first, insetBottom = insets.second, screenW = screenW(), screenH = screenH(),
                    listener = overlayListener,
                    initialPage = restore?.page ?: 0,
                    initialCountdownElapsedMs = restore?.countdownElapsedMs ?: 0L,
                    startResumed = activity.lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED),
                )
                session = ShowSession(popup, display, view)
                restore?.let { session.impressed.addAll(it.impressed) }
                overlay = session
                overlayLayer.addView(view, FrameLayout.LayoutParams(matchParent(), matchParent()))
                notifyH5(true, popup.position)
                tabsView?.setHidden(popup.id, true)
            }
            PopupPosition.P2 -> {
                nonOverlay.remove(PopupPosition.P2)?.let { nonOverlayLayer.removeView(it.view) }
                val view = FloatBallView(activity, popup, bitmaps[popup.cards.first().imageUrl], insets.first, insets.second,
                    nonOverlayListener(PopupPosition.P2))
                session = ShowSession(popup, display, view)
                restore?.let { session.impressed.addAll(it.impressed) }
                nonOverlay[PopupPosition.P2] = session
                view.visibility = View.INVISIBLE // 先定位再显示，避免首帧闪在 (0,0)
                nonOverlayLayer.addView(view, FrameLayout.LayoutParams(view.frameSize, view.frameSize))
                view.post {
                    view.placeInitially(nonOverlayLayer.width.takeIf { it > 0 } ?: screenW(), nonOverlayLayer.height.takeIf { it > 0 } ?: screenH())
                    view.visibility = View.VISIBLE
                }
            }
            PopupPosition.P3 -> {
                nonOverlay.remove(PopupPosition.P3)?.let { nonOverlayLayer.removeView(it.view) }
                val view = BannerView(activity, popup, bitmaps, nonOverlayListener(PopupPosition.P3))
                session = ShowSession(popup, display, view)
                restore?.let { session.impressed.addAll(it.impressed) }
                nonOverlay[PopupPosition.P3] = session
                nonOverlayLayer.addView(view, FrameLayout.LayoutParams(matchParent(), (64 * density).toInt(), android.view.Gravity.BOTTOM).apply {
                    leftMargin = (10 * density).toInt(); rightMargin = (10 * density).toInt()
                    bottomMargin = insets.second + (8 * density).toInt()
                })
                tabsView?.setHidden(popup.id, true)
            }
            PopupPosition.P5 -> {
                nonOverlay.remove(PopupPosition.P5)?.let { nonOverlayLayer.removeView(it.view) }
                val view = TopBarView(activity, popup, nonOverlayListener(PopupPosition.P5))
                session = ShowSession(popup, display, view)
                restore?.let { session.impressed.addAll(it.impressed) }
                nonOverlay[PopupPosition.P5] = session
                nonOverlayLayer.addView(view, FrameLayout.LayoutParams(matchParent(), (44 * density).toInt(), android.view.Gravity.TOP).apply {
                    topMargin = insets.first
                })
            }
            PopupPosition.P8 -> return
        }
        if (!layersVisible) (session.view as? HostLifecycleAware)?.onHostPause()
        // 频控：真实渲染成功时写入；便条重开不写。
        if (display == "auto" && restore == null) rt.frequency.onRendered(popup) // 重建恢复不重复写频控
        Log.d(TAG, "渲染 ${popup.position.code} id=${popup.id} display=$display")
    }

    // ============================== 回调 ==============================

    private val overlayListener = object : OverlayPopupView.Listener {
        override fun onCardDwell(index: Int) { overlay?.let { recordImpression(it, index) } }
        override fun onSlide(from: Int, to: Int) {
            overlay?.let { rt.track(PopupEvents.slide(it.popup, from, to, rt.now())) }
        }
        override fun onCta(index: Int) { overlay?.let { clickOverlay(it, index) } }
        override fun onCloseRequested(method: String) { closeOverlay(method) }
    }

    private fun nonOverlayListener(pos: PopupPosition) = object : NonOverlayListener {
        override fun onCardDwell(index: Int) { nonOverlay[pos]?.let { recordImpression(it, index) } }
        override fun onCta(index: Int) {
            val s = nonOverlay[pos] ?: return
            val card = s.popup.cards.getOrNull(index) ?: return
            if (!LinkResolver.isClickable(s.popup.openMode, card.linkUrl)) return // 链接空 / 非法：不可点、不记 click
            rt.track(PopupEvents.click(s.popup, card, index, s.display, rt.now()))
            openLink(s.popup, card)
        }
        override fun onCloseRequested() { closeNonOverlay(pos) }
    }

    private fun recordImpression(s: ShowSession, index: Int) {
        val card = s.popup.cards.getOrNull(index) ?: return
        if (s.impressed.add(card.id)) { // 同一次展示同一卡片只记一次
            rt.track(PopupEvents.impression(s.popup, card, index, s.display, rt.now()))
        }
    }

    /** 点击 CTA / 整图：popup_click → 关闭弹窗（不收起为便条）→ 跳转。 */
    private fun clickOverlay(s: ShowSession, index: Int) {
        val card = s.popup.cards.getOrNull(index) ?: return
        rt.track(PopupEvents.click(s.popup, card, index, s.display, rt.now()))
        removeOverlay(s)
        openLink(s.popup, card)
    }

    /** 关闭（按钮 / 遮罩 / 返回）：popup_close → 便条态开则 popup_collapse 并生成 / 恢复便条。 */
    private fun closeOverlay(method: String) {
        val s = overlay ?: return
        rt.track(PopupEvents.close(s.popup, method, rt.now()))
        rt.frequency.onClosed(s.popup)
        removeOverlay(s)
        if (s.popup.position == PopupPosition.P6) {
            when (P6Exit.afterClose(method)) {
                P6Exit.Action.EXIT_NOW -> exitApp()
                P6Exit.Action.EXIT_ON_NEXT_BACK -> markExitOnNextBack()
            }
            return
        }
        collapseIfNeeded(s.popup)
    }

    private fun closeNonOverlay(pos: PopupPosition) {
        val s = nonOverlay.remove(pos) ?: return
        rt.track(PopupEvents.close(s.popup, "button", rt.now()))
        rt.frequency.onClosed(s.popup)
        nonOverlayLayer.removeView(s.view)
        collapseIfNeeded(s.popup)
    }

    private fun collapseIfNeeded(popup: Popup) {
        if (!popup.supportsTab) return
        rt.track(PopupEvents.collapse(popup, rt.now()))
        addTab(popup)
    }

    private fun removeOverlay(s: ShowSession) {
        overlayLayer.removeView(s.view)
        if (overlay === s) overlay = null
        notifyH5(false, s.popup.position)
        tabsView?.setHidden(s.popup.id, false)
    }

    // ============================== 便条 ==============================

    private val tabListener = object : TabStripView.Listener {
        override fun onTabClick(key: Long) {
            val ref = tabRefs.firstOrNull { it.popup.id == key } ?: return
            // 以当前有效配置为准：弹窗已下线 → 移除便条、不展示
            val latest = rt.repo.effective?.popups?.firstOrNull { it.id == key }
            if (latest == null) { removeTab(key); return }
            ref.popup = latest
            // 遮罩正在显示 / 加载中：不记 tab_click（避免只记点击不展示）
            if (!ref.independent && (latest.position.overlay && (overlay != null || overlayBusy) || nonOverlay.containsKey(latest.position))) return
            rt.track(PopupEvents.tabClick(ref.popup, rt.now()))
            if (ref.independent) {
                ref.popup.cards.firstOrNull()?.let { openLink(ref.popup, it) }
            } else {
                // 重新展示弹窗：不计 trigger、不重复写频控、不重跑倒计时
                launchShow(ref.popup, "tab", recordTrigger = false)
            }
        }

        override fun onTabLongPress(key: Long) {
            val ref = tabRefs.firstOrNull { it.popup.id == key } ?: return
            Toast.makeText(activity, "Hidden for today", Toast.LENGTH_SHORT).show()
            rt.track(PopupEvents.tabDismiss(ref.popup, rt.now()))
            rt.frequency.dismissToday(ref.popup)
            removeTab(ref.popup.id)
        }

        override fun onTabShown(key: Long) {
            val ref = tabRefs.firstOrNull { it.popup.id == key } ?: return
            if (PopupSession.tabImpressedAdd(key)) rt.track(PopupEvents.tabImpression(ref.popup, rt.now()))
        }
    }

    private fun ensureTabsView(): TabStripView =
        tabsView ?: TabStripView(activity, screenH(), tabListener).also {
            tabsView = it
            nonOverlayLayer.addView(it, FrameLayout.LayoutParams(matchParent(), matchParent()))
        }.also { it.insetBottom = insetBottom }

    private fun addTab(popup: Popup) {
        rt.tabs.add(popup)
        tabRefs.removeAll { it.popup.id == popup.id && !it.independent }
        tabRefs.add(TabRef(popup, false))
        trimTabs()
        refreshTabs()
    }

    private fun addIndependentTab(popup: Popup) {
        if (tabRefs.any { it.popup.id == popup.id }) return
        tabRefs.add(TabRef(popup, true))
        trimTabs()
        refreshTabs()
    }

    /** 同时最多 2 条，第 3 条替换最旧。 */
    private fun trimTabs() {
        while (tabRefs.size > TabStore.MAX_TABS) {
            val old = tabRefs.removeAt(0)
            rt.tabs.remove(old.popup.id)
            PopupSession.tabImpressedRemove(old.popup.id)
        }
    }

    private fun removeTab(id: Long) {
        tabRefs.removeAll { it.popup.id == id }
        rt.tabs.remove(id)
        PopupSession.tabImpressedRemove(id)
        refreshTabs()
    }

    private fun restoreTabs(cfg: PopupConfig) {
        val byId = cfg.popups.associateBy { it.id }
        for (e in rt.tabs.load()) {
            val p = byId[e.popupId]
            if (p == null || !p.supportsTab || rt.frequency.isDismissedToday(p)) { rt.tabs.remove(e.popupId); continue }
            if (tabRefs.none { it.popup.id == p.id }) tabRefs.add(TabRef(p, false))
        }
        trimTabs()
        refreshTabs()
    }

    /** 配置刷新后：下线已不存在 / 已过期的便条与非遮罩弹窗，并把便条换成最新版本。 */
    private fun reconcileWithConfig() {
        val cfg = rt.repo.effective ?: return
        val byId = cfg.popups.associateBy { it.id }
        val valid = rt.tabs.load().map { it.popupId }.toSet()
        var changed = false
        val it = tabRefs.iterator()
        while (it.hasNext()) {
            val ref = it.next()
            val latest = byId[ref.popup.id]
            val stillValid = latest != null && (ref.independent || (ref.popup.id in valid && latest.supportsTab))
            if (!stillValid) { it.remove(); if (!ref.independent) rt.tabs.remove(ref.popup.id); PopupSession.tabImpressedRemove(ref.popup.id); changed = true }
            else if (latest !== ref.popup) { ref.popup = latest!!; changed = true }
        }
        if (changed) refreshTabs()
        pendingShows.values.toList().forEach { if (it.popup.id !in byId) it.job.cancel() } // 加载中 / 等待 RESUMED 的也下线
        overlay?.let { if (it.popup.id !in byId) removeOverlay(it) } // 已下线的遮罩：关闭，不收起便条
        nonOverlay.entries.toList().forEach { (pos, s) ->
            if (s.popup.id !in byId) { nonOverlay.remove(pos); nonOverlayLayer.removeView(s.view) }
        }
    }

    private fun refreshTabs() {
        val gen = ++tabGen
        if (tabRefs.isEmpty()) { tabsView?.setItems(emptyList()); return }
        val refs = tabRefs.toList()
        scope.launch {
            val items = withContext(Dispatchers.IO) {
                val px = (20 * density).toInt()
                refs.map { r ->
                    TabStripView.Item(r.popup.id, r.popup.tabText.ifBlank { r.popup.cards.firstOrNull()?.title.orEmpty() },
                        r.popup.tabIcon.takeIf { it.isNotBlank() }?.let { rt.cache.decode(it, px, px) })
                }
            }
            if (gen != tabGen) return@launch // 只应用最新一次，旧快照不得覆盖新列表
            val view = ensureTabsView()
            view.setItems(items)
            // 正在展示中的弹窗对应的便条保持隐藏
            overlay?.let { view.setHidden(it.popup.id, true) }
            nonOverlay.values.forEach { view.setHidden(it.popup.id, true) }
        }
    }

    // ============================== 跳转 / H5 / 杂项 ==============================

    private fun openLink(popup: Popup, card: PopupCard) {
        val action = LinkResolver.resolve(popup.openMode, card.linkUrl, host.domain, BuildConfig.PAL_CODE) ?: return
        when (action) {
            is LinkAction.LoadInWebView ->
                if (action.external) host.openWebUrl(action.url) else host.webView.loadUrl(host.decorateLoadUrl(action.url))
            is LinkAction.OpenBrowser -> view(action.url)
            is LinkAction.OpenStore -> if (!view(action.url) && action.fallbackUrl != null) view(action.fallbackUrl)
        }
    }

    private fun view(url: String): Boolean = try {
        activity.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        true
    } catch (e: ActivityNotFoundException) {
        Log.w(TAG, "无应用可打开: $url"); false
    } catch (e: Exception) {
        Log.w(TAG, "打开失败: $url ${e.message}"); false
    }

    /** 遮罩类显示 / 隐藏时通知 H5 暂停自身弹窗。 */
    private fun notifyH5(visible: Boolean, pos: PopupPosition) {
        val js = "window.__HYBRID_NATIVE_POPUP__=$visible;" +
            "window.dispatchEvent(new CustomEvent('hybrid:native-popup',{detail:{visible:$visible,position:'${pos.code}'}}));"
        runCatching { host.webView.evaluateJavascript(js, null) }
    }

    private fun screenW() = rootLayout.width.takeIf { it > 0 } ?: activity.resources.displayMetrics.widthPixels
    private fun screenH() = rootLayout.height.takeIf { it > 0 } ?: activity.resources.displayMetrics.heightPixels

    private fun registerNetwork() {
        val cm = activity.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return
        val cb = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) { rt.flushEvents() } // 网络恢复补传
        }
        runCatching { cm.registerDefaultNetworkCallback(cb) }.onSuccess { netCallback = cb }
    }

    companion object {
        private const val TAG = "HybridPopup"
        private const val ASSET_TIMEOUT_MS = 3_000L
        private val COLD_POSITIONS = setOf(
            PopupPosition.P4, PopupPosition.P7, PopupPosition.P1,
            PopupPosition.P2, PopupPosition.P3, PopupPosition.P5, PopupPosition.P8,
        )
    }
}
