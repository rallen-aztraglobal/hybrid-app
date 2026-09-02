# App Store Listing — GridSlide

> 本文是 App Store Connect「App 信息 / 版本信息」页要填的全部文案与选项。
> **英文块可直接粘进 App Store Connect；中文引用块是内部说明，不要粘进去。**
>
> 注意：这是 **App Store** 的表单，字段与 Google Play 完全不同（没有「短描述」，
> 多了副标题 / 宣传文本 / 关键词 / 年龄分级问卷 / 出口合规）。
> 不要照抄仓库里其余 Android 包的 `PLAY_STORE_LISTING.md` 字段。
>
> ⚠️ 本包是 **Games › Puzzle** 类目的休闲游戏，不是工具类。仓库里同批次的 iOS 上架包
> `pocketledger` 是 Finance 类目的记账工具 —— 它的类目、关键词、年龄分级答案、
> 描述结构**一条都不能照搬**，只有文档骨架是共用的。

## 基本事实（填表时对照用）

| 项 | 值 |
| --- | --- |
| Bundle ID | `com.fernvale.gridslide` |
| 平台 | iOS only（无 iPadOS、无 macOS、无 visionOS） |
| 设备 | 仅 iPhone（`TARGETED_DEVICE_FAMILY = 1`，`SUPPORTS_MACCATALYST = NO`） |
| 最低系统 | iOS 15.6 |
| 方向 | 竖屏锁定（`INFOPLIST_KEY_UISupportedInterfaceOrientations = UIInterfaceOrientationPortrait`） |
| 主题 | **深色**（`Core/Theme/AppTheme.swift`，背景 `#101418`，强调色 `#4CC2A0`） |
| 版本 / 构建号 | `MARKETING_VERSION = 1.0` / `CURRENT_PROJECT_VERSION = 1` |
| 价格 | Free（无内购、无订阅） |

---

## App Name（≤ 30 字符）

```
GridSlide
```

> 9 字符。若 App Store 上重名被占，备选：
>
> ```
> GridSlide: Number Puzzle
> ```
>
> 24 字符（含空格），仍在 30 以内。
> 不要退成 `Slide Puzzle` / `Number Puzzle` 这类纯通用词 —— 既搜不到，
> 也容易被 2.3.7（名称误导 / 关键词堆砌）挑。

## Subtitle（≤ 30 字符）

```
Slide the numbers into order
```

> 28 字符。副标题在搜索里参与索引，所以这里放的是玩法本身的词，而不是一句口号。
> 备选（同为 28 字符）：`3x3, 4x4 and 5x5 tile puzzle`。
>
> ⚠️ 选定哪一句，下面的关键词就要避开哪一句里已有的词（Apple 已经索引了副标题）。
> 下面那份关键词是按**第一句**写的，所以里面没有 `slide` / `number`。

## Promotional Text（≤ 170 字符）

```
Three board sizes, whole-row sliding, and separate best-moves and best-time records. No ads, no in-app purchases, no sign-in, and it works with no connection.
```

> 158 字符。宣传文本**不需要重新提交审核就能改**，适合以后放「新增了什么」。
> 首发就用这段说清四个真实卖点即可。

## Description（≤ 4000 字符）

