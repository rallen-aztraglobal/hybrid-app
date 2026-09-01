# Store Assets — PocketLedger

> 本文列出 **App Store** 需要的素材、各自的准确规格与产出办法。
>
> ⚠️ **规格与 Google Play 完全不同** —— 没有「特色图（feature graphic）」，
> 截图尺寸是按 **iPhone 显示屏英寸档位**定的固定像素值，不是「最长边不超过最短边 2 倍」
> 那种比例规则。不要照抄仓库里 Android 包的 `STORE_ASSETS.md`。

---

## 现状 —— **截图一张都没有**

| 素材 | 规格 | 状态 |
| --- | --- | --- |
| App 图标 | 1024×1024，PNG，**24 位无 alpha** | ✅ **已就位** —— `PocketLedger/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png` |
| 6.9" iPhone 截图 | 1320×2868 或 1290×2796，竖版 | ❌ **一张都没有** |
| 6.5" iPhone 截图 | 1284×2778 或 1242×2688，竖版 | ❌ **一张都没有** |
| App 预览视频（选填） | 见下 | ❌ 没有（**不做也能上架**） |
| 隐私政策页 | — | ✅ 已写好（`store/privacy-policy.html`），**待托管** |
| iPad 截图 | — | **不需要**（`TARGETED_DEVICE_FAMILY = 1`，仅 iPhone） |

### 为什么一张都没有

**本工程从未编译、从未运行过。** 写代码的机器是 Windows，没有 Xcode，也没有 Swift 工具链。
没有可运行的 App，就没有可截的画面。

**必须先在 Mac 上把它跑起来，才谈得上截图。** 顺序是死的：

1. 在 Mac 上编译通过（命令见 `README_GATE.md` 末节）
2. 在模拟器里跑通 `README_GATE.md` 那 9 步手验路径
3. 造出一份**看得过去的演示数据**（见下文「截图里该出现什么」）
4. 逐屏截图
5. 按下面的规格核对尺寸与色彩空间
6. 上传

**不要用设计稿、拼图或 Figma 摆出来的假界面充数。** Apple 明确要求截图反映 App 的实际
使用画面（Guideline 2.3.3），画面里出现 App 里根本不存在的界面是驳回项。

---

## 截图规格（Apple 现行要求）

### 必需档位

**App Store Connect 目前对 iPhone 只强制要求 6.9" 一档**，其余档位不传的话由 Apple
自动缩放填充。但**建议 6.9" 与 6.5" 两档都单独出**：自动缩放对文字密集的界面
（本 App 的流水列表就是）会糊，而 6.5" 是老机型用户实际看到的那一张。

| 档位 | 接受的像素尺寸（竖版） | 对应机型（举例） |
| --- | --- | --- |
| **6.9"**（必需） | **1320 × 2868** 或 **1290 × 2796** | iPhone 16 Pro Max / 16 Plus / 15 Pro Max / 15 Plus |
| **6.5"**（建议） | **1284 × 2778** 或 **1242 × 2688** | iPhone 11 Pro Max / XS Max / 8 Plus 之后的大屏档 |
| 6.3" / 6.1"（选填） | 1206 × 2622 或 1179 × 2556 | iPhone 16 Pro / 16 / 15 Pro / 15 |
| 5.5"（选填） | 1242 × 2208 | iPhone 8 Plus |

> - **本 App 竖屏锁定**（`UISupportedInterfaceOrientations = Portrait`），
>   所以**只出竖版**，不需要横版（横版是把宽高对调，例如 2868×1320）。
> - 每个档位最多 **10 张**，至少 **1 张**。建议出 **4 张**（四个标签页各一张）。
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

> 模拟器 ⌘S 截出来的 PNG 默认就是 RGB 无 alpha，一般不用处理。但**归档前核一遍**：
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
xcodebuild -scheme PocketLedger -project PocketLedger.xcodeproj \
  -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 16 Pro Max' build

# 3. 截图（命令行截的是原生分辨率，比 ⌘S 可靠）
xcrun simctl io booted screenshot store/screenshot-69-1.png
```

6.5" 那一档换一台 **iPhone 11 Pro Max**（原生 1242×2688）重截一遍，
**不要**把 6.9" 的图缩放过去 —— 缩放会让文字发虚，而且尺寸也对不上。

截图前把状态栏弄干净（时间统一、满格信号、满电）：

```bash
xcrun simctl status_bar booted override \
  --time "9:41" --cellularBars 4 --wifiBars 3 --batteryState charged --batteryLevel 100
