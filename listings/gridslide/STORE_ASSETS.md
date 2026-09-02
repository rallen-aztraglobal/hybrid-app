# Store Assets — GridSlide

> 本文列出 **App Store** 需要的素材、各自的准确规格与产出办法。
>
> ⚠️ **规格与 Google Play 完全不同** —— 没有「特色图（feature graphic）」，
> 截图尺寸是按 **iPhone 显示屏英寸档位**定的固定像素值，不是「最长边不超过最短边 2 倍」
> 那种比例规则。不要照抄仓库里 Android 包的 `STORE_ASSETS.md`。
>
> **本文整篇是一份待做清单。** 除了图标，本包的商店素材一件都还没有。

---

## 现状 —— **截图一张都没有**

| 素材 | 规格 | 状态 |
| --- | --- | --- |
| App 图标 | 1024×1024，PNG，**24 位无 alpha** | ✅ **已就位** —— `GridSlide/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png` |
| 6.9" iPhone 截图 | 1320×2868 或 1290×2796，竖版 | ❌ **一张都没有** |
| 6.5" iPhone 截图 | 1284×2778 或 1242×2688，竖版 | ❌ **一张都没有** |
| App 预览视频（选填） | 见下 | ❌ 没有（**不做也能上架**） |
| 隐私政策页 | — | ✅ 已写好（`store/privacy-policy.html`），**待托管** |
| iPad 截图 | — | **不需要**（`TARGETED_DEVICE_FAMILY = 1`，仅 iPhone） |

### 为什么一张都没有

**本工程从未编译、从未运行过。** 写代码的机器是 Windows，没有 Xcode，也没有 Swift 工具链。
没有可运行的 App，就没有可截的画面。

**必须先在 Mac 上把它跑起来，才谈得上截图。** 顺序是死的，一步都跳不过去：

1. 在 Mac 上**编译通过**（命令见 `README_GATE.md` 末节）
2. 在模拟器里跑通 `README_GATE.md` 那 **12 步手验路径**
3. 手玩出一份**看得过去的成绩数据**（见下文「截图里该出现什么」）
4. 逐屏截图
5. 按下面的规格核对尺寸、色彩空间与 alpha
6. 上传

**不要用设计稿、拼图或 Figma 摆出来的假界面充数。** Apple 明确要求截图反映 App 的实际
使用画面（Guideline 2.3.3），画面里出现 App 里根本不存在的界面是驳回项。
对本包尤其要注意一点：**别在截图上叠加「排行榜」「奖励」「每日任务」之类的营销装饰** ——
App 里没有这些东西，加上去既是 2.3.3，也会把年龄分级问卷里那三项博彩相关答案变得可疑。

---

## 截图规格（Apple 现行要求）

### 必需档位

**App Store Connect 目前对 iPhone 只强制要求 6.9" 一档**，其余档位不传的话由 Apple
自动缩放填充。但**要求 6.9" 与 6.5" 两档都单独出**：自动缩放对本 App 这种
「大块纯色方块 + 等宽数字」的画面会让边缘发虚、数字发糊，而 6.5" 是老机型用户
实际看到的那一张。

| 档位 | 接受的像素尺寸（竖版） | 对应机型（举例） |
| --- | --- | --- |
| **6.9"**（必填） | **1320 × 2868** 或 **1290 × 2796** | iPhone 16 Pro Max / 16 Plus / 15 Pro Max / 15 Plus |
| **6.5"**（必填，本包自己的要求） | **1284 × 2778** 或 **1242 × 2688** | iPhone 11 Pro Max / XS Max / 8 Plus 之后的大屏档 |
| 6.3" / 6.1"（选填） | 1206 × 2622 或 1179 × 2556 | iPhone 16 Pro / 16 / 15 Pro / 15 |
| 5.5"（选填） | 1242 × 2208 | iPhone 8 Plus |

