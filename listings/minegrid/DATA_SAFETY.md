# Play Console — Data safety 表单答案

逐项照填即可。取证依据（实测权限清单）与待定口径见 `SUBMISSION_NOTES.md`。

依据是本包 `pubspec.yaml` 的实际依赖与**合并后**的 AndroidManifest，不是照抄其余上架包。
**改依赖后必须回来同步。**

## 三个总体问题

| 问题 | 答案 |
| --- | --- |
| Does your app collect or share any of the required user data types? | **Yes** |
| Is all of the user data collected by your app encrypted in transit? | **Yes**（全部走 HTTPS） |
| Do you provide a way for users to request that their data be deleted? | **Yes**（隐私政策里的联系邮箱） |

**第一个问题必须答 Yes。** 本包只在本机存两个整数，很容易顺手答成 No —— 但包里带着
AppsFlyer / Adjust / Firebase Messaging，那三个 SDK 会上报设备标识，答 No 就是虚假申报。

## 需要申报的数据类型

### Device or other IDs

| 字段 | 值 |
| --- | --- |
| Collected | Yes |
| Shared | Yes（AppsFlyer、Adjust、Google） |
| Processed ephemerally | No |
| Required or optional | Required |
| Purposes | Analytics · Advertising or marketing · App functionality |

内容：Google Advertising ID、Firebase 注册 token、install referrer。

> 取证：源 manifest 里只写了 `INTERNET`，但合并后的 manifest 会多出
> `com.google.android.gms.permission.AD_ID`、
> `com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE`、
> `com.google.android.c2dm.permission.RECEIVE`、`POST_NOTIFICATIONS`、`WAKE_LOCK`、
> `ACCESS_NETWORK_STATE` —— 全部由 SDK 注入。**按合并后的 manifest 申报，不是按源文件。**

### App activity → App interactions

| 字段 | 值 |
| --- | --- |
| Collected | Yes |
| Shared | Yes（AppsFlyer、Adjust） |
| Required or optional | Required |
| Purposes | Analytics · Advertising or marketing |

内容：app 打开与会话事件、归因自定义事件。

### App info and performance → Other app performance data

| 字段 | 值 |
| --- | --- |
| Collected | Yes |
| Shared | No |
| Required or optional | Required |
| Purposes | App functionality · Analytics |

内容：设备型号与操作系统版本（随 push token 注册一并上报）。

### Location → Approximate location

**待定 —— 见 `SUBMISSION_NOTES.md`。** App 不申请任何定位权限、读不到 GPS；
服务端会收到请求来源 IP。是否按「粗略位置」申报需要先定口径。
与 calcpad / hexacolorsort 是同一个待定项，所有包必须同口径。

## 明确不申报

Name · Email · Phone · Address · Photos · Videos · Files · Contacts · Calendar ·
Precise location · Health · Financial info · Messages · Audio ·
Web browsing history · Installed apps 列表 —— App 均不收集。

## 本地持久化：有，但不申报

`pubspec.yaml` 里有 `shared_preferences`，`lib/storage/progress_store.dart` 用它存
**八个整数**：

| key | 值 |
| --- | --- |
| `minegrid_wins_<档>` | 该难度档赢过多少局 |
| `minegrid_best_<档>` | 该难度档最快用时（秒） |

`<档>` 取 `normal` / `hard` / `expert` / `master`。

写入时机：赢一局时 +1，用时更快则刷新纪录。读写失败一律当「没有记录」处理
（存档坏了顶多丢战绩，不影响开局）。

这些整数**只存在本机 SharedPreferences 里**：不随网关判定请求上报、不进 push token
注册体、不作为归因事件参数，卸载即随应用数据一并删除。

Play 的 Data safety 只要求申报**离开设备**的数据，纯本地数据不在申报范围内，
故 **Data safety 表单里不需要为它勾任何东西**。但隐私政策里要如实写出来（已写，
见 `PRIVACY_POLICY.md` 的 "Stored only on your device"）—— 商店页面上说「什么都不存」
而实际存了战绩，是没必要给自己留的漏洞。
