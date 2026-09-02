# TileFit — AB 面网关接入说明

本工程（Flutter，**仅 Android**，包名 `com.emberlane.tilefit8264`）已接入上架包 AB 面网关
（见 [ADR-0014](../../docs/adr/0014-listing-ab-gate.md) / [docs/admin/09-listing.md](../../docs/admin/09-listing.md)）。
启动即向渠道中台判定 A/B：A 面进游戏本体（`GameScreen`），
B 面按 `openMode` 内开 `WebScreen` 或外开系统浏览器。判定失败一律 A 面。

接入方式与 [colorstack](../colorstack/README_GATE.md) / [calcpad](../calcpad/README_GATE.md)
完全同构，仅 A 面入口不同（本包是 `GameScreen`，calcpad 是 `CalculatorScreen`，
colorstack 是 `HomeScreen`）。

## A 面是什么

TileFit 是一个 8×8 的方块拼放游戏：每轮发 3 个形状，拖到棋盘上；填满整行或整列即消除；
三个用完补发新的一手；剩下的形状在棋盘上都放不下时本局结束。纯离线、无广告、无内购、无账号。

游戏本体与网关**完全解耦**：`lib/logic/` · `lib/models/` · `lib/screens/` · `lib/widgets/` ·
`lib/storage/` 里没有一处 import 到 `lib/gate/`。把 `main.dart` 的 `home:` 换回
`GameScreen` 就是一个干净的单机游戏。

## 目录

```
lib/
  gate/          AB 面网关（gate_config / gate_service / gate_screen / web_screen）
  push/          FCM token 注册（带 gateMode）
  tracking/      AppsFlyer / Adjust 上报
  logic/         棋盘规则与一局游戏的状态机（纯 Dart，无 Flutter 依赖）
  models/        方块形状定义与形状表
  screens/       游戏主界面
  storage/       最高分本地存档（shared_preferences）
  theme/         配色
  widgets/       棋盘 / 待放区 / 单格的绘制
tool/            图标生成脚本（dart run tool/generate_icons.dart）
icon*.png        图标源图，由 flutter_launcher_icons 生成 Android 资源
```

## 与 calcpad 的两处有意差异

其余部分都是逐字节同构的移植，只有这两处是刻意改的：

1. **`web_screen.dart` 的亮度判定修了个潜在坑。** 原实现是
   `(0.299*r + 0.587*g + 0.114*b) / 255.0 > 0.6`，但 `Color.r/g/b` 在 Flutter 3.27 之后
   已经是 **0..1 的 double**，再除 255 会把结果压到千分位 —— 任何颜色都恒判"深色"。
   当前填充色 `#1C1D27` 两种算法结果相同（都是深色、配浅色图标），所以 calcpad 上看不出问题；
   但一旦本包改挂 `ap`/`bp` 这类浅色站、填充色换成白色，除 255 的版本会继续给白底配白图标，
   状态栏图标直接隐形。本包去掉了那个 `/255`。
2. **多了 `lib/storage/`**（最高分存档）。calcpad 完全无本地持久化，本包存一个整数
   （`shared_preferences`，key `tilefit_best_score`）。这一项在 `DATA_SAFETY.md` 里如实申报。

## 上线前必做

### 1. 后台建 listing 条目 —— 待做
Console → 上架包，新建一条：platform=`android`、bundleId=`com.emberlane.tilefit8264`，
挂到对应品牌下。保存顺序按 [09-listing.md §4](../../docs/admin/09-listing.md) 的约束：
**先存网关规则（国家白名单必填非空）再开总开关**，否则 `PUT /listings/:id {gateEnabled:true}` 会 400。

条目建好之前，服务端对本包的 bundleId 查不到 listing，判定恒为 A 面 —— 这是 fail-closed
的预期行为，不是故障。

### 2. 填 `lib/gate/gate_config.dart`
- `apiBases`：已预置为与其余上架包相同的基址 `https://api.fortunegems-jackpot.online`。
  **这是网关 API，不是 B 面地址**；基址迁移时改这里，可加多个候选抗封。
