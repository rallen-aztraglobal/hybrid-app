# Play Console — 问卷答案（可直接粘贴）

> 本文是 Play Console 提交流程里那些自由作答题的答案，英文段落可原样粘贴。
> 中文引用块与末尾清单是内部说明，**不要**粘进 Console。
> Data safety 表单另见 `DATA_SAFETY.md`。

---

## What SDKs does your app use and why?

TileFit is a Flutter application. Its runtime dependencies are:

- `flutter` — UI framework
- `shared_preferences` — stores the player's best score locally on the device
- `webview_flutter` — renders web content the app is configured to show
- `url_launcher` — opens links in the user's browser
- `flutter_timezone` — reads the device time zone name, sent with the
  configuration request
- `firebase_core`, `firebase_messaging` — notifications
- `appsflyer_sdk`, `adjust_sdk` — installation attribution

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
requested at runtime and the game works normally if it is denied.

---

## Your app's core functionality

TileFit is a single-player block puzzle in portrait orientation. The board is an
8×8 grid. Each round the player is dealt three shapes and drags them onto empty
squares; a row or column that becomes completely filled is cleared. When all
three shapes have been placed, three more are dealt. The round ends when none of
the remaining shapes can fit anywhere on the board. Placing a shape scores one
point per square, and clearing lines scores an additional bonus that grows with
the number of lines cleared at once. Shapes cannot be rotated.

The only thing the app remembers is the player's highest score, stored locally on
the device. There is no account, no leaderboard, no chat and no user-generated
content. All gameplay runs entirely on the device and works with no network
connection.

---

## Does your app function differently based on a user's geolocation or language?

Yes. Some content shown in the app is loaded from our server, and that content
can vary by region. The game itself and its user interface are identical for all
users. The app does not read device location; where regional differences apply
they are determined server-side from the request.

---

## Have you uploaded all Proof of Permission for any intellectual property that appears in your app?

**Select:** No third party intellectual property appears in my app

Supporting note: the app uses original layout and artwork created for this
project, together with standard Flutter and Material UI components and Material
icon glyphs. The puzzle uses plain geometric shapes and no third-party brands,
characters, licensed music or other rights-holder content.

---

## Please select the statement that applies to you (login wall)

**Select:** I do not have any content locked behind a login wall

The whole app is reachable immediately after launch. There is no account, no
sign-in and no paywall, so no demo video is required.

---

## Ads declaration

**Select:** No, my app does not contain ads

The app shows no advertising placements and integrates no ad-serving SDK. The
AppsFlyer and Adjust SDKs are used for installation attribution only.

---

## Developer account ownership

按真实情况回答。若账号由你本人注册并管理，答 No；若他人代注册，说明是谁、原因，
并写明你是本应用所有者、负责更新与政策合规。

---

## 提交前清单

- [x] Adjust 后台已建本包条目（App name `TileFit`），App Token 与 `OpenBLanding`
      event token 均已填进 `lib/gate/gate_config.dart`。`adjustContentViewToken` 仍为空
      —— 内开进 B 面时 Adjust 侧不发事件，与 colorstack / hexacolorsort 现状一致，
      属已知缺口而非本包遗漏
- [ ] `android/app/google-services.json` 已放入（且**已裁剪**成只留本包 client）
- [x] 已生成正式 keystore（`android/tilefit8264.jks`，alias `tilefit8264`），已用
      `keytool -printcert -jarfile` 确认 release AAB 的签名主体是
      `CN=TileFit, O=Emberlane`，不是 debug 签名
- [ ] 服务端 listing 条目已建，bundleId = `com.emberlane.tilefit8264`
- [ ] 隐私政策已托管到公网 URL 并填进 Console（仓库里的 md 不算），**且不与其他上架包共用同一 URL**
- [x] 支持邮箱已填为 `tilefit@outlook.com`（`PRIVACY_POLICY.md` 与
      `store/privacy-policy.html` 两处都已同步，本包专用、不与其他上架包共用）
- [ ] `PLAY_STORE_LISTING.md` 的商店名已确认在 Play 上可用
- [ ] Data safety 按 `DATA_SAFETY.md` 逐项填完，其中 Approximate location 的待定口径已定
- [x] 商店素材已产出（`store/icon-512.png` · `store/feature-graphic.png` · 三张 1200×2400 截图），
      **待上传**；规格与可选补拍见 `STORE_ASSETS.md`
- [x] release 包已在 Android 模拟器上实跑（进 A 面不崩、落子计分与最高分存档均正确）
- [ ] release 包在**真机**上再跑一次（目前只在模拟器上跑过）
- [ ] 内容分级问卷已完成
