import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';

import '../logic/game.dart';
import '../theme/app_colors.dart';

/// 雷区：网格 + 数字 + 旗子 + 雷。整块用一层 CustomPaint 画。
///
/// 为什么不用 widget 树：最大档 11×16 是 176 格，做成 widget 每次翻开都要重建
/// 一大片；一次空格连锁展开会同时改动几十格，掉帧看得出来。
///
/// 命中判定与绘制共用同一套几何量（[_Geometry]），
/// 所以「看到的格子」和「点到的格子」不会错位。
class BoardView extends StatefulWidget {
  const BoardView({
    super.key,
    required this.game,
    required this.mode,
    required this.onChanged,
  });

  final MineGame game;

  /// 当前点击含义。长按永远执行相反的那个动作。
  final InputMode mode;

  final VoidCallback onChanged;

  @override
  State<BoardView> createState() => _BoardViewState();
}

class _BoardViewState extends State<BoardView>
    with SingleTickerProviderStateMixin {
  /// 单格掀开的时长。
  static const Duration _revealDuration = Duration(milliseconds: 170);

  /// 连锁展开时，每远离一圈就晚起跑这么久。
  ///
  /// 这是整个界面里最值得花的一笔：一次连锁能翻开几十格，同时闪出来是一片
  /// 突兀的白，从落指处一圈圈荡开则能让人看清「刚才发生了什么」。
  static const Duration _ringDelay = Duration(milliseconds: 20);

  /// 延迟的圈数上限。不封顶的话，大盘上一次全展开要等一秒多才停。
  static const int _maxRings = 12;

  static const Duration _flagDuration = Duration(milliseconds: 180);
  static const Duration _mineDuration = Duration(milliseconds: 240);

  late final Ticker _ticker = createTicker(_onTick);

  /// 单调递增的动画时钟。Ticker 每次 start 的 elapsed 都从 0 起，
  /// 所以停表时把当前值存进 [_base]，重启后接着往上加。
  Duration _base = Duration.zero;
  Duration _now = Duration.zero;

  /// 各类动画的起跑时刻。查不到 = 这件事早就完成了（进度直接算满）。
  final Map<int, Duration> _revealAt = <int, Duration>{};
  final Map<int, Duration> _flagAt = <int, Duration>{};
  final Map<int, Duration> _mineAt = <int, Duration>{};

  /// 所有动画的收尾时刻，到了就停表 —— 空闲时不该每帧都唤醒。
  Duration _animEnd = Duration.zero;

  late List<bool> _seenRevealed;
  late List<bool> _seenFlagged;
  bool _seenLost = false;

  /// 手指正按着的格子，用来给一点即时的按下反馈。
  int? _pressed;

  @override
  void initState() {
    super.initState();
    _resetTracking();
  }

  @override
  void didUpdateWidget(BoardView oldWidget) {
    super.didUpdateWidget(oldWidget);
    // 换了一局（新开 / 换难度）→ 动画状态全部作废
    if (!identical(oldWidget.game, widget.game)) {
      setState(_resetTracking);
    }
  }

  @override
  void dispose() {
    _ticker.dispose();
    super.dispose();
  }

  void _resetTracking() {
    _revealAt.clear();
    _flagAt.clear();
    _mineAt.clear();
    _seenRevealed = List<bool>.filled(widget.game.cellCount, false);
    _seenFlagged = List<bool>.filled(widget.game.cellCount, false);
    _seenLost = false;
    _animEnd = _now;
    _pressed = null;
  }

  void _onTick(Duration elapsed) {
    setState(() => _now = _base + elapsed);
    if (_now >= _animEnd) {
      _base = _now;
      _ticker.stop();
    }
  }

  void _extend(Duration end) {
    if (end > _animEnd) _animEnd = end;
  }

  /// 棋盘上的「圈数」距离（切比雪夫距离）。
  /// 用它而不是欧氏距离，是因为连锁展开本来就是按八邻一圈圈铺出去的。
  int _rings(int a, int b) {
    final cols = widget.game.difficulty.cols;
    final dr = (a ~/ cols - b ~/ cols).abs();
    final dc = (a % cols - b % cols).abs();
    return max(dr, dc);
  }

  /// 比对上一帧与当前局面，给新出现的变化排上动画。
  void _schedule(int? origin) {
    final game = widget.game;
    final won = game.status == GameStatus.won;
    final centre = game.cellCount ~/ 2;
    var scheduled = false;

    for (var i = 0; i < game.cellCount; i++) {
      if (game.isRevealed(i) && !_seenRevealed[i]) {
        _seenRevealed[i] = true;
        final rings = origin == null ? 0 : min(_rings(origin, i), _maxRings);
        final start = _now + _ringDelay * rings;
        _revealAt[i] = start;
        _extend(start + _revealDuration);
        scheduled = true;
      }

      final flagged = game.isFlagged(i);
      if (flagged && !_seenFlagged[i]) {
        _seenFlagged[i] = true;
        // 通关时系统会把剩下的旗一次补齐 —— 让它们从盘面中心荡开，
        // 比几十面旗同时啪一下出现体面得多。
        final rings = won ? min(_rings(centre, i), _maxRings) : 0;
        final start = _now + _ringDelay * rings;
        _flagAt[i] = start;
        _extend(start + _flagDuration);
        scheduled = true;
      } else if (!flagged && _seenFlagged[i]) {
        _seenFlagged[i] = false;
        _flagAt.remove(i);
      }
    }

    if (game.status == GameStatus.lost && !_seenLost) {
      _seenLost = true;
      final boom = game.explodedIndex ?? 0;
      for (var i = 0; i < game.cellCount; i++) {
        if (!game.isMineAt(i) || game.isFlagged(i)) continue;
        // 其余的雷从爆点向外一圈圈亮出来
        final start = _now + _ringDelay * min(_rings(boom, i), _maxRings);
        _mineAt[i] = start;
        _extend(start + _mineDuration);
      }
      scheduled = true;
    }

    if (scheduled && !_ticker.isActive) _ticker.start();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final size = constraints.biggest;
        final geo = _Geometry(
          cols: widget.game.difficulty.cols,
          rows: widget.game.difficulty.rows,
          size: size,
        );
        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTapDown: (d) => _press(geo.indexAt(d.localPosition)),
          onTapCancel: () => _press(null),
          onTapUp: (d) {
            _press(null);
            _handle(geo, d.localPosition, longPress: false);
          },
          onLongPressStart: (d) {
            _press(null);
            _handle(geo, d.localPosition, longPress: true);
          },
          child: CustomPaint(
            size: size,
            painter: _BoardPainter(
              game: widget.game,
              now: _now,
              revealAt: _revealAt,
              flagAt: _flagAt,
              mineAt: _mineAt,
              pressed: _pressed,
              revealDuration: _revealDuration,
              flagDuration: _flagDuration,
              mineDuration: _mineDuration,
            ),
          ),
        );
      },
    );
  }

  void _press(int? index) {
    if (_pressed == index) return;
    // 只有还盖着的格子才给按下反馈 —— 已翻开的格子按下去也没得挖
    setState(() =>
        _pressed = index != null && !widget.game.isRevealed(index) ? index : null);
  }

  void _handle(_Geometry geo, Offset local, {required bool longPress}) {
    final index = geo.indexAt(local);
    if (index == null) return;
    final game = widget.game;

    // 点已翻开的数字 = 和弦。这是最常用的操作，不该藏在某个模式里，
    // 所以两种模式下都直接生效。
    if (game.isRevealed(index)) {
      if (game.chord(index)) {
        HapticFeedback.selectionClick();
        _schedule(index);
        widget.onChanged();
      }
      return;
    }

    final dig = longPress ? widget.mode == InputMode.flag : widget.mode == InputMode.dig;
    final changed = dig ? game.dig(index) : game.toggleFlag(index);
    if (!changed) return;

    // 触感分两级：挖是轻点一下，插旗重一点 —— 插旗是「我确定这里有雷」的判断，
    // 手上该有个更实的确认。
    if (dig) {
      HapticFeedback.selectionClick();
    } else {
      HapticFeedback.lightImpact();
    }
    _schedule(index);
    widget.onChanged();
  }
}

