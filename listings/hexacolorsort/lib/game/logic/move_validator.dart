import 'package:flutter/foundation.dart';

import '../models/stack_model.dart';

@immutable
class MoveEvaluation {
  final bool isLegal;
  final int moveCount;
  final String? reason;

  const MoveEvaluation.legal(this.moveCount) : isLegal = true, reason = null;

  const MoveEvaluation.illegal(this.reason) : isLegal = false, moveCount = 0;
}

/// Pure rules for whether pieces may move from one stack to another, and
/// how many pieces such a move would carry.
class MoveValidator {
  const MoveValidator._();

  static MoveEvaluation evaluate({
    required StackModel from,
    required StackModel to,
  }) {
    if (from.id == to.id) {
      return const MoveEvaluation.illegal('same stack');
    }
    if (from.isEmpty) {
      return const MoveEvaluation.illegal('source is empty');
    }
    if (to.remainingCapacity <= 0) {
      return const MoveEvaluation.illegal('destination is full');
    }
    if (!to.isEmpty && to.topColorId != from.topColorId) {
      return const MoveEvaluation.illegal('color mismatch');
    }
    final movable = from.topRunLength;
    final moveCount = movable < to.remainingCapacity
        ? movable
        : to.remainingCapacity;
    if (moveCount <= 0) {
      return const MoveEvaluation.illegal('nothing to move');
    }
    return MoveEvaluation.legal(moveCount);
  }
}
