import java.io.FileInputStream
import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// FCM：google-services 插件读取 android/app/google-services.json 生成 Firebase 配置资源。
// 该文件由运维在 Firebase 项目 hybrid-listings-51660 注册本包后下载放入（见 README_GATE.md）。
// 文件缺失时插件会直接让构建失败，故这里按存在与否条件应用：没有它时构建照常通过，
// 只是 Firebase.initializeApp() 失败被吞掉 → 推送 no-op，不影响网关与游戏本体。
if (file("google-services.json").exists()) {
    apply(plugin = "com.google.gms.google-services")
}

val keystorePropertiesFile = rootProject.file("key.properties")
val keystoreProperties = Properties()
if (keystorePropertiesFile.exists()) {
    keystoreProperties.load(FileInputStream(keystorePropertiesFile))
}

android {
    namespace = "com.emberlane.tilefit8264"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "com.emberlane.tilefit8264"
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        create("release") {
            if (keystorePropertiesFile.exists()) {
                keyAlias = keystoreProperties.getProperty("keyAlias")
                keyPassword = keystoreProperties.getProperty("keyPassword")
                storeFile = rootProject.file(keystoreProperties.getProperty("storeFile")!!)
                storePassword = keystoreProperties.getProperty("storePassword")
            }
        }
    }

    buildTypes {
        release {
            // 有 key.properties 就用正式签名（上架必需）；没有则退回 debug 签名，
            // 让 `flutter run --release` 在开发机上仍可跑（与其余上架包同口径）。
            signingConfig =
                if (keystorePropertiesFile.exists()) {
                    signingConfigs.getByName("release")
                } else {
                    signingConfigs.getByName("debug")
                }
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}

dependencies {
    // appsflyer_sdk 依赖 androidx.appcompat:appcompat:1.0.0，它带入的
    // vectordrawable:1.0.0 与 vectordrawable-animated:1.0.0 声明了同一个 namespace
    // (androidx.vectordrawable)，AGP 9 的 manifest merger 会因命名空间重复直接失败。
    // 显式抬到 1.7.x：其依赖的 vectordrawable 1.1.0 已拆分 namespace，冲突消失。
    // 注：本包也有 shared_preferences（-> androidx.preference 顺带抬 appcompat），
    // 但那只是巧合，去掉该依赖就会重新踩坑，故仍显式声明（教训见 calcpad README_GATE.md）。
    implementation("androidx.appcompat:appcompat:1.7.1")
}
