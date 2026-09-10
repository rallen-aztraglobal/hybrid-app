/// 输入框里那串数字的状态机。
///
/// 自己维护一个字符串而不是用 [TextField]，是因为这里要的是**计算器式**的输入：
/// 固定的自定义键盘、没有光标、不弹系统键盘（系统键盘一弹布局就跳，
/// 而且各家输入法的数字键位差别很大，还可能塞进全角字符）。
///
/// 保持字符串而不是 double，是为了让「1.」「0.50」这类中间态原样显示 ——
/// 每敲一位就 parse 回 double 再转回字符串的话，小数点会被吃掉、
/// 末尾的零也留不住，输入过程会跳来跳去。
class NumericEntry {
  NumericEntry([String initial = '0']) : _text = initial;

  /// 位数上限。double 只有约 15~17 位十进制精度，再长也没意义，
  /// 而且显示区容不下。
  static const int maxDigits = 15;

  String _text;

  String get text => _text;

  /// 当前值。空串、单个负号、单个小数点这些中间态一律当 0。
  double get value => double.tryParse(_text) ?? 0;

  bool get isZero => value == 0;

  /// 数字位数（不含符号和小数点）。
  int get _digitCount =>
      _text.replaceAll('-', '').replaceAll('.', '').length;

  /// 追加一位数字。返回 false 表示这一下没有产生变化。
  bool digit(String d) {
    if (_digitCount >= maxDigits) return false;
    // 前导零直接被顶掉：敲 5 得到 "5" 而不是 "05"
    if (_text == '0') {
      _text = d;
      return true;
    }
    if (_text == '-0') {
      _text = '-$d';
      return true;
    }
    _text += d;
    return true;
  }

  /// 小数点。已经有一个就忽略。
  bool dot() {
    if (_text.contains('.')) return false;
    _text += '.';
    return true;
  }

  /// 退格。删到空就回到 "0"。
  bool backspace() {
    if (_text.length <= 1 || (_text.length == 2 && _text.startsWith('-'))) {
      if (_text == '0') return false;
      _text = '0';
      return true;
    }
    _text = _text.substring(0, _text.length - 1);
    if (_text == '-') _text = '0';
    return true;
  }

  bool clear() {
    if (_text == '0') return false;
    _text = '0';
    return true;
  }

  /// 正负号。0 也允许带负号（"-0"），继续输入会变成 "-5"。
  bool toggleSign() {
    _text = _text.startsWith('-') ? _text.substring(1) : '-$_text';
    return true;
  }

  /// 用一个现成的数值替换全部内容 —— 点某一行把它设为源单位时用。
  void setValue(String text) => _text = text;
}
