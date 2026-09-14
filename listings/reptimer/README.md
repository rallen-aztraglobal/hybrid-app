# RepTimer

间隔计时器，Flutter，**仅 Android**，包名 `com.quillford.reptimer7719`。
是 `listings/` 下的上架包之一 —— A 面是工具本体，外层套 AB 面网关（见 `README_GATE.md`）。
深色主题。

## 卖点与它的落点

**0 秒的阶段整段不存在，最后一组之后不留多余休息。**

这句话不是形容词，它有一条测试钉着：

`test/logic/interval_plan_test.dart`，29 项。两组针对这类 App 最常见的毛病：

**0 秒的段整段消失**，而不是「显示 0 秒再跳走」。把休息设成 0（连续做组）是很常见的
用法；若保留一个 0 秒的段，界面会闪一下再跳，而「每秒减一、减到 0 才切换」那种实现
还会在这种段上多跑一秒甚至卡住。这里在**构建时间轴时**就剔除了，运行时不存在这种段。
有一条测试对六种参数组合断言「时间轴里不存在任何 0 秒的段」。

**最后一组之后不留休息**，最后一个大循环之后也不留长休息。练完了就是练完了，
不该再等一段无意义的倒数。有一条测试直接断言「时间轴最后一段永远是 work」，
覆盖五种参数组合。

**删掉那条测试之前，必须先把这句话从 `PLAY_STORE_LISTING.md` 里删掉。**

## 目录

```
lib/
  logic/     工具本体的全部逻辑，不依赖 Flutter
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
cd listings/reptimer
flutter pub get
flutter analyze
flutter test                    # 29 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，29 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/reptimer7719.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `9sgzjiaxgfeo` + `OpenBLanding` 事件 `av2feh` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `toolsa069@gmail.com`（与其余工具包共用）|
| 隐私政策 URL | ✅ `https://splendid-frangipane-a4c517.netlify.app/`（本包专用站点）|
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
| `RELEASE_SIGNING.md` | keystore 与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
