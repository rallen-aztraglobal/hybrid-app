import 'package:flutter/material.dart';

import '../logic/date_math.dart';
import '../storage/prefs_store.dart';
import '../theme/app_colors.dart';

/// A 面本体：日期计算器。
///
/// 三个功能页共用一套「日期按钮 + 结果块」的骨架，靠顶部分段控件切换。
/// 不用底部导航栏：只有三项，而且是同一件事的三个角度，分段控件更贴切，
/// 也把整个屏幕留给内容。
class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _tab = 0;

  // 天数差
  late DateTime _from = normalize(DateTime.now());
  late DateTime _to = normalize(DateTime.now()).add(const Duration(days: 30));
  bool _inclusive = false;

  // 加减天数
  late DateTime _base = normalize(DateTime.now());
  int _offset = 30;
  bool _businessOnly = false;

  // 工作日
  late DateTime _bizFrom = normalize(DateTime.now());
  late DateTime _bizTo = normalize(DateTime.now()).add(const Duration(days: 30));

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final tab = await PrefsStore.loadTab();
    final inclusive = await PrefsStore.loadInclusive();
    if (!mounted) return;
    setState(() {
      _tab = tab;
      _inclusive = inclusive;
    });
  }

  Future<void> _pick(DateTime initial, ValueChanged<DateTime> onPicked) async {
    final picked = await showDatePicker(
      context: context,
      initialDate: initial,
      // 范围给宽：算合同、算生日、算纪念日都可能落在很远的年份。
      firstDate: DateTime.utc(1900, 1, 1),
      lastDate: DateTime.utc(2200, 12, 31),
    );
    if (picked == null) return;
    onPicked(normalize(picked));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            const _Header(),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              child: _Segmented(
                index: _tab,
                labels: const ['Difference', 'Add / subtract', 'Business days'],
                onChanged: (i) {
                  setState(() => _tab = i);
                  PrefsStore.saveTab(i);
                },
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                child: switch (_tab) {
                  0 => _buildDifference(),
                  1 => _buildOffset(),
                  _ => _buildBusiness(),
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  // ——— 页 1：两个日期相差多少 ———
  Widget _buildDifference() {
    final days = daysBetween(_from, _to, inclusive: _inclusive);
    final b = breakdown(_from, _to);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Card(children: [
          _DateRow(
            label: 'From',
            value: _from,
            onTap: () => _pick(_from, (d) => setState(() => _from = d)),
          ),
          const SizedBox(height: 10),
          _DateRow(
            label: 'To',
            value: _to,
            onTap: () => _pick(_to, (d) => setState(() => _to = d)),
          ),
          const SizedBox(height: 6),
          SwitchListTile.adaptive(
            contentPadding: EdgeInsets.zero,
            activeThumbColor: AppColors.accent,
            value: _inclusive,
            onChanged: (v) {
              setState(() => _inclusive = v);
              PrefsStore.saveInclusive(v);
            },
            title: const Text('Count both end dates',
                style: TextStyle(
                    color: AppColors.primaryText, fontWeight: FontWeight.w600)),
            // 这两种口径日常都在用，而且差一天最容易让人以为算错了，
            // 所以把差别直说出来，而不是藏在某个「关于」页里。
            subtitle: const Text(
              'Off: nights between the dates. On: days you would tick on a calendar.',
              style: TextStyle(color: AppColors.mutedText, fontSize: 12),
            ),
          ),
        ]),
        const SizedBox(height: 14),
        _Result(
          big: '$days',
          unit: days == 1 ? 'day' : 'days',
          detail: '${b.years}y ${b.months}m ${b.days}d',
        ),
        const SizedBox(height: 12),
        // 同一个区间的另外几种看法。
        //
        // 放在这里不是为了填满屏幕：问「相差多少天」的人，下一个问题通常就是
        // 「那是几周」或「去掉周末还剩几天」。先算好摆出来，省一次来回切页。
        _StatStrip(items: <(String, String)>[
          ('Weeks', (daysBetween(_from, _to) / 7).toStringAsFixed(1)),
          ('Business', '${businessDaysBetween(_from, _to)}'),
          (
            'Weekend',
            '${daysBetween(_from, _to) - businessDaysBetween(_from, _to)}'
          ),
        ]),
      ],
    );
  }

  // ——— 页 2：某日加减 N 天 ———
  Widget _buildOffset() {
    final result = _businessOnly
        ? addBusinessDays(_base, _offset)
        : addDays(_base, _offset);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Card(children: [
          _DateRow(
            label: 'Start',
            value: _base,
            onTap: () => _pick(_base, (d) => setState(() => _base = d)),
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              const Text('Days',
                  style: TextStyle(
                      color: AppColors.mutedText, fontWeight: FontWeight.w600)),
              const Spacer(),
              _Stepper(
                value: _offset,
                onChanged: (v) => setState(() => _offset = v),
              ),
            ],
          ),
          const SizedBox(height: 6),
          SwitchListTile.adaptive(
            contentPadding: EdgeInsets.zero,
            activeThumbColor: AppColors.accent,
            value: _businessOnly,
            onChanged: (v) => setState(() => _businessOnly = v),
            title: const Text('Business days only',
                style: TextStyle(
                    color: AppColors.primaryText, fontWeight: FontWeight.w600)),
            subtitle: const Text(
              'Skips Saturdays and Sundays. Public holidays are not included.',
              style: TextStyle(color: AppColors.mutedText, fontSize: 12),
            ),
          ),
        ]),
        const SizedBox(height: 14),
        _Result(
          big: _formatDate(result),
          unit: _weekdayName(result),
          detail: _offset >= 0
              ? '$_offset ${_businessOnly ? "business " : ""}days later'
              : '${-_offset} ${_businessOnly ? "business " : ""}days earlier',
        ),
      ],
    );
  }

  // ——— 页 3：两个日期之间有几个工作日 ———
  Widget _buildBusiness() {
    final biz = businessDaysBetween(_bizFrom, _bizTo);
    final all = daysBetween(_bizFrom, _bizTo);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _Card(children: [
          _DateRow(
            label: 'From',
            value: _bizFrom,
            onTap: () => _pick(_bizFrom, (d) => setState(() => _bizFrom = d)),
          ),
          const SizedBox(height: 10),
          _DateRow(
            label: 'To',
            value: _bizTo,
            onTap: () => _pick(_bizTo, (d) => setState(() => _bizTo = d)),
          ),
          const SizedBox(height: 10),
          const Text(
            'Counts the start date, not the end date — "Monday to Saturday" is 5 business days. '
            'Public holidays are not included.',
            style: TextStyle(color: AppColors.mutedText, fontSize: 12),
          ),
        ]),
        const SizedBox(height: 14),
        _Result(
          big: '$biz',
          unit: biz == 1 ? 'business day' : 'business days',
          detail: '$all calendar days · ${all - biz} weekend days',
        ),
      ],
    );
  }
}

