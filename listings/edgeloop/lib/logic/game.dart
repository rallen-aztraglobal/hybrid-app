import 'puzzle.dart';

/// 一条边在玩家眼里的三种状态。
///
/// `cross`（打叉 = 确定不画）不是装饰 —— 它是这类题的主要推理工具，
/// 「这里一定没有线」往往比「这里一定有线」先被推出来。少了它，中等以上的题基本没法下手。
enum EdgeMark { none, line, cross }

/// 一局游戏的可变状态。
///
/// 与题面（[EdgeLoopPuzzle]，不可变）分开：重开一局只是把 marks 清空，题面不动。
class EdgeLoopGame {
  EdgeLoopGame(this.puzzle)
      : marks = List<EdgeMark>.filled(puzzle.edgeCount, EdgeMark.none);

  final EdgeLoopPuzzle puzzle;
  final List<EdgeMark> marks;

  /// 点一下边：none → line → cross → none。
  ///
  /// 三态循环而不是两态，是因为打叉必须能取消 —— 推错了要能退回来。
  void cycle(int edge) {
    marks[edge] = switch (marks[edge]) {
      EdgeMark.none => EdgeMark.line,
      EdgeMark.line => EdgeMark.cross,
      EdgeMark.cross => EdgeMark.none,
    };
  }

  void clear() {
    for (var i = 0; i < marks.length; i++) {
      marks[i] = EdgeMark.none;
    }
  }

  int get lineCount => marks.where((m) => m == EdgeMark.line).length;

  /// 某格已画的边数 —— 界面用它把「已满足」的提示数变灰。
  int linesAroundCell(int r, int c) {
    var n = 0;
    for (final e in puzzle.cellEdges(r, c)) {
      if (marks[e] == EdgeMark.line) n++;
    }
    return n;
  }

  /// 该格的提示是否已被违反（画多了）。界面据此标红。
  ///
  /// 只判「超了」，不判「还差」—— 没画完不是错误，是进行中。
  bool cellViolated(int r, int c) {
    final clue = puzzle.clueAt(r, c);
    if (clue < 0) return false;
    return linesAroundCell(r, c) > clue;
  }

  /// 某点已画的边数。度 >2 是明确的错误（一个点最多连出两条线）。
  int linesAroundDot(int r, int c) {
    var n = 0;
    for (final e in puzzle.dotEdges(r, c)) {
      if (marks[e] == EdgeMark.line) n++;
    }
    return n;
  }

  /// 是否已解出。
  ///
  /// 判定口径**必须**与生成器所用的求解器完全一致，否则「唯一解」这个卖点就名不副实：
  /// 生成器认的解，这里也得认；这里认的，生成器也必须认得。三条缺一不可 ——
  ///
  ///   1. 每个有提示的格子，画上的边数正好等于提示数；
  ///   2. 每个点的度是 0 或 2；
  ///   3. 画上的边构成**恰好一条**闭合回路（不是两个环，也不是空盘）。
  ///
  /// 打叉不参与判定：它只是玩家的备忘，画没画线才算数。
  bool get isSolved {
    for (var r = 0; r < puzzle.rows; r++) {
      for (var c = 0; c < puzzle.cols; c++) {
        final clue = puzzle.clueAt(r, c);
        if (clue >= 0 && linesAroundCell(r, c) != clue) return false;
      }
    }
    for (var r = 0; r <= puzzle.rows; r++) {
      for (var c = 0; c <= puzzle.cols; c++) {
        final d = linesAroundDot(r, c);
        if (d != 0 && d != 2) return false;
      }
    }
    return _isSingleLoop();
  }

  bool _isSingleLoop() {
    final on = <int>[];
    for (var e = 0; e < marks.length; e++) {
      if (marks[e] == EdgeMark.line) on.add(e);
    }
    if (on.isEmpty) return false;

    final ends = <int, List<int>>{}; // 边 -> 两端点
    final adj = <int, List<int>>{}; // 点 -> 关联的已画边
    for (final e in on) {
      final pair = _endsOf(e);
      ends[e] = pair;
      for (final d in pair) {
        (adj[d] ??= <int>[]).add(e);
      }
    }

    final seen = <int>{};
    final start = on.first;
    var edge = start;
    var dot = ends[start]![0];
    while (true) {
      seen.add(edge);
      final pair = ends[edge]!;
      dot = pair[0] == dot ? pair[1] : pair[0];
      final next = adj[dot]!;
      if (next.length != 2) return false;
      final nextEdge = next[0] == edge ? next[1] : next[0];
      if (nextEdge == start) break;
      if (seen.contains(nextEdge)) return false;
      edge = nextEdge;
    }
    return seen.length == on.length;
  }

  /// 边 -> 它两端点的 id（点 id = r*(cols+1)+c）。
  List<int> _endsOf(int edge) {
    final p = puzzle;
    final stride = p.cols + 1;
    if (edge < p.hCount) {
      final r = edge ~/ p.cols;
      final c = edge % p.cols;
      return <int>[r * stride + c, r * stride + c + 1];
    }
    final v = edge - p.hCount;
    final r = v ~/ stride;
    final c = v % stride;
    return <int>[r * stride + c, (r + 1) * stride + c];
  }
}
