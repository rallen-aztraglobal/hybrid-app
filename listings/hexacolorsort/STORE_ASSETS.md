# Store assets — Hexa Color Sort

素材都在 `store/`，全部由本仓库的构建产物生成（模拟器实机截图 / 自绘），可直接上传。

| 文件 | 尺寸 | Play 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（App icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic，必填） |
| `store/screenshot-1-home.png` | 1200×2400 | 手机截图 — 首页 |
| `store/screenshot-2-how-to-play.png` | 1200×2400 | 手机截图 — 玩法说明 |
| `store/screenshot-3-gameplay.png` | 1200×2400 | 手机截图 — 对局中 |
| `store/screenshot-4-selection.png` | 1200×2400 | 手机截图 — 选中方块 |

Play 要求手机截图至少 2 张，这里给了 4 张。

## 为什么截图是 1200×2400 而不是 1080×2400

Play 规定截图**最长边不能超过最短边的 2 倍**。设备原生截图是 1080×2400 = 2.22 倍，
会被拒。这里按行取左右边缘像素向两侧各延展 60px 补到 1200 宽，比例正好 2.0。
游戏背景是纯竖向渐变（`AppTheme.backgroundGradient` 是 topCenter→bottomCenter，
同一行水平方向同色），所以延展出来的边缘与原图完全连续，看不出接缝。

重新出截图时记得做同样的处理，别直接上传 1080×2400。

## 截图是怎么拍的

Android 35 模拟器（Pixel 6 / x86_64，`-gpu host`），装 release APK，
并开了系统 UI demo 模式让状态栏干净（只剩 9:00 与满电池，没有调试图标）：

```bash
adb shell settings put global sysui_demo_allowed 1
adb shell am broadcast -a com.android.systemui.demo -e command enter
adb shell am broadcast -a com.android.systemui.demo -e command clock -e hhmm 0900
adb shell am broadcast -a com.android.systemui.demo -e command network -e wifi show -e level 4 -e mobile hide
adb shell am broadcast -a com.android.systemui.demo -e command notifications -e visible false
adb shell am broadcast -a com.android.systemui.demo -e command battery -e level 100 -e plugged false
```

另外先 `adb shell pm grant com.slatecove.hexasort4173 android.permission.POST_NOTIFICATIONS`，
否则每次冷启动都会弹通知授权框挡住画面（这也顺带验证了 `requestPermission()` 确实
会触发 Android 13+ 的运行时弹窗）。

## 隐私政策托管与支持邮箱

`store/privacy-policy.html` 是可直接托管的独立页面（无外链、自带深浅色样式、手机可读），
内容与 `PRIVACY_POLICY.md` 一致。丢到任意免费静态托管（GitHub Pages / Vercel /
Cloudflare Pages，都自带 HTTPS）即可拿到 Play 必填的 URL。

**已就位。** 政策页托管在 Netlify（Netlify Drop 拖 store/privacy-policy.html 的副本，站点名随机、
不暴露归属），地址 **https://stalwart-khapse-3231a0.netlify.app** ；支持邮箱沿用 decktallypro
上架时用的 `hgwlryr@outlook.com`。未注册专用域名。

> ⚠️ Netlify Drop 认领站点后会套用 team 的默认可见性，本次落成 **Private** —— 外部访问返回
> 401 登录墙。必须在项目页点 `Make public`。已从外部拉取确认现在返回完整政策正文
> （标题、生效日期、AppsFlyer/Adjust/FCM/AD_ID 四项披露、联系邮箱均在）。
> 重新部署或换站点后要再验一次 —— 政策 URL 挂登录墙会被 Play 直接驳回。

参考：另两个上架包的做法各不相同，都不涉及公司域名 ——
- colorstack：注册 `colorstack.app` + ImprovMX 免费转发（实测其 MX 指向 `mx1/mx2.improvmx.com`）
- decktallypro：`hgwlryr@outlook.com` + privacypolicies.com 托管的政策页

> 为什么不用公司域名的邮箱（如 `@aztraglobal.com`）：Play 会把支持邮箱**公开显示在商店页面**，
> 用公司域名等于直接暴露归属，与本包刻意分开的包名（`com.slatecove.*`）、证书 `O=SlateCove`、
> 裁剪过的 google-services.json、独立 keystore 是相互矛盾的。

**提交 Play 前必须逐项确认，否则 Google 的政策通知发不到人手上（漏看整改期限可能直接下架）：**
1. 政策页 URL 能公开访问（无登录、不跳登录页、HTTPS）
2. `hgwlryr@outlook.com` **确实有人能登录并会看** —— 该地址取自 decktallypro 的既有政策页，
   本仓库无法验证其归属与是否有人监控，接手时务必自己发一封测试邮件确认收得到
3. 该邮箱同时出现在 DeckTallyPro 与本包的商店页面上，两者由此可被公开关联 —— 已知取舍

## 特色图

`store/feature-graphic.png` 是自绘的：左侧四色叠块（与应用图标同一套画法、
用游戏 `kColorPalette` 的原色），右侧标题 + tagline，底色沿用游戏深紫渐变。
24 位 PNG 无 alpha（Play 不接受带透明通道的特色图）。文字与图形都留在中间区域，
避免被各展示位裁切。
