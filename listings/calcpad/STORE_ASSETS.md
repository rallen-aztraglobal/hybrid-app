# Store assets — CalcPad

素材都在 `store/`，全部由本仓库的构建产物生成（模拟器实机截图 / 自绘），可直接上传。

| 文件 | 尺寸 | Play 用途 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512 | 商店图标（App icon） |
| `store/feature-graphic.png` | 1024×500 | 特色图（Feature graphic，必填） |
| `store/screenshot-1-keypad.png` | 1200×2400 | 手机截图 — 键盘（初始态） |
| `store/screenshot-2-equation.png` | 1200×2400 | 手机截图 — 算式预览 + 运算键高亮 |
| `store/screenshot-3-result.png` | 1200×2400 | 手机截图 — 结果 |
| `store/screenshot-4-decimal.png` | 1200×2400 | 手机截图 — 小数除法 |

Play 要求手机截图至少 2 张，这里给了 4 张。

## 为什么截图是 1200×2400 而不是 1080×2400

Play 规定截图**最长边不能超过最短边的 2 倍**。设备原生截图是 1080×2400 = 2.22 倍，
会被拒。这里左右各补 60px 到 1200 宽，比例正好 2.0。

补的是纯色 `#121212` —— 与 `AppColors.background` 完全同色，而计算器背景就是这一个纯色，
所以接缝在视觉上不存在（hexa 的背景是渐变，当时得按行取边缘像素延展；本包不需要）。

重新出截图时记得做同样的处理，别直接上传 1080×2400。

## 截图是怎么拍的

Android 35 模拟器（Pixel 6 / x86_64，`-gpu host`），装 **release APK**（不是 debug），
并开了系统 UI demo 模式让状态栏干净（只剩 9:00 与满电池，没有 VPN / 调试图标）：

```bash
adb shell settings put global sysui_demo_allowed 1
adb shell am broadcast -a com.android.systemui.demo -e command enter
adb shell am broadcast -a com.android.systemui.demo -e command clock -e hhmm 0900
adb shell am broadcast -a com.android.systemui.demo -e command network -e wifi show -e level 4 -e mobile hide
adb shell am broadcast -a com.android.systemui.demo -e command notifications -e visible false
adb shell am broadcast -a com.android.systemui.demo -e command battery -e level 100 -e plugged false
```

另外先 `adb shell pm grant com.northglade.calcpad5170 android.permission.POST_NOTIFICATIONS`，
否则每次冷启动都会弹通知授权框挡住画面（这也顺带验证了 `requestPermission()` 确实
会触发 Android 13+ 的运行时弹窗）。

截图里的两个算式是真算的，不是摆拍：`1234 × 56 = 69104`、`45.6 ÷ 8 = 5.7`。

## 隐私政策托管与支持邮箱 —— 待做

`store/privacy-policy.html` 是可直接托管的独立页面（无外链、自带深浅色样式、手机可读），
内容与 `PRIVACY_POLICY.md` 一致。丢到任意免费静态托管（GitHub Pages / Vercel /
Cloudflare Pages / Netlify Drop，都自带 HTTPS）即可拿到 Play 必填的 URL。

**上架前必须做的三件事：**

1. **托管政策页并确认公开可访问。** Netlify Drop 认领站点后可能套用 team 默认可见性
   落成 **Private**（外部访问返回 401 登录墙）—— hexacolorsort 上架时就踩过，必须在项目页
   点 `Make public`，然后从外部（换 IP 或用无痕）真拉一次确认返回的是政策正文。
   **政策 URL 挂登录墙会被 Play 直接驳回。**
2. **不要复用 hexacolorsort 的政策 URL**（`https://stalwart-khapse-3231a0.netlify.app`）。
   两个上架包指向同一个政策页面，等于在 Play 侧把它们公开关联起来，与各包刻意分开的
   包名、证书 `O=`、裁剪过的 google-services.json 是相互矛盾的。**建一个新站点。**
3. **支持邮箱要另取一个，并确认真的有人看。** Play 会把支持邮箱**公开显示在商店页面**。
   - 不要用公司域名邮箱（如 `@aztraglobal.com`）—— 直接暴露归属。
   - 也不要复用 `hgwlryr@outlook.com` —— 该地址已同时出现在 DeckTallyPro 与
     Hexa Color Sort 的商店页面上，再加一个就是第三个公开关联点。
   - 选定后自己发一封测试邮件确认收得到。Google 的政策通知走这个地址，漏看整改期限
     可能直接下架。

> 参考：其余上架包各不相同，都不涉及公司域名 ——
> - colorstack：注册 `colorstack.app` + ImprovMX 免费转发
> - decktallypro / hexacolorsort：`hgwlryr@outlook.com` + 托管的政策页

## 特色图

`store/feature-graphic.png` 是自绘的：左侧 2×2 运算键（与应用图标同一套画法、
用 `AppColors` 的原色），右侧标题 + tagline，底色沿用 `#121212`，右侧一道极淡的橙色斜向
光晕避免整幅纯黑。24 位 PNG 无 alpha（Play 不接受带透明通道的特色图，已确认
`PixelFormat=Format24bppRgb`）。文字与图形都留在中间区域，避免被各展示位裁切。

## 图标

`store/icon-512.png` 是 `icon.png`（1024×1024）等比缩到 512。两者与 App 内的
launcher 图标是同一份源图，见 `README_GATE.md` §5。