```
GridSlide is a number slide puzzle for iPhone. The tiles start scrambled; you slide them back into order, from 1 up to the last one, with the empty square ending in the bottom-right corner.

THREE BOARD SIZES

- 3x3 - eight tiles. A minute or two, and the size to learn the moves on.
- 4x4 - fifteen tiles. The classic form of this puzzle.
- 5x5 - twenty-four tiles. A long, deliberate solve.

Switch size at the top of the Play screen, or set the one you want as your default in Settings. Each size keeps its own records.

SLIDE A WHOLE ROW AT ONCE

Tap any tile that shares a row or a column with the empty square and it slides in. Tap one that is several places away and every tile between it and the gap moves along with it, in one gesture, counted as a single move. You never have to tap your way across the board one square at a time.

Tiles that are already sitting in their finished position turn the accent colour, so you can see how much of the board is solved without reading the numbers.

TWO RECORDS, KEPT APART

Every solve is timed and counted, and your fewest moves and your fastest time are stored separately, for each board size. A careful run that finds a short solution and a quick run that hurries are both worth something, and neither overwrites the other. Finish a board and a summary shows the moves and the time you took, and tells you which record you just beat, if any.

The Records tab puts all three sizes side by side: best moves, best time, and how many times you have solved that size. A size you have not finished yet shows a dash rather than a zero.

AN HONEST CLOCK

Timing starts on your first move, not when the app opens, so a board left sitting on screen costs you nothing. Leave the app or switch tabs and the clock stops; come back and it picks up where it left off. What it measures is the time you actually spent playing.

SETTINGS

- Your default board size.
- Haptic feedback, on or off.
- Reset all records, after a confirmation.

OFFLINE, AND QUIET ABOUT IT

- No ads, anywhere.
- No in-app purchases and no subscription.
- No account, no sign-in and no third-party login.
- No leaderboards, no profiles and no social features.
- The game itself never needs a connection.

Your records are kept on this iPhone, in the app's own storage. They are not uploaded, not synced to anything, and not shared with anyone. Deleting the app deletes them with it.

Portrait, dark, one-handed, and iPhone only.
```

> 约 2 440 字符（上限 4 000），留足余量。写作口径与其余上架包一致：讲清做什么、不夸功能、
> 把「无广告 / 无内购 / 无账号 / 离线」放在末尾成组说明。
>
> **不要**在这段里出现任何内部字眼（网关、AB 面、B 面、渠道、落地页），
> 也不要提及其他上架包的名字或 bundleId。
>
> 文案里的每一条都能在源码里找到出处（改了功能就回来改这段）：
>
> | 文案里的说法 | 出处 |
> | --- | --- |
> | 3×3 / 4×4 / 5×5 三种尺寸 | `Core/Services/SettingsStore.swift:boardSizes = [3, 4, 5]` |
> | 默认 4×4 | 同上，`boardSize` 的 getter 兜底值 |
> | 同行/同列可滑、斜向不可 | `Core/Models/SlidePuzzle.swift:canMove` |
> | **整排滑动、只算一步** | `SlidePuzzle.move`（空格朝目标方向逐格吞并）+ `PlayModule.boardViewDidMove`（每次合法点击 `moves += 1`） |
> | 归位的块变强调色 | `SlidePuzzle.isTileInPlace` + `AppTheme.tilePlacedFill` |
> | **步数与时间各记各的、按尺寸分开** | `Core/Services/RecordStore.swift:submit`（`newMovesBest` 与 `newTimeBest` 相互独立）；`recordKey(for:)` 按尺寸分键 |
> | 结算浮层显示破纪录情况 | `PlayModule.WinOverlayView.configure`（"New best moves and time" / "New best moves" / "New best time"） |
> | Records 页三张卡、未通关显示破折号 | `MVP/Records/RecordsModule.swift:RecordCardView.configure` |
> | 第一步才开始计时 | `PlayModule.boardViewDidMove`（`if moves == 0 { clock.start() }`） |
> | 切后台 / 切标签页暂停计时 | `PlayModule.appDidEnterBackground` / `viewWillDisappear` + `GameClock`（按时间戳累计） |
> | 三项设置 | `MVP/Settings/SettingsModule.swift`（Board size / Haptics / Reset all records） |
> | 成绩只存本机 | `RecordStore`（UserDefaults 键 `gsl.records`） |
>
> 另：本包是**深色主题**（`AppTheme.background = #101418`），所以描述里写了 "dark"。
> 若以后改成浅色或加了主题切换，这一句要跟着改。

## Keywords（≤ 100 字符，逗号分隔、**不加空格**）

