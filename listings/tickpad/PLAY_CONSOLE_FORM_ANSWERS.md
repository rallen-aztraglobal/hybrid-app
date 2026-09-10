# Play Console — 各表单答案

逐项照填即可。Data safety 另见 `DATA_SAFETY.md`；内部讨论与待定口径见
`SUBMISSION_NOTES.md`（**那份不要粘进 Console**）。

## 应用类型

| 字段 | 值 |
| --- | --- |
| 这是应用还是游戏 | 应用 |
| 类别 | Tools |
| 免费或付费 | 免费 |
| 内含广告 | 否 |
| 应用内购买 | 否 |

## 内容分级问卷（IARC）

分类选 **应用（非游戏）→ 参考/工具**，其余问题按下表：

| 问题 | 答案 |
| --- | --- |
| 暴力（写实 / 卡通 / 幻想） | 否 |
| 性内容、裸露 | 否 |
| 粗俗语言 | 否 |
| 毒品、酒精、烟草 | 否 |
| 赌博（含模拟赌博） | 否 |
| 恐怖 / 惊吓元素 | 否 |
| 用户之间可以互动或交流 | 否 |
| 分享用户位置 | 否 |
| 允许购买数字商品 | 否 |
| 内含用户生成内容 | 否 |

预期结果：Everyone / 3+。

## 目标受众和内容

| 字段 | 值 |
| --- | --- |
| 目标年龄段 | 13-15、16-17、18 及以上 |
| 是否面向儿童 | 否 |
| 广告内容是否适合儿童 | 不适用（无广告） |

## 广告 ID

| 问题 | 答案 |
| --- | --- |
| 应用是否使用广告 ID | 是 |
| 用途 | Analytics · App functionality |

## 应用访问权限（App access）

选 **All functionality is available without special access**。
全部功能无需登录即可使用，不需要提供审核账号，也不需要上传演示视频。

## 核心功能说明（若表单要求描述）

> TickPad is a tally counter. The user creates one or more counters, gives each a
> name, and taps a card to count up; a minus button counts down and there is a
> thirty-step undo. Counters can have a step size, a target and a colour. A round
> of counting can be saved with a timestamp and reviewed later, or copied to the
> clipboard as plain text. Optionally the hardware volume keys can be used to
> count (off by default). Everything runs on the device. There is no login, no
> subscription and no content downloaded for the counters themselves. Counters,
> saved sessions and that one setting are stored on the device.

## 第三方 SDK 与用途（若表单要求列出）

> The app uses the Flutter SDK and the following third-party packages:
>
> - AppsFlyer and Adjust — install attribution and app-open/session analytics.
> - Firebase Cloud Messaging (firebase_core, firebase_messaging) — push
>   notifications.
> - webview_flutter — renders remote content inside the app.
> - url_launcher — opens links in the device browser.
> - flutter_timezone — reads the device's IANA time zone.
> - shared_preferences — stores the counters, saved sessions and the volume-key
>   setting, locally on the device.
>
> The app also contains a small amount of Kotlin in its own MainActivity, which
> intercepts the hardware volume keys and forwards them to the app as count events
> over a Flutter MethodChannel. This is off by default and is switched on by the
> user. It is the app's own code, not a third-party SDK.
>
> These SDKs are configured according to their published documentation. No
> advertising is displayed in the app. The app requests only the INTERNET
> permission in its own manifest; the additional permissions visible in the merged
> manifest (AD_ID, install-referrer, C2DM receive, POST_NOTIFICATIONS, WAKE_LOCK,
> ACCESS_NETWORK_STATE) are declared by those SDKs.

## 知识产权

选 **No third party intellectual property appears in my app**。

> The app uses original artwork and puzzle designs created for this project, plus
> standard Flutter/Material UI components. It contains no third-party brands,
> licensed characters or copyrighted music.

## 是否按地理位置或语言改变行为

**待定 —— 口径见 `SUBMISSION_NOTES.md`，六个新包与 calcpad / hexacolorsort 必须一致。**
不要凭印象填「否」：本包带 AB 面网关，服务端会按请求来源国家决定下发内容。

## 隐私政策 URL

`https://delicate-seahorse-a2b478.netlify.app/`

本包**专用**的 Netlify 站点，不与其余五个新包共用。内容是 `store/privacy-policy.html`
改名 `index.html` 部署的，所以根路径直接就是政策页。

2026-09-10 实测（未登录）：HTTP 200、标题 `Privacy Policy — TickPad`、
正文里的联系邮箱正确、无残留占位符。

> **改了仓库里的 `store/privacy-policy.html` 不等于线上改了** —— 静态托管是把文件传上去的，
> 改完必须重新部署一次，否则线上还是旧版。

> 线上页面比仓库里的文件多几行：Netlify 会自己注入一段托管声明注释、两个 meta 和一个 HUD
> 脚本。政策正文一字未改（diff 只有新增）。

## 联系邮箱

`toolsa069@gmail.com`

本批三个工具包共用这一个地址（本包 + UnitShift / CheckLane）。
**这是运营的决定，不是漏填** —— 代价是支持邮箱会公开显示在商店页，
三个包填同一个地址等于把它们公开关联起来；取舍与理由见 `SUBMISSION_NOTES.md`。

## 提交前清单

- [ ] keystore 与 `key.properties` 就位，签名核对过（`RELEASE_SIGNING.md`）
- [x] 隐私政策 URL 已上线且可匿名访问 —— `https://delicate-seahorse-a2b478.netlify.app/`，实测 HTTP 200、标题 `Privacy Policy — TickPad`
- [ ] 支持邮箱 `toolsa069@gmail.com` 能收信（发一封测试邮件确认，别只是看着像对的）
- [ ] Data safety 的 Approximate location 口径已定并与其余包一致
- [ ] 「按地理位置改变行为」口径已定并与其余包一致
- [ ] Console 侧本包 listing 的网关配置已核对
- [ ] IARC 问卷由能代表发行方的人本人提交
