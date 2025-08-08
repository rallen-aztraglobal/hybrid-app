package com.hybrid.android

import android.annotation.SuppressLint
import android.content.ActivityNotFoundException
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
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.OnBackPressedCallback
import androidx.core.content.edit
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updateLayoutParams
import androidx.lifecycle.lifecycleScope
import com.appsflyer.AFInAppEventType
import com.appsflyer.AppsFlyerLib
import com.appsflyer.attribution.AppsFlyerRequestListener
import com.hybrid.android.bridge.WebAppBridge
import com.hybrid.android.utils.toLocalDateOrNull
import kotlinx.coroutines.launch
import org.json.JSONObject
import kotlinx.coroutines.delay

class WebViewActivity : ComponentActivity() {
    private lateinit var webView: WebView
    private lateinit var splashImageView: ImageView

    private var currentPath: String? = null
    private var firstDepositFlag: Boolean? = null
    private val palCode = BuildConfig.PAL_CODE

    private var domain = "https://www.bingoplus.com"

    private var lastUserApiJson: String? = null

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
//        AppsFlyerLib.getInstance().setDebugLog(true)
        AppsFlyerLib.getInstance().init("fXoKsKQwxPCRdhD8CD8q6F", null, this)
        AppsFlyerLib.getInstance().start(this)

        firstDepositFlag = loadFirstDepositFlag(this);

        val prefs = getSharedPreferences("af_install", Context.MODE_PRIVATE)
        if (!prefs.getBoolean("install_tracked", false)) {
            sendAFEvent("Install")
            prefs.edit { putBoolean("install_tracked", true) }
        }

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

        // 注入 JSBridge
        webView.addJavascriptInterface(
            WebAppBridge(webView) { apiUrl, fullRequestDataJson -> handleApiResponse(apiUrl, fullRequestDataJson) },
            "JSBridge"
        )

