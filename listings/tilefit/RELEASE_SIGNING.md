# Release 签名 —— 生成 keystore（你自己执行，我不碰密钥）

> 本文是本包出正式签名包的操作手册：生成 keystore、写 `key.properties`、验签、备份。
> 本包目前**没有** keystore，release 包会回退 debug 签名，**Play 拒收**。上架前必须做完本文。

## 0. 现状

`android/app/build.gradle.kts` 里的判定是「`android/key.properties` 存在就用正式签名，
不存在就退回 debug 签名」。仓库里现在**没有** `key.properties`，也没有任何 `.jks`，
所以已经跑通的那次 `flutter build apk --release`（48.5 MB）产出的是 **debug 签名的包**，
本地能装能跑，但**不能上架**。

## 1. 生成 keystore

`keytool` 随 JDK 提供（Android Studio 自带的 JBR 里就有）。在
`listings/tilefit/android/` 目录下执行，把 `<你的口令>` 换成自己的强口令：

```bash
keytool -genkeypair -v \
  -keystore tilefit8264.jks \
  -alias tilefit8264 \
  -keyalg RSA -keysize 2048 \
  -validity 10000 \
  -storepass '<你的口令>' -keypass '<你的口令>' \
  -dname "CN=TileFit, OU=Mobile, O=<你们的公司名>, L=<城市>, C=<国家代码>"
```

- `-validity 10000`（约 27 年）是 Google 的建议下限：有效期必须覆盖 2033-10-22 之后。
- alias 和文件名用什么都行，但要和下一步填的一致。
- `-dname` 里的信息会写进证书，填真实信息。
- **`O=` 不要填成和其余上架包相同的组织名。** 证书主体是公开可见的
  （`apksigner verify --print-certs` 谁都能跑），几个包用同一个 `O=` 等于自己把它们串起来，
  与本包单独取 `emberlane` 命名空间的初衷相冲突。本包另取一个。

## 2. 写 key.properties

在同一目录建 `key.properties`（照 `key.properties.example`）：

```properties
storePassword=<你的口令>
keyPassword=<你的口令>
keyAlias=tilefit8264
storeFile=tilefit8264.jks
```

`android/app/build.gradle.kts` 会自动读取它并用于 release 签名；文件不存在时才回退 debug。

> 填的时候注意别把 `key.properties.example` 里的占位符
> （`REPLACE_WITH_STORE_PASSWORD` 之类）留在里面。留着的话构建会报
> `keystore password was incorrect` —— 那不是 keystore 坏了，是口令没换。

## 3. 确认生效

```bash
flutter build apk --release
```

然后验签。**不要用 `keytool -printcert -jarfile`** —— 现代 APK 只带 APK Signature
Scheme v2/v3，没有 v1 的 `META-INF/*.RSA`，keytool 会报 `Not a signed jar file`。
用 SDK 里的 `apksigner`：

```bash
"$ANDROID_HOME/build-tools/36.0.0/apksigner" verify --print-certs \
  build/app/outputs/flutter-apk/app-release.apk
```

输出里的 Signer #1 certificate DN 应该是你填的 `-dname`，**而不是**
`CN=Android Debug, O=Android, C=US`。看到 Android Debug 就说明 `key.properties` 没被读到
（现在的产物就是这种情况）。

上架实际要传的是 AAB，不是 APK：

```bash
flutter build appbundle --release
```

产物在 `build/app/outputs/bundle/release/app-release.aab`。

> 参考量级：当前 debug 签名的 release APK 是 48.5 MB。换成正式签名不会明显改变体积；
> AAB 会比 APK 小一些，Play 分发给单一 ABI 的设备时更小。体积出现数量级变化就要查依赖。

## 4. 备份（最重要的一步）

`key.properties`、`*.jks`、`*.keystore` 都在 `.gitignore` 里，**不进 git**。
keystore 文件请放在 `android/` 下（不是包根目录）—— `.gitignore` 里的
`*.jks` / `*.keystore` 两条在任何层级都生效，但 `android/key.properties` 那条是带路径的，
放错地方就不被忽略了。

> ⚠️ **这把 key 丢了，这个应用就再也无法更新** —— 只能换包名重新上架，下载量和评价全部作废。
> 请把 keystore 文件与口令离线备份到至少两个地方（密码管理器 + 公司加密存储），
> 别只留在开发机上。

如果启用了 Play App Signing（新应用默认启用），你手里这把是 **upload key**，
丢了可以向 Google 申请重置；但 keystore 本身仍应妥善备份，重置流程要几天。

## 5. 交给 CI 时

不要把 keystore 提交上去。用 CI secret 注入 `key.properties` 的四个值，
keystore 文件用 base64 存 secret、构建时解码到 `android/` 下。
