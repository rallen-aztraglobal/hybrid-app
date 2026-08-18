import 'package:flutter_test/flutter_test.dart';
import 'package:hexa_color_sort/game/models/color_piece.dart';
import 'package:hexa_color_sort/game/models/stack_model.dart';

void main() {
  group('StackModel.topRunLength', () {
    test('non-consecutive same-color pieces are not counted as one run', () {
      // Five pieces of color 0 exist in this stack, but they are not all
      // consecutive from the top, so the top run must not reach 5.
      final stack = StackModel(
        id: 0,
        capacity: 8,
        pieces: const [
          ColorPiece(0),
          ColorPiece(0),
          ColorPiece(0),
          ColorPiece(0),
          ColorPiece(1),
          ColorPiece(0),
        ],
      );

      expect(stack.topRunLength, 1);
    });

    test(
      'consecutive same-color pieces from the top are counted correctly',
      () {
        final stack = StackModel(
          id: 0,
          capacity: 8,
          pieces: const [
            ColorPiece(1),
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
          ],
        );

        expect(stack.topRunLength, 5);
      },
    );

    test('empty stack has a zero top run', () {
      const stack = StackModel(id: 0, capacity: 8, pieces: []);
      expect(stack.topRunLength, 0);
      expect(stack.isEmpty, isTrue);
      expect(stack.topColorId, isNull);
    });
  });
}
