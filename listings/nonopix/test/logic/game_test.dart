import 'package:flutter_test/flutter_test.dart';
import 'package:nonopix/logic/game.dart';
import 'package:nonopix/logic/nonogram.dart';

Nonogram puzzleOf(List<String> rows) => Nonogram.fromSolution(<List<bool>>[
      for (final row in rows) <bool>[for (final ch in row.split('')) ch == '#'],
    ]);

/// 按答案把该涂的格子全涂上，用来快速走到「就差一步」的局面。
void fillAllCorrect(NonogramGame g) {
  for (var r = 0; r < g.puzzle.rows; r++) {
    for (var c = 0; c < g.puzzle.cols; c++) {
      if (g.puzzle.isFilledAt(r, c)) g.apply(r, c, CellMark.filled);
    }
  }
}

void main() {
  group('落笔', () {
    test('新局全是空标记', () {
      final g = NonogramGame(puzzleOf(<String>['##', '.#']));
      for (var r = 0; r < 2; r++) {
        for (var c = 0; c < 2; c++) {
          expect(g.markAt(r, c), CellMark.blank);
        }
      }
    });

    test('涂黑后标记生效', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.filled);
      expect(g.markAt(0, 0), CellMark.filled);
    });

    test('同一标记再点一次等于撤销', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.filled);
      g.apply(0, 0, CellMark.filled);
      expect(g.markAt(0, 0), CellMark.blank);
    });

    test('打叉与涂黑可以互相覆盖', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.crossed);
      expect(g.markAt(0, 0), CellMark.crossed);
      g.apply(0, 0, CellMark.filled);
      expect(g.markAt(0, 0), CellMark.filled);
    });
  });

  group('涂错计数（当前态，不是累计）', () {
    test('涂到该留空的格子会计入', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      expect(g.apply(0, 1, CellMark.filled), isTrue); // 返回值表示「这一笔错了」
      expect(g.wrongCount, 1);
    });

    test('涂对不计入', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      expect(g.apply(0, 0, CellMark.filled), isFalse);
      expect(g.wrongCount, 0);
    });

    test('打叉永远不计入 —— 叉只是备忘', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      expect(g.apply(0, 0, CellMark.crossed), isFalse); // 叉在该涂的格子上
      expect(g.wrongCount, 0);
    });

    test('撤销后归零 —— 棋盘空了就不该再报错', () {
      // 这是当初的 bug：累计计数在撤销后仍留着，导致棋盘明明是空的、
      // 界面却写着「1 wrong」。计数改成当前态之后才自洽。
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 1, CellMark.filled);
      expect(g.wrongCount, 1);
      g.apply(0, 1, CellMark.filled); // 再点一次取消
      expect(g.wrongCount, 0);
    });

    test('把错格改成叉也算改正', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 1, CellMark.filled);
      expect(g.wrongCount, 1);
      g.apply(0, 1, CellMark.crossed);
      expect(g.wrongCount, 0);
    });

    test('多处涂错会累加，逐个改正会逐个减少', () {
      final g = NonogramGame(puzzleOf(<String>['#..']));
      g.apply(0, 1, CellMark.filled);
      g.apply(0, 2, CellMark.filled);
      expect(g.wrongCount, 2);
      g.apply(0, 1, CellMark.filled); // 撤销其中一个
      expect(g.wrongCount, 1);
    });

    test('新局为 0', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      expect(g.wrongCount, 0);
    });
  });

  group('判胜', () {
    test('涂满该涂的就算赢，不要求打满叉', () {
      final g = NonogramGame(puzzleOf(<String>['#.', '.#']));
      expect(g.isSolved, isFalse);
      g.apply(0, 0, CellMark.filled);
      g.apply(1, 1, CellMark.filled);
      expect(g.isSolved, isTrue);
    });

    test('多涂一格就不算赢', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.filled);
      expect(g.isSolved, isTrue);
      g.apply(0, 1, CellMark.filled);
      expect(g.isSolved, isFalse);
    });

    test('少涂一格就不算赢', () {
      final g = NonogramGame(puzzleOf(<String>['##']));
      g.apply(0, 0, CellMark.filled);
      expect(g.isSolved, isFalse);
    });

    test('叉的位置不影响判胜', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.filled);
      g.apply(0, 1, CellMark.crossed);
      expect(g.isSolved, isTrue);
    });

    test('全空的题一开始就是解开的', () {
      final g = NonogramGame(puzzleOf(<String>['..']));
      expect(g.isSolved, isTrue);
    });
  });

  group('进度与线索满足', () {
    test('correctCount 只数涂对的', () {
      final g = NonogramGame(puzzleOf(<String>['#.', '##']));
      g.apply(0, 0, CellMark.filled); // 对
      g.apply(0, 1, CellMark.filled); // 错
      expect(g.correctCount, 1);
    });

    test('整行涂对后该行线索判为已满足', () {
      final g = NonogramGame(puzzleOf(<String>['#.', '##']));
      expect(g.isRowSatisfied(0), isFalse);
      g.apply(0, 0, CellMark.filled);
      expect(g.isRowSatisfied(0), isTrue);
    });

    test('整列涂对后该列线索判为已满足', () {
      final g = NonogramGame(puzzleOf(<String>['#.', '#.']));
      expect(g.isColSatisfied(0), isFalse);
      g.apply(0, 0, CellMark.filled);
      g.apply(1, 0, CellMark.filled);
      expect(g.isColSatisfied(0), isTrue);
    });

    test('多涂会让该行从已满足退回未满足', () {
      final g = NonogramGame(puzzleOf(<String>['#.']));
      g.apply(0, 0, CellMark.filled);
      expect(g.isRowSatisfied(0), isTrue);
      g.apply(0, 1, CellMark.filled);
      expect(g.isRowSatisfied(0), isFalse);
    });
  });

  group('重开', () {
    test('清空全部标记，谜题不变', () {
      final p = puzzleOf(<String>['#.', '.#']);
      final g = NonogramGame(p);
      fillAllCorrect(g);
      g.apply(0, 1, CellMark.filled); // 故意涂错一格
      expect(g.isSolved, isFalse);
      expect(g.wrongCount, 1);

      g.reset();

      expect(g.wrongCount, 0);
      expect(g.correctCount, 0);
      expect(identical(g.puzzle, p), isTrue);
      for (var r = 0; r < 2; r++) {
        for (var c = 0; c < 2; c++) {
          expect(g.markAt(r, c), CellMark.blank);
        }
      }
    });
  });
}
