import 'package:flutter_test/flutter_test.dart';
import 'package:tickpad/logic/board.dart';
import 'package:tickpad/logic/counter.dart';

/// 整屏状态机。**撤销是这里的重点** ——
/// 它是这个 App 最重要的功能，也是最容易在某个动作上漏掉的。
void main() {
  CounterBoard board() => CounterBoard(<Counter>[
        Counter(id: 'a', label: 'A'),
        Counter(id: 'b', label: 'B', step: 5),
      ]);

  group('加减', () {
    test('按各自的步长加', () {
      final b = board();
      b.bump('a', 1);
      b.bump('b', 1);
      expect(b.byId('a')!.value, 1);
      expect(b.byId('b')!.value, 5);
    });

    test('总和是所有计数器之和', () {
      final b = board();
      b.bump('a', 3);
      b.bump('b', 2);
      expect(b.total, 3 + 10);
    });

    test('已经是 0 还要减 → 返回 false，且不占撤销栈', () {
      // 这条很要紧：如果空操作也压快照，用户点几下减号之后再撤销，
      // 会连撤好几次才回到真正上一步 —— 看着像撤销坏了。
      final b = board();
      expect(b.bump('a', -1), isFalse);
      expect(b.canUndo, isFalse);
    });

    test('不存在的 id 不做任何事', () {
      final b = board();
      expect(b.bump('nope', 1), isFalse);
      expect(b.total, 0);
    });
  });

  group('撤销', () {
    test('撤销一次加法', () {
      final b = board();
      b.bump('a', 1);
      expect(b.byId('a')!.value, 1);
      expect(b.undo(), isTrue);
      expect(b.byId('a')!.value, 0);
    });

    test('撤销连续多次，一步一步退回去', () {
      final b = board();
      b.bump('a', 1);
      b.bump('a', 1);
      b.bump('a', 1);
      expect(b.byId('a')!.value, 3);
      b.undo();
      expect(b.byId('a')!.value, 2);
      b.undo();
      expect(b.byId('a')!.value, 1);
    });

    test('撤销删除 —— 卡片连同它的计数一起回来，位置也不变', () {
      // 存快照而不是存反向操作，图的就是这个：删除的逆操作得记住原来的下标，
      // 单独写很容易把它插回队尾。
      final b = board();
      b.bump('a', 7);
      b.remove('a');
      expect(b.byId('a'), isNull);
      expect(b.undo(), isTrue);
      expect(b.byId('a')!.value, 7);
      expect(b.counters.first.id, 'a', reason: '应当回到原来的位置');
    });

    test('撤销全部清零', () {
      final b = board();
      b.bump('a', 4);
      b.bump('b', 2);
      b.resetAll();
      expect(b.total, 0);
      b.undo();
      expect(b.byId('a')!.value, 4);
      expect(b.byId('b')!.value, 10);
    });

    test('撤销改名与改步长', () {
      final b = board();
      b.edit('a', label: 'Renamed', step: 10);
      expect(b.byId('a')!.label, 'Renamed');
      b.undo();
      expect(b.byId('a')!.label, 'A');
      expect(b.byId('a')!.step, 1);
    });

    test('撤销新建', () {
      final b = board();
      b.add(Counter(id: 'c', label: 'C'));
      expect(b.length, 3);
      b.undo();
      expect(b.length, 2);
    });

    test('没得撤时返回 false', () {
      final b = board();
      expect(b.canUndo, isFalse);
      expect(b.undo(), isFalse);
    });

    test('撤销栈有深度上限，最早的会被挤掉', () {
      final b = board();
      for (var i = 0; i < CounterBoard.maxUndo + 10; i++) {
        b.bump('a', 1);
      }
      var count = 0;
      while (b.undo()) {
        count++;
      }
      expect(count, CounterBoard.maxUndo);
    });

    test('撤销之后拿到的是独立副本，继续操作不会串味', () {
      final b = board();
      b.bump('a', 5);
      b.undo();
      b.bump('a', 1);
      expect(b.byId('a')!.value, 1);
    });
  });

  group('空操作不进撤销栈', () {
    test('全是 0 时 resetAll 什么都不做', () {
      final b = board();
      expect(b.resetAll(), isFalse);
      expect(b.canUndo, isFalse);
    });

    test('已经是 0 的计数器 resetOne 什么都不做', () {
      final b = board();
      expect(b.resetOne('a'), isFalse);
      expect(b.canUndo, isFalse);
    });
  });

  group('增删改', () {
    test('新建追加到末尾', () {
      final b = board();
      b.add(Counter(id: 'c', label: 'C'));
      expect(b.counters.last.id, 'c');
    });

    test('删除不存在的 id 返回 false', () {
      final b = board();
      expect(b.remove('nope'), isFalse);
    });

    test('改名会去掉首尾空格；空名字不生效', () {
      final b = board();
      b.edit('a', label: '  Boxes  ');
      expect(b.byId('a')!.label, 'Boxes');
      b.edit('a', label: '   ');
      expect(b.byId('a')!.label, 'Boxes', reason: '空名字不该把原名冲掉');
    });

    test('步长必须为正，0 和负数被忽略', () {
      final b = board();
      b.edit('a', step: 0);
      expect(b.byId('a')!.step, 1);
      b.edit('a', step: -3);
      expect(b.byId('a')!.step, 1);
    });
  });

  group('目标值', () {
    test('设了目标就有进度和完成态', () {
      final b = board();
      b.edit('a', target: 4);
      final c = b.byId('a')!;
      expect(c.hasTarget, isTrue);
      expect(c.isComplete, isFalse);
      b.bump('a', 4);
      expect(c.progress, 1.0);
      expect(c.isComplete, isTrue);
    });

    test('超过目标不封顶 —— 数超了是真事，替用户抹掉就是骗人', () {
      final b = board();
      b.edit('a', target: 3);
      b.bump('a', 5);
      expect(b.byId('a')!.value, 5);
      expect(b.byId('a')!.progress, 1.0, reason: '进度条封顶，但数值不封');
    });

    test('target 传 0 是「取消目标」，不是「不改」', () {
      // 这条必须成立，否则用户设了目标之后就再也回不到「不设目标」。
      final b = board();
      b.edit('a', target: 10);
      expect(b.byId('a')!.hasTarget, isTrue);
      b.edit('a', target: 0);
      expect(b.byId('a')!.hasTarget, isFalse);
    });

    test('没设目标时进度恒为 0，不会除以零', () {
      final b = board();
      b.bump('a', 5);
      expect(b.byId('a')!.progress, 0);
      expect(b.byId('a')!.isComplete, isFalse);
    });

    test('撤销能把目标改回去', () {
      final b = board();
      b.edit('a', target: 7);
      b.undo();
      expect(b.byId('a')!.target, 0);
    });
  });

  group('颜色标签', () {
    test('改颜色并可撤销', () {
      final b = board();
      b.edit('a', colorIndex: 3);
      expect(b.byId('a')!.colorIndex, 3);
      b.undo();
      expect(b.byId('a')!.colorIndex, 0);
    });

    test('负数下标被忽略', () {
      final b = board();
      b.edit('a', colorIndex: -1);
      expect(b.byId('a')!.colorIndex, 0);
    });
  });

  group('拖动排序', () {
    test('往后拖', () {
      final b = board();
      expect(b.move(0, 1), isTrue);
      expect(b.counters.map((c) => c.id).toList(), <String>['b', 'a']);
    });

    test('往前拖', () {
      final b = CounterBoard(<Counter>[
        Counter(id: 'a', label: 'A'),
        Counter(id: 'b', label: 'B'),
        Counter(id: 'c', label: 'C'),
      ]);
      expect(b.move(2, 0), isTrue);
      expect(b.counters.map((c) => c.id).toList(), <String>['c', 'a', 'b']);
    });

    test('排序可撤销', () {
      final b = board();
      b.move(0, 1);
      b.undo();
      expect(b.counters.map((c) => c.id).toList(), <String>['a', 'b']);
    });

    test('越界或原地不动一律不做事，也不占撤销栈', () {
      // ReorderableListView 的下标语义容易搞错，宁可什么都不做也不要错位。
      final b = board();
      expect(b.move(0, 0), isFalse);
      expect(b.move(-1, 0), isFalse);
      expect(b.move(0, 9), isFalse);
      expect(b.canUndo, isFalse);
    });

    test('拖动不影响计数值', () {
      final b = board();
      b.bump('a', 3);
      b.move(0, 1);
      expect(b.byId('a')!.value, 3);
      expect(b.total, 3);
    });
  });

  group('存档', () {
    test('toJson 出来的顺序与界面一致', () {
      final b = board();
      b.bump('a', 2);
      final json = b.toJson();
      expect(json.length, 2);
      expect(json.first['id'], 'a');
      expect(json.first['value'], 2);
    });

    test('新字段进了 JSON', () {
      final b = board();
      b.edit('a', target: 20, colorIndex: 2);
      final row = b.toJson().first;
      expect(row['target'], 20);
      expect(row['color'], 2);
    });
  });
}
