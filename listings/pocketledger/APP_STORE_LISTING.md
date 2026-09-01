# App Store Listing — PocketLedger

> 本文是 App Store Connect「App 信息 / 版本信息」页要填的全部文案与选项。
> **英文块可直接粘进 App Store Connect；中文引用块是内部说明，不要粘进去。**
>
> 注意：这是 **App Store** 的表单，字段与 Google Play 完全不同（没有「短描述」，
> 多了副标题 / 宣传文本 / 关键词 / 年龄分级问卷 / 出口合规）。
> 不要照抄仓库里其余 Android 包的 `PLAY_STORE_LISTING.md` 字段。

## 基本事实（填表时对照用）

| 项 | 值 |
| --- | --- |
| Bundle ID | `com.stillwater.pocketledger` |
| 平台 | iOS only（无 iPadOS、无 macOS、无 visionOS） |
| 设备 | 仅 iPhone（`TARGETED_DEVICE_FAMILY = 1`，`SUPPORTS_MACCATALYST = NO`） |
| 最低系统 | iOS 15.6 |
| 方向 | 竖屏锁定（`UISupportedInterfaceOrientations = Portrait`） |
| 版本 / 构建号 | `MARKETING_VERSION = 1.0` / `CURRENT_PROJECT_VERSION = 1` |
| 价格 | Free（无内购、无订阅） |

---

## App Name（≤ 30 字符）

```
PocketLedger
```

> 12 字符。若 App Store 上重名被占，备选：
>
> ```
> PocketLedger: Expense Tracker
> ```
>
> 29 字符（含空格），仍在 30 以内。
> 不要退成 `Expense Tracker` 这类纯通用词 —— 既搜不到，也容易被 2.3.7（名称误导）挑。

## Subtitle（≤ 30 字符）

```
Track cards, wallets and cash
```

> 29 字符。副标题在搜索里参与索引，所以这里把三个账户类型的词都放进去了，
> 而不是写一句纯口号。备选（26 字符）：`Cards, wallets, cash, bank`。

## Promotional Text（≤ 170 字符）

```
Give every card, e-wallet, cash stash and bank account its own balance. Log expenses, income and transfers between them. Works offline, with no ads and no sign-in.
```

> 163 字符。宣传文本**不需要重新提交审核就能改**，适合以后放「新增了什么」。
> 首发就用这段说清卖点即可。

## Description（≤ 4000 字符）

```
PocketLedger is a plain, private money tracker for your iPhone. You build your own list of accounts, decide what each one actually is, and every entry you record lands on the right balance.

PICK WHAT EACH ACCOUNT IS

When you add an account you choose its type yourself:

- Card — a credit or debit card. The balance is allowed to go negative when you owe, and it turns red so you can see it at a glance.
- E-wallet — a digital wallet or mobile payment app, kept as its own balance instead of being folded into your bank.
- Cash — the notes in your pocket or at home.
- Bank account — a savings or checking account.

Keeping a card and an e-wallet apart is the whole point. Topping up a wallet from your bank card is not spending, and a card balance you still owe is not money you have. PocketLedger treats them as separate places your money sits, so the numbers match what is really there.

WHAT YOU CAN RECORD

- Expenses and income, each on a specific account, with a category, a date and a note.
- Transfers between two of your accounts — from a bank card to an e-wallet, from cash into the bank. A transfer moves the money and leaves your net balance untouched, because nothing was spent.
- 14 categories are ready on first launch, ten for spending and four for income, so you can record your first entry immediately.

WHAT YOU SEE

Overview
- Net balance across every account you keep.
- What you spent and what you received this month.
- Where it went: your top five spending categories this month, with a bar for each.
- Your last five entries.

Transactions
- Every entry, newest first, grouped by day, with each day's net figure on its header.

Accounts
- Every account with its type, colour and current balance, and the total underneath.

Settings
- Switch between 17 currencies. Every amount in the app changes with it.
- Export everything as a CSV file and send it wherever you like — mail, Files, a spreadsheet.
- Erase all data, with a confirmation first.

WHERE YOUR DATA LIVES

Your accounts and entries are kept in the app's own storage on this iPhone. They are never uploaded, never synced to an account, and never shared with anyone — the only way anything leaves the device is the CSV export, which you start yourself and send where you choose. Deleting the app deletes the ledger with it.

There is no sign-up and no login, so there is nothing to remember and nothing to lose. The app opens straight into your Overview and works with no connection at all.

- No ads.
- No in-app purchases and no subscription.
- No account, no password, no cloud.
- No leaderboards and no social features.

Portrait, one-handed, and quiet about it.
```

