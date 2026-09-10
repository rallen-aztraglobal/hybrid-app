# MineGrid

扫雷，Flutter，**仅 Android**，包名 `com.pinehollow.minegrid5083`。
是 `listings/` 下的上架包之一 —— A 面是游戏本体，外层套 AB 面网关（见 `README_GATE.md`）。
六个新包里**唯三走深色主题**的之一。

## 一条设计上的硬要求：每盘都不用猜

扫雷最劝退的一件事是：一步没走错，最后剩两格五五开，蒙错前功尽弃。
那种失败跟水平无关，纯运气。

所以**出题时就把这种局筛掉**：随机布雷 → 用逻辑求解器走一遍 → 走不通就重布。
只有能纯靠推理走完的布局才发给玩家。于是「输了」永远是自己算错。

求解器（`lib/logic/solver.dart`）用的就是人真正会用的三条规则：

1. **基本规则** —— 数字周围雷已标满 ⇒ 剩下的都安全；剩下的格数正好等于还缺的雷数 ⇒ 都是雷
2. **子集规则** —— A 的未知格集合是 B 的子集时，B 多出来那部分装着 (B缺 − A缺) 颗雷。
   这是「1-2-1」「1-2-2-1」这类手法的一般形式
3. **总数规则** —— 用剩余雷数兜底，收拾那些谁都挨不着的格子

规则集比「完美求解器」**弱**，这个方向是**安全**的：只会把某些其实可解的盘误判成
不可解（代价是多重布几次），**绝不会把要猜的盘放行**。反过来做才会出事。

## 重布的代价（测试里打出来的）

| 档 | 盘面 | 雷 | 密度 | 最多试了 |
| --- | --- | --- | --- | --- |
| Normal | 8×10 | 10 | 12.5% | 4 次 |
| Hard | 9×12 | 20 | 18.5% | 11 次 |
| Expert | 10×14 | 30 | 21.4% | 47 次 |
| Master | 11×16 | 42 | 23.9% | 231 次 |

预算给了 4000 次，最坏只用到 231 —— 开局按下去是瞬间出题，玩家察觉不到。
（对比：桌面版三档密度是 12.3% / 15.6% / 20.6%，这里最高档比它还高一档。）

## 其他手感

- **首点必然安全，且必然是空格**（3×3 邻域也无雷）—— 开局只翻出个孤零零的数字的话，
  接下来就只能瞎点了
- **和弦** —— 点已翻开的数字，旗数对得上就一次挖开周围。没有它中后期每格单独点，
  累得没法玩。旗插错了会当场炸，这是应有的代价
- 手机没有右键，所以 `Dig / Flag` 显式开关 + 长按永远执行相反动作
- 连锁展开做了**波纹式错峰**（从落指处一圈圈荡开，每远一圈晚 20ms，封顶 12 圈）——
  几十格同时闪出来是一片突兀的亮色，看不清刚才发生了什么
- 动画时钟是自维护的单调 Ticker，**没有动画在跑时会停表**，空闲时不重绘

## 目录

```
lib/
  logic/
    difficulty.dart   四个难度档（盘面 + 雷数）
    mine_field.dart   布好的雷区，格子用扁平下标
    solver.dart       三条规则的逻辑求解器
    generator.dart    随机布 + 逻辑验 + 不合格重来
    game.dart         一局的状态机（挖 / 插旗 / 和弦 / 判胜）
  screens/game_screen.dart
  widgets/board_view.dart   整块雷区一层 CustomPaint；动画时钟也在这
  storage/progress_store.dart
  theme/app_colors.dart
  gate/               AB 面网关，游戏本体对它零感知
tool/
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

## 常用命令

```bash
cd listings/minegrid
flutter pub get
flutter analyze
flutter test                    # 28 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，28 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/minegrid5083.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `u57nz7eapfcw` + `OpenBLanding` 事件 `obtq6c` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `appaztra@outlook.com`（与另两个游戏包共用）|
| 隐私政策 URL | ✅ `https://benevolent-moxie-f5b881.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `gp`（GameZone），外开 |

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
