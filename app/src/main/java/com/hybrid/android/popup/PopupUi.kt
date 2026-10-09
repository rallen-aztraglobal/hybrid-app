package com.hybrid.android.popup

import android.animation.ValueAnimator
import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.RectF
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.view.MotionEvent
import android.view.VelocityTracker
import android.view.View
import android.view.ViewConfiguration
import android.view.ViewGroup
import android.view.animation.DecelerateInterpolator
import android.widget.LinearLayout
import android.widget.TextView
import kotlin.math.abs
import kotlin.math.max

internal fun Context.dp(v: Float): Int = (v * resources.displayMetrics.density + 0.5f).toInt()
internal fun Context.dpf(v: Float): Float = v * resources.displayMetrics.density

internal fun roundRect(color: Int, radiusPx: Float, strokePx: Int = 0, strokeColor: Int = 0) =
    GradientDrawable().apply {
        shape = GradientDrawable.RECTANGLE
        cornerRadius = radiusPx
        setColor(color)
        if (strokePx > 0) setStroke(strokePx, strokeColor)
    }

/** CTA 胶囊按钮（原型 .pw-cta：#ffe28a→#f5b800，文字 #5b3400 粗体）。 */
internal fun ctaButton(context: Context, text: String, textSp: Float, padHDp: Float): TextView =
    TextView(context).apply {
        this.text = text
        setTextColor(Color.parseColor("#5B3400"))
        setTextSize(android.util.TypedValue.COMPLEX_UNIT_SP, textSp)
        typeface = Typeface.DEFAULT_BOLD
        maxLines = 1
        ellipsize = android.text.TextUtils.TruncateAt.END
        gravity = android.view.Gravity.CENTER
        setPadding(context.dp(padHDp), context.dp(10f), context.dp(padHDp), context.dp(10f))
        background = GradientDrawable(
            GradientDrawable.Orientation.TOP_BOTTOM,
            intArrayOf(Color.parseColor("#FFE28A"), Color.parseColor("#F5B800"))
        ).apply { cornerRadius = context.dpf(30f) }
        elevation = context.dpf(4f)
    }

/**
 * 关闭按钮：视觉是「小圆圈 + 细 X」（圆 30dp、1.5dp 白描边、半透明深底、X 约 15dp 盒），
 * 但 View 本身固定 44×44dp，触摸热区因此 ≥ 44dp（不放大视觉）。
 * 倒计时态：圆缩到 0.8、描边变淡，显示数字 + 进度圆环，且点击无效（由调用方判定）。
 */
class CloseButtonView(context: Context, private val bgAlpha: Int = 72) : View(context) {
    private val fill = Paint(Paint.ANTI_ALIAS_FLAG).apply { style = Paint.Style.FILL }
    private val stroke = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = context.dpf(1.5f); color = Color.WHITE
    }
    private val xPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = context.dpf(1.5f); color = Color.WHITE; strokeCap = Paint.Cap.ROUND
    }
    private val ring = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE; strokeWidth = context.dpf(2.4f); color = Color.WHITE; strokeCap = Paint.Cap.ROUND
    }
    private val text = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = Color.WHITE; typeface = Typeface.DEFAULT_BOLD; textSize = context.dpf(13f); textAlign = Paint.Align.CENTER
    }
    private val oval = RectF()

    private var counting = false
    private var label = ""
    private var progress = 0f // 0..1，已走完的比例

    init { contentDescription = "Close" }

    fun showCountdown(seconds: Int, progress: Float) {
        counting = true; label = seconds.toString(); this.progress = progress.coerceIn(0f, 1f)
        invalidate()
    }

    fun showNormal() { counting = false; invalidate() }

    override fun onMeasure(w: Int, h: Int) {
        val s = context.dp(44f)
        setMeasuredDimension(s, s)
    }

    override fun onDraw(c: Canvas) {
        val cx = width / 2f
        val cy = height / 2f
        val r = context.dpf(15f) * (if (counting) 0.8f else 1f)
        fill.color = Color.argb(bgAlpha, 0, 0, 0)
        c.drawCircle(cx, cy, r, fill)
        stroke.alpha = if (counting) 64 else 217
        c.drawCircle(cx, cy, r - stroke.strokeWidth / 2f, stroke)
        if (counting) {
            val rr = r + context.dpf(2f)
            oval.set(cx - rr, cy - rr, cx + rr, cy + rr)
            c.drawArc(oval, -90f, 360f * (1f - progress), false, ring)
            val fm = text.fontMetrics
            c.drawText(label, cx, cy - (fm.ascent + fm.descent) / 2f, text)
        } else {
            val h = context.dpf(5.5f)
            c.drawLine(cx - h, cy - h, cx + h, cy + h, xPaint)
            c.drawLine(cx - h, cy + h, cx + h, cy - h, xPaint)
        }
    }
}

/** 轮播圆点（原型 .dots：当前页 16×6 白色长条，其余 6×6 40% 白）。 */
class DotsView(context: Context) : LinearLayout(context) {
    private var count = 0

    init { orientation = HORIZONTAL; gravity = android.view.Gravity.CENTER }

