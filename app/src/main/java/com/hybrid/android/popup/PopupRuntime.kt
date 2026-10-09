package com.hybrid.android.popup

import android.content.Context
import android.os.SystemClock
import android.provider.Settings
import android.util.Log
import com.hybrid.android.BuildConfig
import com.hybrid.android.domain.PrefsConfigStore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import okhttp3.Call
import okhttp3.Callback
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume

/**
 * 进程级单例装配：配置仓库 / 素材缓存 / 时钟 / 频控 / 便条 / 埋点上报。
 * 与 Activity 解耦——Activity 销毁后埋点仍能在应用作用域里继续补传。
 */
class PopupRuntime private constructor(appContext: Context) {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val kv: KvStore = PrefsKv(appContext.getSharedPreferences("popup_state", Context.MODE_PRIVATE))
    private val configUrl = PrefsConfigStore(appContext).configUrl()

    private val http: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(8, TimeUnit.SECONDS)
        .callTimeout(20, TimeUnit.SECONDS)
        .build()

    val clock = TrustedClock(kv, object : ClockEnv {
        override fun wallMs() = System.currentTimeMillis()
        override fun elapsedMs() = SystemClock.elapsedRealtime()
        override fun bootCount(): Long =
            runCatching { Settings.Global.getLong(appContext.contentResolver, Settings.Global.BOOT_COUNT, 0L) }.getOrDefault(0L)
    })
    val frequency = FrequencyPolicy(kv, clock)
    val tabs = TabStore(kv, clock)
    val cache = AssetCache(java.io.File(appContext.filesDir, "popup/assets"), http)
    val repo = PopupRepository(
        java.io.File(appContext.filesDir, "popup"), clock, cache, ::fetchConfig, SystemClock::elapsedRealtime
    )

    private val reporter = PopupReporter(
        EventQueue(
            PrefsKv(appContext.getSharedPreferences("popup_ev_queue", Context.MODE_PRIVATE)),
            PrefsKv(appContext.getSharedPreferences("popup_ev_inflight", Context.MODE_PRIVATE)),
            persistExecutor = Executors.newSingleThreadExecutor { r -> Thread(r, "popup-ev-persist").apply { isDaemon = true } },
        ), scope,
        BatchMeta(BuildConfig.APPLICATION_ID, BuildConfig.PAL_CODE, BuildConfig.VERSION_NAME, BuildConfig.VERSION_CODE),
        ::sendBatch,
    )

    /** 新老用户：进程内一次性确定（契约 §7.8，见 [resolveNewUser]）。 */
    val isNewUser: Boolean

    @Volatile private var refreshJob: Job? = null

    init {
        val firstLaunchKey = "first_launch_at"
        isNewUser = if (kv.getLong(firstLaunchKey, 0L) > 0L) {
            false
        } else {
            val n = newUserHint ?: true
            kv.putLong(firstLaunchKey, System.currentTimeMillis())
            n
        }
    }

    fun now(): Long = clock.now()

    fun track(event: JSONObject) = reporter.record(event)
    fun flushEvents() = reporter.flushAsync()

    /** 启动一次拉取（并发安全：已在拉则复用）。 */
    fun refreshAsync(): Job {
        refreshJob?.takeIf { it.isActive }?.let { return it }
        return scope.launch { repo.refresh() }.also { refreshJob = it }
    }

    fun currentRefresh(): Job? = refreshJob

    private suspend fun fetchConfig(): String? {
        val url = PopupEndpoints.configUrl(configUrl, BuildConfig.APPLICATION_ID) ?: return null
        return httpCall(Request.Builder().url(url).header("Accept", "application/json").header("Cache-Control", "no-cache").build())
            ?.takeIf { it.first == 200 }?.second
    }

    private suspend fun sendBatch(body: String): Int {
        val url = PopupEndpoints.eventsUrl(configUrl) ?: return -1
        val req = Request.Builder().url(url).post(body.toRequestBody(JSON)).header("Accept", "application/json").build()
        return httpCall(req)?.first ?: -1
    }

    /** 可取消的 OkHttp 调用，返回 (状态码, body)；网络异常返回 null。 */
    private suspend fun httpCall(req: Request): Pair<Int, String>? = suspendCancellableCoroutine { cont ->
        val call = http.newCall(req)
        cont.invokeOnCancellation { call.cancel() }
        call.enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) { if (cont.isActive) cont.resume(null) }
            override fun onResponse(call: Call, response: Response) {
                val r = try { response.use { it.code to (it.body?.string().orEmpty()) } } catch (_: Exception) { null }
                if (cont.isActive) cont.resume(r)
            }
        })
    }

    companion object {
        private const val TAG = "HybridPopup"
        private val JSON = "application/json; charset=utf-8".toMediaType()

        @Volatile private var instance: PopupRuntime? = null

        /** 在 Activity.onCreate 最前调用：升级用户（已有 AppsFlyer 安装记录）不应被当成新用户。 */
        @Volatile private var newUserHint: Boolean? = null

        fun prepare(context: Context) {
            if (instance != null) return
            val installed = context.applicationContext
                .getSharedPreferences("af_install", Context.MODE_PRIVATE).getBoolean("install_tracked", false)
            newUserHint = !installed
        }

        fun get(context: Context): PopupRuntime =
            instance ?: synchronized(this) {
                instance ?: PopupRuntime(context.applicationContext).also {
                    instance = it
                    Log.d(TAG, "PopupRuntime 初始化 isNewUser=${it.isNewUser}")
                }
            }
    }
}
