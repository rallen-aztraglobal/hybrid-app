import 'dart:math' as math;

/// 数字显示。
///
/// 换算 App 的成败一半在这里。直接 `toString()` 会满屏都是
/// `0.30000000000000004`、`2.5399999999999996` 这种浮点噪声 ——
/// 算得对但看着像坏了。
///
/// 规则：
/// - 按**有效数字**决定保留几位小数，而不是固定小数位。
///   固定两位的话，0.0001 会显示成 0.00，大数又会拖一串没意义的零。
/// - 大整数照原样显示，不截成有效数字。1 TiB = 1099511627776 B 是精确值，
///   显示成 1.0995e12 反而不如原样有用。
/// - 只在真正读不动的量级才切科学计数法。
class NumberFormatter {
  NumberFormatter._();

  /// 默认有效数字位数。double 有约 15~17 位十进制精度，取 8 位远在安全线内，
  /// 换算带来的末位误差不会露出来。
  static const int defaultSignificantDigits = 8;

  /// 超出这个范围就切科学计数法。
  ///
  /// 下界 1e-5 是和 [_maxDecimals] 绑死的：小数位最多留 12 位，
  /// 比 1e-12 更小的数会被四舍五入成 0 —— 那是**把值丢了**，比难看严重得多。
  /// 留足余量取 1e-5。
  static const double _scientificBelow = 1e-5;
  static const double _scientificAbove = 1e15;
  static const int _maxDecimals = 12;

  static String format(
    double value, {
    int significantDigits = defaultSignificantDigits,
  }) {
    if (value.isNaN || value.isInfinite) return '—';
    if (value == 0) return '0'; // 顺带把 -0.0 归一成 0

    final abs = value.abs();
    if (abs < _scientificBelow || abs >= _scientificAbove) {
      return _scientific(value, significantDigits);
    }

    // 整数部分有几位（abs < 1 时为 0 或负，代表前导零）
    final intDigits = (math.log(abs) / math.ln10).floor() + 1;
    final decimals = (significantDigits - intDigits).clamp(0, _maxDecimals);

    final fixed = value.toStringAsFixed(decimals);
    return _group(_stripTrailingZeros(fixed));
  }

  /// 去掉小数点后多余的零，以及只剩个光杆的小数点。
  static String _stripTrailingZeros(String s) {
    if (!s.contains('.')) return s;
    var out = s;
    while (out.endsWith('0')) {
      out = out.substring(0, out.length - 1);
    }
    if (out.endsWith('.')) out = out.substring(0, out.length - 1);
    // 四舍五入后可能变成 "-0"
    return out == '-0' ? '0' : out;
  }

  /// 整数部分每三位加一个千分位逗号。小数部分不加。
  static String _group(String s) {
    final negative = s.startsWith('-');
    final body = negative ? s.substring(1) : s;
    final dot = body.indexOf('.');
    final intPart = dot < 0 ? body : body.substring(0, dot);
    final rest = dot < 0 ? '' : body.substring(dot);

    final buffer = StringBuffer();
    for (var i = 0; i < intPart.length; i++) {
      if (i > 0 && (intPart.length - i) % 3 == 0) buffer.write(',');
      buffer.write(intPart[i]);
    }
    return '${negative ? '-' : ''}$buffer$rest';
  }

  static String _scientific(double value, int significantDigits) {
    final abs = value.abs();
    var exponent = (math.log(abs) / math.ln10).floor();
    var mantissa = value / math.pow(10, exponent);

    // log 的舍入会让尾数落在 [1,10) 之外一点点，归一回去
    if (mantissa.abs() >= 10) {
      mantissa /= 10;
      exponent += 1;
    } else if (mantissa.abs() < 1) {
      mantissa *= 10;
      exponent -= 1;
    }

    final digits = (significantDigits - 1).clamp(0, 10);
    final text = _stripTrailingZeros(mantissa.toStringAsFixed(digits));
    return '${text}e$exponent';
  }
}
