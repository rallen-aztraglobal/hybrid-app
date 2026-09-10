import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/game.dart';
import '../logic/puzzle_library.dart';
import '../storage/progress_store.dart';
import '../theme/app_colors.dart';
import '../widgets/board_view.dart';

/// A 面：连线本体。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  Difficulty _difficulty = Difficulty.normal;
  int _levelIndex = 0;
  late FlowGame _game = FlowGame(levelsFor(_difficulty).first.build());

  int _solved = 0;
  bool _celebrating = false;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  /// 读回这一档的存档：打到第几关、解开过几道。
  Future<void> _restore() async {
    final difficulty = _difficulty;
    final index = await ProgressStore.levelIndex(difficulty);
    final solved = await ProgressStore.solvedCount(difficulty);
    // 读盘期间玩家可能又切了档，那就丢弃这次结果。
    if (!mounted || difficulty != _difficulty) return;
    setState(() {
      _levelIndex = index;
      _game = FlowGame(levelsFor(difficulty)[index].build());
      _solved = solved;
      _celebrating = false;
    });
  }

  void _switchDifficulty(Difficulty difficulty) {
    if (difficulty == _difficulty) return;
    setState(() {
      _difficulty = difficulty;
      _levelIndex = 0;
      _game = FlowGame(levelsFor(difficulty).first.build());
      _celebrating = false;
    });
    _restore();
  }

  /// 下一关。关卡在档内是按难度升序排的，所以顺着走就是难度递增，
  /// 不像随机抽题那样一会儿难一会儿易。走到底再从头绕回来。
  Future<void> _advance() async {
    final pool = levelsFor(_difficulty);
    final next = (_levelIndex + 1) % pool.length;
    setState(() {
      _levelIndex = next;
      _game = FlowGame(pool[next].build());
      _celebrating = false;
    });
    await ProgressStore.setLevelIndex(_difficulty, next);
  }

  Future<void> _onChanged() async {
    setState(() {});
    if (_game.isSolved && !_celebrating) {
      setState(() => _celebrating = true);
      await HapticFeedback.mediumImpact();
      final n = await ProgressStore.recordSolved(_difficulty);
      if (mounted) setState(() => _solved = n);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: AnnotatedRegion<SystemUiOverlayStyle>(
        // 浅色底必须配深色状态栏图标，否则系统图标会白底白字看不见。
        value: SystemUiOverlayStyle.dark.copyWith(
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
                  _difficultyBar(),
                  // 棋盘 + 进度作为一个整体，在上面的控件与底部按钮之间垂直居中。
                  Expanded(
                    child: Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: <Widget>[
                          AspectRatio(
                            aspectRatio: 1,
                            child: BoardView(
                              game: _game,
                              onChanged: _onChanged,
                            ),
                          ),
                          const SizedBox(height: 22),
                          _progress(),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
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
    final total = levelsFor(_difficulty).length;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: <Widget>[
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              const Text(
                'LinkFlow',
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
                _solved == 1 ? '1 puzzle solved' : '$_solved puzzles solved',
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 12.5,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 7),
          decoration: BoxDecoration(
            color: AppColors.boardSurface,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Text(
            'Level ${_levelIndex + 1} / $total',
            style: const TextStyle(
              color: AppColors.primaryText,
              fontSize: 12.5,
              fontWeight: FontWeight.w700,
              fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
            ),
          ),
        ),
      ],
    );
  }

  /// 四档难度。占满一整行 —— 四个词挤在标题旁边放不下，
  /// 而且难度是这个游戏最主要的入口，值得给一整行。
  Widget _difficultyBar() {
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: AppColors.boardSurface,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: <Widget>[
          for (final d in Difficulty.values)
            Expanded(
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: () => _switchDifficulty(d),
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 140),
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  decoration: BoxDecoration(
                    color: d == _difficulty ? AppColors.accent : Colors.transparent,
                    borderRadius: BorderRadius.circular(9),
                  ),
                  child: Text(
                    d.label,
                    textAlign: TextAlign.center,
                    style: TextStyle(
                      color: d == _difficulty ? Colors.white : AppColors.mutedText,
                      fontSize: 12.5,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  /// 进度：已连通的条数 + 铺满比例。
  ///
  /// 两个都显示是有意的 —— 连线游戏最常见的困惑是「线都连上了却没过关」，
  /// 原因是棋盘没铺满。把两个指标并排放出来，玩家一眼知道差在哪。
  Widget _progress() {
    final total = _game.puzzle.colorKeys.length;
    final done = _game.connectedCount;
    final cells = _game.puzzle.side * _game.puzzle.side;
    final ratio = _game.filledCount / cells;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Row(
          children: <Widget>[
            Text(
              _celebrating ? 'Solved' : 'Connected',
              style: TextStyle(
                color: _celebrating ? AppColors.accent : AppColors.mutedText,
                fontSize: 12.5,
                fontWeight: _celebrating ? FontWeight.w700 : FontWeight.w600,
                letterSpacing: 0.2,
              ),
            ),
            const Spacer(),
            Text(
              '$done / $total',
              style: const TextStyle(
                color: AppColors.primaryText,
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
            const SizedBox(width: 12),
            Text(
              '${(ratio * 100).round()}% full',
              style: TextStyle(
                color: ratio == 1.0 ? AppColors.accent : AppColors.mutedText,
                fontSize: 12.5,
                fontWeight: FontWeight.w600,
                fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: TweenAnimationBuilder<double>(
            tween: Tween<double>(begin: 0, end: ratio),
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeOut,
            builder: (context, value, _) => LinearProgressIndicator(
              value: value,
              minHeight: 5,
              backgroundColor: AppColors.emptyCell,
              valueColor: const AlwaysStoppedAnimation<Color>(AppColors.accent),
            ),
          ),
        ),
      ],
    );
  }

  Widget _actions() {
    final solved = _celebrating;
    return Row(
      children: <Widget>[
        Expanded(
          child: SizedBox(
            height: 46,
            child: TextButton.icon(
              onPressed: solved ? null : () => setState(_game.reset),
              style: TextButton.styleFrom(
                foregroundColor: AppColors.mutedText,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              icon: const Icon(Icons.refresh_rounded, size: 18),
              label: const Text(
                'Clear',
                style: TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          flex: 2,
          child: SizedBox(
            height: 46,
            child: FilledButton.icon(
              onPressed: _advance,
              style: FilledButton.styleFrom(
                backgroundColor: solved ? AppColors.accent : AppColors.boardSurface,
                foregroundColor: solved ? Colors.white : AppColors.mutedText,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              icon: const Icon(Icons.arrow_forward_rounded, size: 18),
              label: Text(
                solved ? 'Next puzzle' : 'Skip',
                style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
