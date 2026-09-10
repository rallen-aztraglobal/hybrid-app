import 'board.dart';

/// 一条通路的答案：从 [start] 出发，按 [moves] 一步步走。
///
/// [moves] 是方向串，`U`/`D`/`L`/`R` 四个字符。这样写谜题最紧凑，
/// 而且不会出现「坐标抄错一个数」这类难查的错。
///
/// **为什么不用「同字母格子」的字符表推端点。** 试过，行不通：
/// 蛇形路径贴着自己走时（右→下→左），相邻的两段在棋盘上是邻接的，
/// 于是中间格子会数出 3 个同色邻居、端点数不出来。显式给出走法就没有这个歧义。
class FlowPath {
  const FlowPath(this.key, {required this.start, required this.moves});

  /// 通路标识。配色按它在 [FlowPuzzle.colorKeys] 里的下标取。
  final String key;
  final Cell start;
  final String moves;

  /// 展开成格子序列。
  List<Cell> get cells {
    final out = <Cell>[start];
    var cursor = start;
    for (final ch in moves.split('')) {
      cursor = switch (ch) {
        'U' => Cell(cursor.row - 1, cursor.col),
        'D' => Cell(cursor.row + 1, cursor.col),
        'L' => Cell(cursor.row, cursor.col - 1),
        'R' => Cell(cursor.row, cursor.col + 1),
        _ => throw ArgumentError('通路 $key 的走法里有非法字符 "$ch"'),
      };
      out.add(cursor);
    }
    return out;
  }
}

/// 一道连线题。答案由若干条 [FlowPath] 给出，必须**不重不漏地铺满整盘**。
class FlowPuzzle {
  FlowPuzzle._({
    required this.side,
    required this.paths,
    required this.colorKeys,
  });

  factory FlowPuzzle(int side, List<FlowPath> definitions) {
    final paths = <String, List<Cell>>{
      for (final d in definitions) d.key: d.cells,
    };
    return FlowPuzzle._(
      side: side,
      paths: paths,
      colorKeys: definitions.map((d) => d.key).toList(),
    );
  }

  final int side;

  /// 每条通路的答案格子序列（首尾即两个端点）。
  final Map<String, List<Cell>> paths;

  /// 通路的键，按定义顺序 —— 配色按下标取，保证同一题每次颜色一致。
  final List<String> colorKeys;

  /// 每条通路的两个端点。
  Map<String, List<Cell>> get endpoints => <String, List<Cell>>{
        for (final e in paths.entries) e.key: <Cell>[e.value.first, e.value.last],
      };

  bool contains(Cell c) =>
      c.row >= 0 && c.row < side && c.col >= 0 && c.col < side;

  /// 这一格是不是某条通路的端点；是则返回该通路的键。
  String? endpointKeyAt(Cell c) {
    for (final e in paths.entries) {
      if (e.value.first == c || e.value.last == c) return e.key;
    }
    return null;
  }

  bool isEndpoint(Cell c) => endpointKeyAt(c) != null;

  /// 校验这道题是否合法。返回问题清单，空表示没问题。
  ///
  /// 手写谜题最容易犯的错：走法串写多写少导致盖不满或者盖重、走出棋盘、
  /// 或者一条路径自己撞自己。这些错在界面上表现得很隐晦 ——
  /// 玩家会发现某条线怎么连都连不完 —— 所以用测试在入库前拦掉。
  List<String> validate() {
    final problems = <String>[];
    final seen = <Cell, String>{};

    for (final entry in paths.entries) {
      final key = entry.key;
      final cells = entry.value;

      if (cells.length < 2) {
        problems.add('通路 $key 只有 ${cells.length} 格，至少要 2 格');
      }

      final own = <Cell>{};
      for (var i = 0; i < cells.length; i++) {
        final c = cells[i];
        if (!contains(c)) {
          problems.add('通路 $key 的第 $i 格 $c 越出了 $side×$side 棋盘');
          continue;
        }
        if (!own.add(c)) {
          problems.add('通路 $key 自己经过 $c 两次');
        }
        final other = seen[c];
        if (other != null && other != key) {
          problems.add('$c 同时被通路 $other 和 $key 占用');
        }
        seen[c] = key;
        if (i > 0 && !c.isAdjacentTo(cells[i - 1])) {
          problems.add('通路 $key 在第 $i 步跳格了：${cells[i - 1]} → $c');
        }
      }
    }

    final total = side * side;
    if (seen.length != total) {
      problems.add('答案只盖住了 ${seen.length} / $total 格，连线题要求铺满');
    }

    return problems;
  }
}
