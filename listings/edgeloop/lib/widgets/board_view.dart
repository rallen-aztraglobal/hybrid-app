import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/game.dart';
import '../logic/hit_test.dart';
import '../logic/puzzle.dart';
import '../theme/app_colors.dart';

/// 棋盘。只负责画与转发触摸 —— 「点在哪 = 哪条边」的判定在 [edgeAt] 里，
/// 那段抽出去是为了能被测试穷举采样（见 test/logic/hit_test_test.dart）。
class BoardView extends StatelessWidget {
  const BoardView({
    super.key,
    required this.game,
    required this.onEdgeTapped,
    this.solved = false,
  });

  final EdgeLoopGame game;
  final ValueChanged<int> onEdgeTapped;
  final bool solved;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final geo = BoardGeometry.fit(
          constraints.biggest.width,
          constraints.biggest.height,
          game.puzzle.rows,
          game.puzzle.cols,
        );
        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTapUp: (details) {
            final edge = edgeAt(
              game.puzzle,
              geo,
              details.localPosition.dx,
              details.localPosition.dy,
            );
            if (edge == null) return;
            // 每次成功落子都给一次轻震：边很细，玩家需要一个「点到了」的确认，
            // 光靠看容易怀疑自己是不是点偏了。
            HapticFeedback.selectionClick();
            onEdgeTapped(edge);
          },
          child: CustomPaint(
            size: constraints.biggest,
            painter: _BoardPainter(game: game, geo: geo, solved: solved),
          ),
        );
      },
    );
  }
}

Offset _dotOffset(BoardGeometry g, int r, int c) {
  final (x, y) = g.dot(r, c);
  return Offset(x, y);
}

class _BoardPainter extends CustomPainter {
  _BoardPainter({
    required this.game,
    required this.geo,
    required this.solved,
  });

  final EdgeLoopGame game;
  final BoardGeometry geo;
  final bool solved;

  @override
  void paint(Canvas canvas, Size size) {
    final p = game.puzzle;
    final cell = geo.cell;

    final boardRect = Rect.fromLTWH(
      geo.originX - cell * 0.35,
      geo.originY - cell * 0.35,
      p.cols * cell + cell * 0.7,
      p.rows * cell + cell * 0.7,
    );
    final rr = RRect.fromRectAndRadius(boardRect, Radius.circular(cell * 0.22));
    canvas.drawRRect(rr, Paint()..color = AppColors.boardSurface);
    canvas.drawRRect(
      rr,
      Paint()
        ..color = AppColors.boardBorder
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.2,
    );

    // 画的顺序有讲究：数字在最下，其次格点，再是叉，线在最上。
    // 线是主角，任何时候都不该被别的元素压住。
    _paintClues(canvas, p, cell);
    _paintDots(canvas, p, cell);
    _paintCrosses(canvas, p, cell);
    _paintLines(canvas, p, cell);
  }

  void _paintClues(Canvas canvas, EdgeLoopPuzzle p, double cell) {
    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        final clue = p.clueAt(r, c);
        if (clue < 0) continue;

        final drawn = game.linesAroundCell(r, c);
        // 画超了标红；正好画满变淡（这格算完了）；否则正常墨色。
        final color = drawn > clue
            ? AppColors.clueViolated
            : (drawn == clue ? AppColors.clueSatisfied : AppColors.clueText);

        final tp = TextPainter(
          text: TextSpan(
            text: '$clue',
            style: TextStyle(
              color: color,
              fontSize: cell * 0.44,
              fontWeight: FontWeight.w600,
            ),
          ),
          textDirection: TextDirection.ltr,
        )..layout();

        final center = Offset(
          geo.originX + (c + 0.5) * cell,
          geo.originY + (r + 0.5) * cell,
        );
        tp.paint(canvas, center - Offset(tp.width / 2, tp.height / 2));
      }
    }
  }

  void _paintDots(Canvas canvas, EdgeLoopPuzzle p, double cell) {
    final paint = Paint()..color = AppColors.dot;
    final radius = cell * 0.055;
    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c <= p.cols; c++) {
        canvas.drawCircle(_dotOffset(geo, r, c), radius, paint);
      }
    }
  }

  void _paintCrosses(Canvas canvas, EdgeLoopPuzzle p, double cell) {
    final paint = Paint()
      ..color = AppColors.cross
      ..strokeWidth = cell * 0.045
      ..strokeCap = StrokeCap.round;
    final arm = cell * 0.10;

    void cross(Offset at) {
      canvas.drawLine(at + Offset(-arm, -arm), at + Offset(arm, arm), paint);
      canvas.drawLine(at + Offset(-arm, arm), at + Offset(arm, -arm), paint);
    }

    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        if (game.marks[p.hIndex(r, c)] == EdgeMark.cross) {
          cross(Offset(geo.originX + (c + 0.5) * cell, geo.originY + r * cell));
        }
      }
    }
    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c <= p.cols; c++) {
        if (game.marks[p.vIndex(r, c)] == EdgeMark.cross) {
          cross(Offset(geo.originX + c * cell, geo.originY + (r + 0.5) * cell));
        }
      }
    }
  }

  void _paintLines(Canvas canvas, EdgeLoopPuzzle p, double cell) {
    final paint = Paint()
      ..color = solved ? AppColors.solved : AppColors.line
      ..strokeWidth = cell * 0.10
      // 圆头：两条线在格点处交汇时接缝才不会露出缺口。
      ..strokeCap = StrokeCap.round;

    for (var r = 0; r <= p.rows; r++) {
      for (var c = 0; c < p.cols; c++) {
        if (game.marks[p.hIndex(r, c)] == EdgeMark.line) {
          canvas.drawLine(
              _dotOffset(geo, r, c), _dotOffset(geo, r, c + 1), paint);
        }
      }
    }
    for (var r = 0; r < p.rows; r++) {
      for (var c = 0; c <= p.cols; c++) {
        if (game.marks[p.vIndex(r, c)] == EdgeMark.line) {
          canvas.drawLine(
              _dotOffset(geo, r, c), _dotOffset(geo, r + 1, c), paint);
        }
      }
    }
  }

  @override
  bool shouldRepaint(covariant _BoardPainter old) => true;
}
