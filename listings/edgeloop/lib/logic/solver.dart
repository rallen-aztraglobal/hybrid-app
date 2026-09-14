import 'puzzle.dart';

/// 求解结果。`solutions` 被 [EdgeLoopSolver.solutionCap] 截断，
/// 所以它只用来回答「0 解 / 1 解 / ≥2 解」，不是完整解集。
class SolveResult {
  const SolveResult({
    required this.solutions,
    required this.nodes,
    required this.exhausted,
  });

  /// 找到的解（每个解是一张 edgeCount 长的 0/1 表）。最多 [EdgeLoopSolver.solutionCap] 个。
  final List<List<int>> solutions;

  /// 搜索展开的节点数，用于衡量题目难度与生成器性能。
  final int nodes;

  /// true = 搜索完整跑完（没被节点上限截断），结论可信。
  /// false = 撞上限提前退出，此时**不能**断言唯一性。
  final bool exhausted;

  /// 恰好一个解，且搜索完整跑完。生成器只认这个。
  bool get isUnique => exhausted && solutions.length == 1;
}

/// 穷举求解器，用于**证明唯一解**。
///
/// 不是给玩家用的提示器 —— 它不模拟人的解题技巧，只回答「解有几个」。
///
/// ## 为什么试边的顺序是关键
///
/// 天真地按「先所有横边、再所有竖边」去试，约束要到最后才凑齐，等于全量穷举。
/// 这里按行交错：
///
///     H(0,*) → V(0,*) → H(1,*) → V(1,*) → … → V(rows-1,*) → H(rows,*)
///
/// 这个顺序下，格子 (r,c) 的四条边在 H(r+1,c) 被赋值的那一刻正好集齐，
/// 点 (r,c) 则在 V(r,c) 赋值时集齐 —— 于是「提示数必须正好相等」「点的度必须是 0 或 2」
/// 这两条硬约束能在搜索很浅的地方就开始杀分支，而不是等到叶子。
///
/// 具体「谁在第几步集齐」不靠手推，而是构造时按边的实际顺序算出来（见 [_completionPlan]）——
/// 手推很容易在边界情形（最后一行、最右一列的点）上出错，而那种错会让求解器漏解，
/// 进而把多解题误判成唯一解，是最危险的一类 bug。
class EdgeLoopSolver {
  EdgeLoopSolver(this.puzzle, {this.solutionCap = 2, this.nodeCap = 2000000}) {
    _buildOrder();
    _completionPlan();
  }

  final EdgeLoopPuzzle puzzle;

  /// 找到这么多解就停 —— 判唯一性只需要知道「是否 ≥2」。
  final int solutionCap;

  /// 节点上限，防止病态题面把生成器卡死。撞上限时 `exhausted=false`。
  final int nodeCap;

  late final List<int> _order; // 赋值顺序：第 i 步处理哪条边
  late final List<List<int>> _cellsOf; // 边 -> 它属于哪些格子（1 或 2 个）
  late final List<List<int>> _dotsOf; // 边 -> 它连接的两个点
  late final List<int> _dotDegree; // 点 -> 关联边数（2/3/4）
  late final List<List<int>> _cellsDoneAt; // 第 i 步之后，哪些格子集齐了
  late final List<List<int>> _dotsDoneAt; // 第 i 步之后，哪些点集齐了

  int get _dotCount => (puzzle.rows + 1) * (puzzle.cols + 1);
  int _dotId(int r, int c) => r * (puzzle.cols + 1) + c;

