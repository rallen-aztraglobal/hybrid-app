import 'package:flutter_test/flutter_test.dart';
import 'package:tickpad/logic/counter.dart';

/// 单个计数器。重点在**存档读回来这一侧** ——
/// 计数是用户唯一的数据，一条读坏了不能带崩整份。
void main() {
  group('加减', () {
    test('按步长加减', () {
      final c = Counter(id: 'a', label: 'A', step: 5);
      c.bump(1);
      expect(c.value, 5);
      c.bump(2);
      expect(c.value, 15);
      c.bump(-1);
      expect(c.value, 10);
    });

    test('减不到负数 —— 连点减号是误触，不该让人再点一路加回来', () {
      final c = Counter(id: 'a', label: 'A', value: 1);
      c.bump(-1);
      expect(c.value, 0);
      c.bump(-1);
      c.bump(-1);
      expect(c.value, 0);
    });

    test('步长大于当前值时也夹在 0', () {
      final c = Counter(id: 'a', label: 'A', value: 3, step: 10);
      c.bump(-1);
      expect(c.value, 0);
    });

    test('reset 归零', () {
      final c = Counter(id: 'a', label: 'A', value: 42);
      c.reset();
      expect(c.value, 0);
    });
  });

  group('存档往返', () {
    test('正常的一条原样回来', () {
      final c = Counter(id: 'x', label: 'Boxes', value: 12, step: 6);
      final back = Counter.fromJson(c.toJson());
      expect(back.id, 'x');
      expect(back.label, 'Boxes');
      expect(back.value, 12);
      expect(back.step, 6);
    });

    test('copy 是深拷贝 —— 撤销快照靠它，不能是同一个对象', () {
      final c = Counter(id: 'x', label: 'A', value: 3);
      final copy = c.copy();
      copy.bump(1);
      expect(c.value, 3, reason: '改副本不该影响原件');
    });
  });

  group('坏存档', () {
    test('缺字段 → 各自退回默认值', () {
      final c = Counter.fromJson(<String, dynamic>{'id': 'x'});
      expect(c.id, 'x');
      expect(c.label, 'Counter');
      expect(c.value, 0);
      expect(c.step, 1);
    });

    test('类型不对 → 当没有', () {
      final c = Counter.fromJson(<String, dynamic>{
        'id': 42,
        'label': <String>['nope'],
        'value': '7',
        'step': 0.5,
      });
      expect(c.id, isNotEmpty, reason: 'id 坏了也要凑一个出来，不能丢这条记录');
      expect(c.label, 'Counter');
      expect(c.value, 0);
      expect(c.step, 1);
    });

    test('负数计数 / 零步长这类越界值被纠正', () {
      final c = Counter.fromJson(<String, dynamic>{
        'id': 'x',
        'value': -5,
        'step': 0,
      });
      expect(c.value, 0);
      expect(c.step, 1);
    });

    test('空名字当没填', () {
      final c = Counter.fromJson(<String, dynamic>{'id': 'x', 'label': '   '});
      expect(c.label, 'Counter');
    });

    test('旧存档没有 target / color 这两个键 —— 这是正常情况，不是损坏', () {
      // 这两个字段是后加的。老用户升级上来，存档里根本没有它们，
      // 必须当作「没设」而不是「读坏了」。
      final c = Counter.fromJson(<String, dynamic>{
        'id': 'x',
        'label': 'Boxes',
        'value': 12,
        'step': 1,
      });
      expect(c.value, 12, reason: '旧数据要原样保住');
      expect(c.hasTarget, isFalse);
      expect(c.colorIndex, 0);
    });

    test('新字段往返', () {
      final c = Counter(
        id: 'x',
        label: 'A',
        value: 5,
        target: 20,
        colorIndex: 3,
      );
      final back = Counter.fromJson(c.toJson());
      expect(back.target, 20);
      expect(back.colorIndex, 3);
    });
  });

  group('目标', () {
    test('progress 在没设目标时为 0，不会除以零', () {
      final c = Counter(id: 'a', label: 'A', value: 7);
      expect(c.progress, 0);
    });

    test('progress 封顶在 1，但 value 不封顶', () {
      final c = Counter(id: 'a', label: 'A', value: 30, target: 10);
      expect(c.progress, 1.0);
      expect(c.value, 30);
      expect(c.isComplete, isTrue);
    });

    test('copy 把目标和颜色一起带走 —— 撤销快照靠它', () {
      final c = Counter(id: 'a', label: 'A', target: 9, colorIndex: 2);
      final copy = c.copy();
      expect(copy.target, 9);
      expect(copy.colorIndex, 2);
    });
  });
}
