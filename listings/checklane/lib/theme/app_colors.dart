import 'package:flutter/material.dart';

/// CheckLane 的配色表 —— 深色主题。
///
/// 核对表是「照着一条条读、一条条勾」的东西，所以整屏只有两种状态要分清：
/// **勾过的**和**没勾的**。配色的全部力气都花在这一对对比上 ——
/// 没勾的正文用最亮的字，勾过的压暗并划掉，扫一眼就知道还剩哪几条。
///
/// 底色带一点紫灰。另外两个深色包分别偏蓝（MineGrid）和偏绿（TickPad），
/// 三个摆在一起像同一家出的，又不会认错。
class AppColors {
  AppColors._();

  static const Color background = Color(0xFF131218);
  static const Color backgroundTint = Color(0xFF1A1822);

  /// 卡片 / 列表行底板。
  static const Color surface = Color(0xFF1D1B26);
  static const Color surfacePressed = Color(0xFF262332);
  static const Color border = Color(0xFF2B2838);

  /// 没勾的正文。整屏最亮的字 —— 那才是还要做的事。
  static const Color primaryText = Color(0xFFEAE8F2);

  /// 次要信息（计数、时间、说明）。
  static const Color mutedText = Color(0xFF8A85A0);

  /// 勾过的正文。压暗到比次要信息还低一档，配上删除线，
  /// 让「已完成」在余光里就退场。
  static const Color doneText = Color(0xFF5C5872);

  /// 主色：紫。清单/计划的语义偏冷静，紫比暖色更贴，
  /// 在深底上也够亮，勾选框一眼能找到。
  static const Color accent = Color(0xFF9B7CFF);

  /// 主色的暗色衬底。
  static const Color accentSoft = Color(0xFF241F3A);

  /// 主色上的文字/图标色。
  static const Color onAccent = Color(0xFF12101A);

  /// 全部勾完。
  static const Color complete = Color(0xFF4ADE80);

  /// 危险操作（清空 / 删除）。
  static const Color danger = Color(0xFFFF5A6E);
}
