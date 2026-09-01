# Store assets — TileFit

> 本文列出 Play 商店页面需要的图形素材、各自的规格与产出办法。
> 图形素材已全部产出，可直接上传；**仍缺的是隐私政策托管 URL 与支持邮箱**（见文末）。

## 现状

| 路径 | 尺寸 / 格式 | 状态 |
| --- | --- | --- |
| `store/icon-512.png` | 512×512，32 位 PNG | ✅ 已产出 |
| `store/feature-graphic.png` | 1024×500，**24 位无 alpha** | ✅ 已产出 |
| `store/screenshot-1.png` | 1200×2400，24 位 | ✅ 已产出 —— 打了二十来手之后的棋盘（SCORE 20），待放区三个方块 |
| `store/screenshot-2.png` | 1200×2400，24 位 | ✅ 已产出 —— 刚起手几步的棋盘（SCORE 12） |
| `store/screenshot-3.png` | 1200×2400，24 位 | ✅ 已产出 —— 开局空棋盘 |
| `store/privacy-policy.html` | — | ✅ 已写好，**待托管** |

三张截图都是 **release APK 在 Android 模拟器上真打出来的局面**，没有摆拍、没有改代码。
Play 要求手机截图至少 2 张，这三张已够上架。

素材由脚本生成、可复现：

```bash
dart run tool/generate_icons.dart          # 三张图标源图
dart run flutter_launcher_icons            # 由源图生成 Android launcher 资源
dart run tool/generate_store_assets.dart <截图1.png> <截图2.png> ...
```

`generate_store_assets.dart` 一次做三件事：把 `icon.png` 缩成 512、自绘 1024×500 特色图、
把传进来的原始截图补边成 1200×2400。**三者都会去掉 alpha 通道**（特色图带透明会被 Play 拒收）。

包根目录的图标源图（`tool/generate_icons.dart` 生成，同时用于 launcher 图标）：

| 文件 | 尺寸 | 用途 |
| --- | --- | --- |
| `icon.png` | 1024×1024 | 方形图标源图，商店图标由它缩放而来 |
| `icon_background.png` | 1024×1024 | 自适应图标背景层 |
| `icon_foreground.png` | 1024×1024 | 自适应图标前景层 |

### 还可以再补的截图（可选，非阻塞）

现有三张都是静态局面。下面这两张能把玩法讲得更完整，但需要人工操作模拟器摆出时机，
脚本化的 `adb input swipe` 打不准：

| 画面 | 为什么值得补 |
| --- | --- |
| 拖动中的落点预览 | 这是玩法的核心交互 —— 方块浮在手指上方、目标格亮起半透明的本色 |
| 整行/整列消除的瞬间 | 消除是得分主来源（同时消 n 条得 10·n² 分） |

## 截图尺寸：1200×2400，不是 1080×2400

Play 规定截图**最长边不能超过最短边的 2 倍**。设备/模拟器原生截图是 1080×2400 = 2.22 倍，
会被拒。左右各补 60px 到 1200 宽，比例正好 2.0。

补的是纯色 `#0E1116` —— 与 `AppColors.background`（`lib/theme/app_colors.dart`）完全同色，
而游戏画面的背景就是这一个纯色（`MaterialApp` 的 `scaffoldBackgroundColor` 直接取它），
所以接缝在视觉上不存在，不需要按行取边缘像素延展。

出截图时记得做这一步，别直接上传 1080×2400。

## 截图是怎么拍的

在 Android 模拟器（AVD `hexa_test`，x86_64，**`-gpu host`**）上装 **release APK**（不是 debug）
实打出来的。同一次实跑顺带确认了：App 正常进 A 面不崩、落子与计分正确、最高分能写盘并在
重启后读回来（`/data/data/com.emberlane.tilefit8264/shared_prefs/FlutterSharedPreferences.xml`
里的 `flutter.tilefit_best_score`）。