> - **本 App 竖屏锁定**（`INFOPLIST_KEY_UISupportedInterfaceOrientations = UIInterfaceOrientationPortrait`），
>   所以**只出竖版**，不需要横版（横版是把宽高对调，例如 2868×1320）。
> - 每个档位最多 **10 张**，至少 **1 张**。建议出 **4 张**（见下表）。
> - **尺寸必须精确匹配**上表里的某一个值，差一个像素就会被 App Store Connect 拒收。
>   与 Play 不同，这里**不能自己补边凑比例**。

### 格式要求

| 项 | 要求 |
| --- | --- |
| 文件格式 | **PNG** 或 JPEG |
| 色彩空间 | **RGB**（不要 CMYK、不要灰度） |
| 透明通道 | **不允许** —— 必须无 alpha |
| 图层 | 扁平化，不能带图层 |
| 分辨率 | 72 dpi |

> 模拟器 ⌘S 截出来的 PNG 默认就是 RGB 无 alpha，一般不用处理。但**上传前核一遍**：
>
> ```bash
> sips -g pixelWidth -g pixelHeight -g hasAlpha -g space screenshot-1.png
> ```
>
> `hasAlpha: no` + 尺寸精确匹配 + `space: RGB` 三项都对才算过。

### 怎么截（在 Mac 上）

```bash
# 1. 起一台 6.9" 的模拟器（iPhone 16 Pro Max，原生 1320×2868）
xcrun simctl list devices | grep "16 Pro Max"
open -a Simulator

# 2. 装并跑起来
xcodebuild -scheme GridSlide -project GridSlide.xcodeproj \
  -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 16 Pro Max' build

# 3. 截图（命令行截的是原生分辨率，比 ⌘S 可靠）
xcrun simctl io booted screenshot store/screenshot-69-1.png
```

6.5" 那一档换一台 **iPhone 11 Pro Max**（原生 1242×2688）重截一遍，
**不要**把 6.9" 的图缩放过去 —— 缩放会让数字发虚，而且尺寸也对不上。

截图前把状态栏弄干净（时间统一、满格信号、满电）：

```bash
xcrun simctl status_bar booted override \
  --time "9:41" --cellularBars 4 --wifiBars 3 --batteryState charged --batteryLevel 100
```

> `9:41` 是 Apple 自己宣传图里的惯例时间，用它最不扎眼。

---

## 截图里该出现什么

四张，按这个顺序（App Store 上第一张最重要，多数人只看第一张）：

| # | 页面 | 画面里要有 |
| --- | --- | --- |
| 1 | **Play（4×4，局中）** | 一个**解到一半**的 4×4 棋盘：**至少 6～8 块已经归位、显示强调色**，其余仍是浅色块。上方 Moves 有两位数、Time 走到 0:30 上下、Best 有一个数字（不是破折号）。这一张要在缩略图尺寸下就能看出「归位的块会变色」这条卖点 |
| 2 | **Play（3×3 或 5×5）** | 换一个尺寸，让人一眼知道**尺寸是可选的**。顶部的 3×3 / 4×4 / 5×5 分段控件要清晰可见、选中态明显。建议用 **5×5**，块多、画面信息量大，与第 1 张形成对比 |
| 3 | **通关结算浮层** | 解开一局后的浮层：Solved + "N moves · MM:SS" + **"New best moves and time"** 那行强调色文字 + Play again 按钮。这一张负责讲「双纪录」 |
| 4 | **Records** | 三张卡都要**有数据**：3×3 / 4×4 / 5×5 各自的 Best moves、Best time、Solved 次数。**三个尺寸都不能是破折号** —— 破折号的空状态适合手验，不适合商店。卡片下方那句 "Best moves and best time are tracked separately..." 要在画面里 |

> 可选的第 5 张：**Settings** 页（Board size / Haptics / Privacy policy 三行 +
> 下方那句「成绩只存本机、没有账号」的说明）。它对「无账号、数据在本机」这条卖点有帮助，
> 但排在前四张之后。

