package com.hybrid.android.push

import android.content.Context
import android.os.Build
import android.util.Log
import com.hybrid.android.BuildConfig
import com.hybrid.android.domain.PrefsConfigStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.util.concurrent.TimeUnit

/**
 * FCM token 上报工具（见 docs/admin/07-push.md §3 / §2.2）。
 *
 * 上报端点：`POST {configBaseUrl}/api/app/push/register-token`
 * body: { appId, token, palcode, platform, model }
 *
 * 调用方使用协程（IO 调度器），失败静默忽略、不阻塞启动（见 07-push.md §3）。
 * 上报地址从 bootstrap.json 的 configUrl 同源派生（与域名容灾配置同一套体系，守 ADR-0002）。
 *
 * ADR-0009：appId 用 BuildConfig.APPLICATION_ID（全局唯一），palcode 仅作 URL 参数随 token 一起存下。
 */
object TokenRegistrar {

    private const val TAG = "HybridPush"
    private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()

    private val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(5, TimeUnit.SECONDS)
            .readTimeout(5, TimeUnit.SECONDS)
            .callTimeout(8, TimeUnit.SECONDS)
            .retryOnConnectionFailure(false)
            .build()
    }

    /**
     * 异步上报 token（已在 IO 调度器，直接同步调用 OkHttp）。
     * 失败只记日志，不抛异常，不影响调用方流程。
     */
    suspend fun register(context: Context, token: String) = withContext(Dispatchers.IO) {
        runCatching {
            val registerUrl = resolveRegisterUrl(context) ?: run {
                Log.w(TAG, "token 上报：无法解析注册端点 URL，跳过")
                return@withContext
            }

            val payload = JSONObject().apply {
                // ADR-0009：以 applicationId 为 appId 键上报，全局唯一
                put("appId", BuildConfig.APPLICATION_ID)
                put("token", token)
                put("palcode", BuildConfig.PAL_CODE)
                put("platform", "android")
                put("model", "${Build.MANUFACTURER} ${Build.MODEL}".trim())
            }.toString()

            val request = Request.Builder()
                .url(registerUrl)
                .post(payload.toRequestBody(JSON_MEDIA))
                .header("Content-Type", "application/json")
                .header("Accept", "application/json")
                .build()

            client.newCall(request).execute().use { resp ->
                if (resp.isSuccessful) {
                    Log.d(TAG, "token 上报成功: appId=${BuildConfig.APPLICATION_ID}")
                } else {
                    Log.w(TAG, "token 上报 HTTP ${resp.code}，已忽略")
                }
            }
        }.onFailure {
            // 静默忽略所有网络/解析错误，不影响 APP 正常启动
            Log.w(TAG, "token 上报失败（已忽略）: ${it.javaClass.simpleName} - ${it.message}")
        }
    }

    /**
     * 从 bootstrap.json 的 configUrl 派生 token 注册端点。
     * configUrl 形如 `https://example.com/api/app/config`，
     * 派生规则：去掉最后一段 `/config`，追加 `/push/register-token`。
     * 如果 configUrl 不含 `/api/app/` 前缀，则回落为直接在 configUrl 基座追加端点路径。
     *
     * 示例：
     *   https://example.com/api/app/config → https://example.com/api/app/push/register-token
     */
    private fun resolveRegisterUrl(context: Context): String? {
        val store = PrefsConfigStore(context)
        val configUrl = store.configUrl().trim().trimEnd('/')
        if (configUrl.isBlank()) return null

        // 如果 configUrl 以 /api/app/config 结尾，替换成 /api/app/push/register-token
        return if (configUrl.endsWith("/api/app/config")) {
            configUrl.removeSuffix("/config") + "/push/register-token"
        } else {
            // 通用回落：去掉最后一段，拼上推送端点
            val base = configUrl.substringBeforeLast('/')
            "$base/push/register-token"
        }
    }
}
