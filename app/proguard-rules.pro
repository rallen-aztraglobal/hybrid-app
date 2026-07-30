# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.
#
# For more details, see
#   http://developer.android.com/guide/developing/tools/proguard.html

# If your project uses WebView with JS, uncomment the following
# and specify the fully qualified class name to the JavaScript interface
# class:
#-keepclassmembers class fqcn.of.javascript.interface.for.webview {
#   public *;
#}

# Uncomment this to preserve the line number information for
# debugging stack traces.
#-keepattributes SourceFile,LineNumberTable

# If you keep the line number information, uncomment this to
# hide the original source file name.
#-renamesourcefileattribute SourceFile

-keep class com.appsflyer.** { *; }
-keep public class com.appsflyer.** { public protected *; }
-dontwarn com.appsflyer.**

# OAID
# sdk
-keep class com.bun.miitmdid.** { *; }
-keep interface com.bun.supplier.** { *; }
# asus
-keep class com.asus.msa.SupplementaryDID.** { *; }
-keep class com.asus.msa.sdid.** { *; }
# freeme
-keep class com.android.creator.** { *; }
-keep class com.android.msasdk.** { *; }
# huawei
-keep class com.huawei.hms.ads.** { *; }
-keep interface com.huawei.hms.ads.** {*; }
# lenovo
-keep class com.zui.deviceidservice.** { *; }
-keep class com.zui.opendeviceidlibrary.** { *; }
# meizu
-keep class com.meizu.flyme.openidsdk.** { *; }
# nubia
-keep class com.bun.miitmdid.provider.nubia.NubiaIdentityImpl { *; }
# oppo
-keep class com.heytap.openid.** { *; }
# samsung
-keep class com.samsung.android.deviceidservice.** { *; }
# vivo
-keep class com.vivo.identifier.** { *; }
# xiaomi
-keep class com.bun.miitmdid.provider.xiaomi.IdentifierManager { *; }
# zte
-keep class com.bun.lib.** { *; }
# coolpad
-keep class com.coolpad.deviceidsupport.** { *; }

# Adjust 归因（ADR-0013）
-keep class com.adjust.sdk.** { *; }
-keep class com.google.android.gms.common.ConnectionResult { int SUCCESS; }
-keep class com.google.android.gms.ads.identifier.AdvertisingIdClient { *; }
-keep class com.google.android.gms.ads.identifier.AdvertisingIdClient$Info { *; }
-keep public class com.android.installreferrer.** { *; }

# Adjust OAID 插件（adjust-android-oaid）对 MSA 移动安全联盟 SDK 的引用。
# 本工程【不打包】MSA：com.appsflyer:oaid 只含 com.appsflyer.oaid.*，靠反射调 MSA，
# 故上面 OAID 段那些 com.bun.miitmdid.** keep 规则一直是对不存在的类生效（no-op）。
# 但 Adjust 的插件是【直接类引用】且 AAR 里没带 consumer proguard 规则，R8 会以
# "Missing classes detected" 直接失败（AdjustOaid.readOaid / MsaSdkClient.getOaidInfo）。
# 运行时安全（已核对字节码）：readOaid 里 MdidSdkHelper.InitCert 的调用点在 catch Throwable
# 的异常表内，缺类走 isMsaSdkAvailable=false；Util 取 OAID 前先判该标志，MsaSdkClient
# 永不被加载。华为 OAID 走的是另一条 HmsSdkClient 路径，不受影响。
-dontwarn com.bun.miitmdid.**