import 'package:shared_preferences/shared_preferences.dart';

/// DaySpan 的本地存档。
///
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 本包只存**两个整数**：上次用的是哪个功能页、天数差是否含首尾。
/// 用户输入的日期本身不存 —— 每次打开都回到今天，这既是隐私上最干净的做法，
/// 也符合实际用法（算完就完了，没人需要下次还看到上次算的那两个日期）。
class PrefsStore {
  PrefsStore._();

  static const String _tabKey = 'dayspan_tab';
  static const String _inclusiveKey = 'dayspan_inclusive';

  /// 上次停在哪个功能页（0=天数差 / 1=加减天数 / 2=工作日）。
  static Future<int> loadTab() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final v = prefs.getInt(_tabKey) ?? 0;
      // 夹在合法范围内：存档可能是旧版本写的，那时的页数不一样。
      return v.clamp(0, 2);
    } catch (_) {
      return 0;
    }
  }

  static Future<void> saveTab(int index) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setInt(_tabKey, index);
    } catch (_) {
      // 存不进去不影响用 —— 下次打开回到第一页而已，不值得打断用户。
    }
  }

  /// 天数差是否把首尾两天都算进去。
  static Future<bool> loadInclusive() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getBool(_inclusiveKey) ?? false;
    } catch (_) {
      return false;
    }
  }

  static Future<void> saveInclusive(bool value) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(_inclusiveKey, value);
    } catch (_) {}
  }
}
