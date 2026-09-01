import 'package:flutter/foundation.dart';

/// 棋盘上的一个格坐标。行自上而下、列自左而右，均从 0 起。
@immutable
class Cell {
  const Cell(this.row, this.col);

  final int row;
  final int col;

  Cell shifted(int dRow, int dCol) => Cell(row + dRow, col + dCol);

  @override
  bool operator ==(Object other) =>
      other is Cell && other.row == row && other.col == col;

  @override
  int get hashCode => Object.hash(row, col);

  @override
  String toString() => '($row,$col)';
}

/// 一个可拖放的方块（polyomino）。
///
/// [cells] 已归一化到左上角：最小行与最小列都是 0。放置时把 [cells] 整体平移到目标
/// 落点即可，无需再做对齐计算。形状固定、**不支持旋转** —— 这是该玩法的惯例：
/// 能否放下完全由发到的形状决定，玩家的决策集合小而清晰。
@immutable
class Piece {
  const Piece({
    required this.id,
    required this.cells,
    required this.colorIndex,
    required this.weight,
  });

  /// 形状的稳定标识，用于测试与调试（不面向玩家）。
  final String id;

  /// 归一化后的格坐标集合。
  final List<Cell> cells;

  /// 配色索引，对应 AppColors.pieceColors 的下标。
  final int colorIndex;

  /// 发牌权重。越大越常出现；大而难放的形状（5 连、3×3）压到最低。
  final int weight;

  int get size => cells.length;

  int get height {
    var maxRow = 0;
    for (final c in cells) {
      if (c.row > maxRow) maxRow = c.row;
    }
    return maxRow + 1;
  }

  int get width {
    var maxCol = 0;
    for (final c in cells) {
      if (c.col > maxCol) maxCol = c.col;
    }
    return maxCol + 1;
  }

  @override
  String toString() => 'Piece($id)';
}

/// 用 ASCII 图案声明形状：`#` 是实心格，其余字符是空格。
///
/// 比手写坐标列表可读得多 —— 形状本身在源码里就是它在屏幕上的样子，
/// 增删形状时不必在脑内做坐标演算。
Piece _pattern(String id, int colorIndex, int weight, List<String> rows) {
  final cells = <Cell>[];
  for (var r = 0; r < rows.length; r++) {
    final line = rows[r];
    for (var c = 0; c < line.length; c++) {
      if (line[c] == '#') cells.add(Cell(r, c));
    }
  }
  assert(cells.isNotEmpty, '形状 $id 是空的');
  assert(
    cells.any((c) => c.row == 0) && cells.any((c) => c.col == 0),
    '形状 $id 未归一化到左上角',
  );
  return Piece(id: id, cells: cells, colorIndex: colorIndex, weight: weight);
}

/// 全部可发的形状。
///
/// 权重取值的依据：单格与 2/3 连最百搭，权重最高；5 连与 3×3 占地大、后期几乎必卡死，
/// 压到 1。整体分布决定了这个游戏的手感 —— 调难度改这里，不必动 Game 的逻辑。
final List<Piece> kPieceCatalog = List<Piece>.unmodifiable(<Piece>[
  _pattern('dot', 0, 3, <String>['#']),

  _pattern('h2', 1, 4, <String>['##']),
  _pattern('v2', 1, 4, <String>['#', '#']),

  _pattern('h3', 2, 4, <String>['###']),
  _pattern('v3', 2, 4, <String>['#', '#', '#']),

  _pattern('h4', 3, 3, <String>['####']),
  _pattern('v4', 3, 3, <String>['#', '#', '#', '#']),

  _pattern('h5', 4, 1, <String>['#####']),
  _pattern('v5', 4, 1, <String>['#', '#', '#', '#', '#']),

  _pattern('sq2', 5, 4, <String>['##', '##']),
  _pattern('sq3', 6, 1, <String>['###', '###', '###']),

  // 2×2 的四个直角三格块。
  _pattern('el3a', 7, 3, <String>['#.', '##']),
  _pattern('el3b', 7, 3, <String>['.#', '##']),
  _pattern('el3c', 7, 3, <String>['##', '#.']),
  _pattern('el3d', 7, 3, <String>['##', '.#']),

  // 3×3 的四个直角五格块。
  _pattern('el5a', 8, 2, <String>['#..', '#..', '###']),
  _pattern('el5b', 8, 2, <String>['..#', '..#', '###']),
  _pattern('el5c', 8, 2, <String>['###', '#..', '#..']),
  _pattern('el5d', 8, 2, <String>['###', '..#', '..#']),
]);
