import 'package:flutter_test/flutter_test.dart';
import 'package:linkflow/logic/board.dart';
import 'package:linkflow/logic/game.dart';
import 'package:linkflow/logic/puzzle.dart';

/// 3×3 测试题：三条横条。端点在每行的两端。
///   aaa
///   bbb
///   ccc
FlowPuzzle threeRows() => FlowPuzzle(3, <FlowPath>[
      const FlowPath('a', start: Cell(0, 0), moves: 'RR'),
      const FlowPath('b', start: Cell(1, 0), moves: 'RR'),
      const FlowPath('c', start: Cell(2, 0), moves: 'RR'),
    ]);

/// 按答案把某条通路一次画通。
void drawFull(FlowGame g, String key) {
  final cells = g.puzzle.paths[key]!;
  g.beginDrag(cells.first);
  for (var i = 1; i < cells.length; i++) {
    g.dragTo(cells[i]);
  }
  g.endDrag();
}

void main() {
  group('起手', () {
    test('按在端点上开始画线', () {
      final g = FlowGame(threeRows());
      expect(g.beginDrag(const Cell(0, 0)), isTrue);
      expect(g.draggingKey, 'a');
      expect(g.pathOf('a'), <Cell>[const Cell(0, 0)]);
    });

    test('按在空白的中段格上不起手', () {
      final g = FlowGame(threeRows());
      expect(g.beginDrag(const Cell(0, 1)), isFalse);
      expect(g.draggingKey, isNull);
    });

    test('按在棋盘外不起手', () {
      final g = FlowGame(threeRows());
      expect(g.beginDrag(const Cell(-1, 0)), isFalse);
      expect(g.beginDrag(const Cell(0, 9)), isFalse);
    });

    test('按在自己已画的中段上从该处截断继续画', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'a'); // (0,0)->(0,1)->(0,2)
      expect(g.pathOf('a').length, 3);
      g.beginDrag(const Cell(0, 1));
      expect(g.pathOf('a'), <Cell>[const Cell(0, 0), const Cell(0, 1)]);
    });
  });

  group('画线', () {
    test('只能走到相邻格，不能跳格', () {
      final g = FlowGame(threeRows());
      g.beginDrag(const Cell(0, 0));
      expect(g.dragTo(const Cell(0, 2)), isFalse); // 隔了一格
      expect(g.pathOf('a').length, 1);
      expect(g.dragTo(const Cell(0, 1)), isTrue);
      expect(g.pathOf('a').length, 2);
    });

    test('往回拖等于撤销上一步', () {
      final g = FlowGame(threeRows());
      g.beginDrag(const Cell(0, 0));
      g.dragTo(const Cell(0, 1));
      expect(g.pathOf('a').length, 2);
      g.dragTo(const Cell(0, 0)); // 退回起点
      expect(g.pathOf('a'), <Cell>[const Cell(0, 0)]);
    });

    test('没有起手时拖动无效', () {
      final g = FlowGame(threeRows());
      expect(g.dragTo(const Cell(0, 1)), isFalse);
    });

    test('连通之后不再继续延伸', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'a');
      expect(g.isConnected('a'), isTrue);
      g.beginDrag(const Cell(0, 2)); // 从另一端起手会重置成单格
      expect(g.pathOf('a').length, 3); // 端点已被占，走的是「截断」分支
    });
  });

  group('与其他通路的关系', () {
    test('画到别人线的中段会把对方从那一格起截断', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'b'); // b = (1,0),(1,1),(1,2)
      expect(g.pathOf('b').length, 3);

      // a 从 (0,0) 起手，走到 (0,1) 再往下踩 b 的中段 (1,1)。
      // (1,1) 不是端点，所以允许穿过，b 被截成只剩 (1,0)。
      g.beginDrag(const Cell(0, 0));
      g.dragTo(const Cell(0, 1));
      expect(g.dragTo(const Cell(1, 1)), isTrue);
      expect(g.pathOf('b'), <Cell>[const Cell(1, 0)]);
      expect(g.pathOf('a').length, 3);
    });

    test('起手时踩在别人的线上，接管的是那条线', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'b');
      g.beginDrag(const Cell(1, 1)); // b 的中段
      expect(g.draggingKey, 'b');
      expect(g.pathOf('b').length, 2, reason: '从被踩那一格起截断');
    });

    test('不能穿过别人的端点，且失败时不会误伤对方的线', () {
      // 这条曾经是个 bug：先截断对方、再判断能不能走，
      // 结果走没走成、对方的线却被截了。
      final puzzle = FlowPuzzle(3, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'RR'),
        const FlowPath('b', start: Cell(1, 0), moves: 'RR'),
        const FlowPath('c', start: Cell(2, 0), moves: 'RR'),
      ]);
      final g = FlowGame(puzzle);
      drawFull(g, 'b');
      expect(g.pathOf('b').length, 3);

      // a 从 (0,0) 起手，试着往下走到 (1,0) —— 那是 b 的端点
      g.beginDrag(const Cell(0, 0));
      expect(g.dragTo(const Cell(1, 0)), isFalse);
      expect(g.pathOf('b').length, 3, reason: 'b 的线不该被动过');
    });
  });

  group('判胜', () {
    test('全部连通且铺满才算通关', () {
      final g = FlowGame(threeRows());
      expect(g.isSolved, isFalse);
      drawFull(g, 'a');
      drawFull(g, 'b');
      expect(g.isSolved, isFalse, reason: '还差一条');
      drawFull(g, 'c');
      expect(g.isSolved, isTrue);
    });

    test('连通了但没铺满不算通关', () {
      // a 直接从 (0,0) 走到 (0,2) 是唯一走法，所以这题用另一道：
      // b 有两种走法，其中一种绕路、另一种直达
      final puzzle = FlowPuzzle(2, <FlowPath>[
        const FlowPath('a', start: Cell(0, 0), moves: 'R'),
        const FlowPath('b', start: Cell(1, 0), moves: 'R'),
      ]);
      final g = FlowGame(puzzle);
      drawFull(g, 'a');
      expect(g.connectedCount, 1);
      expect(g.isSolved, isFalse, reason: '只连通了一条、棋盘也没满');
      expect(g.filledCount, 2);
    });

    test('connectedCount 统计已连通的条数', () {
      final g = FlowGame(threeRows());
      expect(g.connectedCount, 0);
      drawFull(g, 'a');
      expect(g.connectedCount, 1);
      drawFull(g, 'c');
      expect(g.connectedCount, 2);
    });
  });

  group('清除', () {
    test('clearPath 只清掉指定的一条', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'a');
      drawFull(g, 'b');
      g.clearPath('a');
      expect(g.pathOf('a'), isEmpty);
      expect(g.pathOf('b').length, 3);
    });

    test('reset 清空全部并结束拖动', () {
      final g = FlowGame(threeRows());
      drawFull(g, 'a');
      g.beginDrag(const Cell(1, 0));
      g.reset();
      expect(g.filledCount, 0);
      expect(g.draggingKey, isNull);
      expect(g.connectedCount, 0);
    });
  });
}
