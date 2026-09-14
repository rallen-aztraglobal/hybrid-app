# Play Console — 商品详情文案

> 直接复制到 Play Console → 主要商品详情。字数按 Play 的上限校过：
> 应用名 ≤30、简短说明 ≤80、完整说明 ≤4000。

## 应用名称（≤30）

```
DaySpan
```

## 简短说明（≤80）

```
Days between two dates, add or subtract days, count business days. No ads, works offline.
```

（89 字符）

## 完整说明（≤4000）

```
Three things, done properly: how far apart two dates are, what date falls N days from a given
one, and how many working days a range contains.

CORRECT ON THE AWKWARD DATES

Leap years, month ends and daylight saving are where date tools quietly go wrong. Adding one day
to the date a clock change falls on can land you on the wrong day; 31 January plus one month is
28 or 29 February, not 3 March. Those cases are covered by tests, not by hope.

WHAT YOU GET

• Days between two dates — with or without counting both end dates
• Add or subtract days from any date, optionally skipping weekends
• Count business days across any range
• See the same span as weeks, business days and weekend days at once
• Any date from 1900 to 2200

NO ADS. NO SIGN-UP. WORKS OFFLINE.

There are no adverts, no accounts and no in-app purchases.

PRIVACY

DaySpan does not ask for your name, email or phone number, and requests no camera,
microphone, location or contacts permission. See the privacy policy for the full detail.
```

## 分类与标签

| 字段 | 值 |
| --- | --- |
| 应用/游戏 | **应用** |
| 类别 | Tools |
| 标签 | Tools · Productivity · Calendar · Offline · No ads |
| 内含广告 | **否** |
| 应用内购买 | **否** |

## 联系方式

| 字段 | 值 |
| --- | --- |
| 电子邮件 | **toolsa069@gmail.com** |
| 网站 | 可留空 |
| 电话 | 留空 |
| 隐私政策 URL | `https://stellular-gumption-304771.netlify.app/` |

## 素材

见 `STORE_ASSETS.md`。上传的文件全在 `store/` 下，不要用 `store/raw/` 里的原始截图 ——
那些是 1080×2400（9:20），超出 Play 允许的 2:1，会被拒。

## 写文案时守住的一条

**不写做不到的事。** 这一条在「跨夏令时、闰年、月末都算得对」上尤其要紧：那句话是可验证的断言，
不是形容词。它由 `test/logic/` 下的测试 钉着，
删掉那条测试就必须同时删掉这句文案。
