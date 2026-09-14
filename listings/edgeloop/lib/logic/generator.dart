import 'dart:math';

import 'puzzle.dart';
import 'solver.dart';

/// 生成保证**唯一解**的 EdgeLoop 题面。
///
/// ## 为什么先造解、再反推提示
///
/// 反过来做（随便填一堆数字，再看有没有解）几乎必然产出无解题面：提示数之间的相容性
/// 很脆。这里走与 LinkFlow 同一条路 —— 先造一个必然合法的解，再从解反推提示数，
/// 最后往回删提示，删到「再删就不唯一」为止。这样每一步都站在合法解上，
/// 不存在「生成了一堆废题再过滤」的浪费。
///
/// ## 回路怎么造
///
/// 关键观察：**一块无洞的连通格子区域，它的外边界就是一条闭合回路。**
/// 所以不去直接画线，而是随机「长」一块区域出来，再取它的边界。
///
/// 两个必须挡掉的退化情形：
///
///   1. **区域有洞** —— 边界会变成两条回路（外圈 + 洞的内圈），不是单回路。
///   2. **夹点（pinch）** —— 区域在某个点上只靠对角相接，那个点的度会是 4，
///      而 Slitherlink 要求每点度为 0 或 2。形如：
///
///          ██·           这两块只在中间那个点相碰，
///          ·██           该点连出四条边界边 → 非法。
///
/// 两者都在 [_boundaryOf] 里检出并拒绝，而不是事后补救 —— 补救要动区域形状，
/// 很容易在修一个夹点时造出另一个。
class EdgeLoopGenerator {
  EdgeLoopGenerator(this.random);

  final Random random;

  /// 尝试生成一道 rows×cols 的唯一解题面。
  ///
  /// [targetCells] 是区域格子数的目标（决定回路长短）；[maxAttempts] 用尽仍未成功返回 null，
  /// 由调用方决定是换参数还是继续试。
  EdgeLoopPuzzle? generate({
    required int rows,
    required int cols,
    required int targetCells,
    int maxAttempts = 200,
  }) {
    for (var attempt = 0; attempt < maxAttempts; attempt++) {
      final region = _growRegion(rows, cols, targetCells);
      if (region == null) continue;

      final full = _boundaryOf(region, rows, cols);
      if (full == null) continue; // 有洞或有夹点

      final puzzle = _reduce(rows, cols, full);
      if (puzzle != null) return puzzle;
    }
    return null;
  }

  /// 随机长出一块连通区域。返回被选中格子的集合（行优先下标），失败返回 null。
  Set<int>? _growRegion(int rows, int cols, int targetCells) {
    final total = rows * cols;
    if (targetCells < 1 || targetCells > total) return null;

    final inRegion = <int>{};
    final start = random.nextInt(total);
    inRegion.add(start);

    // 边沿：与区域相邻、尚未加入的格子。每次随机挑一个加进来。
    final frontier = <int>{};
    void pushNeighbours(int cell) {
      final r = cell ~/ cols, c = cell % cols;
      for (final d in const [
        [-1, 0],
        [1, 0],
        [0, -1],
        [0, 1]
      ]) {
        final nr = r + d[0], nc = c + d[1];
        if (nr < 0 || nr >= rows || nc < 0 || nc >= cols) continue;
        final n = nr * cols + nc;
        if (!inRegion.contains(n)) frontier.add(n);
      }
    }

    pushNeighbours(start);
    while (inRegion.length < targetCells && frontier.isNotEmpty) {
      final pick = frontier.elementAt(random.nextInt(frontier.length));
      frontier.remove(pick);
      inRegion.add(pick);
      pushNeighbours(pick);
    }
    if (inRegion.length < targetCells) return null;
    return inRegion;
  }

