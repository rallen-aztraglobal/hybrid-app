# Play Console — 各类问卷答案（Hexa Color Sort）

> **不要照抄 colorstack 那份。** 它写着「不使用广告/分析 SDK、不使用 Firebase、
> pubspec 里没有任何第三方运行时依赖、行为不随地理位置变化」——这四条对本包和对
> 现在的 colorstack 都不成立。下面是按本包实际情况写的。

---

## What SDKs does your app use and why?

Hexa Color Sort is a Flutter app. Its runtime dependencies are:

- `flutter` — UI framework
- `shared_preferences` — stores the local best score and the sound / vibration
  toggles on the device only
- `webview_flutter` — renders web content the app is configured to display
- `url_launcher` — opens links in the user's browser
- `flutter_timezone` — reads the device IANA time zone name, sent with the
  configuration request
- `firebase_core` + `firebase_messaging` — push notifications (Firebase Cloud
  Messaging)
- `appsflyer_sdk` — install attribution / marketing measurement
- `adjust_sdk` — install attribution / marketing measurement
- `cupertino_icons` — icon assets

The app has no advertising (ad-serving) SDK, no payment SDK, no social login SDK,
and requests no location, contacts, camera, microphone or storage permissions.

---

## Explain how you ensure that any 3rd party code and SDKs used in your app comply with our policies

All third-party SDKs are current releases obtained from pub.dev, published by the
vendors themselves (Google, AppsFlyer, Adjust). Before each release we review
`pubspec.yaml` and the merged Android manifest so that the store listing, the
privacy policy and the Data safety declaration match what the build actually
contains — including SDK-injected permissions such as
`com.google.android.gms.permission.AD_ID`.

The advertising identifier is used only for install attribution, is disclosed in
our privacy policy and Data safety form, and users can reset it or opt out of
personalisation in Android settings. Notification permission is requested at
runtime and the app functions fully if it is denied.

---

## Your app's core functionality

Hexa Color Sort is an original portrait puzzle game. The player taps a stack to
pick up the run of matching colours on top, taps another stack to move them, and
clears five of the same colour stacked consecutively; clears chained within a
short window build a combo multiplier. The build includes home, how-to-play,
active game, pause and result screens, undo, restart, a locally stored best
score, and sound / vibration toggles. Board generation guarantees at least one
legal move, and a deadlock detector ends the round when none remains.

The game logic runs entirely on the device. It is not a wrapper around another
app or website.

---

## Have you uploaded all Proof of Permission for any intellectual property that appears in your app?

**Select:** No third party intellectual property appears in my app

Supporting note: the game uses original rules, layout and artwork built for this
project, plus standard Flutter / Material UI components and Material icon glyphs.
No third-party brands, characters, licensed music or other rights-holder content
appears in the app.

---

## Please select the statement that applies to you (login wall)

**Select:** I do not have any content locked behind a login wall

The whole app is reachable immediately after launch — there is no account, no
sign-in and no paywall.

---

## Does your app function differently based on a user's geolocation or language?

> ⚠️ **这一条必须你来定，我不替你填。**
>
> 事实是：本包的 AB 面网关**就是按请求来源 IP 的国家（外加可选的时区 / IP 规则）
> 决定下发什么内容**（见 `docs/admin/09-listing.md` 与 ADR-0014）。所以按事实回答，
> 这题是 **Yes**。
>
> 据此写的诚实答案：
>
> > Yes. The app requests a display configuration from our server on launch, and
> > that configuration can differ by country, so users in different regions may
> > see different content. Gameplay, scoring and UI are identical everywhere;
> > only the configured content differs. The app itself does not read device
> > location — the country is derived server-side from the request IP.
>
> colorstack 那份填的是 **No**（「不使用定位服务、不按国家或地区改变内容」）。
> 那是在加网关**之前**写的，现在已与事实不符，没有随之更新。
>
> 明知不符仍按 No 提交，属于向 Google 作虚假陈述 —— 被判定为规避审核的后果是
> 应用下架、并可能连带开发者账号被封。这个权衡不该由我替你做，也不该悄悄糊过去：
> 请你和法务/渠道负责人确认口径后再填。

---

## Developer account ownership（如被问到）

按真实情况回答。若账号是你本人注册并管理，答 No；若是他人代注册，说明是谁、为什么，
并写明你是本应用的所有者、负责更新与政策合规。

---

## 提交前清单

- [ ] 隐私政策已托管到公网 URL 并填进 Play Console（仓库里的 md 不算）
- [ ] Data safety 按 `DATA_SAFETY.md` 逐项填（含 Advertising ID / 归因事件）
- [ ] Approximate location 这一条已定口径
- [ ] 「是否按地理位置改变行为」这一条已定口径（见上面的 ⚠️）
- [ ] 内容分级问卷已完成
- [ ] 商店文案、截图、特色图已上传（见 `STORE_ASSETS.md`）
- [ ] release 包用正式 keystore 签名，不是 debug 签名
