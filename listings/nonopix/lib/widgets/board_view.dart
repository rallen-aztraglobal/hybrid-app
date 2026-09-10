import 'package:flutter/material.dart';

import '../logic/game.dart';
import '../logic/picture_library.dart';
import '../theme/app_colors.dart';

/// 棋盘 + 行列线索，整块用 CustomPaint 画。
///
/// 为什么不用 GridView：10×10 是 100 个格子，加上两侧线索槽会变成一百多个 widget，
/// 而每次落笔都要重建。直接画一层省掉整棵子树，拖动涂格时明显更跟手。
///
/// 命中判定自己算：手指位置减去线索槽偏移再除以格宽。与绘制共用同一套几何量
/// （[_Geometry]），所以「看到的格子」和「点到的格子」不会错位。
class BoardView extends StatefulWidget {
  const BoardView({
    super.key,
    required this.game,
    required this.picture,
    required this.intent,
    required this.onChanged,
    required this.onMistake,
  });

  final NonogramGame game;


  /// 当前这张画。已涂格按它的配色上色 —— 涂开之后浮现的是一幅彩图，

  /// 而不是一片同色方块。规则不受影响：线索仍是单色的。

  final Picture picture;
  /// 当前笔的含义：涂黑还是打叉。由上层的模式开关决定。
  final CellMark intent;

  /// 任何一格发生变化后回调，上层据此 setState 并检查是否通关。
  final VoidCallback onChanged;

  /// 涂错时回调，上层用来震动提示。
  final VoidCallback onMistake;

  @override
  State<BoardView> createState() => _BoardViewState();
}

class _BoardViewState extends State<BoardView> {
  /// 本次拖动已经处理过的格子。数织习惯是「按住划一串」，
  /// 不记下来的话手指在同一格里抖一下就会反复切换标记。
  final Set<int> _touchedThisGesture = <int>{};

  void _handleAt(Offset local, Size size) {
    final geo = _Geometry(puzzleGame: widget.game, available: size.shortestSide);
    final cell = geo.cellAt(local);
    if (cell == null) return;

    final key = cell.$1 * widget.game.puzzle.cols + cell.$2;
    if (!_touchedThisGesture.add(key)) return; // 这一笔里已经处理过

    final wrong = widget.game.apply(cell.$1, cell.$2, widget.intent);
    widget.onChanged();
    if (wrong) widget.onMistake();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final side = constraints.biggest.shortestSide;
        final size = Size(side, side);
        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onPanStart: (d) {
            _touchedThisGesture.clear();
            _handleAt(d.localPosition, size);
          },
          onPanUpdate: (d) => _handleAt(d.localPosition, size),
          onPanEnd: (_) => _touchedThisGesture.clear(),
          onTapUp: (d) {
            _touchedThisGesture.clear();
            _handleAt(d.localPosition, size);
          },
          child: CustomPaint(
            size: size,
            painter: _BoardPainter(game: widget.game, picture: widget.picture),
          ),
        );
      },
    );
  }
}

/// 棋盘的几何计算。绘制与命中判定共用，保证两者一致。
class _Geometry {
  _Geometry({required NonogramGame puzzleGame, required this.available})
      : side = puzzleGame.puzzle.rows {
    // 本题任意一行/列最多有几段线索 —— 槽宽由它决定。
    var maxClues = 1;
    for (final c in puzzleGame.puzzle.rowClues) {
      if (c.length > maxClues) maxClues = c.length;
    }
    for (final c in puzzleGame.puzzle.colClues) {
      if (c.length > maxClues) maxClues = c.length;
    }

    pad = available * 0.035;

    // 每段线索占的宽度定成格宽的固定比例，槽宽 = 段数 × 这个比例。
    //
    // 之前是反过来的：先按经验给槽一个百分比，再用「槽宽 ÷ maxClues」平分。
    // 那样段数少的线索会被摊开在整条槽里，两个数字之间空一大截；
    // 而且不同行的间距还不一致，看着散。现在间距固定、每组线索紧凑地
    // 贴着网格排，行与行之间也对得齐。
    //
    // 解一元一次方程得到格宽：
    //   available = 2·pad + maxClues·(k·cell) + side·cell
    const k = 0.56;
    cell = (available - 2 * pad) / (side + k * maxClues);
    clueStep = cell * k;
    gutter = pad + clueStep * maxClues;
    board = cell * side;
  }

  final int side;
  final double available;

  /// 卡片四周的内边距。
  late final double pad;

  /// 从卡片左（上）边缘到网格起点的距离 —— 含内边距与线索槽。
  late final double gutter;
  late final double board;
  late final double cell;