> 2 668 字符（上限 4 000），留足余量。写作口径与其余上架包一致：讲清做什么、不夸功能、
> 把「无广告 / 无内购 / 无账号 / 离线」放在末尾成组说明。
>
> **不要**在这段里出现任何内部字眼（网关、AB、渠道、落地页），也不要提及其他上架包的名字。
> 文案里的每一条都能在源码里找到出处：
> 四种账户类型 `Core/Models/LedgerModels.swift:AccountKind`；14 个默认分类
> `DefaultCategories.make()`（10 支出 + 4 收入）；转账不计收支
> `Core/Models/LedgerMath.swift:total(kind:...)`；Top5 与最近 5 笔
> `MVP/Overview/OverviewModule.swift`（`totals.prefix(5)` / `prefix(5)`）；17 种币种
> `Core/Services/UserSettingsStore.swift:supportedCurrencies`；CSV 导出与清空
> `MVP/Settings/SettingsModule.swift`；本地 JSON 落盘 `Core/Services/LedgerStore.swift`。
> 改了功能就回来改这段。
>
> 另：本包是**浅色主题**（`Core/Theme/AppTheme.swift`），所以描述里**没有**写「深色模式」。
> 别顺手加上。

## Keywords（≤ 100 字符，逗号分隔、**不加空格**）

```
expense,budget,spending,money,finance,wallet,cash,ewallet,ledger,tracker,offline,account,CSV,daily
```

> 98 字符。规则提醒：
> - 逗号后**不要加空格**，空格会占掉配额。
> - 不要重复 App 名称与副标题里已有的词（Apple 已经索引了），也不要写分类名
>   （`Finance` 由类目本身覆盖）。
> - 不要写竞品名或商标词 —— 会被 2.3.7 驳回。

## 类目

| 项 | 值 |
| --- | --- |
| Primary Category | **Finance** |
| Secondary Category | **Productivity** |

> 主类目选 Finance：App 的全部功能都是记录与查看自己的收支。
> 次类目 Productivity 而不是 Utilities —— 记账在 App Store 上更常与待办 / 笔记类工具同列。
>
> ⚠️ Finance 类目会触发审核对**金融属性**的额外注意（Guideline 3.2.1）。本 App
> **不接触任何真实资金**：不连银行、不读短信、不做支付、不给投资建议，只是手工记录数字。
> 这一点已经写进 `APP_REVIEW_NOTES.md`，提交时务必带上。

## 年龄分级（Age Rating 问卷）

> App Store Connect 的分级问卷现行档位是 **4+ / 9+ / 13+ / 16+ / 18+**。
> 本 App 全部内容项应答 **None**，结果落在 **4+**。

| 问卷项 | 答案 | 理由 |
| --- | --- | --- |
| Cartoon or Fantasy Violence | None | 无任何暴力内容 |
| Realistic Violence / Prolonged Violence | None | 同上 |
| Sexual Content or Nudity | None | 无 |
| Profanity or Crude Humor | None | 界面文案全部为中性功能描述 |
| Alcohol, Tobacco, or Drug Use or References | None | 无 |
| Mature/Suggestive Themes | None | 无 |
| Horror/Fear Themes | None | 无 |
| Medical/Treatment Information | None | 无 |
| Simulated Gambling | None | **无**。App 内没有任何博彩、抽奖或赌博玩法 |
| Contests | None | 无 |
| Unrestricted Web Access | **No** | 记账本体不含浏览器、不含任意网页入口 |
| Does the app contain user-generated content? | **No** | 用户输入的账目只存在本机，不发布、不共享、不与他人可见 |
| Does the app include messaging or chat? | **No** | 无任何用户间通信 |
| Does the app include advertising? | **No** | 无广告位、无广告 SDK 展示位 |
| In-app controls / parental gate needed? | **No** | 无需要限制的功能 |
| Age Assurance / age-restricted features | None | 无 |

> 三点要注意：
> 1. **Simulated Gambling 必须答 None，并且实现上必须真的没有。** 记账 App 一旦被
>    发现有赌博相关内容，是 4.3 / 5.3 的直接下架项，不是补交材料能救的。
> 2. **Unrestricted Web Access 是按「App 自身功能」回答的** —— 记账本体确实没有
>    任何可打开任意网址的入口（设置页的 Privacy policy 行调 `UIApplication.open`
>    打开的是**我们自己托管的政策页**，是固定单一 URL，不是浏览器）。
>    若后续在 App 内新增任何能打开任意网页的界面，**必须回到这张表重答此项**。
> 3. 分级问卷答案与实现不符是常见的下架原因。改功能后回来复核。

## 出口合规（Export Compliance）

| 问题 | 答案 |
| --- | --- |
| Does your app use encryption? | **No** |
| `ITSAppUsesNonExemptEncryption` | 已在 `Info.plist` 里写死 `false` |

> 口径：App 自身不实现、不包含任何加密算法；用到的只有 iOS 系统与 SDK 提供的
> **标准 HTTPS/TLS**，属于 Apple 明确列出的豁免项。因此 `ITSAppUsesNonExemptEncryption = false`
> 是正确答案，且因为 plist 里已经写死，**每次上传构建版本时 App Store Connect 不会再问**。
>
> 若以后自己实现了任何加密（例如给本地账本加密、自签名请求），这个答案就不再成立，
> 必须改 plist 并重新走出口合规申报（可能需要 ERN/CCATS）。

## IDFA / App Tracking Transparency

| 项 | 状态 |
| --- | --- |
| `NSUserTrackingUsageDescription` | ✅ 已在 `Info.plist` 里 |
| ATT 弹窗调用（`ATTrackingManager.requestTrackingAuthorization`） | ❌ **代码里没有**，见下 |
| 上传构建版本时的「Does this app use the Advertising Identifier (IDFA)?」 | 见下 |

