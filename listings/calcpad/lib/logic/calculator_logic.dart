/// Pure calculation engine for the calculator app.
///
/// This class holds no Flutter/UI dependencies so it can be unit tested
/// in isolation from the widget tree.
class CalculatorLogic {
  static const String initialDisplay = '0';
  static const String divideByZeroError = '错误：不能除以0';
  static const int _maxDigits = 15;

  String _display = initialDisplay;
  double _accumulator = 0;
  String? _pendingOperator;
  bool _isEnteringNewNumber = true;
  bool _hasError = false;

  /// The text that should be shown in the main display.
  String get display => _display;

  /// The operator waiting to be applied (null when no calculation pending).
  String? get pendingOperator => _pendingOperator;

  /// Whether the calculator is currently showing an error state.
  bool get hasError => _hasError;

  /// A small preview of the in-progress equation, e.g. "12 +", shown above
  /// the main display. Empty when there is nothing pending.
  String get equationPreview {
    if (_pendingOperator == null) return '';
    return '${_formatNumber(_accumulator)} $_pendingOperator';
  }

  /// Appends a digit ('0'-'9') to the current input.
  void inputDigit(String digit) {
    if (_hasError) _reset();

    if (_isEnteringNewNumber) {
      _display = digit == '0' ? '0' : digit;
      _isEnteringNewNumber = false;
      return;
    }

    if (_display == '0') {
      _display = digit == '0' ? '0' : digit;
      return;
    }

    final digitCount = _display.replaceAll(RegExp(r'[-.]'), '').length;
    if (digitCount >= _maxDigits) return;

    _display += digit;
  }

  /// Inserts a decimal point, if one is not already present.
  void inputDot() {
    if (_hasError) _reset();

    if (_isEnteringNewNumber) {
      _display = '0.';
      _isEnteringNewNumber = false;
      return;
    }

    if (!_display.contains('.')) {
      _display += '.';
    }
  }

  /// Toggles the sign of the currently displayed number.
  void toggleSign() {
    if (_hasError) return;
    if (_display == '0') return;

    if (_display.startsWith('-')) {
      _display = _display.substring(1);
    } else {
      _display = '-$_display';
    }
  }

  /// Converts the current number to a percentage. When there is a pending
  /// operator, the percentage is taken relative to the accumulated value
  /// (e.g. 200 + 10% -> 20), matching common calculator behavior.
  void inputPercent() {
    if (_hasError) return;

    final current = double.tryParse(_display) ?? 0;
    final result = _pendingOperator != null
        ? _accumulator * current / 100
        : current / 100;

    _display = _formatNumber(result);
    _isEnteringNewNumber = true;
  }

  /// Deletes the last character of the current input (backspace).
  void delete() {
    if (_hasError) {
      _reset();
      return;
    }

    if (_isEnteringNewNumber) return;

    final isNegativeSingleDigit =
        _display.length == 2 && _display.startsWith('-');
    if (_display.length <= 1 || isNegativeSingleDigit) {
      _display = initialDisplay;
      _isEnteringNewNumber = true;
    } else {
      _display = _display.substring(0, _display.length - 1);
    }
  }

  /// Clears all calculator state (the "AC" button).
  void clearAll() {
    _display = initialDisplay;
    _accumulator = 0;
    _pendingOperator = null;
    _isEnteringNewNumber = true;
    _hasError = false;
  }

  /// Registers an operator ('+', '-', '×', '÷') to be applied between the
  /// previously entered number and the next one.
  void setOperator(String op) {
    if (_hasError) return;

    final current = double.tryParse(_display) ?? 0;

    if (_pendingOperator != null && !_isEnteringNewNumber) {
      final result = _compute(_accumulator, current, _pendingOperator!);
      if (result == null) {
        _setDivideByZeroError();
        return;
      }
      _accumulator = result;
      _display = _formatNumber(result);
    } else {
      _accumulator = current;
    }

    _pendingOperator = op;
    _isEnteringNewNumber = true;
  }

  /// Evaluates the pending operation (the "=" button).
  void calculateEquals() {
    if (_hasError) return;
    if (_pendingOperator == null) return;

    final current = double.tryParse(_display) ?? 0;
    final result = _compute(_accumulator, current, _pendingOperator!);
    if (result == null) {
      _setDivideByZeroError();
      return;
    }

    _display = _formatNumber(result);
    _accumulator = result;
    _pendingOperator = null;
    _isEnteringNewNumber = true;
  }

  double? _compute(double a, double b, String op) {
    switch (op) {
      case '+':
        return a + b;
      case '-':
        return a - b;
      case '×':
        return a * b;
      case '÷':
        if (b == 0) return null;
        return a / b;
      default:
        return b;
    }
  }

  void _setDivideByZeroError() {
    _display = divideByZeroError;
    _hasError = true;
    _pendingOperator = null;
    _isEnteringNewNumber = true;
  }

  void _reset() {
    _display = initialDisplay;
    _hasError = false;
    _isEnteringNewNumber = true;
  }

  String _formatNumber(double value) {
    if (!value.isFinite) return divideByZeroError;

    if (value == value.truncateToDouble() && value.abs() < 1e15) {
      return value.toStringAsFixed(0);
    }

    var text = value.toStringAsFixed(10);
    if (text.contains('.')) {
      text = text.replaceFirst(RegExp(r'0+$'), '');
      text = text.replaceFirst(RegExp(r'\.$'), '');
    }
    return text;
  }
}
