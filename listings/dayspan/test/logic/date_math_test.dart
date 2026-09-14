import 'package:dayspan3306/logic/date_math.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('闰年', () {
    test('四年一闰、百年不闰、四百年再闰', () {
      expect(isLeapYear(2024), isTrue);
      expect(isLeapYear(2023), isFalse);
      expect(isLeapYear(1900), isFalse, reason: '整百年不闰 —— 最经典的反例');
      expect(isLeapYear(2000), isTrue, reason: '四百年再闰');
      expect(isLeapYear(2100), isFalse);
    });

    test('二月天数随闰年变化', () {
      expect(daysInMonth(2024, 2), 29);
      expect(daysInMonth(2023, 2), 28);
      expect(daysInMonth(1900, 2), 28);
      expect(daysInMonth(2000, 2), 29);
    });
  });

  group('天数差', () {
    test('同一天：间隔 0 天，含首尾 1 天', () {
      final d = DateTime.utc(2026, 3, 15);
      expect(daysBetween(d, d), 0);
      expect(daysBetween(d, d, inclusive: true), 1);
    });

    test('两种口径差恰好一天', () {
      final a = DateTime.utc(2026, 3, 1);
      final b = DateTime.utc(2026, 3, 3);
      expect(daysBetween(a, b), 2, reason: '相隔 2 天');
      expect(daysBetween(a, b, inclusive: true), 3, reason: '从 1 号到 3 号共 3 天');
    });

    test('顺序颠倒结果相同（恒非负）', () {
      final a = DateTime.utc(2026, 1, 1);
      final b = DateTime.utc(2026, 12, 31);
      expect(daysBetween(b, a), daysBetween(a, b));
    });

    test('跨闰年 2 月', () {
      // 2024-02-28 → 2024-03-01 跨过 2 月 29 日，相隔 2 天
      expect(
        daysBetween(DateTime.utc(2024, 2, 28), DateTime.utc(2024, 3, 1)),
        2,
      );
      // 2023 年没有 2 月 29 日，只隔 1 天
      expect(
        daysBetween(DateTime.utc(2023, 2, 28), DateTime.utc(2023, 3, 1)),
        1,
      );
    });

    test('整年：平年 365、闰年 366', () {
      expect(
        daysBetween(DateTime.utc(2023, 1, 1), DateTime.utc(2024, 1, 1)),
        365,
      );
      expect(
        daysBetween(DateTime.utc(2024, 1, 1), DateTime.utc(2025, 1, 1)),
        366,
      );
    });

    test('带时分秒的输入会被规约掉，只看日期', () {
      final a = DateTime.utc(2026, 3, 1, 23, 59, 59);
      final b = DateTime.utc(2026, 3, 2, 0, 0, 1);
      expect(daysBetween(a, b), 1, reason: '相差 2 秒，但跨了一天');
    });
  });

  group('夏令时不能影响结果', () {
    // 这是日期计算器最难自查的一类错：本地时区里「一天」不一定是 24 小时。
    // 内部规约到 UTC 之后，下面这些跨夏令时切换的日期必须算得和平常一样。
    test('跨美国夏令时开始日（3 月第二个周日）加一天仍是次日', () {
      // 2026-03-08 是美国夏令时开始日，本地只有 23 小时
      final d = DateTime(2026, 3, 8); // 故意用本地时间构造
      final next = addDays(d, 1);
      expect(next.year, 2026);
      expect(next.month, 3);
      expect(next.day, 9, reason: '本地时间直接 add 24h 会落在 3 月 8 日 23 点');
    });

    test('跨夏令时结束日（11 月第一个周日）加一天仍是次日', () {
      final d = DateTime(2026, 11, 1); // 本地有 25 小时
      final next = addDays(d, 1);
      expect(next.day, 2);
      expect(next.month, 11);
    });

    test('跨越整个夏令时区间的天数差是精确的', () {
      // 2026-03-01 → 2026-12-01，中间经历夏令时开始与结束各一次
      expect(
        daysBetween(DateTime(2026, 3, 1), DateTime(2026, 12, 1)),
        275,
        reason: '3月30+4月30+5月31+6月30+7月31+8月31+9月30+10月31+11月30 = 275',
      );
    });
  });

  group('加减天数', () {
    test('加 0 天是本身', () {
      final d = DateTime.utc(2026, 5, 20);
      expect(addDays(d, 0), d);
    });

    test('负数往前数', () {
      expect(addDays(DateTime.utc(2026, 3, 1), -1), DateTime.utc(2026, 2, 28));
      expect(addDays(DateTime.utc(2024, 3, 1), -1), DateTime.utc(2024, 2, 29),
          reason: '闰年要落在 2 月 29 日');
    });

    test('跨年', () {
      expect(addDays(DateTime.utc(2026, 12, 31), 1), DateTime.utc(2027, 1, 1));
      expect(addDays(DateTime.utc(2026, 1, 1), -1), DateTime.utc(2025, 12, 31));
    });

    test('与天数差互逆', () {
      final base = DateTime.utc(2026, 7, 4);
      for (final n in <int>[1, 7, 30, 365, 1000]) {
        expect(daysBetween(base, addDays(base, n)), n);
      }
    });
  });

  group('工作日', () {
    test('周末判定', () {
      // 2026-03-14 是周六，15 是周日，16 是周一
      expect(isWeekend(DateTime.utc(2026, 3, 14)), isTrue);
      expect(isWeekend(DateTime.utc(2026, 3, 15)), isTrue);
      expect(isWeekend(DateTime.utc(2026, 3, 16)), isFalse);
    });

    test('周一到周六是 5 个工作日', () {
      // 含起始日、不含结束日
      expect(
        businessDaysBetween(DateTime.utc(2026, 3, 16), DateTime.utc(2026, 3, 21)),
        5,
      );
    });

    test('整周恒为 5 个工作日', () {
      final a = DateTime.utc(2026, 3, 16); // 周一
      expect(businessDaysBetween(a, addDays(a, 7)), 5);
      expect(businessDaysBetween(a, addDays(a, 14)), 10);
      expect(businessDaysBetween(a, addDays(a, 70)), 50);
    });

    test('整周快捷路径与逐日累加结果一致', () {
      // 整周批量算 + 零头逐日补 是性能优化，必须与朴素实现等价。
      final start = DateTime.utc(2026, 1, 1);
      for (var n = 0; n <= 90; n++) {
        final end = addDays(start, n);
        var naive = 0;
        var cur = start;
        while (cur.isBefore(end)) {
          if (!isWeekend(cur)) naive++;
          cur = addDays(cur, 1);
        }
        expect(businessDaysBetween(start, end), naive,
            reason: 'n=$n 时快捷路径与逐日累加不一致');
      }
    });

    test('顺序颠倒结果相同', () {
      final a = DateTime.utc(2026, 3, 16);
      final b = DateTime.utc(2026, 4, 20);
      expect(businessDaysBetween(b, a), businessDaysBetween(a, b));
    });

    test('周末起算：往后数工作日会跳过周末', () {
      // 2026-03-14 周六，往后 1 个工作日 = 3 月 16 日周一
      expect(
        addBusinessDays(DateTime.utc(2026, 3, 14), 1),
        DateTime.utc(2026, 3, 16),
      );
    });

    test('往前数工作日', () {
      // 2026-03-16 周一，往前 1 个工作日 = 3 月 13 日周五
      expect(
        addBusinessDays(DateTime.utc(2026, 3, 16), -1),
        DateTime.utc(2026, 3, 13),
      );
    });

    test('加 0 个工作日是本身，即使那天是周末', () {
      final sat = DateTime.utc(2026, 3, 14);
      expect(addBusinessDays(sat, 0), sat);
    });

    test('addBusinessDays 的结果永远不是周末', () {
      final base = DateTime.utc(2026, 3, 11);
      for (var n = -30; n <= 30; n++) {
        if (n == 0) continue;
        expect(isWeekend(addBusinessDays(base, n)), isFalse, reason: 'n=$n');
      }
    });
  });

  group('加减月份 —— 月末截断', () {
    test('1 月 31 日加一个月是 2 月末，不是 3 月初', () {
      expect(addMonths(DateTime.utc(2026, 1, 31), 1), DateTime.utc(2026, 2, 28));
      expect(addMonths(DateTime.utc(2024, 1, 31), 1), DateTime.utc(2024, 2, 29),
          reason: '闰年落在 29 日');
    });

    test('3 月 31 日加一个月是 4 月 30 日', () {
      expect(addMonths(DateTime.utc(2026, 3, 31), 1), DateTime.utc(2026, 4, 30));
    });

    test('跨年加减', () {
      expect(addMonths(DateTime.utc(2026, 11, 15), 3), DateTime.utc(2027, 2, 15));
      expect(addMonths(DateTime.utc(2026, 2, 15), -3), DateTime.utc(2025, 11, 15));
    });

    test('加 12 个月就是加一年（日期不在月末时）', () {
      final d = DateTime.utc(2026, 6, 15);
      expect(addMonths(d, 12), DateTime.utc(2027, 6, 15));
    });
  });

  group('年月日拆分', () {
    test('整年', () {
      expect(
        breakdown(DateTime.utc(2000, 5, 10), DateTime.utc(2026, 5, 10)),
        const DateBreakdown(years: 26, months: 0, days: 0),
      );
    });

    test('差一天不满整年', () {
      // 锚点是 2026-04-10（25 年 11 个月），到 5 月 9 日是 29 天 ——
      // 4 月 10 日 → 5 月 10 日才是 30 天。
      expect(
        breakdown(DateTime.utc(2000, 5, 10), DateTime.utc(2026, 5, 9)),
        const DateBreakdown(years: 25, months: 11, days: 29),
      );
    });

    test('自洽性：from 加上算出的年月日，必须正好回到 to', () {
      // breakdown 与 addMonths/addDays 是同一套口径的两个方向，必须互逆。
      // 这条比逐个举例更有力：它覆盖了月末、闰年、跨年的各种组合。
      final samples = <List<DateTime>>[
        [DateTime.utc(2026, 1, 31), DateTime.utc(2026, 3, 1)],
        [DateTime.utc(2024, 1, 31), DateTime.utc(2024, 3, 1)],
        [DateTime.utc(2000, 5, 10), DateTime.utc(2026, 5, 9)],
        [DateTime.utc(1999, 12, 31), DateTime.utc(2026, 1, 1)],
        [DateTime.utc(2024, 2, 29), DateTime.utc(2025, 2, 28)],
        [DateTime.utc(2026, 3, 31), DateTime.utc(2026, 4, 30)],
      ];
      for (final s in samples) {
        final b = breakdown(s[0], s[1]);
        final back = addDays(addMonths(s[0], b.years * 12 + b.months), b.days);
        expect(back, s[1], reason: '${s[0]} → ${s[1]} 拆分后加不回去：$b');
      }
    });

    test('天数部分恒非负', () {
      final base = DateTime.utc(2026, 1, 31);
      for (var n = 0; n < 400; n++) {
        final b = breakdown(base, addDays(base, n));
        expect(b.days, greaterThanOrEqualTo(0), reason: 'n=$n 时 days=${b.days}');
        expect(b.months, inInclusiveRange(0, 11));
      }
    });

    test('借月时借的是上个月的实际天数，不是固定 30', () {
      // 2026-01-31 → 2026-03-01：退一个月后从 2 月借，2026 年 2 月是 28 天
      final b = breakdown(DateTime.utc(2026, 1, 31), DateTime.utc(2026, 3, 1));
      expect(b, const DateBreakdown(years: 0, months: 1, days: 1));
    });

    test('闰年借 2 月是 29 天', () {
      final b = breakdown(DateTime.utc(2024, 1, 31), DateTime.utc(2024, 3, 1));
      expect(b, const DateBreakdown(years: 0, months: 1, days: 1));
    });

    test('同一天是全 0', () {
      final d = DateTime.utc(2026, 8, 8);
      expect(breakdown(d, d), const DateBreakdown(years: 0, months: 0, days: 0));
    });

    test('顺序颠倒结果相同', () {
      final a = DateTime.utc(1990, 2, 20);
      final b = DateTime.utc(2026, 9, 11);
      expect(breakdown(b, a), breakdown(a, b));
    });
  });
}
