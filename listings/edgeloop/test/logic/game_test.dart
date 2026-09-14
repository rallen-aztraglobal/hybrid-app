import 'dart:math';

import 'package:edgeloop8451/logic/game.dart';
import 'package:edgeloop8451/logic/generator.dart';
import 'package:edgeloop8451/logic/puzzle.dart';
import 'package:edgeloop8451/logic/solver.dart';
import 'package:flutter_test/flutter_test.dart';

/// 把求解器给出的解铺到玩家状态上。
EdgeLoopGame _applySolution(EdgeLoopPuzzle p, List<int> solution) {
  final g = EdgeLoopGame(p);
  for (var e = 0; e < solution.length; e++) {
    if (solution[e] == 1) g.marks[e] = EdgeMark.line;
  }
  return g;
}

void main() {
  group('标记的三态循环', () {
    test('none → line → cross → none', () {
      final g = EdgeLoopGame(EdgeLoopPuzzle(
        rows: 2,
        cols: 2,
        clues: List<int?>.filled(4, null),
      ));
      expect(g.marks[0], EdgeMark.none);
      g.cycle(0);
      expect(g.marks[0], EdgeMark.line);
      g.cycle(0);
      expect(g.marks[0], EdgeMark.cross, reason: '打叉必须能标出来，否则中等以上的题没法推');
      g.cycle(0);
      expect(g.marks[0], EdgeMark.none, reason: '打叉必须能取消 —— 推错了要能退回来');
    });

    test('clear 只清标记，题面不动', () {
      final p = EdgeLoopPuzzle(rows: 2, cols: 2, clues: <int?>[2, 2, 2, 2]);
      final g = EdgeLoopGame(p)
        ..cycle(0)
        ..cycle(1);
      g.clear();
      expect(g.marks.every((m) => m == EdgeMark.none), isTrue);
      expect(g.puzzle.clues, <int?>[2, 2, 2, 2]);
    });
  });

  group('isSolved 必须与生成器的求解器同口径', () {
    // 这是「关卡保证唯一解」这句宣传语的落点。
    // 如果 isSolved 比求解器松，玩家能用别的摆法过关 —— 那就不是唯一解；
    // 如果比求解器紧，玩家摆出正解却过不了关。两种都会让卖点变成假话。
    //
    // 删掉本组测试之前，必须先删掉商店文案里「每关有且仅有一个解」那句。
    test('求解器给的解，游戏必须判为已解出', () {
      final gen = EdgeLoopGenerator(Random(31337));
      var checked = 0;
      for (var i = 0; i < 8; i++) {
        final p = gen.generate(rows: 5, cols: 5, targetCells: 10);
        if (p == null) continue;
        final r = EdgeLoopSolver(p).solve();
        expect(r.isUnique, isTrue);
        final g = _applySolution(p, r.solutions.single);
        expect(g.isSolved, isTrue, reason: '正解必须被认可');
        checked++;
      }
      expect(checked, greaterThan(0));
    });

    test('正解上翻掉任意一条边，都不再算解出', () {
      final gen = EdgeLoopGenerator(Random(555));
      final p = gen.generate(rows: 5, cols: 5, targetCells: 10);
      expect(p, isNotNull);
      final solution = EdgeLoopSolver(p!).solve().solutions.single;

      for (var e = 0; e < solution.length; e++) {
        final g = _applySolution(p, solution);
        g.marks[e] =
            solution[e] == 1 ? EdgeMark.none : EdgeMark.line; // 翻这一条
        expect(g.isSolved, isFalse,
            reason: '第 $e 条边被改动后仍判解出，说明判定太松');
      }
    });

    test('打叉不影响判定 —— 它只是备忘', () {
      final gen = EdgeLoopGenerator(Random(4242));
      final p = gen.generate(rows: 5, cols: 5, targetCells: 10);
      expect(p, isNotNull);
      final solution = EdgeLoopSolver(p!).solve().solutions.single;
      final g = _applySolution(p, solution);
      // 把所有没画线的边全部打叉
      for (var e = 0; e < g.marks.length; e++) {
        if (g.marks[e] == EdgeMark.none) g.marks[e] = EdgeMark.cross;
      }
      expect(g.isSolved, isTrue, reason: '打满叉不该影响判定');
    });

    test('空盘不算解出', () {
      final p = EdgeLoopPuzzle(rows: 2, cols: 2, clues: <int?>[0, 0, 0, 0]);
      expect(EdgeLoopGame(p).isSolved, isFalse);
    });
  });

  group('界面用的即时反馈', () {
    test('画超了才算违反，没画完不算', () {
      final p = EdgeLoopPuzzle(rows: 1, cols: 1, clues: <int?>[2]);
      final g = EdgeLoopGame(p);
      expect(g.cellViolated(0, 0), isFalse, reason: '一条没画 —— 是进行中，不是错');
      g.marks[p.cellEdges(0, 0)[0]] = EdgeMark.line;
      g.marks[p.cellEdges(0, 0)[1]] = EdgeMark.line;
      expect(g.cellViolated(0, 0), isFalse, reason: '正好画到 2');
      g.marks[p.cellEdges(0, 0)[2]] = EdgeMark.line;
      expect(g.cellViolated(0, 0), isTrue, reason: '画到 3 超了');
    });

    test('无提示的格子永远不算违反', () {
      final p = EdgeLoopPuzzle(rows: 1, cols: 1, clues: <int?>[null]);
      final g = EdgeLoopGame(p);
      for (final e in p.cellEdges(0, 0)) {
        g.marks[e] = EdgeMark.line;
      }
      expect(g.cellViolated(0, 0), isFalse);
    });
  });
}
