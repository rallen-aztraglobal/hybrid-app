import 'puzzle.dart';

/// 棋盘几何 + 「手指点在哪 → 点的是哪条边」。
///
/// 从界面里抽出来是因为这段逻辑是这个游戏在手机上**最容易毁掉体验**的地方，
/// 必须能被测试钉住：边只有两三像素宽，判定一旦有死区，玩家点下去没反应，
/// 会直接以为 App 坏了。纯函数化之后，「棋盘内任何一点都能命中某条边」
/// 这句话就是一条可以跑的断言，而不是一句自我感觉。
class BoardGeometry {
  const BoardGeometry({
    required this.cell,
    required this.originX,
    required this.originY,
  });

  final double cell;
  final double originX;
  final double originY;

  /// 棋盘四周留出的边距，单位是「格」。
  ///
  /// 下限由绘制决定：底板圆角矩形比格点阵外扩 0.35 格，最外圈的线再占半个笔宽
  /// （0.05 格）。取 0.45 留一点富余，同时让棋盘尽量占满屏宽 ——
  /// 早先取 0.6，棋盘只用到约 80% 的宽度，在手机上显得小。
  static const double pad = 0.45;

  /// 在给定画布尺寸里放下 rows×cols 的棋盘，取最大的正方格并居中。
  factory BoardGeometry.fit(double width, double height, int rows, int cols) {
    final byWidth = width / (cols + pad * 2);
    final byHeight = height / (rows + pad * 2);
    final cell = byWidth < byHeight ? byWidth : byHeight;
    return BoardGeometry(
      cell: cell,
      originX: (width - cols * cell) / 2,
      originY: (height - rows * cell) / 2,
    );
  }

  /// 点 (r,c) 的画布坐标。
  (double, double) dot(int r, int c) =>
      (originX + c * cell, originY + r * cell);
}

/// 触点离目标边超过这么多「格」就不算命中。
///
/// **必须正好是 0.5，不能更小。** 棋盘内任意一点到最近横边、最近竖边的距离都不超过
/// 0.5 格，而这个上界恰恰在**格子正中心**取到（那里两个方向的距离同时等于 0.5）。
/// 所以阈值一旦小于 0.5，每个格子的正中心就成了死区 —— 手指点在格子中间毫无反应。
///
/// 这不是推演出来的，是 test/logic/hit_test_test.dart 里的密集采样跑出来的：
/// 早先图省事取了 0.45，测试立刻在格子中心报错。
///
/// 取 0.5 也不会「太贪」：越界由 hr/hc/vr/vc 的范围检查挡住，
/// 可响应区域正好是棋盘向外扩半格，刚好落在棋盘 0.6 格的留白之内。
const double kMaxPickDistance = 0.5;

/// 把画布坐标翻译成边的下标，命中不了返回 null。
///
/// 判定**不看线实际画在哪**，而是按格子的小数坐标算最近的横边与竖边，取近的那条。
/// 早先试过「到边中点距离小于阈值」，结果格子正中心那一带谁也够不着 ——
/// 点下去毫无反应，比点错更让人困惑。
int? edgeAt(EdgeLoopPuzzle p, BoardGeometry g, double x, double y) {
  final fx = (x - g.originX) / g.cell;
  final fy = (y - g.originY) / g.cell;

  // 最近的横边：在点行 round(fy) 上，占第 floor(fx) 列
  final hr = fy.round();
  final hc = _spanIndex(fx, p.cols);
  final hDist = (fy - hr).abs();
  final hValid = hr >= 0 && hr <= p.rows && hc != null;

  // 最近的竖边：在点列 round(fx) 上，占第 floor(fy) 行
  final vc = fx.round();
  final vr = _spanIndex(fy, p.rows);
  final vDist = (fx - vc).abs();
  final vValid = vc >= 0 && vc <= p.cols && vr != null;

  if (hValid && (!vValid || hDist <= vDist)) {
    return hDist <= kMaxPickDistance ? p.hIndex(hr, hc) : null;
  }
  if (vValid) {
    return vDist <= kMaxPickDistance ? p.vIndex(vr, vc) : null;
  }
  return null;
}

/// 边沿着哪一「格」延伸 —— 把小数坐标 [f] 换成 0..[count]-1 的格下标，超出范围返回 null。
///
/// 直接 `f.floor()` 在**正好落在棋盘最右/最下边界**时会取到 count（越界），
/// 于是横竖两个候选同时判无效，那个点就成了点不中的缺口。
/// 棋盘右下角那个点恰好如此：fx、fy 同时等于 count。
///
/// 这个缺口只有一个点那么大，但它不是理论问题 —— 改了 [BoardGeometry.pad] 之后
/// 采样点的浮点运算正好落在上面，被测试当场抓出来。
///
/// 做法是在边界外半格之内把下标夹回来，与 [kMaxPickDistance] 的半格口径一致：
/// 刚出界一点仍算命中最外圈那一格，出界超过半格才真的不响应。
int? _spanIndex(double f, int count) {
  final i = f.floor();
  if (i >= 0 && i < count) return i;
  if (i >= count && f - count <= kMaxPickDistance) return count - 1;
  if (i < 0 && -f <= kMaxPickDistance) return 0;
  return null;
}
