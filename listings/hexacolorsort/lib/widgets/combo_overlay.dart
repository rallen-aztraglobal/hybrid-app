import 'package:flutter/material.dart';

import '../core/constants/game_constants.dart';
import '../core/theme/app_theme.dart';

/// A short-lived "+score" / combo callout that pops up and fades away.
class ComboPopup extends StatefulWidget {
  final String label;
  final Color color;
  final VoidCallback onComplete;

  const ComboPopup({
    super.key,
    required this.label,
    required this.color,
    required this.onComplete,
  });

  @override
  State<ComboPopup> createState() => _ComboPopupState();
}

class _ComboPopupState extends State<ComboPopup>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _scale;
  late final Animation<double> _opacity;
  late final Animation<Offset> _rise;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: GameConstants.comboPopupDuration,
    );
    _scale = TweenSequence<double>([
      TweenSequenceItem(tween: Tween(begin: 0.4, end: 1.2), weight: 40),
      TweenSequenceItem(tween: Tween(begin: 1.2, end: 1.0), weight: 60),
    ]).animate(CurvedAnimation(parent: _controller, curve: Curves.easeOut));
    _opacity = TweenSequence<double>([
      TweenSequenceItem(tween: Tween(begin: 0.0, end: 1.0), weight: 20),
      TweenSequenceItem(tween: Tween(begin: 1.0, end: 1.0), weight: 55),
      TweenSequenceItem(tween: Tween(begin: 1.0, end: 0.0), weight: 25),
    ]).animate(_controller);
    _rise = Tween<Offset>(
      begin: Offset.zero,
      end: const Offset(0, -0.6),
    ).animate(CurvedAnimation(parent: _controller, curve: Curves.easeOut));
    _controller.forward().whenComplete(() {
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
      builder: (context, child) {
        return Opacity(
          opacity: _opacity.value.clamp(0.0, 1.0),
          child: FractionalTranslation(
            translation: _rise.value,
            child: Transform.scale(scale: _scale.value, child: child),
          ),
        );
      },
      child: Text(
        widget.label,
        style: TextStyle(
          fontSize: 26,
          fontWeight: FontWeight.w800,
          color: widget.color,
          shadows: const [Shadow(color: Colors.black54, blurRadius: 6)],
        ),
      ),
    );
  }
}

/// A brief radial burst used behind a clearing hex tray.
class ClearBurst extends StatefulWidget {
  final Color color;
  final VoidCallback onComplete;

  const ClearBurst({super.key, required this.color, required this.onComplete});

  @override
  State<ClearBurst> createState() => _ClearBurstState();
}

class _ClearBurstState extends State<ClearBurst>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        AnimationController(
            vsync: this,
            duration: GameConstants.clearAnimationDuration,
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
        final t = _controller.value;
        final size = 40 + t * 90;
        final opacity = (1 - t).clamp(0.0, 1.0);
        return Opacity(
          opacity: opacity,
          child: Container(
            width: size,
            height: size,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              gradient: RadialGradient(
                colors: [
                  widget.color.withValues(alpha: 0.85),
                  widget.color.withValues(alpha: 0.0),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

/// Displays the running combo count near the bottom action bar.
class ComboBadge extends StatelessWidget {
  final int combo;

  const ComboBadge({super.key, required this.combo});

  @override
  Widget build(BuildContext context) {
    final visible = combo > 1;
    return AnimatedOpacity(
      opacity: visible ? 1 : 0,
      duration: GameConstants.selectAnimationDuration,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        decoration: BoxDecoration(
          color: AppTheme.accent.withValues(alpha: 0.25),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppTheme.accent),
        ),
        child: Text(
          'Combo x$combo',
          style: const TextStyle(
            color: AppTheme.textPrimary,
            fontWeight: FontWeight.bold,
            fontSize: 14,
          ),
        ),
      ),
    );
  }
}