        // 设置 WebViewClient
        webView.webViewClient = object : WebViewClient() {
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

//            override fun shouldInterceptRequest(
//                view: WebView?,
//                request: WebResourceRequest?
//            ): WebResourceResponse? {
//                request?.url?.toString()?.let { url ->
//                    if (url.contains("websdk.appsflyer.com")) {
//                        // 返回一个空的 JS 响应，阻止加载
//                        val emptyStream = ByteArrayInputStream("".toByteArray(Charsets.UTF_8))
//                        return WebResourceResponse("application/javascript", "UTF-8", emptyStream)
//                    }
//                }
//                return super.shouldInterceptRequest(view, request)
//            }

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
                // ✅ Complete Registration：路径包含 /joinsuccess
                if (view.url?.contains("/joinsuccess") ?: false) {
                    sendAFEvent(AFInAppEventType.COMPLETE_REGISTRATION)
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

    override fun onNewIntent(intent: Intent?) {
        super.onNewIntent(intent)
        handleDeepLink(intent)
    }

    private fun handleDeepLink(intent: Intent?) {
        val data: Uri? = intent?.data
        if (data != null && data.scheme == "bingo") {
            val path = data.host
            if (path == "deposit-success-back" && firstDepositFlag == true) {
                sendAFEvent("AddToCart")
            }
        }
    }

    override fun onResume() {
        super.onResume()
        Log.d("Last", lastUserApiJson ?: "");
        if (firstDepositFlag == false) {
            if (currentPath?.contains("wallet") ?: false) {
                webView.post {
                    var url = "${domain}/wallet?t=${System.currentTimeMillis()}"
                    webView.loadUrl(url)
                }
            }
//            lifecycleScope.launch {
//                delay(1000)
//                replayUserApi()
//            }
        }
    }

    fun saveFirstDepositFlag(context: Context, flag: Boolean?) {
        val prefs = context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
        prefs.edit { putBoolean("first_deposit_flag", flag ?: false) }
    }

    fun loadFirstDepositFlag(context: Context): Boolean? {
        val prefs = context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
        return if (prefs.contains("first_deposit_flag")) {
            prefs.getBoolean("first_deposit_flag", false)
        } else {
            null
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

    private fun injectInterceptor() {
        val js = """
        (function() {
            if (window._apiInterceptorInjected) return;
            window._apiInterceptorInjected = true;

            function shouldIntercept(url) {
               return url.includes("_glaxy_c66_");
            }

            function reportRequest(data) {
                JSBridge.onApiResponse(JSON.stringify(data));
            }

            const origFetch = window.fetch;
            window.fetch = function(input, init = {}) {
                const url = typeof input === 'string' ? input : input.url;
                const method = init.method || 'GET';
                const headers = init.headers || {};
                const body = init.body || null;
                return origFetch(input, init).then(resp => {
                    if (shouldIntercept(url)) {
                        resp.clone().text().then(text => {
                            reportRequest({ url, method, headers, body, response: text });
                        });
                    }
                    return resp;
                });
            };

            const origOpen = XMLHttpRequest.prototype.open;
            const origSend = XMLHttpRequest.prototype.send;
            const origSetRequestHeader = XMLHttpRequest.prototype.setRequestHeader;

            XMLHttpRequest.prototype.open = function(method, url) {
                this._url = url;
                this._method = method;
                this._headers = {};
                return origOpen.apply(this, arguments);
            };
            XMLHttpRequest.prototype.setRequestHeader = function(key, value) {
                this._headers[key] = value;
                return origSetRequestHeader.call(this, key, value);
            };
            XMLHttpRequest.prototype.send = function(body) {
                this._body = body;
                this.addEventListener('load', function() {
                    if (shouldIntercept(this._url)) {
                        reportRequest({
                            url: this._url,
                            method: this._method,
                            headers: this._headers,
                            body: this._body,
                            response: this.responseText
                        });
                    }
                });
                return origSend.call(this, body);
            };

            // 新增：重放请求方法
            window.replayRequest = function(lastUserApiJsonStr) {
                try {
                    const data = JSON.parse(lastUserApiJsonStr);
                    const url = data.url || data.apiUrl;
                    const method = (data.method || 'GET').toUpperCase();
                    const headers = data.headers || {};
                    const body = data.body || null;

                    const fetchOptions = { method, headers };

                    if (method === 'POST' || method === 'PUT') {
                        fetchOptions.body = body;
                    }

                    return fetch(url, fetchOptions)
                        .then(response => response.text())
                        .then(text => {
                            console.log('Replay response:', text);
                            return text;
                        })
                        .catch(err => {
                            console.error('Replay error:', err);
                            throw err;
                        });
                } catch (e) {
                    console.error('Invalid JSON for replayLastRequest:', e);
                    return Promise.reject(e);
                }
            };
        })();
    """.trimIndent()
        webView.evaluateJavascript(js, null)
    }

    private fun replayUserApi() {
        lastUserApiJson?.let {
            showToast("回到App，发起检测")
            webView.evaluateJavascript("window.replayRequest(${it.quoted()})", null)
        }
    }

    private fun String.quoted(): String =
        replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", "\\n").let { "\"$it\"" }


    private fun showToast(text: String) {
       //Toast.makeText(this@WebViewActivity, text, Toast.LENGTH_LONG).show()
    }

    private fun handleApiResponse(apiUrl: String, fullRequestDataJson: String) {

        val json = JSONObject(fullRequestDataJson)
        var responseJson = json.getString("response")
        var response = JSONObject(responseJson)
        var data = response.getJSONObject("body")

        Log.d("拦截到URL ====> ", apiUrl)
        // ✅ 处理 getById / getByLoginName 响应
        if ((apiUrl.contains("getById") || apiUrl.contains("getByLoginName"))) {
            lastUserApiJson = fullRequestDataJson;
            var flag = data.optBoolean("firstDepositFlag", false)
            val firstDepositDateStr = data.optString("firstDepositDate", "")
            val registDateStr = data.optString("registDate", "")
            val firstDepositDate = firstDepositDateStr.toLocalDateOrNull()
            val registDate = registDateStr.toLocalDateOrNull()
            if (firstDepositFlag == null) {
                showToast("首存状态已就绪：firstDepositFlag = $flag");
            } else if (firstDepositFlag == false) {
                showToast("检测首存状态： firstDepositFlag = $flag 历史首存状态：$firstDepositFlag")
            }
            if (firstDepositFlag == false && flag) {
                val sameDay = firstDepositDate == registDate
                if (sameDay) {
                    sendAFEvent("Purchase")
                } else {
                    sendAFEvent("OldRegPurchase")
                }
                sendAFEvent("TPFirstDeposit")
            }
            saveFirstDepositFlag(this, flag);
            firstDepositFlag = flag
            return
        }

        if (apiUrl.contains("billDetail")) {
            showToast("拦截到billDetail接口响应： status = ${data.getInt("status")}")
            if (firstDepositFlag == true && data.getInt("status") == 0) {
                sendAFEvent("AddToCart")
            }
        }

        // ✅ Login 事件：来自 loginAndRegisterV4 且 login 为 true
        if (apiUrl.contains("loginAndRegisterV4") && data.optBoolean("login", false)) {
            sendAFEvent(AFInAppEventType.LOGIN)
            // 如果当前在钱包页面 回到首页
            if (currentPath != null && currentPath!!.contains("wallet")) {
                webView.post {
                    webView.loadUrl("${domain}/wallet")
                }
            }
        }
    }

    private fun sendAFEvent(eventName: String) {
        AppsFlyerLib.getInstance().logEvent(
            this,
            eventName,
            null,
            object : AppsFlyerRequestListener {
                override fun onSuccess() {
                    Log.d("Appsflyer", "Sent event SUCCESS: $eventName")
                    runOnUiThread {
                        showToast("事件发送成功: $eventName")
                    }
                }
                override fun onError(errorCode: Int, p1: String) {
                    Log.e("Appsflyer", "Sent event FAILED: $eventName, errorCode: $errorCode, message: $p1")
                    runOnUiThread {
                        showToast("事件发送失败: $p1")
                    }
                }
            }
        )
    }
}