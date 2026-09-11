import 'package:flutter/material.dart';

/// Tunable gameplay constants. Kept in one place so difficulty and pacing
/// can be adjusted without hunting through game logic files.
class GameConstants {
  GameConstants._();

  static const int initialStackCount = 6;
  static const int stackCapacity = 8;
  static const int initialColorCount = 4;
  static const int maxColorCount = 6;
  static const int initialFillPerStack = 5;
  static const int clearThreshold = 5;

  /// Total pieces cleared before the difficulty stage advances.
  static const int piecesPerStage = 15;

  /// A clear that happens within this window of the previous clear extends
  /// the active combo streak instead of resetting it.
  static const Duration comboWindow = Duration(milliseconds: 2500);

  // Animation durations (kept within the 150-400ms range requested).
  static const Duration selectAnimationDuration = Duration(milliseconds: 150);
  static const Duration moveAnimationDuration = Duration(milliseconds: 320);
  static const Duration clearAnimationDuration = Duration(milliseconds: 350);
  static const Duration shakeAnimationDuration = Duration(milliseconds: 300);
  static const Duration comboPopupDuration = Duration(milliseconds: 400);
  static const Duration splashDuration = Duration(milliseconds: 1000);

  static const int baseClearScore = 100;
}

/// One visual identity per color: a distinct hue plus a distinct icon so
/// color is never the only channel carrying information.
class ColorPieceStyle {
  final Color color;
  final Color highlight;
  final IconData icon;

  const ColorPieceStyle({
    required this.color,
    required this.highlight,
    required this.icon,
  });
}

const List<ColorPieceStyle> kColorPalette = [
  ColorPieceStyle(
    color: Color(0xFFE85D5D),
    highlight: Color(0xFFFF8A8A),
    icon: Icons.circle,
  ),
  ColorPieceStyle(
    color: Color(0xFF4FC3F7),
    highlight: Color(0xFF8EE1FF),
    icon: Icons.change_history,
  ),
  ColorPieceStyle(
    color: Color(0xFFFFC94F),
    highlight: Color(0xFFFFE28A),
    icon: Icons.square,
  ),
  ColorPieceStyle(
    color: Color(0xFF66D98E),
    highlight: Color(0xFFA0F0BC),
    icon: Icons.star,
  ),
  ColorPieceStyle(
    color: Color(0xFFBA8CE8),
    highlight: Color(0xFFDCC2FF),
    icon: Icons.diamond,
  ),
  ColorPieceStyle(
    color: Color(0xFFFF9E5E),
    highlight: Color(0xFFFFC194),
    icon: Icons.favorite,
  ),
];
