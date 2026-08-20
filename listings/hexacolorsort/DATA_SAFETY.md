# Play Console — Data safety form answers (Hexa Color Sort)

Play 的 Data safety 表单是最容易填错、也是最容易被查的地方（申报与实际不符会被下架）。
下面每一条都对应本包**实际集成的 SDK**，依据写在最后。

## Does your app collect or share any of the required user data types?
**Yes.**

## Is all of the user data collected by your app encrypted in transit?
**Yes** — 网关请求、token 注册、AppsFlyer / Adjust / FCM 全部走 HTTPS。

## Do you provide a way for users to request that their data be deleted?
**Yes**（隐私政策里给了联系邮箱作为删除请求入口）。填 Play 时需要给出该说明的 URL。

---

## Data types to declare

### Device or other IDs → **Collected + Shared**
- 内容：Google Advertising ID（GAID）、FCM registration token、install referrer
- Purpose：**Analytics**、**Advertising or marketing**、**App functionality**（推送）
- Is this data required? **Required**（SDK 在启动即初始化，用户不可关闭归因）
- 依据：`appsflyer_sdk`、`adjust_sdk`、`firebase_messaging`；合并 manifest 含
  `com.google.android.gms.permission.AD_ID` 与
  `com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE`

### App activity → App interactions → **Collected + Shared**
- 内容：app 打开 / 会话事件、以及进入配置内容时上报的自定义事件
  （AF 标准事件 `af_content_view`、自定义事件 `OpenBLanding`）
- Purpose：**Analytics**、**Advertising or marketing**
- Required
- 依据：`lib/tracking/tracking_service.dart`

### App info and performance → Other app performance data → **Collected**
- 内容：设备型号与 OS 版本（随 token 注册上报，字段 `model`）
- Purpose：**App functionality**、**Analytics**
- Required
- 依据：`lib/push/push_service.dart` 的 `Platform.operatingSystemVersion`

### Location → Approximate location → **需要你决定，见下**
- App **没有**申请任何 Android 定位权限，也读不到 GPS。
- 但服务端会用请求的来源 IP 推出国家，并据此决定下发什么配置。
- Play 的 Data safety 问的是「你的 app 收集或分享的数据」，包含**离开设备**的数据。
  IP 本身不在 Play 的数据类型清单里，但「由 IP 推得的粗略位置且用于改变行为」，
  保守做法是申报 **Approximate location / Collected / App functionality**。
- 这一条我不替你拍：申报了要能自圆其说，不申报又有被判定为漏报的风险。
  建议按保守口径申报，或先问一下法务/渠道那边现在的统一口径。

### 明确**不**申报（App 确实不收）
Name、Email、Phone、Address、Photos、Videos、Files、Contacts、Calendar、
Precise location、Health、Financial info、Messages、Audio、Web browsing history、
Installed apps 列表。

---

## Data types NOT collected（本地存储，不出设备）
最高分与「音效 / 振动」开关存在 `shared_preferences`（Android app 私有存储），
不上传、不共享，卸载即删。Play 的 Data safety **不需要**申报纯本地数据。

---

## 事实依据（复核用）
实机 `adb shell dumpsys package com.slatecove.hexasort4173` 的 requested permissions：

```
android.permission.INTERNET
android.permission.ACCESS_NETWORK_STATE
android.permission.POST_NOTIFICATIONS
android.permission.WAKE_LOCK
com.google.android.c2dm.permission.RECEIVE
com.google.android.gms.permission.AD_ID
com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE
com.samsung.android.mapsagent.permission.READ_APP_INFO
com.huawei.appmarket.service.commondata.permission.GET_COMMON_DATA
```

后三条是 AppsFlyer / Adjust 为读取各应用商店安装来源加入的 —— 它们本身就说明
「本包做归因」，与「无任何分析 SDK」的说法直接矛盾。
