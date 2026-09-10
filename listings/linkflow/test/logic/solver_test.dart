import 'package:flutter_test/flutter_test.dart';
import 'package:linkflow/logic/board.dart';
import 'package:linkflow/logic/puzzle.dart';
import 'package:linkflow/logic/solver.dart';

/// 求解器是题库的把关人 —— 它说「唯一解」，出厂的题才算数。
/// 所以它自己得先被验一遍：多解题不能被判成唯一，唯一解题不能被判成多解。
void main() {
  group('唯一解', () {
    test('三条横条：每条都被上下的端点夹死，只有一种走法', () {
      final puzzle = FlowPuzzle(3, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RR'),
        const FlowPath('b', start: Cell(1, 0), moves: 'RR'),
        const FlowPath('c', start: Cell(2, 0), moves: 'RR'),
      ]);
      final r = FlowSolver(puzzle).run();
      expect(r.exhausted, isTrue);
      expect(r.solutions, 1);
      expect(r.isUnique, isTrue);
      expect(r.nodes, greaterThan(0));
    });

    test('2×2 两条横条：想绕路就得穿过对方的端点，绕不了', () {
      final puzzle = FlowPuzzle(2, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'R'),
        const FlowPath('b', start: Cell(1, 0), moves: 'R'),
      ]);
      expect(FlowSolver(puzzle).run().isUnique, isTrue);
    });
  });

  group('多解', () {
    test('单条通路走遍 3×3：从一角到对角有不止一种走法', () {
      // 给的答案是其中一种，但棋盘上还存在别的哈密顿路径 —— 应当被判成多解。
      final puzzle = FlowPuzzle(3, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RRDLLDRR'),
      ]);
      final r = FlowSolver(puzzle).run();
      expect(r.solutions, 2, reason: '数到 solutionCap 就停，够判「不唯一」了');
      expect(r.isUnique, isFalse);
    });

    test('两个端点挨着的长通路：它可以缩成两格，剩下的格子随便别人分', () {
      // 正是生成器要筛掉的那一类：答案本身合法，但玩家有别的连法也能过关。
      final puzzle = FlowPuzzle(4, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RRR'),
        const FlowPath('b', start: Cell(1, 0), moves: 'RRRDLLL'), // 首尾 (1,0)/(2,0) 相邻
        const FlowPath('c', start: Cell(3, 0), moves: 'RRR'),
      ]);
      expect(puzzle.validate(), isEmpty, reason: '答案本身是合法的');
      expect(FlowSolver(puzzle).run().solutions, greaterThan(1));
    });
  });

  group('坏题', () {
    test('两条通路的端点撞在同一格 —— 判 0 解，不崩', () {
      final puzzle = FlowPuzzle(3, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RR'),
        const FlowPath('b', start: Cell(0, 2), moves: 'DD'), // 起点正是 a 的终点
        const FlowPath('c', start: Cell(1, 0), moves: 'R'),
      ]);
      final r = FlowSolver(puzzle).run();
      expect(r.solutions, 0);
      expect(r.exhausted, isTrue);
    });

    test('只有一格的通路（首尾同格）—— 判 0 解，不崩', () {
      final puzzle = FlowPuzzle(2, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: ''),
        const FlowPath('b', start: Cell(1, 0), moves: 'R'),
      ]);
      expect(FlowSolver(puzzle).run().solutions, 0);
    });
  });

  group('节点上限', () {
    test('撞上上限就把 exhausted 置 false，不谎报唯一', () {
      final puzzle = FlowPuzzle(3, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RRDLLDRR'),
      ]);
      final r = FlowSolver(puzzle, nodeCap: 1).run();
      expect(r.exhausted, isFalse);
      expect(r.isUnique, isFalse, reason: '没搜完就不能说唯一');
    });
  });
}