/// 棋盘几何。绘制与命中判定共用。
class _Geometry {
  _Geometry({required this.cols, required this.rows, required this.size}) {
    final pad = size.shortestSide * 0.02;
    cell = ((size.width - 2 * pad) / cols)
        .clamp(0.0, (size.height - 2 * pad) / rows);
    originX = (size.width - cell * cols) / 2;
    originY = (size.height - cell * rows) / 2;
  }

  final int cols;
  final int rows;
  final Size size;
  late final double cell;
  late final double originX;
  late final double originY;

  Rect rectOf(int index) => Rect.fromLTWH(
        originX + (index % cols) * cell,
        originY + (index ~/ cols) * cell,
        cell,
        cell,
      );

  int? indexAt(Offset p) {
    if (cell <= 0) return null;
    final c = ((p.dx - originX) / cell).floor();
    final r = ((p.dy - originY) / cell).floor();
    if (r < 0 || r >= rows || c < 0 || c >= cols) return null;
    return r * cols + c;
  }
}

class _BoardPainter extends CustomPainter {
  _BoardPainter({
    required this.game,
    required this.now,
    required this.revealAt,
    required this.flagAt,
    required this.mineAt,
    required this.pressed,
    required this.revealDuration,
    required this.flagDuration,
    required this.mineDuration,
  });

