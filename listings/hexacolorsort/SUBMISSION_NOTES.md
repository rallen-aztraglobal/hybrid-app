# 上架内部说明（**不要粘进 Play Console**）

对外文档（`PLAY_STORE_LISTING.md` · `PRIVACY_POLICY.md` · `DATA_SAFETY.md` ·
`PLAY_CONSOLE_FORM_ANSWERS.md`）已清理成纯可粘贴文本。所有内部讨论集中在本文。

## 口径原则：准确但不铺开
对外答案只需为真，不需要主动交代实现细节。例如「是否按地理位置改变行为」，答案是

> Yes. Some content shown in the app is loaded from our server, and that content
> can vary by region. Gameplay, scoring and the user interface are identical for
> all users. The app does not read device location; where regional differences
> apply they are determined server-side from the request.

每句都为真，也没有一句在描述网关的判定规则。

## 为什么没有照抄 colorstack 的对应文档
colorstack 的 `PLAY_CONSOLE_FORM_ANSWERS.md` 写着：不使用广告/分析 SDK、不使用
Firebase、pubspec 无任何第三方运行时依赖、行为不随地理位置变化；`PRIVACY_POLICY.md`
写着不使用任何第三方分析/广告/追踪技术。

这四条对本包与对现在的 colorstack 都不成立 —— 两者都装了 `appsflyer_sdk`、
`adjust_sdk`、`firebase_messaging`，且 AB 面网关本身就是按来源 IP 的国家改变下发内容
（`docs/admin/09-listing.md` / ADR-0014）。那些文档应是加网关之前写的、之后未同步。

实务角度：Google 会扫 APK。申报「无第三方依赖」而包里带着 `AD_ID` 权限，是申报与实现
不符 —— 这本身就是招查的点。而诚实申报「用 AppsFlyer/Adjust 做归因」是手游最普通的一栏，
毫不敏感。因此对外文档按事实写，风险更低而不是更高。

> colorstack 是**已上架**应用，那两份失效声明现在就挂在 Play 上，暴露比尚未提交的本包更实际。
> 是否同步是运营决定，本包未越界处理。

## 两处待定口径（不是漏填）

**1. Data safety 的 Approximate location**
App 不申请任何定位权限、读不到 GPS，但服务端会收到请求来源 IP 并据此判断国家。
Play 问的是「你的 app 收集或分享的数据」，含离开设备的数据。IP 本身不在 Play 的数据
类型清单里，但「由 IP 推得的粗略位置且用于改变行为」，保守做法是申报
Approximate location / Collected / App functionality。申报了要能自圆其说，不申报有被判
漏报的风险 —— 建议先对齐法务或渠道现有口径。

**2. 「是否按地理位置改变行为」**
按事实是 Yes（措辞见上）。colorstack 填的是 No。明知不符仍填 No 属于向 Google 作虚假
陈述，被判定为规避审核的后果是下架并可能连带封号。这一条属运营与法务的判断。

## 取证依据（复核用）
实机 `adb shell dumpsys package com.slatecove.hexasort4173` 的 requested permissions：

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
```

后四条本身就说明本包做归因。改动依赖（尤其加/删 SDK）后必须重跑此命令并同步
`DATA_SAFETY.md` 与 `PRIVACY_POLICY.md` —— 申报与实现不符是下架的常见原因。
