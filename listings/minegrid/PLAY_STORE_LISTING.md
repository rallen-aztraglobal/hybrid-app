# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
MineGrid
```

## 简短说明（≤80）

```
Minesweeper you never have to guess at. Every board is solvable by logic alone.
```

（78 字符）

## 完整说明（≤4000）

```
MineGrid is minesweeper — with the guessing taken out.

Tap a square to uncover it. The number tells you how many mines touch that square.
Work out where they are, flag them, and clear the rest of the board.

NO 50/50s. EVER.

The worst thing about minesweeper is getting to the end of a careful board, finding
two squares and no way to tell which one hides the mine, and losing a coin flip. That
loss has nothing to do with how well you played.

MineGrid deals a board, then checks it with a solver before handing it to you. If the
board cannot be finished by reasoning alone, it is thrown away and another is dealt.
Every board you get can be cleared without a single guess. When you lose, you got
something wrong — and there was something there to get right.

FOUR SIZES

• Normal — 8×10, 10 mines
• Hard — 9×12, 20 mines
• Expert — 10×14, 30 mines
• Master — 11×16, 42 mines, denser than the classic expert board

BUILT FOR A THUMB

• The whole board is one drawing surface, so it stays smooth even when a large area
  opens at once
• Dig / flag switch at the bottom, and a long press always does the other one
• Chording: tap a number whose mines are all flagged and the rest opens at once
• Flags you placed wrongly are crossed out when a board ends, so you can see where
  your reasoning went off
• Fastest time per size is kept on your device

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases. Boards are generated on
your phone, so it works with no connection at all.

PRIVACY

MineGrid does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. Your record is kept on your phone. See
the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **游戏** |
| 类别 | Puzzle |
| 标签 | Puzzle · Brain games · Logic · Classic · Single player |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **appaztra@outlook.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://benevolent-moxie-f5b881.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的两条

1. **不写做不到的事。** 没有每日挑战、没有排行榜、没有联机，就一句都不提。
2. **"never have to guess" 是真的能兑现的。** 每盘发牌前都由
   `lib/logic/solver.dart` 用「人真正会用的三条规则」（基本 / 子集 / 总数）走一遍，
   走不通就重发；`test/logic/generator_test.dart` 每次跑测试都会抽样复验四个档
   各 25 盘，并独立再验一次返回的那盘。
   求解器故意做得比「完美求解器」弱 —— 那个方向是**安全**的：只会把某些其实可解的盘
   误判成不可解（代价是多重发几次），绝不会把要猜的盘放行。
   **如果哪天去掉了那道筛选或那条测试，这句宣传语必须先删掉。**
