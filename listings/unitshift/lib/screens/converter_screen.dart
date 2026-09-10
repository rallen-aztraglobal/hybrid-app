import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/catalog.dart';
import '../logic/entry.dart';
import '../logic/formatter.dart';
import '../logic/unit.dart';
import '../storage/prefs_store.dart';
import '../theme/app_colors.dart';
import '../widgets/keypad.dart';

/// A 面：换算本体。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
///
/// 交互取向只有一条：**输入一次，所有单位同时出结果**。
/// 常见换算 App 是「选源单位 → 选目标单位 → 看一个数」，改一次目标要点两下；
/// 这里把整类单位一次全列出来，边打边全部跟着变，不存在「选目标」这一步。
class ConverterScreen extends StatefulWidget {
  const ConverterScreen({super.key});

  @override
  State<ConverterScreen> createState() => _ConverterScreenState();
}

class _ConverterScreenState extends State<ConverterScreen> {
  UnitCategory _category = Catalog.categories.first;
  late Unit _unit = _category.byId(_category.defaultUnitId) ?? _category.units.first;
  final NumericEntry _entry = NumericEntry();

  final ScrollController _categoryScroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _restore();
  }

  @override
  void dispose() {
    _categoryScroll.dispose();
    super.dispose();
  }

  Future<void> _restore() async {
    final categoryId = await PrefsStore.lastCategory();
    final category = categoryId == null ? null : Catalog.byId(categoryId);
    if (category == null) return;
    final unitId = await PrefsStore.lastUnit(category.id);
    if (!mounted) return;
    setState(() {
      _category = category;
      _unit = category.byId(unitId ?? category.defaultUnitId) ??
          category.units.first;
    });
  }

  Future<void> _selectCategory(UnitCategory category) async {
    if (category.id == _category.id) return;
    HapticFeedback.selectionClick();
    final unitId = await PrefsStore.lastUnit(category.id);
    if (!mounted) return;
    setState(() {
      _category = category;
      _unit = category.byId(unitId ?? category.defaultUnitId) ??
          category.units.first;
      _entry.clear();
    });
    await PrefsStore.setLastCategory(category.id);
  }

  /// 点某一行 → 把它变成源单位。
  ///
  /// 关键是**把这一行当前显示的数值搬进输入框**：用户看到的 5000 g 点一下变成
  /// 源单位后还是 5000 g，而不是突然变回原来输入的 5 kg。数值的含义不跳，
  /// 才敢连着点两下换算链。
  void _selectUnit(Unit unit, double displayedValue) {
    if (unit.id == _unit.id) return;
    HapticFeedback.selectionClick();
    setState(() {
      _unit = unit;
      _entry.setValue(_plain(displayedValue));
    });
    PrefsStore.setLastUnit(_category.id, unit.id);
  }

  /// 搬进输入框的写法：不带千分位、不带科学计数法 —— 输入框只认纯数字串。
  String _plain(double value) {
    if (!value.isFinite) return '0';
    final text = value.toStringAsFixed(10);
    var out = text;
    while (out.contains('.') && (out.endsWith('0') || out.endsWith('.'))) {
      out = out.substring(0, out.length - 1);
    }
    return out.isEmpty ? '0' : out;
  }

  void _onKey(KeyPress press) {
    setState(() {
      switch (press.action) {
        case KeyAction.digit:
          _entry.digit(press.digit!);
        case KeyAction.dot:
          _entry.dot();
        case KeyAction.sign:
          _entry.toggleSign();
        case KeyAction.backspace:
          _entry.backspace();
        case KeyAction.clear:
          _entry.clear();
      }
    });
  }

  Future<void> _copy(Unit unit, double value) async {
    await Clipboard.setData(
      ClipboardData(text: '${NumberFormatter.format(value)} ${unit.symbol}'),
    );
    await HapticFeedback.mediumImpact();
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..clearSnackBars()
      ..showSnackBar(
        SnackBar(
          content: Text('Copied ${unit.name.toLowerCase()} value'),
          duration: const Duration(milliseconds: 1200),
          behavior: SnackBarBehavior.floating,
          backgroundColor: AppColors.primaryText,
        ),
      );
  }

  @override
  Widget build(BuildContext context) {
    final values = _category.convertAll(_entry.value, _unit);

    return Scaffold(
      backgroundColor: AppColors.background,
      body: AnnotatedRegion<SystemUiOverlayStyle>(
        // 浅色底必须配深色状态栏图标，否则系统图标会白底白字看不见。
        value: SystemUiOverlayStyle.dark.copyWith(
          statusBarColor: Colors.transparent,
        ),
        child: SafeArea(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              _header(),
              _categoryStrip(),
              const SizedBox(height: 10),
              Expanded(child: _unitList(values)),
              Padding(
                padding: const EdgeInsets.fromLTRB(10, 10, 10, 8),
                child: Keypad(onKey: _onKey),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _header() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: <Widget>[
          const Expanded(
            child: Text(
              'UnitShift',
              style: TextStyle(
                color: AppColors.primaryText,
                fontSize: 22,
                fontWeight: FontWeight.w700,
                letterSpacing: -0.3,
              ),
            ),
          ),
          // 当前输入。数字用等宽字形，边打边不会左右抖。
          Flexible(
            flex: 3,
            child: FittedBox(
              fit: BoxFit.scaleDown,
              alignment: Alignment.centerRight,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.baseline,
                textBaseline: TextBaseline.alphabetic,
                children: <Widget>[
                  Text(
                    _entry.text,
                    style: const TextStyle(
                      color: AppColors.primaryText,
                      fontSize: 30,
                      fontWeight: FontWeight.w700,
                      height: 1.0,
                      fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                    ),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    _unit.symbol,
                    style: const TextStyle(
                      color: AppColors.accent,
                      fontSize: 16,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _categoryStrip() {
    return SizedBox(
      height: 36,
      child: ListView.separated(
        controller: _categoryScroll,
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        itemCount: Catalog.categories.length,
        separatorBuilder: (_, _) => const SizedBox(width: 8),
        itemBuilder: (context, i) {
          final category = Catalog.categories[i];
          final active = category.id == _category.id;
          return GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: () => _selectCategory(category),
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 160),
              padding: const EdgeInsets.symmetric(horizontal: 14),
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: active ? AppColors.accent : AppColors.surface,
                borderRadius: BorderRadius.circular(18),
                border: Border.all(
                  color: active ? AppColors.accent : AppColors.border,
                ),
              ),
              child: Text(
                category.label,
                style: TextStyle(
                  color: active ? Colors.white : AppColors.mutedText,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  /// 行高按可用空间分摊，卡片不撑满也不拖白。
  ///
  /// 各类的单位数差得很远（温度 4 个，体积 11 个）。固定行高的话，
  /// 温度那一屏卡片只占顶上一小条、底下拖一大片空白，看着像没加载完；
  /// 而让卡片硬撑满，又会变成一大块什么都没有的白。
  /// 折中办法是把剩余高度摊到各行上，并卡在一个上下限里 ——
  /// 少的类行更松，多的类行更紧、超出就滚动。
  Widget _unitList(List<double> values) {
    const minRowHeight = 46.0;
    const maxRowHeight = 76.0;

    return LayoutBuilder(
      builder: (context, constraints) {
        final count = _category.units.length;
        final dividers = count - 1;
        final rowHeight = ((constraints.maxHeight - dividers) / count)
            .clamp(minRowHeight, maxRowHeight);

        final rows = <Widget>[];
        for (var i = 0; i < count; i++) {
          if (i > 0) {
            rows.add(
              const Divider(height: 1, thickness: 1, color: AppColors.divider),
            );
          }
          final unit = _category.units[i];
          final value = values[i];
          rows.add(
            _UnitRow(
              unit: unit,
              value: value,
              height: rowHeight,
              active: unit.id == _unit.id,
              onTap: () => _selectUnit(unit, value),
              onLongPress: () => _copy(unit, value),
            ),
          );
        }

        return SingleChildScrollView(
          child: Container(
            margin: const EdgeInsets.symmetric(horizontal: 12),
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: AppColors.border),
            ),
            clipBehavior: Clip.antiAlias,
            child: Column(mainAxisSize: MainAxisSize.min, children: rows),
          ),
        );
      },
    );
  }
}

class _UnitRow extends StatelessWidget {
  const _UnitRow({
    required this.unit,
    required this.value,
    required this.height,
    required this.active,
    required this.onTap,
    required this.onLongPress,
  });

  final Unit unit;
  final double value;
  final double height;
  final bool active;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      onLongPress: onLongPress,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 160),
        height: height,
        color: active ? AppColors.accentSoft : AppColors.surface,
        padding: const EdgeInsets.symmetric(horizontal: 14),
        child: Row(
          children: <Widget>[
            SizedBox(
              width: 62,
              child: Text(
                unit.symbol,
                style: TextStyle(
                  color: active ? AppColors.accent : AppColors.primaryText,
                  fontSize: 14,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            Expanded(
              child: Text(
                unit.name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 12.5,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
            const SizedBox(width: 10),
            // 数值右对齐 + 等宽字形：整列小数点对得上，扫一眼就能比大小
            Text(
              NumberFormatter.format(value),
              style: TextStyle(
                color: active ? AppColors.accent : AppColors.primaryText,
                fontSize: 15.5,
                fontWeight: FontWeight.w700,
                fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
