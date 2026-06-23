package com.hybrid.android

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Color
import android.net.Uri
import android.os.Bundle
import android.util.Base64
import android.util.Log
import android.view.View
import android.view.ViewGroup
import android.webkit.PermissionRequest
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.core.content.edit
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updateLayoutParams
import com.appsflyer.AFInAppEventType
import com.appsflyer.AppsFlyerLib
import com.appsflyer.attribution.AppsFlyerRequestListener
import com.hybrid.android.brand.BrandHost
import com.hybrid.android.brand.BrandStrategies
import com.hybrid.android.brand.BrandStrategy
import com.hybrid.android.bridge.WebAppBridge
import org.json.JSONObject

/**
 * 统一的 WebView 壳：负责 WebView/启动图/沉浸式/JS 注入等与品牌无关的通用逻辑。
 * 各大渠道（AP/BP/GP）的差异化行为由 [BrandStrategy] 插件承担，
 * 通过 [BrandStrategies.create] 按 BuildConfig.BRAND 选取。
 */
class WebViewActivity : ComponentActivity(), BrandHost {
    private lateinit var _webView: WebView
    private lateinit var splashImageView: ImageView

    private var currentPathValue: String? = null
    private val eventValues = HashMap<String, Any>()

    private val strategy: BrandStrategy = BrandStrategies.create()

    // ---- BrandHost ----
    override val context: Context get() = this
    override val domain: String = BuildConfig.DOMAIN
    override val currentPath: String? get() = currentPathValue
    override val webView: WebView get() = _webView
    override fun putEventValue(key: String, value: Any) { eventValues[key] = value }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // AppsFlyer 初始化 + 加载持久化归因状态（品牌差异）
        strategy.initTracking(this)

        // 安装事件（仅首次），可选的测试事件由打包开关 ENABLE_TEST_EVENTS 控制
        val prefs = getSharedPreferences("af_install", Context.MODE_PRIVATE)
        if (!prefs.getBoolean("install_tracked", false)) {
            sendAFEvent("Install")
            prefs.edit { putBoolean("install_tracked", true) }

            if (BuildConfig.ENABLE_TEST_EVENTS) {
                // 测试用：首次安装时一次性发送全部事件
                sendAFEvent(AFInAppEventType.LOGIN)
                sendAFEvent(AFInAppEventType.COMPLETE_REGISTRATION)
                sendAFEvent("Purchase")
                sendAFEvent("OldRegPurchase")
                sendAFEvent("TPFirstDeposit")
                sendAFEvent("AddToCart")
            }
        }

        // 根容器
        val rootLayout = FrameLayout(this).apply {
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
            )
        }

        // 全屏 splash 背景图（保持比例不变形）
        splashImageView = ImageView(this).apply {
            setImageResource(R.drawable.splash_fullscreen)
            scaleType = ImageView.ScaleType.CENTER_CROP
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
        }

        // WebView
        _webView = WebView(this).apply {
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            setBackgroundColor(Color.WHITE)
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
        }

        // 添加视图（splash 在上层）
        rootLayout.addView(_webView)
        rootLayout.addView(splashImageView)
        setContentView(rootLayout)

        // 沉浸式状态栏
        enterImmersiveMode()

        val settings: WebSettings = _webView.settings
        settings.javaScriptEnabled = true
        settings.mediaPlaybackRequiresUserGesture = false
        settings.mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW

        // window insets（状态栏/导航栏 padding）
        ViewCompat.setOnApplyWindowInsetsListener(_webView) { view, insets ->
            val systemInsets = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            view.updateLayoutParams<ViewGroup.MarginLayoutParams> {
                topMargin = systemInsets.top
                bottomMargin = systemInsets.bottom
            }
            insets
        }

        // 注入 JSBridge
        _webView.addJavascriptInterface(
            WebAppBridge(_webView) { apiUrl, fullRequestDataJson ->
                handleApiResponse(apiUrl, fullRequestDataJson)
            },
            "JSBridge"
        )

        _webView.webChromeClient = object : WebChromeClient() {
            override fun onPermissionRequest(request: PermissionRequest?) {
                runOnUiThread {
                    request?.grant(request.resources) // 允许 JS 使用相机/麦克风
                }
            }
        }

        _webView.webViewClient = object : WebViewClient() {
            override fun onPageFinished(view: WebView, url: String?) {
                super.onPageFinished(view, url)
                injectInterceptor()
                window.decorView.systemUiVisibility =
                    View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                            View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
                // 淡出 splash 背景
                splashImageView.animate()
                    .alpha(0f)
                    .setDuration(1000)
                    .withEndAction { splashImageView.visibility = View.GONE }
                    .start()
                splashImageView.visibility = View.GONE

                hideAppDownloadEntry(view)
            }

            override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                return strategy.shouldOverrideUrl(request.url.toString(), this@WebViewActivity)
            }

            override fun doUpdateVisitedHistory(view: WebView, url: String, isReload: Boolean) {
                super.doUpdateVisitedHistory(view, url, isReload)
                currentPathValue = view.url
                Log.d("URL", "历史记录 URL 更新为：$currentPathValue")
                if (currentPathValue?.contains("kyc") == true) {
                    checkAndRequestPermission()
                }
            }
        }

        // 加载首页
        _webView.loadUrl("${domain}/?palcode=${BuildConfig.PAL_CODE}")

        // 返回键
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (_webView.canGoBack()) _webView.goBack() else finish()
            }
        })
    }

    private val requestPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { /* no-op */ }

    private fun checkAndRequestPermission() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        intent?.data?.let { strategy.onDeepLinkIntent(it, this) }
    }

    override fun onResume() {
        super.onResume()
        strategy.onResume(this)
    }

    private fun enterImmersiveMode() {
        window.decorView.systemUiVisibility =
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
                    View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                    View.SYSTEM_UI_FLAG_LAYOUT_STABLE
        window.statusBarColor = Color.TRANSPARENT
        window.navigationBarColor = Color.TRANSPARENT
    }

    /** 注入接口拦截脚本（assets/interceptor.js）。 */
    private fun injectInterceptor() {
        val js = assets.open("interceptor.js").bufferedReader().use { it.readText() }
        _webView.evaluateJavascript(js, null)
    }

    /** 隐藏页面内的 App 下载入口。 */
    private fun hideAppDownloadEntry(view: WebView) {
        val css = ".app-download { display: none !important; }"
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

    private fun handleApiResponse(apiUrl: String, fullRequestDataJson: String) {
        try {
            val json = JSONObject(fullRequestDataJson)
            val response = JSONObject(json.getString("response"))
            val body = response.getJSONObject("body")
            Log.d("拦截到URL ====> ", apiUrl)
            strategy.onApiResponse(apiUrl, fullRequestDataJson, body, this)
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    override fun showToast(text: String) {
        Toast.makeText(this@WebViewActivity, text, Toast.LENGTH_LONG).show()
    }

    override fun sendAFEvent(eventName: String) {
        AppsFlyerLib.getInstance().logEvent(
            this,
            eventName,
            eventValues,
            object : AppsFlyerRequestListener {
                override fun onSuccess() {
                    Log.d("Appsflyer", "Sent event SUCCESS: $eventName")
                    runOnUiThread { showToast("事件发送成功: $eventName") }
                }

                override fun onError(errorCode: Int, p1: String) {
                    Log.e("Appsflyer", "Sent event FAILED: $eventName, errorCode: $errorCode, message: $p1")
                    runOnUiThread { showToast("事件发送失败: $p1") }
                }
            }
        )
    }
}