**再拍时记得先开系统 UI demo 模式让状态栏干净** —— 现有三张截图没做这一步，状态栏里带着
模拟器自己的时间与图标。不影响上架（Play 不要求），但重拍时值得顺手做掉：

```bash
adb shell settings put global sysui_demo_allowed 1
adb shell am broadcast -a com.android.systemui.demo -e command enter
adb shell am broadcast -a com.android.systemui.demo -e command clock -e hhmm 0900
adb shell am broadcast -a com.android.systemui.demo -e command network -e wifi show -e level 4 -e mobile hide
adb shell am broadcast -a com.android.systemui.demo -e command notifications -e visible false
adb shell am broadcast -a com.android.systemui.demo -e command battery -e level 100 -e plugged false
```

另外先预授通知权限，否则每次冷启动都会弹授权框挡住画面：

```bash
adb shell pm grant com.emberlane.tilefit8264 android.permission.POST_NOTIFICATIONS
```

截图里的局面要真打出来，不要摆拍改代码 —— 尤其「同时消两条线」那张，靠真打更快，
也顺便验证了计分（落子得格数分，同时消 n 条额外得 10·n² 分）。

## 特色图

`store/feature-graphic.png` 由 `tool/generate_store_assets.dart` 自绘：一条 12×4 的网格带，
上面摆着形状表里真实存在的几个块（直角三格 / 2×2 / 三连 / 二连 / 单格），
用 `AppColors.pieceColors` 的本色，底色 `#0E1116` —— 与应用图标同一套画法。

**图里不放文字**：Play 会在特色图上叠自己的应用名与安装按钮，图里再写一遍标题只会打架；
而且 `package:image` 只有位图字体，排出来的字远不如纯图形干净。

已确认是 **24 位无 alpha**（`Format24bppRgb`）—— Play 不接受带透明通道的特色图。
生成脚本落盘前会显式 `convert(numChannels: 3)`，改动这段时别把它去掉。

## 隐私政策托管与支持邮箱 —— 待做

`store/privacy-policy.html` 是可直接托管的独立页面（无外链、自带深浅色样式、手机可读），
内容与 `PRIVACY_POLICY.md` 一致。丢到任意免费静态托管（GitHub Pages / Vercel /
Cloudflare Pages / Netlify Drop，都自带 HTTPS）即可拿到 Play 必填的 URL。

**上架前必须做的三件事：**

1. **托管政策页并确认公开可访问。** 有的托管商（如 Netlify）认领站点后会套用 team 默认
   可见性落成 **Private**，外部访问返回 401 登录墙。发布后必须从外部（换 IP 或用无痕）
   真拉一次，确认返回的是政策正文。**政策 URL 挂登录墙会被 Play 直接驳回。**
2. **不要复用其余上架包的政策 URL —— 新建一个站点。** 两个包指向同一个政策页面，等于在
   Play 侧把它们公开关联起来，与本包单独用 `emberlane` 命名空间、单独的证书 `O=`、
   裁剪过的 `google-services.json` 是相互矛盾的。
3. **支持邮箱要另取一个，并确认真的有人看。** Play 会把支持邮箱**公开显示在商店页面**。
   - 不要用公司域名邮箱（如 `@aztraglobal.com`）—— 直接暴露归属。
   - **不要复用任何已出现在其他上架包商店页面上的邮箱** —— 每复用一次就多一个公开关联点。
   - 选定后自己发一封测试邮件确认收得到。Google 的政策通知走这个地址，漏看整改期限
     可能直接下架。
   - 可行做法：注册一个与本包同名的独立域名 + 免费转发（ImprovMX 之类），或新开一个
     免费邮箱账号。

定好之后同步替换三处的 `TODO_SUPPORT_EMAIL` / TODO URL：
`PRIVACY_POLICY.md`、`store/privacy-policy.html`、`PLAY_STORE_LISTING.md`。
