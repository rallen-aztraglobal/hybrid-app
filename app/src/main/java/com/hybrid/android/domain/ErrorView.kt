package com.hybrid.android.domain

import android.content.Context
import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.util.TypedValue
import android.view.Gravity
import android.view.ViewGroup
import android.widget.Button
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView

/** 错误页类型，对应 STEP3 的两种归因（A 本机网络 / B 域名服务）。 */
enum class ErrorKind {
    /** A：本机网络问题。文案「网络异常，请检查你的网络连接」。 */
    NO_NETWORK,

    /** B：域名/服务问题，备用已穷尽。文案「服务暂时不可用，请稍后重试」。 */
    SERVICE_DOWN,
}

/**
 * 原生错误页（非网页——因为页面根本加载不出来）：图标 + 文案 + 刷新按钮。
 * 纯代码构建，无需 XML，风格与 [com.hybrid.android.WebViewActivity] 一致。
 *
 * 自动重试由 Activity 注册网络恢复广播后调用 [onRetry] 触发（见 WebViewActivity）。
 */
class ErrorView(context: Context) : FrameLayout(context) {

    private val iconView: TextView
    private val titleView: TextView
    private val messageView: TextView
    private val refreshButton: Button

    /** 刷新点击回调（Activity 注入 → 重走 resolve）。 */
    var onRefresh: (() -> Unit)? = null

    init {
        setBackgroundColor(Color.WHITE)
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT
        )
        isClickable = true // 吃掉点击，避免穿透到下层 WebView

        val column = LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER
            layoutParams = LayoutParams(
                LayoutParams.MATCH_PARENT,
                LayoutParams.MATCH_PARENT
            ).apply { gravity = Gravity.CENTER }
            setPadding(dp(32), dp(32), dp(32), dp(32))
        }

        // 图标：用 emoji 文本占位（避免新增二进制资源，渠道资源目录受 sourceSets 约束）。
        iconView = TextView(context).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 56f)
            gravity = Gravity.CENTER
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
            )
        }

        titleView = TextView(context).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 18f)
            setTextColor(Color.parseColor("#222222"))
            gravity = Gravity.CENTER
            setPadding(0, dp(20), 0, dp(8))
        }

        messageView = TextView(context).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            setTextColor(Color.parseColor("#888888"))
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, dp(28))
        }

        refreshButton = Button(context).apply {
            text = "Refresh"
            isAllCaps = false
            setTextColor(Color.WHITE)
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
            background = GradientDrawable().apply {
                cornerRadius = dp(24).toFloat()
                setColor(Color.parseColor("#2E6BE6"))
            }
            val lp = LinearLayout.LayoutParams(dp(160), dp(46))
            layoutParams = lp
            setOnClickListener { onRefresh?.invoke() }
        }

        column.addView(iconView)
        column.addView(titleView)
        column.addView(messageView)
        column.addView(refreshButton)
        addView(column)
    }

    /** 按归因绑定文案。 */
    fun bind(kind: ErrorKind) {
        when (kind) {
            ErrorKind.NO_NETWORK -> {
                iconView.text = "📶" // 📶
                titleView.text = "No Internet Connection"
                messageView.text = "Please check your network connection and try again."
            }
            ErrorKind.SERVICE_DOWN -> {
                iconView.text = "🛠" // 🛠
                titleView.text = "Service Temporarily Unavailable"
                messageView.text = "We're working on it. Please try again later."
            }
        }
    }

    private fun dp(value: Int): Int =
        (value * resources.displayMetrics.density).toInt()
}
