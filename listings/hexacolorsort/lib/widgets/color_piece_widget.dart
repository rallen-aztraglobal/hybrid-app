import 'package:flutter/material.dart';

import '../core/constants/game_constants.dart';

/// Renders one color block. An icon is layered on top of the fill color so
/// color is never the only way to tell two pieces apart.
///
/// Either pass [size] for the original squarish-pill proportions, or pass
/// explicit [width]/[height] (used for the slimmer bands stacked inside a
/// [ColourStack]).
class ColorPieceWidget extends StatelessWidget {
  final int colorId;
  final double size;
  final double? width;
  final double? height;

  const ColorPieceWidget({
    super.key,
    required this.colorId,
    this.size = 44,
    this.width,
    this.height,
  });

  @override
  Widget build(BuildContext context) {
    final style = kColorPalette[colorId % kColorPalette.length];
    final w = width ?? size;
    final h = height ?? size * 0.62;
    final radius = (w < h ? w : h) * 0.32;
    return Container(
      width: w,
      height: h,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(radius),
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [style.highlight, style.color],
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.35),
            blurRadius: 4,
            offset: const Offset(0, 3),
          ),
        ],
        border: Border.all(color: Colors.white.withValues(alpha: 0.25)),
      ),
      alignment: Alignment.center,
      child: Icon(
        style.icon,
        size: h * 0.55,
        color: Colors.white.withValues(alpha: 0.9),
      ),
    );
  }
}
