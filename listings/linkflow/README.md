# LinkFlow

连线小游戏（flow / connect-the-dots），Flutter，**仅 Android**，包名
`com.wrenfield.linkflow6142`。是 `listings/` 下的上架包之一 —— A 面是游戏本体，
外层套 AB 面网关（见 `README_GATE.md`）。

## 玩法

棋盘上散着若干对同色端点。从一端拖到另一端画出一条管子。两条规则：
**线不能交叉**，**整盘必须铺满**。连通是容易的一半，不留空格才是难点。

## 两条设计上的硬要求

**① 每道题恰好一个解。**
`test/logic/puzzle_library_test.dart` 的「每道题都只有一个解」会对出厂的每一关
跑一遍 `FlowSolver`，枚举全部填法。多解题是这类游戏最伤口碑的地方 ——
要么"随便乱连也能过"，要么玩家算出的另一种解不被认。

判定口径和 `FlowGame.isSolved` **完全一致**（只要求全连通 + 铺满，不额外禁止通路
自己贴着自己走）。口径更宽 ⇒ 解更多 ⇒ 唯一性要求更严；反过来做会漏掉玩家实际能用的
第二种走法。

**② 关卡按难度升序。**
生成器用求解器的搜索节点数当难度代理量，档内由易到难排，顺着打是一条平滑的坡。

## 题库是生成的，不是手写的

`tool/generate_levels.dart`（**开发期工具，不进 App**）：

1. 先造一条走遍全盘的蛇（哈密顿路径，用 backbite 随机化），再随机剪成 k 段。
   每段天然是一条合法通路，合起来正好铺满 —— **从根上不可能造出「连不完」的坏题**。
   早先手写谜题时反复出现的「盖不满 / 盖重了」，这个办法直接消掉了。
2. 剪完用求解器数解，只留唯一解的。
3. 按搜索节点数排序出厂。

固定随机种子，同一份代码跑出同一套题，改了参数才会变，方便 review diff。

```bash
cd listings/linkflow && dart run tool/generate_levels.dart   # 覆写 lib/logic/puzzle_library.dart
```

## 出厂题库

| 档 | 盘面 | 通路数 | 关数 | 求解节点 |
| --- | --- | --- | --- | --- |
| Normal | 5×5 | 4–5 | 8 | 25 … 31 |
| Hard | 7×7 | 6–7 | 14 | 174 … 2761 |
| Expert | 8×8 | 9–10 | 14 | 209 … 5909 |
| Master | 9×9 | 11–12 | 10 | 1115 … 6644 |

> Master 只有 10 关（其余档 14 关）。9×9 是唯一解最难碰的一档 ——
> 36 万种剪法只攒到 19 道候选。试过收成「只用 12 条线」想提高命中率，结果反而更差
> （24 万刀只得 7 道，且最难的一道才 697 节点）。原因写在生成器的注释里了。

## 目录

```
lib/
  logic/
    board.dart          格子坐标
    puzzle.dart         一道题（起点 + 方向串），validate() 校验不重不漏
    solver.dart         数解用；应用不引用它，release 会被 tree shaking 摘掉
    puzzle_library.dart 生成出来的题库，**不要手改**
    game.dart           一局的状态机（拖线 / 截断 / 判胜）
  screens/game_screen.dart
  widgets/board_view.dart   整块棋盘用一层 CustomPaint 画
  storage/progress_store.dart
  theme/app_colors.dart
  gate/                 AB 面网关，游戏本体对它零感知
tool/
  generate_levels.dart        题库生成器
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

## 常用命令

```bash
cd listings/linkflow
flutter pub get
flutter analyze
flutter test                    # 34 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，34 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/linkflow6142.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `r1akj2bp7xts` + `OpenBLanding` 事件 `fqjboe` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `appaztra@outlook.com`（与另两个游戏包共用）|
| 隐私政策 URL | ✅ `https://frolicking-capybara-46647a.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `bp`（BingoPlus），外开 |

上架前的完整清单见 `PLAY_CONSOLE_FORM_ANSWERS.md` 末尾与 `SUBMISSION_NOTES.md`。

## 相关文档

| 文件 | 内容 |
| --- | --- |
| `README_GATE.md` | AB 面网关怎么接的、怎么验 |
| `PRIVACY_POLICY.md` | 隐私政策正文（需托管成网页） |
| `DATA_SAFETY.md` | Play 的 Data safety 表单逐项答案 |
| `PLAY_CONSOLE_FORM_ANSWERS.md` | 其余各表单答案 + 提交前清单 |
| `PLAY_STORE_LISTING.md` | 商店文案 |
| `STORE_ASSETS.md` | 素材清单与重新生成方法 |
| `RELEASE_SIGNING.md` | keystore 生成与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
