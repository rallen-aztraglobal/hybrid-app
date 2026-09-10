# LinkFlow — Privacy Policy

**Last updated: 2026-09-09**

LinkFlow ("the app") is published by Aztra Global. This policy explains what the app
collects, why, and what it does not collect. It covers the Android app distributed on
Google Play under the package name `com.wrenfield.linkflow6142`.

Contact: **appaztra@outlook.com**

---

## The short version

- The app has **no accounts and no sign-in**. We never ask for your name, email,
  phone number, or address.
- Your puzzle progress stays **on your device only**.
- The app does include analytics and attribution SDKs that report a **device
  identifier**. That is the only category of personal data that leaves your device.

---

## What we collect

### Device and app identifiers

The app bundles the following third-party SDKs, which collect identifiers so we can
measure installs and app usage:

| SDK | What it reports |
| --- | --- |
| AppsFlyer | Google Advertising ID, install referrer, app open and session events |
| Adjust | Google Advertising ID, app open and session events |
| Firebase Cloud Messaging | A push registration token, device model, OS version |

These are used to understand where installs come from and how many people open the
app. They are **not** used to build an advertising profile about you inside the app,
and the app shows no ads.

### Network requests

When the app starts it makes a request to our server to fetch its configuration. Like
any internet request, that request carries your **IP address**. We use it to determine
the country the request came from. We do not store it as part of a user profile and
we do not use it to locate you more precisely than country level.

### What we never collect

Name · Email address · Phone number · Postal address · Photos · Videos · Files ·
Contacts · Calendar · Precise location (GPS) · Health data · Financial information ·
Messages · Audio recordings · Web browsing history · List of installed apps.

The app requests only the `INTERNET` permission in its own manifest. It has no camera,
microphone, location, contacts, or storage permissions.

---

## Stored only on your device

The app saves your progress in each difficulty tier, using Android's
standard app-preferences storage:

| Key | Value |
| --- | --- |
| `linkflow_solved_<tier>` | puzzles solved in that difficulty tier |
| `linkflow_level_<tier>` | which level you are on in that tier |

where `<tier>` is one of `normal`, `hard`, `expert`, `master` — eight integers in total.

They are never uploaded, never attached to analytics events,
and never sent to our server. Uninstalling the app deletes them.

---

## Sharing

We share the identifiers listed above with AppsFlyer, Adjust, and Google, acting as
our data processors for attribution and messaging. We do not sell your data.

- AppsFlyer privacy policy: https://www.appsflyer.com/legal/privacy-policy/
- Adjust privacy policy: https://www.adjust.com/terms/privacy-policy/
- Google privacy policy: https://policies.google.com/privacy

## Security

All network traffic uses HTTPS.

## Children

The app is not directed at children under 13 and we do not knowingly collect data from
them.

## Your choices

- You can reset your advertising identifier, or opt out of ad personalisation, in
  Android **Settings → Privacy → Ads**.
- You can delete everything the app stores locally by uninstalling it, or via
  **Settings → Apps → LinkFlow → Storage → Clear data**.
- To request deletion of data held by us or our processors, email the address at the
  top of this document.

## Changes

If this policy changes we will update the date at the top and publish the new version
at the same URL.
