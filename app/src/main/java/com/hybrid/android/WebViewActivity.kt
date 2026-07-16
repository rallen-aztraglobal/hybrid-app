package com.hybrid.android

import android.Manifest
import android.annotation.SuppressLint
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Color
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.util.Base64
import android.util.Log
import android.view.View
import android.view.ViewGroup
import android.webkit.PermissionRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import android.webkit.WebChromeClient
import android.widget.FrameLayout
import android.widget.ImageView
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updateLayoutParams

class WebViewActivity : ComponentActivity() {
    private lateinit var webView: WebView
    private lateinit var splashImageView: ImageView

    private var currentPath: String? = null
    private val palCode = BuildConfig.PAL_CODE

    private var domain = "https://www.bingoplus.com"

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // 创建 FrameLayout 根容器
        val rootLayout = FrameLayout(this).apply {
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
        }

        // 创建全屏 splash 背景图（1920x1080，保持比例不变形）
        splashImageView = ImageView(this).apply {
            setImageResource(R.drawable.splash_fullscreen) // 替换成你的图片资源
            scaleType = ImageView.ScaleType.CENTER_CROP
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
        }

        // 创建 WebView
        webView = WebView(this).apply {
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            setBackgroundColor(Color.WHITE)
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
        }

        // 添加视图到根布局，注意顺序（splashImageView 在上层）
        rootLayout.addView(webView)
        rootLayout.addView(splashImageView)
        setContentView(rootLayout)

        // 设置沉浸式状态栏
        enterImmersiveMode()

        // 处理 window insets（状态栏/导航栏 padding）
        ViewCompat.setOnApplyWindowInsetsListener(webView) { view, insets ->
            val systemInsets = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            view.updateLayoutParams<ViewGroup.MarginLayoutParams> {
                topMargin = systemInsets.top
                bottomMargin = systemInsets.bottom
            }
            insets
        }

        val settings: WebSettings = webView.settings
        settings.javaScriptEnabled = true
        settings.mediaPlaybackRequiresUserGesture = false

        settings.mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW


        // 设置 WebViewClient
        webView.webViewClient = object : WebViewClient() {
            override fun onPageFinished(view: WebView, url: String?) {
                super.onPageFinished(view, url)
                window.decorView.systemUiVisibility =
                    View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                            View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
                // 淡出 splash 背景
                splashImageView.animate()
                    .alpha(0f)
                    .setDuration(1000)
                    .withEndAction {
                        splashImageView.visibility = View.GONE
                    }
                    .start()
                splashImageView.visibility = View.GONE

                val css = """
                    .app-download {
                        display: none !important;
                    }
                """.trimIndent()

                val encodedCss = Base64.encodeToString(css.toByteArray(), Base64.NO_WRAP)
                val js = """
                    (function() {
                        var style = document.createElement('style');
                        style.innerHTML = window.atob('$encodedCss');
                        document.head.appendChild(style);
                    })();
                """.trimIndent()
                view.evaluateJavascript(js, null)
            }

            // 拦截 window.open 的跳转（包括 target="_blank"）
            override fun shouldOverrideUrlLoading(view: WebView, request: android.webkit.WebResourceRequest): Boolean {
                val url = request.url.toString()
                return handleDeeplink(view.getContext(), url);
            }
            // 监控 url变化
            override fun doUpdateVisitedHistory(view: WebView, url: String, isReload: Boolean) {
                super.doUpdateVisitedHistory(view, url, isReload)
                currentPath = view.url
                Log.d("URL", "历史记录 URL 更新为：$currentPath")
                if (view.url?.contains("kyc") ?: false) {
                    checkAndRequestPermission()
                }
            }
        }

        webView.webChromeClient = object : WebChromeClient() {
            override fun onPermissionRequest(request: PermissionRequest?) {
                runOnUiThread {
                    request?.grant(request.resources) // 允许 JS 使用相机/麦克风
                }
            }
        }

        // 加载网页
        webView.loadUrl("${domain}/?palcode=${palCode}")

        // 处理返回键
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (webView.canGoBack()) webView.goBack() else finish()
            }
        })
    }


    private val requestPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { isGranted: Boolean ->
            if (isGranted) {
                // 权限已授予
            } else {
                // 权限被拒绝
            }
        }

    // 调用相机权限申请
    fun checkAndRequestPermission() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            == PackageManager.PERMISSION_GRANTED
        ) {
            // 已有权限
        } else {
            requestPermissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    private fun handleDeeplink(context: Context, url: String?): Boolean {
        val uri = Uri.parse(url)
        val scheme = uri.getScheme()

        // 标准网页直接加载
        if ("http".equals(scheme, ignoreCase = true) || "https".equals(scheme, ignoreCase = true)) {
            if (url?.startsWith(domain) ?: false) {
                // 是自己域名，WebView 内加载

                return false
            } else {
                // 外部网站，用系统浏览器打开
                try {
                    val browserIntent = Intent(Intent.ACTION_VIEW, uri)
                    context.startActivity(browserIntent)
                    // if ("$url".contains("payments") && webView.canGoBack()) webView.goBack()
                } catch (e: java.lang.Exception) {
                    e.printStackTrace()
                }
                if (currentPath?.contains("/confirm") == true) {
                    if (webView.canGoBack()) webView.goBack()
                }
                return true
            }
        } else {
            // 其他自定义 scheme（如 wechat://、alipays:// 等）
            try {
                val intent = Intent(Intent.ACTION_VIEW, uri)
                context.startActivity(intent)
            } catch (e: ActivityNotFoundException) {
                // 如果未安装对应 App，可提示或忽略
                Log.w("WebView", "App 未安装: " + url)
            }
        }

        // intent:// 类型处理
        if ("intent".equals(scheme, ignoreCase = true)) {
            try {
                val intent = Intent.parseUri(url, Intent.URI_INTENT_SCHEME)
                // 检查是否有对应的 App
                if (intent.getPackage() != null) {
                    if (isAppInstalled(context, intent.getPackage()!!)) {
                        context.startActivity(intent)
                    } else {
                        // 跳转应用市场
                        val marketIntent = Intent(Intent.ACTION_VIEW)
                        marketIntent.setData(Uri.parse("market://details?id=" + intent.getPackage()))
                        context.startActivity(marketIntent)
                    }
                    return true
                }
            } catch (e: Exception) {
                e.printStackTrace()
            }
            return true
        }
        return false // 不管成功失败，都拦截 WebView 自己处理
    }

    private fun isAppInstalled(context: Context, packageName: String): Boolean {
        try {
            context.getPackageManager().getPackageInfo(packageName, 0)
            return true
        } catch (e: PackageManager.NameNotFoundException) {
            return false
        }
    }

    private fun enterImmersiveMode() {
        // 设置沉浸式状态栏
        window.decorView.systemUiVisibility =
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
                    View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                    View.SYSTEM_UI_FLAG_LAYOUT_STABLE
        window.statusBarColor = Color.TRANSPARENT
        window.navigationBarColor = Color.TRANSPARENT
    }

}