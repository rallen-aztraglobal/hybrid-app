import 'package:flutter_test/flutter_test.dart';
import 'package:hexa_color_sort/game/models/color_piece.dart';
import 'package:hexa_color_sort/game/models/stack_model.dart';
import 'package:hexa_color_sort/game/logic/deadlock_detector.dart';

StackModel _stack(int id, List<int> colors, {int capacity = 8}) {
  return StackModel(
    id: id,
    capacity: capacity,
    pieces: colors.map(ColorPiece.new).toList(),
  );
}

void main() {
  group('DeadlockDetector', () {
    test(
      'reports a deadlock when every stack is full with mismatched tops',
      () {
        final stacks = [
          _stack(0, List.filled(8, 0)),
          _stack(1, List.filled(8, 1)),
          _stack(2, List.filled(8, 2)),
        ];

        expect(DeadlockDetector.isDeadlocked(stacks), isTrue);
      },
    );

    test('is not deadlocked when an empty stack exists alongside pieces', () {
      final stacks = [
        _stack(0, [0, 0, 0]),
        _stack(1, []),
        _stack(2, List.filled(8, 2)),
      ];

      expect(DeadlockDetector.isDeadlocked(stacks), isFalse);
    });

    test('is not deadlocked when two stacks share a top color with room', () {
      final stacks = [
        _stack(0, [1, 0]),
        _stack(1, [2, 0, 0, 0]),
        _stack(2, List.filled(8, 2)),
      ];

      expect(DeadlockDetector.isDeadlocked(stacks), isFalse);
    });
  });
}
