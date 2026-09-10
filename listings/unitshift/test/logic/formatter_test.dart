import 'package:flutter_test/flutter_test.dart';
import 'package:unitshift/logic/formatter.dart';

/// 显示格式的验收。算得对但显示成 `2.5399999999999996`，用户只会以为 App 坏了。
void main() {
  String f(double v) => NumberFormatter.format(v);

  group('浮点噪声不能露出来', () {
    test('0.1 + 0.2 显示成 0.3', () {
      expect(f(0.1 + 0.2), '0.3');
    });

    test('1 in 换成 cm 显示成 2.54', () {
      expect(f(1 * 0.0254 / 0.01), '2.54');
    });

    test('往返换算带回来的 0.99999999… 显示成 1', () {
      expect(f(0.9999999999), '1');
      expect(f(1.0000000001), '1');
    });

    test('末尾的零一律去掉', () {
      expect(f(2.500), '2.5');
      expect(f(7.0), '7');
    });
  });

  group('千分位', () {
    test('整数部分每三位一个逗号', () {
      expect(f(1000), '1,000');
      expect(f(1234567), '1,234,567');
      expect(f(100), '100');
    });

    test('小数部分不加逗号', () {
      expect(f(1609.344), '1,609.344');
    });

    test('负数的逗号加在符号后面', () {
      expect(f(-1234.5), '-1,234.5');
    });
  });

  group('量级', () {
    test('大整数照原样给，不截成有效数字', () {
      // 1 TiB 的字节数是精确值，显示成 1.0995e12 反而不如原样有用
      expect(f(1099511627776), '1,099,511,627,776');
    });

    test('极小的数走科学计数法，不会被抹成 0', () {
      // 这是最要命的一种错：把值丢了还显示得像个正常结果
      expect(f(1e-9), '1e-9');
      expect(f(2.5e-12), '2.5e-12');
      expect(f(0), '0');
      expect(f(1e-9) == '0', isFalse);
    });

    test('极大的数走科学计数法', () {
      expect(f(1e18), '1e18');
    });

    test('临界点两侧都读得动', () {
      expect(f(0.00001), '0.00001'); // 刚好在阈值上，仍用普通写法
      expect(f(0.000001), '1e-6'); // 越过阈值切科学计数法
    });
  });

  group('边角情况', () {
    test('零就是 0，不带符号', () {
      expect(f(0), '0');
      expect(f(-0.0), '0');
    });

    test('四舍五入到 0 的负数不显示成 -0', () {
      expect(f(-0.000000000001 * 1e6), isNot(startsWith('-0')));
    });

    test('非数与无穷显示成占位符，不抛异常', () {
      expect(f(double.nan), '—');
      expect(f(double.infinity), '—');
      expect(f(double.negativeInfinity), '—');
    });

    test('有效数字位数可调', () {
      expect(NumberFormatter.format(1 / 3, significantDigits: 3), '0.333');
      expect(NumberFormatter.format(1 / 3, significantDigits: 8), '0.33333333');
    });
  });
}
