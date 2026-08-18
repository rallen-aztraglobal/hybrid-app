import '../../core/constants/game_constants.dart';

/// Pure scoring rules, kept free of timing/state concerns so they can be
/// unit tested as simple functions of the current combo streak.
class ScoringService {
  const ScoringService._();

  /// Score awarded for a clear that lands as the [comboStreak]-th
  /// consecutive combo hit (1-indexed): 1 -> 100, 2 -> 200, 3 -> 300, ...
  static int scoreForCombo(int comboStreak) {
    assert(comboStreak >= 1);
    return GameConstants.baseClearScore * comboStreak;
  }
}
