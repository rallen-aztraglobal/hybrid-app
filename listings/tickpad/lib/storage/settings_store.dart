import 'package:shared_preferences/shared_preferences.dart';

/// 开关类偏好。读写失败一律当作「用默认值」，不抛。
class SettingsStore {
  static const String _volumeKeysKey = 'tickpad_volume_keys';

  /// 是否用音量键计数。
  ///
  /// **默认关。** 接管音量键是件挺霸道的事 —— 用户可能只是想调一下音量，
  /// 结果计数被加了一个。所以默认不开，在菜单里给出开关，
  /// 底部提示里也提一句让它能被发现。
  static Future<bool> volumeKeys() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getBool(_volumeKeysKey) ?? false;
    } catch (_) {
      return false;
    }
  }

  static Future<void> setVolumeKeys(bool enabled) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(_volumeKeysKey, enabled);
    } catch (_) {
      // 记不上就算了
    }
  }
}
