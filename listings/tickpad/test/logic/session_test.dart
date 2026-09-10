import 'package:flutter_test/flutter_test.dart';
import 'package:tickpad/logic/counter.dart';
import 'package:tickpad/logic/session.dart';
import 'package:tickpad/logic/summary.dart';

void main() {
  final at = DateTime(2026, 9, 9, 14, 32);

  group('拍快照', () {
    test('存的是当时的名字和数字', () {
      final counters = <Counter>[
        Counter(id: 'a', label: 'Boxes', value: 12),
        Counter(id: 'b', label: 'Pallets', value: 3),
      ];
      final s = Session.snapshot(counters, at);
      expect(s.entries.length, 2);
      expect(s.entries.first.label, 'Boxes');
      expect(s.total, 15);
    });

    test('之后改名或删掉计数器，历史记录不跟着变', () {
      // 历史是账，不该跟着现状变 —— 所以存的是名字的副本，不是计数器 id。
      final counter = Counter(id: 'a', label: 'Boxes', value: 12);
      final s = Session.snapshot(<Counter>[counter], at);
      counter.label = 'Renamed';
      counter.value = 99;
      expect(s.entries.first.label, 'Boxes');
      expect(s.entries.first.value, 12);
    });
  });

  group('存档往返', () {
    test('正常的一条原样回来', () {
      final s = Session.snapshot(
        <Counter>[Counter(id: 'a', label: 'Boxes', value: 12)],
        at,
      );
      final back = Session.fromJson(s.toJson());
      expect(back.savedAt, at);
      expect(back.entries.first.label, 'Boxes');
      expect(back.entries.first.value, 12);
    });

    test('时间戳坏了 → 退到纪元 0，但记录本身还在', () {
      final back = Session.fromJson(<String, dynamic>{
        'at': 'not-a-number',
        'entries': <dynamic>[
          <String, dynamic>{'label': 'A', 'value': 3},
        ],
      });
      expect(back.savedAt.millisecondsSinceEpoch, 0);
      expect(back.total, 3);
    });

    test('entries 不是数组 → 当空，不抛', () {
      final back = Session.fromJson(<String, dynamic>{'at': 0, 'entries': 42});
      expect(back.entries, isEmpty);
      expect(back.total, 0);
    });

    test('数组里混进坏条目 → 只跳过那一条', () {
      final back = Session.fromJson(<String, dynamic>{
        'at': 0,
        'entries': <dynamic>[
          <String, dynamic>{'label': 'A', 'value': 3},
          'garbage',
          <String, dynamic>{'label': 'B', 'value': 4},
        ],
      });
      expect(back.entries.length, 2);
      expect(back.total, 7);
    });
  });

  group('复制成文本', () {
    test('名字左对齐、数字右对齐，末行是合计', () {
      final text = Summary.forCounters(
        <Counter>[
          Counter(id: 'a', label: 'Boxes', value: 12),
          Counter(id: 'b', label: 'Pallets', value: 3),
        ],
        at,
      );
      // 名字列宽 = 最长的 "Pallets"(7)，数值列宽 = 最长的 "15"(2)，中间两个空格
      expect(text, '''
TickPad · 2026-09-09 14:32
Boxes    12
Pallets   3
Total    15''');
    });

    test('名字比 Total 短时，列宽按 Total 算 —— 否则末行对不齐', () {
      final text = Summary.forCounters(
        <Counter>[Counter(id: 'a', label: 'A', value: 1)],
        at,
      );
      expect(text, '''
TickPad · 2026-09-09 14:32
A      1
Total  1''');
    });

    test('一个计数器都没有时也给一段能读的文本', () {
      final text = Summary.forCounters(<Counter>[], at);
      expect(text.contains('no counters'), isTrue);
    });

    test('时间格式补零', () {
      expect(
        Summary.formatDateTime(DateTime(2026, 1, 2, 3, 4)),
        '2026-01-02 03:04',
      );
    });
  });
}
