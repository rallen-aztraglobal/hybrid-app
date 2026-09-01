# Play Console — Data safety 表单答案

> 本文是 Play Console「Data safety」表单的逐项答案，照填即可。
> 依据是本包**合并后**的 AndroidManifest（`build/app/intermediates/merged_manifests/release/`）
> 与 `pubspec.yaml` 的实际内容，不是照抄其余上架包。改依赖后必须回来同步。

## 三个总体问题
| 问题 | 答案 |
| --- | --- |
| Does your app collect or share any of the required user data types? | **Yes** |
| Is all of the user data collected by your app encrypted in transit? | **Yes**（全部走 HTTPS） |
| Do you provide a way for users to request that their data be deleted? | **Yes**（隐私政策内的联系邮箱） |

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

> 取证：合并后的 manifest 里确实带有 `com.google.android.gms.permission.AD_ID`、
> `com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE`、
> `com.google.android.c2dm.permission.RECEIVE`、`POST_NOTIFICATIONS`、`WAKE_LOCK`、
> `ACCESS_NETWORK_STATE`（均由 SDK 注入，源 manifest 里只写了 `INTERNET`）。
>
> **Adjust 目前是占位 token（`TODO_ADJUST_APP_TOKEN`），SDK 全链路 no-op、实际不上报。**
> 但 SDK 已打进包里，且填上 token 就会开始上报，故 Data safety 按「会收集并共享」申报，
> 不按当前的休眠状态申报 —— 申报比实现宽是安全的，反过来才是下架风险。

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

内容：设备型号与操作系统版本（随 token 注册上报）。

### Location → Approximate location
**待定口径。** App 不申请任何定位权限（合并 manifest 里没有 `ACCESS_*_LOCATION`）、
读不到 GPS；但服务端会收到请求来源 IP，而 Play 对「由 IP 推得的粗略位置」是否需要申报
一直有解释空间。与其余上架包是同一个待定项，**几个包必须取同一口径**，不要一个申报一个不申报。

- 保守口径：申报 Approximate location，Collected = Yes / Shared = No /
  Purposes = App functionality（内容写「服务端从请求 IP 得到的国家级粗略位置」）。
- 宽松口径：不申报，理由是 App 本身不采集位置、IP 只是网络传输的必然产物。

上架前定下来并在这里写死，别留着。

## 明确不申报
Name · Email · Phone · Address · Photos · Videos · Files · Contacts · Calendar ·
Precise location · Health · Financial info · Messages · Audio ·
Web browsing history · Installed apps 列表 —— App 均不收集。

## 本地持久化：有一项，但不申报

**本包有本地存储，这点与其余上架包不同，别照抄「完全无本地存储」的说法。**

`pubspec.yaml` 里有 `shared_preferences`，`lib/storage/best_score_store.dart` 用它存
**一个整数**：

- key：`tilefit_best_score`
- 值：本机历史最高分
- 写入时机：一局结束、且本局分数高于已存值
- 读写失败一律当「没有记录」处理（存档坏了顶多丢最高分，不影响开局）

这个整数**只存在本机 SharedPreferences 里**：不随网关判定请求上报、不进 push token 注册体、
不作为归因事件参数，卸载即随应用数据一并删除。

Play 的 Data safety 只要求申报**离开设备**的数据，纯本地数据不在申报范围内，
故 **Data safety 表单里不需要为它勾任何东西**。但隐私政策里要如实写出来（已写，
见 `PRIVACY_POLICY.md` 的 “Stored only on your device”）—— 商店页面上说「什么都不存」
而实际存了最高分，是没必要给自己留的漏洞。
