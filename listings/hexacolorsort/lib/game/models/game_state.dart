import 'package:flutter/foundation.dart';

import 'move_record.dart';
import 'stack_model.dart';

/// The complete, immutable snapshot of a game in progress.
@immutable
class GameState {
  final List<StackModel> stacks;
  final int score;
  final int bestScore;
  final int stage;
  final int colorCount;
  final int comboCount;
  final int maxCombo;
  final int piecesClearedTotal;
  final int? selectedStackIndex;
  final bool isGameOver;
  final MoveRecord? lastMove;

  const GameState({
    required this.stacks,
    required this.score,
    required this.bestScore,
    required this.stage,
    required this.colorCount,
    this.comboCount = 0,
    this.maxCombo = 0,
    this.piecesClearedTotal = 0,
    this.selectedStackIndex,
    this.isGameOver = false,
    this.lastMove,
  });

  GameState copyWith({
    List<StackModel>? stacks,
    int? score,
    int? bestScore,
    int? stage,
    int? colorCount,
    int? comboCount,
    int? maxCombo,
    int? piecesClearedTotal,
    int? selectedStackIndex,
    bool clearSelection = false,
    bool? isGameOver,
    MoveRecord? lastMove,
    bool clearLastMove = false,
  }) {
    return GameState(
      stacks: stacks ?? this.stacks,
      score: score ?? this.score,
      bestScore: bestScore ?? this.bestScore,
      stage: stage ?? this.stage,
      colorCount: colorCount ?? this.colorCount,
      comboCount: comboCount ?? this.comboCount,
      maxCombo: maxCombo ?? this.maxCombo,
      piecesClearedTotal: piecesClearedTotal ?? this.piecesClearedTotal,
      selectedStackIndex: clearSelection
          ? null
          : (selectedStackIndex ?? this.selectedStackIndex),
      isGameOver: isGameOver ?? this.isGameOver,
      lastMove: clearLastMove ? null : (lastMove ?? this.lastMove),
    );
  }

  /// A snapshot suitable for storing inside a [MoveRecord]: it never holds
  /// on to a previous move record, keeping undo history exactly one deep.
  GameState get asUndoSnapshot => copyWith(clearLastMove: true);
}
