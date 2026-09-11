# Hexa Color Sort

A portrait, single-hand casual puzzle game for Android: tap a vertical
color stack to pick up its top run of matching colors, tap another stack
to move them over, and clear five stacked same-color pieces in a row to
score and build combos.

Built with Flutter (stable channel) and Material 3, no login. Local best score
and settings are stored with `shared_preferences`.

This copy lives in the hybrid-app monorepo as a store listing package, so on top
of the game it carries the AB gate integration (`lib/gate/`, `lib/push/`,
`lib/tracking/`) shared with the other listing packages — see
[README_GATE.md](README_GATE.md). The game itself (`core/`, `game/`, `screens/`,
`widgets/`) is unchanged and never talks to a backend.

## Project structure

```
lib/
  main.dart                 App entry point, portrait lock
  app.dart                  MaterialApp / theme wiring
  core/
    constants/               App strings (for future localization) and
                              tunable gameplay constants / color palette
    theme/                   Dark Material 3 theme
    services/                SettingsService (shared_preferences),
                              HapticService, SoundService (reserved, no-op)
  game/
    models/                  Immutable data: ColorPiece, StackModel,
                              GameState, MoveRecord
    logic/                   BoardGenerator, MoveValidator,
                              DeadlockDetector, ScoringService,
                              GameController (the only stateful piece,
                              a ChangeNotifier that owns all transitions)
  screens/                   Splash, Home, Game, Result, How to Play
  widgets/                   ColourStack, ColorPieceWidget, GameBoard,
                              ScoreHeader, ComboOverlay, PauseDialog
  gate/                      AB gate: compile-time config, decision request,
                              launch gate (app entry), B-side WebView
  push/                      FCM token registration carrying the gate result
  tracking/                  AppsFlyer / Adjust attribution (no-op until keyed)
test/                        Unit tests for game/ logic and models, one gate
                              invariant test, plus one widget test
                              (Home -> Play -> Game)
```

Game rules never live in widget `build()` methods: `GameController`
performs every state transition, `MoveValidator` decides legality,
`BoardGenerator` builds the reproducible initial board,
`DeadlockDetector` detects when no legal move remains, and
`ScoringService` is a pure combo-score function. Widgets only read
`GameController.state` and call its methods.

## Requirements

- Flutter 3.x stable (developed and tested against Flutter 3.44.4 / Dart 3.12.2)
- Android SDK + accepted licenses (`flutter doctor --android-licenses`) to build/run on Android

## Running

```bash
flutter pub get
flutter run            # on a connected Android device/emulator
```

## Testing

```bash
flutter analyze
flutter test
dart format .
```

The suite covers: legal moves onto empty/matching stacks, illegal moves
(color mismatch, full destination), destination-capacity capping,
top-run clearing at 5, non-consecutive runs not clearing, combo scoring,
undo restoring board/score/combo, deadlock detection, board-generation
guarantees (at least one legal move, reproducible for a given seed), and
input locking during an in-flight move/animation — plus a widget test
that the Home screen renders and Play navigates to the Game screen.

## Building the APK

```bash
flutter build apk --debug     # debug APK, no signing required
flutter build apk --release   # release APK, needs a signing config for distribution
```

The debug APK is written to:

```
build/app/outputs/flutter-apk/app-debug.apk
```

## Sound

No audio assets ship with this version. `SoundService` is a reserved,
disabled-by-default no-op interface so a real audio backend can be wired
in later without touching game logic. The in-app "Sound" toggle persists
a preference but does not yet play anything.

## Known follow-ups

- Sound effects are not implemented (see above).
- Only Android has been built/verified; other platforms are not configured.
- Piece-flying animation position is computed analytically from the grid
  layout rather than measured `RenderBox` positions; this is correct for
  the current fixed grid but would need revisiting if the board layout
  becomes more dynamic (e.g. drag-and-drop reordering).
