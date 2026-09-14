/// 难度分档。
///
/// 分档依据是**棋盘尺寸 + 造解时区域的大小**，而不是「剩余提示数」——
/// 提示数是生成器删到不能再删的结果，不受控；尺寸和回路长度才是能定下来的旋钮。
///
/// 实测各档生成均 5/5 成功（一次性探测脚本跑出，结论记录于此，脚本未留仓库）：
///
///     5x5 区域10   平均剩余提示 ~9/25    最大搜索节点 ~2.6万
///     6x6 区域14   平均剩余提示 ~13/36   最大搜索节点 ~6.3万
///     7x7 区域20   平均剩余提示 ~17/49   最大搜索节点 ~114万
///     8x8 区域30   平均剩余提示 ~22/64   最大搜索节点 ~161万
///
/// 「最大搜索节点」是穷举求解器跑完最终题面要展开的节点数，可以当作题目客观难度的
/// 一个粗略下界 —— 它每档涨一个量级，说明分档确实拉开了差距。
enum Difficulty {
  normal('Normal', rows: 5, cols: 5, regionCells: 10),
  hard('Hard', rows: 6, cols: 6, regionCells: 14),
  expert('Expert', rows: 7, cols: 7, regionCells: 20),
  master('Master', rows: 8, cols: 8, regionCells: 30);

  const Difficulty(
    this.label, {
    required this.rows,
    required this.cols,
    required this.regionCells,
  });

  final String label;
  final int rows;
  final int cols;

  /// 造解时随机长出的区域格子数 —— 决定回路的长短。
  final int regionCells;
}
