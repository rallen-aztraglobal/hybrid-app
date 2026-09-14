import 'package:flutter/material.dart';

/// DaySpan 的配色表。浅色系。
///
/// 日期工具是「看一眼就走」的东西，配色刻意做得安静：
/// 大面积浅灰白，只有结果数字用主色 —— 用户打开它就是为了看那个数字。
///
/// 主色取暖橙 `#B45309`，与另外两个新包完全错开
/// （EdgeLoop 墨蓝 `#2B3A67`、RepTimer 青绿 `#2DD4BF`）。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFFF6F5F2);
  static const Color backgroundTint = Color(0xFFFCFBF9);

  /// 卡片底：纯白。
  static const Color surface = Color(0xFFFFFFFF);
  static const Color border = Color(0xFFE6E2DA);

  /// 输入区块（日期按钮）的底色，比卡片略深，点起来有「这是可点的」的暗示。
  static const Color field = Color(0xFFF1EEE8);

  static const Color primaryText = Color(0xFF29303A);
  static const Color mutedText = Color(0xFF877F72);

  /// 主色。结果数字、选中态、按钮。
  static const Color accent = Color(0xFFB45309);

  /// 结果区的底色 —— 主色的极淡版，把答案从输入区里托出来。
  static const Color accentWash = Color(0xFFFDF3E7);

  /// 周末标记。
  static const Color weekend = Color(0xFFC2410C);
}
