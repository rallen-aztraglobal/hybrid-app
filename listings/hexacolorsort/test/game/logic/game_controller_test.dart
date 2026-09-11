import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:hexa_color_sort/game/logic/game_controller.dart';
import 'package:hexa_color_sort/game/models/color_piece.dart';
import 'package:hexa_color_sort/game/models/stack_model.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  group('GameController clearing and scoring', () {
    test('a top run reaching five pieces is cleared and scored', () async {
      final stacks = [
        StackModel(id: 0, capacity: 8, pieces: const [ColorPiece(0)]),
        StackModel(
          id: 1,
          capacity: 8,
          pieces: const [
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
          ],
        ),
        StackModel(
          id: 2,
          capacity: 8,
          pieces: const [ColorPiece(1), ColorPiece(1)],
        ),
      ];
      final controller = GameController(
        initialStacksOverride: stacks,
        moveAnimationDuration: Duration.zero,
        clearAnimationDuration: Duration.zero,
      );

      await controller.selectStack(0);
      await controller.selectStack(1);

      expect(controller.state.stacks[0].isEmpty, isTrue);
      expect(controller.state.stacks[1].isEmpty, isTrue);
      expect(controller.state.score, 100);
      expect(controller.state.comboCount, 1);
      expect(controller.state.maxCombo, 1);
    });

    test(
      'two clears within the combo window escalate the combo score',
      () async {
        final stacks = [
          StackModel(id: 0, capacity: 8, pieces: const [ColorPiece(0)]),
          StackModel(
            id: 1,
            capacity: 8,
            pieces: const [
              ColorPiece(0),
              ColorPiece(0),
              ColorPiece(0),
              ColorPiece(0),
            ],
          ),
          StackModel(id: 2, capacity: 8, pieces: const [ColorPiece(1)]),
          StackModel(
            id: 3,
            capacity: 8,
            pieces: const [
              ColorPiece(1),
              ColorPiece(1),
              ColorPiece(1),
              ColorPiece(1),
            ],
          ),
        ];
        final fixedNow = DateTime(2024, 1, 1);
        final controller = GameController(
          initialStacksOverride: stacks,
          moveAnimationDuration: Duration.zero,
          clearAnimationDuration: Duration.zero,
          clock: () => fixedNow,
        );

        await controller.selectStack(0);
        await controller.selectStack(1);
        expect(controller.state.comboCount, 1);
        expect(controller.state.score, 100);

        await controller.selectStack(2);
        await controller.selectStack(3);
        expect(controller.state.comboCount, 2);
        expect(controller.state.score, 300);
        expect(controller.state.maxCombo, 2);
      },
    );
  });

  group('GameController undo', () {
    test('undo restores the pre-move board, score, and combo', () async {
      final stacks = [
        StackModel(id: 0, capacity: 8, pieces: const [ColorPiece(0)]),
        StackModel(
          id: 1,
          capacity: 8,
          pieces: const [
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
            ColorPiece(0),
          ],
        ),
        StackModel(
          id: 2,
          capacity: 8,
          pieces: const [ColorPiece(1), ColorPiece(1)],
        ),
      ];
      final controller = GameController(
        initialStacksOverride: stacks,
        moveAnimationDuration: Duration.zero,
        clearAnimationDuration: Duration.zero,
      );

      expect(controller.canUndo, isFalse);

      await controller.selectStack(0);
      await controller.selectStack(1);

      expect(controller.state.score, 100);
      expect(controller.canUndo, isTrue);

      controller.undo();

      expect(controller.state.score, 0);
      expect(controller.state.comboCount, 0);
      expect(controller.state.maxCombo, 0);
      expect(controller.state.stacks[0].pieces.length, 1);
      expect(controller.state.stacks[1].pieces.length, 4);
      expect(controller.canUndo, isFalse);
    });
  });

  group('GameController input locking', () {
    test(
      'taps during an in-flight move are ignored, preserving piece count',
      () async {
        final stacks = [
          StackModel(
            id: 0,
            capacity: 8,
            pieces: const [ColorPiece(0), ColorPiece(0)],
          ),
          StackModel(id: 1, capacity: 8, pieces: const []),
          StackModel(id: 2, capacity: 8, pieces: const []),
        ];
        final controller = GameController(
          initialStacksOverride: stacks,
          moveAnimationDuration: const Duration(milliseconds: 20),
          clearAnimationDuration: Duration.zero,
        );

        await controller.selectStack(0);

        final moveFuture = controller.selectStack(1);
        expect(controller.isAnimating, isTrue);

        // This tap arrives while the first move is still animating and must
        // be dropped rather than starting a second, conflicting move.
        await controller.selectStack(2);

        await moveFuture;

        expect(controller.state.stacks[1].pieces.length, 2);
        expect(controller.state.stacks[2].pieces.length, 0);
        expect(controller.isAnimating, isFalse);
      },
    );
  });
}
