import 'package:shared_preferences/shared_preferences.dart';

import '../logic/difficulty.dart';

/// 本地存档：每个难度档赢过多少局、最快用时多少秒。
///
/// 只存整数，不存棋盘也不存任何可识别用户的东西 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 读写全部包了 try/catch：首次运行、存档损坏都只当作「没记上」，
/// 绝不因为一次读盘失败把玩家挡在门外。
class ProgressStore {
  static String _winsKey(Difficulty d) => 'minegrid_wins_${d.name}';
  static String _bestKey(Difficulty d) => 'minegrid_best_${d.name}';

  static Future<int> wins(Difficulty difficulty) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_winsKey(difficulty)) ?? 0;
    } catch (_) {
      return 0;
    }
  }

  /// 最快用时（秒）。还没赢过返回 null。
  static Future<int?> bestSeconds(Difficulty difficulty) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_bestKey(difficulty));
    } catch (_) {
      return null;
    }
  }

  /// 记一局胜利，返回 (累计胜场, 最快用时)。
  static Future<(int, int?)> recordWin(Difficulty difficulty, int seconds) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final wins = (prefs.getInt(_winsKey(difficulty)) ?? 0) + 1;
      await prefs.setInt(_winsKey(difficulty), wins);

      final previous = prefs.getInt(_bestKey(difficulty));
      final best = previous == null || seconds < previous ? seconds : previous;
      if (best != previous) await prefs.setInt(_bestKey(difficulty), best);

      return (wins, best);
    } catch (_) {
      return (0, null);
    }
  }
}
