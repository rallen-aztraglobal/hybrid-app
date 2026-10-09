package com.hybrid.android.popup

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.Call
import okhttp3.Callback
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import java.io.File
import java.io.IOException
import java.security.MessageDigest
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap
import kotlin.coroutines.resume

/**
 * 素材磁盘缓存：文件名 = URL 的 sha256（换图必换 URL，旧文件自然失效）。
 * 不引入 Glide/Coil：OkHttp 下载 + BitmapFactory(inSampleSize) 解码。
 */
class AssetCache(
    private val dir: File,
    private val client: OkHttpClient,
    /** 校验文件是否为可解码图片（默认读图片头）；单测可注入。 */
    private val validator: (File) -> Boolean = ::decodableBounds,
) {

    private val locks = ConcurrentHashMap<String, Mutex>()
    /** 已校验通过的文件名（进程内）。磁盘上历史遗留文件首次 isReady 时校验一次，坏文件当场删除。 */
    private val verified = ConcurrentHashMap.newKeySet<String>()
    /** 正在下载的 tmp 文件名，cleanup 不得删除。 */
    private val activeTmp = ConcurrentHashMap.newKeySet<String>()
    /** decode 与 cleanup 互斥，避免解码到一半文件被清掉。 */
    private val fileLock = Any()

    init { dir.mkdirs() }

    fun fileFor(url: String): File = File(dir, sha256(url))

    fun isReady(url: String): Boolean {
        val f = fileFor(url)
        if (!f.isFile || f.length() <= 0L) return false
        if (f.name in verified) return true
        if (validator(f)) { verified.add(f.name); return true }
        f.delete() // 坏文件（HTML / 截断）不能被永久当成就绪
        return false
    }

    /** 解码失败时调用：删掉缓存文件，下次重新下载。 */
    fun invalidate(url: String) {
        val f = fileFor(url)
        verified.remove(f.name)
        f.delete()
    }

    /** 保证素材在磁盘上；已命中直接 true；否则下载，[timeoutMs] 内未完成（含取消底层请求）视为失败。 */
    suspend fun ensure(url: String, timeoutMs: Long): Boolean {
        if (withContext(Dispatchers.IO) { isReady(url) }) return true
        return withTimeoutOrNull(timeoutMs) { download(url) } ?: false
    }

    suspend fun ensureAll(urls: Collection<String>, timeoutMs: Long): Boolean {
        val deadline = System.currentTimeMillis() + timeoutMs
        for (u in urls) {
            val left = deadline - System.currentTimeMillis()
            if (!ensure(u, maxOf(left, 1L))) return false
        }
        return true
    }

    private suspend fun download(url: String): Boolean {
        val lock = locks.getOrPut(url) { Mutex() }
        return lock.withLock { withContext(Dispatchers.IO) {
            if (isReady(url)) return@withContext true
            val target = fileFor(url)
            val tmp = File(dir, target.name + "." + UUID.randomUUID().toString().take(8) + ".tmp")
            activeTmp.add(tmp.name)
            try {
                val ok = fetchToFile(url, tmp)
                // 先校验能解码出尺寸再 rename：CDN 返回 HTML / 截断文件不会进缓存
                if (!ok || tmp.length() <= 0L || !validator(tmp) || !tmp.renameTo(target)) {
                    tmp.delete()
                    false
                } else {
                    verified.add(target.name)
                    true
                }
            } finally {
                activeTmp.remove(tmp.name)
                locks.remove(url)
            }
        } }
    }

    /** 可取消的下载：协程取消（含超时）时 cancel 底层 Call，真正掐断连接。 */
    private suspend fun fetchToFile(url: String, tmp: File): Boolean =
        suspendCancellableCoroutine { cont ->
            val call = try {
                client.newCall(Request.Builder().url(url).build())
            } catch (_: Exception) {
                cont.resume(false); return@suspendCancellableCoroutine
            }
            cont.invokeOnCancellation { call.cancel() }
            call.enqueue(object : Callback {
                override fun onFailure(call: Call, e: IOException) { if (cont.isActive) cont.resume(false) }
                override fun onResponse(call: Call, response: Response) {
                    val ok = try {
                        response.use { r ->
                            val body = r.body
                            if (!r.isSuccessful || body == null) false
                            else { tmp.outputStream().use { out -> body.byteStream().copyTo(out) }; true }
                        }
                    } catch (_: Exception) { false }
                    if (cont.isActive) cont.resume(ok)
                }
            })
        }

    /**
     * 先 inSampleSize 粗降采样，再缩放到「刚好覆盖」目标尺寸（centerCrop 够用），控制内存。
     * 不透明 JPEG 用 RGB_565。失败返回 null。
     */
    fun decode(url: String, reqW: Int, reqH: Int): Bitmap? = synchronized(fileLock) {
        val f = fileFor(url)
        if (!f.isFile) return null
        try {
            val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
            BitmapFactory.decodeFile(f.path, bounds)
            if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null
            val opts = BitmapFactory.Options().apply {
                inSampleSize = sampleSize(bounds.outWidth, bounds.outHeight, reqW, reqH)
                if (bounds.outMimeType == "image/jpeg") inPreferredConfig = Bitmap.Config.RGB_565
            }
            val bmp = BitmapFactory.decodeFile(f.path, opts) ?: return null
            val scale = scaleToCover(bmp.width, bmp.height, reqW, reqH)
            if (scale < 0.95f) {
                val scaled = Bitmap.createScaledBitmap(bmp, (bmp.width * scale).toInt().coerceAtLeast(1), (bmp.height * scale).toInt().coerceAtLeast(1), true)
                if (scaled !== bmp) bmp.recycle()
                scaled
            } else bmp
        } catch (_: Throwable) { // 含 OOM
            null
        }
    }

    /** 删除不在 [keep] 内的素材文件（含残留 .tmp）。 */
    fun cleanup(keep: Set<String>) = synchronized(fileLock) {
        val keepNames = keep.map { sha256(it) }.toSet()
        dir.listFiles()?.forEach {
            if (it.name in activeTmp) return@forEach // 正在下载的不动
            if (it.name.substringBefore('.') !in keepNames) { verified.remove(it.name); it.delete() }
        }
    }

    companion object {
        /** 最大的 2 的幂 inSampleSize，使解码后仍 ≥ 目标尺寸。 */
        fun sampleSize(w: Int, h: Int, reqW: Int, reqH: Int): Int {
            var s = 1
            if (reqW <= 0 || reqH <= 0) return s
            while (w / (s * 2) >= reqW && h / (s * 2) >= reqH) s *= 2
            return s
        }

        /** 缩放到刚好覆盖 reqW×reqH 的比例（只缩小，≤1）。 */
        fun scaleToCover(w: Int, h: Int, reqW: Int, reqH: Int): Float {
            if (reqW <= 0 || reqH <= 0 || w <= 0 || h <= 0) return 1f
            return minOf(1f, maxOf(reqW / w.toFloat(), reqH / h.toFloat()))
        }

        /** 默认校验：图片头可解析出正的宽高。 */
        fun decodableBounds(f: File): Boolean = try {
            val o = BitmapFactory.Options().apply { inJustDecodeBounds = true }
            BitmapFactory.decodeFile(f.path, o)
            o.outWidth > 0 && o.outHeight > 0
        } catch (_: Throwable) { false }

        fun sha256(s: String): String =
            MessageDigest.getInstance("SHA-256").digest(s.toByteArray()).joinToString("") { "%02x".format(it) }
    }
}