- `appsFlyerDevKey`：已填 `fXoKsKQwxPCRdhD8CD8q6F`（账号级 key，与 colorstack /
  decktallypro / hexacolorsort / calcpad 同一 AF 账号）。`appsFlyerAppId` 在 Android 侧
  即包名，无需改。
- `adjustAppToken`：**已填 `vh0qyczvs6bk`** ✅。Adjust 里的 App 名为 **`TileFit`**，
  Android 平台已配 `com.emberlane.tilefit8264`（store `google` / `google_play`），
  reporting currency **PHP**、`no_eea_users: true`，与 ColorStack 同口径。
  这是本包**专属**的 token，与其余包互不相同（colorstack `bytg13h7yubk` /
  decktallypro `sn947o53ym80` / hexacolorsort `2yhxl7paa3ls` / calcpad `g7gf649m26f4`）——
  复用会把本包的安装与会话归到别的 App 上。
- `adjustOpenBLandingToken`：**已填 `es4a16`** ✅（取自 `TileFit` 的 Events 页）。
  `TileFit` 下已建齐与 ColorStack 相同的 7 个事件：`AddToCart` / `CompleteRegistration` /
  `Login` / `OldRegPurchase` / **`OpenBLanding`** / `Purchase` / `TPFirstDeposit`（均非 unique）。
  其余 6 个是渠道壳 APK 的事件契约（ADR-0013），上架包本体只发 `OpenBLanding`，
  建齐只是为了与 ColorStack 保持同构。
- `adjustContentViewToken`：保持空串。与 colorstack / calcpad 一致，本包不发这个事件。

### 3. FCM（推送）—— 已就位 ✅
`android/app/google-services.json` **已放入**。本包已在 Firebase 项目
**`hybrid-listings-51660`**（project_number `609439342540`）下以包名
`com.emberlane.tilefit8264` 注册（`mobilesdk_app_id` =
`1:609439342540:android:16eeab52067a58eaa98769`）。该文件非机密（随 APK 分发），进 git，
与其余上架包同口径。

**已按规矩裁剪 —— 只保留本包一个 client 条目。**
Firebase 控制台下载的原始文件会把该项目下**所有** Android App 都列进来（本次下载里就有
`com.vividnest.colorstack5821` / `com.slatecove.hexasort4173` / `com.northglade.calcpad5170`
三条，已全部裁掉）。带着它出包，等于在 TileFit 的 APK 里明写另外几个上架包的包名，
任何人解压 APK 就能看出这几个包同属一家 —— 这正是各包使用互不相关的厂商命名空间
（`vividnest` / `deck` / `slatecove` / `northglade` / `emberlane`）想避免的事。
google-services 插件只按 applicationId 取匹配的那条 client，多余条目会被忽略。

**放入后已实测**（Android 35 模拟器，release APK）：日志出现
`FirebaseApp initialization successful`，App 正常进 A 面、不崩。
放入之前跑的是降级路径（`Default FirebaseApp failed to initialize because no default
options were found`，推送 no-op），两条路径都验过。

> 注意 `android/app/build.gradle.kts` 里 google-services 插件是**按文件存在与否条件应用**的。
> 现在文件在了，插件会真正生效 —— 加/删这个文件会改变构建路径，改动后记得重新构建一次。

### 4. 签名 —— 待做
本包目前**没有** release keystore。`android/key.properties`（照 `key.properties.example` 填）
+ keystore 放到 `android/` 下。两者都在 `.gitignore` 里，**不进 git**。
没有 key.properties 时 release 退回 debug 签名，仅供本地跑通 —— **Play 拒收 debug 签名的包**。
生成步骤见 `RELEASE_SIGNING.md`。

### 5. 应用图标 —— 已就位 ✅
自绘的 2×2 方格：左上/右上/左下三个实心块用游戏里的方块本色
（`#4ECDC4` / `#5AA9E6` / `#7C7CE0`），右下留一个空槽（填 `#1E242E` + 描边 `#4A5462`），
底色 `#0E1116` —— 一眼就是"手上的块要拼进空出来的位置"。

