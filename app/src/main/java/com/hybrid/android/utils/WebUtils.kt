package com.hybrid.android.utils

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.util.Log
import androidx.core.net.toUri
import java.time.LocalDate
import java.time.format.DateTimeFormatter

object WebUtils {
    fun openInExternalBrowser(context: Context, url: String) {
        try {
            val intent = Intent(Intent.ACTION_VIEW, url.toUri()).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
            context.startActivity(intent)
        } catch (e: Exception) {
            Log.e("WebUtils", "Failed to open browser: $e")
        }
    }
}

