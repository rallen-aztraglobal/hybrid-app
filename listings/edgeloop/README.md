# EdgeLoop

画一条闭合回路的逻辑谜题，Flutter，**仅 Android**，包名 `com.bellcroft.edgeloop8451`。
是 `listings/` 下的上架包之一 —— A 面是游戏本体，外层套 AB 面网关（见 `README_GATE.md`）。
浅色主题。

## 卖点与它的落点

**每一关有且仅有一个解。**

这句话不是形容词，它有一条测试钉着：

`test/logic/puzzle_library_test.dart` 的「每一关都恰好一个解」——
它对关卡库里**每一关**跑穷举求解器，断言解的个数正好是 1、且搜索完整跑完
（撞节点上限时 `exhausted=false`，唯一性无从谈起，一律判失败）。

另有「每一关的解，游戏都判为已解出」把 `EdgeLoopGame.isSolved` 与求解器的口径钉死：
求解器认的解，游戏必须也认。两者一旦分家，「唯一解」这句话就不成立了 ——
玩家可能摆出别的样子也过关。

**删掉那条测试之前，必须先把这句话从 `PLAY_STORE_LISTING.md` 里删掉。**

## 目录

```
lib/
  logic/     游戏本体的全部逻辑，不依赖 Flutter
  screens/   界面
  storage/   本地存档
  theme/     配色
  gate/      AB 面网关，本体对它零感知
tool/
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

## 常用命令

```bash
cd listings/edgeloop
flutter pub get
flutter analyze
flutter test                    # 33 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，33 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/edgeloop8451.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `g1usefapvwg0` + `OpenBLanding` 事件 `tmzqo0` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `appaztra@outlook.com`（与其余游戏包共用）|
| 隐私政策 URL | ✅ `https://precious-douhua-53c7dc.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `ap`（ArenaPlus），外开 |

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
| `RELEASE_SIGNING.md` | keystore 与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
