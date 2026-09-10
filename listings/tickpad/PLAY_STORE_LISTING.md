# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
TickPad
```

## 简短说明（≤80）

```
Tally counters that keep your count. Undo, volume keys, saved sessions.
```

（70 字符）

## 完整说明（≤4000）

```
TickPad is a set of tally counters for when you are counting real things and cannot
trust yourself to remember: stock on a shelf, people through a door, laps, reps,
cars, birds, votes.

TAP ANYWHERE ON THE CARD

The whole card counts up, not just a small button — you are usually looking at what
you are counting, not at your phone. The minus button sits off to the side, because
correcting is not the main action.

VOLUME KEYS

Switch it on and the volume buttons count for you. Hands busy, phone in your pocket,
eyes on the shelf. It is off by default and one tap to turn on.

UNDO

Thirty steps of it. Counting is something you do fast and half-blind — one tap too
many, the wrong card, an accidental reset. Undo covers all of it, including deleting
a counter, which comes back with its count and its place in the list.

NEVER LOSES YOUR COUNT

Every tap is written to storage immediately, not when you close the app. Counters are
usually used with the phone getting locked and pocketed between counts, and Android
can end an app at any time.

MORE THAN ONE THING AT A TIME

• As many counters as you need, each with its own name and colour
• A running total across all of them
• Step size — count by 1, 2, 5, 10 or 12 when things come by the case
• Set a target and watch a progress bar; a firmer buzz tells you it is reached
• Drag to reorder

SAVE A ROUND, THEN START AGAIN

Stocktakes happen in rounds. Save the session before you reset and the numbers are
kept with a timestamp, so resetting is safe. Copy any round as plain text to paste
into a message or a spreadsheet.

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases.

PRIVACY

TickPad does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. Your counters — including the names you
give them — stay on your phone. See the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **应用** |
| 类别 | Tools |
| 标签 | Tools · Utilities · Productivity · Counter · Offline |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **toolsa069@gmail.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://delicate-seahorse-a2b478.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的三条

1. **不写做不到的事。** 没有云同步、没有导出 CSV、没有小组件、没有多设备，
   就一句都不提。「Copy any round as plain text」写的是**复制到剪贴板**，
   不是导出文件 —— 措辞要留住这个区别。
2. **"never loses your count" 是有依据的**：每次改动立刻写盘
   （`lib/storage/counter_store.dart`），读那一侧全程兜底、绝不抛异常。
   实测跑过「杀进程 → 重启 → 计数原样还在」。
3. **音量键那段要如实说「默认关」。** 接管音量键是件霸道的事，文案里含糊过去、
   用户装上发现调不了音量，是差评的直接来源。
