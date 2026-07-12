package com.hybrid.android

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Color
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkRequest
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.util.Base64
import android.util.Log
import android.view.View
import android.view.ViewGroup
import android.webkit.PermissionRequest
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.Toast
import android.widget.TextView
import android.view.Gravity
import android.graphics.drawable.GradientDrawable
import android.util.TypedValue
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.core.content.edit
import androidx.core.view.ViewCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updateLayoutParams
import androidx.lifecycle.lifecycleScope
import com.appsflyer.AFInAppEventType
import com.appsflyer.AppsFlyerLib
import com.appsflyer.attribution.AppsFlyerRequestListener
import com.google.firebase.messaging.FirebaseMessaging
import com.hybrid.android.brand.BrandHost
import com.hybrid.android.brand.BrandStrategies
import com.hybrid.android.brand.BrandStrategy
import com.hybrid.android.bridge.WebAppBridge
import com.hybrid.android.domain.DomainResolver
import com.hybrid.android.domain.ErrorKind
import com.hybrid.android.domain.ErrorView
import com.hybrid.android.domain.ResolveResult
import com.hybrid.android.push.HybridMessagingService
import com.hybrid.android.push.PushBootstrap
import com.hybrid.android.push.TokenRegistrar
import com.hybrid.android.track.AdjustBootstrap
import kotlinx.coroutines.launch
import org.json.JSONObject

/**
 * 统一的 WebView 壳：负责 WebView/启动图/沉浸式/JS 注入等与品牌无关的通用逻辑。
 * 各大渠道（AP/BP/GP）的差异化行为由 [BrandStrategy] 插件承担，
 * 通过 [BrandStrategies.create] 按 BuildConfig.BRAND 选取。
 */
class WebViewActivity : ComponentActivity(), BrandHost {
    private lateinit var _webView: WebView
    private lateinit var rootLayout: FrameLayout
    private lateinit var splashImageView: ImageView
    private lateinit var errorView: ErrorView

    private var currentPathValue: String? = null
    private val eventValues = HashMap<String, Any>()

    private val strategy: BrandStrategy = BrandStrategies.create()

    // 域名容灾
    private val domainResolver: DomainResolver by lazy { DomainResolver(this) }

    /** 运行中容灾防抖：上次重走容灾的时刻 + 30s 窗口内最多一次（见 02 文档「运行中容灾」）。 */
    private var lastFailoverAtMs = 0L
    private val failoverDebounceMs = 30_000L

    /** 顶层路由「再按一次返回退出」：上次按返回的时刻 + 2s 窗口。 */
    private var lastBackPressedAtMs = 0L
    private val exitConfirmWindowMs = 2_000L
    /** 居中的「再按一次退出」提示视图（自绘，避免 targetSdk≥30 时 Toast.setGravity 失效）。 */
    private var exitHintView: TextView? = null
    private val hideExitHint = Runnable { exitHintView?.visibility = View.GONE }

    /** 容灾结果是否已成功加载过页面（用于区分「首屏未出」与「运行中掉线」）。 */
    private var hasLoadedOnce = false

    /** 网络恢复监听只注册一次。 */
    private var networkCallback: ConnectivityManager.NetworkCallback? = null

    /**
     * 推送通知点击时携带的相对 deeplink path（如 `/promo/618`）。
     * 在 DomainResolver 解析出可用域名后，拼成完整 URL 并加载。
     * 仅当本次启动/切换来自通知点击时才有值，否则为 null（加载正常首页）。
     */
    private var pendingPushPath: String? = null

    // ---- BrandHost ----
    override val context: Context get() = this
    // 域名运行时解析：返回当前实际加载的域名（容灾切到备用后随之更新），
    // 让 BrandStrategy 的同源判断（如 Bp 的 url.startsWith(host.domain)）始终正确。
    // 解析完成前回落 BuildConfig.DOMAIN（仅作占位，不作为加载数据源）。
    private var resolvedDomain: String? = null
    override val domain: String get() = resolvedDomain ?: BuildConfig.DOMAIN
    override val currentPath: String? get() = currentPathValue
    override val webView: WebView get() = _webView
    override fun putEventValue(key: String, value: Any) { eventValues[key] = value }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // 推送通知点击时携带的相对 deeplink path（无通知点击则为 null）。
        // 必须在 startResolve() 之前读取，解析完域名后再拼成完整 URL。
        pendingPushPath = intent?.getStringExtra(HybridMessagingService.EXTRA_PUSH_PATH)

