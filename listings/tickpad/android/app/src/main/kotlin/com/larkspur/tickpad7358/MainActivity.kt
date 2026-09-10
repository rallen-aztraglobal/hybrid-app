package com.larkspur.tickpad7358

import android.view.KeyEvent
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

/**
 * 除了标准的 FlutterActivity，这里只多做一件事：**把音量键转成计数事件**。
 *
 * 为什么值得为它写原生代码：计数器最常见的用法是手上在忙、眼睛不看屏幕
 * （盘点、点人数、观鸟）。有实体键就不用摸索屏幕上的位置，闭着眼按都不会错。
 * 这也是这一类 App 里公认的必备功能。
 *
 * 开关由 Dart 侧控制（默认关）。关着的时候一律走 super，音量键就是音量键 ——
 * 接管系统按键是件霸道的事，不该在用户没要求时发生。
 *
 * onKeyUp 也要拦：只拦 down 的话，系统仍会在抬手时弹出音量条。
 */
class MainActivity : FlutterActivity() {

    private var channel: MethodChannel? = null

    /** Dart 侧的开关。false 时本类完全不介入按键。 */
    private var countingEnabled = false

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        channel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL).apply {
            setMethodCallHandler { call, result ->
                when (call.method) {
                    "setEnabled" -> {
                        countingEnabled = call.arguments as? Boolean ?: false
                        result.success(null)
                    }
                    else -> result.notImplemented()
                }
            }
        }
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent): Boolean {
        if (countingEnabled) {
            when (keyCode) {
                KeyEvent.KEYCODE_VOLUME_UP -> {
                    channel?.invokeMethod("count", "up")
                    return true
                }
                KeyEvent.KEYCODE_VOLUME_DOWN -> {
                    channel?.invokeMethod("count", "down")
                    return true
                }
            }
        }
        return super.onKeyDown(keyCode, event)
    }

    override fun onKeyUp(keyCode: Int, event: KeyEvent): Boolean {
        if (countingEnabled &&
            (keyCode == KeyEvent.KEYCODE_VOLUME_UP || keyCode == KeyEvent.KEYCODE_VOLUME_DOWN)
        ) {
            return true
        }
        return super.onKeyUp(keyCode, event)
    }

    private companion object {
        const val CHANNEL = "tickpad/volume_keys"
    }
}
