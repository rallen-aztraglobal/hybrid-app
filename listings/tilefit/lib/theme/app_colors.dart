import 'package:flutter/material.dart';

/// TileFit 的深色配色表。集中在这里，图标与商店素材也按同一组色生成。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFF0E1116);

  /// 棋盘底板，比背景略亮一档，让棋盘从背景里"浮"出来。
  static const Color boardSurface = Color(0xFF161B22);

  /// 空格。比底板再亮一点点，形成可见但不抢眼的网格。
  static const Color emptyCell = Color(0xFF1E242E);

  static const Color primaryText = Color(0xFFE6EBF2);
  static const Color mutedText = Color(0xFF7C8797);
  static const Color accent = Color(0xFF4ECDC4);

  /// 拖动时的落点预览：能放下画方块本色的半透明，放不下画红。
  static const Color ghostInvalid = Color(0x55FF6B6B);

  /// 待放区里放不下的方块整体压暗成这个色，一眼看出"这个没地方放了"。
  static const Color deadPiece = Color(0xFF39414E);

  /// 方块配色，下标即 [Piece.colorIndex]。
  ///
  /// 同一色系内按形状复杂度递进（越大越暖），避免相邻格撞色难分辨。
  static const List<Color> pieceColors = <Color>[
    Color(0xFF4ECDC4), // 0 单格
    Color(0xFF5AA9E6), // 1 二连
    Color(0xFF7C7CE0), // 2 三连
    Color(0xFFB97FE0), // 3 四连
    Color(0xFFE87FA8), // 4 五连
    Color(0xFFF0885A), // 5 2×2
    Color(0xFFF0C15A), // 6 3×3
    Color(0xFF8FCF6E), // 7 三格直角
    Color(0xFF5FD0A8), // 8 五格直角
  ];

  /// 取配色，越界时回落到强调色而不是崩 —— 新增形状忘了配色不该让 App 挂掉。
  static Color piece(int index) =>
      (index >= 0 && index < pieceColors.length) ? pieceColors[index] : accent;
}
