import 'dart:math';

import 'difficulty.dart';
import 'mine_field.dart';
import 'solver.dart';

/// 生成结果。[noGuess] 为 false 表示试满了预算也没抽到「不用猜」的布局，
/// 只能拿最后一次的凑合 —— 调用方照玩，但那一盘不保证无需猜。
class GeneratedField {
  const GeneratedField(this.field, {required this.noGuess, required this.attempts});

  final MineField field;
  final bool noGuess;
  final int attempts;
}

/// 布雷。
///
/// 做法是**随机布 + 逻辑验 + 不合格就重来**，而不是「布完再修补」。
/// 理由是重来这条路好证也好测：接受的每一盘都真的被求解器走通过一遍；
/// 修补法要小心翼翼地保证改完不破坏别处，出错了还极难复现。
/// 代价是要多试几次，实测在本包的四个档位下都在毫秒级（见 generator_test.dart）。
class MineGenerator {
  MineGenerator._();

  /// 首点及其 3×3 邻域一定没有雷 —— 第一下必然是安全的，而且必然是个空格，
  /// 一按就摊开一片。扫雷开局如果只翻出个孤零零的数字，接下来就只能瞎点了。
  static GeneratedField generate(
    Difficulty difficulty,
    int firstClick, {
    Random? random,
    int maxAttempts = 4000,
  }) {
    final rng = random ?? Random();
    final cols = difficulty.cols;
    final rows = difficulty.rows;
    final total = difficulty.cellCount;

    final safe = _blockAround(firstClick, cols, rows);
    final candidates = <int>[
      for (var i = 0; i < total; i++)
        if (!safe.contains(i)) i,
    ];

    MineField? last;
    for (var attempt = 1; attempt <= maxAttempts; attempt++) {
      final mines = _pick(candidates, difficulty.mines, rng);
      final field = MineField(cols: cols, rows: rows, mines: mines);
      last = field;
      if (MineSolver.isNoGuess(field, firstClick)) {
        return GeneratedField(field, noGuess: true, attempts: attempt);
      }
    }

    // 兜底：预算用完还没抽到。宁可发一盘可能要猜的，也不能卡住不开局。
    return GeneratedField(last!, noGuess: false, attempts: maxAttempts);
  }

  /// 从 [candidates] 里等概率取 [count] 个（部分 Fisher-Yates，不动原列表）。
  static Set<int> _pick(List<int> candidates, int count, Random rng) {
    final pool = List<int>.of(candidates);
    final picked = <int>{};
    for (var i = 0; i < count && i < pool.length; i++) {
      final j = i + rng.nextInt(pool.length - i);
      final t = pool[i];
      pool[i] = pool[j];
      pool[j] = t;
      picked.add(pool[i]);
    }
    return picked;
  }

  static Set<int> _blockAround(int index, int cols, int rows) {
    final r = index ~/ cols;
    final c = index % cols;
    final out = <int>{};
    for (var dr = -1; dr <= 1; dr++) {
      for (var dc = -1; dc <= 1; dc++) {
        final nr = r + dr;
        final nc = c + dc;
        if (nr < 0 || nr >= rows || nc < 0 || nc >= cols) continue;
        out.add(nr * cols + nc);
      }
    }
    return out;
  }
}
