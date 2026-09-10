import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../logic/counter.dart';

/// 计数器的本地存档。
///
/// 只写一个 JSON 字符串到 shared_preferences，不出设备、不含任何可识别用户的内容 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// **这是用户唯一的数据**，所以读这一侧写得很保守：键不存在、不是合法 JSON、
/// 顶层不是数组、某一条不是对象，全都各自兜住，能救几条救几条，
/// 实在读不出来就给一份默认计数器 —— 但**绝不抛异常**。
/// 一个空白页面加一句「加载失败」，对着一份数了半天的计数来说是最糟的结果。
class CounterStore {
  /// 键名带版本号。将来存档结构真要改，可以并存新旧两个键平滑迁移，
  /// 而不是就地覆盖、让装了旧版的人一升级就丢数据。
  static const String _key = 'tickpad_counters_v1';

  /// 第一次打开时给的那一个计数器。空白一片会让人不知道从哪下手。
  static List<Counter> defaults() => <Counter>[
        Counter(id: 'default-1', label: 'Count', step: 1),
      ];

  static Future<List<Counter>> load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_key);
      if (raw == null || raw.isEmpty) return defaults();

      final decoded = jsonDecode(raw);
      if (decoded is! List) return defaults();

      final counters = <Counter>[];
      for (final item in decoded) {
        if (item is Map<String, dynamic>) {
          counters.add(Counter.fromJson(item));
        }
        // 不是对象的条目直接跳过，不让它带崩整份存档
      }
      return counters.isEmpty ? defaults() : counters;
    } catch (_) {
      return defaults();
    }
  }

  static Future<void> save(List<Counter> counters) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(
        _key,
        jsonEncode(<Map<String, dynamic>>[
          for (final c in counters) c.toJson(),
        ]),
      );
    } catch (_) {
      // 写盘失败只当作这一下没记上，不打断使用
    }
  }
}
