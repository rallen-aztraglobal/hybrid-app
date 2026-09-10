import 'dart:math';

import 'package:flutter_test/flutter_test.dart';
import 'package:nonopix/logic/generator.dart';
import 'package:nonopix/logic/nonogram.dart';

List<List<bool>> grid(List<String> rows) => <List<bool>>[
      for (final row in rows) <bool>[for (final ch in row.split('')) ch == '#'],
    ];

void main() {
  group('Nonogram.fromSolution', () {
    test('行列线索都由答案导出', () {
      final p = Nonogram.fromSolution(grid(<String>[
        '##.',
        '.#.',
        '.##',
      ]));
      expect(p.rows, 3);
      expect(p.cols, 3);
      expect(p.rowClues, <List<int>>[
        <int>[2],
        <int>[1],
        <int>[2],
      ]);
      expect(p.colClues, <List<int>>[
        <int>[1],
        <int>[3],
        <int>[1],
      ]);
    });

    test('空行空列的线索是 [0]', () {
      final p = Nonogram.fromSolution(grid(<String>[
        '#.',
        '..',
      ]));
      expect(p.rowClues[1], <int>[0]);
      expect(p.colClues[1], <int>[0]);
    });

    test('filledCount 数的是答案里要涂的格子', () {
      final p = Nonogram.fromSolution(grid(<String>[
        '##',
        '#.',
      ]));
      expect(p.filledCount, 3);
    });

    test('拒绝非矩形的答案表', () {
      expect(
        () => Nonogram.fromSolution(<List<bool>>[
          <bool>[true, false],
          <bool>[true],
        ]),
        throwsArgumentError,
      );
    });

    test('线索表是不可变的', () {
      final p = Nonogram.fromSolution(grid(<String>['#.']));
      expect(() => p.rowClues[0].add(9), throwsUnsupportedError);
    });
  });

  group('isLineSolvable', () {
    test('线索唯一确定答案时判为可解', () {
      // 每行每列都被线索唯一确定
      final p = Nonogram.fromSolution(grid(<String>[
        '###',
        '#.#',
        '###',
      ]));
      expect(p.isLineSolvable, isTrue);
    });

    test('需要猜的题判为不可解', () {
      // 经典的二义图案：对角线互换后线索完全相同，逻辑推不出来
      final p = Nonogram.fromSolution(grid(<String>[
        '#.',
        '.#',
      ]));
      expect(p.isLineSolvable, isFalse);
    });

    test('全空的题平凡可解', () {
      final p = Nonogram.fromSolution(grid(<String>[
        '..',
        '..',
      ]));
      expect(p.isLineSolvable, isTrue);
    });
  });

  group('dealPuzzle', () {
    test('两种尺寸都能发牌，且尺寸正确', () {
      for (final size in PuzzleSize.values) {
        final p = dealPuzzle(size, random: Random(7));
        expect(p.rows, size.side);
        expect(p.cols, size.side);
      }
    });

    test('发出来的题一定是纯逻辑可解的 —— 各尺寸各抽 30 次', () {
      for (final size in PuzzleSize.values) {
        for (var seed = 0; seed < 30; seed++) {
          final p = dealPuzzle(size, random: Random(seed));
          expect(
            p.isLineSolvable,
            isTrue,
            reason: '${size.label} seed=$seed 发到了需要猜的题',
          );
        }
      }
    });

    test('不会发到全空或全满的题', () {
      for (var seed = 0; seed < 30; seed++) {
        final p = dealPuzzle(PuzzleSize.small, random: Random(seed));
        expect(p.filledCount, greaterThan(0));
        expect(p.filledCount, lessThan(p.rows * p.cols));
      }
    });

    test('同一种子发到同一道题 —— 可复现', () {
      final a = dealPuzzle(PuzzleSize.large, random: Random(42));
      final b = dealPuzzle(PuzzleSize.large, random: Random(42));
      expect(a.solution, b.solution);
    });

    test('avoid 能防止连着发到同一张', () {
      final first = dealPicture(PuzzleSize.large, random: Random(3));
      for (var seed = 0; seed < 20; seed++) {
        final next =
            dealPicture(PuzzleSize.large, random: Random(seed), avoid: first);
        expect(identical(next, first), isFalse,
            reason: 'seed=$seed 时又发到了同一张 ${first.name}');
      }
    });

    test('线索与答案自洽 —— 用线索反解能还原出答案', () {
      final p = dealPuzzle(PuzzleSize.small, random: Random(11));
      // isLineSolvable 走完推演后必须落到唯一答案上，这里再独立验一次线索导出
      for (var r = 0; r < p.rows; r++) {
        var run = 0;
        final derived = <int>[];
        for (var c = 0; c < p.cols; c++) {
          if (p.solution[r][c]) {
            run++;
          } else if (run > 0) {
            derived.add(run);
            run = 0;
          }
        }
        if (run > 0) derived.add(run);
        expect(p.rowClues[r], derived.isEmpty ? <int>[0] : derived);
      }
    });
  });
}