  final MineGame game;
  final Duration now;
  final Map<int, Duration> revealAt;
  final Map<int, Duration> flagAt;
  final Map<int, Duration> mineAt;
  final int? pressed;
  final Duration revealDuration;
  final Duration flagDuration;
  final Duration mineDuration;

  /// 数字的 TextPainter 缓存。
  ///
  /// 一帧要画上百个数字，每个都现 layout 一次是实打实的开销；
  /// 而实际只有 8 种数字、一种字号，缓存命中率接近 100%。
  static final Map<String, TextPainter> _textCache = <String, TextPainter>{};

  /// 某件事的进度 0~1。表里没有记录 = 早就完成了。
  double _progress(Map<int, Duration> starts, int index, Duration duration) {
    final start = starts[index];
    if (start == null) return 1;
    final t = (now - start).inMicroseconds / duration.inMicroseconds;
    return t.clamp(0.0, 1.0);
  }

  @override
  void paint(Canvas canvas, Size size) {
    final geo = _Geometry(
      cols: game.difficulty.cols,
      rows: game.difficulty.rows,
      size: size,
    );

    final rr = RRect.fromRectAndRadius(
      Offset.zero & size,
      const Radius.circular(18),
    );
    canvas.drawRRect(rr, Paint()..color = AppColors.boardSurface);
    canvas.drawRRect(
      rr,
      Paint()
        ..color = AppColors.boardBorder
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.4,
    );

    final lost = game.status == GameStatus.lost;
    final gap = geo.cell * 0.06;
    final radius = Radius.circular((geo.cell * 0.2).clamp(2.0, 8.0));

    for (var i = 0; i < game.cellCount; i++) {
      final rect = geo.rectOf(i).deflate(gap);

      if (game.isRevealed(i)) {
        _paintRevealed(canvas, rect, radius, i);
        continue;
      }

      // 败局：把没插旗的雷都亮出来，踩到的那颗单独标红
      if (lost && game.isMineAt(i) && !game.isFlagged(i)) {
        _paintMineCell(canvas, rect, radius, i);
        continue;
      }

      _paintCovered(canvas, rect, radius, alpha: 1, scale: 1, dim: i == pressed);
      if (game.isFlagged(i)) {
        final p = Curves.easeOutBack.transform(
          _progress(flagAt, i, flagDuration),
        );
        _paintFlag(canvas, rect, p, wrong: lost && !game.isMineAt(i));
      }
    }
  }

  /// 已翻开的格子：底下是坑和数字，上面盖着一块正在退场的「盖板」。
  void _paintRevealed(Canvas canvas, Rect rect, Radius radius, int index) {
    final p = _progress(revealAt, index, revealDuration);
    final eased = Curves.easeOutCubic.transform(p);

    canvas.drawRRect(
      RRect.fromRectAndRadius(rect, radius),
      Paint()..color = AppColors.revealedCell,
    );

    final value = game.valueAt(index) ?? 0;
    if (value > 0 && p > 0) {
      // 数字随盖板掀开一起长出来
      canvas.save();
      canvas.translate(rect.center.dx, rect.center.dy);
      canvas.scale(0.72 + 0.28 * eased);
      canvas.translate(-rect.center.dx, -rect.center.dy);
      _paintNumber(canvas, rect, value);
      canvas.restore();
    }

    if (p < 1) {
      // 盖板一边缩一边淡出，像被掀掉
      final shrink = rect.width * 0.14 * eased;
      _paintCovered(
        canvas,
        rect.deflate(shrink),
        radius,
        alpha: 1 - eased,
        scale: 1,
        dim: false,
      );
    }
  }