        // AppsFlyer 初始化 + 加载持久化归因状态（品牌差异）
        strategy.initTracking(this)
        // Adjust 归因初始化（feature gate，未绑定 App Token 时 no-op，见 ADR-0013）
        AdjustBootstrap.init(this)

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
        rootLayout = FrameLayout(this).apply {
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
            // 禁用边缘 overscroll 拉伸（Android 12+ 默认会把整页含 position:fixed 底部导航一起拉伸变形）。
            overScrollMode = View.OVER_SCROLL_NEVER
            layoutParams = FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT
            )
        }

        // 原生错误页（最上层，默认隐藏）。点刷新重走容灾。
        errorView = ErrorView(this).apply {
            visibility = View.GONE
            onRefresh = { retryResolve() }
        }

        // 添加视图（splash 在 WebView 之上，errorView 在最上层）
        rootLayout.addView(_webView)
        rootLayout.addView(splashImageView)
        rootLayout.addView(errorView)
        setContentView(rootLayout)

        // 沉浸式状态栏 + 品牌固定系统栏配色（加载期即生效，避免白闪）
        enterImmersiveMode()
        applyBrandSystemBars()

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
                window.decorView.systemUiVisibility = View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                // 状态栏/底部导航栏按品牌固定配色（GP 深色不露白、图标明暗自动反转）。
                applyBrandSystemBars()
                // 页面真正出来了：记一次成功，隐藏错误页/splash。
                hasLoadedOnce = true
                errorView.visibility = View.GONE
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

            // 运行中容灾：仅对主框架错误生效（忽略图片/JS 等子资源），且带防抖，避免狂换。
            override fun onReceivedError(
                view: WebView,
                request: WebResourceRequest?,
                error: WebResourceError?
            ) {
                super.onReceivedError(view, request, error)
                if (request?.isForMainFrame == true) {
                    Log.w("DomainResolver", "主框架 onReceivedError → 触发运行中容灾")
                    onMainFrameError()
                }
            }

            override fun onReceivedHttpError(
                view: WebView,
                request: WebResourceRequest?,
                errorResponse: WebResourceResponse?
            ) {
                super.onReceivedHttpError(view, request, errorResponse)
                // 仅对主框架的 5xx 触发容灾（4xx 多为页面级，不应换域名）。
                val code = errorResponse?.statusCode ?: 0
                if (request?.isForMainFrame == true && code in 500..599) {
                    Log.w("DomainResolver", "主框架 onReceivedHttpError $code → 触发运行中容灾")
                    onMainFrameError()
                }
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

        // 加载首页：走域名容灾（运行时拉取 + 主→备用 + 区分域名故障/本机网络），不再编译期硬编码域名。
        startResolve()

        // FCM：申请通知权限（Android 13+），主动触发一次 token 注册（补充 onNewToken 时序）。
        requestNotificationPermissionIfNeeded()
        triggerFcmTokenRegistrationIfAvailable()

        // 返回键
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (_webView.canGoBack()) {
                    _webView.goBack()
                    return
                }
                // 顶层路由：2s 内再按一次才退出，否则居中提示。
                val now = System.currentTimeMillis()
                if (now - lastBackPressedAtMs <= exitConfirmWindowMs) {
                    exitHintView?.removeCallbacks(hideExitHint)
                    exitHintView?.visibility = View.GONE
                    finish()
                } else {
                    lastBackPressedAtMs = now
                    showExitHint()
                }
            }
        })
    }

    /** 屏幕居中展示「再按一次退出」提示（自绘 toast，2s 后自动隐藏）。 */
    private fun showExitHint() {
        val density = resources.displayMetrics.density
        val tv = exitHintView ?: TextView(this).apply {
            text = "Press back again to exit"
            setTextColor(Color.WHITE)
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            val padH = (20 * density).toInt()
            val padV = (12 * density).toInt()
            setPadding(padH, padV, padH, padV)
            background = GradientDrawable().apply {
                cornerRadius = 24 * density
                setColor(Color.parseColor("#CC000000"))
            }
            addContentView(
                this,
                FrameLayout.LayoutParams(
                    FrameLayout.LayoutParams.WRAP_CONTENT,
                    FrameLayout.LayoutParams.WRAP_CONTENT
                ).apply { gravity = Gravity.CENTER }
            )
            exitHintView = this
        }
        tv.visibility = View.VISIBLE
        tv.removeCallbacks(hideExitHint)
        tv.postDelayed(hideExitHint, exitConfirmWindowMs)
    }

    // ========================== 域名容灾接入 ==========================

    /** 跑一次状态机：命中→加载；否则按归因显示原生错误页。splash 在解析期间持续展示，无白屏。 */
    private fun startResolve() {
        lifecycleScope.launch {
            when (val r = domainResolver.resolve()) {
                is ResolveResult.Loadable -> {
                    resolvedDomain = r.domain // 供 BrandStrategy 同源判断使用
                    errorView.visibility = View.GONE
                    if (splashImageView.visibility != View.VISIBLE) {
                        splashImageView.alpha = 1f
                        splashImageView.visibility = View.VISIBLE
                    }
                    // 推送点击深链：有 pendingPushPath 时用其覆盖加载 URL（相对 path + palcode）。
                    // 域名来自 DomainResolver（运行时值），绝不编译期硬编码（守 ADR-0002）。
                    val loadUrl = pendingPushPath
                        ?.takeIf { it.isNotBlank() }
                        ?.let { path ->
                            val cleanPath = if (path.startsWith("/")) path else "/$path"
                            "${r.domain}$cleanPath?palcode=${BuildConfig.PAL_CODE}"
                                .also {
                                    Log.d("HybridPush", "推送 deeplink 加载: $it")
                                    strategy.onPushOpen(path, r.domain, this@WebViewActivity)
                                }
                        } ?: r.url
                    pendingPushPath = null  // 消费后清空，防止 retryResolve 重复加载
                    _webView.loadUrl(loadUrl)
                }
                ResolveResult.ServiceDown -> showErrorView(ErrorKind.SERVICE_DOWN) // B：域名/服务
                ResolveResult.NoNetwork -> showErrorView(ErrorKind.NO_NETWORK)     // A：本机网络
            }
        }
    }

    /** 刷新/自动重试：隐藏错误页、重新展示 splash，再走一次容灾。 */
    private fun retryResolve() {
        errorView.visibility = View.GONE
        splashImageView.alpha = 1f
        splashImageView.visibility = View.VISIBLE
        startResolve()
    }

    /** 显示原生错误页（非网页）：图标 + 文案 + 刷新；并注册网络恢复自动重试。 */
    private fun showErrorView(kind: ErrorKind) {
        splashImageView.visibility = View.GONE
        errorView.bind(kind)
        errorView.visibility = View.VISIBLE
        registerNetworkRecoveryOnce()
    }

    /**
     * 运行中容灾：主框架报错时，带防抖地重走容灾（30s 窗口内最多一次），避免抖动网络下狂换。
     * 仅在已成功加载过页面后才介入（首屏失败由 startResolve 的错误页处理）。
     */
    private fun onMainFrameError() {
        if (!hasLoadedOnce) return
        val now = System.currentTimeMillis()
        if (now - lastFailoverAtMs < failoverDebounceMs) {
            Log.d("DomainResolver", "运行中容灾防抖命中（30s 内已重走过），忽略本次")
            return
        }
        lastFailoverAtMs = now
        // 重走 STEP0~3（STEP0 仍先闸一道）。
        startResolve()
    }

    /** 注册一次网络恢复回调：网络回来后自动重试（错误页场景）。 */
    private fun registerNetworkRecoveryOnce() {
        if (networkCallback != null) return
        val cm = getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return
        val cb = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                runOnUiThread {
                    // 仅当错误页正展示时才自动重试，避免无谓刷新。
                    if (errorView.visibility == View.VISIBLE) retryResolve()
                }
            }
        }
        val request = NetworkRequest.Builder()
            .addCapability(android.net.NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        runCatching { cm.registerNetworkCallback(request, cb) }
            .onSuccess { networkCallback = cb }
    }

    private fun unregisterNetworkRecovery() {
        val cm = getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
        networkCallback?.let { cb -> runCatching { cm?.unregisterNetworkCallback(cb) } }
        networkCallback = null
    }

    private val requestPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { /* no-op */ }

    /** POST_NOTIFICATIONS（Android 13+）权限申请结果回调：结果仅记录日志，不影响启动流程。 */
    private val requestNotificationPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            Log.d("HybridPush", "POST_NOTIFICATIONS 申请结果: $granted")
        }

    private fun checkAndRequestPermission() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.CAMERA)
            != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissionLauncher.launch(Manifest.permission.CAMERA)
        }
    }

    /**
     * Android 13+（API 33）需要动态申请 POST_NOTIFICATIONS 权限。
     * minSdk 29，仅在 33+ 设备上才需要申请；旧版本权限由安装时自动授予。
     * 未配置 Firebase 时此函数仍安全（仅申请权限，不依赖 Firebase）。
     */
    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED
            ) {
                requestNotificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            }
        }
    }

    /**
     * 主动触发一次 FCM token 注册（首次启动 / token 轮换后 onNewToken 会自动触发，
     * 此处是补充保险：确保 token 注册不依赖 onNewToken 回调的时序）。
     * 仅当 Firebase 可用时才执行，否则 no-op。
     */
    private fun triggerFcmTokenRegistrationIfAvailable() {
        if (!PushBootstrap.guard(this, "主动 token 注册")) return
        FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
            if (token.isNotBlank()) {
                lifecycleScope.launch {
                    TokenRegistrar.register(this@WebViewActivity, token)
                }
            }
        }.addOnFailureListener {
            Log.w("HybridPush", "获取 FCM token 失败（已忽略）: ${it.message}")
        }
    }

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        setIntent(intent)  // 更新 Activity 的 intent，使 getIntent() 始终返回最新值

        // 推送通知点击（Activity 已在前台/后台时）：有 push path 则经 DomainResolver 跳转。
        val pushPath = intent?.getStringExtra(HybridMessagingService.EXTRA_PUSH_PATH)
        if (!pushPath.isNullOrBlank()) {
            Log.d("HybridPush", "onNewIntent 收到推送 path: $pushPath")
            val domain = resolvedDomain
            if (domain != null) {
                // 已有解析好的域名：直接拼 URL 加载，无需重走容灾（域名已验活）
                val cleanPath = if (pushPath.startsWith("/")) pushPath else "/$pushPath"
                val url = "$domain$cleanPath?palcode=${BuildConfig.PAL_CODE}"
                Log.d("HybridPush", "推送 deeplink（快路径）加载: $url")
                strategy.onPushOpen(pushPath, domain, this)
                _webView.loadUrl(url)
            } else {
                // 尚未完成首次解析（极少情况）：保存 path，下次 startResolve 消费
                pendingPushPath = pushPath
                startResolve()
            }
            return
        }

        // 普通系统 deeplink
        intent?.data?.let { strategy.onDeepLinkIntent(it, this) }
    }

    override fun onResume() {
        super.onResume()
        strategy.onResume(this)
    }

    override fun onDestroy() {
        unregisterNetworkRecovery()
        super.onDestroy()
    }

    private fun enterImmersiveMode() {
        window.decorView.systemUiVisibility =
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
                    View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
                    View.SYSTEM_UI_FLAG_LAYOUT_STABLE
        window.statusBarColor = Color.TRANSPARENT
        window.navigationBarColor = Color.TRANSPARENT
    }

    /**
     * 按品牌固定状态栏 / 底部导航栏配色（图标明暗按亮度自动反转）：
     *   - ap / bp（浅色站）→ 纯白
     *   - gp（深色站）→ #1C1D27
     * 各站页头/页脚是图片或动态色、采样不稳，故用与设计稿一致的固定值，且加载期就生效、无白闪。
     */
    private fun applyBrandSystemBars() {
        val color = when (BuildConfig.BRAND) {
            "gp" -> Color.parseColor("#1C1D27")
            else -> Color.WHITE            // ap / bp 及其它浅色站
        }
        applySystemBarColors(color, color)
    }

    private fun applySystemBarColors(topColor: Int, bottomColor: Int) {
        val top = topColor or 0xFF000000.toInt()
        val bottom = bottomColor or 0xFF000000.toInt()
        // 状态栏区域用 rootLayout 背景上色（MIUI 全屏 flag 下 statusBarColor 常不生效；
        // rootLayout 在状态栏后面、必然渲染）。底部导航栏 navigationBarColor 可直接生效。
        rootLayout.setBackgroundColor(top)
        window.statusBarColor = top
        window.navigationBarColor = bottom
        WindowCompat.getInsetsController(window, window.decorView).apply {
            isAppearanceLightStatusBars = isLightColor(top)       // 浅底→深色图标
            isAppearanceLightNavigationBars = isLightColor(bottom)
        }
    }

    /** 颜色亮度判断（感知加权）：> 0.6 视为浅色。 */
    private fun isLightColor(c: Int): Boolean {
        val luminance = (0.299 * Color.red(c) + 0.587 * Color.green(c) + 0.114 * Color.blue(c)) / 255.0
        return luminance > 0.6
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
        // Adjust 同源 fan-out：与 AppsFlyer 共用同一批事件；未绑定 App Token 时 no-op（ADR-0013）。
        AdjustBootstrap.trackEvent(eventName, eventValues)
        AppsFlyerLib.getInstance().logEvent(
            this,
            eventName,
            eventValues,
            object : AppsFlyerRequestListener {
                override fun onSuccess() {
                    Log.d("Appsflyer", "Sent event SUCCESS: $eventName")
                    // Toast 仅在开启测试事件时展示，生产包静默发送
                    if (BuildConfig.ENABLE_TEST_EVENTS) {
                        runOnUiThread { showToast("事件发送成功: $eventName") }
                    }
                }

                override fun onError(errorCode: Int, p1: String) {
                    Log.e("Appsflyer", "Sent event FAILED: $eventName, errorCode: $errorCode, message: $p1")
                    // Toast 仅在开启测试事件时展示，生产包静默发送
                    if (BuildConfig.ENABLE_TEST_EVENTS) {
                        runOnUiThread { showToast("事件发送失败: $p1") }
                    }
                }
            }
        )
    }
}