    fun setCount(n: Int) {
        count = n
        removeAllViews()
        repeat(n) {
            addView(View(context), LayoutParams(context.dp(6f), context.dp(6f)).apply {
                marginStart = context.dp(2.5f); marginEnd = context.dp(2.5f)
            })
        }
        setSelected(0)
        visibility = if (n > 1) VISIBLE else GONE
    }

    fun setSelected(index: Int) {
        for (i in 0 until childCount) {
            val v = getChildAt(i)
            val on = i == index
            v.layoutParams = (v.layoutParams as LayoutParams).apply { width = context.dp(if (on) 16f else 6f) }
            v.background = roundRect(if (on) Color.WHITE else Color.argb(102, 255, 255, 255), context.dpf(3f))
        }
        requestLayout()
    }
}

/**
 * 极简横向翻页容器（不引入 ViewPager2 / RecyclerView）：最多 5 页，scrollX 驱动，松手按位移/速度吸附。
 * 单页时禁用滑动。
 */
class SwipePager(context: Context) : ViewGroup(context) {
    var onPageSelected: ((from: Int, to: Int) -> Unit)? = null
    var swipeEnabled = true
    var currentPage = 0
        private set

    private val slop = ViewConfiguration.get(context).scaledTouchSlop
    private val minFling = ViewConfiguration.get(context).scaledMinimumFlingVelocity
    private var downX = 0f
    private var downY = 0f
    private var startScroll = 0
    private var dragging = false
    private var tracker: VelocityTracker? = null
    private var anim: ValueAnimator? = null

    val pageCount: Int get() = childCount

    fun setPages(pages: List<View>) {
        removeAllViews()
        pages.forEach { addView(it, LayoutParams(LayoutParams.MATCH_PARENT, LayoutParams.MATCH_PARENT)) }
        currentPage = 0
        scrollTo(0, 0)
    }

    /** 恢复用：直接指定当前页（不触发回调，布局时 scrollTo 到位）。 */
    fun setInitialPage(i: Int) { currentPage = i.coerceIn(0, max(0, childCount - 1)) }

    override fun onMeasure(wSpec: Int, hSpec: Int) {
        val w = MeasureSpec.getSize(wSpec)
        val h = MeasureSpec.getSize(hSpec)
        for (i in 0 until childCount) {
            getChildAt(i).measure(
                MeasureSpec.makeMeasureSpec(w, MeasureSpec.EXACTLY),
                MeasureSpec.makeMeasureSpec(h, MeasureSpec.EXACTLY)
            )
        }
        setMeasuredDimension(w, h)
    }

    override fun onLayout(changed: Boolean, l: Int, t: Int, r: Int, b: Int) {
        val w = r - l
        for (i in 0 until childCount) getChildAt(i).layout(i * w, 0, (i + 1) * w, b - t)
        scrollTo(currentPage * w, 0)
    }

    private fun canSwipe() = swipeEnabled && childCount > 1

    override fun onInterceptTouchEvent(ev: MotionEvent): Boolean {
        if (!canSwipe()) return false
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> { downX = ev.x; downY = ev.y; startScroll = scrollX; dragging = false; anim?.cancel() }
            MotionEvent.ACTION_MOVE -> {
                val dx = ev.x - downX
                if (abs(dx) > slop && abs(dx) > abs(ev.y - downY)) {
                    dragging = true
                    parent?.requestDisallowInterceptTouchEvent(true)
                    return true
                }
            }
        }
        return false
    }

    override fun onTouchEvent(ev: MotionEvent): Boolean {
        if (!canSwipe()) return false
        if (tracker == null) tracker = VelocityTracker.obtain()
        tracker?.addMovement(ev)
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> { downX = ev.x; downY = ev.y; startScroll = scrollX; anim?.cancel(); return true }
            MotionEvent.ACTION_MOVE -> {
                val maxScroll = (childCount - 1) * width
                scrollTo((startScroll - (ev.x - downX)).toInt().coerceIn(0, max(0, maxScroll)), 0)
            }
            MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL -> {
                tracker?.computeCurrentVelocity(1000)
                val vx = tracker?.xVelocity ?: 0f
                tracker?.recycle(); tracker = null
                val w = max(1, width)
                var target = currentPage
                val delta = scrollX - currentPage * w
                if (abs(vx) > minFling * 4 && abs(delta) > w * 0.05f) target += if (vx < 0) 1 else -1
                else if (abs(delta) > w * 0.25f) target += if (delta > 0) 1 else -1
                goTo(target.coerceIn(0, childCount - 1), true)
            }
        }
        return true
    }

    fun goTo(page: Int, animate: Boolean) {
        val from = currentPage
        currentPage = page
        val to = page * max(1, width)
        anim?.cancel()
        if (animate && width > 0) {
            anim = ValueAnimator.ofInt(scrollX, to).apply {
                duration = 250; interpolator = DecelerateInterpolator()
                addUpdateListener { scrollTo(it.animatedValue as Int, 0) }
                start()
            }
        } else scrollTo(to, 0)
        if (from != page) onPageSelected?.invoke(from, page)
    }

    override fun onDetachedFromWindow() { anim?.cancel(); tracker?.recycle(); tracker = null; super.onDetachedFromWindow() }
}


internal fun matchParent() = ViewGroup.LayoutParams.MATCH_PARENT
