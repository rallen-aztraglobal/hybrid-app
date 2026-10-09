package com.hybrid.android.popup

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.Outline
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewOutlineProvider
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import kotlin.math.ceil
import kotlin.math.min

/**
 * 遮罩类弹窗（P1 / P4 / P6 居中卡片，P7 全屏）。纯 View 代码构建。
 *
 * 职责：布局、轮播、CTA、关闭按钮 + 3 秒倒计时、卡片停留 ≥1s 回调；
 * 不碰埋点 / 频控 / 跳转——全部经 [Listener] 交回 [PopupManager]。
 */
class OverlayPopupView(
    context: Context,
    private val popup: Popup,
    private val bitmaps: Map<String, Bitmap?>,
    /** 是否跑倒计时（P1/P7 且 countdown，且不是便条重开）。 */
    private val runCountdown: Boolean,
    private val insetTop: Int,
    private val insetBottom: Int,
    private val screenW: Int,
    private val screenH: Int,
    private val listener: Listener,
    /** 配置变更重建时恢复：原来所在的卡片页、已走过的倒计时。 */
    private val initialPage: Int = 0,
    private val initialCountdownElapsedMs: Long = 0L,
    startResumed: Boolean = true,
) : FrameLayout(context), HostLifecycleAware {

    interface Listener {
        /** 某卡片可见停留 ≥1s。 */
        fun onCardDwell(index: Int)
        fun onSlide(from: Int, to: Int)
        /** 用户点击了 CTA / 整图（该卡片链接可点）。 */
        fun onCta(index: Int)
        /** method = button | mask | back */
        fun onCloseRequested(method: String)
    }

    private val handler = Handler(Looper.getMainLooper())
    private val fullscreen = popup.position == PopupPosition.P7
    private var pager: SwipePager? = null
    private var dots: DotsView? = null
    private var closeBtn: CloseButtonView? = null

    // ---- 倒计时 ----
    private var cdElapsedMs = initialCountdownElapsedMs
    private var cdLastTick = 0L
    private var cdRunning = false
    @Volatile var countdownActive = runCountdown && !popup.forced && initialCountdownElapsedMs < COUNTDOWN_MS
        private set
    private val tick = object : Runnable {
        override fun run() {
            if (!cdRunning) return
            val now = SystemClock.uptimeMillis()
            cdElapsedMs += now - cdLastTick
            cdLastTick = now
            if (cdElapsedMs >= COUNTDOWN_MS) {
                countdownActive = false
                cdRunning = false
                closeBtn?.showNormal()
                return
            }
            val remain = ceil((COUNTDOWN_MS - cdElapsedMs) / 1000.0).toInt()
            closeBtn?.showCountdown(remain, cdElapsedMs / COUNTDOWN_MS.toFloat())
            handler.postDelayed(this, 50)
        }
    }

    // ---- 卡片停留 ----
    private var dwellIndex = -1
    private val dwell = Runnable { if (dwellIndex >= 0) listener.onCardDwell(dwellIndex) }
    private var resumed = startResumed

    init {
        isClickable = true // 吞掉遮罩层触摸，防止穿透到 WebView
        if (fullscreen) buildFullscreen() else buildCard()
        setOnClickListener { onMaskClicked() }
    }

    // ---------------- 构建 ----------------

    private fun buildCard() {
        setBackgroundColor(Color.parseColor("#9E050814")) // rgba(5,8,20,.62)
        val availH = screenH - insetTop - insetBottom - context.dp(94f)
        var cardW = min(context.dp(300f), (screenW * 0.84f).toInt())
        if (cardW * 4 / 3 > availH) cardW = availH * 3 / 4
        val cardH = cardW * 4 / 3
        val extra = context.dp(47f)

        val container = FrameLayout(context).apply { clipChildren = false; clipToPadding = false }
        addView(container, LayoutParams(cardW + context.dp(14f), cardH + extra * 2, Gravity.CENTER))

        val card = FrameLayout(context).apply {
            isClickable = true // 卡片自身吞点击，不当作遮罩点击
            clipToOutline = true
            outlineProvider = object : ViewOutlineProvider() {
                override fun getOutline(v: View, o: Outline) { o.setRoundRect(0, 0, v.width, v.height, context.dpf(16f)) }
            }
            elevation = context.dpf(12f)
        }
        container.addView(card, LayoutParams(cardW, cardH).apply { gravity = Gravity.CENTER_HORIZONTAL; topMargin = extra })

        val p = SwipePager(context)
        p.setPages(popup.cards.mapIndexed { i, c -> buildPage(i, c, ctaSp = 14f, ctaPadH = 28f, ctaBottomDp = 14f, extraBottomPx = 0) })
        p.swipeEnabled = popup.cards.size > 1
        p.onPageSelected = { from, to -> onPageSelected(from, to) }
        card.addView(p, LayoutParams(matchParent(), matchParent()))
        pager = p

        val d = DotsView(context).apply { setCount(popup.cards.size) }
        container.addView(d, LayoutParams(LayoutParams.WRAP_CONTENT, context.dp(6f)).apply {
            gravity = Gravity.CENTER_HORIZONTAL or Gravity.TOP
            topMargin = extra + cardH + context.dp(12f)
        })
        dots = d

        if (!popup.forced) {
            val x = CloseButtonView(context, 72)
            x.setOnClickListener { onCloseButtonClicked() }
            // 视觉圆右缘 = 卡片右缘；视觉圆底边在卡片上方 10dp（44dp 视图的底边在卡片上方 3dp）
            container.addView(x, LayoutParams(context.dp(44f), context.dp(44f), Gravity.TOP or Gravity.END).apply {
                topMargin = extra - context.dp(47f)
            })
            closeBtn = x
        }
        initCloseState()
    }

    private fun buildFullscreen() {
        setBackgroundColor(Color.BLACK)
        val p = SwipePager(context)
        p.setPages(popup.cards.mapIndexed { i, c ->
            buildPage(i, c, ctaSp = 13f, ctaPadH = 30f, ctaBottomDp = 79f, extraBottomPx = insetBottom)
        })
        p.swipeEnabled = popup.cards.size > 1
        p.onPageSelected = { from, to -> onPageSelected(from, to) }
        addView(p, LayoutParams(matchParent(), matchParent()))
        pager = p

        val d = DotsView(context).apply { setCount(popup.cards.size) }
        addView(d, LayoutParams(LayoutParams.WRAP_CONTENT, context.dp(6f), Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL).apply {
            bottomMargin = insetBottom + context.dp(46f)
        })
        dots = d

        val x = CloseButtonView(context, 90)
        x.setOnClickListener { onCloseButtonClicked() }
        addView(x, LayoutParams(context.dp(44f), context.dp(44f), Gravity.TOP or Gravity.END).apply {
            topMargin = insetTop + context.dp(5f)
            marginEnd = context.dp(5f)
        })
        closeBtn = x
        initCloseState()
    }

    private fun buildPage(index: Int, card: PopupCard, ctaSp: Float, ctaPadH: Float, ctaBottomDp: Float, extraBottomPx: Int): View {
        val clickable = LinkResolver.isClickable(popup.openMode, card.linkUrl)
        val page = FrameLayout(context)
        val bmp = bitmaps[card.imageUrl]
        if (card.imageUrl.isNotBlank()) {
            page.setBackgroundColor(Color.parseColor("#1E1B4B"))
            page.addView(ImageView(context).apply {
                scaleType = ImageView.ScaleType.CENTER_CROP
                setImageBitmap(bmp)
            }, LayoutParams(matchParent(), matchParent()))
        } else {
            // 无图（P4）：title / description 文字卡片
            page.background = GradientDrawable(
                GradientDrawable.Orientation.TL_BR,
                intArrayOf(Color.parseColor("#1E1B4B"), Color.parseColor("#312E81"))
            )
            val col = LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                gravity = Gravity.CENTER
                setPadding(context.dp(24f), context.dp(24f), context.dp(24f), context.dp(72f))
            }
            if (card.title.isNotBlank()) col.addView(TextView(context).apply {
                text = card.title; setTextColor(Color.WHITE); typeface = Typeface.DEFAULT_BOLD
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f); gravity = Gravity.CENTER
            })
            if (card.description.isNotBlank()) col.addView(TextView(context).apply {
                text = card.description; setTextColor(Color.argb(210, 255, 255, 255))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f); gravity = Gravity.CENTER
                setPadding(0, context.dp(12f), 0, 0)
            })
            page.addView(col, LayoutParams(matchParent(), matchParent()))
        }
        if (clickable) {
            if (card.buttonText.isNotBlank()) {
                val btn = ctaButton(context, card.buttonText, ctaSp, ctaPadH)
                btn.setOnClickListener { listener.onCta(index) }
                page.addView(btn, LayoutParams(LayoutParams.WRAP_CONTENT, LayoutParams.WRAP_CONTENT, Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL).apply {
                    bottomMargin = context.dp(ctaBottomDp) + extraBottomPx
                    marginStart = context.dp(16f); marginEnd = context.dp(16f)
                })
            } else {
                page.setOnClickListener { listener.onCta(index) } // 无按钮 = 整图可点
            }
        }
        return page
    }

    private fun initCloseState() {
        if (countdownActive) {
            closeBtn?.showCountdown(ceil((COUNTDOWN_MS - cdElapsedMs) / 1000.0).toInt().coerceIn(1, 3), cdElapsedMs / COUNTDOWN_MS.toFloat())
        } else closeBtn?.showNormal()
        if (initialPage in 1 until popup.cards.size) {
            pager?.setInitialPage(initialPage)
            dots?.setSelected(initialPage)
        }
    }

    /** 当前所在卡片页 / 已走过的倒计时毫秒（配置变更重建时用来恢复）。 */
    fun currentPage(): Int = pager?.currentPage ?: 0
    fun countdownElapsedMs(): Long = if (countdownActive) cdElapsedMs else COUNTDOWN_MS

    // ---------------- 交互 ----------------

    private fun onCloseButtonClicked() {
        if (countdownActive) return // 倒计时中按钮无效（返回键仍可关，由 Manager 直接 close）
        listener.onCloseRequested("button")
    }

    private fun onMaskClicked() {
        if (fullscreen || popup.forced) return
        if (countdownActive) return // 倒计时期间遮罩点击无效
        if (popup.maskClosable) listener.onCloseRequested("mask")
    }

    private fun onPageSelected(from: Int, to: Int) {
        dots?.setSelected(to)
        listener.onSlide(from, to)
        restartDwell(to)
    }

    private fun restartDwell(index: Int) {
        handler.removeCallbacks(dwell)
        dwellIndex = index
        if (resumed && isAttachedToWindow) handler.postDelayed(dwell, DWELL_MS)
    }

    // ---------------- 生命周期 ----------------

    override fun onAttachedToWindow() {
        super.onAttachedToWindow()
        if (resumed) { startCountdownIfNeeded(); restartDwell(pager?.currentPage ?: 0) }
    }

    override fun onDetachedFromWindow() {
        handler.removeCallbacksAndMessages(null)
        cdRunning = false
        super.onDetachedFromWindow()
    }

    /** Activity.onPause：暂停倒计时与停留计时。 */
    override fun onHostPause() {
        resumed = false
        cdRunning = false
        handler.removeCallbacks(tick)
        handler.removeCallbacks(dwell)
    }

    /** Activity.onResume：续计倒计时；停留计时重新计 1 秒。 */
    override fun onHostResume() {
        resumed = true
        startCountdownIfNeeded()
        restartDwell(pager?.currentPage ?: max0(dwellIndex))
    }

    private fun max0(i: Int) = if (i < 0) 0 else i

    private fun startCountdownIfNeeded() {
        if (!countdownActive || cdRunning) return
        cdRunning = true
        cdLastTick = SystemClock.uptimeMillis()
        handler.post(tick)
    }

    companion object {
        const val COUNTDOWN_MS = 3000L
        const val DWELL_MS = 1000L
    }
}
