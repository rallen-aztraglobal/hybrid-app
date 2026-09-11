import 'package:flutter_test/flutter_test.dart';
import 'package:hexa_color_sort/game/logic/board_generator.dart';
import 'package:hexa_color_sort/game/logic/deadlock_detector.dart';

void main() {
  group('BoardGenerator', () {
    test('produces a board with at least one legal move available', () {
      final stacks = BoardGenerator.generate(seed: 12345);

      expect(DeadlockDetector.hasAnyLegalMove(stacks), isTrue);
    });

    test('produces at least one empty stack for movement space', () {
      final stacks = BoardGenerator.generate(seed: 999);

      expect(stacks.any((s) => s.isEmpty), isTrue);
    });

    test('the same seed always produces the same board', () {
      final first = BoardGenerator.generate(seed: 42);
      final second = BoardGenerator.generate(seed: 42);

      expect(first, equals(second));
    });

    test('different seeds can produce different boards', () {
      final first = BoardGenerator.generate(seed: 1);
      final second = BoardGenerator.generate(seed: 2);

      expect(first, isNot(equals(second)));
    });
  });
}
