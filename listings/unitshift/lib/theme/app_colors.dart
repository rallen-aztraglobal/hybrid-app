import 'package:flutter/material.dart';

/// UnitShift 的配色表 —— 浅色。
///
/// 工具类和游戏的取向不一样：这是一个「看一眼数字就走」的 App，
/// 亮底 + 高对比的数字最省事。底色比游戏那几个更冷更干净（偏灰白而非奶油色），
/// 少一点性格、多一点像仪表。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFFF4F5F7);
  static const Color backgroundTint = Color(0xFFFBFBFC);

  /// 卡片/列表底板。
  static const Color surface = Color(0xFFFFFFFF);
  static const Color border = Color(0xFFE3E5EA);

  /// 分隔线。比边框更淡，用于列表行之间。
  static const Color divider = Color(0xFFEDEFF2);

  static const Color primaryText = Color(0xFF1B1F27);
  static const Color mutedText = Color(0xFF767D8C);

  /// 主色：明确的蓝。工具语义，也和三个游戏（靛 / 青 / 琥珀）都错开。
  static const Color accent = Color(0xFF2F6FED);

  /// 主色的浅底，用于选中的分类胶囊和当前源单位那一行。
  static const Color accentSoft = Color(0xFFE8F0FE);

  /// 键盘按键。
  static const Color key = Color(0xFFFFFFFF);
  static const Color keyPressed = Color(0xFFE9ECF1);

  /// 功能键（退格 / 清空）。
  static const Color keyMuted = Color(0xFFEAEDF2);
}