  /// 每段线索占的宽度。用槽宽平分，线索自然贴着网格排。
  late final double clueStep;

  Rect cellRect(int row, int col) => Rect.fromLTWH(
        gutter + col * cell,
        gutter + row * cell,
        cell,
        cell,
      );

  /// 把手指位置换算成格子坐标。落在线索槽或棋盘外返回 null。
  (int, int)? cellAt(Offset p) {
    final x = p.dx - gutter;
    final y = p.dy - gutter;
    if (x < 0 || y < 0) return null;
    final col = (x / cell).floor();
    final row = (y / cell).floor();
    if (row < 0 || row >= side || col < 0 || col >= side) return null;
    return (row, col);
  }
}

class _BoardPainter extends CustomPainter {
  _BoardPainter({required this.game, required this.picture});

  final NonogramGame game;

  /// 当前这张画。已涂格按它的配色上色 —— 涂开之后浮现的是一幅彩图，
  /// 而不是一片同色方块。规则不受影响：线索仍是单色的。
  final Picture picture;

  @override
  void paint(Canvas canvas, Size size) {
    final geo = _Geometry(puzzleGame: game, available: size.shortestSide);

    _paintBoardSurface(canvas, size);
    _paintCells(canvas, geo);
    _paintGrid(canvas, geo);
    _paintClues(canvas, geo);
  }

  /// 底板铺满整个方形 —— 连线索槽一起罩进去。
  ///
  /// 之前只在网格那一块画底板，结果线索浮在卡片外面的深色背景上，
  /// 读起来像两块不相干的东西。线索本来就是谜题的一部分，
  /// 让它们和网格同处一张卡片，整体才立得住。
  ///
  /// 底板走一道极淡的竖向渐变 + 一圈顶部高光描边。单一纯色的卡片在
  /// 深色 App 里会显得像一张贴纸，加一点点厚度感差别很大。
  void _paintBoardSurface(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final rr = RRect.fromRectAndRadius(rect, const Radius.circular(22));

    canvas.drawRRect(rr, Paint()..color = AppColors.boardSurface);
    // 浅色主题里用描边分层，比阴影干净
    canvas.drawRRect(
      rr,
      Paint()
        ..color = AppColors.boardBorder
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.4,
    );
  }