  /// 取区域边界，同时验证它是一条合法的单回路。
  ///
  /// 返回 edgeCount 长的 0/1 表；若区域有洞或存在夹点则返回 null。
  List<int>? _boundaryOf(Set<int> region, int rows, int cols) {
    final p = EdgeLoopPuzzle(
      rows: rows,
      cols: cols,
      clues: List<int?>.filled(rows * cols, null),
    );
    final edges = List<int>.filled(p.edgeCount, 0);

    bool inside(int r, int c) =>
        r >= 0 && r < rows && c >= 0 && c < cols && region.contains(r * cols + c);

    // 一条边在边界上 ⟺ 它两侧的格子一个在区域内、一个不在（棋盘外视作不在）。
    for (var r = 0; r <= rows; r++) {
      for (var c = 0; c < cols; c++) {
        if (inside(r - 1, c) != inside(r, c)) edges[p.hIndex(r, c)] = 1;
      }
    }
    for (var r = 0; r < rows; r++) {
      for (var c = 0; c <= cols; c++) {
        if (inside(r, c - 1) != inside(r, c)) edges[p.vIndex(r, c)] = 1;
      }
    }

    // 夹点检查：任何点的度都必须是 0 或 2。度为 4 即对角相接。
    for (var r = 0; r <= rows; r++) {
      for (var c = 0; c <= cols; c++) {
        var deg = 0;
        for (final e in p.dotEdges(r, c)) {
          deg += edges[e];
        }
        if (deg != 0 && deg != 2) return null;
      }
    }

    // 有洞检查：从棋盘外围做洪泛，若有「区域外的格子」泛不到，那就是被围住的洞。
    if (_hasHole(region, rows, cols)) return null;

    return edges;
  }

  /// 区域外的格子若不能全部从棋盘边缘连通到，说明区域内部包了洞。
  bool _hasHole(Set<int> region, int rows, int cols) {
    final outside = <int>{};
    final queue = <int>[];
    void seed(int r, int c) {
      final id = r * cols + c;
      if (region.contains(id) || outside.contains(id)) return;
      outside.add(id);
      queue.add(id);
    }

    for (var c = 0; c < cols; c++) {
      seed(0, c);
      seed(rows - 1, c);
    }
    for (var r = 0; r < rows; r++) {
      seed(r, 0);
      seed(r, cols - 1);
    }

    while (queue.isNotEmpty) {
      final cur = queue.removeLast();
      final r = cur ~/ cols, c = cur % cols;
      for (final d in const [
        [-1, 0],
        [1, 0],
        [0, -1],
        [0, 1]
      ]) {
        final nr = r + d[0], nc = c + d[1];
        if (nr < 0 || nr >= rows || nc < 0 || nc >= cols) continue;
        seed(nr, nc);
      }
    }

    final outsideTotal = rows * cols - region.length;
    return outside.length != outsideTotal;
  }

  /// 从完整提示出发随机删提示，删到不能再删（再删就不唯一）为止。
  EdgeLoopPuzzle? _reduce(int rows, int cols, List<int> solution) {
    final p0 = EdgeLoopPuzzle(
      rows: rows,
      cols: cols,
      clues: List<int?>.filled(rows * cols, null),
    );

    // 从解反推每格的提示数：该格四条边里有几条在回路上。
    final clues = List<int?>.filled(rows * cols, null);
    for (var r = 0; r < rows; r++) {
      for (var c = 0; c < cols; c++) {
        var n = 0;
        for (final e in p0.cellEdges(r, c)) {
          n += solution[e];
        }
        clues[r * cols + c] = n;
      }
    }

    // 提示全给时必然唯一吗？不一定 —— 所以先验一次，不合格直接换一个回路重来。
    final full = EdgeLoopPuzzle(rows: rows, cols: cols, clues: clues);
    if (!EdgeLoopSolver(full).solve().isUnique) return null;

    final order = List<int>.generate(rows * cols, (i) => i)..shuffle(random);
    for (final idx in order) {
      final keep = clues[idx];
      if (keep == null) continue;
      clues[idx] = null;
      final probe = EdgeLoopPuzzle(
        rows: rows,
        cols: cols,
        clues: List<int?>.from(clues),
      );
      if (!EdgeLoopSolver(probe).solve().isUnique) {
        clues[idx] = keep; // 删不得，放回去
      }
    }

    return EdgeLoopPuzzle(rows: rows, cols: cols, clues: clues);
  }
}
