package com.hybrid.android.track

import android.content.Context
import android.content.SharedPreferences
import android.os.Build
import android.util.Log
import androidx.core.content.edit
import com.adjust.sdk.Adjust
import com.google.android.gms.ads.identifier.AdvertisingIdClient
import com.hybrid.android.BuildConfig
import com.hybrid.android.domain.PrefsConfigStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.security.MessageDigest
import java.util.UUID
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume

/**
 * 设备信息（GAID / Adjust ADID）采集与上报工具。
 *
 * 上报端点：`POST {configBaseUrl}/api/app/device/register`
 * body: { appId, palcode, deviceKey, deviceName, gaid, adid }
 *
 * 只上报「有有效 GAID」的设备：GAID 是 Meta/TikTok 受众上传的唯一有效标识，
 * 拿不到（无 GMS / 用户关闭广告个性化返回全零）时整条不上报，服务端同样拒收。
 *
 * 结构与风格完全仿照 [com.hybrid.android.push.TokenRegistrar]：懒加载 OkHttpClient、
 * 短超时、失败静默忽略、suspend + IO 调度器、URL 从 bootstrap 的 configUrl 同源派生。
 *
 * 节流：同一设备/同一标识组合 24 小时内只上报一次（哈希对比），避免每次启动都打后台。
 * adid 参与哈希计算——首次启动 Adjust 归因未完成时 adid 为空，二次启动归因完成后哈希变化，
 * 会自动触发一次补报，无需额外机制。
 */
object DeviceInfoRegistrar {

    private const val TAG = "HybridDevice"
    private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()

    private const val PREFS_NAME = "device_report"
    private const val KEY_DEVICE_KEY = "device_key"
    private const val KEY_LAST_HASH = "last_hash"
    private const val KEY_LAST_AT = "last_at"

    /** 全零 GAID：用户关闭广告个性化后系统返回的占位值，等价于「无标识」。 */
    private const val ZERO_GAID = "00000000-0000-0000-0000-000000000000"

    /** 节流窗口：24 小时。 */
    private const val THROTTLE_MS = 24 * 60 * 60 * 1000L

    /** ADID 回调超时：Adjust SDK 尚未拿到归因结果时不应阻塞启动流程。 */
    private const val ADID_TIMEOUT_MS = 3000L

