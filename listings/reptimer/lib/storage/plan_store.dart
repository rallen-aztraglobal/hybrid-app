import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../logic/interval_plan.dart';

/// RepTimer 的本地存档。
///
/// **别把这句读成「本 App 什么都不收集」** —— 包里带着 AppsFlyer / Adjust / FCM，
/// 那几个 SDK 确实会上报设备标识，Data safety 表单按「收集并共享 Device or other IDs」申报。
/// 这里说的只是：**本地这份数据不离开设备**，因而不属于 Data safety 的申报范围
/// （该表单只管离开设备的数据）。隐私政策里仍要如实写出它存了什么。
///
/// 与另外两个新包的实质差别：这里存的 `name` 是**用户自由输入的文字**
/// （方案名，比如「腿日」「早上那套」），不是几个整数。
/// 所以 DATA_SAFETY / PRIVACY_POLICY / SECURITY_NOTES 里都要单独写明，
/// **不能照抄 DaySpan「只存两个整数」的说法**。
class PlanStore {
  PlanStore._();

  static const String _key = 'reptimer_plans_v1';

  /// 首次打开时给的示例方案。
  ///
  /// 给的是 Tabata（20 秒做 / 10 秒休 × 8）—— 它是这类 App 最广为人知的用法，
  /// 一眼就能看懂各个数字是干什么的，比给一个空列表强。
  static List<IntervalPlan> defaults() => const <IntervalPlan>[
        IntervalPlan(
          name: 'Tabata',
          prepareSeconds: 10,
          workSeconds: 20,
          restSeconds: 10,
          rounds: 8,
        ),
        IntervalPlan(
          name: 'Strength sets',
          prepareSeconds: 15,
          workSeconds: 45,
          restSeconds: 90,
          rounds: 5,
        ),
      ];

  /// 读出全部方案。
  ///
  /// 读那一侧全程兜底：键不存在、不是合法 JSON、顶层不是数组、某条不是对象 ——
  /// 各自兜住，能救几条救几条，绝不抛异常。
  ///
  /// 但**空数组是合法的** —— 用户可能真把方案都删了，那时不该硬塞回示例方案，
  /// 否则删了又冒出来，像 App 不听话。只有「键从没写过」才给示例。
  static Future<List<IntervalPlan>> load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      if (!prefs.containsKey(_key)) return defaults();

      final raw = prefs.getString(_key);
      if (raw == null) return defaults();

      final decoded = jsonDecode(raw);
      if (decoded is! List) return defaults();

      final out = <IntervalPlan>[];
      for (final item in decoded) {
        if (item is! Map) continue;
        try {
          out.add(IntervalPlan.fromJson(Map<String, dynamic>.from(item)));
        } catch (_) {
          // 单条坏了就跳过这条，不连累其余的。
        }
      }
      return out;
    } catch (_) {
      return defaults();
    }
  }

  static Future<void> save(List<IntervalPlan> plans) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(
        _key,
        jsonEncode(plans.map((p) => p.toJson()).toList()),
      );
    } catch (_) {
      // 存不进去不打断当前这次训练。
    }
  }
}
