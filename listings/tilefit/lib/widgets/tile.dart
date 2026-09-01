import 'package:flutter/material.dart';

import '../models/piece.dart';
import '../theme/app_colors.dart';

/// 一格的边长中，真正画出来的比例。剩下的留作格与格之间的缝，
/// 缝由底色透出来，所以棋盘不需要额外画网格线。
const double kTileFillRatio = 0.88;

/// 圆角占格边长的比例。
const double kTileRadiusRatio = 0.22;

/// 单个方格。[color] 为 null 时画成空格。
///
/// 实心格用一道自左上到右下的浅→深渐变做出微弱的立体感；空格是纯色。
/// 不用阴影：8×8 棋盘加上待放区一屏最多百来个格，阴影在低端机上掉帧明显，
/// 而渐变是纯着色、几乎零成本。
class TileSquare extends StatelessWidget {
  const TileSquare({super.key, required this.size, this.color});

  final double size;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final fill = size * kTileFillRatio;
    final radius = size * kTileRadiusRatio;
    final c = color;

    return SizedBox(
      width: size,
      height: size,
      child: Center(
        child: Container(
          width: fill,
          height: fill,
          decoration: BoxDecoration(
            color: c == null ? AppColors.emptyCell : null,
            gradient: c == null
                ? null
                : LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: <Color>[
                      Color.lerp(c, Colors.white, 0.18)!,
                      c,
                      Color.lerp(c, Colors.black, 0.14)!,
                    ],
                    stops: const <double>[0.0, 0.55, 1.0],
                  ),
            borderRadius: BorderRadius.circular(radius),
          ),
        ),
      ),
    );
  }
}

/// 把一个 [Piece] 按 [cellSize] 画出来，尺寸恰好是它的外接矩形。
///
/// 待放区的缩略图、拖动时跟手的浮层、棋盘上的落点预览都用它，
/// 三处只差 [cellSize] 与 [color] —— 同一个形状在三个地方长得一模一样。
class PieceView extends StatelessWidget {
  const PieceView({
    super.key,
    required this.piece,
    required this.cellSize,
    this.color,
    this.opacity = 1.0,
  });

  final Piece piece;
  final double cellSize;

  /// 覆盖方块本色（待放区里放不下的方块画灰）。null 表示用 [Piece.colorIndex] 的本色。
  final Color? color;

  final double opacity;

  @override
  Widget build(BuildContext context) {
    final fill = color ?? AppColors.piece(piece.colorIndex);

    return Opacity(
      opacity: opacity,
      child: SizedBox(
        width: piece.width * cellSize,
        height: piece.height * cellSize,
        child: Stack(
          children: <Widget>[
            for (final cell in piece.cells)
              Positioned(
                left: cell.col * cellSize,
                top: cell.row * cellSize,
                child: TileSquare(size: cellSize, color: fill),
              ),
          ],
        ),
      ),
    );
  }
}
