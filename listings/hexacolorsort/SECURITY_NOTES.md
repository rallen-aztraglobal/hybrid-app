# Security Notes — Hexa Color Sort

## 签名材料绝不进仓库
`android/key.properties`、`*.jks`、`*.keystore` 都在 `.gitignore` 里。本包目前**还没有**
release keystore —— `android/app/build.gradle.kts` 在缺 `key.properties` 时回退 debug 签名，
本地能跑通，但 **Play 拒收 debug 签名的包**，上架前必须先生成正式 keystore。

生成后照 `android/key.properties.example` 填 `android/key.properties`，keystore 文件放
`android/` 下（或放仓库外、CI 用 secret 注入）。

> ⚠️ 这把 key 丢了就再也无法更新这个应用（除非启用 Play App Signing 并走密钥重置流程）。
> 请离线备份 keystore 与口令，别只留在开发机上。

## 非机密、可进仓库的
- `android/app/google-services.json` —— 随 APK 分发，反编译即可得，非机密。
  但仓库里这份是**裁剪过的**（只留本包 client），原因见 `README_GATE.md` §3。
- `lib/gate/gate_config.dart` 里的 AppsFlyer devKey / Adjust token —— 同样随包分发。

## 真正的机密（不在本包内，也不要放进来）
- Firebase service account 私钥（`FIREBASE_SA_LISTINGS`）—— 服务端用，项目级，已配好。
- Adjust 个人 **API token** —— 与本包用的 App Token 完全不同，前者能读账号数据，
  不要写进代码、不要贴到聊天或工单里。

## 申报与实现必须一致
`DATA_SAFETY.md` 与 `PRIVACY_POLICY.md` 是按合并后 AndroidManifest 与 pubspec 的实际
内容写的。**改动依赖（尤其加/删 SDK）后必须回来同步这两份**，否则申报与实现不符 ——
这是应用被下架的常见原因，而不是理论风险。

复核命令（装到设备后）：
```bash
adb shell dumpsys package com.slatecove.hexasort4173 | sed -n '/requested permissions/,/install permissions/p'
```
