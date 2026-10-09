package com.hybrid.android.popup

import android.content.Context
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.Outline
import android.graphics.Typeface
import android.os.Handler
import android.os.Looper
import android.util.TypedValue
import android.view.Gravity
import android.view.MotionEvent
import android.view.View
import android.view.ViewConfiguration
import android.view.ViewOutlineProvider
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.TextView
import kotlin.math.abs

/** 非遮罩类视图的公共回调。 */
interface NonOverlayListener {
    fun onCardDwell(index: Int)
    fun onCta(index: Int)
    fun onCloseRequested()
}

/** 非遮罩视图通用的生命周期接口（Activity pause/resume）。 */
interface HostLifecycleAware {
    fun onHostPause()
    fun onHostResume()
}

private fun circleOutline() = object : ViewOutlineProvider() {
    override fun getOutline(v: View, o: Outline) { o.setOval(0, 0, v.width, v.height) }
}

private fun roundOutline(radiusPx: Float) = object : ViewOutlineProvider() {
    override fun getOutline(v: View, o: Outline) { o.setRoundRect(0, 0, v.width, v.height, radiusPx) }
}

/** 停留 ≥1s 的统一计时器。 */
internal class DwellTimer(private val onDwell: (Int) -> Unit) {
    private val handler = Handler(Looper.getMainLooper())
    private var index = -1
    private val r = Runnable { if (index >= 0) onDwell(index) }
    fun restart(i: Int) { handler.removeCallbacks(r); index = i; handler.postDelayed(r, OverlayPopupView.DWELL_MS) }
    fun cancel() { handler.removeCallbacks(r) }
    fun resume() { if (index >= 0) restart(index) }
}

/**
 * P2 悬浮球：60dp 圆形、可拖动、松手吸左右边、红点角标、左上小关闭 x（热区 ≥ 44dp）。
 */
class FloatBallView(
    context: Context,
    private val popup: Popup,
    bitmap: Bitmap?,
    private val insetTop: Int,
    private val insetBottom: Int,
    private val listener: NonOverlayListener,
) : FrameLayout(context), HostLifecycleAware {

    private val size = context.dp(60f)
    /** 球外围各留 14dp 透明边，给小 x 的 44dp 热区与角标留空间。 */
    val frameSize = size + context.dp(28f)
    private val dwell = DwellTimer { listener.onCardDwell(it) }
    private val slop = ViewConfiguration.get(context).scaledTouchSlop
    private var downRawX = 0f
    private var downRawY = 0f
    private var startX = 0f
    private var startY = 0f
    private var moved = false

    init {
        clipChildren = false
        val ball = ImageView(context).apply {
            scaleType = ImageView.ScaleType.CENTER_CROP
            setImageBitmap(bitmap)
            setBackgroundColor(Color.parseColor("#312E81"))
            outlineProvider = circleOutline(); clipToOutline = true
            elevation = context.dpf(6f)
        }
        // 白描边：用外层圆形背景 + 内缩 2dp
        val ring = FrameLayout(context).apply {
            background = roundRect(Color.WHITE, context.dpf(30f))
            setPadding(context.dp(2f), context.dp(2f), context.dp(2f), context.dp(2f))
            elevation = context.dpf(6f)
        }
        ring.addView(ball, LayoutParams(matchParent(), matchParent()))
        addView(ring, LayoutParams(size, size, Gravity.CENTER))

        if (popup.badge) {
            addView(View(context).apply {
                background = roundRect(Color.parseColor("#EF4444"), context.dpf(8f), context.dp(1.5f), Color.WHITE)
                elevation = context.dpf(8f)
            }, LayoutParams(context.dp(14f), context.dp(14f), Gravity.TOP or Gravity.END).apply {
                topMargin = context.dp(12f); marginEnd = context.dp(12f)
            })
        }
        // 小关闭 x：视觉 16dp，热区 44dp
        val x = FrameLayout(context).apply { elevation = context.dpf(8f) }
        x.addView(TextView(context).apply {
            text = "×"; setTextColor(Color.WHITE); gravity = Gravity.CENTER
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f); typeface = Typeface.DEFAULT_BOLD
            background = roundRect(Color.argb(191, 15, 23, 42), context.dpf(8f))
        }, LayoutParams(context.dp(16f), context.dp(16f), Gravity.CENTER))
        x.setOnClickListener { listener.onCloseRequested() }
        x.contentDescription = "Close"
        addView(x, LayoutParams(context.dp(44f), context.dp(44f), Gravity.TOP or Gravity.START).apply {
            // 热区贴在球左上角外围（视觉 16dp 的 x 落在其中心）
        })
    }

    override fun onTouchEvent(ev: MotionEvent): Boolean {
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downRawX = ev.rawX; downRawY = ev.rawY; startX = x; startY = y; moved = false
                parent?.requestDisallowInterceptTouchEvent(true)
            }
            MotionEvent.ACTION_MOVE -> {
                val dx = ev.rawX - downRawX
                val dy = ev.rawY - downRawY
                if (!moved && (abs(dx) > slop || abs(dy) > slop)) moved = true
                if (moved) { x = startX + dx; y = startY + dy }
            }
            MotionEvent.ACTION_UP -> {
                if (moved) snapToEdge() else { performClick(); listener.onCta(0) }
            }
        }
        return true
    }

    override fun performClick(): Boolean = super.performClick()

    private fun snapToEdge() {
        val p = parent as? View ?: return
        val margin = context.dpf(6f) - context.dpf(14f) // 球外围留了 14dp 透明边
        val toLeft = x + width / 2f < p.width / 2f
        val targetX = if (toLeft) margin else p.width - width - margin
        val targetY = y.coerceIn(insetTop.toFloat(), (p.height - insetBottom - height).toFloat().coerceAtLeast(insetTop.toFloat()))
        animate().x(targetX).y(targetY).setDuration(200).start()
    }

    /** 初始位置：右侧吸边，距顶约 56% 屏高。 */
    fun placeInitially(parentW: Int, parentH: Int) {
        x = parentW - frameSize + context.dpf(8f) // 球视觉右缘距屏幕边 6dp
        y = (parentH * 0.56f).coerceAtMost((parentH - insetBottom - frameSize).toFloat())
    }

    override fun onAttachedToWindow() { super.onAttachedToWindow(); dwell.restart(0) }
    override fun onDetachedFromWindow() { dwell.cancel(); super.onDetachedFromWindow() }
    override fun onHostPause() = dwell.cancel()
    override fun onHostResume() = dwell.resume()
}

