import 'package:flutter/material.dart';

/// RepTimer 的配色表。深色系。
///
/// 深色不是为了好看，是为了这个 App 的实际使用场景：
/// 手机架在地上或器械上，练的时候瞟一眼。深底亮字在远距离和侧视角下都更清楚，
/// 而且健身房灯光刺眼时不会再糊一层白屏上去。
///
/// 各阶段用三种明确区分的颜色 —— **练的时候只有余光**，靠读字来不及，
/// 必须靠颜色一眼分辨现在是做还是休。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFF11151A);
  static const Color surface = Color(0xFF1A2027);
  static const Color surfaceHigh = Color(0xFF232B34);
  static const Color border = Color(0xFF2E3741);

  static const Color primaryText = Color(0xFFEDF1F5);
  static const Color mutedText = Color(0xFF8A97A5);

  /// 主色。
  static const Color accent = Color(0xFF2DD4BF);

  // —— 各阶段的颜色。彼此的色相拉到最开，余光扫一眼就能分辨。——

  /// 准备：琥珀。介于「还没开始」和「马上开始」之间的警示感。
  static const Color prepare = Color(0xFFF59E0B);

  /// 做：青绿。最亮、饱和度最高 —— 这是唯一需要用力的阶段。
  static const Color work = Color(0xFF2DD4BF);

  /// 休息：蓝。冷色，和「做」的暖青绿在色相上分得开。
  static const Color rest = Color(0xFF60A5FA);

  /// 大循环休息：紫。比普通休息更「远」，暗示这是段长的。
  static const Color cycleRest = Color(0xFFA78BFA);

  /// 结束。
  static const Color done = Color(0xFF4ADE80);
}
