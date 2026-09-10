import 'package:flutter/material.dart';

/// LinkFlow 的配色表。与 NonoPix 同一套底色规范（米白底 + 白卡片），
/// 区别在主色和通路配色 —— 连线本身就是彩色的，所以卡片要够素才衬得住。
class AppColors {
  AppColors._();

  /// 背景。带一点暖，比纯白柔和，长时间看不刺眼。
  static const Color background = Color(0xFFF2EFE9);

  /// 背景渐变的另一端（更亮）。
  static const Color backgroundTint = Color(0xFFFAF8F4);

  /// 棋盘底板：纯白。
  static const Color boardSurface = Color(0xFFFFFFFF);

  /// 底板边框。浅色主题里靠描边分层，比阴影干净。
  static const Color boardBorder = Color(0xFFE4DFD6);

  /// 空格。比底板略深的暖灰，形成可见但不吵的网格。
  static const Color emptyCell = Color(0xFFEDE9E2);

  static const Color primaryText = Color(0xFF2E3440);
  static const Color mutedText = Color(0xFF8C8578);

  /// 主色。用于按钮与进度。
  ///
  /// 取的是偏暗的青，和 [pathColors] 里那支亮青（`#00BCD4`）明暗差得开。
  /// 两者出现的位置也不同 —— 一个在界面控件上，一个在棋盘里 —— 不会看混。
  static const Color accent = Color(0xFF0E9594);

  /// 通路配色。按通路在题目里的下标取。
  ///
  /// 12 支，都能在白底上看清。
  ///
  /// **排序不是按色相排的，是按「前 N 支要尽量拉开」排的。**
  /// 一道题用几条线就取前几支，所以低难度的 4~5 色盘面拿到的是
  /// 红/蓝/绿/橙/紫这种一眼分得开的组合。
  /// 早先按色相顺着排，5 色题就摊上了红橙黄草绿深绿一串邻近色 —— 满盘看着糊成一片。
  ///
  /// 为什么要凑到 12 支：难度档靠「盘子更大 + 通路更多」往上堆，
  /// 最高档一盘要同时放 12 条线。超出会循环取色，同一题里就会有两条同色的线，
  /// 所以题库生成器的通路数上限必须钉死在这个长度以内。
  static const List<Color> pathColors = <Color>[
    Color(0xFFE53935), // 红
    Color(0xFF1E88E5), // 蓝
    Color(0xFF2E7D32), // 深绿
    Color(0xFFFB8C00), // 橙
    Color(0xFF8E24AA), // 紫
    Color(0xFF00BCD4), // 亮青
    Color(0xFFEC407A), // 粉
    Color(0xFFFBC02D), // 黄
    Color(0xFF795548), // 棕
    Color(0xFF303F9F), // 靛
    Color(0xFF7CB342), // 草绿
    Color(0xFF607D8B), // 石板灰
  ];

  static Color path(int index) => pathColors[index % pathColors.length];

  /// 涂错时的提示色。
  static const Color mistake = Color(0xFFE5484D);
}
