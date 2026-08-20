# Privacy Policy for Hexa Color Sort

Effective date: TODO_SET_ON_PUBLISH

This policy describes what Hexa Color Sort ("the app") collects and why.

> 内部说明（不要发布这一段）
> 本文按 App 实际集成的 SDK 写。**不要照抄 colorstack 的 PRIVACY_POLICY.md** ——
> 那份写着「不使用任何第三方分析/广告/追踪 SDK」，而它和本包一样装了 AppsFlyer、
> Adjust、Firebase Messaging，那份声明已与事实不符。
> 事实依据：pubspec.yaml 的依赖列表 + 合并后 AndroidManifest 含
> `com.google.android.gms.permission.AD_ID`、`BIND_GET_INSTALL_REFERRER_SERVICE`、
> `com.google.android.c2dm.permission.RECEIVE`（实机 dumpsys 验证过）。

## Data stored only on your device
Your best score and your sound / vibration preferences are saved locally with
Android's app preferences storage. They are never transmitted anywhere and are
removed when you uninstall the app.

## Data collected and shared with third parties

**Advertising identifier and install attribution.** The app includes the
AppsFlyer and Adjust mobile measurement SDKs. They collect your device's Google
Advertising ID (GAID), device model, operating system version, app version,
install referrer, and app open / session events, and send them to AppsFlyer and
Adjust so we can tell which marketing source an install came from. The app
declares `com.google.android.gms.permission.AD_ID` for this purpose.

**Push notification token.** The app includes Firebase Cloud Messaging. When you
launch the app it obtains a Firebase registration token for this installation and
sends it to our server together with your device's OS version, so we can deliver
notifications to this device.

**Configuration request.** On launch the app asks our server for its display
configuration. The request contains the platform (android), the application id,
and your device's IANA time zone name (for example `Asia/Manila`). As with any
internet request, our server also observes the originating IP address, which is
used to derive an approximate country. We do not request Android location
permission and the app has no access to GPS or precise location.

## What we do not collect
The app does not ask for or collect your name, email address, phone number,
contacts, photos, files, precise location, or any account credentials. There is
no sign-up and no login.

## Third parties that receive data
- AppsFlyer — install attribution and marketing measurement
- Adjust — install attribution and marketing measurement
- Google (Firebase Cloud Messaging) — push notification delivery

Each processes data under its own privacy policy.

## Your choices
- Notifications: you can deny the notification permission when prompted, or turn
  notifications off later in Android Settings › Apps › Hexa Color Sort.
- Advertising ID: you can reset it or opt out of personalisation in
  Android Settings › Privacy › Ads.
- Uninstalling the app removes all locally stored data.

## Children
The app is not directed to children and we do not knowingly collect data from
children.

## Data deletion
To request deletion of data associated with your device, contact us at the
address below and include the date and approximate time of installation.

## Changes
Material changes to this policy will be published here with a new effective date.

## Contact
TODO_FILL_REAL_SUPPORT_MAILBOX
