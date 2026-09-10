# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
UnitShift
```

## 简短说明（≤80）

```
Type a number once and every unit updates at the same time. No target picker.
```

（76 字符）

## 完整说明（≤4000）

```
Most converters make you pick a unit to convert from, then pick a unit to convert to,
then read one number. Change your mind about the target and that is two more taps.

UnitShift lists the whole category at once. Type 5 and you see it in metres, feet,
inches, miles and everything else in that category, all at the same time. There is no
target to pick.

Tap any row to make it the source. The number you were looking at stays put — 5000 g
becomes 5000 g, not 5 kg — so you can chain conversions without losing your place.
Long-press a row to copy that value.

TEN CATEGORIES

Length · Mass · Temperature · Area · Volume · Speed · Time · Data · Pressure · Energy

THE NUMBERS ARE RIGHT

Conversion factors are the exact defined values, written out in full: 1 in = 0.0254 m,
1 lb = 0.45359237 kg, 1 mile = 1609.344 m. Rounded-off factors drift on long
conversions.

Temperature is handled separately because it has an offset, not just a factor —
0°C is 32°F, not 0°F. That is the classic converter bug and it is covered by a test
here.

US and imperial units are listed separately where they differ, because they really do
differ: a US gallon is 3.785 L and a UK gallon is 4.546 L, about 20% apart. Data has
both the binary prefixes (KiB, MiB, GiB) and the decimal ones (KB, MB, GB), since
drive makers and operating systems do not use the same one.

BUILT-IN KEYPAD

The number pad is part of the app, not the system keyboard — so the layout never jumps
around, the keys are large, and nothing but digits can get in.

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases. Everything is computed on
your phone, so it works with no connection at all.

PRIVACY

UnitShift does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. The numbers you type are never saved.
See the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **应用** |
| 类别 | Tools |
| 标签 | Tools · Utilities · Productivity · Calculator · Offline |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **toolsa069@gmail.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://thriving-banoffee-396732.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的两条

1. **不写做不到的事。** 没有汇率、没有历史记录、没有自定义单位、没有小组件，
   就一句都不提。汇率尤其别写 —— 那需要联网取价，而这个包是纯离线的。
2. **文案里报出来的数都能对上。** `1 in = 0.0254 m` / `1 lb = 0.45359237 kg` /
   `1 mile = 1609.344 m` / 美制加仑 3.785 L / 英制 4.546 L —— 全部在
   `test/logic/catalog_test.dart` 里对着**权威定义值**验过（不是拿代码自己的结果对自己）。
   温度那条单独钉了一条测试：`0°C 换出来不等于 0`。
   **改了单位表就要重跑那些测试，测试挂了先改表、不要改文案。**