```

> `9:41` 是 Apple 自己宣传图里的惯例时间，用它最不扎眼。

---

## 截图里该出现什么

四个标签页各一张，按这个顺序（App Store 上第一张最重要，多数人只看第一张）：

| # | 页面 | 画面里要有 |
| --- | --- | --- |
| 1 | **Overview** | 一个像样的净资产数字、本月已有支出与收入、"Where it went" 至少 4 根分类条、"Recent" 5 行都填满 |
| 2 | **Accounts** | **四种类型各至少一个账户** —— 这是本 App 的卖点，要一眼看到 Card / E-wallet / Cash / Bank 四种图标并列。其中 Card 账户余额为**负数并标红** |
| 3 | **Transactions** | 至少跨 3 天的流水，含一条 **Transfer**（能看出两个账户之间的箭头），每天的分组头上有当天净额 |
| 4 | **Settings** | 币种、Export as CSV、Privacy policy 三行 + 下方那句「数据只存在本机」的说明 |

**演示数据要手打进去，不要改代码造假。** 大约 15～20 笔、跨 3～5 天、分布在 5～6 个分类上
就够了。理由和其他包一样：手打一遍顺带就把功能验了 —— 尤其第 3 张里那笔 Transfer，
截图的同时正好确认「净资产不变、不计入本月支出」这条口径是对的。

金额用**目标市场看着合理的数字**（默认币种是 PHP），别出现 `1234567.89` 这种一眼假的数。

> ⚠️ **别把真实的个人财务数据截进去。** 演示数据就用编的。

---

## App 图标 —— 已就位

`PocketLedger/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png`
1024×1024，**24 位无 alpha**（iOS 图标带透明通道会被 App Store 拒收）。

Xcode 14 起 App Icon 只需要这一张 1024 的源图，其余尺寸由工具链生成 ——
`Contents.json` 里也确实只声明了一个 `universal / ios / 1024x1024` 条目，是对的。

### 设计参数（需要重画时照着来）

图案是**品牌蓝渐变底 + 三根递增的白色圆角柱 + 一条基线**，纯几何、不依赖字体渲染
（不用字体的原因：跨机器渲染结果不一致，而且中文/西文字体的授权也是个麻烦）。

| 元素 | 参数 |
| --- | --- |
| 底色 | 品牌蓝渐变，主色 **`#2F6BFF`**（= `AppTheme.accent`，也是 `AccentColor.colorset` 里那个 sRGB 0.184 / 0.420 / 1.000） |
| 前景 | 三根**白色**圆角柱，高度递增（左矮右高），下方一条白色基线 |
| 圆角 | 柱子自身带圆角；**整张图不要预先切圆角** —— iOS 会自己套 squircle 遮罩，自己切会露出双重圆角 |
| 边距 | 图形四周留白，别顶边 |
| 通道 | **必须 24 位 RGB 无 alpha** |

> 生成脚本**没有进仓库**：那是个一次性的 Dart 脚本，而本包是纯 Swift 工程，
> 塞个 Dart 包进来不合适。需要改时照上面的参数重画即可。
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

> 建议**首发不做**。一个静态记账界面拍成 15 秒视频，价值不高，而制作与审核成本实打实。
> 真要做，`xcrun simctl io booted recordVideo preview.mov` 录一段，
> 内容就演「建一个 E-wallet 账户 → 记一笔支出 → 记一笔 Transfer → 回 Overview 看数字变了」。

---

## 隐私政策托管、Support URL 与支持邮箱 —— 待做

`store/privacy-policy.html` 是可直接托管的独立页面（**无外链、自带深浅色适配、手机可读**），
内容与 `PRIVACY_POLICY.md` 一致。丢到任意免费静态托管
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
   与本包单独用 `stillwater` 命名空间的初衷直接矛盾。
   （同类顾虑还有一个更硬的：**Seller 名称**。见 `RELEASE_SIGNING.md` 第 1 节。）

定好之后要同步替换的**四处**：

1. `APP_STORE_LISTING.md` 的 URL 表
2. `PRIVACY_POLICY.md` 的 Contact 段
3. `store/privacy-policy.html` 的 Contact 段与页脚
4. `PocketLedger/Core/Services/UserSettingsStore.swift` 的
   `privacyPolicyURLString` 与 `supportEmail`
   —— **改这一处需要重新出构建版本**，别留到最后

---

## 完整待做清单（本文范围内）

- [ ] 在 Mac 上编译通过并跑起来（**一切的前提**）
- [ ] 手打 15～20 笔演示数据，覆盖四种账户类型与一笔 Transfer
- [ ] 出 4 张 **6.9"** 截图（1320×2868 或 1290×2796）
- [ ] 出 4 张 **6.5"** 截图（1284×2778 或 1242×2688）
- [ ] 用 `sips` 核对每一张的尺寸 / 无 alpha / RGB
- [ ] 托管 `store/privacy-policy.html`，外部验证可访问
- [ ] 定 Support URL
- [ ] 定支持邮箱并发测试邮件验证
- [ ] 四处 TODO 全部替换（含需要重新出包的那一处）
- [ ] 归档后确认 `.ipa` 里的图标仍是无 alpha