String _formatDate(DateTime d) =>
    '${d.year}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

String _weekdayName(DateTime d) => const <String>[
      'Monday',
      'Tuesday',
      'Wednesday',
      'Thursday',
      'Friday',
      'Saturday',
      'Sunday',
    ][d.weekday - 1];

class _Header extends StatelessWidget {
  const _Header();

  @override
  Widget build(BuildContext context) => const Padding(
        padding: EdgeInsets.fromLTRB(16, 14, 16, 10),
        child: Align(
          alignment: Alignment.centerLeft,
          child: Text(
            'DaySpan',
            style: TextStyle(
              color: AppColors.primaryText,
              fontSize: 24,
              fontWeight: FontWeight.w700,
              letterSpacing: -0.3,
            ),
          ),
        ),
      );
}

class _Segmented extends StatelessWidget {
  const _Segmented({
    required this.index,
    required this.labels,
    required this.onChanged,
  });

  final int index;
  final List<String> labels;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: AppColors.field,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: List.generate(labels.length, (i) {
          final selected = i == index;
          return Expanded(
            child: GestureDetector(
              onTap: () => onChanged(i),
              behavior: HitTestBehavior.opaque,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 160),
                curve: Curves.easeOut,
                padding: const EdgeInsets.symmetric(vertical: 9),
                decoration: BoxDecoration(
                  color: selected ? AppColors.surface : Colors.transparent,
                  borderRadius: BorderRadius.circular(9),
                ),
                child: Text(
                  labels[i],
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 12.5,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    color: selected ? AppColors.accent : AppColors.mutedText,
                  ),
                ),
              ),
            ),
          );
        }),
      ),
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.children});
  final List<Widget> children;

  @override
  Widget build(BuildContext context) => Container(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppColors.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: children,
        ),
      );
}

