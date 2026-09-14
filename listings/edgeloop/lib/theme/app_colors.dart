import 'package:flutter/material.dart';

/// EdgeLoop 的配色表。
///
/// 走「纸笔」路线：暖白纸底 + 深墨线。这类题本来就是拿笔在纸上画的，
/// 配色越像纸，线条的存在感越强 —— 而线条是这个游戏唯一的主角。
///
/// 与 LinkFlow 同属浅色系，但主色完全错开（LinkFlow 是青 `#0E9594`，
/// 这里是墨蓝 `#2B3A67`），并排放在商店里不会显得是同一个模板出的。
class AppColors {
  AppColors._();

  /// 背景：偏暖的纸色。
  static const Color background = Color(0xFFF4F1EA);

  /// 背景渐变的另一端（更亮）。
  static const Color backgroundTint = Color(0xFFFBF9F5);

  /// 棋盘底板：纯白，衬托墨线。
  static const Color boardSurface = Color(0xFFFFFFFF);

  /// 底板边框。
  static const Color boardBorder = Color(0xFFE2DDD2);

  /// 格点。没画线时也要能看见，否则玩家不知道能往哪儿连。
  ///
  /// 特意取得比线浅很多：点只是「路标」，抢了线的视觉就乱了。
  static const Color dot = Color(0xFFBFB8AA);

  /// 画上的线 —— 墨蓝。这是整个界面对比度最高的元素，理应如此。
  static const Color line = Color(0xFF2B3A67);

  /// 打叉（确定没有线）。灰而不弱：它承载的推理信息和线一样重，
  /// 但不能和线抢视觉，所以用中性灰而非彩色。
  static const Color cross = Color(0xFF9E9689);

  /// 提示数字。
  static const Color clueText = Color(0xFF2E3440);

  /// 已满足的提示数字 —— 变淡，让玩家一眼看出「这格算完了」。
  static const Color clueSatisfied = Color(0xFFC2BCB0);

  /// 画超了的提示数字。
  static const Color clueViolated = Color(0xFFE5484D);

  static const Color primaryText = Color(0xFF2E3440);
  static const Color mutedText = Color(0xFF8C8578);

  /// 主色：按钮、进度。
  static const Color accent = Color(0xFF2B3A67);

  /// 解出时整条回路的高亮色。
  static const Color solved = Color(0xFF1B873F);
}
