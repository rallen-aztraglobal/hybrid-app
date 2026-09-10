import 'package:checklane/logic/checklist.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Checklist sample() => Checklist(
        id: 'l1',
        title: 'Trip',
        items: <ChecklistItem>[
          ChecklistItem(id: 'a', text: 'Keys'),
          ChecklistItem(id: 'b', text: 'Wallet'),
          ChecklistItem(id: 'c', text: 'Phone'),
        ],
      );

  group('进度', () {
    test('勾几条就算几条', () {
      final l = sample();
      expect(l.doneCount, 0);
      expect(l.remaining, 3);
      l.byId('a')!.done = true;
      expect(l.doneCount, 1);
      expect(l.remaining, 2);
      expect(l.progress, closeTo(1 / 3, 1e-9));
    });

    test('全勾完是 complete', () {
      final l = sample();
      for (final i in l.items) {
        i.done = true;
      }
      expect(l.isComplete, isTrue);
      expect(l.progress, 1.0);
    });

    test('空清单：进度 0，且不算完成 —— 也不能除以零', () {
      // 「一条都没有的清单显示已完成」是这类 App 的经典尴尬。
      final l = Checklist(id: 'x', title: 'Empty');
      expect(l.progress, 0);
      expect(l.isComplete, isFalse);
    });
  });

  group('重置 —— 这个 App 的核心动作', () {
    test('只清勾选，条目一条不少', () {
      final l = sample();
      l.byId('a')!.done = true;
      l.byId('b')!.done = true;
      expect(l.reset(), isTrue);
      expect(l.doneCount, 0);
      expect(l.total, 3, reason: '条目必须留着，那才叫「再走一遍」');
      expect(l.items.map((i) => i.text).toList(),
          <String>['Keys', 'Wallet', 'Phone']);
    });

    test('本来就没勾时返回 false', () {
      expect(sample().reset(), isFalse);
    });
  });

  group('复制成模板', () {
    test('副本是未勾选的，条目文字照搬', () {
      final l = sample();
      for (final i in l.items) {
        i.done = true;
      }
      final copy = l.duplicateAs('l2', 'Trip copy', 'seed');
      expect(copy.title, 'Trip copy');
      expect(copy.doneCount, 0, reason: '复制出来就是要重新走一遍的');
      expect(copy.items.map((i) => i.text).toList(),
          <String>['Keys', 'Wallet', 'Phone']);
    });

    test('副本每条都是新 id —— 沿用旧 id 两份会互相串', () {
      final l = sample();
      final copy = l.duplicateAs('l2', 'copy', 'seed');
      final originalIds = l.items.map((i) => i.id).toSet();
      for (final i in copy.items) {
        expect(originalIds.contains(i.id), isFalse);
      }
      expect(copy.items.map((i) => i.id).toSet().length, copy.items.length);
    });

    test('改副本不影响原件', () {
      final l = sample();
      final copy = l.duplicateAs('l2', 'copy', 'seed');
      copy.items.first.text = 'Changed';
      copy.items.first.done = true;
      expect(l.items.first.text, 'Keys');
      expect(l.items.first.done, isFalse);
    });
  });

  group('深拷贝', () {
    test('copy 之后改副本不影响原件 —— 撤销快照靠它', () {
      final l = sample();
      final c = l.copy();
      c.title = 'Changed';
      c.items.first.done = true;
      expect(l.title, 'Trip');
      expect(l.items.first.done, isFalse);
    });
  });

  group('坏存档', () {
    test('缺字段 → 各自退回默认值', () {
      final l = Checklist.fromJson(<String, dynamic>{});
      expect(l.id, isNotEmpty);
      expect(l.title, 'Checklist');
      expect(l.items, isEmpty);
    });

    test('items 不是数组 → 当空，不抛', () {
      final l = Checklist.fromJson(<String, dynamic>{'id': 'x', 'items': 7});
      expect(l.items, isEmpty);
    });

    test('数组里混进坏条目 → 只跳过那一条', () {
      final l = Checklist.fromJson(<String, dynamic>{
        'id': 'x',
        'items': <dynamic>[
          <String, dynamic>{'id': 'a', 'text': 'Keys', 'done': true},
          'garbage',
          <String, dynamic>{'id': 'b', 'text': 'Wallet'},
        ],
      });
      expect(l.total, 2);
      expect(l.doneCount, 1);
    });

    test('done 不是布尔 → 当未勾', () {
      final i = ChecklistItem.fromJson(
        <String, dynamic>{'id': 'a', 'text': 'x', 'done': 'yes'},
      );
      expect(i.done, isFalse);
    });

    test('空文字 → 给个占位，不让这条消失', () {
      final i = ChecklistItem.fromJson(<String, dynamic>{'id': 'a', 'text': '  '});
      expect(i.text, 'Untitled');
    });

    test('往返', () {
      final l = sample();
      l.byId('b')!.done = true;
      final back = Checklist.fromJson(l.toJson());
      expect(back.title, 'Trip');
      expect(back.total, 3);
      expect(back.byId('b')!.done, isTrue);
    });
  });
}