class _DateRow extends StatelessWidget {
  const _DateRow({
    required this.label,
    required this.value,
    required this.onTap,
  });

  final String label;
  final DateTime value;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
        decoration: BoxDecoration(
          color: AppColors.field,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: [
            SizedBox(
              width: 52,
              child: Text(label,
                  style: const TextStyle(
                      color: AppColors.mutedText,
                      fontSize: 12.5,
                      fontWeight: FontWeight.w600)),
            ),
            Text(
              _formatDate(value),
              style: const TextStyle(
                color: AppColors.primaryText,
                fontSize: 17,
                fontWeight: FontWeight.w700,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),
            const SizedBox(width: 10),
            Text(
              _weekdayName(value).substring(0, 3),
              style: TextStyle(
                color: isWeekend(value)
                    ? AppColors.weekend
                    : AppColors.mutedText,
                fontSize: 12.5,
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            const Icon(Icons.calendar_today_outlined,
                size: 16, color: AppColors.mutedText),
          ],
        ),
      ),
    );
  }
}

/// 数字增减控件。
///
/// 不用文本框：手机上弹数字键盘、再收起，比点几下慢得多，
/// 而且这个场景的常用值就是 ±1 / ±7 / ±30 这几档。
class _Stepper extends StatelessWidget {
  const _Stepper({required this.value, required this.onChanged});

  final int value;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    Widget btn(String text, int delta) => GestureDetector(
          onTap: () => onChanged(value + delta),
          behavior: HitTestBehavior.opaque,
          child: Container(
            width: 38,
            height: 34,
            alignment: Alignment.center,
            margin: const EdgeInsets.symmetric(horizontal: 2),
            decoration: BoxDecoration(
              color: AppColors.field,
              borderRadius: BorderRadius.circular(9),
            ),
            child: Text(text,
                style: const TextStyle(
                    color: AppColors.accent,
                    fontSize: 12.5,
                    fontWeight: FontWeight.w700)),
          ),
        );

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        btn('−30', -30),
        btn('−1', -1),
        Container(
          constraints: const BoxConstraints(minWidth: 58),
          alignment: Alignment.center,
          child: Text(
            '$value',
            style: const TextStyle(
              color: AppColors.primaryText,
              fontSize: 19,
              fontWeight: FontWeight.w700,
              fontFeatures: [FontFeature.tabularFigures()],
            ),
          ),
        ),
        btn('+1', 1),
        btn('+30', 30),
      ],
    );
  }
}

class _Result extends StatelessWidget {
  const _Result({required this.big, required this.unit, required this.detail});

  final String big;
  final String unit;
  final String detail;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 18),
      decoration: BoxDecoration(
        color: AppColors.accentWash,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          FittedBox(
            fit: BoxFit.scaleDown,
            child: Text(
              big,
              style: const TextStyle(
                color: AppColors.accent,
                fontSize: 46,
                fontWeight: FontWeight.w800,
                height: 1.05,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),
          ),
          const SizedBox(height: 4),
          Text(unit,
              style: const TextStyle(
                  color: AppColors.accent,
                  fontSize: 14,
                  fontWeight: FontWeight.w600)),
          const SizedBox(height: 10),
          Text(detail,
              textAlign: TextAlign.center,
              style: const TextStyle(
                  color: AppColors.mutedText, fontSize: 12.5)),
        ],
      ),
    );
  }
}

/// 三格并排的小统计条。
///
/// 用等宽数字（tabularFigures）：数值会随日期变化跳动，非等宽字体下
/// 每跳一次三格就各自错位一下，看着很躁。
class _StatStrip extends StatelessWidget {
  const _StatStrip({required this.items});

  final List<(String, String)> items;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        for (var i = 0; i < items.length; i++) ...<Widget>[
          if (i > 0) const SizedBox(width: 10),
          Expanded(
            child: Container(
              padding: const EdgeInsets.symmetric(vertical: 14),
              decoration: BoxDecoration(
                color: AppColors.surface,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: AppColors.border),
              ),
              child: Column(
                children: [
                  Text(
                    items[i].$2,
                    style: const TextStyle(
                      color: AppColors.primaryText,
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                      fontFeatures: [FontFeature.tabularFigures()],
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    items[i].$1,
                    style: const TextStyle(
                        color: AppColors.mutedText, fontSize: 11.5),
                  ),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }
}
