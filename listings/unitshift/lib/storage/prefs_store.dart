import 'package:shared_preferences/shared_preferences.dart';

/// 本地偏好：上次用的分类，以及每个分类各自上次选的源单位。
///
/// 只存几个字符串键，不出设备，不含任何可识别用户的东西 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 按分类分别记源单位，而不是全局记一个：换到「温度」还停在「公里」上是没意义的。
///
/// 读写全部包了 try/catch：首次运行、存档损坏都只当作「没记上」，
/// 绝不因为一次读盘失败把用户挡在门外。
class PrefsStore {
  static const String _categoryKey = 'unitshift_category';
  static String _unitKey(String categoryId) => 'unitshift_unit_$categoryId';

  static Future<String?> lastCategory() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getString(_categoryKey);
    } catch (_) {
      return null;
    }
  }

  static Future<void> setLastCategory(String categoryId) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_categoryKey, categoryId);
    } catch (_) {
      // 记不上就算了，不影响这一次使用
    }
  }

  static Future<String?> lastUnit(String categoryId) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getString(_unitKey(categoryId));
    } catch (_) {
      return null;
    }
  }

  static Future<void> setLastUnit(String categoryId, String unitId) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_unitKey(categoryId), unitId);
    } catch (_) {
      // 同上
    }
  }
}
