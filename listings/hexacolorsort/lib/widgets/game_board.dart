import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';

import '../core/constants/game_constants.dart';
import '../game/logic/game_controller.dart';
import '../game/logic/game_event.dart';
import 'color_piece_widget.dart';
import 'colour_stack.dart';
import 'combo_overlay.dart';

class _GridMetrics {
  final int columns;
  final double cellWidth;
  final double cellHeight;
  final double scale;
  final double spacing;

  const _GridMetrics({
    required this.columns,
    required this.cellWidth,
    required this.cellHeight,
    required this.scale,
    required this.spacing,
  });

  Offset centerOf(int index) {
    final row = index ~/ columns;
    final col = index % columns;
    final x = col * cellWidth + cellWidth / 2;
    final y = row * cellHeight + cellHeight / 2;
    return Offset(x, y);
  }
}

/// Renders the board of vertical color stacks responsively and layers
/// transient animations (flying pieces, clear bursts, combo popups) above
/// it.
class GameBoard extends StatefulWidget {
  final GameController controller;

  const GameBoard({super.key, required this.controller});

  @override
  State<GameBoard> createState() => _GameBoardState();
}

class _GameBoardState extends State<GameBoard> {
  final Map<int, int> _shakeSignals = {};
  final Map<int, Widget> _overlayItems = {};
  int _overlayIdCounter = 0;
  StreamSubscription<GameEvent>? _subscription;
  _GridMetrics? _metrics;

  @override
  void initState() {
    super.initState();
    _subscription = widget.controller.events.listen(_handleEvent);
  }

  @override
  void dispose() {
    _subscription?.cancel();
    super.dispose();
  }

  void _handleEvent(GameEvent event) {
    if (!mounted || _metrics == null) return;
    final metrics = _metrics!;
    switch (event) {
      case IllegalMoveEvent(:final stackIndex):
        setState(
          () =>
              _shakeSignals[stackIndex] = (_shakeSignals[stackIndex] ?? 0) + 1,
        );
      case MoveEvent(:final fromIndex, :final toIndex, :final colorId):
        final id = _overlayIdCounter++;
        final start = metrics.centerOf(fromIndex);
        final end = metrics.centerOf(toIndex);
        setState(() {
          _overlayItems[id] = _FlyingPiece(
            start: start,
            end: end,
            colorId: colorId,
            scale: metrics.scale,
            onComplete: () => _removeOverlay(id),
          );
        });
      case ClearEvent(
        :final stackIndex,
        :final colorId,
        :final comboStreak,
        :final scoreAwarded,
      ):
        final burstId = _overlayIdCounter++;
        final popupId = _overlayIdCounter++;
        final center = metrics.centerOf(stackIndex);
        final color = kColorPalette[colorId % kColorPalette.length].color;
        setState(() {
          _overlayItems[burstId] = Positioned(
            left: center.dx - 65,
            top: center.dy - 65,
            child: ClearBurst(
              color: color,
              onComplete: () => _removeOverlay(burstId),
            ),
          );
          _overlayItems[popupId] = Positioned(
            left: center.dx - 60,
            top: center.dy - 100,
            child: SizedBox(
              width: 120,
              child: Center(
                child: ComboPopup(
                  label: comboStreak > 1
                      ? 'Combo x$comboStreak +$scoreAwarded'
                      : '+$scoreAwarded',
                  color: color,
                  onComplete: () => _removeOverlay(popupId),
                ),
              ),
            ),
          );
        });
      default:
        break;
    }
  }

  void _removeOverlay(int id) {
    if (!mounted) return;
    setState(() => _overlayItems.remove(id));
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.controller,
      builder: (context, _) {
        final state = widget.controller.state;
        final stacks = state.stacks;
        return LayoutBuilder(
          builder: (context, constraints) {
            final columns = stacks.length <= 6 ? 3 : 4;
            final rows = (stacks.length / columns).ceil();
            const spacing = 10.0;
            final baseCellWidth = ColourStack.totalWidth() + spacing;
            final baseCellHeight =
                ColourStack.totalHeight(GameConstants.stackCapacity) + spacing;
            final scaleW = constraints.maxWidth / (columns * baseCellWidth);
            final scaleH = constraints.maxHeight / (rows * baseCellHeight);
            final scale = min(1.0, min(scaleW, scaleH)).clamp(0.45, 1.0);

            final cellWidth = baseCellWidth * scale;
            final cellHeight = baseCellHeight * scale;
            _metrics = _GridMetrics(
              columns: columns,
              cellWidth: cellWidth,
              cellHeight: cellHeight,
              scale: scale,
              spacing: spacing * scale,
            );

            final boardWidth = columns * cellWidth;
            final boardHeight = rows * cellHeight;

            return Center(
              child: SizedBox(
                width: boardWidth,
                height: boardHeight,
                child: Stack(
                  clipBehavior: Clip.none,
                  children: [
                    for (var i = 0; i < stacks.length; i++)
                      Positioned(
                        left: (i % columns) * cellWidth,
                        top: (i ~/ columns) * cellHeight,
                        width: cellWidth,
                        height: cellHeight,
                        child: Center(
                          child: ColourStack(
                            stack: stacks[i],
                            isSelected: state.selectedStackIndex == i,
                            shakeSignal: _shakeSignals[i] ?? 0,
                            scale: scale,
                            onTap: () => widget.controller.selectStack(i),
                          ),
                        ),
                      ),
                    for (final entry in _overlayItems.entries) entry.value,
                  ],
                ),
              ),
            );
          },
        );
      },
    );
  }
}

class _FlyingPiece extends StatefulWidget {
  final Offset start;
  final Offset end;
  final int colorId;
  final double scale;
  final VoidCallback onComplete;

  const _FlyingPiece({
    required this.start,
    required this.end,
    required this.colorId,
    required this.scale,
    required this.onComplete,
  });

  @override
  State<_FlyingPiece> createState() => _FlyingPieceState();
}

class _FlyingPieceState extends State<_FlyingPiece>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        AnimationController(
            vsync: this,
            duration: GameConstants.moveAnimationDuration,
          )
          ..forward().whenComplete(() {
            if (mounted) widget.onComplete();
          });
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) {
        final t = Curves.easeInOut.transform(_controller.value);
        final arc = sin(pi * t) * 40;
        final dx = widget.start.dx + (widget.end.dx - widget.start.dx) * t;
        final dy =
            widget.start.dy + (widget.end.dy - widget.start.dy) * t - arc;
        final pieceWidth =
            (ColourStack.baseWidth - ColourStack.baseHorizontalPadding * 2) *
            widget.scale;
        final pieceHeight = (ColourStack.basePieceHeight - 3) * widget.scale;
        return Positioned(
          left: dx - pieceWidth / 2,
          top: dy - pieceHeight / 2,
          child: ColorPieceWidget(
            colorId: widget.colorId,
            width: pieceWidth,
            height: pieceHeight,
          ),
        );
      },
    );
  }
}
