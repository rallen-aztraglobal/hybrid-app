package com.hybrid.android.brand

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.content.SharedPreferences
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
 * AP（ArenaPlus）策略。原与 GP 共用 StandardStrategy，因首存判定改走 getById 而拆出。
 *
 * 归因逻辑：
 *  - getById / getByLoginName → 首存判定（跳变 + 首日兜底，见 [trackFirstDepositIfNeeded]）：
 *      注册日=首存日 → Purchase，否则 → OldRegPurchase，并同时上报 TPFirstDeposit
 *  - loginAndRegisterV4       → LOGIN / COMPLETE_REGISTRATION
 *  - onResume                 → 未首存且停留钱包页时强刷页面，逼出新的 getById（同 BpStrategy 兜底）
 *
 * 不再拦截 checkDepositTransV2（原首存判定入口，已由 getById 取代；AddToCart 随之不再上报）。
 */
class ApStrategy : BrandStrategy {

    /** 是否已上报过首存（设备级防重闩锁，持久化）。 */
    private var firstDepositReported = false

    /** 上次观测到的 firstDepositFlag；null = 本设备从未观测过（新装/换机首次拉取前）。 */
    private var lastKnownFlag: Boolean? = null

    override fun initTracking(activity: Activity) {
        // OAID 采集：华为设备无 GMS → 拿不到 GAID，不开这项 AppsFlyer 就没有任何广告标识。
        AppsFlyerLib.getInstance().setCollectOaid(true)
        AppsFlyerLib.getInstance().setDebugLog(true)
        AppsFlyerLib.getInstance().init(BuildConfig.AF_DEV_KEY, null, activity)
        AppsFlyerLib.getInstance().start(activity)
        val prefs = prefs(activity)
        firstDepositReported = prefs.getBoolean(KEY_REPORTED, false)
        lastKnownFlag =
            if (prefs.contains(KEY_LAST_FLAG)) prefs.getBoolean(KEY_LAST_FLAG, false) else null
    }

    override fun onResume(host: BrandHost) {
        // 充值常跳出到第三方支付 App，回来后页面不一定主动复检首存状态；
        // 未首存且停留钱包页时强刷一次，逼出新的 getById（同 BpStrategy 的兜底）。
        if (lastKnownFlag == false && host.currentPath?.contains("wallet") == true) {
            host.webView.post {
                host.webView.loadUrl("${host.domain}/wallet?t=${System.currentTimeMillis()}")
            }
        }
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
        // ✅ getById / getByLoginName：首存判定
        if (apiUrl.contains("getById") || apiUrl.contains("getByLoginName")) {
            trackFirstDepositIfNeeded(body, host)
            return
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

    /**
     * 首存判定（跳变 + 首日兜底）：
     *  - 跳变：上次观测 flag=false、本次 =true → 首存就发生在两次观测之间，直接上报，
     *    不看日期——充值后杀进程、跨午夜、次日才打开、设备时区偏移都不丢。
     *  - 首日兜底：本设备从未观测过 flag（新装包首次拉到用户信息时已 =true），仅当
     *    firstDepositDate=当日才上报——历史首存（重装/换机拉到旧数据）不补报；升级安装
     *    （旧包留有 registDateStr）也不走兜底，旧包已经按 checkDepositTransV2 报过，避免双报。
     */
    private fun trackFirstDepositIfNeeded(body: JSONObject, host: BrandHost) {
        val flag = body.optBoolean("firstDepositFlag", false)
        val prevFlag = lastKnownFlag

        if (!firstDepositReported && flag) {
            val firstDepositDate = body.optString("firstDepositDate", "").toLocalDateOrNull()
            val shouldReport = when (prevFlag) {
                false -> true
                null -> firstDepositDate == LocalDate.now() && !isUpgradedInstall(host.context)
                true -> false
            }
            if (shouldReport) {
                val registDate = body.optString("registDate", "").toLocalDateOrNull()
                // 注册日与首存日同为服务端时间，直接互比，不与设备本地日期跨时区错位
                if (registDate == firstDepositDate) {
                    host.sendAFEvent("Purchase")
                } else {
                    host.sendAFEvent("OldRegPurchase")
                }
                host.sendAFEvent("TPFirstDeposit")
                firstDepositReported = true
                Log.d(TAG, "首存已上报: firstDepositDate=$firstDepositDate registDate=$registDate prevFlag=$prevFlag")
            } else {
                Log.d(TAG, "首存不上报: prevFlag=$prevFlag firstDepositDate=$firstDepositDate")
            }
        }

        lastKnownFlag = flag
        prefs(host.context).edit {
            putBoolean(KEY_LAST_FLAG, flag)
            putBoolean(KEY_REPORTED, firstDepositReported)
        }
    }

    /** 装过旧首存逻辑（StandardStrategy）的升级安装：旧包每次 getById 都会写 registDateStr。 */
    private fun isUpgradedInstall(context: Context): Boolean =
        prefs(context).contains(KEY_LEGACY_REGIST_DATE)

    private fun prefs(context: Context): SharedPreferences =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    private companion object {
        const val TAG = "ApStrategy"
        const val PREFS = "user_cache"
        const val KEY_REPORTED = "first_deposit_reported"
        const val KEY_LAST_FLAG = "last_first_deposit_flag"

        /** 旧 StandardStrategy 写入的注册日缓存 key，仅作升级安装标记读取。 */
        const val KEY_LEGACY_REGIST_DATE = "registDateStr"
    }
}
