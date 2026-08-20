# Release 签名 —— 生成 keystore（你自己执行，我不碰密钥）

本包目前**没有** keystore，release 包会回退 debug 签名，**Play 拒收**。上架前必须做完本文。

## 1. 生成 keystore

`keytool` 随 JDK 提供（Android Studio 自带的 JBR 里就有）。在 `listings/hexacolorsort/android/`
目录下执行，把 `<你的口令>` 换成自己的强口令：

```bash
keytool -genkeypair -v \
  -keystore hexasort4173.jks \
  -alias hexasort4173 \
  -keyalg RSA -keysize 2048 \
  -validity 10000 \
  -storepass '<你的口令>' -keypass '<你的口令>' \
  -dname "CN=Hexa Color Sort, OU=Mobile, O=<你们的公司名>, L=<城市>, C=<国家代码>"
```

- `-validity 10000`（约 27 年）是 Google 的建议下限：有效期必须覆盖 2033-10-22 之后。
- alias 和文件名用什么都行，但要和下一步填的一致。
- `-dname` 里的信息会写进证书，填真实公司信息。

## 2. 写 key.properties

在同一目录建 `key.properties`（照 `key.properties.example`）：

```properties
storePassword=<你的口令>
keyPassword=<你的口令>
keyAlias=hexasort4173
storeFile=hexasort4173.jks
```

`android/app/build.gradle.kts` 会自动读取它并用于 release 签名；文件不存在时才回退 debug。

## 3. 确认生效

```bash
flutter build apk --release
keytool -printcert -jarfile build/app/outputs/flutter-apk/app-release.apk
```

输出里的 Owner 应该是你填的 `-dname`，**而不是** `CN=Android Debug, O=Android, C=US`。
看到 Android Debug 就说明 `key.properties` 没被读到。

上架实际要传的是 AAB，不是 APK：

```bash
flutter build appbundle --release
```

产物在 `build/app/outputs/bundle/release/app-release.aab`。

## 4. 备份（最重要的一步）

`key.properties`、`*.jks`、`*.keystore` 都在 `.gitignore` 里，**不进 git**。

> ⚠️ **这把 key 丢了，这个应用就再也无法更新** —— 只能换包名重新上架，下载量和评价全部作废。
> 请把 keystore 文件与口令离线备份到至少两个地方（密码管理器 + 公司加密存储），
> 别只留在开发机上。

如果启用了 Play App Signing（新应用默认启用），你手里这把是 **upload key**，
丢了可以向 Google 申请重置；但 keystore 本身仍应妥善备份，重置流程要几天。

## 5. 交给 CI 时
不要把 keystore 提交上去。用 CI secret 注入 `key.properties` 的四个值，
keystore 文件用 base64 存 secret、构建时解码到 `android/` 下。
