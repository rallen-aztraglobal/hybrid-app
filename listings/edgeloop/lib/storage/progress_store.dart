import 'package:shared_preferences/shared_preferences.dart';

import '../logic/difficulty.dart';

/// EdgeLoop 的本地进度存档。
///
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 每档存两个整数：解出几关、当前停在第几关。四档共八个整数，没有别的。
class ProgressStore {
  ProgressStore._();

  static String _solvedKey(Difficulty d) => 'edgeloop_solved_${d.name}';
  static String _levelKey(Difficulty d) => 'edgeloop_level_${d.name}';

  static Future<int> solvedCount(Difficulty d) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_solvedKey(d)) ?? 0;
    } catch (_) {
      return 0;
    }
  }

  /// 当前停在第几关（下标）。越界会被调用方按关卡数夹住 ——
  /// 关卡库更新后关数可能变少，存档里的旧下标不能直接信。
  static Future<int> currentLevel(Difficulty d) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final v = prefs.getInt(_levelKey(d)) ?? 0;
      return v < 0 ? 0 : v;
    } catch (_) {
      return 0;
    }
  }

  static Future<void> saveLevel(Difficulty d, int index) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setInt(_levelKey(d), index);
    } catch (_) {}
  }

  static Future<void> recordSolved(Difficulty d) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final next = (prefs.getInt(_solvedKey(d)) ?? 0) + 1;
      await prefs.setInt(_solvedKey(d), next);
    } catch (_) {}
  }
}