  void _paintCells(Canvas canvas, _Geometry geo) {
    final empty = Paint()..color = AppColors.emptyCell;
    final cross = Paint()
      ..color = AppColors.crossMark
      ..strokeWidth = (geo.cell * 0.09).clamp(1.6, 3.4)
      ..strokeCap = StrokeCap.round;

    // 格子之间留一条细缝，底板的颜色透出来就是网格线 ——
    // 比额外画线干净，也不会在缩放时出现半像素毛边。
    final gap = geo.cell * 0.05;

    // 圆角随格子大小收缩。10×10 时格子只有 5×5 的一半，
    // 沿用同一个比例会让小格子看起来像一堆散落的药丸而不是一张网格。
    final radius = Radius.circular((geo.cell * 0.18).clamp(2.5, 12.0));

    for (var r = 0; r < geo.side; r++) {
      for (var c = 0; c < geo.side; c++) {
        final rect = geo.cellRect(r, c).deflate(gap);
        final rr = RRect.fromRectAndRadius(rect, radius);

        switch (game.markAt(r, c)) {
          case CellMark.filled:
            // 颜色取自这张画在该格的本色 —— 树是绿的、树干是棕的，
            // 解开之后浮现的是一幅彩图而不是一片同色方块。
            // 单色图不给 palette，会回落到主题主色。
            final base = Color(
              picture.colorAt(r, c, AppColors.filledCell.toARGB32()),
            );
            // 竖向渐变：顶部提亮一点点，落笔那一下有「按下去一块实体」的感觉。
            canvas.drawRRect(
              rr,
              Paint()
                ..shader = LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: <Color>[
                    Color.lerp(base, Colors.white, 0.18)!,
                    base,
                  ],
                ).createShader(rect),
            );
          case CellMark.crossed:
            canvas.drawRRect(rr, empty);
            final inset = rect.deflate(rect.width * 0.3);
            canvas.drawLine(inset.topLeft, inset.bottomRight, cross);
            canvas.drawLine(inset.topRight, inset.bottomLeft, cross);
          case CellMark.blank:
            canvas.drawRRect(rr, empty);
        }
      }
    }
  }

  /// 每 5 格一条粗线。数织靠它数格子，10×10 没有这个基本没法玩。
  /// 画在格子之上，用半透明，压过缝隙但不遮住已涂的格。
  void _paintGrid(Canvas canvas, _Geometry geo) {
    if (geo.side <= 5) return; // 5×5 不需要，画了反而碎

    final major = Paint()
      ..color = AppColors.majorGrid
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round;

    for (var i = 5; i < geo.side; i += 5) {
      final x = geo.gutter + i * geo.cell;
      canvas.drawLine(
        Offset(x, geo.gutter + geo.cell * 0.1),
        Offset(x, geo.gutter + geo.board - geo.cell * 0.1),
        major,
      );
      final y = geo.gutter + i * geo.cell;
      canvas.drawLine(
        Offset(geo.gutter + geo.cell * 0.1, y),
        Offset(geo.gutter + geo.board - geo.cell * 0.1, y),
        major,
      );
    }
  }

  void _paintClues(Canvas canvas, _Geometry geo) {
    final puzzle = game.puzzle;
    // 字号跟着格宽走。下限提到 13 —— 之前是 9，8×8 时算出来才 11 出头，
    // 一屏几十个小数字看着非常累。宁可槽稍宽也要让线索读得轻松。
    final fontSize = (geo.cell * 0.46).clamp(13.0, 22.0);

    for (var r = 0; r < puzzle.rows; r++) {
      _paintClueLine(
        canvas: canvas,
        clues: puzzle.rowClues[r],
        done: game.isRowSatisfied(r),
        fontSize: fontSize,
        horizontal: true,
        edge: geo.gutter,
        cross: geo.gutter + r * geo.cell + geo.cell / 2,
        step: geo.clueStep,
        cellSize: geo.cell,
      );
    }

    for (var c = 0; c < puzzle.cols; c++) {
      _paintClueLine(
        canvas: canvas,
        clues: puzzle.colClues[c],
        done: game.isColSatisfied(c),
        fontSize: fontSize,
        horizontal: false,
        edge: geo.gutter,
        cross: geo.gutter + c * geo.cell + geo.cell / 2,
        step: geo.clueStep,
        cellSize: geo.cell,
      );
    }
  }

  /// 画一条线索。数字从贴着网格的一端往外排，所以最后一个线索离网格最近 ——
  /// 这与纸上数织的习惯一致，玩家的视线从网格边缘往外读。
  ///
  /// 每个数字带一个浅色底格。参考同类成熟产品的做法：数字直接浮在白底上时，
  /// 一屏几十个数字会糊成一片、分不清哪几个属于同一行；有了底格就自然分组，
  /// 而且行与行之间也对得齐。
  void _paintClueLine({
    required Canvas canvas,
    required List<int> clues,
    required bool done,
    required double fontSize,
    required bool horizontal,
    required double edge,
    required double cross,
    required double step,
    required double cellSize,
  }) {
    final color = done ? AppColors.clueDone : AppColors.primaryText;
    final slotPaint = Paint()
      ..color = done ? AppColors.clueSlotDone : AppColors.clueSlot;

    for (var i = 0; i < clues.length; i++) {
      // 先铺底格
      final slotFromEdgeBg = (clues.length - i) * step;
      final Rect slot;
      if (horizontal) {
        slot = Rect.fromLTWH(
          edge - slotFromEdgeBg,
          cross - cellSize / 2,
          step,
          cellSize,
        ).deflate(cellSize * 0.05);
      } else {
        slot = Rect.fromLTWH(
          cross - cellSize / 2,
          edge - slotFromEdgeBg,
          cellSize,
          step,
        ).deflate(cellSize * 0.05);
      }
      canvas.drawRRect(
        RRect.fromRectAndRadius(slot, Radius.circular(cellSize * 0.16)),
        slotPaint,
      );
    }

    for (var i = 0; i < clues.length; i++) {
      final tp = TextPainter(
        text: TextSpan(
          text: '${clues[i]}',
          style: TextStyle(
            color: color,
            fontSize: fontSize,
            fontWeight: FontWeight.w700,
            height: 1.0,
            fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout();

      // 距网格边缘的距离：最后一个线索占第一格
      final slotFromEdge = (clues.length - i) * step;
      final Offset topLeft;
      if (horizontal) {
        topLeft = Offset(
          edge - slotFromEdge + (step - tp.width) / 2,
          cross - tp.height / 2,
        );
      } else {
        topLeft = Offset(
          cross - tp.width / 2,
          edge - slotFromEdge + (step - tp.height) / 2,
        );
      }
      tp.paint(canvas, topLeft);
    }
  }

  @override
  bool shouldRepaint(covariant _BoardPainter old) => true;
}
