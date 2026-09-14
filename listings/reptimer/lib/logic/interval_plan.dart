/// 一轮里的阶段。
enum Phase {
  /// 开始前的准备倒数。
  prepare('Get ready'),

  /// 做。
  work('Work'),

  /// 组间休息。
  rest('Rest'),

  /// 大循环之间的长休息。
  cycleRest('Cycle rest'),

  /// 全部跑完。
  done('Done');

  const Phase(this.label);
  final String label;
}

/// 时间轴上的一段。
class Segment {
  const Segment({
    required this.phase,
    required this.seconds,
    required this.round,
    required this.cycle,
  });

  final Phase phase;
  final int seconds;

  /// 第几组（从 1 起）。准备段与大循环休息段为 0。
  final int round;

  /// 第几个大循环（从 1 起）。
  final int cycle;
}

/// 此刻的状态：处在哪一段、这一段还剩几秒。
class TimerSnapshot {
  const TimerSnapshot({
    required this.phase,
    required this.remainingInPhase,
    required this.round,
    required this.cycle,
    required this.totalRemaining,
  });

  final Phase phase;
  final int remainingInPhase;
  final int round;
  final int cycle;
  final int totalRemaining;

  bool get isDone => phase == Phase.done;
}

/// 一份间隔计时方案。
///
/// 结构：`准备 → [ (做 → 休) × rounds ] × cycles`，大循环之间插入长休息。
///
/// ## 两个必须处理的退化情形
///
/// **1. 时长为 0 的阶段必须整段跳过，而不是「显示 0 秒然后跳走」。**
/// 用户把休息设成 0（连续做组）是很常见的用法。若保留一个 0 秒的段，
/// 界面会闪一下「Rest 0」再跳走 —— 看着像 bug，而且如果实现里用
/// 「每秒减一，减到 0 就切换」的写法，0 秒段会让它卡死或者多跑一秒。
/// 这里在**构建时间轴时**就把 0 秒段剔除，运行时根本不存在这种段。
///
/// **2. 最后一组之后不留休息。**
/// 练完最后一组还倒数一段休息才结束，是这类 App 最常见的设计错误 ——
/// 用户已经结束了，却被迫等一段无意义的倒数。同理，最后一个大循环之后
/// 也不留大循环休息。
class IntervalPlan {
  const IntervalPlan({
    required this.name,
    required this.prepareSeconds,
    required this.workSeconds,
    required this.restSeconds,
    required this.rounds,
    this.cycles = 1,
    this.cycleRestSeconds = 0,
  });

  /// 方案名 —— **用户自由输入的文字**。
  /// 这是本 App 唯一会存下的自由文本，隐私政策与 Data safety 据此单独写明。
  final String name;

  final int prepareSeconds;
  final int workSeconds;
  final int restSeconds;

  /// 每个大循环里做几组。
  final int rounds;

  /// 大循环数。
  final int cycles;

  /// 大循环之间的长休息。
  final int cycleRestSeconds;

  IntervalPlan copyWith({
    String? name,
    int? prepareSeconds,
    int? workSeconds,
    int? restSeconds,
    int? rounds,
    int? cycles,
    int? cycleRestSeconds,
  }) =>
      IntervalPlan(
        name: name ?? this.name,
        prepareSeconds: prepareSeconds ?? this.prepareSeconds,
        workSeconds: workSeconds ?? this.workSeconds,
        restSeconds: restSeconds ?? this.restSeconds,
        rounds: rounds ?? this.rounds,
        cycles: cycles ?? this.cycles,
        cycleRestSeconds: cycleRestSeconds ?? this.cycleRestSeconds,
      );

  /// 方案是否可以开始 —— 做的时长必须 >0，组数必须 >0。
  ///
  /// 其余都可以是 0（不准备、不休息、单循环）。
  bool get isRunnable => workSeconds > 0 && rounds > 0 && cycles > 0;

