import 'dart:math';

import 'package:flutter/foundation.dart';

class GameStats {
  const GameStats._();

  // TODO: Persist bestScore with local storage before release so the
  // player's best result survives app restarts.
  static int bestScore = 0;

  static int updateBestScore(int score) {
    bestScore = max(bestScore, score);
    return bestScore;
  }

  @visibleForTesting
  static void reset() {
    bestScore = 0;
  }
}
