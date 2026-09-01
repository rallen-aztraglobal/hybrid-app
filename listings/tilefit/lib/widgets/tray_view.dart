import 'package:flutter/material.dart';

import '../models/piece.dart';
import '../theme/app_colors.dart';
import 'tile.dart';

/// 拖动时方块中心相对手指上浮的距离，以格边长为单位。
///
/// 不上浮的话方块正好被手指盖住，落点全靠猜。1.4 格是实机试出来的：
/// 再小挡指节，再大则方块跑得离手指太远、对不准。
const double kDragLiftCells = 1.4;

/// 待放区：三个槽，每个槽里一个可拖的方块。
///
/// 拖动浮层按**棋盘的**格边长绘制，缩略图按更小的 [thumbCellSize] —— 于是手指一按下去
/// 方块就"长"到真实大小，玩家看到的即是它落到棋盘上的样子。
class TrayView extends StatelessWidget {
  const TrayView({
    super.key,
    required this.pieces,
    required this.isPlaceable,
    required this.thumbCellSize,
    required this.dragCellSize,
    required this.onDragEnd,
  });

  /// 三个槽的内容，null 表示该槽已用掉。
  final List<Piece?> pieces;

  /// 第 index 个方块在棋盘上还有没有落点。没有的画灰。
  final bool Function(int index) isPlaceable;

  final double thumbCellSize;
  final double dragCellSize;

  /// 拖动结束（无论落成还是取消）时回调，供上层收掉落点预览。
  final VoidCallback onDragEnd;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: <Widget>[
        for (var i = 0; i < pieces.length; i++) Expanded(child: _slot(i)),
      ],
    );
  }

  /// 把缩略图铺满整个槽位，且整块区域都可命中。
  ///
  /// **别只把缩略图本身作为 Draggable 的 child。** 缩略图是一堆 `Positioned` 的方格，
  /// 格与格之间留了 12% 的缝、L 形还有整块空缺，这些位置底下没有任何可命中的 widget，
  /// 手指按上去拖不起来 —— 而"两格之间的那道缝"恰恰是玩家最容易按到的地方
  /// （宽或高是偶数的形状，其外接矩形的正中心就落在缝上）。
  /// 铺一层透明的 [ColoredBox]（它的命中行为是 opaque）把整个槽位变成把手，
  /// 按到这一格附近的任何地方都能把它拖起来。
  Widget _grip(Widget child) => ColoredBox(
    color: Colors.transparent,
    child: Center(child: child),
  );

  Widget _slot(int index) {
    final piece = pieces[index];
    if (piece == null) return const SizedBox.expand();

    final placeable = isPlaceable(index);
    final thumb = PieceView(
      key: ValueKey<String>('tray_$index'),
      piece: piece,
      cellSize: thumbCellSize,
      color: placeable ? null : AppColors.deadPiece,
    );

    return Draggable<int>(
      data: index,
      // 让浮层中心浮在手指上方，且水平居中于手指。
      // dragStartPoint 是"手指在浮层坐标系里的位置"，故这里给的偏移越大，浮层越往左上跑。
      dragAnchorStrategy: (_, _, _) => Offset(
        piece.width * dragCellSize / 2,
        piece.height * dragCellSize / 2 + kDragLiftCells * dragCellSize,
      ),
      feedback: PieceView(piece: piece, cellSize: dragCellSize),
      // 拖起来后原位留个淡影，提示这个槽正在被拖、还没消耗掉。
      childWhenDragging: _grip(
        PieceView(piece: piece, cellSize: thumbCellSize, opacity: 0.18),
      ),
      onDragEnd: (_) => onDragEnd(),
      child: _grip(thumb),
    );
  }
}
