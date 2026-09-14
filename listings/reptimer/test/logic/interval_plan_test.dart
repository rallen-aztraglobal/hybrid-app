import 'package:flutter_test/flutter_test.dart';
import 'package:reptimer7719/logic/interval_plan.dart';

IntervalPlan plan({
  int prepare = 10,
  int work = 30,
  int rest = 15,
  int rounds = 3,
  int cycles = 1,
  int cycleRest = 0,
}) =>
    IntervalPlan(
      name: 'T',
      prepareSeconds: prepare,
      workSeconds: work,
      restSeconds: rest,
      rounds: rounds,
      cycles: cycles,
      cycleRestSeconds: cycleRest,
    );

void main() {
  group('时间轴结构', () {
    test('准备 + 3 组，最后一组之后不留休息', () {
      final t = plan().buildTimeline();
      expect(t.map((s) => s.phase).toList(), <Phase>[
        Phase.prepare,
        Phase.work,
        Phase.rest,
        Phase.work,
        Phase.rest,
        Phase.work, // 最后一组后面没有 rest
      ]);
    });

    test('总时长 = 各段之和', () {
      final p = plan(prepare: 10, work: 30, rest: 15, rounds: 3);
      expect(p.totalSeconds, 10 + 30 + 15 + 30 + 15 + 30);
    });

    test('组数与循环数相乘', () {
      expect(plan(rounds: 4, cycles: 3).totalRounds, 12);
    });
  });

  group('0 秒的段必须整段消失，而不是显示 0 秒', () {
    // 把休息设成 0（连续做组）是很常见的用法。
    // 若保留一个 0 秒的段，界面会闪一下「Rest 0」再跳走，看着像 bug；
    // 而且「每秒减一、减到 0 才切换」那种写法会在 0 秒段上多跑一秒甚至卡住。
    test('休息为 0 时时间轴里没有 rest 段', () {
      final t = plan(rest: 0, rounds: 3).buildTimeline();
      expect(t.any((s) => s.phase == Phase.rest), isFalse);
      expect(t.map((s) => s.phase).toList(), <Phase>[
        Phase.prepare,
        Phase.work,
        Phase.work,
        Phase.work,
      ]);
    });

    test('准备为 0 时时间轴直接从做开始', () {
      final t = plan(prepare: 0).buildTimeline();
      expect(t.first.phase, Phase.work);
    });

    test('大循环休息为 0 时不插入该段', () {
      final t = plan(rounds: 2, cycles: 2, cycleRest: 0).buildTimeline();
      expect(t.any((s) => s.phase == Phase.cycleRest), isFalse);
    });

    test('时间轴里不存在任何 0 秒的段', () {
      for (final p in <IntervalPlan>[
        plan(prepare: 0),
        plan(rest: 0),
        plan(prepare: 0, rest: 0),
        plan(rounds: 1),
        plan(rounds: 1, cycles: 3, cycleRest: 0),
        plan(cycles: 4, cycleRest: 60),
      ]) {
        for (final s in p.buildTimeline()) {
          expect(s.seconds, greaterThan(0),
              reason: '${s.phase} 段是 0 秒，应当在构建时就被剔除');
        }
      }
    });
  });

  group('最后一段不留多余休息', () {
    test('单组时只有准备 + 做', () {
      final t = plan(rounds: 1).buildTimeline();
      expect(t.map((s) => s.phase).toList(), <Phase>[Phase.prepare, Phase.work]);
    });

    test('最后一个大循环之后没有 cycleRest', () {
      final t = plan(rounds: 2, cycles: 2, cycleRest: 60).buildTimeline();
      expect(t.last.phase, Phase.work, reason: '练完就结束，不该再等一段长休息');
      expect(t.where((s) => s.phase == Phase.cycleRest).length, 1,
          reason: '两个循环之间只插一次');
    });

    test('时间轴最后一段永远是 work', () {
      for (final p in <IntervalPlan>[
        plan(),
        plan(rest: 0),
        plan(rounds: 1),
        plan(cycles: 3, cycleRest: 30),
        plan(prepare: 0, rest: 0, rounds: 5, cycles: 2, cycleRest: 20),
      ]) {
        expect(p.buildTimeline().last.phase, Phase.work);
      }
    });
  });

  group('某一时刻的状态', () {
    final p = plan(prepare: 10, work: 30, rest: 15, rounds: 3);

    test('第 0 秒在准备段，剩满', () {
      final s = p.snapshotAt(0);
      expect(s.phase, Phase.prepare);
      expect(s.remainingInPhase, 10);
      expect(s.totalRemaining, p.totalSeconds);
    });

    test('准备段最后一秒仍属于准备', () {
      expect(p.snapshotAt(9).phase, Phase.prepare);
      expect(p.snapshotAt(9).remainingInPhase, 1);
    });

    test('准备结束的那一秒切到第 1 组', () {
      final s = p.snapshotAt(10);
      expect(s.phase, Phase.work);
      expect(s.round, 1);
      expect(s.remainingInPhase, 30);
    });

    test('组号随进度递增', () {
      expect(p.snapshotAt(10 + 30 + 15).round, 2, reason: '第一次休息之后是第 2 组');
      expect(p.snapshotAt(10 + (30 + 15) * 2).round, 3);
    });

    test('跑满即完成', () {
      final s = p.snapshotAt(p.totalSeconds);
      expect(s.isDone, isTrue);
      expect(s.totalRemaining, 0);
    });

    test('超出总时长不越界、不抛异常 —— tick 与重建之间总有延迟', () {
      expect(p.snapshotAt(p.totalSeconds + 1000).isDone, isTrue);
    });

    test('负数按 0 处理', () {
      expect(p.snapshotAt(-5).phase, p.snapshotAt(0).phase);
    });

    test('逐秒推进：阶段只会前进，剩余时间单调不增', () {
      var lastTotal = p.totalSeconds + 1;
      for (var t = 0; t <= p.totalSeconds; t++) {
        final s = p.snapshotAt(t);
        expect(s.totalRemaining, lessThan(lastTotal),
            reason: 't=$t 时总剩余没有减少');
        lastTotal = s.totalRemaining;
        if (!s.isDone) {
          expect(s.remainingInPhase, greaterThan(0),
              reason: 't=$t 时本段剩余为 0 却还没结束 —— 会卡住一秒');
        }
      }
    });

    test('休息为 0 时逐秒推进也不会出现 rest', () {
      final q = plan(prepare: 0, rest: 0, rounds: 4);
      for (var t = 0; t < q.totalSeconds; t++) {
        expect(q.snapshotAt(t).phase, Phase.work);
      }
    });
  });

  group('不可运行的方案', () {
    test('做的时长为 0 不可运行，时间轴为空', () {
      final p = plan(work: 0);
      expect(p.isRunnable, isFalse);
      expect(p.buildTimeline(), isEmpty);
      expect(p.totalSeconds, 0);
      expect(p.snapshotAt(0).isDone, isTrue);
    });

    test('组数为 0 不可运行', () {
      expect(plan(rounds: 0).isRunnable, isFalse);
    });

    test('其余字段为 0 仍可运行', () {
      expect(plan(prepare: 0, rest: 0, cycles: 1).isRunnable, isTrue);
    });
  });

  group('存档读写', () {
    test('往返不丢信息', () {
      const p = IntervalPlan(
        name: 'Tabata',
        prepareSeconds: 5,
        workSeconds: 20,
        restSeconds: 10,
        rounds: 8,
        cycles: 2,
        cycleRestSeconds: 60,
      );
      final back = IntervalPlan.fromJson(p.toJson());
      expect(back.name, p.name);
      expect(back.workSeconds, p.workSeconds);
      expect(back.rounds, p.rounds);
      expect(back.cycles, p.cycles);
      expect(back.cycleRestSeconds, p.cycleRestSeconds);
    });

    test('字段缺失 / 类型不对 / 为负，都走兜底而不是崩', () {
      final back = IntervalPlan.fromJson(<String, dynamic>{
        'name': 42, // 类型不对
        'work': -5, // 负数
        'rounds': 'many', // 类型不对
      });
      expect(back.name, 'Untitled');
      expect(back.workSeconds, 30);
      expect(back.rounds, 8);
    });

    test('空名字走兜底', () {
      final back = IntervalPlan.fromJson(<String, dynamic>{'name': '   '});
      expect(back.name, 'Untitled');
    });

    test('组数被夹在合理范围内', () {
      final back =
          IntervalPlan.fromJson(<String, dynamic>{'rounds': 100000});
      expect(back.rounds, 999);
    });
  });

  group('时间格式化', () {
    test('一小时以内是 m:ss', () {
      expect(formatSeconds(0), '0:00');
      expect(formatSeconds(9), '0:09');
      expect(formatSeconds(60), '1:00');
      expect(formatSeconds(605), '10:05');
    });

    test('超过一小时是 h:mm:ss', () {
      expect(formatSeconds(3600), '1:00:00');
      expect(formatSeconds(3661), '1:01:01');
    });

    test('负数当 0', () {
      expect(formatSeconds(-10), '0:00');
    });
  });
}
