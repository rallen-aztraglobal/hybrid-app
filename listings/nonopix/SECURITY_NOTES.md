# Security Notes — NonoPix

## 绝不进 git 的东西

| 类型 | 例子 | 现状 |
| --- | --- | --- |
| 签名密钥 | `*.jks`、`*.keystore` | `.gitignore` 已挡 |
| 签名口令 | `android/key.properties` | `.gitignore` 已挡 |
| Firebase 配置 | `android/app/google-services.json` | **已放入**，随 APK 分发、反编译即可得，非机密，进 git（与 calcpad 同口径）。**只留本包 client** |
| Apple 相关 | `*.p8`、`*.p12`、`*.mobileprovision` | 本包不发 iOS，不涉及 |

口令也不要出现在 Markdown、commit message、issue 或聊天记录里。

## 包里有什么、没有什么

**编译期烧录的（非机密，可以进 git）**

- 网关 API 基址 —— 它只是一个返回 A/B 的接口。**泄露它不等于泄露 B 面**：
  B 面 URL 由服务端在判定为 B 时下发，客户端从不内置，静态扫描 APK 也扫不到。
  即便该 API 被封，客户端拿不到判定只会**回退 A 面**（fail-closed）。
- AppsFlyer devKey —— AF 按账号发，随包分发，本来就是要出现在客户端里的。
- Adjust App Token —— 同理。目前仍是 `TODO_` 占位。

**不在包里的**

- 任何 B 面域名或落地页地址
- 任何服务端凭据、数据库连接串、管理后台地址

## google-services.json 的裁剪

放入前必须**只保留本包的 client**。Firebase 控制台下载的那份可能包含同一个项目下
其他 App 的 `client` 条目 —— 带着它出包，等于在 NonoPix 的 APK 里明写另外几个上架包的
包名，正是这批包用不相关的厂商前缀（`oakmere` / `wrenfield` / `pinehollow` /
`mossgate` / `larkspur` / `thornbury`）想避免的事。

放入后自查（已跑过，✅ 只有一行）：

```bash
grep -o '"package_name": "[^"]*"' android/app/google-services.json | sort -u
# 只应出现 com.oakmere.nonopix3927 一行
```

更硬的一条是**扫成品 APK**，因为它连插件生成的资源一起查：

```bash
unzip -p build/app/outputs/flutter-apk/app-release.apk \
  | grep -acE 'com.northglade.calcpad5170|com.vividnest.colorstack5821|com.slatecove.hexasort4173'
# 应为 0；把本包包名代进去应 > 0（阳性对照，证明这条命令真的读到了包）
```

> 阳性对照那半句不是多余的：少了它，命令写错时会稳定返回 0，看起来像「干净」。
> 实测本包 release APK：其他上架包包名 0 处，本包包名 4 处。

## 权限

源 manifest 里只声明了 `INTERNET`。合并后会多出一批由 SDK 注入的权限
（`AD_ID`、`BIND_GET_INSTALL_REFERRER_SERVICE`、`RECEIVE`(C2DM)、`POST_NOTIFICATIONS`、
`WAKE_LOCK`、`ACCESS_NETWORK_STATE`）。**Data safety 按合并后的清单申报，不是按源文件。**

装好后核对实际权限：

```bash
adb shell dumpsys package com.oakmere.nonopix3927 \
  | sed -n '/requested permissions/,/install permissions/p'
```

没有相机、麦克风、定位、通讯录、外部存储 —— 出现任何一个都说明有依赖被换过，要查。

## 本地存档

`lib/storage/progress_store.dart` 用 SharedPreferences 存两个整数
（`nonopix_solved_small` / `nonopix_solved_large`）。非 root 设备上其他应用读不到，
卸载即删。不含任何可识别用户的内容，也不上报。

## APK 自查（打完包顺手做）

```bash
# 1. 包里不该出现任何 B 面域名或其他上架包的包名
unzip -p build/app/outputs/flutter-apk/app-release.apk assets/flutter_assets/kernel_blob.bin 2>/dev/null \
  | strings | grep -Ei 'vividnest|slatecove|northglade|emberlane|wrenfield|pinehollow|mossgate|larkspur|thornbury' \
  | grep -v oakmere || echo 'ok：没有别的包名'

# 2. 签名是正式签名而不是 debug（见 RELEASE_SIGNING.md）
```

## 一条务实的提醒

这些包的 A 面必须是**真能用、审核挑不出毛病**的 App。任何为了过审而在文档或商店文案里
写下与实现不符的话（"不收集任何数据"、"完全离线"），风险都远高于收益 ——
Data safety 与隐私政策的口径必须和 `pubspec.yaml` 里的依赖对得上。
