import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/difficulty.dart';
import '../logic/game.dart';
import '../logic/puzzle_library.dart';
import '../storage/progress_store.dart';
import '../theme/app_colors.dart';
import '../widgets/board_view.dart';

/// A 面本体：游戏页。
class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  Difficulty _difficulty = Difficulty.normal;
  int _levelIndex = 0;
  late EdgeLoopGame _game = EdgeLoopGame(levelsFor(_difficulty).first);
  bool _solved = false;
  int _solvedCount = 0;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final level = await ProgressStore.currentLevel(_difficulty);
    final solved = await ProgressStore.solvedCount(_difficulty);
    if (!mounted) return;
    setState(() {
      _solvedCount = solved;
      _loadLevel(level);
    });
  }

  /// 载入某一关。下标一律按关卡数取模 —— 关卡库更新后关数可能变少，
  /// 存档里的旧下标直接用会越界崩溃。
  void _loadLevel(int index) {
    final levels = levelsFor(_difficulty);
    _levelIndex = index % levels.length;
    _game = EdgeLoopGame(levels[_levelIndex]);
    _solved = false;
    // ignore: discarded_futures
    ProgressStore.saveLevel(_difficulty, _levelIndex);
  }

  Future<void> _switchDifficulty(Difficulty d) async {
    final level = await ProgressStore.currentLevel(d);
    final solved = await ProgressStore.solvedCount(d);
    if (!mounted) return;
    setState(() {
      _difficulty = d;
      _solvedCount = solved;
      _loadLevel(level);
    });
  }

  void _onEdgeTapped(int edge) {
    if (_solved) return;
    setState(() {
      _game.cycle(edge);
      if (_game.isSolved) {
        _solved = true;
        HapticFeedback.heavyImpact();
        _solvedCount++;
        // ignore: discarded_futures
        ProgressStore.recordSolved(_difficulty);
      }
    });
  }

  void _next() => setState(() => _loadLevel(_levelIndex + 1));

  @override
  Widget build(BuildContext context) {
    final levels = levelsFor(_difficulty);
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _Header(
              difficulty: _difficulty,
              onChanged: _switchDifficulty,
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: Row(
                children: [
                  Text(
                    'Level ${_levelIndex + 1} / ${levels.length}',
                    style: const TextStyle(
                      color: AppColors.mutedText,
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    'Solved $_solvedCount',
                    style: const TextStyle(
                        color: AppColors.mutedText, fontSize: 13),
                  ),
                ],
              ),
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(10, 4, 10, 4),
                child: BoardView(
                  game: _game,
                  solved: _solved,
                  onEdgeTapped: _onEdgeTapped,
                ),
              ),
            ),
            _Footer(
              solved: _solved,
              onClear: () => setState(_game.clear),
              onNext: _next,
            ),
            const SizedBox(height: 12),
          ],
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.difficulty, required this.onChanged});

  final Difficulty difficulty;
  final ValueChanged<Difficulty> onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 14, 20, 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text(
            'EdgeLoop',
            style: TextStyle(
              color: AppColors.primaryText,
              fontSize: 24,
              fontWeight: FontWeight.w700,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: Difficulty.values.map((d) {
              final selected = d == difficulty;
              return Expanded(
                child: GestureDetector(
                  onTap: () => onChanged(d),
                  behavior: HitTestBehavior.opaque,
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 160),
                    margin: const EdgeInsets.symmetric(horizontal: 3),
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    decoration: BoxDecoration(
                      color: selected
                          ? AppColors.accent
                          : AppColors.boardSurface,
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(
                        color: selected
                            ? AppColors.accent
                            : AppColors.boardBorder,
                      ),
                    ),
                    child: Text(
                      d.label,
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                        color: selected
                            ? AppColors.boardSurface
                            : AppColors.mutedText,
                      ),
                    ),
                  ),
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }
}

class _Footer extends StatelessWidget {
  const _Footer({
    required this.solved,
    required this.onClear,
    required this.onNext,
  });

  final bool solved;
  final VoidCallback onClear;
  final VoidCallback onNext;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20),
      child: solved
          ? FilledButton(
              onPressed: onNext,
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.solved,
                foregroundColor: Colors.white,
                minimumSize: const Size.fromHeight(50),
              ),
              child: const Text('Next level',
                  style: TextStyle(fontWeight: FontWeight.w700)),
            )
          : Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: onClear,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: AppColors.mutedText,
                      side: const BorderSide(color: AppColors.boardBorder),
                      minimumSize: const Size.fromHeight(46),
                    ),
                    child: const Text('Clear'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: OutlinedButton(
                    onPressed: onNext,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: AppColors.accent,
                      side: const BorderSide(color: AppColors.boardBorder),
                      minimumSize: const Size.fromHeight(46),
                    ),
                    child: const Text('Skip'),
                  ),
                ),
              ],
            ),
    );
  }
}
