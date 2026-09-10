import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/game.dart';
import '../logic/generator.dart';
import '../logic/picture_library.dart';
import '../storage/progress_store.dart';
import '../theme/app_colors.dart';
import '../widgets/board_view.dart';

/// A 面：数织本体。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  PuzzleSize _size = PuzzleSize.small;

  /// 当前这道题画的是什么。解开后显示名字 —— 那是数织的回报所在。
  late Picture _picture = dealPicture(_size);
  late NonogramGame _game = NonogramGame(_picture.toNonogram());

  /// 当前笔的含义。数织常态是涂黑，打叉是辅助，所以默认涂黑。
  CellMark _intent = CellMark.filled;

  int _solved = 0;
  bool _celebrating = false;

  @override
  void initState() {
    super.initState();
    _loadSolved();
  }

  Future<void> _loadSolved() async {
    final n = await ProgressStore.solvedCount(_size);
    if (mounted) setState(() => _solved = n);
  }

  void _newPuzzle({PuzzleSize? size}) {
    setState(() {
      final changingSize = size != null && size != _size;
      if (size != null) _size = size;
      // 换尺寸时不必避让（图库本来就不同）；同尺寸下一题要避开刚解完的那张，
      // 连着两次同一幅画很出戏。
      _picture = dealPicture(_size, avoid: changingSize ? null : _picture);
      _game = NonogramGame(_picture.toNonogram());
      _celebrating = false;
    });
    _loadSolved();
  }

  Future<void> _onChanged() async {
    setState(() {});
    if (_game.isSolved && !_celebrating) {
      setState(() => _celebrating = true);
      await HapticFeedback.mediumImpact();
      final n = await ProgressStore.recordSolved(_size);
      if (mounted) setState(() => _solved = n);
    }
  }

  Future<void> _onMistake() => HapticFeedback.selectionClick();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      // 背景走一道极淡的渐变。棋盘是正方形，竖屏下上下必然留白，
      // 一整块纯色会让那片留白显得像没做完；一点点色温变化就够了。
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
            colors: <Color>[
              AppColors.backgroundTint,
              AppColors.background,
            ],
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
              // 棋盘 + 进度条作为一个整体，在标题与底部控件之间垂直居中。
              //
              // 试过两种都不行：紧贴标题会让下半屏空着、视觉重心偏上；
              // 用 Spacer 把控件推到底则把空白挪到进度条下面，中间断开一大截。
              // 居中之后富余高度平分到上下，屏幕多高都不塌。
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
                          picture: _picture,
                          intent: _intent,
                          onChanged: _onChanged,
                          onMistake: _onMistake,
                        ),
                      ),
                      const SizedBox(height: 22),
                      _progress(),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              _penToggle(),
              const SizedBox(height: 10),
              _secondaryAction(),
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
      crossAxisAlignment: CrossAxisAlignment.center,
      children: <Widget>[
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              const Text(
                'NonoPix',
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
        _sizeSwitch(),
      ],
    );
  }

  /// 尺寸切换做成一个整体的分段控件，而不是两个各自独立的按钮 ——
  /// 后者看起来像两个操作，实际是二选一。
  Widget _sizeSwitch() {
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: AppColors.boardSurface,
        borderRadius: BorderRadius.circular(11),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          for (final size in PuzzleSize.values)
            GestureDetector(
              onTap: size == _size ? null : () => _newPuzzle(size: size),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 140),
                padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 7),
                decoration: BoxDecoration(
                  color: size == _size ? AppColors.accent : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  size.label,
                  style: TextStyle(
                    color: size == _size ? Colors.white : AppColors.mutedText,
                    fontSize: 12.5,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  /// 进度条 + 计数。原来只有角落一行 `0 / 15`，又小又没存在感，
  /// 而这是玩家最常瞟的一眼。
  Widget _progress() {
    final total = _game.puzzle.filledCount;
    final done = _game.correctCount;
    final ratio = total == 0 ? 1.0 : done / total;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Row(
          children: <Widget>[
            // 解开后把画的名字亮出来 —— 这是整局的回报，
            // 比继续显示「Progress」有意义得多。
            Text(
              _celebrating ? 'Solved · ${_picture.name}' : 'Progress',
              style: TextStyle(
                color: _celebrating ? AppColors.accent : AppColors.mutedText,
                fontSize: 12.5,
                fontWeight: _celebrating ? FontWeight.w700 : FontWeight.w600,
                letterSpacing: 0.2,
              ),
            ),
            const Spacer(),
            if (_game.wrongCount > 0) ...<Widget>[
              Text(
                '${_game.wrongCount} wrong',
                style: const TextStyle(
                  color: AppColors.mistake,
                  fontSize: 12.5,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(width: 10),
            ],
            Text(
              '$done / $total',
              style: const TextStyle(
                color: AppColors.primaryText,
                fontSize: 12.5,
                fontWeight: FontWeight.w700,
                fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: TweenAnimationBuilder<double>(
            tween: Tween<double>(begin: 0, end: ratio),
            duration: const Duration(milliseconds: 220),
            curve: Curves.easeOut,
            builder: (context, value, _) => LinearProgressIndicator(
              value: value,
              minHeight: 5,
              backgroundColor: AppColors.emptyCell,
              valueColor: AlwaysStoppedAnimation<Color>(
                _celebrating ? AppColors.accent : AppColors.filledCell,
              ),
            ),
          ),
        ),
      ],
    );
  }

  /// 笔的模式：涂 / 叉。做成分段控件，一眼看出是二选一而不是两个按钮。
  Widget _penToggle() {
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: AppColors.boardSurface,
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: <Widget>[
          _penOption(
            label: 'Fill',
            icon: Icons.square_rounded,
            mark: CellMark.filled,
          ),
          _penOption(
            label: 'Cross',
            icon: Icons.close_rounded,
            mark: CellMark.crossed,
          ),
        ],
      ),
    );
  }

  Widget _penOption({
    required String label,
    required IconData icon,
    required CellMark mark,
  }) {
    final selected = _intent == mark;
    return Expanded(
      child: GestureDetector(
        onTap: () => setState(() => _intent = mark),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 140),
          height: 46,
          decoration: BoxDecoration(
            color: selected ? AppColors.accent : Colors.transparent,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: <Widget>[
              Icon(
                icon,
                size: 17,
                color: selected ? Colors.white : AppColors.mutedText,
              ),
              const SizedBox(width: 7),
              Text(
                label,
                style: TextStyle(
                  color: selected ? Colors.white : AppColors.mutedText,
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// 次要操作：未通关时是「重来」，通关后变成「下一题」。
  /// 用文字按钮而不是实心块，避免和上面的模式开关抢注意力。
  Widget _secondaryAction() {
    final solved = _celebrating;
    return SizedBox(
      height: 42,
      child: TextButton.icon(
        onPressed: () {
          if (solved) {
            _newPuzzle();
          } else {
            setState(_game.reset);
          }
        },
        style: TextButton.styleFrom(
          foregroundColor: solved ? AppColors.accent : AppColors.mutedText,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(10),
          ),
        ),
        icon: Icon(solved ? Icons.arrow_forward_rounded : Icons.refresh_rounded,
            size: 18),
        label: Text(
          solved ? 'Next puzzle' : 'Clear board',
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
        ),
      ),
    );
  }
}
