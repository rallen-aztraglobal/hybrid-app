import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/logic/board.dart';
import 'package:tilefit/models/piece.dart';

Piece _piece(String id) => kPieceCatalog.firstWhere((p) => p.id == id);

/// 逐格填入而不触发消除：直接铺 dot，且调用方保证不会填满任何整行整列。
void _fill(Board board, Iterable<Cell> cells) {
  final dot = _piece('dot');
  for (final cell in cells) {
    board.place(dot, cell.row, cell.col);
  }
}

void main() {
  late Board board;

  setUp(() {
    board = Board();
  });

  group('canPlace', () {
    test('空棋盘上放得下', () {
      expect(board.canPlace(_piece('sq2'), 0, 0), isTrue);
      expect(board.canPlace(_piece('sq2'), 6, 6), isTrue);
    });

    test('越界放不下', () {
      final sq2 = _piece('sq2');
      expect(board.canPlace(sq2, 7, 7), isFalse, reason: '右下角只剩 1×1');
      expect(board.canPlace(sq2, -1, 0), isFalse);
      expect(board.canPlace(sq2, 0, -1), isFalse);
      expect(
        board.canPlace(_piece('h5'), 0, 4),
        isFalse,
        reason: '5 连从第 4 列起会出界',
      );
      expect(board.canPlace(_piece('h5'), 0, 3), isTrue);
    });

    test('压到已填格放不下', () {
      board.place(_piece('dot'), 4, 4);
      expect(board.canPlace(_piece('dot'), 4, 4), isFalse);
      expect(
        board.canPlace(_piece('sq2'), 3, 3),
        isFalse,
        reason: '右下角压到 (4,4)',
      );
      expect(board.canPlace(_piece('sq2'), 3, 5), isTrue);
    });
  });

  group('place', () {
    test('落子把格填成方块的配色索引', () {
      final sq2 = _piece('sq2');
      final result = board.place(sq2, 2, 3);

      expect(result.cellsPlaced, 4);
      expect(result.linesCleared, 0);
      expect(board.filledCount, 4);
      for (final cell in sq2.cells) {
        expect(board.cellAt(2 + cell.row, 3 + cell.col), sq2.colorIndex);
      }
      expect(board.isFilled(2, 5), isFalse, reason: '方块外的格不该被动到');
    });

    test('非法落子抛异常而不是静默忽略', () {
      board.place(_piece('dot'), 0, 0);
      expect(() => board.place(_piece('dot'), 0, 0), throwsStateError);
      expect(() => board.place(_piece('dot'), 8, 0), throwsStateError);
    });

    test('填满一行即消除该行', () {
      // 先铺满第 3 行的前 7 格，此时还不该消。
      _fill(board, <Cell>[for (var c = 0; c < 7; c++) Cell(3, c)]);
      expect(board.filledCount, 7);

      final result = board.place(_piece('dot'), 3, 7);
      expect(result.clearedRows, <int>[3]);
      expect(result.clearedCols, isEmpty);
      expect(result.linesCleared, 1);
      expect(board.filledCount, 0);
    });

    test('填满一列即消除该列', () {
      _fill(board, <Cell>[for (var r = 0; r < 7; r++) Cell(r, 2)]);

      final result = board.place(_piece('dot'), 7, 2);
      expect(result.clearedCols, <int>[2]);
      expect(result.clearedRows, isEmpty);
      expect(board.filledCount, 0);
    });

    test('同时填满的行与列一起消除（相交格不能让其中一条漏判）', () {
      // 第 3 行差 (3,7)，第 7 列差 (3,7)：补上这一格后两条线同时满。
      // 若实现是「边找边清」，先清掉第 3 行会在第 7 列打出缺口，导致列漏判。
      _fill(board, <Cell>[
        for (var c = 0; c < 7; c++) Cell(3, c),
        for (var r = 0; r < 8; r++)
          if (r != 3) Cell(r, 7),
      ]);
      expect(board.filledCount, 14);

      final result = board.place(_piece('dot'), 3, 7);

      expect(result.clearedRows, <int>[3]);
      expect(result.clearedCols, <int>[7]);
      expect(result.linesCleared, 2);
      expect(board.filledCount, 0, reason: '两条线的并集就是全部已填格');
    });

    test('多行同时满则一起消除', () {
      _fill(board, <Cell>[
        for (var c = 0; c < 7; c++) ...<Cell>[Cell(0, c), Cell(1, c)],
      ]);
      // v2 竖着补上第 7 列的第 0、1 行，两行同时满。
      final result = board.place(_piece('v2'), 0, 7);

      expect(result.clearedRows, <int>[0, 1]);
      expect(result.linesCleared, 2);
      expect(board.filledCount, 0);
    });
  });

  group('hasAnyPlacement', () {
    test('空棋盘上任何形状都有落点', () {
      for (final piece in kPieceCatalog) {
        expect(board.hasAnyPlacement(piece), isTrue, reason: piece.id);
      }
    });

    test('棋盘塞到只剩零散单格时，大形状再无落点', () {
      // 填成国际象棋盘的黑格：任意两个相邻格不同时被填，故任何 2 连都放不下，单格还能放。
      // 每行每列都恰好填 4 格、永远凑不满 8 格，填的过程中不会触发消除。
      for (var r = 0; r < 8; r++) {
        for (var c = 0; c < 8; c++) {
          if ((r + c).isEven) board.place(_piece('dot'), r, c);
        }
      }
      expect(board.hasAnyPlacement(_piece('dot')), isTrue);
      expect(board.hasAnyPlacement(_piece('h2')), isFalse);
      expect(board.hasAnyPlacement(_piece('v2')), isFalse);
      expect(board.hasAnyPlacement(_piece('sq2')), isFalse);
    });
  });

  test('越界读格返回空而不是抛异常', () {
    expect(board.cellAt(-1, 0), Board.emptyCell);
    expect(board.cellAt(0, 8), Board.emptyCell);
    expect(board.isFilled(99, 99), isFalse);
  });

  test('clear 把棋盘还原成全空', () {
    board.place(_piece('sq3'), 0, 0);
    expect(board.filledCount, 9);
    board.clear();
    expect(board.filledCount, 0);
  });
}