```
tile,tiles,15,fifteen,brain,logic,offline,minimal,solve,timer,moves,classic,board,relax
```

> 87 字符。规则提醒：
> - 逗号后**不要加空格**，空格会占掉配额。
> - **不要重复 App 名称与副标题里已有的词** —— `grid` / `slide` 在名称里，
>   `number` / `order` 在副标题里，Apple 都已经索引了，再写一遍是浪费配额。
>   （所以这份关键词里一个都没有。换副标题就要回来重排。）
> - **不要写分类名** —— `game` / `puzzle` 由 Games › Puzzle 这个类目本身覆盖。
> - 不要写竞品名或商标词（这个玩法有几个很有名的商业化实现，**一个都不能写**）——
>   会被 2.3.7 直接驳回。
> - `15` 与 `fifteen` 都留着：这个玩法的通用叫法（十五数码 / fifteen puzzle）在两种写法上
>   都有搜索量，而它们是**玩法名称**、不是任何人的商标。

## 类目

| 项 | 值 |
| --- | --- |
| Primary Category | **Games** |
| Primary Subcategory 1 | **Puzzle** |
| Primary Subcategory 2 | **Board** |
| Secondary Category | **Entertainment**（可留空） |

> - 主类目 Games、第一子类目 Puzzle：这就是一个滑块拼图，没有别的解读。
>   第二子类目选 Board 而不是 Strategy —— 它是在一块固定棋盘上按规则挪子，
>   够不上策略游戏那一档，选错子类目会把 App 放进一堆完全不同的产品中间。
> - 次类目 Entertainment 是选填的，**拿不准就留空**；留空不影响审核。
> - ⚠️ **Games 类目下不要出现任何博彩暗示** —— 玩法、图标、截图、文案里都不能有
>   老虎机、转盘、筹码、赔率、"jackpot" / "bet" / "win big" 这类词。本 App 本身
>   完全没有这些东西，注意别在做截图和宣传语时顺手加。相关口径见下面的年龄分级一节
>   与 `APP_REVIEW_NOTES.md`。

## 年龄分级（Age Rating 问卷）

> App Store Connect 的分级问卷现行档位是 **4+ / 9+ / 13+ / 16+ / 18+**。
> 本 App 全部内容项应答 **None**，结果落在 **4+**。

| 问卷项 | 答案 | 理由 |
| --- | --- | --- |
| Cartoon or Fantasy Violence | None | 无任何暴力内容。屏幕上只有编号方块 |
| Realistic Violence | None | 同上 |
| Prolonged Graphic or Sadistic Realistic Violence | None | 同上 |
| Sexual Content or Nudity | None | 无 |
| Graphic Sexual Content and Nudity | None | 无 |
| Profanity or Crude Humor | None | 界面文案只有 Play / Records / Settings / Moves / Time / Best / Solved 这类功能词 |
| Mature/Suggestive Themes | None | 无 |
| Horror/Fear Themes | None | 无 |
| Alcohol, Tobacco, or Drug Use or References | None | 无 |
| Medical/Treatment Information | None | 无 |
| **Simulated Gambling** | **None** | **见下** |
| **Gambling**（真实赌博 / 博彩） | **No** | **见下** |
| **Contests** | **None** | **见下** |
| Unrestricted Web Access | **No** | 游戏本体不含浏览器、不含任意网页入口 |
| Does the app contain user-generated content? | **No** | 玩家只产生「步数与用时」两个数字，存在本机，不发布、不共享、不与他人可见 |
| Does the app include messaging or chat? | **No** | 无任何用户间通信 |
| Does the app include advertising? | **No** | **无广告位、无广告展示 SDK** |
| In-app controls / parental gate needed? | **No** | 无需要限制的功能，无内购 |
| Age Assurance / age-restricted features | None | 无 |

### 博彩相关三项 —— 逐条说明（Games 类目必须答得干净）