  void _buildOrder() {
    final p = puzzle;
    final order = <int>[];
    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        order.add(p.hIndex(r, c));
      }
      if (r < p.rows) {
        for (var c = 0; c <= p.cols; c++) {
          order.add(p.vIndex(r, c));
        }
      }
    }
    assert(order.length == p.edgeCount);
    _order = order;

    _cellsOf = List.generate(p.edgeCount, (_) => <int>[]);
    _dotsOf = List.generate(p.edgeCount, (_) => <int>[]);
    _dotDegree = List.filled(_dotCount, 0);

    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        final cell = r * p.cols + c;
        for (final e in p.cellEdges(r, c)) {
          _cellsOf[e].add(cell);
        }
      }
    }
    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        final e = p.hIndex(r, c);
        _dotsOf[e] = <int>[_dotId(r, c), _dotId(r, c + 1)];
      }
    }
    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c <= p.cols; c++) {
        final e = p.vIndex(r, c);
        _dotsOf[e] = <int>[_dotId(r, c), _dotId(r + 1, c)];
      }
    }
    for (var e = 0; e < p.edgeCount; e++) {
      for (final d in _dotsOf[e]) {
        _dotDegree[d]++;
      }
    }
  }

  /// 算出「第 i 步赋值之后，哪些格子/点的边正好全部赋完」。
  /// 做法是取每个格子/点所含边在 [_order] 中的最大位置 —— 那一步就是它集齐的时刻。
  void _completionPlan() {
    final p = puzzle;
    final pos = List.filled(p.edgeCount, -1);
    for (var i = 0; i < _order.length; i++) {
      pos[_order[i]] = i;
    }

    _cellsDoneAt = List.generate(_order.length, (_) => <int>[]);
    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        var last = -1;
        for (final e in p.cellEdges(r, c)) {
          if (pos[e] > last) last = pos[e];
        }
        _cellsDoneAt[last].add(r * p.cols + c);
      }
    }

    _dotsDoneAt = List.generate(_order.length, (_) => <int>[]);
    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c <= p.cols; c++) {
        var last = -1;
        for (final e in p.dotEdges(r, c)) {
          if (pos[e] > last) last = pos[e];
        }
        _dotsDoneAt[last].add(_dotId(r, c));
      }
    }
  }

  SolveResult solve() {
    final p = puzzle;
    final assign = List.filled(p.edgeCount, 0);
    final cellOn = List.filled(p.cellCount, 0);
    final cellAssigned = List.filled(p.cellCount, 0);
    final dotOn = List.filled(_dotCount, 0);
    final dotAssigned = List.filled(_dotCount, 0);

    final solutions = <List<int>>[];
    var nodes = 0;
    var hitCap = false;

    void dfs(int i) {
      if (hitCap || solutions.length >= solutionCap) return;
      if (nodes++ > nodeCap) {
        hitCap = true;
        return;
      }
      if (i == _order.length) {
        if (_isSingleLoop(assign)) {
          solutions.add(List<int>.from(assign));
        }
        return;
      }

      final e = _order[i];
      for (var v = 0; v <= 1; v++) {
        assign[e] = v;

        // 先无条件记账，再统一检查、统一回滚 —— 记账与回滚必须严格对称，
        // 「判死就提前 break 出记账循环」会让计数器漏减，那种错只在深层搜索里
        // 偶发地漏解，极难查。
        for (final k in _cellsOf[e]) {
          cellAssigned[k]++;
          cellOn[k] += v;
        }
        for (final d in _dotsOf[e]) {
          dotAssigned[d]++;
          dotOn[d] += v;
        }

        var ok = true;
        for (final k in _cellsOf[e]) {
          final clue = p.clues[k];
          if (clue == null) continue;
          // 已画的不能超过提示数；剩下的边也必须够补足差额。
          if (cellOn[k] > clue || clue - cellOn[k] > 4 - cellAssigned[k]) {
            ok = false;
            break;
          }
        }
        if (ok) {
          for (final d in _dotsOf[e]) {
            // 点的度只能是 0 或 2：超过 2 立刻死；正好凑齐时若是 1 也死。
            if (dotOn[d] > 2 ||
                (dotAssigned[d] == _dotDegree[d] && dotOn[d] == 1)) {
              ok = false;
              break;
            }
          }
        }

        if (ok) {
          for (final k in _cellsDoneAt[i]) {
            final clue = p.clues[k];
            if (clue != null && cellOn[k] != clue) {
              ok = false;
              break;
            }
          }
        }
        if (ok) {
          for (final d in _dotsDoneAt[i]) {
            if (dotOn[d] != 0 && dotOn[d] != 2) {
              ok = false;
              break;
            }
          }
        }

        if (ok) dfs(i + 1);

        for (final k in _cellsOf[e]) {
          cellAssigned[k]--;
          cellOn[k] -= v;
        }
        for (final d in _dotsOf[e]) {
          dotAssigned[d]--;
          dotOn[d] -= v;
        }
        if (hitCap || solutions.length >= solutionCap) break;
      }
      assign[e] = 0;
    }

    dfs(0);
    return SolveResult(
      solutions: solutions,
      nodes: nodes,
      exhausted: !hitCap,
    );
  }

  /// 判断这组边是否构成**恰好一条**闭合回路。
  ///
  /// 走到这里时每个点的度已经保证是 0 或 2（DFS 里查过），所以画上的边必然是若干条
  /// 不相交的闭环。本函数要排除的是「两个及以上环」和「一条边都没画」这两种情况 ——
  /// 它们都满足全部提示数与度约束，却不是合法解。
  bool _isSingleLoop(List<int> assign) {
    final on = <int>[];
    for (var e = 0; e < assign.length; e++) {
      if (assign[e] == 1) on.add(e);
    }
    if (on.isEmpty) return false; // 空盘不算一条回路

    final adj = <int, List<int>>{};
    for (final e in on) {
      for (final d in _dotsOf[e]) {
        (adj[d] ??= <int>[]).add(e);
      }
    }

    // 从任意一条边出发沿着环走，看能否覆盖全部画上的边。
    final seen = <int>{};
    final start = on.first;
    var edge = start;
    var dot = _dotsOf[start][0];
    while (true) {
      seen.add(edge);
      // 走到边的另一端
      final ends = _dotsOf[edge];
      dot = ends[0] == dot ? ends[1] : ends[0];
      final next = adj[dot]!;
      if (next.length != 2) return false; // 理论上不会发生（度已保证为 2）
      final nextEdge = next[0] == edge ? next[1] : next[0];
      if (nextEdge == start) break;
      if (seen.contains(nextEdge)) return false;
      edge = nextEdge;
    }
    return seen.length == on.length;
  }
}
