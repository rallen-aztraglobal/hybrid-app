# TickPad

多计数器（tally counter），Flutter，**仅 Android**，包名 `com.larkspur.tickpad7358`。
是 `listings/` 下的上架包之一 —— A 面是工具本体，外层套 AB 面网关（见 `README_GATE.md`）。
深色主题。

## 用来干嘛

数东西的时候按一下加一：仓库盘点、门口点人数、点票点钞、健身计次、跑圈、
打牌记分、观鸟车流统计。共同点是**手在忙、眼睛不一定看屏幕、数着数着会被打断**。

## 三条设计上的硬要求

**① 撤销是这个 App 最重要的功能。**
计数器是闭着眼快点的东西 —— 多点一下、点错卡片、手滑清零都是家常便饭。
没有撤销的话用户只能凭记忆改回去，而他之所以用计数器，正是因为记不住。

实现上存的是**整份状态的快照**，不是「反向操作」。反向操作要给每种动作单独写逆操作，
删除卡片的逆操作还得记住它原来的位置，漏一个就会出现「撤销之后数据变了样」这种
最难查的 bug。快照顶多几十个整数、不到 1 KB。撤销栈深 30 层。

> 有一条容易漏的：**空操作不进撤销栈**。已经是 0 还点减号、全是 0 还点清零 ——
> 如果这些也压快照，用户点几下之后再撤销会连撤好几次才回到真正的上一步，
> 看着像撤销坏了。`test/logic/board_test.dart` 里单独钉了这条。

**② 每一次改动立刻写盘**，不做防抖、不等退出。计数器经常是「按完就锁屏塞回口袋」，
进程随时被系统回收，晚写一秒就可能少记一个。
读那一侧写得很保守：键不存在、不是合法 JSON、顶层不是数组、某条不是对象、
字段类型不对、计数是负数、步长是 0 —— 全都各自兜住，**绝不抛异常**。

**③ 整张卡片都是加一的按钮。** 不看屏幕也要能按，按钮做小了就得瞄准。
减一是个小按钮靠边放（纠错动作不该抢注意力），到 0 就淡掉。

## 音量键计数（唯一一处原生代码）

`android/.../MainActivity.kt` 拦 `onKeyDown` / `onKeyUp`，通过 MethodChannel
（`tickpad/volume_keys`）转成计数事件。

- **默认关。** 接管音量键很霸道 —— 用户可能只是想调音量，结果计数被加了一个。
- `onKeyUp` 也要拦：只拦 down 的话，系统仍会在抬手时弹出音量条。
- 作用在高亮那张卡片上（点过谁就是谁），卡片描边和底部提示都会变绿。
- Dart 侧全部包了 try/catch —— 这是锦上添花的功能，平台通道出任何问题都不该影响
  用手点数这条主路径。

## 目录

```
lib/
  logic/
    counter.dart   一个计数器（值 / 步长 / 目标 / 颜色），fromJson 逐字段兜底
    board.dart     整屏状态机 + 快照式撤销
    session.dart   存下来的一轮计数（存名字的副本，不是 id）
    summary.dart   排成可直接发出去的纯文本
  platform/volume_keys.dart   音量键的 MethodChannel 封装
  screens/counters_screen.dart · history_screen.dart
  widgets/counter_card.dart
  storage/counter_store.dart · session_store.dart · settings_store.dart
  theme/app_colors.dart
  gate/            AB 面网关，工具本体对它零感知
```

## 常用命令

```bash
cd listings/tickpad
flutter pub get
flutter analyze
flutter test                    # 58 项
flutter build apk --release
```

## 现状

| 项 | 状态 |
| --- | --- |
| 代码 | ✅ analyze 干净，58 项测试全过 |
| 图标 | ✅ `tool/generate_icons.dart` 生成 |
| 商店素材 | ✅ `store/` 下齐了（图标 / 特色图 / 2 张截图） |
| keystore | ✅ `android/tickpad7358.jks`（不进 git），release APK 已核验为自己的证书 |
| Adjust token | ✅ App Token `oi2l8k6c3k00` + `OpenBLanding` 事件 `9it54r` |
| `google-services.json` | ✅ 已放入（**裁剪过，只留本包 client**）|
| 支持邮箱 | ✅ `toolsa069@gmail.com`（与另两个工具包共用）|
| 隐私政策 URL | ✅ `https://delicate-seahorse-a2b478.netlify.app/`（本包专用站点）|
| 挂哪个品牌 | ✅ `bp`（BingoPlus），外开 |

## 一件没查清的事（留个记录）

加「目标 / 颜色 / 会话」那批功能的**那一次升级安装之后**，我见过一次计数变成 0
（卡片名字和数量都在，只有数值没了）。之后：又重装两次数据都完好；把**旧格式的存档**
（没有 `target` / `color` 这两个新键）直接写进设备再启动，`Boxes 7 / Pallets 5` 原样读出、
文件也没被改写；单元测试里也钉了「旧存档缺新字段是正常情况，不是损坏」。

**没能复现，原因不明。** 目前证据是存储层正确。后面几轮打包盯着这个点。

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
