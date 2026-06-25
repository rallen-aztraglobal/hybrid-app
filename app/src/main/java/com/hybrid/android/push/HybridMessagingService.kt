package com.hybrid.android.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.BitmapFactory
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.hybrid.android.R
import com.hybrid.android.WebViewActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.net.URL

/**
 * FCM 推送接收服务（见 docs/admin/07-push.md §3）。
 *
 * 门控：所有方法首先通过 [PushBootstrap.guard] 检测 Firebase 是否初始化，
 * 未初始化时立即 return（no-op），不抛任何异常。这确保无 google-services.json 时绝不崩溃。
 *
 * token 上报：[onNewToken] 上报给后端 `/api/app/push/register-token`，
 * 以 [BuildConfig.APPLICATION_ID] 作 appId 键（ADR-0009），palcode 仅随 token 存档。
 *
 * 点击跳转：[onMessageReceived] 构建通知，点击后进 [WebViewActivity]，携带相对 path。
 * WebViewActivity 收到 path 后经 DomainResolver 拼出当前可用域名再加载（守 ADR-0002，域名不焊死）。
 *
 * 通知图片：data payload 中可携带 `imageUrl` 字段，服务端推送的是 OSS URL；
 * 客户端在通知构建时异步拉取（IO 线程），失败降级为纯文字通知。
 */
class HybridMessagingService : FirebaseMessagingService() {

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * Firebase SDK 颁发/轮换 token 时回调。
     * 将新 token 上报给后端 token 注册库；失败静默忽略，不阻塞。
     */
    override fun onNewToken(token: String) {
        super.onNewToken(token)
        if (!PushBootstrap.guard(applicationContext, "onNewToken")) return

        Log.d(TAG, "FCM token 已更新，开始上报")
        serviceScope.launch {
            TokenRegistrar.register(applicationContext, token)
        }
    }

    /**
     * 收到推送消息时回调。
     * - data payload 中取 title/body/imageUrl/deeplinkPath（见 07-push.md §3 投递规格）。
     * - notification payload 作回落（若 data 中无 title/body）。
     * - 点击通知 → WebViewActivity，携带 EXTRA_PUSH_PATH，由 Activity 经 DomainResolver 解析。
     */
    override fun onMessageReceived(message: RemoteMessage) {
        super.onMessageReceived(message)
        if (!PushBootstrap.guard(applicationContext, "onMessageReceived")) return

        val data = message.data
        val notification = message.notification

        // title/body 优先取 data payload（服务端 FCM HTTP v1 REST 用 data 字段精确控制）
        val title = data["title"]?.takeIf { it.isNotBlank() }
            ?: notification?.title
            ?: return  // 无 title 不展示

        val body = data["body"]?.takeIf { it.isNotBlank() }
            ?: notification?.body
            ?: ""

        val imageUrl = data["imageUrl"]?.takeIf { it.isNotBlank() }
            ?: notification?.imageUrl?.toString()

        val deeplinkPath = data["deeplinkPath"]?.trim()

        Log.d(TAG, "收到推送: title=$title path=$deeplinkPath imageUrl=$imageUrl")

        // 图片拉取是 IO 操作；通知构建在 IO 协程中完成
        serviceScope.launch {
            showNotification(title, body, imageUrl, deeplinkPath)
        }
    }

    /** 构建并展示系统通知（含图片），点击时带 deeplinkPath 进 WebViewActivity。 */
    private fun showNotification(
        title: String,
        body: String,
        imageUrl: String?,
        deeplinkPath: String?,
    ) {
        ensureNotificationChannel()

        // 点击 Intent：进 WebViewActivity，携带相对 path（不含域名，由 Activity 经 DomainResolver 拼）
        val clickIntent = Intent(applicationContext, WebViewActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (!deeplinkPath.isNullOrBlank()) {
                putExtra(EXTRA_PUSH_PATH, deeplinkPath)
            }
        }
        val pendingFlags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        } else {
            PendingIntent.FLAG_UPDATE_CURRENT
        }
        val pendingIntent = PendingIntent.getActivity(
            applicationContext, PENDING_REQUEST_CODE, clickIntent, pendingFlags
        )

        // 尝试拉取大图（失败降级为纯文字）
        val bigPicture = imageUrl?.let { url ->
            runCatching {
                val conn = URL(url).openConnection().apply { connectTimeout = 3000; readTimeout = 5000 }
                BitmapFactory.decodeStream(conn.getInputStream())
            }.onFailure {
                Log.w(TAG, "推送大图拉取失败（降级为文字通知）: ${it.message}")
            }.getOrNull()
        }

        val builder = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setPriority(NotificationCompat.PRIORITY_HIGH)

        if (bigPicture != null) {
            builder.setLargeIcon(bigPicture)
            builder.setStyle(
                NotificationCompat.BigPictureStyle()
                    .bigPicture(bigPicture)
                    .bigLargeIcon(null as android.graphics.Bitmap?)  // 展开后隐藏大图标
                    .setSummaryText(body)
            )
        } else if (body.length > 40) {
            // 无图时长文用 BigTextStyle 展开
            builder.setStyle(NotificationCompat.BigTextStyle().bigText(body))
        }

        val nm = applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        nm.notify(NOTIFICATION_ID, builder.build())
    }

    /** 确保通知渠道存在（Android 8+）。安全幂等，多次调用无副作用。 */
    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val nm = applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            if (nm.getNotificationChannel(CHANNEL_ID) == null) {
                val channel = NotificationChannel(
                    CHANNEL_ID,
                    CHANNEL_NAME,
                    NotificationManager.IMPORTANCE_HIGH
                ).apply {
                    description = CHANNEL_DESC
                    enableVibration(true)
                }
                nm.createNotificationChannel(channel)
            }
        }
    }

    companion object {
        private const val TAG = "HybridPush"

        /** 推送点击携带的相对 deeplink path 的 Intent extra key。 */
        const val EXTRA_PUSH_PATH = "push_deeplink_path"

        private const val CHANNEL_ID = "hybrid_push"
        private const val CHANNEL_NAME = "推送通知"
        private const val CHANNEL_DESC = "应用推送消息"
        private const val NOTIFICATION_ID = 1001
        private const val PENDING_REQUEST_CODE = 2001
    }
}
