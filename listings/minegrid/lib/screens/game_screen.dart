import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/difficulty.dart';
import '../logic/game.dart';
import '../storage/progress_store.dart';
import '../theme/app_colors.dart';
import '../widgets/board_view.dart';

/// A 面：扫雷本体。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  Difficulty _difficulty = Difficulty.normal;
  late MineGame _game = MineGame(_difficulty);
  InputMode _mode = InputMode.dig;

  Timer? _ticker;
  int _seconds = 0;

  int _wins = 0;
  int? _best;

  /// 这一局是不是刷新了最好成绩。通关那一下拿来做个额外的强调。
  bool _newRecord = false;

  @override
  void initState() {
    super.initState();
    _loadStats();
  }

  @override
  void dispose() {
    _ticker?.cancel();
    super.dispose();
  }

  Future<void> _loadStats() async {
    final difficulty = _difficulty;
    final wins = await ProgressStore.wins(difficulty);
    final best = await ProgressStore.bestSeconds(difficulty);
    // 读盘期间玩家可能又切了档，那就丢弃这次结果。
    if (!mounted || difficulty != _difficulty) return;
    setState(() {
      _wins = wins;
      _best = best;
    });
  }

  void _switchDifficulty(Difficulty difficulty) {
    if (difficulty == _difficulty) return;
    HapticFeedback.selectionClick();
    _stopTimer();
    setState(() {
      _difficulty = difficulty;
      _game = MineGame(difficulty);
      _mode = InputMode.dig;
      _seconds = 0;
      _newRecord = false;
    });
    _loadStats();
  }

  void _newGame() {
    HapticFeedback.selectionClick();
    _stopTimer();
    setState(() {
      _game = MineGame(_difficulty);
      _mode = InputMode.dig;
      _seconds = 0;
      _newRecord = false;
    });
  }

  void _startTimer() {
    _ticker?.cancel();
    _ticker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;
      setState(() => _seconds++);
    });
  }

  void _stopTimer() {
    _ticker?.cancel();
    _ticker = null;
  }

  Future<void> _onChanged() async {
    final wasRunning = _ticker != null;
    setState(() {});

    if (_game.status == GameStatus.playing && !wasRunning) {
      _startTimer();
      return;
    }

    if (_game.status == GameStatus.won) {
      _stopTimer();
      final previous = _best;
      final (wins, best) = await ProgressStore.recordWin(_difficulty, _seconds);
      if (mounted) {
        setState(() {
          _wins = wins;
          _best = best;
          _newRecord = previous == null || _seconds < previous;
        });
      }
      // 通关的触感放在最后：让它压在「补旗」动画的开头，两件事一起发生
      await HapticFeedback.mediumImpact();
    } else if (_game.status == GameStatus.lost) {
      _stopTimer();
      await HapticFeedback.heavyImpact();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: AnnotatedRegion<SystemUiOverlayStyle>(
        // 深色底必须配浅色状态栏图标，否则系统图标会黑底黑字看不见。
        value: SystemUiOverlayStyle.light.copyWith(
          statusBarColor: Colors.transparent,
        ),
        child: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: <Color>[AppColors.backgroundTint, AppColors.background],
              stops: <double>[0.0, 0.55],
            ),
          ),
          child: SafeArea(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(14, 8, 14, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: <Widget>[
                  _header(),
                  const SizedBox(height: 12),
                  _SegmentedBar<Difficulty>(
                    values: Difficulty.values,
                    selected: _difficulty,
                    labelOf: (d) => d.label,
                    onSelect: _switchDifficulty,
                  ),
                  const SizedBox(height: 14),
                  _statusRow(),
                  const SizedBox(height: 10),
                  Expanded(
                    child: Center(
                      child: AspectRatio(
                        aspectRatio: _difficulty.cols / _difficulty.rows,
                        child: BoardView(
                          game: _game,
                          mode: _mode,
                          onChanged: _onChanged,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 14),
                  _actions(),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _header() {
    return Row(
      children: <Widget>[
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              const Text(
                'MineGrid',
                style: TextStyle(
                  color: AppColors.primaryText,
                  fontSize: 22,
                  fontWeight: FontWeight.w700,
                  letterSpacing: -0.3,
                  height: 1.1,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                _wins == 1 ? '1 board cleared' : '$_wins boards cleared',
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 12.5,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
        AnimatedSwitcher(
          duration: const Duration(milliseconds: 220),
          child: _best == null
              ? const SizedBox.shrink()
              : Container(
                  key: ValueKey<String>('best-$_best-$_newRecord'),
                  padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 7),
                  decoration: BoxDecoration(
                    color: _newRecord
                        ? AppColors.success.withValues(alpha: 0.18)
                        : AppColors.boardSurface,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(
                    'Best ${_format(_best!)}',
                    style: TextStyle(
                      color: _newRecord ? AppColors.success : AppColors.primaryText,
                      fontSize: 12.5,
                      fontWeight: FontWeight.w700,
                      fontFeatures: const <FontFeature>[
                        FontFeature.tabularFigures(),
                      ],
                    ),
                  ),
                ),
        ),
      ],
    );
  }

  /// 剩余雷数 · 结果 · 计时。
  Widget _statusRow() {
    final finished =
        _game.status == GameStatus.won || _game.status == GameStatus.lost;
    final won = _game.status == GameStatus.won;

    return SizedBox(
      height: 24,
      child: Row(
        children: <Widget>[
          Expanded(
            child: Align(
              alignment: Alignment.centerLeft,
              child: _pill(
                icon: Icons.flag_rounded,
                text: '${_game.minesLeft}',
                color: AppColors.accent,
              ),
            ),
          ),
          AnimatedSwitcher(
            duration: const Duration(milliseconds: 260),
            transitionBuilder: (child, animation) => FadeTransition(
              opacity: animation,
              child: ScaleTransition(scale: animation, child: child),
            ),
            child: !finished
                ? const SizedBox.shrink()
                : Text(
                    won
                        ? (_newRecord ? 'New record' : 'Cleared')
                        : 'Boom',
                    key: ValueKey<String>('$won$_newRecord'),
                    style: TextStyle(
                      color: won ? AppColors.success : AppColors.explosion,
                      fontSize: 13,
                      fontWeight: FontWeight.w800,
                      letterSpacing: 0.3,
                    ),
                  ),
          ),
          Expanded(
            child: Align(
              alignment: Alignment.centerRight,
              child: _pill(
                icon: Icons.timer_outlined,
                text: _format(_seconds),
                color: AppColors.mutedText,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _pill({
    required IconData icon,
    required String text,
    required Color color,
  }) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Icon(icon, size: 16, color: color),
        const SizedBox(width: 5),
        Text(
          text,
          style: const TextStyle(
            color: AppColors.primaryText,
            fontSize: 14,
            fontWeight: FontWeight.w700,
            fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
          ),
        ),
      ],
    );
  }

  Widget _actions() {
    final finished =
        _game.status == GameStatus.won || _game.status == GameStatus.lost;

    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 240),
      switchInCurve: Curves.easeOutCubic,
      child: finished
          ? SizedBox(
              key: const ValueKey<String>('finished'),
              height: 48,
              child: FilledButton.icon(
                onPressed: _newGame,
                style: FilledButton.styleFrom(
                  backgroundColor: _game.status == GameStatus.won
                      ? AppColors.success
                      : AppColors.accent,
                  foregroundColor: const Color(0xFF10131A),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(12),
                  ),
                ),
                icon: const Icon(Icons.refresh_rounded, size: 18),
                label: const Text(
                  'New game',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
                ),
              ),
            )
          : Row(
              key: const ValueKey<String>('playing'),
              children: <Widget>[
                Expanded(
                  child: _SegmentedBar<InputMode>(
                    height: 48,
                    values: InputMode.values,
                    selected: _mode,
                    labelOf: (m) => m == InputMode.dig ? 'Dig' : 'Flag',
                    iconOf: (m) => m == InputMode.dig
                        ? Icons.touch_app_rounded
                        : Icons.flag_rounded,
                    onSelect: (m) {
                      if (m == _mode) return;
                      HapticFeedback.selectionClick();
                      setState(() => _mode = m);
                    },
                  ),
                ),
                const SizedBox(width: 10),
                SizedBox(
                  height: 48,
                  width: 56,
                  child: TextButton(
                    onPressed: _newGame,
                    style: TextButton.styleFrom(
                      backgroundColor: AppColors.boardSurface,
                      foregroundColor: AppColors.mutedText,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                    child: const Icon(Icons.refresh_rounded, size: 20),
                  ),
                ),
              ],
            ),
    );
  }

  String _format(int seconds) {
    final m = (seconds ~/ 60).toString().padLeft(2, '0');
    final s = (seconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }
}

/// 分段选择器：一颗会滑动的高亮块 + 若干等宽的标签。
///
/// 用「一块高亮滑过去」而不是「旧的褪色、新的上色」——
/// 后者是两个独立的淡入淡出，眼睛跟不上焦点去了哪；滑动是一条连续的轨迹，
/// 一眼就知道从哪到哪。难度和挖/插旗两处都用它，手感一致。
class _SegmentedBar<T> extends StatelessWidget {
  const _SegmentedBar({
    super.key,
    required this.values,
    required this.selected,
    required this.labelOf,
    required this.onSelect,
    this.iconOf,
    this.height = 38,
  });

  final List<T> values;
  final T selected;
  final String Function(T) labelOf;
  final IconData? Function(T)? iconOf;
  final ValueChanged<T> onSelect;
  final double height;

  @override
  Widget build(BuildContext context) {
    final index = values.indexOf(selected);
    return Container(
      height: height,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: AppColors.boardSurface,
        borderRadius: BorderRadius.circular(12),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final segment = constraints.maxWidth / values.length;
          return Stack(
            children: <Widget>[
              AnimatedPositioned(
                duration: const Duration(milliseconds: 240),
                curve: Curves.easeOutCubic,
                left: segment * index,
                top: 0,
                bottom: 0,
                width: segment,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: AppColors.accent,
                    borderRadius: BorderRadius.circular(9),
                  ),
                ),
              ),
              Row(
                children: <Widget>[
                  for (final value in values)
                    Expanded(
                      child: GestureDetector(
                        behavior: HitTestBehavior.opaque,
                        onTap: () => onSelect(value),
                        child: _label(value, value == selected),
                      ),
                    ),
                ],
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _label(T value, bool active) {
    // 选中项的字色要在滑块底下，所以用深色；跟随滑块一起淡入淡出。
    final color = active ? const Color(0xFF10131A) : AppColors.mutedText;
    final icon = iconOf?.call(value);
    return Center(
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (icon != null) ...<Widget>[
            TweenAnimationBuilder<Color?>(
              duration: const Duration(milliseconds: 240),
              tween: ColorTween(end: color),
              builder: (context, c, _) => Icon(icon, size: 17, color: c),
            ),
            const SizedBox(width: 6),
          ],
          AnimatedDefaultTextStyle(
            duration: const Duration(milliseconds: 240),
            style: TextStyle(
              color: color,
              fontSize: 12.5,
              fontWeight: FontWeight.w700,
            ),
            child: Text(labelOf(value)),
          ),
        ],
      ),
    );
  }
}
