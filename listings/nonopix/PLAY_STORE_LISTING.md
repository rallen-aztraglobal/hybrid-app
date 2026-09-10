# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
NonoPix
```

## 简短说明（≤80）

```
Nonogram puzzles that always solve by logic — never a guess. Reveal the picture.
```

（79 字符）

## 完整说明（≤4000）

```
NonoPix is a picture logic puzzle — also known as nonogram, picross or griddler.

The numbers along each row and column tell you how many squares to fill and in what
order. Work out which squares are filled, and a small picture appears.

WHAT MAKES A GOOD NONOGRAM

Every puzzle in NonoPix is checked before it ships: each one can be solved by
reasoning alone, one line at a time. You never have to guess and hope. If you are
stuck, there is always a next step you have not spotted yet.

And every puzzle is a real picture — a boat, a tree, a cat, a heart. Randomly
generated noise makes a valid puzzle but gives you nothing at the end. Here the
payoff is seeing what you drew.

FEATURES

• Two board sizes: 5×5 to warm up, 8×8 when you want to sit down with it
• Hand-drawn pictures, in colour
• Fill and cross modes — mark the squares you have ruled out
• Row and column clues dim once they are satisfied, so you can see what is left
• Progress is saved on your device
• No timer, no lives, no energy meter

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases. Puzzles are built into the
app, so it works with no connection at all.

PRIVACY

NonoPix does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. How many puzzles you have solved is kept
on your phone. See the privacy policy for the full detail.
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
| 隐私政策 URL | `https://moonlit-kangaroo-0a8292.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的两条

1. **不写做不到的事。** 没有每日挑战、没有排行榜、没有 5000 关，就一句都不提。
   商店文案写了而 App 里没有，是差评的第一来源。
2. **"never a guess" 是真的能兑现的。** 图库里每一张图都被
   `test/logic/picture_library_test.dart` 的「每一张图都能纯逻辑解出，不需要猜」
   逐张验过（依据是 `Nonogram.isLineSolvable` 的逐行推演）。加新图时那条测试会拦下
   需要猜的图案。
   **如果哪天去掉了那条测试或改了出题方式，这句宣传语必须先删掉。**
