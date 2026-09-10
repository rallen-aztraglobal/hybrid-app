import 'package:flutter_test/flutter_test.dart';
import 'package:nonopix/logic/line_solver.dart';

void main() {
  group('cluesOf', () {
    test('空行的线索是 [0]', () {
      expect(cluesOf(<bool>[false, false, false]), <int>[0]);
    });

    test('整行涂满就是一段', () {
      expect(cluesOf(<bool>[true, true, true]), <int>[3]);
    });

    test('中间断开算两段', () {
      expect(cluesOf(<bool>[true, true, false, true]), <int>[2, 1]);
    });

    test('首尾的空格不产生段', () {
      expect(cluesOf(<bool>[false, true, false]), <int>[1]);
    });
  });

  group('enumerateLines', () {
    test('线索 [0] 只有全空一种涂法', () {
      final all = enumerateLines(<int>[0], 4);
      expect(all.length, 1);
      expect(all.single, <bool>[false, false, false, false]);
    });

    test('段刚好占满时唯一', () {
      final all = enumerateLines(<int>[2, 1], 4);
      expect(all.length, 1);
      expect(all.single, <bool>[true, true, false, true]);
    });

    test('段之间强制留至少一格空白', () {
      for (final line in enumerateLines(<int>[1, 1], 4)) {
        // 不允许出现相邻两格同时涂黑 —— 那样两段就并成一段了
        for (var i = 0; i + 1 < line.length; i++) {
          expect(line[i] && line[i + 1], isFalse);
        }
      }
    });

    test('枚举数量符合组合数', () {
      // 线索 [1,1] 放进 4 格：C(3,2) = 3 种
      expect(enumerateLines(<int>[1, 1], 4).length, 3);
      // 线索 [1] 放进 5 格：5 种
      expect(enumerateLines(<int>[1], 5).length, 5);
    });

    test('放不下时返回空', () {
      expect(enumerateLines(<int>[3, 3], 5), isEmpty);
    });

    test('每种涂法反推回去都等于原线索', () {
      for (final line in enumerateLines(<int>[2, 1], 6)) {
        expect(cluesOf(line), <int>[2, 1]);
      }
    });
  });

  group('refineLine', () {
    test('唯一涂法时全部格子被确定', () {
      final r = refineLine(<int>[2, 1], List<bool?>.filled(4, null));
      expect(r, <bool?>[true, true, false, true]);
    });

    test('取候选的交集，两可的格子留 null', () {
      // [1] 放进 3 格有 3 种，没有任何一格是所有候选都相同的
      final r = refineLine(<int>[1], List<bool?>.filled(3, null));
      expect(r, <bool?>[null, null, null]);
    });

    test('长段会迫出中间必涂的格子', () {
      // [4] 放进 5 格：两种位置，中间三格无论如何都涂黑
      final r = refineLine(<int>[4], List<bool?>.filled(5, null));
      expect(r, <bool?>[null, true, true, true, null]);
    });

    test('已知信息能进一步收紧结论', () {
      // [1] 放进 3 格，已知第一格是空 → 只剩两种，仍不确定
      var r = refineLine(<int>[1], <bool?>[false, null, null]);
      expect(r, <bool?>[false, null, null]);
      // 再已知最后一格也是空 → 唯一解
      r = refineLine(<int>[1], <bool?>[false, null, false]);
      expect(r, <bool?>[false, true, false]);
    });

    test('与线索矛盾时返回 null', () {
      // [3] 要求连续三格，但中间被判定为空
      expect(refineLine(<int>[3], <bool?>[null, false, null]), isNull);
    });

    test('不会产出与已知冲突的结论', () {
      final known = <bool?>[true, null, null, null];
      final r = refineLine(<int>[1, 1], known)!;
      expect(r[0], isTrue); // 已知的不会被推翻
    });
  });
}
