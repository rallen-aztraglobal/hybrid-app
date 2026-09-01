import '../models/piece.dart';

/// 一次成功放置的结果。分数计算与动画都只依赖这里的数据。
class PlacementResult {
  const PlacementResult({
    required this.cellsPlaced,
    required this.clearedRows,
    required this.clearedCols,
  });

  /// 本次落子填入的格数（= 方块的格数）。
  final int cellsPlaced;

  /// 本次被消除的行号 / 列号。
  final List<int> clearedRows;
  final List<int> clearedCols;

  /// 本次消除的总线数（行 + 列）。
  int get linesCleared => clearedRows.length + clearedCols.length;
}

/// 棋盘：一个 [size]×[size] 的格子矩阵，每格要么空、要么记着填它那个方块的配色索引。
///
/// 这里不含任何 Flutter 依赖，也不管分数与发牌 —— 那些在 [Game] 里。
/// 拆开是为了让「能不能放下」「消不消得掉」这类规则可以被穷举测试。
class Board {
  Board({this.size = 8})
    : _cells = List<int>.filled(size * size, emptyCell, growable: false);

  /// 空格的哨兵值。配色索引恒 >= 0，故用 -1 表示空不会和真实值撞。
  static const int emptyCell = -1;

  final int size;
  final List<int> _cells;

  int _index(int row, int col) => row * size + col;

  bool _inBounds(int row, int col) =>
      row >= 0 && row < size && col >= 0 && col < size;

  /// 取某格的配色索引；空格返回 [emptyCell]。越界也返回 [emptyCell]，
  /// 让绘制侧不必逐格判边界。
  int cellAt(int row, int col) =>
      _inBounds(row, col) ? _cells[_index(row, col)] : emptyCell;

  bool isFilled(int row, int col) => cellAt(row, col) != emptyCell;

  /// 已填格数。用于「棋盘空了」之类的判断与测试。
  int get filledCount => _cells.where((c) => c != emptyCell).length;

  /// [piece] 的左上角落在 ([row], [col]) 时能否放下：所有格都要在界内且为空。
  bool canPlace(Piece piece, int row, int col) {
    for (final cell in piece.cells) {
      final r = row + cell.row;
      final c = col + cell.col;
      if (!_inBounds(r, c)) return false;
      if (_cells[_index(r, c)] != emptyCell) return false;
    }
    return true;
  }

  /// 棋盘上是否还存在任一能放下 [piece] 的位置。游戏结束判定靠它。
  bool hasAnyPlacement(Piece piece) {
    // 只需扫到方块放得下的最后一个左上角，越界的起点不必试。
    final maxRow = size - piece.height;
    final maxCol = size - piece.width;
    for (var r = 0; r <= maxRow; r++) {
      for (var c = 0; c <= maxCol; c++) {
        if (canPlace(piece, r, c)) return true;
      }
    }
    return false;
  }

  /// 落子并结算消除。调用前必须 [canPlace] 为真，否则抛 [StateError]——
  /// 非法落子是调用方的 bug，静默忽略只会让棋盘悄悄和 UI 不一致。
  PlacementResult place(Piece piece, int row, int col) {
    if (!canPlace(piece, row, col)) {
      throw StateError('方块 ${piece.id} 放不到 ($row,$col)');
    }

    for (final cell in piece.cells) {
      _cells[_index(row + cell.row, col + cell.col)] = piece.colorIndex;
    }

    // 先把要消的行列都找齐、再统一清空。若边找边清，先清掉的行会把与它相交的
    // 那一列打出缺口，导致本该同时消除的列漏判。
    final rows = <int>[];
    final cols = <int>[];
    for (var r = 0; r < size; r++) {
      if (_isRowFull(r)) rows.add(r);
    }
    for (var c = 0; c < size; c++) {
      if (_isColFull(c)) cols.add(c);
    }
    for (final r in rows) {
      for (var c = 0; c < size; c++) {
        _cells[_index(r, c)] = emptyCell;
      }
    }
    for (final c in cols) {
      for (var r = 0; r < size; r++) {
        _cells[_index(r, c)] = emptyCell;
      }
    }

    return PlacementResult(
      cellsPlaced: piece.size,
      clearedRows: List<int>.unmodifiable(rows),
      clearedCols: List<int>.unmodifiable(cols),
    );
  }

  bool _isRowFull(int row) {
    for (var c = 0; c < size; c++) {
      if (_cells[_index(row, c)] == emptyCell) return false;
    }
    return true;
  }

  bool _isColFull(int col) {
    for (var r = 0; r < size; r++) {
      if (_cells[_index(r, col)] == emptyCell) return false;
    }
    return true;
  }

  void clear() {
    _cells.fillRange(0, _cells.length, emptyCell);
  }
}
