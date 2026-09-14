/// 日期运算。全部是纯函数，没有「现在几点」这种隐含输入 —— 除了显式传入的今天。
///
/// ## 为什么内部一律用 UTC
///
/// 这是日期计算器最经典、也最难自查的一类 bug：本地时间的一天不一定是 24 小时。
/// 夏令时切换那天只有 23 小时（或 25 小时），于是
///
///     DateTime(2026, 3, 8).add(Duration(days: 1))   // 美东：得到 3 月 9 日 0 点？
///
/// 在跨夏令时的时区里会落到 3 月 8 日 23 点或 3 月 9 日 1 点，`.day` 取出来就错了一天。
/// `difference(...).inDays` 同理会少算或多算。
///
/// 这里把所有日期都规约成 **UTC 当天 0 点**：UTC 没有夏令时，一天恒等于 24 小时，
/// 上面两个运算就都是精确的。用户看到的年月日不受影响 —— 我们从头到尾只关心
/// 「哪一天」，从不关心「几点」。
library;

/// 把任意 DateTime 规约成 UTC 当天 0 点，丢掉时分秒与时区。
///
/// 入口处统一调用它，是这套运算不出夏令时问题的前提。
DateTime normalize(DateTime d) => DateTime.utc(d.year, d.month, d.day);

/// 两个日期之间相隔几天。
///
/// [inclusive] = true 时把首尾两天都算进去（「从 1 号到 3 号一共几天」= 3），
/// false 时算的是间隔（「1 号到 3 号相隔几天」= 2）。
///
/// 这两种口径在日常语境里都常用，而且**没有哪个更「正确」**，
/// 所以做成显式选项而不是替用户选一个 —— 差一天的结果最容易让人以为 App 算错了。
int daysBetween(DateTime a, DateTime b, {bool inclusive = false}) {
  final from = normalize(a);
  final to = normalize(b);
  final diff = to.difference(from).inDays.abs();
  return inclusive ? diff + 1 : diff;
}

/// 某日加减 n 天。n 可以为负。
DateTime addDays(DateTime date, int n) =>
    normalize(date).add(Duration(days: n));

/// 是否周末（周六、周日）。
///
/// 只按星期判断，不含任何法定假日 —— 假日表因国家/年份而异，
/// 内置一份必然过时且必然不适用于某些用户。文案里也据此写明「不含节假日」。
bool isWeekend(DateTime d) {
  final w = normalize(d).weekday;
  return w == DateTime.saturday || w == DateTime.sunday;
}

/// 两个日期之间有几个工作日。
///
/// 口径：**含起始日、不含结束日**（区间 [from, to)）——
/// 与「从周一到周六有 5 个工作日」的日常说法一致。
/// 顺序反了会自动交换，结果恒非负。
int businessDaysBetween(DateTime a, DateTime b) {
  var from = normalize(a);
  var to = normalize(b);
  if (from.isAfter(to)) {
    final t = from;
    from = to;
    to = t;
  }

  // 先按整周算，再补零头 —— 逐日循环在跨越几十年时会慢到卡界面。
  final totalDays = to.difference(from).inDays;
  final fullWeeks = totalDays ~/ 7;
  var count = fullWeeks * 5;

  var cursor = from.add(Duration(days: fullWeeks * 7));
  while (cursor.isBefore(to)) {
    if (!isWeekend(cursor)) count++;
    cursor = cursor.add(const Duration(days: 1));
  }
  return count;
}

/// 从某日起，往后（或往前）数 n 个工作日。
///
/// 起始日本身不计入，与「今天起 3 个工作日后」的说法一致。
DateTime addBusinessDays(DateTime date, int n) {
  var cursor = normalize(date);
  if (n == 0) return cursor;
  final step = n > 0 ? 1 : -1;
  var remaining = n.abs();
  while (remaining > 0) {
    cursor = cursor.add(Duration(days: step));
    if (!isWeekend(cursor)) remaining--;
  }
  return cursor;
}

/// 两个日期相差的「年 / 月 / 日」拆分。
class DateBreakdown {
  const DateBreakdown({
    required this.years,
    required this.months,
    required this.days,
  });

  final int years;
  final int months;
  final int days;

  @override
  String toString() => '$years 年 $months 个月 $days 天';

  @override
  bool operator ==(Object other) =>
      other is DateBreakdown &&
      other.years == years &&
      other.months == months &&
      other.days == days;

  @override
  int get hashCode => Object.hash(years, months, days);
}

/// 把两个日期的间隔拆成年/月/日。用于算年龄、工龄这类。
///
/// 算法是「先尽量进整年，再进整月，剩下的是天」，与人的直觉一致：
/// 2024-01-31 到 2024-03-01 是「1 个月零 1 天」，而不是「30 天」。
///
/// 月份长度不等带来的歧义无法消除 —— 1 月 31 日往后一个月是几号？
/// 这里的口径是**按目标月的实际天数截断**（1/31 + 1 月 = 2/28 或 2/29），
/// 与绝大多数日历应用一致；`breakdown` 内部据此回推，所以
/// `breakdown(a, b)` 与 [addMonths] 是自洽的。
DateBreakdown breakdown(DateTime a, DateTime b) {
  var from = normalize(a);
  var to = normalize(b);
  if (from.isAfter(to)) {
    final t = from;
    from = to;
    to = t;
  }

  // 做法：先求「从 from 出发最多能进几个整月而不越过 to」，再把余下的算成天。
  //
  // 不用「年月日各自相减、不够就借位」那套 —— 它在跨月末时会借不够。
  // 2026-01-31 → 2026-03-01：日差 = 1-31 = -30，退一个月后从 2 月借 28 天，
  // 结果还是 -2，仍要再借一次。多次借位的边界条件很难写对。
  //
  // 以 [addMonths] 为锚就没有这个问题：它自己处理了月末截断，
  // 于是 breakdown 与 addMonths 必然自洽 —— from 加上算出的年月日，正好回到 to。
  // 这条自洽性在测试里钉住了。
  var totalMonths = (to.year - from.year) * 12 + (to.month - from.month);
  if (addMonths(from, totalMonths).isAfter(to)) totalMonths--;

  final anchor = addMonths(from, totalMonths);
  final days = to.difference(anchor).inDays;

  return DateBreakdown(
    years: totalMonths ~/ 12,
    months: totalMonths % 12,
    days: days,
  );
}

/// 某年某月有多少天。闰年判断走标准格里高利规则。
int daysInMonth(int year, int month) {
  const lengths = <int>[31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  if (month == 2 && isLeapYear(year)) return 29;
  return lengths[month - 1];
}

/// 是否闰年。
///
/// 四年一闰、百年不闰、四百年再闰 —— 1900 不是闰年，2000 是。
/// 这两个是检验闰年实现最经典的用例，测试里都钉了。
bool isLeapYear(int year) =>
    (year % 4 == 0 && year % 100 != 0) || year % 400 == 0;

/// 某日加减 n 个月。日超出目标月天数时截断到该月最后一天。
///
/// 1 月 31 日 + 1 个月 = 2 月 28/29 日，而不是溢出到 3 月 3 日 ——
/// 后者是直接用 `DateTime(y, m+1, d)` 会得到的结果，也是这类工具最常见的错。
DateTime addMonths(DateTime date, int n) {
  final d = normalize(date);
  final total = d.year * 12 + (d.month - 1) + n;
  final year = total ~/ 12;
  final month = total % 12 + 1;
  final day = d.day <= daysInMonth(year, month) ? d.day : daysInMonth(year, month);
  return DateTime.utc(year, month, day);
}
