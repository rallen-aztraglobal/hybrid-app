/// Centralized user-facing text. Keeping all strings here makes future
/// localization straightforward (e.g. swapping this file for an
/// intl-generated lookup) without touching widget code.
class AppStrings {
  AppStrings._();

  static const String appName = 'Hexa Color Sort';
  static const String tagline = 'Sort. Stack. Clear.';

  // Home screen
  static const String play = 'Play';
  static const String bestScore = 'Best Score';
  static const String howToPlay = 'How to Play';
  static const String sound = 'Sound';
  static const String vibration = 'Vibration';

  // Game screen
  static const String score = 'Score';
  static const String best = 'Best';
  static const String stage = 'Stage';
  static const String combo = 'Combo';
  static const String restart = 'Restart';
  static const String undo = 'Undo';

  // Pause dialog
  static const String paused = 'Paused';
  static const String resume = 'Resume';

  // Result screen
  static const String gameOver = 'Game Over';
  static const String finalScore = 'Final Score';
  static const String maxCombo = 'Max Combo';
  static const String stageReached = 'Stage Reached';
  static const String newBest = 'New Best!';
  static const String playAgain = 'Play Again';
  static const String home = 'Home';

  // How to play
  static const String howToPlayStep1 =
      'Tap a stack to select matching top colors.';
  static const String howToPlayStep2 =
      'Move them onto an empty stack or the same color.';
  static const String howToPlayStep3 =
      'Match five colors to clear them and build combos.';
  static const String gotIt = 'Got It';
}
