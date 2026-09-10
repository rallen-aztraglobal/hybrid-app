# 上架内部说明（**不要粘进 Play Console**）

对外文档（`PLAY_STORE_LISTING.md` · `PRIVACY_POLICY.md` · `DATA_SAFETY.md` ·
`PLAY_CONSOLE_FORM_ANSWERS.md`）已清理成纯可粘贴文本。所有内部讨论集中在本文。

## 口径原则：准确但不铺开

对外答案只需为真，不需要主动交代实现细节。例如「是否按地理位置改变行为」，
答案是

> Yes. Some content shown in the app is loaded from our server, and that content
> can vary by region. The puzzles and the user interface are identical for all
> users. The app does not read device location; where regional differences apply
> they are determined server-side from the request.

每句都为真，也没有一句在描述网关的判定规则。

## 与另外五个新包的关系：口径必须一致，素材必须不同

本包与 nonopix / linkflow / minegrid / tickpad / checklane 是同一批、同一套 SDK、
同一套网关行为，**申报口径必须一致** —— 六个包答案不同才是问题。

但**对外可见的素材必须各自独立**，否则 Play 侧会把几个包公开关联起来：

| 项目 | 要求 |
| --- | --- |
| 包名 | 已分开（`oakmere` / `wrenfield` / `pinehollow` / `mossgate` / `larkspur` / `thornbury`） |
| 证书 `O=` | ✅ 已各取各的（本包 `O=Mossgate`；六个包分别是 Oakmere / Wrenfield / Pinehollow / Mossgate / Larkspur / Thornbury）|
| 隐私政策 URL | ✅ **已各建各的** —— 本包 `https://thriving-banoffee-396732.netlify.app/` |
| 支持邮箱 | ⚠️ **本批三个工具包共用** `toolsa069@gmail.com`（与 TickPad / CheckLane 同一个）——
见 `STORE_ASSETS.md` 与本文「邮箱共用的取舍」 |
| `google-services.json` | 必须裁剪成只留本包 client —— 见 `SECURITY_NOTES.md` |
| Adjust App Token | 必须新建 App，不可复用 —— 见 `README_GATE.md` |

> 关联性的实话：即便以上全部做到，几个包仍共用同一个 Firebase 项目
> （`hybrid-listings-51660`），因此 `google-services.json` 里的 `project_number` 与
> `api_key` 相同 —— 有人同时解压多个 APK 逐字段比对，依然能看出同属一个项目。
> 要彻底切断就得每个上架包一个独立 Firebase 项目，那是架构级改动，不在本包范围内。
> AppsFlyer devKey 是账号级的，同理。

## 邮箱共用的取舍

本包的支持邮箱是 `toolsa069@gmail.com`，与 UnitShift / TickPad / CheckLane **共用一个**。

这与本文上面那张表原先的口径（「每包另取」）相反，是**运营明确决定的**，不是漏填。
记在这里是为了别在下一次改文档时又按老口径改回去。

要清楚代价：**支持邮箱会公开显示在 Play 商店页**。三个包填同一个地址，
任何人对比三张商店页就能看出它们同属一家 —— 这正是各取独立厂商命名空间
（`mossgate` / `larkspur` / `thornbury`）、
各建独立 Adjust App、各用独立签名证书想避免的事。

换句话说：包名、证书 `O=`、Adjust token 这几层的隔离仍然成立，
**但邮箱这一层是主动放弃的**。隐私政策 URL 仍要求每包一个，别把它也合并了 ——
两层都合并的话，前面那些隔离基本就白做了。

> 跨组之间没有关联：游戏那三个用 `appaztra@outlook.com`，
> 工具那三个用 `toolsa069@gmail.com`，两组互不相干。

## 为什么没有照抄 colorstack 的对应文档

colorstack 的 `PLAY_CONSOLE_FORM_ANSWERS.md` 写着：不使用广告/分析 SDK、不使用
Firebase、pubspec 无任何第三方运行时依赖、行为不随地理位置变化；`PRIVACY_POLICY.md`
写着不使用任何第三方分析/广告/追踪技术。

这四条对本包与对现在的 colorstack 都不成立 —— 两者都装了 `appsflyer_sdk`、
`adjust_sdk`、`firebase_messaging`（colorstack 的 Adjust token `bytg13h7yubk` 是填好的），
且 AB 面网关本身就是按来源 IP 的国家改变下发内容。那些文档应是加网关之前写的、
之后未同步。

