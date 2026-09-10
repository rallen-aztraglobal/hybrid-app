import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../logic/session.dart';

/// 会话历史的本地存档。
///
/// 和计数存档同一套口径：只写一个 JSON 字符串，不出设备，读的一侧全程兜底、绝不抛异常。
class SessionStore {
  static const String _key = 'tickpad_sessions_v1';

  /// 最多留这么多条。历史是给人翻的，不是数据库 ——
  /// 无上限地攒下去，既让列表没法看，也会把 shared_preferences 撑大。
  static const int maxSessions = 50;

  static Future<List<Session>> load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final raw = prefs.getString(_key);
      if (raw == null || raw.isEmpty) return <Session>[];

      final decoded = jsonDecode(raw);
      if (decoded is! List) return <Session>[];

      final sessions = <Session>[];
      for (final item in decoded) {
        if (item is Map<String, dynamic>) sessions.add(Session.fromJson(item));
      }
      // 新的排前面
      sessions.sort((a, b) => b.savedAt.compareTo(a.savedAt));
      return sessions;
    } catch (_) {
      return <Session>[];
    }
  }

  static Future<void> save(List<Session> sessions) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final trimmed = sessions.length > maxSessions
          ? sessions.sublist(0, maxSessions)
          : sessions;
      await prefs.setString(
        _key,
        jsonEncode(<Map<String, dynamic>>[
          for (final s in trimmed) s.toJson(),
        ]),
      );
    } catch (_) {
      // 写盘失败只当作这一次没存上，不打断使用
    }
  }
}