> **这是本包目前最需要拍板的一处，必须在提交前解决。**
>
> 现状：`Info.plist` 里有 ATT 文案，但全工程 grep 不到 `ATTrackingManager` /
> `AppTrackingTransparency`（`GateCoordinator` 里只调 `TrackingService.shared.start()`）。
> 也就是说 **App 目前不会弹 ATT 授权框**，AppsFlyer / Adjust 拿不到 IDFA。
>
> 两条路，二选一，并与 `APP_PRIVACY.md` 的申报保持一致：
>
> - **路线 A（推荐，与申报口径一致）**：在 App 里补上 ATT 请求
>   （`import AppTrackingTransparency`，在界面呈现之后调
>   `ATTrackingManager.requestTrackingAuthorization`），IDFA 问卷答 **Yes**
>   （用途勾 "Attribute this app installation to a previously served advertisement"
>   与 "Attribute an action taken within this app to a previously served advertisement"，
>   并勾 "I confirm that this app... uses the App Tracking Transparency framework"）。
>   **不补 ATT 就答 Yes 会被 5.1.2 驳回。**
> - **路线 B**：暂不启用归因（保留 `TODO_*` 占位符不填），IDFA 问卷答 **No**，
>   并把 `APP_PRIVACY.md` 里的 "Data Used to Track You" 一并改成 No。
>   注意这条路线下 `NSUserTrackingUsageDescription` 留着无害，但**申报必须同步收窄**。
>
> 不要出现「plist 有文案 + 问卷答 Yes + 代码不弹框」这种三方打架的状态。

## URL 与联系方式

| 字段 | 值 | 状态 |
| --- | --- | --- |
| Support URL（必填） | `TODO_SUPPORT_URL` | ❌ 待定 |
| Marketing URL（选填） | `TODO_MARKETING_URL`（可留空） | ❌ 待定 / 可不填 |
| Privacy Policy URL（必填） | `TODO_PRIVACY_POLICY_URL` | ❌ 待定，页面已写好待托管 |
| App Review 联系邮箱 | `TODO_SUPPORT_EMAIL` | ❌ 待定 |

> - **Support URL 是 App Store 必填项，且会公开显示在商品页上。** 可以就是一个静态页面，
>   上面写清 App 是做什么的 + 一个可达的联系邮箱即可，不需要工单系统。
> - **Marketing URL 是选填的，拿不准就留空** —— 填一个空壳站点反而多一个可被关联的落点。
> - **Privacy Policy URL 必填**，内容用本包的 `store/privacy-policy.html`
>   （托管步骤与验收见 `STORE_ASSETS.md`）。
> - ⚠️ **三个 URL 与那个邮箱都不要复用其他上架包已经在用的。** 政策 URL 与支持 URL
>   都公开显示在 App Store 商品页上，两个 App 指向同一个域名 / 同一个邮箱，等于在
>   商店侧把它们公开关联起来 —— 与本包单独取 `stillwater` 命名空间的初衷直接冲突。
>   各自新建。细则见 `STORE_ASSETS.md` 末尾。
>
> 定好之后要同步替换的地方（**四处，别漏**）：
> 1. 本文这张表
> 2. `PRIVACY_POLICY.md` 的 Contact 段
> 3. `store/privacy-policy.html` 的 Contact 段与页脚
> 4. `PocketLedger/Core/Services/UserSettingsStore.swift` 的
>    `privacyPolicyURLString` 与 `supportEmail`（现在是 `TODO_PRIVACY_POLICY_URL`
>    / `TODO_SUPPORT_EMAIL`；不改的话设置页点「Privacy policy」只会弹一句
>    "The privacy policy link is not set up yet."）—— **改这一处需要重新出构建版本。**

## 卖家名称（Seller / 开发者名称）—— 待运营拍板

> App Store 商品页上的 **Seller 是公开的**，取自 Apple Developer 账号的法定名称。
> 本包若与 `decktallypro` 用同一个 Apple Developer 账号，两个 App 的 Seller 会**完全一致**，
> 任何人点开都能看出同属一家 —— 这与「各包使用互不相关的厂商命名空间」的初衷直接冲突。
>
> `project.pbxproj` 里的 `DEVELOPMENT_TEAM` 现在是**空的**，就是在等这个决定。
> 要么接受这层公开关联，要么另开一个开发者账号。详见 `README_GATE.md` §5 与
> `RELEASE_SIGNING.md`。

## App Store Connect 上还需要填、但不在本文范围的

- **App Privacy（隐私营养标签）** → 见 `APP_PRIVACY.md`
- **App Review Information（审核备注 / 演示账号）** → 见 `APP_REVIEW_NOTES.md`
- **截图与预览** → 见 `STORE_ASSETS.md`（**目前一张都没有**）
- **本地化**：首发只做 **English (U.S.)** 一种。App 界面文案全是英文硬编码，
  没有 `.strings` 本地化文件，多加一种语言只会得到一个中英混排的商品页。
