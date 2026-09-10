# Release signing — CheckLane

本包用**自己的** keystore，不与主工程（`app/`）或其余上架包共用。
一把 keystore 只签一个包 —— 丢了或泄露了，波及面就限在这一个 App 里。

## 现状

**keystore 已生成** ✅ —— `android/checklane2894.jks`（PKCS12 / RSA 2048 / 有效期到 2054-01），
alias `checklane2894`，证书主体 `CN=CheckLane, OU=Mobile, O=Thornbury, L=Unknown, ST=Unknown, C=US`。
`android/key.properties` 已按它填好。两者都被 `.gitignore` 挡住，**不进 git**。

> `O=Thornbury` 是本包**专用**的组织名 —— 证书主体谁都能看
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
出来的包照样能装能跑，只是 Play 拒收。所以每次出上架包都要跑一遍文末的核对命令。

## 生成

在 `listings/checklane/android/` 目录下执行，把 `<你的口令>` 换成自己的强口令：

```bash
keytool -genkeypair -v \
  -keystore checklane2894.jks \
  -alias checklane2894 \
  -keyalg RSA -keysize 2048 -validity 10000 \
  -storepass '<你的口令>' -keypass '<你的口令>' \
  -dname "CN=CheckLane, OU=Mobile, O=<你们的公司名>, L=<城市>, C=<国家代码>"
```

- `-validity 10000`（约 27 年）：Play 要求签名证书有效期至少到 2033 年之后。
- storepass 与 keypass 取同一个值，和其余上架包同口径，少一处能填错的地方。

## 配置

在 `listings/checklane/android/` 下新建 `key.properties`：

```properties
storePassword=<你的口令>
keyPassword=<你的口令>
keyAlias=checklane2894
storeFile=checklane2894.jks
```

`storeFile` 是**相对 `android/` 目录**的路径（`build.gradle.kts` 里用
`rootProject.file(...)` 解析）。

## 绝不进 git

`.gitignore` 已经挡了 `*.jks` 与 `key.properties`。提交前再确认一次：

```bash
git status --porcelain listings/checklane | grep -E '\.jks|key\.properties' && echo '停下！' || echo 'ok'
```

口令也不要写进任何 Markdown、issue、聊天记录。丢了 keystore 就没法给这个包发更新了 ——
Play 不接受换签名的更新（除非申请 key upgrade，很麻烦）。**离线备份一份。**

## 出包

```bash
cd listings/checklane
flutter build appbundle --release     # 上架用 .aab
flutter build apk --release           # 本地验证用 .apk
```

## 核对签名（上架前必做）

```bash
# 看 APK 到底是用哪把 key 签的
"$ANDROID_HOME/build-tools/$(ls "$ANDROID_HOME/build-tools" | tail -1)/apksigner" verify --print-certs \
  build/app/outputs/flutter-apk/app-release.apk
```

**已实测通过** — 本包当前 release APK：

```
V2 Signer: certificate DN: CN=CheckLane, OU=Mobile, O=Thornbury, L=Unknown, ST=Unknown, C=US
V2 Signer: certificate SHA-256 digest: 8a38da575edc70dcf62756e27210e4deee84dd8731d61032a90171d147c0d68e
```

字段名随 build-tools 版本变：新版打印 `V2 Signer: certificate DN`，老版是
`Signer #1 certificate DN` —— 认 `certificate DN` 这半截就行，它必须是上面 `-dname` 那串。

> 这串 SHA-256 就是本包的**签名身份**。以后任何一次出包指纹变了，都说明换了 key ——
> 那样的包 Play 会拒（除非申请 key upgrade）。首次上架后这个值就定死了。

**如果显示的是 `CN=Android Debug`，说明 `key.properties` 没被读到** —— 检查文件位置
和文件名，别把它放进了 `android/app/` 而不是 `android/`。

> 注意不能用 `keytool -printcert -jarfile` 查 —— 现代 APK 只有 V2/V3 签名，
> 那条命令会报 "Not a signed jar file"，容易误判成没签名。
