import 'package:checklane/logic/book.dart';
import 'package:checklane/logic/checklist.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  ChecklistBook book() => ChecklistBook(<Checklist>[
        Checklist(
          id: 'l1',
          title: 'Trip',
          items: <ChecklistItem>[
            ChecklistItem(id: 'a', text: 'Keys'),
            ChecklistItem(id: 'b', text: 'Wallet'),
          ],
        ),
        Checklist(id: 'l2', title: 'Shop'),
      ]);

  group('勾选', () {
    test('来回切换', () {
      final b = book();
      expect(b.toggle('l1', 'a'), isTrue);
      expect(b.byId('l1')!.byId('a')!.done, isTrue);
      b.toggle('l1', 'a');
      expect(b.byId('l1')!.byId('a')!.done, isFalse);
    });

    test('不存在的清单或条目不做任何事', () {
      final b = book();
      expect(b.toggle('nope', 'a'), isFalse);
      expect(b.toggle('l1', 'nope'), isFalse);
      expect(b.canUndo, isFalse);
    });
  });

  group('条目增删改', () {
    test('新增追加到末尾', () {
      final b = book();
      b.addItem('l1', 'Phone', 'c');
      expect(b.byId('l1')!.items.last.text, 'Phone');
    });

    test('空文字不新增', () {
      final b = book();
      expect(b.addItem('l1', '   ', 'c'), isFalse);
      expect(b.byId('l1')!.total, 2);
      expect(b.canUndo, isFalse);
    });

    test('新增会去掉首尾空格', () {
      final b = book();
      b.addItem('l1', '  Phone  ', 'c');
      expect(b.byId('l1')!.byId('c')!.text, 'Phone');
    });

    test('改文字；空文字不生效', () {
      final b = book();
      b.editItem('l1', 'a', 'House keys');
      expect(b.byId('l1')!.byId('a')!.text, 'House keys');
      b.editItem('l1', 'a', '  ');
      expect(b.byId('l1')!.byId('a')!.text, 'House keys');
    });

    test('删除', () {
      final b = book();
      expect(b.removeItem('l1', 'a'), isTrue);
      expect(b.byId('l1')!.byId('a'), isNull);
      expect(b.removeItem('l1', 'a'), isFalse);
    });
  });

  group('重置', () {
    test('清掉勾选，条目留着', () {
      final b = book();
      b.toggle('l1', 'a');
      expect(b.resetList('l1'), isTrue);
      expect(b.byId('l1')!.doneCount, 0);
      expect(b.byId('l1')!.total, 2);
    });

    test('本来没勾 → 什么都不做，也不占撤销栈', () {
      // 空操作压快照的话，用户之后再撤销会连撤好几次才回到真正上一步。
      final b = book();
      expect(b.resetList('l1'), isFalse);
      expect(b.canUndo, isFalse);
    });
  });

  group('清单增删改', () {
    test('新建、改名、删除', () {
      final b = book();
      b.addList(Checklist(id: 'l3', title: 'New'));
      expect(b.length, 3);
      b.renameList('l3', 'Renamed');
      expect(b.byId('l3')!.title, 'Renamed');
      b.removeList('l3');
      expect(b.byId('l3'), isNull);
    });

    test('空名字不生效', () {
      final b = book();
      expect(b.renameList('l1', '   '), isFalse);
      expect(b.byId('l1')!.title, 'Trip');
    });

    test('复制插在原件后面，且是未勾选的', () {
      final b = book();
      b.toggle('l1', 'a');
      b.duplicateList('l1', 'l1c', 'seed');
      expect(b.length, 3);
      expect(b.lists[1].id, 'l1c', reason: '副本要紧挨着原件，方便找');
      expect(b.lists[1].title, 'Trip copy');
      expect(b.lists[1].doneCount, 0);
      expect(b.lists[0].doneCount, 1, reason: '原件的勾不该被动');
    });
  });

  group('排序', () {
    test('清单排序', () {
      final b = book();
      expect(b.moveList(0, 1), isTrue);
      expect(b.lists.map((l) => l.id).toList(), <String>['l2', 'l1']);
    });

    test('条目排序', () {
      final b = book();
      expect(b.moveItem('l1', 0, 1), isTrue);
      expect(b.byId('l1')!.items.map((i) => i.id).toList(), <String>['b', 'a']);
    });

    test('越界或原地不动一律不做事，也不占撤销栈', () {
      final b = book();
      expect(b.moveList(0, 0), isFalse);
      expect(b.moveList(-1, 0), isFalse);
      expect(b.moveItem('l1', 0, 9), isFalse);
      expect(b.moveItem('nope', 0, 1), isFalse);
      expect(b.canUndo, isFalse);
    });
  });

  group('撤销', () {
    test('撤销勾选', () {
      final b = book();
      b.toggle('l1', 'a');
      b.undo();
      expect(b.byId('l1')!.byId('a')!.done, isFalse);
    });

    test('撤销重置 —— 勾回来', () {
      // 这是最要紧的一条：核对表走到一半误按了重置，没有撤销就得从头再勾一遍。
      final b = book();
      b.toggle('l1', 'a');
      b.toggle('l1', 'b');
      b.resetList('l1');
      expect(b.byId('l1')!.doneCount, 0);
      b.undo();
      expect(b.byId('l1')!.doneCount, 2);
    });

    test('撤销删除清单 —— 连条目带位置一起回来', () {
      final b = book();
      b.toggle('l1', 'a');
      b.removeList('l1');
      expect(b.byId('l1'), isNull);
      b.undo();
      expect(b.byId('l1')!.total, 2);
      expect(b.byId('l1')!.doneCount, 1);
      expect(b.lists.first.id, 'l1', reason: '应当回到原来的位置');
    });

    test('撤销删除条目', () {
      final b = book();
      b.removeItem('l1', 'a');
      b.undo();
      expect(b.byId('l1')!.byId('a'), isNotNull);
      expect(b.byId('l1')!.items.first.id, 'a');
    });

    test('撤销复制', () {
      final b = book();
      b.duplicateList('l1', 'l1c', 'seed');
      b.undo();
      expect(b.length, 2);
    });

    test('撤销栈有深度上限', () {
      final b = book();
      for (var i = 0; i < ChecklistBook.maxUndo + 5; i++) {
        b.toggle('l1', 'a');
      }
      var count = 0;
      while (b.undo()) {
        count++;
      }
      expect(count, ChecklistBook.maxUndo);
    });

    test('撤销之后拿到的是独立副本，继续操作不会串味', () {
      final b = book();
      b.toggle('l1', 'a');
      b.undo();
      b.toggle('l1', 'b');
      expect(b.byId('l1')!.byId('a')!.done, isFalse);
      expect(b.byId('l1')!.byId('b')!.done, isTrue);
    });
  });

  group('存档', () {
    test('toJson 的顺序与界面一致', () {
      final b = book();
      final json = b.toJson();
      expect(json.length, 2);
      expect(json.first['id'], 'l1');
      expect((json.first['items'] as List<dynamic>).length, 2);
    });
  });
}
