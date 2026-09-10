import 'board.dart';
import 'puzzle.dart';

/// 一次求解的结果。
class SolveResult {
  const SolveResult({
    required this.solutions,
    required this.nodes,
    required this.exhausted,
  });

  /// 找到的解数。搜索可能因为达到 [FlowSolver.solutionCap] 提前停，
  /// 所以这个数不一定是全部解 —— 判唯一要连 [exhausted] 一起看。
  final int solutions;

  /// 搜索访问过的节点数。生成题库时拿它当难度的代理量。
  final int nodes;

  /// 搜索空间是否走完了。false 表示撞上节点上限，结论不完整、不可信。
  final bool exhausted;

  /// 唯一解：搜完了，而且只有一个解。
  bool get isUnique => exhausted && solutions == 1;
}

/// 连线题求解器：枚举「每条通路两端相连 + 整盘铺满」的所有填法。
///
/// 两个用途：
/// - 生成题库时筛掉多解题（见 `tool/generate_levels.dart`）；
/// - 测试里守住「每道出厂题都只有一个解」这条不变量。
///
/// **判定口径与 [FlowGame.isSolved] 完全一致**：只要求全连通 + 铺满，
/// 不额外禁止一条通路自己贴着自己走。口径更宽 ⇒ 解更多 ⇒ 唯一性要求更严。
/// 反过来做（按「不许自贴」去数解）会把实际游戏里存在的第二种走法漏掉，
/// 玩家就会用一种「非标准但游戏认可」的画法过关，而我们还以为题是唯一解的。
///
/// 应用代码不引用这个类，release 构建会被 tree shaking 摘掉。
class FlowSolver {
  FlowSolver(
    FlowPuzzle puzzle, {
    this.solutionCap = 2,
    this.nodeCap = 400000,
  })  : _side = puzzle.side,
        _n = puzzle.side * puzzle.side,
        _colors = puzzle.colorKeys.length {
    _grid = List<int>.filled(_n, -1);
    _startIdx = List<int>.filled(_colors, 0);
    _endIdx = List<int>.filled(_colors, 0);
    _head = List<int>.filled(_colors, 0);
    _done = List<bool>.filled(_colors, false);
    _visit = List<int>.filled(_n, 0);
    _queue = List<int>.filled(_n, 0);

    _nbrs = List<List<int>>.generate(_n, (i) {
      final cell = Cell(i ~/ _side, i % _side);
      return <int>[
        for (final q in cell.neighbours)
          if (q.row >= 0 && q.row < _side && q.col >= 0 && q.col < _side)
            q.row * _side + q.col,
      ];
    });

    for (var i = 0; i < _colors; i++) {
      final cells = puzzle.paths[puzzle.colorKeys[i]]!;
      final s = _index(cells.first);
      final e = _index(cells.last);
      // 端点重合或与别的通路撞车 —— 这题本身就是坏的，直接判 0 解。
      if (s == e || _grid[s] != -1 || _grid[e] != -1) {
        _broken = true;
        return;
      }
      _startIdx[i] = s;
      _endIdx[i] = e;
      _head[i] = s;
      _grid[s] = i;
      _grid[e] = i;
    }
  }

  /// 找到这么多解就停。判唯一只需要 2。
  final int solutionCap;

  /// 节点上限。撞上就放弃这题（生成器会换一道），避免个别病态题卡死。
  final int nodeCap;

  final int _side;
  final int _n;
  final int _colors;

  late final List<List<int>> _nbrs;
  late final List<int> _grid; // -1 空，否则是通路下标
  late final List<int> _startIdx;
  late final List<int> _endIdx;
  late final List<int> _head; // 每条通路当前画到哪
  late final List<bool> _done;

  // BFS 复用的暂存区。用「时间戳」代替每次清零的 visited 数组。
  late final List<int> _visit;
  late final List<int> _queue;
  int _stamp = 0;

  bool _broken = false;
  int _nodes = 0;
  int _solutions = 0;
  bool _hitNodeCap = false;

  int _index(Cell c) => c.row * _side + c.col;

  SolveResult run() {
    if (_broken) {
      return const SolveResult(solutions: 0, nodes: 0, exhausted: true);
    }
    _nodes = 0;
    _solutions = 0;
    _hitNodeCap = false;
    _search();
    return SolveResult(
      solutions: _solutions,
      nodes: _nodes,
      exhausted: !_hitNodeCap,
    );
  }

