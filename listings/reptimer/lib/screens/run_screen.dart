import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import '../logic/interval_plan.dart';
import '../theme/app_colors.dart';

/// 计时页。
///
/// ## 为什么用「开始时刻 + 现在时刻」算进度，而不是每秒减一
///
/// 「每秒减一」的写法有两个绕不开的问题：一是 `Timer.periodic` 并不保证精确一秒，
/// 累计几百次之后会明显偏；二是 App 切到后台再回来，漏掉的那些 tick 补不回来，
/// 计时会凭空变慢。
///
/// 这里只记住「开始的时刻」，每次刷新用 `now - start` 反推当前状态 ——
/// tick 只负责触发重绘，本身不携带状态。于是漏几帧、卡一下、切后台再回来，
/// 显示的时间始终是对的。
///
/// ## 屏幕常亮
///
/// 练的时候没人会去戳屏幕，默认息屏时间一到就黑。这里在计时期间打开 wakelock，
/// 离开页面时**一定**关掉 —— 忘了关会让手机一直亮着耗电，属于用户不会立刻发现、
/// 发现了会很生气的那类 bug。
class RunScreen extends StatefulWidget {
  const RunScreen({super.key, required this.plan});

  final IntervalPlan plan;

  @override
  State<RunScreen> createState() => _RunScreenState();
}

class _RunScreenState extends State<RunScreen>
    with SingleTickerProviderStateMixin {
  late final Ticker _ticker = createTicker(_onTick);

  /// 已累计的秒数来源：暂停前累计的 + 本次运行至今。
  Duration _accumulated = Duration.zero;
  DateTime? _runningSince;

  Phase _lastPhase = Phase.prepare;

  @override
  void initState() {
    super.initState();
    _start();
  }

  @override
  void dispose() {
    _ticker.dispose();
    // 无论怎么离开这一页（返回、被系统回收、异常），都要把常亮关掉。
    // ignore: discarded_futures
    WakelockPlus.disable();
    super.dispose();
  }

  int get _elapsedSeconds {
    var total = _accumulated;
    final since = _runningSince;
    if (since != null) total += DateTime.now().difference(since);
    return total.inSeconds;
  }

  bool get _isRunning => _runningSince != null;

  void _start() {
    setState(() => _runningSince = DateTime.now());
    if (!_ticker.isActive) _ticker.start();
    // ignore: discarded_futures
    WakelockPlus.enable();
  }

  void _pause() {
    final since = _runningSince;
    if (since == null) return;
    setState(() {
      _accumulated += DateTime.now().difference(since);
      _runningSince = null;
    });
    _ticker.stop();
    // 暂停时放掉常亮：暂停可能是去做别的事，没必要一直亮着。
    // ignore: discarded_futures
    WakelockPlus.disable();
  }

  void _reset() {
    setState(() {
      _accumulated = Duration.zero;
      _runningSince = null;
      _lastPhase = Phase.prepare;
    });
    _ticker.stop();
    // ignore: discarded_futures
    WakelockPlus.disable();
  }

  void _onTick(Duration _) {
    final snap = widget.plan.snapshotAt(_elapsedSeconds);
    // 阶段切换时给一次较重的震动 —— 练的时候眼睛未必在屏幕上，
    // 「该换了」这个信号必须能靠手感接收到。
    if (snap.phase != _lastPhase) {
      _lastPhase = snap.phase;
      if (snap.isDone) {
        HapticFeedback.heavyImpact();
        _pause();
      } else {
        HapticFeedback.mediumImpact();
      }
    }
    setState(() {});
  }

  Color _phaseColor(Phase p) => switch (p) {
        Phase.prepare => AppColors.prepare,
        Phase.work => AppColors.work,
        Phase.rest => AppColors.rest,
        Phase.cycleRest => AppColors.cycleRest,
        Phase.done => AppColors.done,
      };

  @override
  Widget build(BuildContext context) {
    final snap = widget.plan.snapshotAt(_elapsedSeconds);
    final color = _phaseColor(snap.phase);

    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        foregroundColor: AppColors.primaryText,
        elevation: 0,
        title: Text(widget.plan.name,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
      ),
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    // 阶段名用颜色而不只是文字 —— 余光扫一眼就要能分辨做还是休。
                    AnimatedContainer(
                      duration: const Duration(milliseconds: 220),
                      padding: const EdgeInsets.symmetric(
                          horizontal: 18, vertical: 7),
                      decoration: BoxDecoration(
                        color: color.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        snap.phase.label.toUpperCase(),
                        style: TextStyle(
                          color: color,
                          fontSize: 14,
                          fontWeight: FontWeight.w800,
                          letterSpacing: 1.4,
                        ),
                      ),
                    ),
                    const SizedBox(height: 18),
                    // 大数字：架在地上也要看得清，所以尽可能占满宽度。
                    FittedBox(
                      fit: BoxFit.scaleDown,
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 24),
                        child: Text(
                          snap.isDone
                              ? 'Done'
                              : formatSeconds(snap.remainingInPhase),
                          style: TextStyle(
                            color: color,
                            fontSize: 108,
                            height: 1.0,
                            fontWeight: FontWeight.w800,
                            fontFeatures: const [FontFeature.tabularFigures()],
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(height: 14),
                    Text(
                      snap.isDone
                          ? '${widget.plan.totalRounds} rounds completed'
                          : 'Round ${snap.round == 0 ? "–" : snap.round} / ${widget.plan.rounds}'
                              '${widget.plan.cycles > 1 ? "   ·   Cycle ${snap.cycle} / ${widget.plan.cycles}" : ""}',
                      style: const TextStyle(
                          color: AppColors.mutedText,
                          fontSize: 14,
                          fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      '${formatSeconds(snap.totalRemaining)} left',
                      style: const TextStyle(
                          color: AppColors.mutedText, fontSize: 12.5),
                    ),
                  ],
                ),
              ),
            ),
            _Controls(
              isRunning: _isRunning,
              isDone: snap.isDone,
              onPlayPause: _isRunning ? _pause : _start,
              onReset: _reset,
            ),
            const SizedBox(height: 18),
          ],
        ),
      ),
    );
  }
}

class _Controls extends StatelessWidget {
  const _Controls({
    required this.isRunning,
    required this.isDone,
    required this.onPlayPause,
    required this.onReset,
  });

  final bool isRunning;
  final bool isDone;
  final VoidCallback onPlayPause;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        // 控件做得很大：练的时候手在抖、可能戴着手套，小按钮点不中。
        _CircleButton(
          icon: Icons.refresh,
          size: 58,
          color: AppColors.surfaceHigh,
          iconColor: AppColors.mutedText,
          onTap: onReset,
        ),
        const SizedBox(width: 22),
        _CircleButton(
          icon: isDone
              ? Icons.replay
              : (isRunning ? Icons.pause_rounded : Icons.play_arrow_rounded),
          size: 86,
          color: AppColors.accent,
          iconColor: AppColors.background,
          onTap: isDone ? onReset : onPlayPause,
        ),
      ],
    );
  }
}

class _CircleButton extends StatelessWidget {
  const _CircleButton({
    required this.icon,
    required this.size,
    required this.color,
    required this.iconColor,
    required this.onTap,
  });

  final IconData icon;
  final double size;
  final Color color;
  final Color iconColor;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: Container(
          width: size,
          height: size,
          alignment: Alignment.center,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          child: Icon(icon, color: iconColor, size: size * 0.46),
        ),
      );
}
