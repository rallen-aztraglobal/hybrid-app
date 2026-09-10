import 'nonogram.dart';

/// 玩家在一格上的标记。
///
/// [crossed] 是「我确定这里是空的」，纯辅助记号 —— 判胜时不看它，
/// 数织的胜负只取决于该涂黑的格子有没有被涂满、且没有多涂。
enum CellMark { blank, filled, crossed }

/// 一局数织的状态机。纯数据，不依赖 Flutter，便于直接测。
///
/// 与答案的关系：[puzzle] 只读，玩家的涂改都记在 [_marks] 里。
/// 这样「重开」只要清空标记，不必重新生成谜题。
class NonogramGame {
  NonogramGame(this.puzzle)
      : _marks = <List<CellMark>>[
          for (var r = 0; r < puzzle.rows; r++)
            List<CellMark>.filled(puzzle.cols, CellMark.blank),
        ];

  final Nonogram puzzle;
  final List<List<CellMark>> _marks;

  CellMark markAt(int row, int col) => _marks[row][col];

  /// **当前**涂错的格子数：现在被涂黑、但答案里该留空的格子。
  ///
  /// 刻意不是累计计数。早先的实现累计「涂错过几次」且撤销不倒扣，结果是
  /// 玩家涂错再撤销后棋盘明明空了、界面却还写着「1 wrong」，自相矛盾 ——
  /// 因为它就显示在进度旁边，整行读起来是在描述当前棋盘。
  ///
  /// 而且累计数在本作里没有玩法作用（没有「三次出局」那类命数机制），
  /// 改成当前态之后它才有用：告诉玩家「还有几格要找出来改掉」。
  int get wrongCount {
    var n = 0;
    for (var r = 0; r < puzzle.rows; r++) {
      for (var c = 0; c < puzzle.cols; c++) {
        if (_marks[r][c] == CellMark.filled && !puzzle.isFilledAt(r, c)) n++;
      }
    }
    return n;
  }

  /// 已正确涂黑的格子数。进度条用。
  int get correctCount {
    var n = 0;
    for (var r = 0; r < puzzle.rows; r++) {
      for (var c = 0; c < puzzle.cols; c++) {
        if (_marks[r][c] == CellMark.filled && puzzle.isFilledAt(r, c)) n++;
      }
    }
    return n;
  }

  /// 是否已完成：答案里每个该涂的格子都涂了，且没有涂到不该涂的格子。
  ///
  /// 刻意不要求把留空处全部打叉 —— 叉只是玩家的备忘，强制打满是无谓的负担。
  bool get isSolved {
    for (var r = 0; r < puzzle.rows; r++) {
      for (var c = 0; c < puzzle.cols; c++) {
        final wantFilled = puzzle.isFilledAt(r, c);
        final isFilled = _marks[r][c] == CellMark.filled;
        if (wantFilled != isFilled) return false;
      }
    }
    return true;
  }

  /// 在一格上落笔。[intent] 是本次手势想要施加的标记。
  ///
  /// 同一标记再点一次 = 撤销，回到 [CellMark.blank]。这样单指就能反复试，
  /// 不需要额外的橡皮擦模式。
  ///
  /// 返回 true 表示这一笔涂错了（把该留空的格子涂黑），调用方可以据此震动提示。
  bool apply(int row, int col, CellMark intent) {
    if (intent == CellMark.blank) {
      _marks[row][col] = CellMark.blank;
      return false;
    }

    if (_marks[row][col] == intent) {
      _marks[row][col] = CellMark.blank; // 再点一次取消
      return false;
    }

    _marks[row][col] = intent;

    // 返回值只用于「这一笔错了」的即时反馈（震动），与 [wrongCount] 是两件事：
    // 前者是事件、后者是状态。
    return intent == CellMark.filled && !puzzle.isFilledAt(row, col);
  }

  /// 清空全部标记，重来一局（谜题不变）。
  void reset() {
    for (var r = 0; r < puzzle.rows; r++) {
      for (var c = 0; c < puzzle.cols; c++) {
        _marks[r][c] = CellMark.blank;
      }
    }
  }

  /// 某一行的线索是否已被满足 —— 该行当前涂黑的分布恰好等于答案。
  ///
  /// 界面用它把已完成的线索淡掉，是数织里很实用的视觉辅助。
  bool isRowSatisfied(int row) {
    for (var c = 0; c < puzzle.cols; c++) {
      if ((_marks[row][c] == CellMark.filled) != puzzle.isFilledAt(row, c)) {
        return false;
      }
    }
    return true;
  }

  bool isColSatisfied(int col) {
    for (var r = 0; r < puzzle.rows; r++) {
      if ((_marks[r][col] == CellMark.filled) != puzzle.isFilledAt(r, col)) {
        return false;
      }
    }
    return true;
  }
}
