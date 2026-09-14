# DaySpan

日期计算器，Flutter，**仅 Android**，包名 `com.haymoor.dayspan3306`。
是 `listings/` 下的上架包之一 —— A 面是工具本体，外层套 AB 面网关（见 `README_GATE.md`）。
浅色主题。

## 卖点与它的落点

**跨夏令时、闰年、月末都算得对。**

这句话不是形容词，它有一条测试钉着：

`test/logic/date_math_test.dart`，36 项。其中三组是这类工具最容易错的地方：

**夏令时** —— 本地时区里「一天」不一定是 24 小时。测试特意用**本地时间**构造了
跨美国夏令时开始日（23 小时）与结束日（25 小时）的用例；内部一律规约到 UTC，
因而不受影响。

**闰年** —— 1900 不是闰年、2000 是（整百年不闰、四百年再闰）。

**月末截断** —— 1 月 31 日加一个月是 2 月 28/29 日，不是溢出到 3 月 3 日。
还有一条自洽性测试：`breakdown` 拆出的年月日加回 `from` 必须正好回到 `to`，
覆盖月末、闰年、跨年的各种组合。

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
cd listings/dayspan
flutter pub get
flutter analyze
flutter test                    # 36 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，36 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/dayspan3306.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `jmnqnzl3ury8` + `OpenBLanding` 事件 `3e00vd` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `toolsa069@gmail.com`（与其余工具包共用）|
| 隐私政策 URL | ✅ `https://stellular-gumption-304771.netlify.app/`（本包专用站点）|
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
| `RELEASE_SIGNING.md` | keystore 与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
