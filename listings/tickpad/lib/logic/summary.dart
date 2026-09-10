import 'counter.dart';
import 'session.dart';

/// 把一屏计数器（或一次会话）排成一段可以直接发出去的纯文本。
///
/// 盘点完总要把数字报给别人 —— 发微信、贴进表格、抄进单子。
/// 让用户对着屏幕一个个手抄是这类工具最典型的断点，复制一段现成的文本就接上了。
///
/// 名字左对齐、数字右对齐，两列之间用空格补齐 ——
/// 等宽字体下看着是张表，非等宽下也仍然读得通。
class Summary {
  Summary._();

  static String forCounters(List<Counter> counters, DateTime at) {
    return _render(
      <(String, int)>[for (final c in counters) (c.label, c.value)],
      at,
    );
  }

  static String forSession(Session session) {
    return _render(
      <(String, int)>[for (final e in session.entries) (e.label, e.value)],
      session.savedAt,
    );
  }

  static String _render(List<(String, int)> rows, DateTime at) {
    final buffer = StringBuffer()..writeln('TickPad · ${formatDateTime(at)}');
    if (rows.isEmpty) {
      buffer.writeln('(no counters)');
      return buffer.toString().trimRight();
    }

    var total = 0;
    // 名字列宽取最长的那个，「Total」自己也要算进去，免得末行对不齐
    var labelWidth = 'Total'.length;
    for (final row in rows) {
      if (row.$1.length > labelWidth) labelWidth = row.$1.length;
      total += row.$2;
    }
    var valueWidth = '$total'.length;
    for (final row in rows) {
      final w = '${row.$2}'.length;
      if (w > valueWidth) valueWidth = w;
    }

    for (final row in rows) {
      buffer.writeln(
        '${row.$1.padRight(labelWidth)}  ${'${row.$2}'.padLeft(valueWidth)}',
      );
    }
    buffer.write(
      '${'Total'.padRight(labelWidth)}  ${'$total'.padLeft(valueWidth)}',
    );
    return buffer.toString();
  }

  /// `2026-09-09 14:32`。不引 intl 只为格式化一个时间戳 ——
  /// 那个包会让 APK 明显变大，而这里只要一种固定写法。
  static String formatDateTime(DateTime at) {
    String two(int n) => n.toString().padLeft(2, '0');
    return '${at.year}-${two(at.month)}-${two(at.day)} '
        '${two(at.hour)}:${two(at.minute)}';
  }
}
