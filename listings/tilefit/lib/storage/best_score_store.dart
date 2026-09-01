import 'package:shared_preferences/shared_preferences.dart';

/// 最高分的本地存档。
///
/// 只存一个整数，全程容错：读写失败一律当"没有记录"处理 —— 存档坏了顶多丢个最高分，
/// 绝不能让游戏打不开。数据不出设备，不参与任何上报（见 DATA_SAFETY.md）。
class BestScoreStore {
  const BestScoreStore();

  static const String _key = 'tilefit_best_score';

  Future<int> read() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getInt(_key) ?? 0;
    } catch (_) {
      return 0;
    }
  }

  /// 仅在 [score] 高于已存值时写入。返回写入后的最高分。
  Future<int> writeIfHigher(int score) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final current = prefs.getInt(_key) ?? 0;
      if (score <= current) return current;
      await prefs.setInt(_key, score);
      return score;
    } catch (_) {
      return score;
    }
  }
}
