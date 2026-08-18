import '../models/stack_model.dart';
import 'move_validator.dart';

/// Determines whether any legal move exists across the whole board. Must
/// check every ordered pair of stacks rather than assuming an empty stack
/// alone implies a move is possible (there might be none left to move).
class DeadlockDetector {
  const DeadlockDetector._();

  static bool hasAnyLegalMove(List<StackModel> stacks) {
    for (var i = 0; i < stacks.length; i++) {
      if (stacks[i].isEmpty) continue;
      for (var j = 0; j < stacks.length; j++) {
        if (i == j) continue;
        final evaluation = MoveValidator.evaluate(
          from: stacks[i],
          to: stacks[j],
        );
        if (evaluation.isLegal) return true;
      }
    }
    return false;
  }

  static bool isDeadlocked(List<StackModel> stacks) => !hasAnyLegalMove(stacks);
}