/**
 * P3 底部横幅：左右留 10dp、高 64dp、白底圆角 16dp；缩略图 40dp（可选）+ 一行文案 + 关闭；
 * 多卡按 autoplaySeconds 自动轮播（淡入淡出），底部小圆点。
 */
class BannerView(
    context: Context,
    private val popup: Popup,
    private val bitmaps: Map<String, Bitmap?>,
    private val listener: NonOverlayListener,
) : FrameLayout(context), HostLifecycleAware {

    private val handler = Handler(Looper.getMainLooper())
    private val items = ArrayList<View>()
    private val dots = ArrayList<View>()
    private var current = 0
    private var resumed = true
    private val dwell = DwellTimer { listener.onCardDwell(it) }
    private val autoplay = object : Runnable {
        override fun run() {
            if (items.size > 1 && resumed) show((current + 1) % items.size)
            handler.postDelayed(this, popup.autoplaySeconds * 1000L)
        }
    }

    init {
        background = roundRect(Color.argb(247, 255, 255, 255), context.dpf(16f))
        elevation = context.dpf(8f)
        popup.cards.forEachIndexed { i, c -> buildItem(i, c) }
        items.forEachIndexed { i, v -> v.alpha = if (i == 0) 1f else 0f; v.visibility = if (i == 0) VISIBLE else GONE }
        if (items.size > 1) buildDots()

        val x = CloseButtonLite(context)
        x.setOnClickListener { listener.onCloseRequested() }
        addView(x, LayoutParams(context.dp(44f), context.dp(44f), Gravity.END or Gravity.CENTER_VERTICAL))
    }

    private fun buildItem(i: Int, c: PopupCard) {
        val row = FrameLayout(context)
        val clickable = LinkResolver.isClickable(popup.openMode, c.linkUrl)
        var textStart = context.dp(14f)
        val bmp = bitmaps[c.imageUrl]
        if (c.imageUrl.isNotBlank()) {
            row.addView(ImageView(context).apply {
                scaleType = ImageView.ScaleType.CENTER_CROP; setImageBitmap(bmp)
                setBackgroundColor(Color.parseColor("#E2E8F0"))
                outlineProvider = roundOutline(context.dpf(9f)); clipToOutline = true
            }, LayoutParams(context.dp(40f), context.dp(40f), Gravity.START or Gravity.CENTER_VERTICAL).apply {
                marginStart = context.dp(12f)
            })
            textStart = context.dp(12f + 40f + 10f)
        }
        row.addView(TextView(context).apply {
            text = c.title; setTextColor(Color.parseColor("#0F172A")); typeface = Typeface.DEFAULT_BOLD
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f); maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
        }, LayoutParams(matchParent(), LayoutParams.WRAP_CONTENT, Gravity.CENTER_VERTICAL).apply {
            marginStart = textStart; marginEnd = context.dp(48f)
        })
        if (clickable) row.setOnClickListener { listener.onCta(i) }
        items.add(row)
        addView(row, LayoutParams(matchParent(), matchParent()))
    }

    private fun buildDots() {
        val box = android.widget.LinearLayout(context).apply { gravity = Gravity.CENTER }
        popup.cards.indices.forEach { _ ->
            val d = View(context)
            dots.add(d)
            box.addView(d, android.widget.LinearLayout.LayoutParams(context.dp(4f), context.dp(4f)).apply {
                marginStart = context.dp(1.5f); marginEnd = context.dp(1.5f)
            })
        }
        addView(box, LayoutParams(LayoutParams.WRAP_CONTENT, context.dp(4f), Gravity.BOTTOM or Gravity.CENTER_HORIZONTAL).apply {
            bottomMargin = context.dp(4f)
        })
        refreshDots()
    }

    private fun refreshDots() {
        dots.forEachIndexed { i, d ->
            val on = i == current
            d.layoutParams = (d.layoutParams as android.widget.LinearLayout.LayoutParams).apply { width = context.dp(if (on) 10f else 4f) }
            d.background = roundRect(if (on) Color.parseColor("#6366F1") else Color.parseColor("#CBD5E1"), context.dpf(2f))
        }
    }

    private fun show(i: Int) {
        if (i == current) return
        val old = items[current]
        val nw = items[i]
        nw.visibility = VISIBLE
        nw.animate().alpha(1f).setDuration(350).start()
        old.animate().alpha(0f).setDuration(350).withEndAction { old.visibility = GONE }.start()
        current = i
        refreshDots()
        dwell.restart(i)
    }

    override fun onAttachedToWindow() {
        super.onAttachedToWindow()
        dwell.restart(current)
        if (items.size > 1) handler.postDelayed(autoplay, popup.autoplaySeconds * 1000L)
    }

    override fun onDetachedFromWindow() {
        handler.removeCallbacksAndMessages(null); dwell.cancel(); super.onDetachedFromWindow()
    }

    override fun onHostPause() { resumed = false; dwell.cancel() }
    override fun onHostResume() { resumed = true; dwell.restart(current) }
}

