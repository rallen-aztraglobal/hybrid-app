import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:colorstack5821/app.dart';
import 'package:colorstack5821/models/game_stats.dart';

/// HomeScreen and GameScreen each run a decorative AnimationController with
/// `..repeat(reverse: true)`, which never lets the frame scheduler go idle.
/// `pumpAndSettle()` keeps pumping frames until it does, which drags the
/// fake clock forward and fast-forwards the game's own `Timer.periodic`
/// round timer along with it. Use a bounded pump instead so navigation
/// settles without racing the round timer.
Future<void> openGame(WidgetTester tester) async {
  await tester.tap(find.text('PLAY'));
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 400));
}

void main() {
  setUp(GameStats.reset);

  testWidgets('shows play button on home screen', (WidgetTester tester) async {
    await tester.pumpWidget(const ColorStackApp());

    expect(find.text('PLAY'), findsOneWidget);
    expect(find.text('COLOR STACK'), findsOneWidget);
  });

  testWidgets('starts game and updates score after placing a color', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const ColorStackApp());

    await openGame(tester);

    expect(find.text('Score'), findsOneWidget);
    expect(find.text('Moves: 0 / 30'), findsOneWidget);

    await tester.tap(find.byKey(const Key('stack-0')));
    await tester.pump();

    expect(find.text('Moves: 1 / 30'), findsOneWidget);
  });

  testWidgets('finishes the round after the move limit', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const ColorStackApp());

    await openGame(tester);

    final stackTarget = find.byKey(const Key('stack-0'));
    for (var i = 0; i < 30; i++) {
      await tester.tap(stackTarget);
      await tester.pump(const Duration(milliseconds: 16));
    }

    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));

    expect(find.text('Round Complete'), findsOneWidget);
    expect(find.text('RESTART'), findsOneWidget);
  });
}
