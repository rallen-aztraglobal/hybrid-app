# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
EdgeLoop
```

## 简短说明（≤80）

```
Draw one closed loop. Every puzzle has exactly one solution — verified, not claimed.
```

（84 字符）

## 完整说明（≤4000）

```
Every cell with a number tells you how many of its four sides belong to the loop. Work out
the rest. When the line closes into a single loop and every number is satisfied, the puzzle is
done.

EVERY PUZZLE HAS EXACTLY ONE SOLUTION

Not "designed to" — checked. Each of the 50 puzzles was run through an exhaustive solver that
counts solutions, and only puzzles with exactly one were kept. You will never find a second
valid loop, and you will never need to guess between two that both look right.

WHAT YOU GET

• Four difficulty tiers, 50 hand-verified puzzles
• Mark edges you have ruled out — the cross is how these puzzles are actually solved
• Clue numbers dim when satisfied and turn red when you overshoot
• Tap anywhere near an edge; the whole board is a target, there are no dead spots
• Your progress is kept per difficulty

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases.

PRIVACY

EdgeLoop does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. See the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **游戏** |
| 类别 | Puzzle |
| 标签 | Puzzle · Logic · Brain · Offline · No ads |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **appaztra@outlook.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://precious-douhua-53c7dc.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的一条

**不写做不到的事。** 这一条在「每一关有且仅有一个解」上尤其要紧：那句话是可验证的断言，
不是形容词。它由 `test/logic/puzzle_library_test.dart` 钉着，
删掉那条测试就必须同时删掉这句文案。