**演示数据要真玩出来，不要改代码造假。** 三个尺寸各解一到两局就够了
（3×3 很快；4×4 慢慢来；5×5 若嫌久，可以先解 5×5 一局拿到一个成绩，
数字难看也没关系，Records 卡上有数就行）。理由和其他包一样：手玩一遍顺带就把功能验了 ——
尤其第 3 张那个浮层，截图的同时正好确认「首次通关显示 New best moves and time」这条是对的。

> ⚠️ 第 1 张要「解到一半 + 多块归位」，靠随机打乱等不来，**要手动滑到那个状态**。
> 值得多花两分钟：这一张决定了大部分人会不会点进来。

---

## App 图标 —— 已就位

`GridSlide/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png`
1024×1024，**24 位无 alpha**（iOS 图标带透明通道会被 App Store 拒收）。

Xcode 14 起 App Icon 只需要这一张 1024 的源图，其余尺寸由工具链生成 ——
`Contents.json` 里也确实只声明了一个 `universal / ios / 1024x1024` 条目，是对的。

### 设计参数（需要重画时照着来）

图案是**深色渐变底 + 3×3 网格，右下角空一格**，空格左边那块用强调色 ——
读起来就是「它正要滑进空位」。纯几何、不依赖字体渲染
（不用字体的原因：跨机器渲染结果不一致，字体授权也是个麻烦）。

| 元素 | 参数 |
| --- | --- |
| 底色 | 深色渐变，与 App 主题一致：`AppTheme.background` **`#101418`** → `AppTheme.surface` **`#1B212B`** |
| 网格 | 3×3，**右下角那一格留空**（滑块拼图之所以能玩全靠那个空格，这是这个图案的全部意思） |
| 普通块 | 浅色实体块，取 `AppTheme.tileFill` **`#E7ECF2`** |
| 强调块 | **空格左边那一块**用强调色 **`#4CC2A0`**（= `AppTheme.accent`，也是 `AccentColor.colorset` 里那个 sRGB 0.298 / 0.761 / 0.627） |
| 圆角 | 每个方块自身带圆角；**整张图不要预先切圆角** —— iOS 会自己套 squircle 遮罩，自己切会露出双重圆角 |
| 边距 | 网格四周留白，别顶边 |
| 通道 | **必须 24 位 RGB 无 alpha** |

> **不要在图标里加数字。** 1024 缩到桌面尺寸后数字糊成一团，而且一旦用了字体，
> 跨机器渲染就不一致了。现在这版是纯色块，缩到 40px 依然读得出「网格缺一角」。
>
> 生成脚本**没有进仓库**：那是个一次性脚本，而本包是纯 Swift 工程。需要改时照上面的参数重画。
>
> 重画后**务必确认无 alpha**：
> ```bash
> sips -g hasAlpha -g pixelWidth -g pixelHeight AppIcon-1024.png
> ```
> 要看到 `hasAlpha: no` 与 1024×1024。**很多图形工具默认导出 32 位 RGBA**，
> 这是这一步最常见的翻车点，也是 App Store 上传时最常见的一条报错。

---

## App 预览视频（选填，**不做也能上架**）

| 项 | 规格 |
| --- | --- |
| 数量 | 每个档位最多 **3** 个 |
| 时长 | **15–30 秒** |
| 格式 | M4V / MP4 / MOV |
| 分辨率 | 与对应档位的截图尺寸一致（6.9" 竖版 = 1290×2796） |
| 大小 | ≤ 500 MB |
| 内容 | 必须是 App 内的实际录屏，不能是纯宣传片 |