    private val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(5, TimeUnit.SECONDS)
            .readTimeout(5, TimeUnit.SECONDS)
            .callTimeout(8, TimeUnit.SECONDS)
            .retryOnConnectionFailure(false)
            .build()
    }

    /**
     * 采集设备标识并上报，已在 IO 调度器执行。失败只记日志，不抛异常，不影响调用方流程。
     */
    suspend fun registerIfNeeded(context: Context) = withContext(Dispatchers.IO) {
        runCatching {
            val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            val deviceKey = resolveDeviceKey(prefs)
            val deviceName = "${Build.MANUFACTURER} ${Build.MODEL}".trim()

            val gaid = collectGaid(context)
            if (gaid.isEmpty()) {
                // 无有效 GAID（无 GMS / 关闭广告个性化）→ 整条不上报；
                // 不写节流记录，用户后续重新开启广告个性化时下次启动会自动上报。
                Log.d(TAG, "无有效 GAID，跳过设备信息上报")
                return@withContext
            }
            val adid = collectAdid()

            val payloadHash = sha256(
                "$deviceKey|$deviceName|$gaid|$adid|${BuildConfig.APPLICATION_ID}"
            )
            val lastHash = prefs.getString(KEY_LAST_HASH, null)
            val lastAt = prefs.getLong(KEY_LAST_AT, 0L)
            val now = System.currentTimeMillis()
            if (payloadHash == lastHash && now - lastAt < THROTTLE_MS) {
                Log.d(TAG, "设备信息未变化且在节流窗口内，跳过上报")
                return@withContext
            }

            val registerUrl = resolveRegisterUrl(context) ?: run {
                Log.w(TAG, "设备信息上报：无法解析注册端点 URL，跳过")
                return@withContext
            }

            val payload = JSONObject().apply {
                put("appId", BuildConfig.APPLICATION_ID)
                put("palcode", BuildConfig.PAL_CODE)
                put("deviceKey", deviceKey)
                put("deviceName", deviceName)
                put("gaid", gaid)
                put("adid", adid)
            }.toString()

            val request = Request.Builder()
                .url(registerUrl)
                .post(payload.toRequestBody(JSON_MEDIA))
                .header("Content-Type", "application/json")
                .header("Accept", "application/json")
                .build()

            client.newCall(request).execute().use { resp ->
                if (resp.isSuccessful) {
                    // 只有成功才写回哈希/时间戳，失败留空以便下次启动自动重试。
                    prefs.edit {
                        putString(KEY_LAST_HASH, payloadHash)
                        putLong(KEY_LAST_AT, now)
                    }
                    Log.d(TAG, "设备信息上报成功: appId=${BuildConfig.APPLICATION_ID}")
                } else {
                    Log.w(TAG, "设备信息上报 HTTP ${resp.code}，已忽略")
                }
            }
        }.onFailure {
            // 静默忽略所有网络/解析错误，不影响 APP 正常启动
            Log.w(TAG, "设备信息上报失败（已忽略）: ${it.javaClass.simpleName} - ${it.message}")
        }
    }

    /** 设备本地唯一 key：首次生成后持久化，跨启动稳定。 */
    private fun resolveDeviceKey(prefs: SharedPreferences): String {
        prefs.getString(KEY_DEVICE_KEY, null)?.takeIf { it.isNotBlank() }?.let { return it }
        val generated = UUID.randomUUID().toString()
        prefs.edit { putString(KEY_DEVICE_KEY, generated) }
        return generated
    }

    /** 采集 GAID（Google 广告 ID）。用户关闭广告个性化 / 拿不到时返回空串。 */
    private fun collectGaid(context: Context): String = runCatching {
        val info = AdvertisingIdClient.getAdvertisingIdInfo(context)
        val id = info.id
        if (info.isLimitAdTrackingEnabled || id.isNullOrBlank() || id == ZERO_GAID) {
            ""
        } else {
            id.lowercase()
        }
    }.getOrDefault("")

    /**
     * 采集 Adjust ADID（设备归因 ID）。仅当该渠道包已绑定 Adjust App Token（[AdjustBootstrap.enabled]）
     * 时才尝试；未绑定时 Adjust SDK 未初始化，调用会一直挂起，故直接跳过。
     * 超时 3s 内拿不到结果视为暂不可用，返回空串（下次启动 hash 变化会自动补报）。
     */
    private suspend fun collectAdid(): String {
        if (!AdjustBootstrap.enabled) return ""
        return withTimeoutOrNull(ADID_TIMEOUT_MS) {
            suspendCancellableCoroutine { cont ->
                runCatching {
                    Adjust.getAdid { adid ->
                        if (cont.isActive) {
                            cont.resume(adid?.takeIf { it.isNotBlank() }?.lowercase() ?: "")
                        }
                    }
                }.onFailure {
                    if (cont.isActive) cont.resume("")
                }
            }
        } ?: ""
    }

    /** sha256 十六进制摘要。 */
    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(input.toByteArray(Charsets.UTF_8))
        return digest.joinToString("") { "%02x".format(it) }
    }

    /**
     * 从 bootstrap.json 的 configUrl 派生设备注册端点，规则同 [com.hybrid.android.push.TokenRegistrar]。
     * configUrl 形如 `https://example.com/api/app/config`，
     * 派生规则：去掉最后一段 `/config`，追加 `/device/register`。
     *
     * 示例：
     *   https://example.com/api/app/config → https://example.com/api/app/device/register
     */
    private fun resolveRegisterUrl(context: Context): String? {
        val store = PrefsConfigStore(context)
        val configUrl = store.configUrl().trim().trimEnd('/')
        if (configUrl.isBlank()) return null

        return if (configUrl.endsWith("/api/app/config")) {
            configUrl.removeSuffix("/config") + "/device/register"
        } else {
            val base = configUrl.substringBeforeLast('/')
            "$base/device/register"
        }
    }
}
