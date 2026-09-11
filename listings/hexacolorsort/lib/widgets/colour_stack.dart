import 'package:flutter/material.dart';

import '../core/constants/game_constants.dart';
import '../core/theme/app_theme.dart';
import '../game/models/stack_model.dart';
import 'color_piece_widget.dart';

/// A single vertical tube-style tray: a rounded column that holds its
/// pieces stacked bottom-to-top, with selection glow, a subtle raise, and
/// a shake reaction for illegal move attempts.
class ColourStack extends StatefulWidget {
  static const double baseWidth = 64;
  static const double basePieceHeight = 30;
  static const double baseVerticalPadding = 10;
  static const double baseHorizontalPadding = 6;

  final StackModel stack;
  final bool isSelected;
  final int shakeSignal;
  final double scale;
  final VoidCallback onTap;

  const ColourStack({
    super.key,
    required this.stack,
    required this.isSelected,
    required this.shakeSignal,
    required this.onTap,
    this.scale = 1.0,
  });

  static double totalHeight(int capacity, {double scale = 1.0}) {
    return (capacity * basePieceHeight + baseVerticalPadding * 2) * scale;
  }

  static double totalWidth({double scale = 1.0}) => baseWidth * scale;

  @override
  State<ColourStack> createState() => _ColourStackState();
}

class _ColourStackState extends State<ColourStack>
    with SingleTickerProviderStateMixin {
  late final AnimationController _shakeController;
  late final Animation<double> _shakeAnimation;

  @override
  void initState() {
    super.initState();
    _shakeController = AnimationController(
      vsync: this,
      duration: GameConstants.shakeAnimationDuration,
    );
    _shakeAnimation = TweenSequence<double>(
      [
        TweenSequenceItem(tween: Tween(begin: 0.0, end: -9.0), weight: 1),
        TweenSequenceItem(tween: Tween(begin: -9.0, end: 9.0), weight: 1),
        TweenSequenceItem(tween: Tween(begin: 9.0, end: -6.0), weight: 1),
        TweenSequenceItem(tween: Tween(begin: -6.0, end: 0.0), weight: 1),
      ],
    ).animate(CurvedAnimation(parent: _shakeController, curve: Curves.easeOut));
  }

  @override
  void didUpdateWidget(covariant ColourStack oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.shakeSignal != oldWidget.shakeSignal) {
      final reduceMotion =
          MediaQuery.maybeOf(context)?.disableAnimations ?? false;
      if (!reduceMotion) {
        _shakeController.forward(from: 0);
      }
    }
  }

  @override
  void dispose() {
    _shakeController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final scale = widget.scale;
    final width = ColourStack.baseWidth * scale;
    final pieceHeight = ColourStack.basePieceHeight * scale;
    final verticalPadding = ColourStack.baseVerticalPadding * scale;
    final horizontalPadding = ColourStack.baseHorizontalPadding * scale;
    final pieces = widget.stack.pieces;
    final capacity = widget.stack.capacity;
    final height = ColourStack.totalHeight(capacity, scale: scale);

    final content = Container(
      width: width,
      height: height,
      padding: EdgeInsets.symmetric(
        vertical: verticalPadding,
        horizontal: horizontalPadding,
      ),
      decoration: BoxDecoration(
        color: widget.isSelected ? AppTheme.surfaceLight : AppTheme.surface,
        borderRadius: BorderRadius.circular(width / 2),
        border: Border.all(
          color: widget.isSelected ? AppTheme.accent : Colors.white24,
          width: (widget.isSelected ? 3 : 1.5) * scale,
        ),
        boxShadow: widget.isSelected
            ? [
                BoxShadow(
                  color: AppTheme.accent.withValues(alpha: 0.55),
                  blurRadius: 14 * scale,
                  spreadRadius: 1 * scale,
                ),
              ]
            : const [],
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          for (final piece in pieces.reversed)
            Padding(
              padding: EdgeInsets.symmetric(vertical: 1.5 * scale),
              child: ColorPieceWidget(
                colorId: piece.colorId,
                width: width - horizontalPadding * 2,
                height: pieceHeight - 3 * scale,
              ),
            ),
        ],
      ),
    );

    return GestureDetector(
      onTap: widget.onTap,
      child: AnimatedBuilder(
        animation: _shakeAnimation,
        builder: (context, child) {
          return Transform.translate(
            offset: Offset(_shakeAnimation.value, 0),
            child: child,
          );
        },
        child: AnimatedScale(
          scale: widget.isSelected ? 1.06 : 1.0,
          duration: GameConstants.selectAnimationDuration,
          child: AnimatedSlide(
            offset: widget.isSelected ? const Offset(0, -0.04) : Offset.zero,
            duration: GameConstants.selectAnimationDuration,
            child: content,
          ),
        ),
      ),
    );
  }
}
