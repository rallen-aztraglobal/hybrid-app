import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/game.dart';
import '../models/piece.dart';
import '../storage/best_score_store.dart';
import '../theme/app_colors.dart';
import '../widgets/board_view.dart';
import '../widgets/tray_view.dart';

/// 游戏主界面（A 面本体）。上方计分条、中间棋盘、下方待放区。
class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  final Game _game = Game();
  final BestScoreStore _store = const BestScoreStore();

  /// 棋盘格区的 RenderBox 入口。拖动时要把手指的全局坐标换算成棋盘格坐标，
  /// 只能靠它拿到棋盘在屏幕上的实际位置。
  final GlobalKey _boardKey = GlobalKey();

  int _best = 0;
  GhostPlacement? _ghost;

  /// 当前布局下的格边长，由 build 里的 LayoutBuilder 算出。
  /// onMove 回调发生在 build 之外，故存成字段供换算使用。
  double _cell = 0;

  @override
  void initState() {
    super.initState();
    _loadBest();
  }

  Future<void> _loadBest() async {
    final best = await _store.read();
    if (!mounted) return;
    setState(() => _best = best);
  }

  /// 棋盘底板的内边距，与 [BoardView] 里的算法保持一致。
  double get _boardPadding => _cell * 0.16;

  /// 把「浮层左上角的全局坐标」换算成它对应的棋盘格。
  ///
  /// 用浮层左上角而不是手指位置：浮层就是方块本身、左上角即方块的第 (0,0) 格，
  /// 换算出来的格子正是玩家眼睛看到的落点。`Draggable` 给的 `details.offset`
  /// 恰好就是浮层左上角的全局坐标（= 手指位置 − dragAnchorStrategy 的偏移）。
  Cell? _cellAt(Offset feedbackTopLeft) {
    final box = _boardKey.currentContext?.findRenderObject() as RenderBox?;
    if (box == null || _cell <= 0) return null;
    final local = box.globalToLocal(feedbackTopLeft);
    final inner = local - Offset(_boardPadding, _boardPadding);
    // round 而非 floor：让方块吸附到最近的格，手指不必精确压在格子左上角。
    return Cell((inner.dy / _cell).round(), (inner.dx / _cell).round());
  }

  void _onDragMove(DragTargetDetails<int> details) {
    final index = details.data;
    final piece = _game.tray[index];
    if (piece == null) return;

    final cell = _cellAt(details.offset);
    if (cell == null) return;

    final valid = _game.canPlace(index, cell.row, cell.col);
    final current = _ghost;
    // 落点与合法性都没变就不 setState —— 拖动过程中 onMove 每帧都触发，
    // 无脑刷新会让整块棋盘（64 格 + 预览）每帧重建。
    if (current != null &&
        current.topLeft == cell &&
        current.valid == valid &&
        identical(current.piece, piece)) {
      return;
    }
    setState(() {
      _ghost = GhostPlacement(piece: piece, topLeft: cell, valid: valid);
    });
  }

  void _onDrop(DragTargetDetails<int> details) {
    final index = details.data;
    final cell = _cellAt(details.offset);
    setState(() {
      _ghost = null;
      if (cell == null) return;

      final result = _game.place(index, cell.row, cell.col);
      if (result == null) return;

      if (result.linesCleared > 0) {
        HapticFeedback.mediumImpact();
      } else {
        HapticFeedback.selectionClick();
      }
      if (_game.score > _best) _best = _game.score;
    });

    // 落子后才可能刷新最高分；写盘不阻塞 UI。
    if (_game.score > 0) {
      // ignore: discarded_futures
      _store.writeIfHigher(_game.score);
    }
  }

  void _restart() {
    setState(() {
      _game.restart();
      _ghost = null;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            _cell = _cellSize(constraints);
            return Stack(
              children: <Widget>[
                _body(),
                if (_game.isOver)
                  _GameOverOverlay(
                    score: _game.score,
                    best: _best,
                    onRestart: _restart,
                  ),
              ],
            );
          },
        ),
      ),
    );
  }

  /// 由可用空间反推格边长，宽高两边各算一次取小的那个。
  ///
  /// 棋盘连底板共 8 + 2×0.16 = 8.32 格宽；待放区最高的形状是 5 连，
  /// 缩略图按 [_thumbRatio] 缩放后占 5 × 0.5 = 2.5 格，上下再各留半格气口。
  double _cellSize(BoxConstraints c) {
    const boardCells = 8 + 0.32;
    const trayCells = 5 * _thumbRatio + 1;
    const chrome = _headerHeight + _gap;

    final fromWidth = (c.maxWidth - _hPadding * 2) / boardCells;
    final fromHeight = (c.maxHeight - chrome) / (boardCells + trayCells);
    final cell = fromWidth < fromHeight ? fromWidth : fromHeight;
    // 极窄/极矮的约束下上面两式可能算出 0 或负数，负的边长会让 SizedBox 直接抛断言。
    // 夹到一个下限：这种尺寸下画面本来就没法看，但至少不崩。
    return cell < 1 ? 1 : cell;
  }

  /// 待放区的高度：最高的形状（5 连缩略图）再加上下各半格气口。
  double get _trayHeight => _cell * (5 * _thumbRatio + 1);

  static const double _thumbRatio = 0.5;
  static const double _headerHeight = 76;
  static const double _gap = 16;
  static const double _hPadding = 16;

  Widget _body() {
    // DragTarget 铺满整块区域而不是只盖住棋盘：浮层比手指高出 1.4 格，
    // 把方块拖到棋盘最下面一行时手指其实已经落到待放区上了。只盖棋盘的话
    // 那一行永远收不到 onMove、也放不下去。落点是否在棋盘内由 _cellAt + canPlace 判断。
    return DragTarget<int>(
      onWillAcceptWithDetails: (_) => true,
      onMove: _onDragMove,
      onLeave: (_) => setState(() => _ghost = null),
      onAcceptWithDetails: _onDrop,
      builder: (context, _, _) => Column(
        children: <Widget>[
          SizedBox(
            height: _headerHeight,
            child: _Header(
              score: _game.score,
              best: _best,
              onRestart: _restart,
            ),
          ),
          // 棋盘在「计分条」与「待放区」之间的剩余空间里垂直居中。
          // 高瘦屏上格边长是被宽度卡住的，竖直方向会多出可观的余量；
          // 若让棋盘贴着计分条、余量全丢给待放区，待放区就会孤零零地飘在一大片空白中间。
          // 居中后余量均分到棋盘上下，看起来是有意留白而不是没排好。
          Expanded(
            child: Center(
              child: BoardView(
                key: _boardKey,
                board: _game.board,
                cellSize: _cell,
                ghost: _ghost,
              ),
            ),
          ),
          SizedBox(
            height: _trayHeight,
            child: TrayView(
              pieces: _game.tray,
              isPlaceable: _game.isPlaceable,
              thumbCellSize: _cell * _thumbRatio,
              dragCellSize: _cell,
              // 拖动被取消（丢在界外、被系统打断）时 onAcceptWithDetails 不会触发，
              // 落点预览要在这里收掉，否则会一直挂在棋盘上。
              onDragEnd: () => setState(() => _ghost = null),
            ),
          ),
          const SizedBox(height: _gap),
        ],
      ),
    );
  }
}

