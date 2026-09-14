/// EdgeLoop 的题面模型与「边」的编号规则。
///
/// 棋盘是 rows×cols 个格子，格子的四角构成 (rows+1)×(cols+1) 个**点**。
/// 玩家画的线段连接相邻两点，称为**边**。边分两类：
///
///   横边 H(r,c)：连 点(r,c) 与 点(r,c+1)     r∈[0,rows]   c∈[0,cols)
///   竖边 V(r,c)：连 点(r,c) 与 点(r+1,c)     r∈[0,rows)   c∈[0,cols]
///
/// 全部边压进一个一维数组，横边在前、竖边在后 —— 求解器要按固定顺序逐边试，
/// 一维下标比二维快得多，而且顺序本身就是剪枝效率的关键（见 solver.dart）。
///
/// 格子 (r,c) 的四条边：上 H(r,c) / 下 H(r+1,c) / 左 V(r,c) / 右 V(r,c+1)。
/// 点 (r,c) 关联的边最多四条：左 H(r,c-1) / 右 H(r,c) / 上 V(r-1,c) / 下 V(r,c)。
class EdgeLoopPuzzle {
  EdgeLoopPuzzle({
    required this.rows,
    required this.cols,
    required this.clues,
  }) : assert(clues.length == rows * cols);

  final int rows;
  final int cols;

  /// 每格的提示数，行优先。`null` = 该格无提示。取值 0..3。
  ///
  /// 不会出现 4：四条边全画上会让这个格子被一圈线包住，而那一圈自己就是一个闭合回路，
  /// 于是整盘至少有两个回路（除非棋盘只有一格），与「有且仅有一条回路」矛盾。
  /// 生成器因此永远产不出 4，但求解器不依赖这一点（它按通用规则算）。
  final List<int?> clues;

  int get cellCount => rows * cols;

  /// 横边总数。竖边下标从这里开始。
  int get hCount => (rows + 1) * cols;

  /// 竖边总数。
  int get vCount => rows * (cols + 1);

  /// 边总数。
  int get edgeCount => hCount + vCount;

  int clueAt(int r, int c) => clues[r * cols + c] ?? -1;

  /// 横边 H(r,c) 的一维下标。
  int hIndex(int r, int c) => r * cols + c;

  /// 竖边 V(r,c) 的一维下标。
  int vIndex(int r, int c) => hCount + r * (cols + 1) + c;

  /// 格子 (r,c) 的四条边下标：上、下、左、右。
  List<int> cellEdges(int r, int c) => <int>[
        hIndex(r, c),
        hIndex(r + 1, c),
        vIndex(r, c),
        vIndex(r, c + 1),
      ];

  /// 点 (r,c) 关联的边下标（边界上的点不足四条）。
  List<int> dotEdges(int r, int c) {
    final out = <int>[];
    if (c > 0) out.add(hIndex(r, c - 1));
    if (c < cols) out.add(hIndex(r, c));
    if (r > 0) out.add(vIndex(r - 1, c));
    if (r < rows) out.add(vIndex(r, c));
    return out;
  }

  /// 提示数个数 —— 越少一般越难，用于难度分档与生成器的收敛判断。
  int get clueCount => clues.where((c) => c != null).length;

  /// 紧凑串行化，给 puzzle_library.dart 的代码生成用。
  /// 形如 `5x5:..3.2.....1.......2.3..`，`.` 表示无提示。
  String encode() {
    final sb = StringBuffer('${rows}x$cols:');
    for (final c in clues) {
      sb.write(c?.toString() ?? '.');
    }
    return sb.toString();
  }

  static EdgeLoopPuzzle decode(String s) {
    final colon = s.indexOf(':');
    final dims = s.substring(0, colon).split('x');
    final rows = int.parse(dims[0]);
    final cols = int.parse(dims[1]);
    final body = s.substring(colon + 1);
    if (body.length != rows * cols) {
      throw FormatException('题面长度 ${body.length} 与 ${rows}x$cols 不符：$s');
    }
    return EdgeLoopPuzzle(
      rows: rows,
      cols: cols,
      clues: body
          .split('')
          .map((ch) => ch == '.' ? null : int.parse(ch))
          .toList(growable: false),
    );
  }
}
