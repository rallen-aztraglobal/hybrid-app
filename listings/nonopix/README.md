# NonoPix

数织（nonogram / picross）小游戏，Flutter，**仅 Android**，包名
`com.oakmere.nonopix3927`。是 `listings/` 下的上架包之一 —— A 面是游戏本体，
外层套 AB 面网关（见 `README_GATE.md`）。

## 玩法

行列两侧的数字告诉你这一行/列有几段连续的涂黑格、各多长。推出哪些格子该涂，
一幅小图就浮出来。

## 两条设计上的硬要求

**① 每道题都能纯逻辑解出，不需要猜。**
`test/logic/picture_library_test.dart` 里的「每一张图都能纯逻辑解出，不需要猜」
会对图库里**每一张图**跑一遍逐行推演（`Nonogram.isLineSolvable`）。
加新图时那条测试会拦下需要猜的图案。

数织最劝退的就是推到一半发现只能试 —— 那种失败跟水平无关。这条是商店简短说明里
"never a guess" 的依据，**去掉那条测试就得先把宣传语删掉**。

**② 每道题都是一幅真的图。**
随机生成的噪声也能构成合法的数织题，但解完什么都看不到，白推半天。
图库是手画的（`lib/logic/picture_library.dart`），每张带自己的配色。

## 目录

```
lib/
  logic/
    line_solver.dart      单行推演：枚举所有合法排布，取交集
    nonogram.dart         题面（答案 + 行列线索），isLineSolvable 在这里
    picture_library.dart  手画图库，每张图带调色板
    generator.dart        发题：按尺寸随机取一张，避开刚解过的
    game.dart             一局的状态机（涂 / 打叉 / 撤销 / 判胜）
  screens/game_screen.dart
  widgets/board_view.dart 整块棋盘用一层 CustomPaint 画
  storage/progress_store.dart
  theme/app_colors.dart
  gate/                   AB 面网关，游戏本体对它零感知
tool/
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

## 常用命令

```bash
cd listings/nonopix
flutter pub get
flutter analyze
flutter test                    # 56 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，56 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/nonopix3927.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `fpny286clibk` + `OpenBLanding` 事件 `ur46ap` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `appaztra@outlook.com`（与另两个游戏包共用）|
| 隐私政策 URL | ✅ `https://moonlit-kangaroo-0a8292.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `ap`（ArenaPlus），外开 |

上架前的完整清单见 `PLAY_CONSOLE_FORM_ANSWERS.md` 末尾。

## 相关文档

| 文件 | 内容 |
| --- | --- |
| `README_GATE.md` | AB 面网关怎么接的、怎么验 |
| `PRIVACY_POLICY.md` | 隐私政策正文（需托管成网页） |
| `DATA_SAFETY.md` | Play 的 Data safety 表单逐项答案 |
| `PLAY_CONSOLE_FORM_ANSWERS.md` | 其余各表单答案 + 发布前清单 |
| `PLAY_STORE_LISTING.md` | 商店文案（应用名 / 简短说明 / 完整说明） |
| `STORE_ASSETS.md` | 素材清单与重新生成方法 |
| `RELEASE_SIGNING.md` | keystore 生成与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
