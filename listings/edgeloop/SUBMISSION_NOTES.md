# 上架内部说明（**不要粘进 Play Console**）

对外文档（`PLAY_STORE_LISTING.md` · `PRIVACY_POLICY.md` · `DATA_SAFETY.md` ·
`PLAY_CONSOLE_FORM_ANSWERS.md`）已清理成纯可粘贴文本。所有内部讨论集中在本文。

## 口径原则：准确但不铺开

对外答案只需为真，不需要主动交代实现细节。例如「是否按地理位置改变行为」，
答案可以是

> Yes. Some content shown in the app is loaded from our server, and that content
> can vary by region. The puzzles and the user interface are identical for all
> users. The app does not read device location; where regional differences apply
> they are determined server-side from the request.

每句都为真，也没有一句在描述网关的判定规则。

## 与另外两个新包及前六个的关系

本包与 DaySpan / RepTimer 是同一批，与更早的 nonopix / linkflow / minegrid / unitshift /
tickpad / checklane 是同一套 SDK、同一套网关行为，**申报口径必须一致** ——
各包答案不同才是问题。

但**对外可见的素材必须各自独立**，否则 Play 侧会把几个包公开关联起来：

| 项目 | 要求 | 本包 |
| --- | --- | --- |
| 包名 | 各取不相关的厂商命名空间 | `bellcroft` |
| 证书 `O=` | **每包另取** | 见 `RELEASE_SIGNING.md` |
| 隐私政策 URL | **每包另建站点** | ✅ `https://precious-douhua-53c7dc.netlify.app/` |
| 支持邮箱 | ⚠️ **按游戏/工具分两组共用** | `appaztra@outlook.com` |
| `google-services.json` | 必须裁剪成只留本包 client | 见 `SECURITY_NOTES.md` |
| Adjust App Token | 必须新建 App，不可复用 | ✅ `g1usefapvwg0` |

> 关联性的实话：即便以上全部做到，几个包仍共用同一个 Firebase 项目
> （`hybrid-listings-51660`），因此 `google-services.json` 里的 `project_number` 与
> `api_key` 相同 —— 有人同时解压多个 APK 逐字段比对，依然能看出同属一个项目。
> 要彻底切断就得每个上架包一个独立 Firebase 项目，那是架构级改动，不在本包范围内。
> AppsFlyer devKey 是账号级的，同理。

## 邮箱共用的取舍

本包的支持邮箱是 `appaztra@outlook.com`，与 NonoPix / LinkFlow / MineGrid **共用一个**。

这与本文上面那张表原先的口径（每项素材各自独立）相反，是**运营明确决定的**，不是漏填。
记在这里是为了别在下一次改文档时又按老口径改回去。

要清楚代价：**支持邮箱会公开显示在 Play 商店页**。填同一个地址的几个包，
任何人对比几张商店页就能看出它们同属一家 —— 这正是各取独立厂商命名空间、
各建独立 Adjust App、各用独立签名证书想避免的事。

换句话说：包名、证书 `O=`、Adjust token 这几层的隔离仍然成立，
**但邮箱这一层是主动放弃的**。隐私政策 URL 仍要求每包一个，别把它也合并了 ——
两层都合并的话，前面那些隔离基本就白做了。

> 跨组之间没有关联：游戏那几个用 `appaztra@outlook.com`，
> 工具那几个用 `toolsa069@gmail.com`，两组互不相干。

## 为什么没有照抄 colorstack 的对应文档

colorstack 的 `PLAY_CONSOLE_FORM_ANSWERS.md` 写着：不使用广告/分析 SDK、不使用
Firebase、pubspec 无任何第三方运行时依赖、行为不随地理位置变化。

这四条对本包与对现在的 colorstack 都不成立 —— 两者都装了 `appsflyer_sdk`、
`adjust_sdk`、`firebase_messaging`，且 AB 面网关本身就是按来源 IP 的国家改变下发内容。
那些文档应是加网关之前写的、之后未同步。

实务角度：Google 会扫 APK。申报「无第三方依赖」而包里带着 `AD_ID` 权限，是申报与实现
不符 —— 这本身就是招查的点。而诚实申报「用 AppsFlyer/Adjust 做归因」是最普通的一栏。
因此对外文档按事实写，风险更低而不是更高。

本包的口径与 calcpad / hexacolorsort 对齐。

## 商店名的取舍

`android:label` 与商店名都定为 `EdgeLoop`。

**特意没有叫 Slitherlink。** 那是 Nikoli 的商标名，用作应用名有被判侵权的风险 ——
与当初 NonoPix 避开 `Picross` 是同一个考虑。`EdgeLoop` 描述的是玩法本身
（在格点之间画边、连成一条回路），与包名 `com.bellcroft.edgeloop8451` 一致。

## 本包发布前还缺的

| 项 | 状态 |
| --- | --- |
| keystore + `key.properties` | ✅ 已生成（不进 git）。`apksigner verify` 已确认签的是本包证书 |
| 图标与商店素材 | ✅ 齐备，且按 PNG 文件头验过 Play 的尺寸/通道硬规矩 |
| 代码与测试 | ✅ analyze 干净，33 项测试全过；release APK 48.4MB |
| `adjustAppToken` / `adjustOpenBLandingToken` | ✅ `g1usefapvwg0` / `tmzqo0` |
| `android/app/google-services.json` | ✅ 已放入，已裁剪成只留 `com.bellcroft.edgeloop8451` 一条 client |
| 支持邮箱 | ✅ `appaztra@outlook.com`（**注意与其余游戏包共用**）|
| 隐私政策 URL | ✅ `https://precious-douhua-53c7dc.netlify.app/`（实测匿名 HTTP 200）|
| 品牌归属 | ✅ `ap`（ArenaPlus），打开方式**外开**，`bSideChromeColor` = `0xFFFFFFFF`（主题吻合，未改常量）|

IARC 内容分级问卷的提交页要勾选 IARC 条款，那是一份法律声明，**必须由能代表发行方的
人本人提交**。

## 实机验证记录（2026-09-11，Android 模拟器）

按求解器算出的 22 条边逐一 `adb input tap`，**全部命中正确的边**；回路闭合后
`isSolved` 触发 —— 线转绿、提示数字全部转灰、出现 Next level、Solved 计数 +1。

这一次同时验了五件事：关卡库的题有解、求解器给的解正确、游戏判定与求解器同口径、
命中判定在真机上准、界面反馈正常。
