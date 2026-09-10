import 'package:flutter_test/flutter_test.dart';
import 'package:unitshift/logic/entry.dart';

/// 输入框状态机。这类代码看着简单，边角情况最多 ——
/// 前导零、只剩一个负号、连打两个小数点，哪一个漏了都会当场显示出一串非法数字。
void main() {
  group('输入数字', () {
    test('前导零被顶掉', () {
      final e = NumericEntry();
      expect(e.text, '0');
      e.digit('5');
      expect(e.text, '5');
      e.digit('0');
      expect(e.text, '50');
    });

    test('位数到上限就不再接受', () {
      final e = NumericEntry();
      for (var i = 0; i < NumericEntry.maxDigits + 5; i++) {
        e.digit('9');
      }
      expect(e.text.length, NumericEntry.maxDigits);
      expect(e.digit('9'), isFalse);
    });
  });

  group('小数点', () {
    test('只能有一个', () {
      final e = NumericEntry();
      e.digit('1');
      expect(e.dot(), isTrue);
      expect(e.dot(), isFalse);
      expect(e.text, '1.');
    });

    test('半截的小数（"1."）当 1 用，不炸', () {
      final e = NumericEntry();
      e.digit('1');
      e.dot();
      expect(e.value, 1);
    });

    test('末尾的零在输入过程中留得住', () {
      // 每敲一位就 parse 回 double 的话 "0.50" 会被吃成 "0.5"，
      // 再敲一位就变成 "0.53" 而不是 "0.503" —— 输入过程会跳。
      final e = NumericEntry();
      e.dot();
      e.digit('5');
      e.digit('0');
      expect(e.text, '0.50');
      e.digit('3');
      expect(e.text, '0.503');
    });
  });

  group('退格', () {
    test('一位一位删，删空回到 0', () {
      final e = NumericEntry();
      e.digit('1');
      e.digit('2');
      e.backspace();
      expect(e.text, '1');
      e.backspace();
      expect(e.text, '0');
    });

    test('已经是 0 时退格没有反应', () {
      final e = NumericEntry();
      expect(e.backspace(), isFalse);
    });

    test('删到只剩一个负号时归零，不会停在 "-" 上', () {
      final e = NumericEntry();
      e.digit('7');
      e.toggleSign();
      expect(e.text, '-7');
      e.backspace();
      expect(e.text, '0');
      expect(e.value, 0);
    });
  });

  group('正负号', () {
    test('来回切换', () {
      final e = NumericEntry();
      e.digit('4');
      e.toggleSign();
      expect(e.value, -4);
      e.toggleSign();
      expect(e.value, 4);
    });

    test('先按负号再输入数字（温度常这么打）', () {
      final e = NumericEntry();
      e.toggleSign();
      expect(e.text, '-0');
      e.digit('4');
      e.digit('0');
      expect(e.text, '-40');
      expect(e.value, -40);
    });
  });

  group('清空与外部赋值', () {
    test('clear 回到 0', () {
      final e = NumericEntry();
      e.digit('9');
      expect(e.clear(), isTrue);
      expect(e.text, '0');
      expect(e.clear(), isFalse);
    });

    test('setValue 直接替换 —— 点某一行改源单位时用', () {
      final e = NumericEntry();
      e.setValue('1609.344');
      expect(e.value, 1609.344);
    });
  });
}
