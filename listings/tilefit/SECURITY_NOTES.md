# Security Notes — TileFit

> 本文是本包的内部安全备忘：哪些东西不能进仓库、哪些是非机密的、上架前对 release 产物
> 要做哪些自查。**面向内部**，不要粘进 Play Console。

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
  放进来之前必须**裁剪**成只留本包的 client 节点：Firebase 控制台下发的原始文件包含同项目下
  其余应用的包名，整份塞进来等于把几个上架包的关联关系直接写进 APK 里。
  当前 `build.gradle.kts` 里 google-services 插件是**按文件存在与否条件应用**的，
  缺文件时构建照常通过，只是 `Firebase.initializeApp()` 失败被吞掉 → 推送 no-op。
- `lib/gate/gate_config.dart` 里的 AppsFlyer devKey（`fXoKsKQwxPCRdhD8CD8q6F`，账号级）
  与 Adjust App Token（当前仍是占位符）—— 同样随包分发，非机密。

## 真正的机密（不在本包内，也不要放进来）

- Firebase service account 私钥 —— 服务端用，项目级。
- Adjust 个人 **API token** —— 与本包用的 App Token 完全不同，前者能读账号数据，
  不要写进代码、不要贴到聊天或工单里。
- 后台管理账号口令 —— 同上。

## 客户端不含线上落地页地址

`lib/gate/gate_config.dart` 是全工程唯一出现域名的地方，里面只有服务端 API 基址
（`apiBases`）。落地页 URL 由服务端在响应里下发，不落任何编译期常量。

上架前的自查（对 release 产物做，而不是对源码）：

```bash
flutter build appbundle --release
# 从 AAB 里取出 AOT snapshot，扫一遍字符串里出现的域名
unzip -p build/app/outputs/bundle/release/app-release.aab \
  base/lib/arm64-v8a/libapp.so | strings | grep -oE '[a-z0-9.-]+\.(com|net|online|io|app|dev)' | sort -u
```

应当只看到配置里的那个 API 域名，以及 Flutter / Firebase / AppsFlyer / Adjust 自带的文档与
端点域名。**出现任何品牌站点域名就是漏了。**

## 本地存储里有什么

`lib/storage/best_score_store.dart` 通过 `shared_preferences` 只写一个整数
（key = `tilefit_best_score`）。SharedPreferences 的 XML 在非 root 设备上其他应用读不到，
且这个值本身不敏感、不含任何标识符，因此不需要加密，也不需要在 Data safety 里申报
（纯本地数据）。**但隐私政策里要如实写**，见 `PRIVACY_POLICY.md`。

不要往这个 store 里加第二个 key 而不回来同步文档 —— 一旦存了任何与用户有关的东西
（标识符、行为记录），申报口径就变了。

## 申报与实现必须一致

`DATA_SAFETY.md` 与 `PRIVACY_POLICY.md` 是按**合并后** AndroidManifest 与 `pubspec.yaml`
的实际内容写的。已核对的合并结果（`build/app/intermediates/merged_manifests/release/`）
包含：`INTERNET`（源 manifest 唯一自己写的）、`ACCESS_NETWORK_STATE`、`POST_NOTIFICATIONS`、
`WAKE_LOCK`、`AD_ID`、`BIND_GET_INSTALL_REFERRER_SERVICE`、`c2dm.permission.RECEIVE`，
以及 Adjust 带入的 Huawei / Samsung 商店信息读取权限。**没有任何定位、存储、通讯录、
相机、麦克风权限。**

**改动依赖（尤其加/删 SDK）后必须回来同步这两份**，否则申报与实现不符 —— 这是应用被下架的
常见原因，而不是理论风险。

复核命令（装到设备后）：

```bash
adb shell dumpsys package com.emberlane.tilefit8264 | sed -n '/requested permissions/,/install permissions/p'
```

> 注：上面这条复核命令**还没在设备上跑过** —— 权限清单是从构建产物里的合并 manifest 读的，
> 两者理论上一致，但没做过交叉验证。
>
> 已完成的验证：`flutter analyze` 无问题、`flutter test` 50 项全过、
> `flutter build apk --release` 成功（48.5 MB，**debug 签名**）、release AOT 产物字符串扫描
> 未发现任何 B 面域名或其他上架包的 token（见 `README_GATE.md`），
> 以及 release APK 在 Android 模拟器上实跑通过（进 A 面不崩、落子计分正确、最高分能存住、
> 缺 `google-services.json` 时推送 no-op 且不影响本体、AppsFlyer 确实初始化）。
>
> 尚未在**真机**上跑过。
