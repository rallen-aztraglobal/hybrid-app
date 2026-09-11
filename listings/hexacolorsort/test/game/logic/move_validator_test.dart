import 'package:flutter_test/flutter_test.dart';
import 'package:hexa_color_sort/game/models/color_piece.dart';
import 'package:hexa_color_sort/game/models/stack_model.dart';
import 'package:hexa_color_sort/game/logic/move_validator.dart';

StackModel _stack(int id, List<int> colors, {int capacity = 8}) {
  return StackModel(
    id: id,
    capacity: capacity,
    pieces: colors.map(ColorPiece.new).toList(),
  );
}

void main() {
  group('MoveValidator', () {
    test('allows moving a run onto an empty stack', () {
      final from = _stack(0, [0, 0, 0]);
      final to = _stack(1, []);

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isTrue);
      expect(result.moveCount, 3);
    });

    test('allows moving onto a stack with a matching top color', () {
      final from = _stack(0, [1, 2, 2]);
      final to = _stack(1, [3, 2]);

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isTrue);
      expect(result.moveCount, 2);
    });

    test('rejects moving onto a stack with a different top color', () {
      final from = _stack(0, [1, 1]);
      final to = _stack(1, [0]);

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isFalse);
    });

    test('rejects moving onto a full stack', () {
      final from = _stack(0, [0]);
      final to = _stack(1, List.filled(8, 0), capacity: 8);

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isFalse);
    });

    test('caps the moved amount at the destination remaining capacity', () {
      final from = _stack(0, [0, 0, 0, 0, 0]); // run of 5
      final to = _stack(1, List.filled(6, 0), capacity: 8); // room for 2

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isTrue);
      expect(result.moveCount, 2);
    });

    test('rejects moving from an empty stack', () {
      final from = _stack(0, []);
      final to = _stack(1, []);

      final result = MoveValidator.evaluate(from: from, to: to);

      expect(result.isLegal, isFalse);
    });
  });
}