  /// 展开成一条时间轴。0 秒的段在这里就被剔除了。
  List<Segment> buildTimeline() {
    final out = <Segment>[];
    if (!isRunnable) return out;

    if (prepareSeconds > 0) {
      out.add(Segment(
        phase: Phase.prepare,
        seconds: prepareSeconds,
        round: 0,
        cycle: 1,
      ));
    }

    for (var cy = 1; cy <= cycles; cy++) {
      for (var r = 1; r <= rounds; r++) {
        out.add(Segment(
          phase: Phase.work,
          seconds: workSeconds,
          round: r,
          cycle: cy,
        ));
        // 最后一组之后不休息 —— 练完了就是练完了。
        final isLastRound = r == rounds;
        if (!isLastRound && restSeconds > 0) {
          out.add(Segment(
            phase: Phase.rest,
            seconds: restSeconds,
            round: r,
            cycle: cy,
          ));
        }
      }
      // 最后一个大循环之后不留大循环休息。
      final isLastCycle = cy == cycles;
      if (!isLastCycle && cycleRestSeconds > 0) {
        out.add(Segment(
          phase: Phase.cycleRest,
          seconds: cycleRestSeconds,
          round: 0,
          cycle: cy,
        ));
      }
    }
    return out;
  }

  /// 总时长（秒）。
  int get totalSeconds =>
      buildTimeline().fold<int>(0, (sum, s) => sum + s.seconds);

  /// 已过去 [elapsed] 秒时的状态。
  ///
  /// [elapsed] 允许超过总时长 —— 返回 [Phase.done]，而不是抛异常或越界：
  /// 计时器的 tick 与界面重建之间总有几毫秒延迟，越界是正常情况而非错误。
  TimerSnapshot snapshotAt(int elapsed) {
    final timeline = buildTimeline();
    final total = timeline.fold<int>(0, (sum, s) => sum + s.seconds);

    if (elapsed < 0) elapsed = 0;
    if (timeline.isEmpty || elapsed >= total) {
      return TimerSnapshot(
        phase: Phase.done,
        remainingInPhase: 0,
        round: rounds,
        cycle: cycles,
        totalRemaining: 0,
      );
    }

    var acc = 0;
    for (final seg in timeline) {
      if (elapsed < acc + seg.seconds) {
        return TimerSnapshot(
          phase: seg.phase,
          remainingInPhase: acc + seg.seconds - elapsed,
          round: seg.round,
          cycle: seg.cycle,
          totalRemaining: total - elapsed,
        );
      }
      acc += seg.seconds;
    }
    // 上面的 elapsed >= total 已经兜住，这里理论上到不了。
    return TimerSnapshot(
      phase: Phase.done,
      remainingInPhase: 0,
      round: rounds,
      cycle: cycles,
      totalRemaining: 0,
    );
  }

  /// 总共要做多少组（所有大循环加起来）。
  int get totalRounds => rounds * cycles;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'name': name,
        'prepare': prepareSeconds,
        'work': workSeconds,
        'rest': restSeconds,
        'rounds': rounds,
        'cycles': cycles,
        'cycleRest': cycleRestSeconds,
      };

  /// 从存档还原。**每个字段都兜底** —— 存档可能是手改过的、也可能是旧版本写的，
  /// 任何一个字段不合法都不该让整个 App 起不来。
  static IntervalPlan fromJson(Map<String, dynamic> json) {
    int readInt(String key, int fallback) {
      final v = json[key];
      if (v is int && v >= 0) return v;
      if (v is num && v >= 0) return v.toInt();
      return fallback;
    }

    final rawName = json['name'];
    return IntervalPlan(
      name: rawName is String && rawName.trim().isNotEmpty
          ? rawName
          : 'Untitled',
      prepareSeconds: readInt('prepare', 10),
      workSeconds: readInt('work', 30),
      restSeconds: readInt('rest', 15),
      rounds: readInt('rounds', 8).clamp(1, 999),
      cycles: readInt('cycles', 1).clamp(1, 99),
      cycleRestSeconds: readInt('cycleRest', 0),
    );
  }
}

/// 把秒数格式化成 `m:ss` 或 `h:mm:ss`。
String formatSeconds(int seconds) {
  if (seconds < 0) seconds = 0;
  final h = seconds ~/ 3600;
  final m = (seconds % 3600) ~/ 60;
  final s = seconds % 60;
  if (h > 0) {
    return '$h:${m.toString().padLeft(2, '0')}:${s.toString().padLeft(2, '0')}';
  }
  return '$m:${s.toString().padLeft(2, '0')}';
}
