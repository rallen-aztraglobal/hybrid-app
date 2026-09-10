import 'package:flutter/material.dart';

/// MineGrid 的配色表 —— 深色主题。
///
/// 三个游戏里只有这一个走深色：扫雷是长时间盯着一片格子做推理的游戏，
/// 暗底浅字看久了眼睛轻松，数字之间的色相差也拉得更开。
/// （NonoPix / LinkFlow 是浅色，那两个是短平快的图形题，亮底更精神。）
///
/// 底色刻意选带蓝的深灰而不是纯黑：纯黑上面所有分层都得靠边框硬画，
/// 层次一压扁就显得廉价。留一点明度，盖着的格子、挖开的格子、底板
/// 三层才能只靠明暗就分得清。
class AppColors {
  AppColors._();

  /// 页面背景。
  static const Color background = Color(0xFF101319);

  /// 背景渐变的顶端（略亮）。
  static const Color backgroundTint = Color(0xFF171B24);

  /// 棋盘底板。比背景亮一档，托住整块盘面。
  static const Color boardSurface = Color(0xFF1A1E27);
  static const Color boardBorder = Color(0xFF272C38);

  /// 没翻开的格子。整个界面最亮的一层 —— 它是唯一可以点的东西，
  /// 视觉上就该最跳。
  static const Color coveredCell = Color(0xFF2C3240);

  /// 没翻开格子的顶部高光。一条更亮的窄边，让格子看着是「盖着的」，
  /// 挖开后这层消失，对比很直白。
  static const Color coveredCellTop = Color(0xFF373E4F);

  /// 翻开后的格子。比底板还暗 —— 挖开了就是个坑，暗下去最符合直觉。
  static const Color revealedCell = Color(0xFF141821);

  static const Color primaryText = Color(0xFFE9ECF2);
  static const Color mutedText = Color(0xFF868EA1);

  /// 主色：琥珀。扫雷的语义是「小心」，暖色比冷色贴；
  /// 而且在深底上比任何冷色都跳，插旗和按钮一眼能找到。
  static const Color accent = Color(0xFFFF9A3C);

  /// 踩到的那一格。
  static const Color explosion = Color(0xFFFF5257);

  /// 雷的图形色。
  static const Color mine = Color(0xFFEDEFF4);

  /// 通关色。
  static const Color success = Color(0xFF4ADE80);

  /// 数字 1~8 的颜色。
  ///
  /// 沿用桌面扫雷「1 蓝 2 绿 3 红」的老约定 —— 玩过的人肌肉记忆直接能用，
  /// 换一套自创配色只会让老玩家读错。但每一支都按深底重新提了明度：
  /// 原版那套是给白底调的，直接搬到深色上 3 号的暗红会糊成一团。
  static const List<Color> numberColors = <Color>[
    Color(0xFF5AA9FF), // 1 蓝
    Color(0xFF4ADE80), // 2 绿
    Color(0xFFFF6B6B), // 3 红
    Color(0xFFB794F6), // 4 紫
    Color(0xFFF6AD55), // 5 琥珀
    Color(0xFF4FD1C5), // 6 青
    Color(0xFFE2E8F0), // 7 近白
    Color(0xFFA0AEC0), // 8 灰
  ];

  static Color number(int value) =>
      numberColors[(value - 1).clamp(0, numberColors.length - 1)];
}
