import 'package:calculator_app/logic/calculator_logic.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CalculatorLogic', () {
    late CalculatorLogic logic;

    setUp(() {
      logic = CalculatorLogic();
    });

    void type(String input) {
      for (final char in input.split('')) {
        switch (char) {
          case '+':
          case '-':
          case '*':
          case '/':
            logic.setOperator({'+': '+', '-': '-', '*': '×', '/': '÷'}[char]!);
            break;
          case '.':
            logic.inputDot();
            break;
          case '=':
            logic.calculateEquals();
            break;
          default:
            logic.inputDigit(char);
        }
      }
    }

    test('initial display is 0', () {
      expect(logic.display, '0');
    });

    test('addition', () {
      type('2+3=');
      expect(logic.display, '5');
    });

    test('subtraction', () {
      type('9-4=');
      expect(logic.display, '5');
    });

    test('multiplication', () {
      type('6*7=');
      expect(logic.display, '42');
    });

    test('division', () {
      type('20/4=');
      expect(logic.display, '5');
    });

    test('decimal input and arithmetic', () {
      type('1.5+2.25=');
      expect(logic.display, '3.75');
    });

    test('does not allow a second decimal point in the same number', () {
      type('1.2.3');
      expect(logic.display, '1.23');
    });

    test('percent on a standalone number divides by 100', () {
      logic.inputDigit('5');
      logic.inputDigit('0');
      logic.inputPercent();
      expect(logic.display, '0.5');
    });

    test('percent within an operation is relative to the accumulator', () {
      type('200+10');
      logic.inputPercent();
      expect(logic.display, '20');
      logic.calculateEquals();
      expect(logic.display, '220');
    });

    test('continuous (chained) calculation', () {
      type('2+3+4=');
      expect(logic.display, '9');
    });

    test('toggling sign switches between positive and negative', () {
      logic.inputDigit('5');
      logic.toggleSign();
      expect(logic.display, '-5');
      logic.toggleSign();
      expect(logic.display, '5');
    });

    test('AC clears the calculator back to its initial state', () {
      type('12+7');
      logic.clearAll();
      expect(logic.display, '0');
      expect(logic.pendingOperator, isNull);
    });

    test('backspace removes the last digit', () {
      logic.inputDigit('1');
      logic.inputDigit('2');
      logic.inputDigit('3');
      logic.delete();
      expect(logic.display, '12');
    });

    test('backspace on a single digit resets display to 0', () {
      logic.inputDigit('7');
      logic.delete();
      expect(logic.display, '0');
    });

    test('division by zero shows a friendly error message', () {
      type('5/0=');
      expect(logic.hasError, isTrue);
      expect(logic.display, CalculatorLogic.divideByZeroError);
    });

    test('entering a new digit after an error resets the calculator', () {
      type('5/0=');
      logic.inputDigit('9');
      expect(logic.hasError, isFalse);
      expect(logic.display, '9');
    });
  });
}