- `icon.png` — 1024×1024，legacy 图标源（`mipmap-*/ic_launcher.png`）
- `icon_foreground.png` / `icon_background.png` — 自适应图标两层（Android 8+）
- 由 `flutter_launcher_icons` 生成，配置在 `pubspec.yaml`
- 三张源图都由 `tool/generate_icons.dart` 生成，可复现：
  `dart run tool/generate_icons.dart && dart run flutter_launcher_icons`

图案是纯几何拼的（不依赖字体渲染），各机型/各 DPI 下都一致。脚本先按 4 倍尺寸画、
再用均值插值缩回 1024 —— 等价于 4×4 超采样，因为 `package:image` 的圆角矩形本身不抗锯齿，
直接画 1024 会露出锯齿边。

**为什么做了自适应图标**（colorstack 只有 legacy）：只给 legacy 图标时，Pixel 启动器会把它
缩小塞进一个自己生成的浅色圆底里，深色图标外面套一圈不搭的亮环。给了前景/背景两层后，
启动器用我们自己的 `#0E1116` 底铺满整个形状。`flutter_launcher_icons` 生成的
`mipmap-anydpi-v26/ic_launcher.xml` 会再给前景套一层 16% inset，故前景源图的图案只占
画布 60%，圆形与方圆形两种遮罩下都填得满、四角又不被切到。

## 本地验证

```bash
flutter pub get
flutter analyze               # No issues found
flutter test                  # 50 passed
flutter build apk --release   # 48.5MB，本地验证用
flutter build appbundle --release   # 47.1MB，上架传这个
```

已在 Flutter 3.44.4 / Dart 3.12.2 + AGP 9.0.1 下跑通。

50 项测试的分布：形状表 8 项（图案解析、归一化、无重复格、配色索引在范围内）、
棋盘规则 13 项、一局游戏 11 项、网关不变量 4 项、待放区 8 项、主界面 6 项。
其中三组值得单独说 —— 它们各自压着一个已经真的踩过的坑：

- **「同时填满的行与列一起消除」** 锁住 `Board.place` 里"先找齐再统一清"的顺序。
  若改成边找边清，先清掉的那一行会在与它相交的那一列上打出缺口，导致该列漏判。
  测试构造的正是这个交叉局面。
- **`test/widgets/tray_view_test.dart` 的「按在缝上也能拖起来」** 是一个真 bug 的回归测试。
  待放区的缩略图是一堆 `Positioned` 的方格，格与格之间留了 12% 的缝、L 形还有整块空缺，
  这些位置底下没有可命中的 widget —— 最初的实现里，**手指按在两格之间的缝上根本拖不动**。
  更糟的是这个缺陷有规律：外接矩形的宽或高只要是偶数，矩形正中心就正好落在缝上，
  于是 `h2` / `v2` / `sq2` / `v4` / 所有直角块按正中心都拖不起来，而 `dot` / `h3` / `h5` / `v5`
  这些奇数尺寸的却正常。修法是铺一层透明的 `ColoredBox`（命中行为是 opaque）把整个槽位
  变成把手。测试逐个形状按下外接矩形正中心，断言浮层确实出现了。
- **「把方块拖到棋盘左上角会落子，得分等于它的格数」** 走的是完整的 `Draggable` →
  `DragTarget` → 坐标换算 → 落子链路，不是直接调 `Game.place`。
  注意它**不是**拖到棋盘中央 —— 发牌是随机的，方块尺寸每次不同，按中心算出来的左上角
  落点可能越界，那样测试就成了碰运气（最早的版本正是如此，通过率约五成）。
  现在测试按 App 的同一套几何反推出「让方块左上角正好落在第 (0,0) 格」的手指位置，
  空棋盘上任何形状都放得下，与发到什么牌无关；断言分数**恰好等于格数**，
  说明方块落在了预期的那一格，而不是"碰巧放下了"。

界面测试直接挂 `GameScreen` 而**不是** `TileFitApp` —— 后者的 `home` 是启动闸，
一 pump 就会发真实的网关判定请求，在测试环境里既慢又不确定，而且测的根本不是游戏。

### AOT 产物的安全校验 ✅
反解 release APK 里 `lib/arm64-v8a/libapp.so` 的字符串：