实务角度：Google 会扫 APK。申报「无第三方依赖」而包里带着 `AD_ID` 权限，是申报与实现
不符 —— 这本身就是招查的点。而诚实申报「用 AppsFlyer/Adjust 做归因」是最普通的一栏，
毫不敏感。因此对外文档按事实写，风险更低而不是更高。

本包的口径与 calcpad / hexacolorsort 对齐。

## 两处待定口径（不是漏填）

**1. Data safety 的 Approximate location**
App 不申请任何定位权限、读不到 GPS，但服务端会收到请求来源 IP 并据此判断国家。
Play 问的是「你的 app 收集或分享的数据」，含离开设备的数据。IP 本身不在 Play 的数据
类型清单里，但「由 IP 推得的粗略位置且用于改变行为」，保守做法是申报
Approximate location / Collected / App functionality。申报了要能自圆其说，不申报有被判
漏报的风险。**与 calcpad / hexacolorsort 是同一个待定项，所有包必须同口径。**

**2. 「是否按地理位置改变行为」**
本包与 calcpad、hexacolorsort、colorstack 的口径必须一致。当前口径以运营与法务的决定
为准，本文不复制具体答案。

## 商店名的取舍

`android:label` 与商店名都定为 `UnitShift`，没有用 `Unit Converter`。
那个词在 Play 上同名应用成千上万，搜索里排不上，且过于通用的名字容易被判为
误导性命名。`UnitShift` 与包名 `com.mossgate.unitshift4610` 一致。
若已被占，退到 `UnitShift — Unit Converter`。

## 提交前必须在 Console 侧确认的

Play 的内部测试轨道一旦发布，任何拿到链接的人都能装。**在把包提交给 Google 审核之前**，
先到渠道中台核对本包 listing 的网关配置。当前值以 Console 为准，本文不复制。

本包挂 **`ap`**（ArenaPlus），Console 里打开方式配**外开**。六个新包按 2/2/2 分到 ap / bp / gp。

## 取证依据（复核用）

装到设备后跑：

```bash
adb shell dumpsys package com.mossgate.unitshift4610 | sed -n '/requested permissions/,/install permissions/p'
```

实测输出（release APK 装在 Android 35 模拟器上，`google-services.json` 尚未放入）：

```
com.samsung.android.mapsagent.permission.READ_APP_INFO                  ← 三星商店安装来源
com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE  ← Play 安装来源
android.permission.POST_NOTIFICATIONS
com.google.android.c2dm.permission.RECEIVE                              ← FCM
android.permission.INTERNET                                             ← 源 manifest 里唯一写了的
com.huawei.appmarket.service.commondata.permission.GET_COMMON_DATA      ← 华为商店安装来源
android.permission.ACCESS_NETWORK_STATE
com.mossgate.unitshift4610.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION        ← androidx.core 的私有权限，非用户可见
com.google.android.gms.permission.AD_ID                                 ← AppsFlyer / Adjust 取广告 ID
android.permission.WAKE_LOCK
```

与 calcpad 逐条相同（同一套 SDK），故申报口径可以直接对齐。

注意：`c2dm.permission.RECEIVE` 与 `AD_ID` 这些权限是 SDK 自己的 manifest 注入的，
**与 `google-services.json` 在不在无关** —— 少了配置文件只是 Firebase 运行时初始化失败，
权限该有的照样有。所以这份清单不会因为后面补上配置文件而变。

改动依赖（尤其加/删 SDK）后必须重跑此命令并同步 `DATA_SAFETY.md` 与
`PRIVACY_POLICY.md` —— 申报与实现不符是下架的常见原因。

## 本包发布前还缺的

| 项 | 状态 |
| --- | --- |
| keystore + `key.properties` | ✅ 已生成（不进 git）。`apksigner verify` 已确认签的是本包证书 |
| `adjustAppToken` / `adjustOpenBLandingToken` | ✅ `qv7x4u8c7bi8` / `7pdbn6` |
| `android/app/google-services.json` | ✅ 已放入，已裁剪成只留 `com.mossgate.unitshift4610` 一条 client |
| 支持邮箱 | ✅ `toolsa069@gmail.com`（**注意与另两个工具包共用**）|
| 隐私政策 URL | ✅ `https://thriving-banoffee-396732.netlify.app/`（实测匿名 HTTP 200）|
| 品牌归属 | ✅ `ap`（ArenaPlus），打开方式**外开**，`bSideChromeColor` = `0xFFFFFFFF` |

IARC 内容分级问卷的提交页要勾选 IARC 条款，那是一份法律声明，**必须由能代表发行方的
人本人提交**。
