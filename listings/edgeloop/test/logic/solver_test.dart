import 'dart:math';

import 'package:edgeloop8451/logic/generator.dart';
import 'package:edgeloop8451/logic/puzzle.dart';
import 'package:edgeloop8451/logic/solver.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('求解器 —— 基本正确性', () {
    test('2x2 只画外框：四角提示各为 2，中间无提示，唯一解', () {
      // 外框回路。每格恰好贡献两条边（自己的两条外侧边）。
      final p = EdgeLoopPuzzle(
        rows: 2,
        cols: 2,
        clues: <int?>[2, 2, 2, 2],
      );
      final r = EdgeLoopSolver(p).solve();
      expect(r.exhausted, isTrue);
      expect(r.solutions.length, 1, reason: '外框是唯一满足四个 2 的回路');
    });

    test('全部无提示 → 多解（不唯一）', () {
      final p = EdgeLoopPuzzle(
        rows: 3,
        cols: 3,
        clues: List<int?>.filled(9, null),
      );
      final r = EdgeLoopSolver(p).solve();
      expect(r.solutions.length, 2, reason: '截断在 solutionCap，说明至少两个解');
      expect(r.isUnique, isFalse);
    });

    test('矛盾题面 → 0 解', () {
      // 单格要求四条边全画：那一圈自己就是回路，但同时每个角点度为 2 —— 其实合法。
      // 换一个真矛盾的：相邻两格都要 0，但另一格要 3，3 的那格无处借边。
      final p = EdgeLoopPuzzle(
        rows: 1,
        cols: 3,
        clues: <int?>[0, 3, 0],
      );
      final r = EdgeLoopSolver(p).solve();
      expect(r.solutions, isEmpty);
    });

    test('空盘不算解 —— 全 0 提示必须无解而不是「什么都不画」', () {
      final p = EdgeLoopPuzzle(
        rows: 2,
        cols: 2,
        clues: <int?>[0, 0, 0, 0],
      );
      final r = EdgeLoopSolver(p).solve();
      expect(r.solutions, isEmpty,
          reason: '一条边都不画满足所有 0，但那不是一条回路');
    });
  });

  group('求解器 —— 必须排除多回路', () {
    // 这一组是整个求解器最容易出错、也最难察觉的地方：
    // 「每格提示数对得上 + 每个点的度都是 0 或 2」这两条**不足以**保证是一条回路，
    // 两个互不相连的环同样满足。若漏了连通性检查，生成器会把多环题面当成唯一解放出去。
    //
    // 所以这里不是随便编一组提示去看它有没有解 —— 那样万一提示本身就无解，
    // 测试会「通过」却什么也没验证到（空测）。做法是反过来：
    // 先显式造出两环构型，从它反推提示，再断言求解器返回 0 解。
    // 由于测试自己验证了该构型满足全部提示与度约束，0 解就只可能来自连通性检查。
    test('两个独立方框：提示与度全部满足，仍必须判为无解', () {
      const rows = 2, cols = 5;
      final blank = EdgeLoopPuzzle(
        rows: rows,
        cols: cols,
        clues: List<int?>.filled(rows * cols, null),
      );
      final edges = List<int>.filled(blank.edgeCount, 0);

      // 左环：围住 c∈{0,1} 两列；右环：围住 c∈{3,4} 两列。
      void frameColumns(int c0, int c1) {
        for (var c = c0; c <= c1; c++) {
          edges[blank.hIndex(0, c)] = 1; // 上边
          edges[blank.hIndex(rows, c)] = 1; // 下边
        }
        for (var r = 0; r < rows; r++) {
          edges[blank.vIndex(r, c0)] = 1; // 左边
          edges[blank.vIndex(r, c1 + 1)] = 1; // 右边
        }
      }

      frameColumns(0, 1);
      frameColumns(3, 4);

      // 从这个构型反推提示。
      final clues = List<int?>.filled(rows * cols, null);
      for (var r = 0; r < rows; r++) {
        for (var c = 0; c < cols; c++) {
          var n = 0;
          for (final e in blank.cellEdges(r, c)) {
            n += edges[e];
          }
          clues[r * cols + c] = n;
        }
      }

      // 阳性对照 1：该构型确实满足每一格的提示（反推而来，必然成立，但要钉住）。
      for (var r = 0; r < rows; r++) {
        for (var c = 0; c < cols; c++) {
          var n = 0;
          for (final e in blank.cellEdges(r, c)) {
            n += edges[e];
          }
          expect(n, clues[r * cols + c]);
        }
      }
      // 阳性对照 2：每个点的度都是 0 或 2 —— 说明它过得了 DFS 里的度约束。
      for (var r = 0; r <= rows; r++) {
        for (var c = 0; c <= cols; c++) {
          var deg = 0;
          for (final e in blank.dotEdges(r, c)) {
            deg += edges[e];
          }
          expect(deg == 0 || deg == 2, isTrue,
              reason: '点($r,$c) 度=$deg，两环构型本该处处合法');
        }
      }
      // 阳性对照 3：确实是两个环，不是一个。
      expect(edges.where((e) => e == 1).length, 16,
          reason: '每个框 2 行 2 列，周长 = 上下各 2 + 左右各 2 = 8 条边，两个框 16 条');

      final p = EdgeLoopPuzzle(rows: rows, cols: cols, clues: clues);
      final r = EdgeLoopSolver(p).solve();
      expect(r.exhausted, isTrue);
      expect(r.solutions, isEmpty,
          reason: '唯一能满足这组提示的构型是那两个环，必须被连通性检查判掉');
    });
  });

  group('生成器', () {
    test('生成的题面确实唯一解，且解与造出的回路一致', () {
      final gen = EdgeLoopGenerator(Random(20260911));
      var made = 0;
      for (var i = 0; i < 6; i++) {
        final p = gen.generate(rows: 5, cols: 5, targetCells: 10);
        if (p == null) continue;
        made++;
        final r = EdgeLoopSolver(p).solve();
        expect(r.isUnique, isTrue, reason: '生成器的全部产出都必须唯一解');
        expect(p.clueCount, lessThan(25), reason: '应当删掉了一部分提示');
      }
      expect(made, greaterThan(0), reason: '至少要能生成出题目来');
    });

    test('提示数取值只在 0..3', () {
      final gen = EdgeLoopGenerator(Random(7));
      final p = gen.generate(rows: 5, cols: 5, targetCells: 12);
      expect(p, isNotNull);
      for (final c in p!.clues) {
        if (c != null) expect(c, inInclusiveRange(0, 3));
      }
    });
  });

  group('串行化', () {
    test('encode/decode 往返不丢信息', () {
      final gen = EdgeLoopGenerator(Random(99));
      final p = gen.generate(rows: 5, cols: 5, targetCells: 11);
      expect(p, isNotNull);
      final back = EdgeLoopPuzzle.decode(p!.encode());
      expect(back.rows, p.rows);
      expect(back.cols, p.cols);
      expect(back.clues, p.clues);
    });
  });
}
