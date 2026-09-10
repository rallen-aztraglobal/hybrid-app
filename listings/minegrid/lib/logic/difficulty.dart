/// 难度档。
///
/// 盘面按手机竖屏比例设计（列少行多），不照搬桌面扫雷的 30×16 —— 那个宽高比
/// 在手机上要么横屏、要么格子小到点不准。
///
/// 雷密度是这个游戏真正的难度旋钮：
/// 12.5% → 大片自动展开，基本是白送；25% 附近几乎每一步都要算。
/// 桌面版三档是 12.3% / 15.6% / 20.6%，这里最高档比它还高一点。
enum Difficulty {
  normal('Normal', cols: 8, rows: 10, mines: 10),
  hard('Hard', cols: 9, rows: 12, mines: 20),
  expert('Expert', cols: 10, rows: 14, mines: 30),
  master('Master', cols: 11, rows: 16, mines: 42);

  const Difficulty(
    this.label, {
    required this.cols,
    required this.rows,
    required this.mines,
  });

  final String label;
  final int cols;
  final int rows;
  final int mines;

  int get cellCount => cols * rows;

  /// 雷占全盘的比例，仅用于展示与测试断言。
  double get density => mines / cellCount;
}
