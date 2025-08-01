// 文件：DateExt.kt
package com.hybrid.android.utils

import java.time.LocalDate
import java.time.format.DateTimeFormatter
fun String.toLocalDateOrNull(): LocalDate? {
    val datePart = this.take(10)  // "yyyy-MM-dd"
    return try {
        LocalDate.parse(datePart, DateTimeFormatter.ISO_LOCAL_DATE)
    } catch (e: Exception) {
        null
    }
}