/// 计分条：当前分、最高分、重开。
class _Header extends StatelessWidget {
  const _Header({
    required this.score,
    required this.best,
    required this.onRestart,
  });

  final int score;
  final int best;
  final VoidCallback onRestart;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20),
      child: Row(
        children: <Widget>[
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisAlignment: MainAxisAlignment.center,
            children: <Widget>[
              const _Label('SCORE'),
              Text(
                '$score',
                key: const ValueKey<String>('scoreText'),
                style: const TextStyle(
                  color: AppColors.primaryText,
                  fontSize: 32,
                  fontWeight: FontWeight.w600,
                  height: 1.1,
                ),
              ),
            ],
          ),
          const Spacer(),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            mainAxisAlignment: MainAxisAlignment.center,
            children: <Widget>[
              const _Label('BEST'),
              Text(
                '$best',
                key: const ValueKey<String>('bestText'),
                style: const TextStyle(
                  color: AppColors.accent,
                  fontSize: 22,
                  fontWeight: FontWeight.w600,
                  height: 1.1,
                ),
              ),
            ],
          ),
          const SizedBox(width: 16),
          IconButton(
            key: const ValueKey<String>('restartButton'),
            onPressed: onRestart,
            tooltip: 'New game',
            icon: const Icon(Icons.refresh_rounded),
            color: AppColors.mutedText,
          ),
        ],
      ),
    );
  }
}

class _Label extends StatelessWidget {
  const _Label(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: const TextStyle(
        color: AppColors.mutedText,
        fontSize: 11,
        letterSpacing: 1.6,
        fontWeight: FontWeight.w600,
      ),
    );
  }
}

/// 结束浮层。盖住棋盘但不挡计分条，让玩家看得到自己这局的成绩。
class _GameOverOverlay extends StatelessWidget {
  const _GameOverOverlay({
    required this.score,
    required this.best,
    required this.onRestart,
  });

  final int score;
  final int best;
  final VoidCallback onRestart;

  @override
  Widget build(BuildContext context) {
    return Positioned.fill(
      key: const ValueKey<String>('gameOverOverlay'),
      child: ColoredBox(
        color: AppColors.background.withValues(alpha: 0.88),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              const Text(
                'No room left',
                style: TextStyle(
                  color: AppColors.primaryText,
                  fontSize: 26,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                score >= best
                    ? 'New best · $score'
                    : 'Score $score · Best $best',
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 15,
                ),
              ),
              const SizedBox(height: 28),
              FilledButton(
                key: const ValueKey<String>('playAgainButton'),
                onPressed: onRestart,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  foregroundColor: AppColors.background,
                  padding: const EdgeInsets.symmetric(
                    horizontal: 34,
                    vertical: 14,
                  ),
                  shape: const StadiumBorder(),
                ),
                child: const Text(
                  'Play again',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
