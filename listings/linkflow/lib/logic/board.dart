/// 棋盘上的一格坐标。值语义 —— 直接进 Set / Map 当键。
class Cell {
  const Cell(this.row, this.col);

  final int row;
  final int col;

  /// 上下左右四邻。不做边界裁剪，调用方自己判断是否越界。
  List<Cell> get neighbours => <Cell>[
        Cell(row - 1, col),
        Cell(row + 1, col),
        Cell(row, col - 1),
        Cell(row, col + 1),
      ];

  /// 是否与 [other] 正交相邻（不含对角）。连线只能走直角，不能斜着走。
  bool isAdjacentTo(Cell other) =>
      (row - other.row).abs() + (col - other.col).abs() == 1;

  @override
  bool operator ==(Object other) =>
      other is Cell && other.row == row && other.col == col;

  @override
  int get hashCode => Object.hash(row, col);

  @override
  String toString() => '($row,$col)';
}
