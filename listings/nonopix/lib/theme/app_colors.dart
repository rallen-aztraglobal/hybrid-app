import 'package:flutter/material.dart';

/// NonoPix 的配色表。集中在这里，图标与商店素材也按同一组色生成。
///
/// **为什么是浅色暖调。** 最初做的是深色青，整屏偏冷偏暗，看久了累，
/// 也不像休闲小游戏该有的样子。改成米白底 + 靛蓝填色之后：
/// 对比度高得多（深色底上再怎么调，空格与底板的差都有限），
/// 视觉负担轻，而且 B 面是浅色站，A→B 切换时不会突然一暗一亮。
class AppColors {
  AppColors._();

  /// 背景。带一点暖，比纯白柔和，长时间看不刺眼。
  static const Color background = Color(0xFFF2EFE9);

  /// 背景渐变的另一端（更亮），做一道极淡的竖向渐变。
  static const Color backgroundTint = Color(0xFFFAF8F4);

  /// 棋盘底板：纯白，从暖底色里干净地浮出来。
  static const Color boardSurface = Color(0xFFFFFFFF);

  /// 底板边框。浅色主题里靠描边而不是阴影分层，更清爽。
  static const Color boardBorder = Color(0xFFE4DFD6);

  /// 未落笔的格子：比底板略深的暖灰，形成可见但不吵的网格。
  static const Color emptyCell = Color(0xFFEBE7E0);

  static const Color primaryText = Color(0xFF2E3440);
  static const Color mutedText = Color(0xFF8C8578);

  /// 主色。靛蓝在米白上对比强、又不像红橙那样躁。
  static const Color accent = Color(0xFF4F46E5);

  /// 已涂格子。整屏最重的色块，落笔的反馈全靠它。
  static const Color filledCell = Color(0xFF4F46E5);

  /// 已涂格子的顶部高光，做一道很轻的渐变，让方块有体积感。
  static const Color filledCellTop = Color(0xFF6C63FF);

  /// 打叉（玩家标记「这里是空的」）的记号颜色。
  static const Color crossMark = Color(0xFFB0A99D);

  /// 每 5 格一条的粗分隔线。数织靠它数格子，必须比格缝明显。
  static const Color majorGrid = Color(0xFFC7C0B4);

  /// 已满足的线索淡成这个色 —— 数织里最实用的视觉辅助，
  /// 让玩家一眼看出哪几行列已经做完。
  static const Color clueDone = Color(0xFFC5BFB4);
  /// 线索底格。让同一行的数字自然成组，不至于一屏数字糊成一片。
  static const Color clueSlot = Color(0xFFF4F1EB);

  /// 已满足的线索底格 —— 连底带字一起淡掉。
  static const Color clueSlotDone = Color(0xFFFAF8F5);


  /// 涂错时的提示色。
  static const Color mistake = Color(0xFFE5484D);
}
