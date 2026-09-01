import 'package:flutter/material.dart';

import '../logic/board.dart';
import '../models/piece.dart';
import '../theme/app_colors.dart';
import 'tile.dart';

/// 拖动过程中的落点预览。
@immutable
class GhostPlacement {
  const GhostPlacement({
    required this.piece,
    required this.topLeft,
    required this.valid,
  });

  final Piece piece;

  /// 方块左上角对应的棋盘格。可能部分或全部在界外（此时 [valid] 必为 false）。
  final Cell topLeft;

  /// 这个落点是否放得下。
  final bool valid;
}

/// 棋盘。边长恒为 [Board.size] × [cellSize]，由外层负责算出合适的 [cellSize]。
class BoardView extends StatelessWidget {
  const BoardView({
    super.key,
    required this.board,
    required this.cellSize,
    this.ghost,
  });

  final Board board;
  final double cellSize;
  final GhostPlacement? ghost;

  @override
  Widget build(BuildContext context) {
    final side = board.size * cellSize;
    // 底板圆角比格子大一档，视觉上"包住"里面的格子。
    final padding = cellSize * 0.16;

    return Container(
      width: side + padding * 2,
      height: side + padding * 2,
      padding: EdgeInsets.all(padding),
      decoration: BoxDecoration(
        color: AppColors.boardSurface,
        borderRadius: BorderRadius.circular(cellSize * 0.45),
      ),
      child: SizedBox(
        width: side,
        height: side,
        child: Stack(children: <Widget>[..._cells(), ..._ghostCells()]),
      ),
    );
  }

  List<Widget> _cells() {
    final widgets = <Widget>[];
    for (var r = 0; r < board.size; r++) {
      for (var c = 0; c < board.size; c++) {
        final value = board.cellAt(r, c);
        widgets.add(
          Positioned(
            left: c * cellSize,
            top: r * cellSize,
            child: TileSquare(
              size: cellSize,
              color: value == Board.emptyCell ? null : AppColors.piece(value),
            ),
          ),
        );
      }
    }
    return widgets;
  }

  /// 落点预览：放得下画方块本色的半透明，放不下画红。
  /// 界外的格子直接跳过 —— 画到棋盘外面会溢出底板。
  List<Widget> _ghostCells() {
    final g = ghost;
    if (g == null) return const <Widget>[];

    final color = g.valid
        ? AppColors.piece(g.piece.colorIndex).withValues(alpha: 0.45)
        : AppColors.ghostInvalid;

    final widgets = <Widget>[];
    for (final cell in g.piece.cells) {
      final r = g.topLeft.row + cell.row;
      final c = g.topLeft.col + cell.col;
      if (r < 0 || r >= board.size || c < 0 || c >= board.size) continue;
      widgets.add(
        Positioned(
          left: c * cellSize,
          top: r * cellSize,
          child: TileSquare(size: cellSize, color: color),
        ),
      );
    }
    return widgets;
  }
}