  void _paintCovered(
    Canvas canvas,
    Rect rect,
    Radius radius, {
    required double alpha,
    required double scale,
    required bool dim,
  }) {
    if (alpha <= 0) return;
    final body = (dim ? AppColors.coveredCellTop : AppColors.coveredCell)
        .withValues(alpha: alpha);
    final top = (dim ? AppColors.coveredCell : AppColors.coveredCellTop)
        .withValues(alpha: alpha);

    canvas.drawRRect(RRect.fromRectAndRadius(rect, radius), Paint()..color = body);
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromLTWH(rect.left, rect.top, rect.width, rect.height * 0.42),
        radius,
      ),
      Paint()..color = top,
    );
  }

  void _paintMineCell(Canvas canvas, Rect rect, Radius radius, int index) {
    final p = Curves.easeOutBack.transform(_progress(mineAt, index, mineDuration));
    final exploded = index == game.explodedIndex;

    canvas.drawRRect(
      RRect.fromRectAndRadius(rect, radius),
      Paint()
        ..color = exploded
            ? AppColors.explosion
            : Color.lerp(AppColors.coveredCell, AppColors.revealedCell, p)!,
    );

    if (p <= 0) return;
    canvas.save();
    canvas.translate(rect.center.dx, rect.center.dy);
    canvas.scale(p.clamp(0.0, 1.2));
    canvas.translate(-rect.center.dx, -rect.center.dy);
    _paintMine(canvas, rect, exploded ? Colors.white : AppColors.mine);
    canvas.restore();
  }

  void _paintNumber(Canvas canvas, Rect rect, int value) {
    final fontSize = rect.height * 0.56;
    final key = '$value@${fontSize.toStringAsFixed(1)}';
    final painter = _textCache.putIfAbsent(key, () {
      final tp = TextPainter(
        text: TextSpan(
          text: '$value',
          style: TextStyle(
            color: AppColors.number(value),
            fontSize: fontSize,
            fontWeight: FontWeight.w800,
            height: 1.0,
          ),
        ),
        textDirection: TextDirection.ltr,
      );
      tp.layout();
      return tp;
    });
    painter.paint(
      canvas,
      rect.center - Offset(painter.width / 2, painter.height / 2),
    );
  }

  void _paintFlag(Canvas canvas, Rect rect, double scale, {required bool wrong}) {
    if (scale <= 0) return;
    final color = wrong ? AppColors.mutedText : AppColors.accent;
    final w = rect.width;
    final cx = rect.center.dx;
    final cy = rect.center.dy;

    canvas.save();
    canvas.translate(cx, cy);
    canvas.scale(scale);
    canvas.translate(-cx, -cy);

    // 旗杆
    canvas.drawLine(
      Offset(cx - w * 0.02, cy - w * 0.26),
      Offset(cx - w * 0.02, cy + w * 0.26),
      Paint()
        ..color = color
        ..strokeWidth = w * 0.07
        ..strokeCap = StrokeCap.round,
    );
    // 旗面
    final flag = Path()
      ..moveTo(cx - w * 0.02, cy - w * 0.28)
      ..lineTo(cx + w * 0.24, cy - w * 0.14)
      ..lineTo(cx - w * 0.02, cy)
      ..close();
    canvas.drawPath(flag, Paint()..color = color);

    if (wrong) {
      final p = Paint()
        ..color = AppColors.explosion
        ..strokeWidth = w * 0.07
        ..strokeCap = StrokeCap.round;
      canvas.drawLine(
        Offset(cx - w * 0.26, cy - w * 0.26),
        Offset(cx + w * 0.26, cy + w * 0.26),
        p,
      );
      canvas.drawLine(
        Offset(cx + w * 0.26, cy - w * 0.26),
        Offset(cx - w * 0.26, cy + w * 0.26),
        p,
      );
    }

    canvas.restore();
  }

  void _paintMine(Canvas canvas, Rect rect, Color color) {
    final r = rect.width * 0.2;
    canvas.drawCircle(rect.center, r, Paint()..color = color);
    final spoke = Paint()
      ..color = color
      ..strokeWidth = rect.width * 0.07
      ..strokeCap = StrokeCap.round;
    for (var k = 0; k < 4; k++) {
      final angle = k * pi / 4;
      final d = Offset(cos(angle), sin(angle)) * (r * 1.75);
      canvas.drawLine(rect.center - d, rect.center + d, spoke);
    }
  }

  @override
  bool shouldRepaint(covariant _BoardPainter old) => true;
}
