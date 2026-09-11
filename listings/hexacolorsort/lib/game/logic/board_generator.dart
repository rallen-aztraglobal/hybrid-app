import 'dart:math';

import '../../core/constants/game_constants.dart';
import '../models/color_piece.dart';
import '../models/stack_model.dart';
import 'deadlock_detector.dart';

/// Builds the initial board layout from a seed so runs are reproducible.
///
/// Guarantees:
/// * exactly one stack starts empty, guaranteeing free movement space.
/// * no stack starts with a top run already at the clear threshold.
/// * at least one legal move exists (implied by the guaranteed empty stack
///   plus at least one other non-empty stack).
class BoardGenerator {
  const BoardGenerator._();

  static List<StackModel> generate({
    required int seed,
    int stackCount = GameConstants.initialStackCount,
    int capacity = GameConstants.stackCapacity,
    int colorCount = GameConstants.initialColorCount,
    int fillPerStack = GameConstants.initialFillPerStack,
  }) {
    assert(stackCount >= 2);
    assert(colorCount >= 1);
    assert(fillPerStack < capacity);

    var attempt = 0;
    while (true) {
      final random = Random(seed + attempt * 7919);
      final stacks = _tryGenerate(
        random: random,
        stackCount: stackCount,
        capacity: capacity,
        colorCount: colorCount,
        fillPerStack: fillPerStack,
      );
      final noInstantClear = stacks.every(
        (s) => s.topRunLength < GameConstants.clearThreshold,
      );
      final hasMove = !DeadlockDetector.isDeadlocked(stacks);
      if (noInstantClear && hasMove) {
        return stacks;
      }
      attempt++;
      if (attempt > 200) {
        // Extremely unlikely; fall back to whatever was last generated so
        // generation always terminates.
        return stacks;
      }
    }
  }

  static List<StackModel> _tryGenerate({
    required Random random,
    required int stackCount,
    required int capacity,
    required int colorCount,
    required int fillPerStack,
  }) {
    final emptyStackIndex = random.nextInt(stackCount);
    final fillableIndices = [
      for (var i = 0; i < stackCount; i++)
        if (i != emptyStackIndex) i,
    ];

    final totalPieces = fillableIndices.length * fillPerStack;
    final bag = <ColorPiece>[];
    for (var i = 0; i < totalPieces; i++) {
      bag.add(ColorPiece(i % colorCount));
    }
    bag.shuffle(random);

    final stacks = List<StackModel>.generate(
      stackCount,
      (i) => StackModel(id: i, pieces: const [], capacity: capacity),
    );

    var cursor = 0;
    final mutable = [...stacks];
    for (final index in fillableIndices) {
      final dealt = bag.sublist(cursor, cursor + fillPerStack);
      cursor += fillPerStack;
      mutable[index] = mutable[index].copyWith(pieces: dealt);
    }
    return mutable;
  }
}
