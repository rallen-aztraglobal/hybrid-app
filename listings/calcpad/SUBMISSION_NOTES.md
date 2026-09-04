# 上架内部说明（**不要粘进 Play Console**）

对外文档（`PLAY_STORE_LISTING.md` · `PRIVACY_POLICY.md` · `DATA_SAFETY.md` ·
`PLAY_CONSOLE_FORM_ANSWERS.md`）已清理成纯可粘贴文本。所有内部讨论集中在本文。

## 口径原则：准确但不铺开
对外答案只需为真，不需要主动交代实现细节。例如「是否按地理位置改变行为」，答案是

> Yes. Some content shown in the app is loaded from our server, and that content
> can vary by region. The calculator itself and the user interface are identical
> for all users. The app does not read device location; where regional differences
> apply they are determined server-side from the request.

每句都为真，也没有一句在描述网关的判定规则。

## 与 hexacolorsort 的关系：口径必须一致，素材必须不同

本包的对外文档是照 hexacolorsort 的口径写的，两包的申报应保持一致（同一套 SDK、
同一套网关行为，答案不同才是问题）。

但**对外可见的素材必须各自独立**，否则 Play 侧会把两包公开关联起来：

| 项目 | 要求 |
| --- | --- |
| 包名 | 已分开（`com.northglade.*` vs `com.slatecove.*`） |
| 证书 `O=` | **必须另取**（hexa 是 `O=SlateCove`）—— 见 `RELEASE_SIGNING.md` |
| 隐私政策 URL | **必须另建站点** —— 见 `STORE_ASSETS.md` |
| 支持邮箱 | **必须另取** —— 见 `STORE_ASSETS.md` |
| google-services.json | 必须裁剪成只留本包 client —— 见 `README_GATE.md` §3 |
| Adjust App Token | 必须新建 App，不可复用 —— 见 `README_GATE.md` §2 |

> 关联性的实话：即便以上全部做到，两包仍共用同一个 Firebase 项目
> （`hybrid-listings-51660`），因此 `google-services.json` 里的 `project_number` 与
> `api_key` 相同 —— 有人同时解压两个 APK 逐字段比对，依然能看出同属一个项目。
> 要彻底切断就得每个上架包一个独立 Firebase 项目，但服务端目前是单一路由键
> （`fcmRouteKeyListings` 是常量、一个 service account 管整个项目），那是架构级改动，
> 不在本包范围内。AppsFlyer devKey 是账号级的，同理。

## 为什么没有照抄 colorstack 的对应文档
colorstack 的 `PLAY_CONSOLE_FORM_ANSWERS.md` 写着：不使用广告/分析 SDK、不使用
Firebase、pubspec 无任何第三方运行时依赖、行为不随地理位置变化；`PRIVACY_POLICY.md`
写着不使用任何第三方分析/广告/追踪技术。

这四条对本包与对现在的 colorstack 都不成立 —— 两者都装了 `appsflyer_sdk`、
`adjust_sdk`、`firebase_messaging`，且 AB 面网关本身就是按来源 IP 的国家改变下发内容
（`docs/admin/09-listing.md` / ADR-0014）。那些文档应是加网关之前写的、之后未同步。

实务角度：Google 会扫 APK。申报「无第三方依赖」而包里带着 `AD_ID` 权限，是申报与实现
不符 —— 这本身就是招查的点。而诚实申报「用 AppsFlyer/Adjust 做归因」是最普通的一栏，
毫不敏感。因此对外文档按事实写，风险更低而不是更高。

## 两处待定口径（不是漏填）

**1. Data safety 的 Approximate location**
App 不申请任何定位权限、读不到 GPS，但服务端会收到请求来源 IP 并据此判断国家。
Play 问的是「你的 app 收集或分享的数据」，含离开设备的数据。IP 本身不在 Play 的数据
类型清单里，但「由 IP 推得的粗略位置且用于改变行为」，保守做法是申报
Approximate location / Collected / App functionality。申报了要能自圆其说，不申报有被判
漏报的风险 —— 建议先对齐法务或渠道现有口径。**与 hexacolorsort 是同一个待定项，
两包必须同口径。**

**2. 「是否按地理位置改变行为」**
本包与 colorstack、hexacolorsort 的口径必须一致。当前口径以运营与法务的决定为准，
本文不复制具体答案。

## 商店名的取舍
`android:label` 与商店名都定为 `CalcPad`，没有用 `Calculator`。
`Calculator` 在 Play 上有成千上万个同名应用，搜索里根本排不上，且过于通用的名字
容易被判为误导性命名。`CalcPad` 与包名 `com.northglade.calcpad5170` 一致。
若 `CalcPad` 已被占，退到 `CalcPad — Calculator`。

## 提交前必须在 Console 侧确认的
Play 的内部测试轨道一旦发布，任何拿到链接的人都能装。**在把包提交给 Google 审核之前**，
先到渠道中台核对本包 listing 的网关配置。当前值以 Console 为准，本文不复制。

## 取证依据（复核用）
装到设备后跑：

```bash
adb shell dumpsys package com.northglade.calcpad5170 | sed -n '/requested permissions/,/install permissions/p'
```

实测输出（release APK 装在 Android 35 模拟器上，`google-services.json` 尚未放入）：

```
android.permission.INTERNET
android.permission.ACCESS_NETWORK_STATE
android.permission.POST_NOTIFICATIONS
android.permission.WAKE_LOCK
com.google.android.c2dm.permission.RECEIVE                              ← FCM
com.google.android.gms.permission.AD_ID                                 ← AppsFlyer / Adjust 取广告 ID
com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE  ← Play 安装来源
com.samsung.android.mapsagent.permission.READ_APP_INFO                  ← 三星商店安装来源
com.huawei.appmarket.service.commondata.permission.GET_COMMON_DATA      ← 华为商店安装来源
com.northglade.calcpad5170.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION     ← androidx.core 自定义的私有权限，非用户可见
```

与 hexacolorsort 逐条相同（同一套 SDK），故 `DATA_SAFETY.md` 与 `PRIVACY_POLICY.md`
的申报口径可以对齐。

注意：`c2dm.permission.RECEIVE` 与 `AD_ID` 这些权限是 SDK 自己的 manifest 注入的，
**与 `google-services.json` 在不在无关** —— 少了配置文件只是 Firebase 运行时初始化失败，
权限该有的照样有。所以这份清单不会因为后面补上配置文件而变。

改动依赖（尤其加/删 SDK）后必须重跑此命令并同步 `DATA_SAFETY.md` 与
`PRIVACY_POLICY.md` —— 申报与实现不符是下架的常见原因。
