# Google Play Console — Form Answers for Color Stack

Copy the answers below into the Play Console Help form.

---

## Did somebody register this developer account on your behalf? If so, please explain why.

**Answer (if you registered it yourself):**

No. I registered and manage this Google Play developer account myself. I am the owner and developer of Color Stack.

**Answer (only use if someone else registered it for you):**

Yes. [Name/company] registered the account on my behalf because [brief reason, e.g. they set up the Play Console account while helping me publish my first app]. I am the owner of Color Stack and I manage all app updates, store listing, and policy compliance.

---

## Your app's core functionality

**Suggested answer (if a text field asks you to describe your app):**

Color Stack is an original arcade puzzle game. Players place colored blocks onto one of five stacks, score points by matching the top color of a stack, and try to beat a 60-second timer within 30 moves. The app provides a complete, self-contained gameplay experience: home screen, active game, score tracking, and replay — with no login, no subscriptions, and no external content feeds.

The game is not a copy or wrapper of another app. It was built as a standalone Flutter game with original UI, scoring rules, and session flow. All gameplay happens locally on the device.

---

## Does your app function differently based on a user's geolocation or language? If yes, why?

**Answer:**

No. Color Stack works the same for all users regardless of geolocation or language. The app does not use location services, does not change content by country or region, and does not provide different features based on device language. Gameplay, scoring, and UI are identical everywhere.

---

## Have you uploaded all Proof of Permission for any intellectual property that appears in your app?

**Select:**

**No third party intellectual property appears in my app**

**Supporting note (if needed):**

Color Stack uses only original game design, standard Flutter/Material UI components, and app assets created for this project. It does not include third-party brands, licensed characters, copyrighted music, or content that requires permission from another rights holder.

---

## Please select the statement that applies to you

**Select:**

**I do not have any content locked behind a login wall.**

**Note:** No demo video upload is required for this option. The entire app is accessible immediately after launch: tap PLAY, complete a round, view your score, and restart or return home.

---

## What SDKs does your app use and why?

**Answer:**

Color Stack uses only the official Flutter SDK and the standard Android embedding libraries that ship with Flutter. These are used to render the game UI and run the app on Android.

The app does **not** use:
- advertising SDKs
- analytics SDKs
- social login SDKs
- payment SDKs
- Firebase or other backend SDKs
- location, contacts, or other sensitive-permission SDKs

The only production dependency is the Flutter framework (`flutter` SDK). There are no additional third-party runtime packages in `pubspec.yaml`.

---

## Explain how you ensure that any 3rd party code and SDKs used in your app comply with our policies

**Answer:**

I keep third-party code to a minimum and use only the official Flutter SDK from Google. Before each release, I review `pubspec.yaml` to confirm no analytics, advertising, tracking, or data-collection libraries are added.

Color Stack does not collect, store, or share personal user data. It does not request dangerous permissions in the Android manifest, does not require account creation, and does not transmit user information to external servers.

I maintain a published privacy policy stating that no personal data is collected and no third-party tracking is used. I also test each release to confirm the app behavior matches the store listing and privacy disclosures.

Because the app uses only Flutter’s standard framework and contains no additional commercial SDKs, policy compliance is maintained by limiting dependencies, avoiding user-data collection, and publishing accurate privacy and functionality disclosures.

---

## Quick checklist before submitting

- [ ] Select **No third party intellectual property appears in my app**
- [ ] Select **I do not have any content locked behind a login wall**
- [ ] Do **not** upload a login-wall demo video (not needed)
- [ ] Confirm your privacy policy URL is live in Play Console
- [ ] Use answers that match your real account ownership situation
