import 'package:shared_preferences/shared_preferences.dart';

import '../logic/generator.dart';

/// 本地进度存档：每种尺寸解开过多少道题。
///
/// 只存两个整数，不存谜题本身也不存任何可识别用户的东西 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
class ProgressStore {
  static const String _prefix = 'nonopix_solved_';

  static String _keyFor(PuzzleSize size) => '$_prefix${size.name}';

  /// 读某个尺寸已解开的题数。读不到（首次运行、存档损坏）一律当 0，不抛。
  static Future<int> solvedCount(PuzzleSize size) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_keyFor(size)) ?? 0;
    } catch (_) {
      return 0;
    }
  }

  /// 解开一题后 +1，返回累计值。写盘失败只当作没记上，不影响玩。
  static Future<int> recordSolved(PuzzleSize size) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final next = (prefs.getInt(_keyFor(size)) ?? 0) + 1;
      await prefs.setInt(_keyFor(size), next);
      return next;
    } catch (_) {
      return 0;
    }
  }
}
