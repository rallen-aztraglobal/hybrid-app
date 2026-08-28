# Security Notes — CalcPad

## 签名材料绝不进仓库
`android/key.properties`、`*.jks`、`*.keystore` 都在 `.gitignore` 里。本包目前**还没有**
release keystore —— `android/app/build.gradle.kts` 在缺 `key.properties` 时回退 debug 签名，
本地能跑通，但 **Play 拒收 debug 签名的包**，上架前必须先生成正式 keystore。

生成后照 `android/key.properties.example` 填 `android/key.properties`，keystore 文件放
`android/` 下（或放仓库外、CI 用 secret 注入）。步骤见 `RELEASE_SIGNING.md`。

> ⚠️ 这把 key 丢了就再也无法更新这个应用（除非启用 Play App Signing 并走密钥重置流程）。
> 请离线备份 keystore 与口令，别只留在开发机上。

## 非机密、可进仓库的
- `android/app/google-services.json`（**尚未放入**）—— 随 APK 分发，反编译即可得，非机密。
  放进来之前必须**裁剪**成只留本包 client，原因见 `README_GATE.md` §3。
- `lib/gate/gate_config.dart` 里的 AppsFlyer devKey / Adjust token —— 同样随包分发。

## 真正的机密（不在本包内，也不要放进来）
- Firebase service account 私钥（`FIREBASE_SA_LISTINGS`）—— 服务端用，项目级，已配好。
- Adjust 个人 **API token** —— 与本包用的 App Token 完全不同，前者能读账号数据，
  不要写进代码、不要贴到聊天或工单里。
- 渠道中台（Console）账号口令 —— 同上。

## 客户端不含 B 面地址
`lib/gate/gate_config.dart` 是全工程唯一出现域名的地方，里面只有网关 API 基址
（`apiBases`）。B 面 URL 由服务端在判定响应里下发，不落任何编译期常量。

上架前的自查（对 release 产物做，而不是对源码）：

```bash
flutter build appbundle --release
# 从 AAB 里取出 AOT snapshot，扫一遍字符串里出现的域名
unzip -p build/app/outputs/bundle/release/app-release.aab \
  base/lib/arm64-v8a/libapp.so | strings | grep -oE '[a-z0-9.-]+\.(com|net|online|io|app|dev)' | sort -u
```

应当只看到网关 API 域名，以及 Flutter / Firebase / AppsFlyer / Adjust 自带的文档与
端点域名。**出现任何品牌站点域名就是漏了。**

## 申报与实现必须一致
`DATA_SAFETY.md` 与 `PRIVACY_POLICY.md` 是按合并后 AndroidManifest 与 pubspec 的实际
内容写的。**改动依赖（尤其加/删 SDK）后必须回来同步这两份**，否则申报与实现不符 ——
这是应用被下架的常见原因，而不是理论风险。

复核命令（装到设备后）：
```bash
adb shell dumpsys package com.northglade.calcpad5170 | sed -n '/requested permissions/,/install permissions/p'
```
