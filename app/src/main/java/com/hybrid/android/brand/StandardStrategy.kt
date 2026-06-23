package com.hybrid.android.brand

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.util.Log
import androidx.core.content.edit
import com.appsflyer.AFInAppEventType
import com.appsflyer.AppsFlyerLib
import com.hybrid.android.BuildConfig
import com.hybrid.android.utils.toLocalDateOrNull
import org.json.JSONObject
import java.time.LocalDate

/**
 * AP / GP 通用策略（两者逻辑完全一致，仅域名不同，域名来自 BuildConfig.DOMAIN）。
 *
 * 归因逻辑：
 *  - getById / getByLoginName → 缓存 registDate
 *  - loginAndRegisterV4       → LOGIN / COMPLETE_REGISTRATION，并缓存 createdDate
 *  - checkDepositTransV2      → 首存判定后上报 Purchase / OldRegPurchase / TPFirstDeposit / AddToCart
 */
class StandardStrategy : BrandStrategy {

    private var registDate: String? = ""

    override fun initTracking(activity: Activity) {
        AppsFlyerLib.getInstance().setDebugLog(true)
        AppsFlyerLib.getInstance().init(BuildConfig.AF_DEV_KEY, null, activity)
        AppsFlyerLib.getInstance().start(activity)
        registDate = loadRegistDate(activity)
    }

    override fun shouldOverrideUrl(url: String, host: BrandHost): Boolean {
        val uri = Uri.parse(url)
        val scheme = uri.scheme
        // 标准网页直接交给 WebView 加载
        if ("http".equals(scheme, ignoreCase = true) || "https".equals(scheme, ignoreCase = true)) {
            return false
        }
        // 其他自定义 scheme（wechat:// / alipays:// 等）交给系统处理
        return try {
            host.context.startActivity(Intent(Intent.ACTION_VIEW, uri))
            true
        } catch (e: ActivityNotFoundException) {
            Log.w("WebView", "App 未安装: $url")
            true
        }
    }

    override fun onApiResponse(apiUrl: String, fullJson: String, body: JSONObject, host: BrandHost) {
        // ✅ getById / getByLoginName：缓存注册日期
        if (apiUrl.contains("getById") || apiUrl.contains("getByLoginName")) {
            val registDateStr = body.optString("registDate", "")
            saveRegistDate(host.context, registDateStr)
            registDate = registDateStr
            return
        }

        // ✅ Login / 注册事件
        if (apiUrl.contains("loginAndRegisterV4")) {
            val isLogin = body.optBoolean("login", false)
            host.putEventValue("mobileNo", body.getString("mobileNo"))
            host.putEventValue("customerId", body.getString("customerId"))
            if (isLogin) {
                host.sendAFEvent(AFInAppEventType.LOGIN)
            } else {
                host.sendAFEvent(AFInAppEventType.COMPLETE_REGISTRATION)
            }
            val registDateStr = body.optString("createdDate", "")
            saveRegistDate(host.context, registDateStr)
            registDate = registDateStr
        }

        // ✅ 充值事件
        if (apiUrl.contains("checkDepositTransV2")) {
            val depositFlag = body.optBoolean("depositFlag", false)
            val firstDepositFlag = body.optBoolean("firstDepositFlag", false)
            host.showToast("拦截到checkDepositTransV2接口响应： depositFlag = $depositFlag")
            if (depositFlag) {
                if (firstDepositFlag) {
                    val sameDay = LocalDate.now() == registDate?.toLocalDateOrNull()
                    if (sameDay) {
                        host.sendAFEvent("Purchase")
                    } else {
                        host.sendAFEvent("OldRegPurchase")
                    }
                    host.sendAFEvent("TPFirstDeposit")
                } else {
                    host.sendAFEvent("AddToCart")
                }
            }
        }
    }

    private fun saveRegistDate(context: Context, str: String) {
        context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
            .edit { putString("registDateStr", str) }
    }

    private fun loadRegistDate(context: Context): String? =
        context.getSharedPreferences("user_cache", Context.MODE_PRIVATE)
            .getString("registDateStr", null)
}
