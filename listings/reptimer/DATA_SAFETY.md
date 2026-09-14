# Play Data safety 表单 —— RepTimer

> 逐项答案，直接照填。**口径必须与其余上架包一致** —— 同一批、同一套 SDK、
> 同一套网关行为，几个包答案不同才是问题。

## 总览

| 问题 | 答案 |
| --- | --- |
| Does your app collect or share any of the required user data types? | **Yes** |
| Is all of the user data collected by your app encrypted in transit? | **Yes**（全部 HTTPS） |
| Do you provide a way for users to request that their data be deleted? | **Yes**（隐私政策里的联系邮箱） |

## 数据类型

### Device or other IDs — **Collected + Shared**

| 字段 | 值 |
| --- | --- |
| Collected | ✅ |
| Shared | ✅（AppsFlyer / Adjust / Google） |
| Processed ephemerally | ❌ |
| Required or optional | **Required** |
| Purposes | Analytics · Advertising or marketing |

来源：AppsFlyer 与 Adjust 取 Google Advertising ID（包里声明了
`com.google.android.gms.permission.AD_ID`），Firebase Cloud Messaging 取推送
注册 token。

> **这一项不能因为「本地不存数据」就不填。** 本地存什么与 Data safety 无关 ——
> 该表单只管**离开设备**的数据。本包的本地存档（计时方案，含**用户自由输入的方案名**）不离开设备，
> 因而不申报；但 SDK 上报的设备标识确实离开设备，必须申报。

### App activity — **Collected + Shared**

| 字段 | 值 |
| --- | --- |
| Collected | ✅ |
| Shared | ✅ |
| Purposes | Analytics |

来源：AppsFlyer / Adjust 的 app open 与 session 事件。

## 不申报的

Name · Email · Phone · Address · Photos · Videos · Files · Contacts · Calendar ·
Precise location · Health · Financial info · Messages · Audio · Web history ·
Installed apps —— 本包一律不收集，也没有对应权限。

## 一处待定口径

**Approximate location。** App 不申请任何定位权限、读不到 GPS，但服务端会收到请求
来源 IP 并据此判断国家。Play 问的是「你的 app 收集或分享的数据」，含离开设备的数据。
IP 本身不在 Play 的数据类型清单里，但「由 IP 推得的粗略位置且用于改变行为」，
保守做法是申报 Approximate location / Collected / App functionality。

申报了要能自圆其说，不申报有被判漏报的风险。**这与 calcpad / hexacolorsort 是同一个
待定项，所有包必须同口径。** 当前值以运营与法务的决定为准，本文不复制具体答案。

## 取证

改动依赖（尤其加/删 SDK）后必须重跑权限核对并同步本文与 `PRIVACY_POLICY.md` ——
申报与实现不符是下架的常见原因。命令见 `SECURITY_NOTES.md`。