  /// 深搜。每一步都**确定性地**挑一条通路来延伸（可选走法最少的那条），
  /// 所以每个完整填法在搜索树里只会被生成一次 —— 解数不会重复计。
  void _search() {
    if (_nodes >= nodeCap) {
      _hitNodeCap = true;
      return;
    }
    _nodes++;
    if (!_prune()) return;

    var best = -1;
    var bestCount = 1 << 30;
    for (var i = 0; i < _colors; i++) {
      if (_done[i]) continue;
      var count = 0;
      for (final q in _nbrs[_head[i]]) {
        if (q == _endIdx[i] || _grid[q] == -1) count++;
      }
      if (count == 0) return; // 这条走死了
      if (count < bestCount) {
        bestCount = count;
        best = i;
      }
    }

    if (best == -1) {
      // 所有通路都接上了 —— 还得整盘铺满才算解
      for (var x = 0; x < _n; x++) {
        if (_grid[x] == -1) return;
      }
      _solutions++;
      return;
    }

    final head = _head[best];
    for (final q in _nbrs[head]) {
      if (q == _endIdx[best]) {
        _done[best] = true;
        _search();
        _done[best] = false;
      } else if (_grid[q] == -1) {
        _grid[q] = best;
        _head[best] = q;
        _search();
        _head[best] = head;
        _grid[q] = -1;
      } else {
        continue;
      }
      if (_solutions >= solutionCap || _hitNodeCap) return;
    }
  }

  /// 剪枝。三条都必须是**保守**的 —— 只砍掉确实不可能有解的分支，
  /// 否则会把真解剪没、把多解题误判成唯一解。
  bool _prune() {
    // ① 每条未完成的通路，head 必须还能穿过空格走到自己的终点
    for (var i = 0; i < _colors; i++) {
      if (_done[i]) continue;
      if (!_canReach(_head[i], _endIdx[i])) return false;
    }

    // ② 每个空格都得至少能被某条未完成通路够到，否则永远填不上
    if (!_allEmptyReachable()) return false;

    // ③ 空格至少要有两个「可用邻居」。
    //    空格一定是某条通路的中段（端点都是预先填好的），中段格在最终答案里
    //    正好有两个同路邻居 —— 一进一出。可用邻居不足 2 个就是死格。
    for (var x = 0; x < _n; x++) {
      if (_grid[x] != -1) continue;
      var usable = 0;
      for (final q in _nbrs[x]) {
        if (_usable(q)) {
          usable++;
          if (usable >= 2) break;
        }
      }
      if (usable < 2) return false;
    }

    return true;
  }

  /// 这一格能不能成为某个空格在最终答案里的同路邻居。
  ///
  /// 空格自己当然可以；已填的格子只有两种还「留着一个接口」：
  /// 未完成通路的 head（还要往外长一格）和它的终点（还要被接上一格）。
  /// 其余已填格的两个接口都已经定死了。
  bool _usable(int q) {
    final g = _grid[q];
    if (g == -1) return true;
    if (_done[g]) return false;
    return q == _head[g] || q == _endIdx[g];
  }

  /// [from] 能否穿过空格抵达 [target]。
  bool _canReach(int from, int target) {
    _stamp++;
    _visit[from] = _stamp;
    _queue[0] = from;
    var qh = 0;
    var qt = 1;
    while (qh < qt) {
      final c = _queue[qh++];
      for (final q in _nbrs[c]) {
        if (q == target) return true;
        if (_grid[q] != -1 || _visit[q] == _stamp) continue;
        _visit[q] = _stamp;
        _queue[qt++] = q;
      }
    }
    return false;
  }

  /// 从所有未完成通路的 head 出发做多源 BFS，看是否覆盖了全部空格。
  bool _allEmptyReachable() {
    _stamp++;
    var qh = 0;
    var qt = 0;
    for (var i = 0; i < _colors; i++) {
      if (_done[i]) continue;
      for (final q in _nbrs[_head[i]]) {
        if (_grid[q] == -1 && _visit[q] != _stamp) {
          _visit[q] = _stamp;
          _queue[qt++] = q;
        }
      }
    }
    while (qh < qt) {
      final c = _queue[qh++];
      for (final q in _nbrs[c]) {
        if (_grid[q] == -1 && _visit[q] != _stamp) {
          _visit[q] = _stamp;
          _queue[qt++] = q;
        }
      }
    }

    var empties = 0;
    for (var x = 0; x < _n; x++) {
      if (_grid[x] == -1) empties++;
    }
    return qt == empties;
  }
}
