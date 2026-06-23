package com.hybrid.android.brand

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.util.Log
import androidx.core.content.edit
import com.appsflyer.AFInAppEventType
import com.appsflyer.AppsFlyerLib
import com.hybrid.android.BuildConfig
import com.hybrid.android.utils.toLocalDateOrNull
import org.json.JSONObject

/**
 * BP（BingoPlus）策略。相对标准策略额外集成 HMS / AppsFlyer OAID 采集，
 * 并使用不同的首存判定与 deeplink 处理逻辑：
 *  - getById / getByLoginName → 基于 firstDepositFlag 变化上报 Purchase / OldRegPurchase / TPFirstDeposit
 *  - billDetail               → status==0 时上报 AddToCart
 *  - loginAndRegisterV4       → LOGIN / COMPLETE_REGISTRATION
 *  - onResume                 → 未首存且停留钱包页时强制刷新钱包页
 *  - bingo://deposit-success-back → 已首存时上报 AddToCart
 *  - shouldOverrideUrl        → 站内直载 / 站外系统浏览器 / intent:// 处理
 */
class BpStrategy : BrandStrategy {

    private var firstDepositFlag: Boolean? = null
    private var lastUserApiJson: String? = null

    override fun initTracking(activity: Activity) {
        AppsFlyerLib.getInstance().setCollectOaid(true)
        AppsFlyerLib.getInstance().init(BuildConfig.AF_DEV_KEY, null, activity)
        AppsFlyerLib.getInstance().start(activity)
        firstDepositFlag = loadFirstDepositFlag(activity)
    }

    override fun onResume(host: BrandHost) {
        Log.d("Last", lastUserApiJson ?: "")
        if (firstDepositFlag == false) {
            if (host.currentPath?.contains("wallet") == true) {
                host.webView.post {
                    val url = "${host.domain}/wallet?t=${System.currentTimeMillis()}"
                    host.webView.loadUrl(url)
                }
            }
        }
    }

    override fun onDeepLinkIntent(uri: Uri, host: BrandHost) {
        if (uri.scheme == BuildConfig.DEEPLINK_SCHEME) {
            if (uri.host == "deposit-success-back" && firstDepositFlag == true) {
                host.sendAFEvent("AddToCart")
            }
        }
    }

    override fun shouldOverrideUrl(url: String, host: BrandHost): Boolean {
        val uri = Uri.parse(url)
        val scheme = uri.scheme

        if ("http".equals(scheme, ignoreCase = true) || "https".equals(scheme, ignoreCase = true)) {
            if (url.startsWith(host.domain)) {
                // 自己域名，WebView 内加载
                return false
            } else {
                // 外部网站，用系统浏览器打开
                try {
                    host.context.startActivity(Intent(Intent.ACTION_VIEW, uri))
                } catch (e: Exception) {
                    e.printStackTrace()
                }
                if (host.currentPath?.contains("/confirm") == true) {
                    if (host.webView.canGoBack()) host.webView.goBack()
                }
                return true
            }
        } else {
            // 其他自定义 scheme（wechat:// / alipays:// 等）
            try {
                host.context.startActivity(Intent(Intent.ACTION_VIEW, uri))
            } catch (e: ActivityNotFoundException) {
                Log.w("WebView", "App 未安装: $url")
            }
        }

        // intent:// 类型处理
        if ("intent".equals(scheme, ignoreCase = true)) {
            try {
                val intent = Intent.parseUri(url, Intent.URI_INTENT_SCHEME)
                if (intent.`package` != null) {
                    if (isAppInstalled(host.context, intent.`package`!!)) {
                        host.context.startActivity(intent)
                    } else {
                        // 跳转应用市场
                        val marketIntent = Intent(
                            Intent.ACTION_VIEW,
                            Uri.parse("market://details?id=" + intent.`package`)
                        )
                        host.context.startActivity(marketIntent)
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

    override fun onApiResponse(apiUrl: String, fullJson: String, body: JSONObject, host: BrandHost) {
        // ✅ getById / getByLoginName：首存状态判定
        if (apiUrl.contains("getById") || apiUrl.contains("getByLoginName")) {
            lastUserApiJson = fullJson
            val flag = body.optBoolean("firstDepositFlag", false)
            val firstDepositDate = body.optString("firstDepositDate", "").toLocalDateOrNull()
            val registDate = body.optString("registDate", "").toLocalDateOrNull()
            if (firstDepositFlag == null) {
                host.showToast("首存状态已就绪：firstDepositFlag = $flag")
            } else if (firstDepositFlag == false) {
                host.showToast("检测首存状态： firstDepositFlag = $flag 历史首存状态：$firstDepositFlag")
            }
            if (firstDepositFlag == false && flag) {
                val sameDay = firstDepositDate == registDate
                if (sameDay) {
                    host.sendAFEvent("Purchase")
                } else {
                    host.sendAFEvent("OldRegPurchase")
                }
                host.sendAFEvent("TPFirstDeposit")
            }
            saveFirstDepositFlag(host.context, flag)
            firstDepositFlag = flag
            return
        }

        if (apiUrl.contains("billDetail")) {
            host.showToast("拦截到billDetail接口响应： status = ${body.getInt("status")}")
            if (firstDepositFlag == true && body.getInt("status") == 0) {
                host.sendAFEvent("AddToCart")
            }
        }

        // ✅ Login / 注册事件
        if (apiUrl.contains("loginAndRegisterV4")) {
            host.putEventValue("mobileNo", body.getString("mobileNo"))
            host.putEventValue("customerId", body.getString("customerId"))
            if (body.optBoolean("login", false)) {
                host.sendAFEvent(AFInAppEventType.LOGIN)
            } else {
                host.sendAFEvent(AFInAppEventType.COMPLETE_REGISTRATION)
            }
        }
    }

    private fun saveFirstDepositFlag(context: Context, flag: Boolean?) {
        context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
            .edit { putBoolean("first_deposit_flag", flag ?: false) }
    }

    private fun loadFirstDepositFlag(context: Context): Boolean? {
        val prefs = context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
        return if (prefs.contains("first_deposit_flag")) {
            prefs.getBoolean("first_deposit_flag", false)
        } else {
            null
        }
    }

    private fun isAppInstalled(context: Context, packageName: String): Boolean {
        return try {
            context.packageManager.getPackageInfo(packageName, 0)
            true
        } catch (e: PackageManager.NameNotFoundException) {
            false
        }
    }
}
