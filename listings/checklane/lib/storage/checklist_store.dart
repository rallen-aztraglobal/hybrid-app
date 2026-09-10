import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../logic/checklist.dart';

/// 清单的本地存档。
///
/// 只写一个 JSON 字符串到 shared_preferences，不出设备、不含任何可识别用户的内容 ——
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 读这一侧全程兜底：键不存在、不是合法 JSON、顶层不是数组、某一条不是对象，
/// 各自兜住，能救几条救几条，**绝不抛异常**。
class ChecklistStore {
  /// 键名带版本号。将来存档结构真要改，可以并存新旧两个键平滑迁移，
  /// 而不是就地覆盖、让装了旧版的人一升级就丢数据。
  static const String _key = 'checklane_lists_v1';

  /// 第一次打开时给的示例清单。
  ///
  /// 空白一片会让人不知道从哪下手，而且看不出这个 App 和普通待办的区别 ——
  /// 给一份**明显是要反复用的**清单（出行前检查），一眼就懂「勾完可以重来」。
  static List<Checklist> defaults() => <Checklist>[
        Checklist(
          id: 'sample-trip',
          title: 'Before I leave',
          items: <ChecklistItem>[
            ChecklistItem(id: 'sample-trip-0', text: 'Keys'),
            ChecklistItem(id: 'sample-trip-1', text: 'Wallet'),
            ChecklistItem(id: 'sample-trip-2', text: 'Phone charger'),
            ChecklistItem(id: 'sample-trip-3', text: 'Lock the windows'),
          ],
        ),
      ];

  static Future<List<Checklist>> load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_key);
      if (raw == null || raw.isEmpty) return defaults();

      final decoded = jsonDecode(raw);
      if (decoded is! List) return defaults();

      final lists = <Checklist>[];
      for (final item in decoded) {
        if (item is Map<String, dynamic>) lists.add(Checklist.fromJson(item));
      }
      // 全空说明存档确实坏了；但**空数组是合法的** —— 用户可能把清单都删了，
      // 那时不该硬塞回示例清单，否则删了又冒出来，像 App 不听话。
      return lists;
    } catch (_) {
      return defaults();
    }
  }

  static Future<void> save(List<Checklist> lists) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(
        _key,
        jsonEncode(<Map<String, dynamic>>[for (final l in lists) l.toJson()]),
      );
    } catch (_) {
      // 写盘失败只当作这一下没记上，不打断使用
    }
  }
}
