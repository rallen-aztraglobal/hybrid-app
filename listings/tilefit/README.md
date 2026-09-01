# TileFit

Flutter 方块拼放游戏，**仅 Android**，作为上架包接入了 AB 面网关。

- 包名 `com.emberlane.tilefit8264`
- 商店名 `TileFit`
- A 面 = 游戏本体（`lib/screens/game_screen.dart`），B 面 = 服务端下发的 web

## 玩法

8×8 棋盘，每轮发 3 个形状。把形状拖到棋盘上，填满整行或整列即消除；三个用完补发新的一手；
剩下的形状在棋盘上都放不下时本局结束。落子得「格数」分，同时消除 n 条线额外得 10·n² 分 ——
所以攒一手同时消多条，明显优于逐条消。形状固定、**不可旋转**。

纯离线，无广告、无内购、无账号、无排行榜。只往本地存一个最高分。

## 从哪份文档开始看

| 想做的事 | 看这份 |
| --- | --- |
| 搞懂网关怎么接的、还差什么没配 | **`README_GATE.md`**（先看这份） |
| 生成 release keystore | `RELEASE_SIGNING.md` |
| 填 Play Console 的问卷 | `PLAY_CONSOLE_FORM_ANSWERS.md` |
| 填 Data safety 表单 | `DATA_SAFETY.md` |
| 上传商店文案 | `PLAY_STORE_LISTING.md` |
| 上传截图 / 特色图 / 托管隐私政策 | `STORE_ASSETS.md` |
| 隐私政策正文 | `PRIVACY_POLICY.md`（可托管版：`store/privacy-policy.html`） |
| 密钥与机密的边界 | `SECURITY_NOTES.md` |

## 当前状态

跑得通的：

```bash
flutter pub get
flutter analyze                     # No issues found
flutter test                        # 50 passed
flutter build apk --release         # 48.5MB，已在 Android 模拟器上实跑验过 A 面
flutter build appbundle --release   # 47.1MB，上架传这个
```

上架前还差的（都在 `README_GATE.md` 里有步骤）：

- [ ] Adjust 后台建 App，填 `adjustAppToken` 与 `adjustOpenBLandingToken`
- [ ] Firebase 注册本包，放入**裁剪过的** `android/app/google-services.json`
- [ ] 生成 release keystore（现在缺 `key.properties`，release 回退 debug 签名，Play 拒收）
- [ ] 渠道中台建 listing 条目（建好之前判定恒为 A 面，属预期的 fail-closed）
- [ ] 托管隐私政策、定支持邮箱（都**不要**复用其他上架包的）
- [ ] 真机实跑一次（目前只在模拟器上跑过）

## 目录

```
lib/
  gate/          AB 面网关（新增，与 colorstack / calcpad 同构）
  push/          FCM token 注册（新增）
  tracking/      AppsFlyer / Adjust 上报（新增）
  logic/         棋盘规则与一局游戏的状态机（纯 Dart，无 Flutter 依赖）
  models/        方块形状定义与形状表
  screens/       游戏主界面
  storage/       最高分本地存档
  theme/         配色
  widgets/       棋盘 / 待放区 / 单格的绘制
test/            50 项：形状表 8 · 棋盘规则 13 · 一局游戏 11 · 网关不变量 4 · 待放区 8 · 主界面 6
tool/            图标与商店素材的生成脚本（可复现）
store/           Play 上架素材（截图、特色图、512 图标、可托管的政策页）
icon*.png        应用图标源图，由 flutter_launcher_icons 生成 Android 资源
```

游戏本体与网关**完全解耦**：`lib/logic/` · `lib/models/` · `lib/screens/` · `lib/widgets/` ·
`lib/storage/` 里没有一处 import 到 `lib/gate/`。把 `main.dart` 的 `home:` 换回 `GameScreen`
就是一个干净的单机游戏。
