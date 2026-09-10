// 私有字段用命名参数赋值，只能写成 `_x = x` —— 命名参数不许以下划线开头，
// 所以 prefer_initializing_formals 建议的 `{this._factor}` 在这里是非法的。
// ignore_for_file: prefer_initializing_formals

/// 一个单位。
///
/// 换算一律走「先折成基准单位，再从基准单位折过去」两步，而不是给每一对单位
/// 都存一个系数 —— n 个单位那样要存 n² 个系数，加一个单位就要补一整排，
/// 而且任何一个抄错都只在某一对组合上出错，极难发现。
class Unit {
  /// 线性单位：`基准值 = 本单位值 × factor`。绝大多数单位都是这一类。
  const Unit({
    required this.id,
    required this.name,
    required this.symbol,
    required double factor,
  })  : _factor = factor,
        _toBase = null,
        _fromBase = null;

  /// 非线性单位：温度那种带零点偏移的，系数表达不了，只能给一对互逆函数。
  const Unit.custom({
    required this.id,
    required this.name,
    required this.symbol,
    required double Function(double) toBase,
    required double Function(double) fromBase,
  })  : _factor = null,
        _toBase = toBase,
        _fromBase = fromBase;

  /// 稳定标识。存档里记的是它，所以**改了名字可以，改 id 会让用户的偏好失效**。
  final String id;

  /// 展示名（英文，跟随商店语言）。
  final String name;

  /// 简写。列表右侧显示的就是它。
  final String symbol;

  final double? _factor;
  final double Function(double)? _toBase;
  final double Function(double)? _fromBase;

  /// 是不是这一类的基准单位（系数正好 1）。
  bool get isBase => _factor == 1.0;

  double toBase(double value) =>
      _factor != null ? value * _factor : _toBase!(value);

  double fromBase(double value) =>
      _factor != null ? value / _factor : _fromBase!(value);
}

/// 一类单位（长度、质量、温度……）。
class UnitCategory {
  const UnitCategory({
    required this.id,
    required this.label,
    required this.units,
    required this.defaultUnitId,
  });

  final String id;
  final String label;
  final List<Unit> units;

  /// 首次进入这一类时选中的单位。
  final String defaultUnitId;

  Unit? byId(String id) {
    for (final u in units) {
      if (u.id == id) return u;
    }
    return null;
  }

  /// 把 [value]（以 [from] 为单位）换算成这一类的所有单位。
  ///
  /// 返回的顺序与 [units] 一致，含 [from] 自己 —— 界面上把源单位也留在列表里，
  /// 位置不跳动，眼睛不用重新找。
  List<double> convertAll(double value, Unit from) {
    final base = from.toBase(value);
    return <double>[for (final u in units) u.fromBase(base)];
  }
}
