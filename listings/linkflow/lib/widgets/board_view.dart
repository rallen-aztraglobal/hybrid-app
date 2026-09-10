import 'package:flutter/material.dart';

import '../logic/board.dart';
import '../logic/game.dart';
import '../theme/app_colors.dart';

/// 棋盘：网格 + 端点圆点 + 玩家画的线。整块用 CustomPaint 画。
///
/// 为什么不用 widget 树：最高难度是 9×9 共 81 格，加上线段会变成上百个 widget，
/// 而拖动时每帧都要重建。直接画一层跟手得多。
///
/// 命中判定与绘制共用同一套几何量（[_Geometry]），
/// 所以「看到的格子」和「拖到的格子」不会错位。
class BoardView extends StatefulWidget {
  const BoardView({
    super.key,
    required this.game,
    required this.onChanged,
  });

  final FlowGame game;

  /// 路径有任何变化后回调，上层据此 setState 并检查是否通关。
  final VoidCallback onChanged;

  @override
  State<BoardView> createState() => _BoardViewState();
}

class _BoardViewState extends State<BoardView> {
  void _down(Offset local, Size size) {
    final geo = _Geometry(side: widget.game.puzzle.side, available: size.shortestSide);
    final cell = geo.cellAt(local);
    if (cell == null) return;
    if (widget.game.beginDrag(cell)) widget.onChanged();
  }

  void _move(Offset local, Size size) {
    final geo = _Geometry(side: widget.game.puzzle.side, available: size.shortestSide);
    final cell = geo.cellAt(local);
    if (cell == null) return;
    if (widget.game.dragTo(cell)) widget.onChanged();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final side = constraints.biggest.shortestSide;
        final size = Size(side, side);
        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onPanStart: (d) => _down(d.localPosition, size),
          onPanUpdate: (d) => _move(d.localPosition, size),
          onPanEnd: (_) {
            widget.game.endDrag();
            widget.onChanged();
          },
          child: CustomPaint(
            size: size,
            painter: _BoardPainter(game: widget.game),
          ),
        );
      },
    );
  }
}

/// 棋盘几何。绘制与命中判定共用。
class _Geometry {
  _Geometry({required this.side, required this.available}) {
    pad = available * 0.045;
    cell = (available - 2 * pad) / side;
  }

  final int side;
  final double available;
  late final double pad;
  late final double cell;

  Offset centerOf(Cell c) => Offset(
        pad + (c.col + 0.5) * cell,
        pad + (c.row + 0.5) * cell,
      );

  Rect rectOf(Cell c) =>
      Rect.fromLTWH(pad + c.col * cell, pad + c.row * cell, cell, cell);

  /// 把手指位置换算成格子坐标。落在棋盘外返回 null。
  Cell? cellAt(Offset p) {
    final col = ((p.dx - pad) / cell).floor();
    final row = ((p.dy - pad) / cell).floor();
    if (row < 0 || row >= side || col < 0 || col >= side) return null;
    return Cell(row, col);
  }
}

class _BoardPainter extends CustomPainter {
  _BoardPainter({required this.game});

  final FlowGame game;

  @override
  void paint(Canvas canvas, Size size) {
    final geo = _Geometry(side: game.puzzle.side, available: size.shortestSide);

    _paintSurface(canvas, size);
    _paintCells(canvas, geo);
    _paintPaths(canvas, geo);
    _paintEndpoints(canvas, geo);
  }

  void _paintSurface(Canvas canvas, Size size) {
    final rr = RRect.fromRectAndRadius(Offset.zero & size, const Radius.circular(22));
    canvas.drawRRect(rr, Paint()..color = AppColors.boardSurface);
    canvas.drawRRect(
      rr,
      Paint()
        ..color = AppColors.boardBorder
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.4,
    );
  }

  void _paintCells(Canvas canvas, _Geometry geo) {
    final paint = Paint()..color = AppColors.emptyCell;
    final gap = geo.cell * 0.06;
    final radius = Radius.circular((geo.cell * 0.16).clamp(3.0, 10.0));
    for (var r = 0; r < geo.side; r++) {
      for (var c = 0; c < geo.side; c++) {
        canvas.drawRRect(
          RRect.fromRectAndRadius(geo.rectOf(Cell(r, c)).deflate(gap), radius),
          paint,
        );
      }
    }
  }

  /// 画玩家已经拉出来的线。
  ///
  /// 用一条带圆角连接的粗折线，而不是逐格填色块 —— 连线游戏的手感有一半在
  /// 「这是一根连续的管子」这个观感上，填色块看起来是散的。
  void _paintPaths(Canvas canvas, _Geometry geo) {
    final width = geo.cell * 0.42;

    for (var i = 0; i < game.puzzle.colorKeys.length; i++) {
      final key = game.puzzle.colorKeys[i];
      final path = game.pathOf(key);
      if (path.isEmpty) continue;

      final color = AppColors.path(i);
      final paint = Paint()
        ..color = color
        ..strokeWidth = width
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round
        ..style = PaintingStyle.stroke;

      if (path.length == 1) {
        // 只按下还没拖：画个小点表示「这条已经起手了」
        canvas.drawCircle(geo.centerOf(path.first), width * 0.5, Paint()..color = color);
        continue;
      }

      final poly = Path()..moveTo(geo.centerOf(path.first).dx, geo.centerOf(path.first).dy);
      for (var j = 1; j < path.length; j++) {
        final p = geo.centerOf(path[j]);
        poly.lineTo(p.dx, p.dy);
      }
      canvas.drawPath(poly, paint);
    }
  }

  /// 画端点。已连通的端点加一圈白边，一眼看出哪几条已经完成。
  void _paintEndpoints(Canvas canvas, _Geometry geo) {
    final radius = geo.cell * 0.3;

    for (var i = 0; i < game.puzzle.colorKeys.length; i++) {
      final key = game.puzzle.colorKeys[i];
      final color = AppColors.path(i);
      final connected = game.isConnected(key);

      for (final end in game.puzzle.endpoints[key]!) {
        final center = geo.centerOf(end);
        canvas.drawCircle(center, radius, Paint()..color = color);
        if (connected) {
          canvas.drawCircle(
            center,
            radius * 0.45,
            Paint()..color = Colors.white,
          );
        }
      }
    }
  }

  @override
  bool shouldRepaint(covariant _BoardPainter old) => true;
}
