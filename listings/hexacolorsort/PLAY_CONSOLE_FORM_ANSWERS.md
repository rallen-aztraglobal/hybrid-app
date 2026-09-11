# Play Console — 问卷答案（可直接粘贴）

内部讨论、取证依据、未定口径见 `SUBMISSION_NOTES.md`。本文只放答案本身。

---

## What SDKs does your app use and why?

Hexa Color Sort is a Flutter application. Its runtime dependencies are:

- `flutter` — UI framework
- `shared_preferences` — stores the local best score and the sound and vibration
  toggles on the device only
- `webview_flutter` — renders web content the app is configured to show
- `url_launcher` — opens links in the user's browser
- `flutter_timezone` — reads the device time zone name, sent with the
  configuration request
- `firebase_core`, `firebase_messaging` — notifications
- `appsflyer_sdk`, `adjust_sdk` — installation attribution
- `cupertino_icons` — icon assets

The app contains no ad-serving SDK, no payment SDK and no social login SDK, and
requests no location, contacts, camera, microphone or storage permissions.

---

## Explain how you ensure that any 3rd party code and SDKs used in your app comply with our policies

All third-party SDKs are current releases published by their vendors (Google,
AppsFlyer, Adjust) and obtained from pub.dev. Before each release we review
`pubspec.yaml` and the merged Android manifest so that the store listing, the
privacy policy and the Data safety declaration match what the build contains,
including permissions injected by SDKs such as
`com.google.android.gms.permission.AD_ID`.

The advertising identifier is used only for installation attribution, is
disclosed in our privacy policy and Data safety form, and users can reset it or
opt out of personalisation in Android settings. The notification permission is
requested at runtime and the game is fully playable if it is denied.

---

## Your app's core functionality

Hexa Color Sort is an original portrait puzzle game. The player taps a stack to
pick up the run of matching colours on top, taps another stack to move them, and
clears five of the same colour stacked consecutively; clears chained within a
short window build a combo multiplier. The build includes home, how-to-play,
active game, pause and result screens, undo, restart, a locally stored best
score, and sound and vibration toggles. Board generation guarantees at least one
legal move and a deadlock detector ends the round when none remains.

The game logic runs entirely on the device.

---

## Does your app function differently based on a user's geolocation or language?

Yes. Some content shown in the app is loaded from our server, and that content
can vary by region. Gameplay, scoring and the user interface are identical for
all users. The app does not read device location; where regional differences
apply they are determined server-side from the request.

---

## Have you uploaded all Proof of Permission for any intellectual property that appears in your app?

**Select:** No third party intellectual property appears in my app

Supporting note: the game uses original rules, layout and artwork created for
this project, together with standard Flutter and Material UI components and
Material icon glyphs. No third-party brands, characters, licensed music or other
rights-holder content appears in the app.

---

## Please select the statement that applies to you (login wall)

**Select:** I do not have any content locked behind a login wall

The whole app is reachable immediately after launch. There is no account, no
sign-in and no paywall, so no demo video is required.

---

## Developer account ownership

按真实情况回答。若账号由你本人注册并管理，答 No；若他人代注册，说明是谁、原因，
并写明你是本应用所有者、负责更新与政策合规。

---

## 提交前清单

- [ ] 隐私政策已托管到公网 URL 并填进 Console（仓库里的 md 不算）
- [ ] `PRIVACY_POLICY.md` 的两个 TODO 已替换为真实邮箱与生效日期
- [ ] Data safety 按 `DATA_SAFETY.md` 逐项填完
- [ ] `SUBMISSION_NOTES.md` 里标出的待定口径已定
- [ ] 内容分级问卷已完成
- [ ] 文案、截图、特色图已上传（见 `STORE_ASSETS.md`）
- [ ] 上传的是正式 keystore 签名的包，不是 debug 签名
