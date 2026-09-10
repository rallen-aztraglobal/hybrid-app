import 'line_solver.dart';

/// 一道数织谜题：答案 + 由答案导出的行列线索。
///
/// 不可变。谜题一旦生成就不再变化，玩家的涂改状态放在 [NonogramGame] 里，
/// 两者分开是为了「重开一局」只需丢掉游戏状态、不必重新生成谜题。
class Nonogram {
  Nonogram._({
    required this.rows,
    required this.cols,
    required this.solution,
    required this.rowClues,
    required this.colClues,
  });

  /// 由答案表构造，线索自动导出。
  ///
  /// [solution] 必须是 rows × cols 的矩形表，`true` 表示该格涂黑。
  factory Nonogram.fromSolution(List<List<bool>> solution) {
    final rows = solution.length;
    if (rows == 0) throw ArgumentError('答案表不能为空');
    final cols = solution.first.length;
    if (cols == 0) throw ArgumentError('答案表的行不能为空');
    for (final row in solution) {
      if (row.length != cols) throw ArgumentError('答案表必须是矩形');
    }

    final rowClues = <List<int>>[
      for (final row in solution) cluesOf(row),
    ];
    final colClues = <List<int>>[
      for (var c = 0; c < cols; c++)
        cluesOf(<bool>[for (var r = 0; r < rows; r++) solution[r][c]]),
    ];

    return Nonogram._(
      rows: rows,
      cols: cols,
      solution: <List<bool>>[for (final row in solution) List<bool>.unmodifiable(row)],
      rowClues: <List<int>>[for (final c in rowClues) List<int>.unmodifiable(c)],
      colClues: <List<int>>[for (final c in colClues) List<int>.unmodifiable(c)],
    );
  }

  final int rows;
  final int cols;
  final List<List<bool>> solution;

  /// 每行的线索，自上而下。
  final List<List<int>> rowClues;

  /// 每列的线索，自左向右。
  final List<List<int>> colClues;

  /// 答案里需要涂黑的格子总数。用于进度显示。
  int get filledCount {
    var n = 0;
    for (final row in solution) {
      for (final cell in row) {
        if (cell) n++;
      }
    }
    return n;
  }

  bool isFilledAt(int row, int col) => solution[row][col];

  /// 本题能否**纯靠逐行逐列的逻辑推演**解出（不需要试错猜测）。
  ///
  /// 反复对每行每列做一次 [refineLine]，把新确定的格子写回，直到某一轮没有任何
  /// 新进展。若此时全部格子都已确定，说明这道题「线可解」——
  /// 而线可解蕴含唯一解，因为推演每一步只产出被迫的结论。
  ///
  /// 生成器用这个作为验收标准，避免出现必须猜的题。
  bool get isLineSolvable {
    final known = <List<bool?>>[
      for (var r = 0; r < rows; r++) List<bool?>.filled(cols, null),
    ];

    var progressed = true;
    while (progressed) {
      progressed = false;

      for (var r = 0; r < rows; r++) {
        final refined = refineLine(rowClues[r], known[r]);
        if (refined == null) return false; // 矛盾，理论上不该发生
        for (var c = 0; c < cols; c++) {
          if (known[r][c] == null && refined[c] != null) {
            known[r][c] = refined[c];
            progressed = true;
          }
        }
      }

      for (var c = 0; c < cols; c++) {
        final column = <bool?>[for (var r = 0; r < rows; r++) known[r][c]];
        final refined = refineLine(colClues[c], column);
        if (refined == null) return false;
        for (var r = 0; r < rows; r++) {
          if (known[r][c] == null && refined[r] != null) {
            known[r][c] = refined[r];
            progressed = true;
          }
        }
      }
    }

    for (var r = 0; r < rows; r++) {
      for (var c = 0; c < cols; c++) {
        if (known[r][c] == null) return false; // 卡住了，需要猜
      }
    }
    return true;
  }
}
