package com.hybrid.android.push

import android.content.Context
import android.util.Log
import com.google.firebase.FirebaseApp

/**
 * FCM 功能门控（feature gate，见 docs/admin/07-push.md §6b）。
 *
 * 未配置 google-services.json 时 [FirebaseApp] 没有被初始化，[isAvailable] 返回 false，
 * 所有 FCM 操作（获取 token、展示通知）全部 no-op，不会抛出任何异常。
 *
 * 拿到账号后只需丢入 google-services.json 重打包，[isAvailable] 自动变为 true，零改码。
 */
object PushBootstrap {

    private const val TAG = "HybridPush"

    /**
     * Firebase 是否已初始化（即 google-services.json 存在且被 google-services 插件处理过）。
     * 使用懒判断：FirebaseApp.getApps() 在 Firebase 未初始化时返回空列表，不会崩溃。
     */
    fun isAvailable(context: Context): Boolean = runCatching {
        FirebaseApp.getApps(context).isNotEmpty()
    }.onFailure {
        Log.w(TAG, "探测 Firebase 可用性时异常（已忽略）: ${it.message}")
    }.getOrDefault(false)

    /**
     * 在 FCM 操作前调用的守卫。未配置时记一条 debug 日志并返回 false，调用方直接 return 即可。
     * 这样业务代码只需 `if (!PushBootstrap.guard(context)) return`，不需要每处都写 isAvailable。
     */
    fun guard(context: Context, opName: String = "FCM 操作"): Boolean {
        val ok = isAvailable(context)
        if (!ok) Log.d(TAG, "$opName：Firebase 未初始化，跳过（无 google-services.json）")
        return ok
    }
}
