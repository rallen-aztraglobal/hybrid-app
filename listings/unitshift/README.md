# UnitShift

单位换算，Flutter，**仅 Android**，包名 `com.mossgate.unitshift4610`。
是 `listings/` 下的上架包之一 —— A 面是工具本体，外层套 AB 面网关（见 `README_GATE.md`）。

## 交互取向：不选目标单位

常见换算 App 是「选源单位 → 选目标单位 → 看一个数」，改一次目标要点两下。
这里**把整类单位一次全列出来，边打边全部跟着变** —— 不存在「选目标」这一步。

点某一行会把它变成源单位，并且**把这一行当前显示的数值搬进输入框**：
看到 4000 mL 点一下，还是 4000 mL，不会突然跳回 4 L。数值含义不跳，才敢连着点两下
换算链。长按任意一行复制。

自己画了数字键盘而不是用系统键盘：系统键盘一弹一收布局就跳、换算列表被顶得看不见，
而且各家输入法的数字键位差别很大，还可能塞进全角字符。

## 数据上的两个硬要求

**① 系数一律写精确定义值。**
`1 in = 0.0254 m`、`1 lb = 0.45359237 kg`、`1 mi = 1609.344 m` 都是国际协定的精确值，
写全位数没有误差；抄个近似值在长距离换算上会一路放大。
每个不显然的系数后面注明了出处。

**② 温度是唯一不能用系数的一类** —— 它带零点偏移。用系数算的话 0°C 会得出 0°F，
差 32 度。这是单位换算 App 最典型也最容易被忽略的 bug，所以温度走的是一对手写互逆
函数，并且单独钉了一条测试：`0°C 换出来不等于 0`。

测试对的是**权威定义值**，不是拿代码自己的结果对自己：
1 lb = 16 oz、1 加仑 = 128 fl oz、−40 是摄氏华氏的交点、0°R = −459.67°F、
1 atm = 760 mmHg、1 TiB = 1099511627776 B。另外每个单位都跑了往返换算
（守住「toBase / fromBase 真是一对互逆函数」，温度那种手写函数最容易写反某一步）。

## 显示格式单独占一个模块

换算 App 的成败一半在这里。直接 `toString()` 满屏都是 `2.5399999999999996` ——
算得对但看着像坏了。

规则：按**有效数字**决定保留几位小数；大整数照原样给（1 TiB 的字节数是精确值，
显示成 `1.0995e12` 不如原样有用）；只在真读不动的量级才切科学计数法。

> 科学计数法的下界（1e-5）和小数位上限是**绑死的**：比 1e-12 更小的数会被四舍五入
> 成 `0` —— 那是**把值丢了**，比难看严重得多，所以留足了余量。

## 目录

```
lib/
  logic/
    unit.dart        一个单位（线性用系数，温度用一对互逆函数）
    catalog.dart     十类单位表，系数写精确定义值
    formatter.dart   数字显示：有效数字 / 千分位 / 科学计数法
    entry.dart       输入框的状态机（前导零 / 小数点 / 退格 / 正负号）
  screens/converter_screen.dart
  widgets/keypad.dart
  storage/prefs_store.dart   只记上次用的分类与源单位
  theme/app_colors.dart
  gate/              AB 面网关，工具本体对它零感知
tool/
  generate_icons.dart         图标源图
  generate_store_assets.dart  商店素材
```

## 常用命令

```bash
cd listings/unitshift
flutter pub get
flutter analyze
flutter test                    # 57 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，57 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/unitshift4610.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `qv7x4u8c7bi8` + `OpenBLanding` 事件 `7pdbn6` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `toolsa069@gmail.com`（与另两个工具包共用）|
| 隐私政策 URL | ✅ `https://thriving-banoffee-396732.netlify.app/`（本包专用站点）|
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
| `RELEASE_SIGNING.md` | keystore 生成与签名核对 |
| `SECURITY_NOTES.md` | 什么绝不进 git、APK 自查 |
| `SUBMISSION_NOTES.md` | **内部**：口径、待定项、取证依据（不要粘进 Console） |
