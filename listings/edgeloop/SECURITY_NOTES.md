# 安全注意事项 — EdgeLoop

## 绝不进 git 的东西

| 类别 | 文件 | 说明 |
| --- | --- | --- |
| 签名密钥 | `android/edgeloop8451.jks` | release 签名密钥，泄露等于别人能签出被 Android 认作本 App 正式更新的包 |
| 签名口令 | `android/key.properties` | 含真实口令。仓库里只放 `key.properties.example`（占位符） |
| 本机配置 | `android/local.properties` | 含本机 SDK 路径 |
| Firebase 配置 | `android/app/google-services.json` | **已放入**，随 APK 分发、反编译即可得，非机密，进 git。**只留本包 client** |

`.gitignore` 已挡住前三类。提交前再确认一次（**要用 `-uall`**）：

```bash
git status --porcelain -uall listings/edgeloop \
  | grep -E '\.jks|key\.properties$|local\.properties$' && echo '停下！' || echo 'ok'
```

> 为什么强调 `-uall`：目录整体未跟踪时，普通 `git status` 只输出一行
> `?? listings/xxx/`，看不到文件级内容 —— 那样查等于没查。

## 可以进 git 的

- `lib/gate/gate_config.dart` 里的 AppsFlyer devKey 与 Adjust token —— 随包分发，
  反编译即可得，非机密（ADR-0013 的口径）。
- 网关 API 基址 —— 它只返回 A/B 判定，泄露它不等于泄露 B 面地址。

## 绝不能出现在包里的

- 任何 B 面域名或落地页地址
- 任何服务端凭据、数据库连接串、管理后台地址

自查（**阳性对照是必要的** —— 少了它，命令写错时会稳定返回 0，看起来像「干净」）：

```bash
unzip -p build/app/outputs/flutter-apk/app-release.apk \
  | grep -acE 'com\.northglade\.calcpad5170|com\.vividnest\.colorstack5821'
# 应为 0；把本包包名 com.bellcroft.edgeloop8451 代进去应 > 0
```

## google-services.json 的裁剪

放入前必须**只保留本包的 client**。Firebase 控制台下载的那份会把同一项目下
**所有** App 都列进来 —— 带着它出包，等于在 EdgeLoop 的 APK 里明写另外十几个上架包的
包名，正是各包使用互不相关的厂商命名空间（`bellcroft` 等）想避免的事。
google-services 插件只按 applicationId 取匹配的那条，多余条目会被忽略。

放入后自查：

```bash
grep -o '"package_name": "[^"]*"' android/app/google-services.json | sort -u
# 只应出现 com.bellcroft.edgeloop8451 一行
```

## 权限

源 manifest 里只声明了 `INTERNET`。合并后会多出一批由 SDK 注入的权限
（`AD_ID`、`BIND_GET_INSTALL_REFERRER_SERVICE`、`RECEIVE`(C2DM)、`POST_NOTIFICATIONS`、
`WAKE_LOCK`、`ACCESS_NETWORK_STATE`）。**Data safety 按合并后的清单申报，不是按源文件。**

实测：本包合并后的权限**共 10 条，与 calcpad 逐条相同**，唯一差异是各自包名前缀的
`DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION`（androidx.core 注入的私有权限，非用户可见）。

装好后核对实际权限：

```bash
adb shell dumpsys package com.bellcroft.edgeloop8451 \
  | sed -n '/requested permissions/,/install permissions/p'
```

没有相机、麦克风、定位、通讯录、外部存储 —— 出现任何一个都说明有依赖被换过，要查。

## 本地存档

> **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
> 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
> 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
> （该表单只管离开设备的数据）。

存的内容见 `PRIVACY_POLICY.md`。
