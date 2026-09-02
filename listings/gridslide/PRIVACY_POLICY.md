# Privacy Policy for GridSlide

> 本文是隐私政策正文（英文，面向用户与 App Review）。`store/privacy-policy.html` 是同一份
> 内容的可托管页面版本，**两者必须保持一致**，改一处就改另一处。
> App Store Connect 必填的是那个页面的**公网 URL**，仓库里的 md 不算。
>
> 内容依据：`Core/Services/RecordStore.swift`（成绩存 UserDefaults）、
> `Core/Services/SettingsStore.swift`（棋盘尺寸与震动开关）、
> `Core/Services/TrackingService.swift`（AppsFlyer / Adjust）、
> `Push/PushService.swift`（FCM token + 系统版本）、
> `Gate/GateService.swift`（启动配置请求 payload）、
> `MVP/Settings/SettingsModule.swift`（清空成绩、政策链接）。

Effective date: 2026-09-02

GridSlide ("the app") is a number slide puzzle game for iPhone. This policy explains
what stays on your device, what leaves it, and who receives it.

## Stored only on your device

What the app keeps about you is short: for each board size, your fewest moves, your
fastest time and how many times you have solved it, plus two settings — your
preferred board size and whether haptic feedback is on. That is the whole list.

All of it is kept in the app's own storage on this iPhone. It is never uploaded,
never synced to any account or cloud service, and is not sent to us or to anyone
else. It is not attached to any advertising or analytics event, and nobody else can
see it — there is no leaderboard, no profile and no way to share a score from inside
the app. Deleting the app deletes it with the app.

You can also clear it yourself at any time: Settings › Reset all records erases the
best moves and best times for every board size, after asking you to confirm.

There is no sign-up and no login. The app has no account for you to create, so there
is no name, email address or password associated with anything you play.

## Collected and shared with third parties

**Measurement identifiers and installation data.** The app includes the AppsFlyer and
Adjust measurement SDKs. They collect your device's identifiers — the Identifier for
Advertisers (only if you allow tracking when asked) and the Identifier for Vendor —
along with device model, operating system version, app version, and app open, session
and attribution events, and send them to AppsFlyer and Adjust so that we can
attribute installations to marketing sources. They are not given your scores or
anything else about how you play.

**Notification token.** The app includes Firebase Cloud Messaging. When notifications
are set up, it obtains a Firebase registration token for this installation and sends
it, with your device's operating system version, to our server so that notifications
can be delivered to this device.

**Configuration request.** On launch the app requests its configuration from our
server. The request contains the platform name, the application identifier, and your
device's time zone name (for example `Asia/Manila`). As with any internet request,
our server also receives the originating IP address.

All of these requests are sent over HTTPS.

## Not collected

The app does not request or collect your name, email address, phone number, postal
address, contacts, calendar, photos, files, microphone or camera input, or your
location — it asks for no location permission and cannot read your GPS position. It
has no access to your payment details, because it has nothing to sell: there are no
advertising placements, no in-app purchases and no subscriptions. There is no
leaderboard, no chat and no social feature of any kind, and the game itself needs no
connection to play.

## Third parties that receive data

- AppsFlyer — installation attribution and marketing measurement
- Adjust — installation attribution and marketing measurement
- Google (Firebase Cloud Messaging) — notification delivery

Each processes the data it receives under its own privacy policy.

## Your choices

- **Tracking:** if the app asks for permission to track, you can decline. You can also
  change the answer later in iOS Settings › Privacy & Security › Tracking. The game
  works exactly the same either way.
- **Notifications:** decline the permission prompt, or turn notifications off later in
  iOS Settings › Notifications › GridSlide.
- **Your records:** use Settings › Reset all records inside the app, or delete the app
  from your iPhone. Either removes everything it stored.

## Children

The app is not directed to children, and we do not knowingly collect data from
children.

## Data deletion requests

Your records are on your device only, so you can delete them yourself at any time
using Settings › Reset all records or by deleting the app. To request deletion of the
installation data described above, contact us at the address below and include the
approximate date of installation.

## Changes to this policy

Material changes will be published here with a new effective date.

## Contact

TODO_SUPPORT_EMAIL

> 上架前把 `TODO_SUPPORT_EMAIL` 换成真实可达的邮箱，并同步改
> `store/privacy-policy.html`、`APP_STORE_LISTING.md` 与
> `GridSlide/Core/Services/SettingsStore.swift`。
> **不要复用其余上架包已公开的邮箱**，理由见 `STORE_ASSETS.md`。
>
> 另外两处口径提醒：
> - "only if you allow tracking when asked" 这句是**按路线 A（补上 ATT 请求）**写的。
>   若最终走路线 B（维持现状、暂不启用归因），要把这句与「Your choices → Tracking」
>   一并改掉，见 `APP_PRIVACY.md` 末节。
> - 政策里写的是「SDK 会收集」的完整口径，而当前构建里 AppsFlyer 与 Adjust 因为
>   `appsFlyerAppleAppID` / `adjustAppToken` 仍是占位符**实际不会初始化**。
>   政策宽于实现是安全方向，不要反过来。
