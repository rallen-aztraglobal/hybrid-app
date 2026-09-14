# Play Console 其余表单答案 —— RepTimer

> Data safety 单独一份，见 `DATA_SAFETY.md`。

## 应用访问权限（App access）

**All functionality is available without special access.**

本包没有登录、没有付费墙、没有区域限定的功能入口 —— 审核员装上就能用全部功能。

## 广告（Ads）

**No, my app does not contain ads.**

包里没有任何广告 SDK、没有广告位。AppsFlyer 与 Adjust 是**归因**SDK，不投广告；
它们取 Advertising ID 是为了把安装归因到投放来源，这在 Data safety 里按
「Device or other IDs / Advertising or marketing」申报了。

## 内容分级（Content rating）

问卷本身要如实作答。本包无暴力、无性内容、无粗口、无赌博、无用户间交流、
无位置分享、无数字购买。

> **IARC 条款要你本人勾。** 内容分级问卷的提交页要勾选一份法律声明，
> **必须由能代表发行方的人本人提交** —— 我不代填。

## 目标受众与内容（Target audience）

目标年龄段建议选 **18 岁以上**（或不含 13 岁以下的任一档）。
选到 13 岁以下会触发 Families 政策的一整套额外要求，而本包带着归因 SDK 与
AD_ID 权限，不适合走那条线。

## 数据安全以外的声明

| 问题 | 答案 |
| --- | --- |
| 政府应用 | 否 |
| 金融功能 | 否 |
| 健康应用 | **否** —— 本包只是计时器，不采集任何健康数据、不给健康建议 |
| 新冠接触者追踪 | 否 |
| 用户生成内容 | 否 |

## 是否按地理位置或语言改变行为

**待定 —— 口径见 `SUBMISSION_NOTES.md`，三个新包与前六个及 calcpad / hexacolorsort
必须一致。** 不要凭印象填「否」：本包带 AB 面网关，服务端会按请求来源国家决定下发内容。

## 隐私政策 URL

`https://splendid-frangipane-a4c517.netlify.app/`

本包**专用**的 Netlify 站点，不与其他上架包共用。内容是 `store/privacy-policy.html`
改名 `index.html` 部署的，所以根路径直接就是政策页。

2026-09-14 实测（未登录）：HTTP 200、标题 `Privacy Policy — RepTimer`、
正文含本包包名与联系邮箱、无残留占位符。

> **改了仓库里的 `store/privacy-policy.html` 不等于线上改了** —— 静态托管是把文件传上去的，
> 改完必须重新部署一次，否则线上还是旧版。

> 线上页面比仓库里的文件多 6 行：Netlify 会自己注入一段托管声明注释、两个 meta 和一个
> HUD 脚本。政策正文一字未改（diff 只有新增，无删除无修改）。

> Netlify 现在把 Drop 上传的站点**默认设为 Private**（匿名访问 401）。
> 控制台里只显示一个很小的 Private 标签，很容易以为已经上线 —— 必须用 curl 实测。

## 联系邮箱

`toolsa069@gmail.com`

本批工具包共用这一个地址（本包 + UnitShift / TickPad / CheckLane / DaySpan）。
**这是运营的决定，不是漏填** —— 代价是支持邮箱会公开显示在商店页，
填同一个地址的几个包因此可被公开关联起来；取舍与理由见 `SUBMISSION_NOTES.md`。

## 提交前清单

- [x] keystore 与 `key.properties` 就位，签名核对过（`RELEASE_SIGNING.md`）
- [x] 图标与商店素材齐备（`STORE_ASSETS.md`）
- [x] 29 项测试全过，analyze 干净
- [x] Adjust App 已建（`9sgzjiaxgfeo`），`adjustAppToken` 与 `adjustOpenBLandingToken`（`av2feh`）已回填；7 个事件齐备
- [x] Firebase 已注册（`1:609439342540:android:d0c6a81f8fd6327da98769`，不加 SHA 指纹），裁剪过的 `google-services.json` 已放入
- [x] 隐私政策 URL 已上线且可匿名访问 —— `https://splendid-frangipane-a4c517.netlify.app/`，实测 HTTP 200
- [ ] 支持邮箱 `toolsa069@gmail.com` 能收信（发一封测试邮件确认，别只是看着像对的）
- [ ] Data safety 的 Approximate location 口径已定并与其余包一致
- [ ] 「按地理位置改变行为」口径已定并与其余包一致
- [ ] Console 侧本包 listing 的网关配置已核对
- [x] 品牌归属已定：`gp`（GameZone），PAL_CODE `1123689307775344641` —— Console 字段，未改任何常量、未重新打包