> 这三项是 Games 类目下最容易被追问、也是答错代价最大的。本 App 是**纯拼图，
> 无任何博彩元素**，逐条把理由写死在这里，免得日后有人拿不准：
>
> | 项 | 答案 | 为什么 |
> | --- | --- | --- |
> | **Simulated Gambling**（模拟赌博） | **None** | App 内**没有**老虎机、轮盘、扑克、宾果、刮刮乐、抽奖转盘或任何以随机结果决胜负的玩法。唯一用到随机数的地方是开局打乱棋盘（`SlidePuzzle.shuffle`），那是**发牌式的初始局面生成**，不是赌局：打乱之后的每一步都由玩家决定，结果完全取决于操作，与随机无关。没有虚拟筹码、没有下注、没有赔付、没有「再来一次」的付费重试 |
> | **Gambling**（真实赌博 / 现金博彩） | **No** | 不涉及任何真实金钱。App 里没有支付、没有内购、没有货币、没有可兑换的物品，也不链接任何博彩站点 |
> | **Contests**（竞赛 / 抽奖） | **None** | 没有排行榜、没有比赛、没有奖品、没有报名、没有他人可见的成绩。成绩只存在这台设备的 UserDefaults 里，玩家自己看 |
>
> ⚠️ **这三项必须答 None/No，并且实现上必须真的没有。** 一个 Games 类目的 App
> 被发现有未申报的博彩或模拟博彩内容，是 Guideline 5.3 / 4.7 的直接下架项，
> 不是补交材料能救的。**如果以后加了任何转盘、抽奖、每日奖励开箱之类的东西，
> 必须回到这张表重答，并同步改 `APP_REVIEW_NOTES.md`。**

> 另外两点：
> 1. **Unrestricted Web Access 是按「App 自身功能」回答的** —— 游戏本体确实没有
>    任何可打开任意网址的入口（设置页的 Privacy policy 行调 `UIApplication.open`
>    打开的是**我们自己托管的政策页**，是固定单一 URL，不是浏览器）。
>    若后续在 App 内新增任何能打开任意网页的界面，**必须回到这张表重答此项**。
> 2. 分级问卷答案与实现不符是常见的下架原因。改功能后回来复核。

## 出口合规（Export Compliance）

| 问题 | 答案 |
| --- | --- |
| Does your app use encryption? | **No** |
| `ITSAppUsesNonExemptEncryption` | 已在 `GridSlide/Info.plist` 里写死 `false` |

> 口径：App 自身不实现、不包含任何加密算法；用到的只有 iOS 系统与 SDK 提供的
> **标准 HTTPS/TLS**，属于 Apple 明确列出的豁免项。因此 `ITSAppUsesNonExemptEncryption = false`
> 是正确答案，且因为 plist 里已经写死，**每次上传构建版本时 App Store Connect 不会再问**。
>
> 若以后自己实现了任何加密，这个答案就不再成立，必须改 plist 并重新走出口合规申报。

## IDFA / App Tracking Transparency

| 项 | 状态 |
| --- | --- |
| `NSUserTrackingUsageDescription` | ✅ 已在 `GridSlide/Info.plist` 里 |
| ATT 弹窗调用（`ATTrackingManager.requestTrackingAuthorization`） | ❌ **代码里没有**，见下 |
| AppsFlyer / Adjust 当前是否真的在跑 | ❌ **都没有** —— 两个 key 都还是占位符 |
| 上传构建版本时的「Does this app use the Advertising Identifier (IDFA)?」 | 见下 |

