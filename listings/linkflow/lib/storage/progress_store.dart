import 'package:shared_preferences/shared_preferences.dart';

import '../logic/puzzle_library.dart';

/// 本地进度存档：每个难度档解开过多少道题、当前打到第几关。
///
/// 只存整数，不存谜题本身也不存任何可识别用户的东西 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 读写全部包了 try/catch：首次运行、存档损坏、低版本迁移都只当作「没记上」，
/// 绝不因为一次读盘失败把玩家挡在门外。
class ProgressStore {
  static String _solvedKey(Difficulty d) => 'linkflow_solved_${d.name}';
  static String _levelKey(Difficulty d) => 'linkflow_level_${d.name}';

  /// 某档已解开的题数。
  static Future<int> solvedCount(Difficulty difficulty) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_solvedKey(difficulty)) ?? 0;
    } catch (_) {
      return 0;
    }
  }

  /// 某档当前停在第几关（0 起）。越界的存档值会被夹回合法范围 ——
  /// 题库改动后关卡数可能变少，不能让旧存档把玩家指到一道不存在的题上。
  static Future<int> levelIndex(Difficulty difficulty) async {
    final total = levelsFor(difficulty).length;
    if (total == 0) return 0;
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getInt(_levelKey(difficulty)) ?? 0;
      return raw.clamp(0, total - 1);
    } catch (_) {
      return 0;
    }
  }

  static Future<void> setLevelIndex(Difficulty difficulty, int index) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setInt(_levelKey(difficulty), index);
    } catch (_) {
      // 记不上就算了，不影响这一局
    }
  }

  /// 解开一题后 +1，返回累计值。
  static Future<int> recordSolved(Difficulty difficulty) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final next = (prefs.getInt(_solvedKey(difficulty)) ?? 0) + 1;
      await prefs.setInt(_solvedKey(difficulty), next);
      return next;
    } catch (_) {
      return 0;
    }
  }
}
