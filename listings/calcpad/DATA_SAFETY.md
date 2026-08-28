# Play Console — Data safety 表单答案

逐项照填即可。取证依据与待定口径见 `SUBMISSION_NOTES.md`。

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
**待定 —— 见 `SUBMISSION_NOTES.md`。** App 不申请任何定位权限、读不到 GPS；
服务端会收到请求来源 IP。是否按「粗略位置」申报需要先定口径。
与 hexacolorsort 是同一个待定项，两包应取同一口径。

## 明确不申报
Name · Email · Phone · Address · Photos · Videos · Files · Contacts · Calendar ·
Precise location · Health · Financial info · Messages · Audio ·
Web browsing history · Installed apps 列表 —— App 均不收集。

**本包比其余上架包更少一项：没有任何本地持久化。** 计算器不存历史、不存设置
（`pubspec.yaml` 里没有 `shared_preferences`），用户输入的数字只存在于内存里、
关掉即无。Play 本来也不要求申报纯本地数据，但这里连本地数据都没有。
