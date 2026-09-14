# Release signing — RepTimer

本包用**自己的** keystore，不与主工程（`app/`）或其余上架包共用。
一把 keystore 只签一个包 —— 丢了或泄露了，波及面就限在这一个 App 里。

## 现状

**keystore 已生成** ✅ —— `android/reptimer7719.jks`（PKCS12 / RSA 2048 / 有效期到 2054-01），
alias `reptimer7719`，证书主体
`CN=RepTimer, OU=Mobile, O=Quillford, L=Unknown, ST=Unknown, C=US`。
`android/key.properties` 已按它填好。两者都被 `.gitignore` 挡住，**不进 git**。

> `O=Quillford` 是本包**专用**的组织名 —— 证书主体谁都能看
> （`apksigner verify --print-certs`），几个上架包共用一个 `O=` 等于自己把它们串起来。

仍要留意 `android/app/build.gradle.kts` 里的这段：

```kotlin
signingConfig = if (keystorePropertiesFile.exists()) {
    signingConfigs.getByName("release")  // ← 现在走的是这条
} else {
    signingConfigs.getByName("debug")
}
```

这个回退是有意的：让没有 keystore 的人也能在本机跑 release 包。
**代价是它不会报错** —— `key.properties` 一旦丢失或改名，构建会**静默**退回 debug 签名，
出来的包照样能装能跑，只是 Play 拒收。所以每次出上架包都要跑一遍下面的核对命令。

## 重新生成（换机/重建时参考）

在 `listings/reptimer/android/` 下执行：

```bash
keytool -genkeypair -v \
  -keystore reptimer7719.jks -alias reptimer7719 \
  -keyalg RSA -keysize 2048 -validity 10000 \
  -storepass '<你的口令>' -keypass '<你的口令>' \
  -dname "CN=RepTimer, OU=Mobile, O=Quillford, L=Unknown, ST=Unknown, C=US"
```

- `-validity 10000`（约 27 年）：Play 要求签名证书有效期覆盖 2033-10-22 之后。
- storepass 与 keypass 取同一个值，与其余上架包同口径，少一处能填错的地方。

## 配置

`android/key.properties`（照 `key.properties.example` 填）：

```properties
storePassword=<你的口令>
keyPassword=<你的口令>
keyAlias=reptimer7719
storeFile=reptimer7719.jks
```

`storeFile` 是**相对 `android/` 目录**的路径。

## 核对签名（上架前必做）

```bash
# 别写死 build-tools 版本，取最新的一个
AS=$(ls -d "$ANDROID_HOME"/build-tools/*/ | tail -1)
"$AS/apksigner" verify --print-certs build/app/outputs/flutter-apk/app-release.apk
```

**已实测通过** — 本包当前 release APK：

```
V2 Signer: certificate DN: CN=RepTimer, OU=Mobile, O=Quillford, L=Unknown, ST=Unknown, C=US
V2 Signer: certificate SHA-256 digest: b3a68e8ae0af2e7c3327e4f22d8345639695c0ef2afd2149dfde25580dc8845f
```

字段名随 build-tools 版本变：新版打印 `V2 Signer: certificate DN`，老版是
`Signer #1 certificate DN` —— 认 `certificate DN` 这半截就行。
**如果显示的是 `CN=Android Debug`，说明 `key.properties` 没被读到** —— 检查文件位置
和文件名，别把它放进了 `android/app/` 而不是 `android/`。

> 这串 SHA-256 就是本包的**签名身份**。以后任何一次出包指纹变了，都说明换了 key ——
> 那样的包 Play 会拒（除非申请 key upgrade）。首次上架后这个值就定死了。

> 注意不能用 `keytool -printcert -jarfile` 查 —— 现代 APK 只有 V2/V3 签名，
> 那条命令会报 "Not a signed jar file"，容易误判成没签名。

## 出包

```bash
flutter build appbundle --release     # 上架用 .aab
flutter build apk --release           # 本地验证用 .apk（52.9MB）
```