/** P5 顶部通栏：贴状态栏下方、高 44dp、浅黄底纯文字 + 关闭。 */
class TopBarView(
    context: Context,
    private val popup: Popup,
    private val listener: NonOverlayListener,
) : FrameLayout(context), HostLifecycleAware {

    private val dwell = DwellTimer { listener.onCardDwell(it) }

    init {
        setBackgroundColor(Color.parseColor("#FFFBEB"))
        elevation = context.dpf(4f)
        val card = popup.cards.first()
        addView(TextView(context).apply {
            text = card.title.ifBlank { card.description }
            setTextColor(Color.parseColor("#92400E")); typeface = Typeface.create(Typeface.DEFAULT, Typeface.BOLD)
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
            gravity = Gravity.CENTER_VERTICAL
        }, LayoutParams(matchParent(), matchParent()).apply { marginStart = context.dp(12f); marginEnd = context.dp(48f) })
        // 底部 1dp 分隔线
        addView(View(context).apply { setBackgroundColor(Color.parseColor("#FDE68A")) },
            LayoutParams(matchParent(), context.dp(1f), Gravity.BOTTOM))
        if (LinkResolver.isClickable(popup.openMode, card.linkUrl)) setOnClickListener { listener.onCta(0) }
        val x = CloseButtonLite(context, Color.parseColor("#92400E"))
        x.setOnClickListener { listener.onCloseRequested() }
        addView(x, LayoutParams(context.dp(44f), context.dp(44f), Gravity.END or Gravity.CENTER_VERTICAL))
    }

    override fun onAttachedToWindow() { super.onAttachedToWindow(); dwell.restart(0) }
    override fun onDetachedFromWindow() { dwell.cancel(); super.onDetachedFromWindow() }
    override fun onHostPause() = dwell.cancel()
    override fun onHostResume() = dwell.resume()
}

/** 横幅 / 通栏用的轻量关闭 x：44dp 热区，无圆圈，细 X。 */
class CloseButtonLite(context: Context, private val color: Int = Color.parseColor("#94A3B8")) : View(context) {
    private val paint = android.graphics.Paint(android.graphics.Paint.ANTI_ALIAS_FLAG).apply {
        style = android.graphics.Paint.Style.STROKE; strokeWidth = context.dpf(1.6f)
        strokeCap = android.graphics.Paint.Cap.ROUND
    }
    init { contentDescription = "Close" }
    override fun onMeasure(w: Int, h: Int) { val s = context.dp(44f); setMeasuredDimension(s, s) }
    override fun onDraw(c: android.graphics.Canvas) {
        paint.color = color
        val cx = width / 2f; val cy = height / 2f; val h = context.dpf(5f)
        c.drawLine(cx - h, cy - h, cx + h, cy + h, paint)
        c.drawLine(cx - h, cy + h, cx + h, cy - h, paint)
    }
}
