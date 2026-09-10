# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
LinkFlow
```

## 简短说明（≤80）

```
Connect the matching dots and fill the board. Every level has one exact solution.
```

（79 字符）

## 完整说明（≤4000）

```
LinkFlow is a quiet, careful puzzle. Pairs of coloured dots sit on a grid. Drag from
one dot to its twin to draw a pipe between them.

Two rules, and that is the whole game:

• Pipes cannot cross each other
• Every square on the board must end up covered

Connecting all the pairs is the easy half. Covering the board without leaving a gap is
what makes you think.

ONE EXACT SOLUTION

Every level shipped with LinkFlow was checked by a solver before it went in: each one
has exactly one way to fill the board. No level can be finished by connecting things
at random, and no level rejects a layout that ought to work. When you find the answer,
it is the answer.

FOUR DIFFICULTY TIERS

• Normal — 5×5, a few lines, to learn the rules
• Hard — 7×7
• Expert — 8×8, up to ten lines at once
• Master — 9×9 with twelve lines, where the board is nearly full before you begin

Levels within a tier are ordered by how much reasoning they actually need, so the
difficulty climbs steadily instead of jumping around.

MADE TO PLAY WITH ONE THUMB

• Drag back along a pipe to undo it a square at a time — no need to redraw the line
• Run into another line and it gives way, so you never have to clear up first
• The progress bar shows both how many pairs are joined and how full the board is —
  because "all connected but not solved" is the thing that confuses people

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases. Every level is built into
the app, so it works with no connection at all.

PRIVACY

LinkFlow does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. Which level you are on is kept on your
phone. See the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **游戏** |
| 类别 | Puzzle |
| 标签 | Puzzle · Brain games · Logic · Casual · Single player |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **appaztra@outlook.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://frolicking-capybara-46647a.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的两条

1. **不写做不到的事。** 没有每日挑战、没有排行榜、没有 1000 关，就一句都不提。
   商店文案写了而 App 里没有，是差评的第一来源。
   全部 46 关（8 / 14 / 14 / 10），文案里没有报关卡数 —— 报了就得一直维护它。
2. **"one exact solution" 是真的能兑现的。** 题库生成时每道题都由
   `lib/logic/solver.dart` 枚举过全部填法，只有**恰好一个解**的才出厂；
   `test/logic/puzzle_library_test.dart` 里的「每道题都只有一个解」会在每次跑测试时复验。
   判定口径和游戏本身完全一致（只要求全连通 + 铺满，不额外禁止通路自己贴着自己走），
   所以玩家用任何游戏认可的画法都算数。
   **如果哪天去掉了那条测试或改了出题方式，这句宣传语必须先删掉。**
