/// 一盘布好的雷区：哪些格是雷，以及每格周围的雷数。
///
/// 格子用**扁平下标**（`row * cols + col`）而不是 (row, col) 对象。
/// 求解器要反复取邻居、做集合运算，用 int 当键省掉大量装箱和哈希开销；
/// 生成一盘要试上千次，这个差别是能感觉到的。
class MineField {
  MineField({
    required this.cols,
    required this.rows,
    required Set<int> mines,
  }) : mines = Set<int>.unmodifiable(mines) {
    _neighbours = List<List<int>>.generate(cellCount, (i) {
      final r = i ~/ cols;
      final c = i % cols;
      final out = <int>[];
      for (var dr = -1; dr <= 1; dr++) {
        for (var dc = -1; dc <= 1; dc++) {
          if (dr == 0 && dc == 0) continue;
          final nr = r + dr;
          final nc = c + dc;
          if (nr < 0 || nr >= rows || nc < 0 || nc >= cols) continue;
          out.add(nr * cols + nc);
        }
      }
      return out;
    });

    _counts = List<int>.generate(cellCount, (i) {
      if (this.mines.contains(i)) return -1;
      var n = 0;
      for (final q in _neighbours[i]) {
        if (this.mines.contains(q)) n++;
      }
      return n;
    });
  }

  final int cols;
  final int rows;
  final Set<int> mines;

  late final List<List<int>> _neighbours;
  late final List<int> _counts;

  int get cellCount => cols * rows;

  /// 八邻（含斜角），已裁掉越界。
  List<int> neighboursOf(int index) => _neighbours[index];

  bool isMine(int index) => _counts[index] < 0;

  /// 周围的雷数。雷格返回 -1。
  int adjacentMines(int index) => _counts[index];

  /// 以 [index] 为中心的 3×3（含自身）—— 首点安全区就是这一块。
  List<int> blockAround(int index) => <int>[index, ..._neighbours[index]];
}