> **这是本包提交前必须拍板的一处，口径与 `README_GATE.md` §3 的提醒一致。**
>
> 现状：`Info.plist` 里有 ATT 文案，但全工程没有 `ATTrackingManager` /
> `AppTrackingTransparency`（`Gate/GateCoordinator` 只调 `TrackingService.shared.start()`）。
> 而 `Core/Services/TrackingService.swift` 里两个 SDK 的初始化都被
> `GateConfig.isConfigured(...)` 挡着，`appsFlyerAppleAppID` 与 `adjustAppToken`
> 现在分别是 `TODO_APPSTORE_APP_ID` 与 `TODO_ADJUST_APP_TOKEN` ——
> **也就是说当前构建里两个归因 SDK 一个都不会初始化，更拿不到 IDFA。**
>
> 两条路，二选一，并与 `APP_PRIVACY.md` 的申报保持一致：
>
> - **路线 A（与 `APP_PRIVACY.md` 的默认申报口径一致）**：在 App 里补上 ATT 请求
>   （`import AppTrackingTransparency`，在界面呈现之后、**不要在冷启动第一屏**调
>   `ATTrackingManager.requestTrackingAuthorization`），并把两个 key 填上，
>   IDFA 问卷答 **Yes**（用途勾 "Attribute this app installation to a previously
>   served advertisement" 与 "Attribute an action taken within this app to a
>   previously served advertisement"，并勾 "I confirm that this app... uses the
>   App Tracking Transparency framework"）。
>   **不补 ATT 就答 Yes 会被 5.1.2 驳回。**
> - **路线 B（首发省事）**：暂不启用归因（`TODO_*` 占位符原样留着，也就是**当前状态**），
>   IDFA 问卷答 **No**，并把 `APP_PRIVACY.md` 里的 "Data Used to Track You"
>   一并改成 No。这条路线下 `NSUserTrackingUsageDescription` 留在 plist 里无害
>   （没调 ATT 就不会用到它），但**申报必须同步收窄**。
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
>   商店侧把它们公开关联起来 —— 与本包单独取 `fernvale` 命名空间的初衷直接冲突。
>   **各自新建**，细则见 `STORE_ASSETS.md` 末尾。
>
> 定好之后要同步替换的地方（**四处，别漏**）：
> 1. 本文这张表
> 2. `PRIVACY_POLICY.md` 的 Contact 段
> 3. `store/privacy-policy.html` 的 Contact 段与页脚
> 4. `GridSlide/Core/Services/SettingsStore.swift` 的
>    `privacyPolicyURLString` 与 `supportEmail`（现在是 `TODO_PRIVACY_POLICY_URL`
>    / `TODO_SUPPORT_EMAIL`；不改的话设置页点「Privacy policy」只会弹一句
>    "The privacy policy link is not set up yet."）—— **改这一处需要重新出构建版本。**

## 卖家名称（Seller / 开发者名称）—— 待运营拍板

> App Store 商品页上的 **Seller 是公开的**，取自 Apple Developer 账号的法定名称。
> 本包若与仓库里其余 iOS 上架包用同一个 Apple Developer 账号，几个 App 的 Seller 会
> **完全一致**，任何人点开都能看出同属一家 —— 这与「各包使用互不相关的厂商命名空间」
> 的初衷直接冲突。
>
> `GridSlide.xcodeproj/project.pbxproj` 里的 `DEVELOPMENT_TEAM` 现在是**空的**，
> 就是在等这个决定。要么接受这层公开关联，要么另开一个开发者账号。
> 详见 `README_GATE.md` §5 与 `RELEASE_SIGNING.md` 第 1 节。

## App Store Connect 上还需要填、但不在本文范围的

- **App Privacy（隐私营养标签）** → 见 `APP_PRIVACY.md`
- **App Review Information（审核备注 / 演示账号）** → 见 `APP_REVIEW_NOTES.md`
- **截图与预览** → 见 `STORE_ASSETS.md`（**目前一张都没有，因为本工程从未编译过**）
- **Game Center**：**不接入**。本 App 没有排行榜、没有成就、没有多人对战，
  App Store Connect 里那一栏保持关闭。（成绩只存本机，见 `RecordStore`。）
- **本地化**：首发只做 **English (U.S.)** 一种。App 界面文案全是英文硬编码，
  没有 `.strings` 本地化文件，多加一种语言只会得到一个中英混排的商品页。
