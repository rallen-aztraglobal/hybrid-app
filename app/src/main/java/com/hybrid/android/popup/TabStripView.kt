package com.hybrid.android.popup

import android.animation.ValueAnimator
import android.content.Context
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.Outline
import android.graphics.Paint
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Handler
import android.os.Looper
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewOutlineProvider
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import kotlin.math.max
import kotlin.math.min

/**
 * 便条层（P8）：贴右边缘，距底部约 25% 屏高（加底部安全区），最多 2 条上下堆叠（下旧上新）。
 * 展开态 = 图标 20dp + 文案（宽 90–110dp、半透明）；静止 3 秒收缩为仅图标（约 32dp、更透明）。
 * 本层自身不拦截触摸，只有便条本体可点。
 */
class TabStripView(
    context: Context,
    private val screenH: Int,
    private val listener: Listener,
) : FrameLayout(context) {

    interface Listener {
        fun onTabClick(key: Long)
        fun onTabLongPress(key: Long)
        /** 便条首次真实渲染到屏幕上。 */
        fun onTabShown(key: Long)
    }

    data class Item(val key: Long, val text: String, val icon: Bitmap?)

    var insetBottom: Int = 0
        set(v) { field = v; relayout() }

    private val handler = Handler(Looper.getMainLooper())
    private val views = LinkedHashMap<Long, TabView>()
    private var mini = false
    private val shrink = Runnable { setMini(true) }

    fun setItems(items: List<Item>) {
        val keys = items.map { it.key }.toSet()
        views.keys.filter { it !in keys }.forEach { removeView(views.remove(it)) }
        items.forEach { item ->
            if (item.key !in views) {
                val tv = TabView(context, item)
                tv.setOnClickListener { listener.onTabClick(item.key) }
                tv.setOnLongClickListener { listener.onTabLongPress(item.key); true }
                views[item.key] = tv
                addView(tv, LayoutParams(tv.expandedWidth, context.dp(28f), Gravity.BOTTOM or Gravity.END))
                tv.applyMini(mini, animate = false)
                tv.post { listener.onTabShown(item.key) }
            }
        }
        // 重排顺序（保持 items 顺序，下旧上新）
        relayout(items.map { it.key })
        notifyActivity()
    }

    /** 弹窗展示期间隐藏对应便条（关闭后恢复）。 */
    fun setHidden(key: Long, hidden: Boolean) { views[key]?.visibility = if (hidden) INVISIBLE else VISIBLE }

    private var order: List<Long> = emptyList()

    private fun relayout(newOrder: List<Long>? = null) {
        if (newOrder != null) order = newOrder
        order.forEachIndexed { i, key ->
            val v = views[key] ?: return@forEachIndexed
            (v.layoutParams as LayoutParams).bottomMargin =
                insetBottom + (screenH * 0.25f).toInt() + i * context.dp(34f)
            v.requestLayout()
        }
    }

    /** 用户触摸页面 / WebView 滚动：恢复展开并重新计时 3 秒后收缩。 */
    fun notifyActivity() {
        handler.removeCallbacks(shrink)
        if (mini) setMini(false)
        if (views.isNotEmpty()) handler.postDelayed(shrink, MINI_DELAY_MS)
    }

    private fun setMini(m: Boolean) {
        mini = m
        views.values.forEach { it.applyMini(m, animate = true) }
    }

    companion object { const val MINI_DELAY_MS = 3_000L }

    fun onHostPause() { handler.removeCallbacks(shrink) }
    fun onHostResume() = notifyActivity()

    override fun onDetachedFromWindow() { handler.removeCallbacksAndMessages(null); super.onDetachedFromWindow() }

    private class TabView(context: Context, item: Item) : LinearLayout(context) {
        val expandedWidth: Int
        private val label: TextView
        private var anim: ValueAnimator? = null
        private val miniWidth = context.dp(32f)

        init {
            orientation = HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            val r = context.dpf(14f)
            background = GradientDrawable().apply {
                setColor(Color.argb(184, 15, 23, 42)) // rgba(15,23,42,.72)
                cornerRadii = floatArrayOf(r, r, 0f, 0f, 0f, 0f, r, r) // 左圆右直
            }
            elevation = context.dpf(3f)
            setPadding(context.dp(4f), 0, context.dp(8f), 0)
            clipToOutline = true
            outlineProvider = object : ViewOutlineProvider() {
                override fun getOutline(v: View, o: Outline) { o.setRoundRect(0, 0, v.width + (r * 2).toInt(), v.height, r) }
            }
            addView(ImageView(context).apply {
                scaleType = ImageView.ScaleType.CENTER_CROP
                setImageBitmap(item.icon)
                background = roundRect(Color.argb(60, 255, 255, 255), context.dpf(6f))
                outlineProvider = object : ViewOutlineProvider() {
                    override fun getOutline(v: View, o: Outline) { o.setRoundRect(0, 0, v.width, v.height, context.dpf(6f)) }
                }
                clipToOutline = true
            }, LayoutParams(context.dp(20f), context.dp(20f)))
            label = TextView(context).apply {
                text = item.text; setTextColor(Color.WHITE); typeface = Typeface.DEFAULT_BOLD
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f); maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
                setPadding(context.dp(5f), 0, 0, 0)
            }
            addView(label, LayoutParams(0, LayoutParams.WRAP_CONTENT, 1f))
            // 宽度：图标 + 文案自然宽，夹在 90–110dp
            val paint = Paint().apply { textSize = context.dpf(12f) * (context.resources.configuration.fontScale); typeface = Typeface.DEFAULT_BOLD }
            val natural = context.dp(4f + 20f + 5f + 8f) + paint.measureText(item.text).toInt()
            expandedWidth = min(context.dp(110f), max(context.dp(90f), natural))
            alpha = 0.94f
        }

        fun applyMini(mini: Boolean, animate: Boolean) {
            val toW = if (mini) miniWidth else expandedWidth
            val toA = if (mini) 0.55f else 0.94f
            anim?.cancel()
            val lp = layoutParams
            if (!animate || lp == null) {
                lp?.width = toW; alpha = toA; label.alpha = if (mini) 0f else 1f
                layoutParams = lp
                return
            }
            val fromW = lp.width
            val fromA = alpha
            val fromL = label.alpha
            anim = ValueAnimator.ofFloat(0f, 1f).apply {
                duration = 350
                addUpdateListener {
                    val f = it.animatedValue as Float
                    lp.width = (fromW + (toW - fromW) * f).toInt()
                    alpha = fromA + (toA - fromA) * f
                    label.alpha = fromL + ((if (mini) 0f else 1f) - fromL) * f
                    layoutParams = lp
                }
                start()
            }
        }
    }
}
