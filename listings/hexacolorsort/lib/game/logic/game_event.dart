import 'package:flutter/foundation.dart';

/// One-off transient occurrences the UI should animate/react to. These are
/// separate from [GameState] because they represent moments in time, not
/// persistent data.
@immutable
sealed class GameEvent {
  const GameEvent();
}

class MoveEvent extends GameEvent {
  final int fromIndex;
  final int toIndex;
  final int count;
  final int colorId;

  const MoveEvent({
    required this.fromIndex,
    required this.toIndex,
    required this.count,
    required this.colorId,
  });
}

class ClearEvent extends GameEvent {
  final int stackIndex;
  final int colorId;
  final int comboStreak;
  final int scoreAwarded;

  const ClearEvent({
    required this.stackIndex,
    required this.colorId,
    required this.comboStreak,
    required this.scoreAwarded,
  });
}

class IllegalMoveEvent extends GameEvent {
  final int stackIndex;

  const IllegalMoveEvent(this.stackIndex);
}

class StageUpEvent extends GameEvent {
  final int newStage;

  const StageUpEvent(this.newStage);
}

class GameOverEvent extends GameEvent {
  final bool isNewBest;

  const GameOverEvent({required this.isNewBest});
}
