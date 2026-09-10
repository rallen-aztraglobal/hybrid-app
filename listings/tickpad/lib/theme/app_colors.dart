import 'package:flutter/material.dart';

/// TickPad 的配色表 —— 深色主题。
///
/// 计数器这类工具经常在光线差的场合用（仓库盘点、夜里数场次、看台上计人数），
/// 深色底比亮底不刺眼，也更省电。数字是整屏唯一的重点，所以除了数字和主色，
/// 其余一律压得很低。
///
/// 底色带一点绿灰，和另外几个包各有各的偏色（MineGrid 偏蓝、NonoPix 偏紫）。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFF101418);
  static const Color backgroundTint = Color(0xFF161B20);

  /// 计数卡片。比背景亮一档。
  static const Color surface = Color(0xFF1A2026);
  static const Color surfacePressed = Color(0xFF232B33);
  static const Color border = Color(0xFF262E36);

  static const Color primaryText = Color(0xFFE9EEF2);
  static const Color mutedText = Color(0xFF7F8B96);

  /// 主色：绿。计数的语义是「又记上一个」，绿色最直白，
  /// 在深底上也够亮，闭着眼按都知道按没按上。
  static const Color accent = Color(0xFF3DDC84);

  /// 主色的暗色衬底，用于步长标签这类次要元素。
  static const Color accentSoft = Color(0xFF153023);

  /// 减一按钮。刻意做得比加一低调 —— 减是纠错动作，不该和主操作抢注意力。
  static const Color minus = Color(0xFF2A333B);

  /// 危险操作（清零 / 删除）。
  static const Color danger = Color(0xFFFF5A5F);

  /// 计数器的颜色标签。
  ///
  /// 快速点数时人是靠颜色扫卡片的，不是靠读名字 —— 一眼的颜色差比一行小字管用。
  /// 第 0 支就是主色，所以不设标签的卡片看起来仍然是「默认的样子」。
  static const List<Color> tags = <Color>[
    accent, // 绿（默认）
    Color(0xFF4DA3FF), // 蓝
    Color(0xFFFFB03C), // 橙
    Color(0xFFFF6B8A), // 粉
    Color(0xFFB78CFF), // 紫
    Color(0xFF35D2E0), // 青
  ];

  static Color tag(int index) => tags[index % tags.length];

  /// 标签色对应的深色文字（用在标签色块上）。
  static const Color onTag = Color(0xFF0E1512);
}
