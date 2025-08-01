package com.hybrid.android.bridge

import android.os.Handler
import android.os.Looper
import android.webkit.JavascriptInterface
import android.webkit.WebView
import org.json.JSONObject
import java.util.concurrent.ConcurrentHashMap

class WebAppBridge(
    private val webView: WebView,
    private val listener: (url: String, fullRequestDataJson: String) -> Unit
) {

    private val mainHandler = Handler(Looper.getMainLooper())
    private val methodMap = ConcurrentHashMap<String, (Any?, ((Any?) -> Unit)?) -> Any?>()
    private var callbackId = 0

    /**
     * JS调用原生的统一接口
     */
    @JavascriptInterface
    fun call(method: String, jsonParams: String): String {
        return try {
            val paramsObj = JSONObject(jsonParams)
            val callbackStub = paramsObj.optString("_dscbstub", null)
            val data = paramsObj.opt("data")

            val methodHandler = methodMap[method]
            if (methodHandler == null) {
                JSONObject().put("data", JSONObject.NULL).toString()
            } else {
                if (!callbackStub.isNullOrEmpty()) {
                    // 异步调用，执行回调时直接调用JS函数
                    methodHandler(data) { callbackData ->
                        val callbackDataJson = JSONObject().put("data", callbackData ?: JSONObject.NULL).toString()
                        val js = "javascript:window.$callbackStub($callbackDataJson);"
                        mainHandler.post {
                            webView.evaluateJavascript(js, null)
                        }
                    }
                    // 先返回空结果给JS，后续通过回调异步返回
                    JSONObject().put("data", JSONObject.NULL).toString()
                } else {
                    // 同步调用，直接返回结果
                    val result = methodHandler(data, null)
                    JSONObject().put("data", result ?: JSONObject.NULL).toString()
                }
            }
        } catch (e: Exception) {
            e.printStackTrace()
            "{}"
        }
    }

    /**
     * JS注入的接口响应数据监听
     */
    @JavascriptInterface
    fun onApiResponse(dataJsonStr: String) {
        try {
            val json = JSONObject(dataJsonStr)
            val url = json.optString("url")
            if (url.isNotEmpty()) {
                listener(url, dataJsonStr)
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    /**
     * 注册JS调用的原生方法
     */
    fun registerHandler(
        methodName: String,
        handler: (data: Any?, callback: ((Any?) -> Unit)?) -> Any?
    ) {
        methodMap[methodName] = handler
    }
}