> 与记账类 App 不同，**本包是少数值得考虑做预览视频的**：整排滑动这条卖点，
> 静态截图讲不清楚 —— 「点一个隔着三格的块，中间的块一起滑过去，只算一步」
> 用两秒的动图一看就懂。
>
> 但**首发仍建议不做**：得先有能跑起来的 App。真要做，
> `xcrun simctl io booted recordVideo preview.mov` 录一段，内容就演
> 「点一次相邻块 → 点一次隔三格的块（整排滑动）→ 几块变成强调色 → 解开 → 结算浮层」。

---

## 隐私政策托管、Support URL 与支持邮箱 —— 待做

`store/privacy-policy.html` 是可直接托管的独立页面（**无外链、自带深浅色适配、手机可读**，
配色跟着本包的深色主题走），内容与 `PRIVACY_POLICY.md` 一致。丢到任意免费静态托管
（GitHub Pages / Vercel / Cloudflare Pages / Netlify Drop，都自带 HTTPS）即可拿到
App Store Connect 必填的 URL。

**上架前必须做的四件事：**

1. **托管政策页并确认公开可访问。** 有的托管商（如 Netlify）认领站点后会套用 team 默认
   可见性落成 **Private**，外部访问返回 401 登录墙。发布后必须从外部（换 IP 或用无痕）
   真拉一次，确认返回的是政策正文。**政策 URL 挂登录墙会被直接驳回。**
2. **Support URL 也要有一个。** 这是 App Store 的**必填项**，且**公开显示在商品页上**。
   可以就是一个静态页面，写清 App 是做什么的 + 一个可达的联系邮箱，不需要工单系统。
3. **支持邮箱另取一个，并确认真的有人看。**
   - 不要用公司域名邮箱（如 `@aztraglobal.com`）—— 直接暴露归属。
   - **不要复用任何已出现在其他上架包商店页面上的邮箱** —— 每复用一次就多一个公开关联点。
   - 选定后自己发一封测试邮件确认收得到。Apple 的政策通知与审核沟通走这个地址，
     漏看整改期限可能直接下架。
   - 可行做法：注册一个与本包同名的独立域名 + 免费转发（ImprovMX 之类），
     或新开一个免费邮箱账号。
4. **不要复用其余上架包的政策 URL / 支持 URL —— 新建站点。**
   两个 App 的商品页指向同一个域名，等于在 App Store 侧把它们公开关联起来，
   与本包单独用 `fernvale` 命名空间的初衷直接矛盾。
   （同类顾虑还有一个更硬的：**Seller 名称**。见 `RELEASE_SIGNING.md` 第 1 节。）

定好之后要同步替换的**四处**：

1. `APP_STORE_LISTING.md` 的 URL 表
2. `PRIVACY_POLICY.md` 的 Contact 段
3. `store/privacy-policy.html` 的 Contact 段与页脚
4. `GridSlide/Core/Services/SettingsStore.swift` 的
   `privacyPolicyURLString` 与 `supportEmail`
   —— **改这一处需要重新出构建版本**，别留到最后

---

## 完整待做清单（本文范围内）

- [ ] **在 Mac 上编译通过并跑起来**（**一切的前提** —— 本工程从未编译过）
- [ ] 跑通 `README_GATE.md` 的 12 步手验路径
- [ ] 手玩出成绩：3×3 / 4×4 / 5×5 三个尺寸**各至少解开一局**（Records 页不能有破折号）
- [ ] 手动把一局 4×4 滑到「多块已归位」的状态，用于第 1 张截图
- [ ] 出 4 张 **6.9"** 截图（1320×2868 或 1290×2796）
- [ ] 出 4 张 **6.5"** 截图（1284×2778 或 1242×2688）
- [ ] 用 `sips` 核对每一张的尺寸 / 无 alpha / RGB
- [ ] 托管 `store/privacy-policy.html`，**从外部**验证可访问（非登录墙）
- [ ] 定 Support URL
- [ ] 定支持邮箱并发测试邮件验证
- [ ] 四处 TODO 全部替换（含需要重新出包的 `SettingsStore` 那一处）
- [ ] 归档后确认 `.ipa` 里的图标仍是无 alpha