| 检查项 | 结果 |
| --- | --- |
| B 面域名 | **无**。出现的域名只有 Flutter/Firebase 自带的文档与插件通道域名（`docs.flutter.dev` · `api.flutter.dev` · `pub.dev` · `firebase.google.com` · `plugins.flutter.io` · `github.com` · `dart.io`），以及网关 API `api.fortunegems-jackpot.online`。无任何品牌站点域名，也无 `example.com` / 临时调试残留 |
| 品牌字样 | **无**。`gzone` / `arenaplus` / `bingoplus` 命中 0 次 |
| 自身标识 | `com.emberlane.tilefit8264` 与 AF devKey `fXoKsKQwxPCRdhD8CD8q6F` 各命中 1 次 |
| 其他包的 Adjust token | **无**。`bytg13h7yubk` / `sn947o53ym80` / `2yhxl7paa3ls` 均命中 0 次 |
| 自身 Adjust token | 占位符 `TODO_ADJUST_APP_TOKEN` 命中 1 次 —— 说明该位置确实会被编进产物，填上真 token 后同样生效 |

### 模拟器实跑 ✅

在 Android 模拟器（AVD `hexa_test`，x86_64，**必须 `-gpu host`**）上装 **release APK** 实跑过：

| 路径 | 结果 |
| --- | --- |
| A 面启动 | 判定 → 加载页 → 游戏，`Fully drawn +2s232ms`，logcat 无 FATAL / AndroidRuntime / ClassNotFound / NoSuchMethod / NoClassDefFound —— 归因与 Firebase 的反射未被 R8 剪坏 |
| 缺 google-services.json 的降级 | Firebase 初始化失败后照常进 A 面、不崩（本包当前状态） |
| 落子与计分 | 清空数据后拖一个单格块进棋盘 → SCORE 1，方块落在手指上方约 1.4 格处（与 `kDragLiftCells` 一致）；连打二十余手后 SCORE 20，消除与配色均正常 |
| 最高分存档 | `/data/data/com.emberlane.tilefit8264/shared_prefs/FlutterSharedPreferences.xml` 里 `flutter.tilefit_best_score` 与当局得分一致，重启后读回正确 |
| AppsFlyer 实际起来了 | 同目录下生成了 `appsflyer-data.xml`（`appsFlyerCount` / `AF_INSTALLATION` / `appsFlyerFirstInstall`），说明 release 包里 AF SDK 确实初始化并记了会话，不是静默 no-op |

`store/` 下的三张商店截图就是这次实跑截的（1080×2400 原图左右补边到 1200×2400，
理由见 `STORE_ASSETS.md`）。

> 模拟器注意：用 `-gpu swiftshader_indirect`（软件 GL）跑 release 会出现单帧 500 秒以上、
> 画面看起来是白屏。那是模拟器 GPU 问题，不是 App 卡死。一定要用 `-gpu host`。

## 仍未验证

- **真机**（只在 Android 模拟器上跑过）。触屏上的拖放手感、`kDragLiftCells` 这个 1.4 格的
  上浮距离合不合适，最终要在真机上定。
- **B 面两条路径（内开 / 外开）** —— 服务端尚未建本包 listing 条目，正常判定恒为 A。
  `lib/gate/web_screen.dart` 与 calcpad 上那份仅差前述亮度判定一处，但本包自身没跑过 B 面。
- 服务端真实判 B（需先在 Console 建 listing 条目）。
- 推送：`google-services.json` 未放，客户端半程都没跑起来（只验了缺文件时的降级路径）。
- Adjust 上报：App Token 与 event token 都还是占位符 —— 产物里能看到占位符被编进去，
  但没有真实上报过。
- 商店截图还缺「拖动中的落点预览」与「消除瞬间」两张（可选，见 `STORE_ASSETS.md`）。

## 接入红线（已落实）

- 客户端不存任何 B 面 URL 常量（只存网关 API 基址）—— 已由上面的产物扫描证实。
- 客户端不解析、不判 IP（服务端按可信代理链取真实 IP 做）。
- 判定失败 / 超时 / 非 200 / 解析失败 / 结果非 B —— 一律 A 面（fail-closed）。
- 推送只发 `last_gate_mode='B'` 的设备，由服务端 `repo.ActiveListingTokensBMode` 硬编码保证，
  客户端无从绕过。
