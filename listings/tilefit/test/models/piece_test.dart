import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/models/piece.dart';

void main() {
  group('Piece', () {
    test('ASCII 图案按行列解析成格坐标', () {
      // el5c = ['###', '#..', '#..']
      final piece = kPieceCatalog.firstWhere((p) => p.id == 'el5c');

      expect(piece.cells, <Cell>[
        const Cell(0, 0),
        const Cell(0, 1),
        const Cell(0, 2),
        const Cell(1, 0),
        const Cell(2, 0),
      ]);
      expect(piece.size, 5);
      expect(piece.width, 3);
      expect(piece.height, 3);
    });

    test('外接矩形取自最大行列，不是格数', () {
      final v5 = kPieceCatalog.firstWhere((p) => p.id == 'v5');
      expect(v5.width, 1);
      expect(v5.height, 5);

      final h5 = kPieceCatalog.firstWhere((p) => p.id == 'h5');
      expect(h5.width, 5);
      expect(h5.height, 1);
    });

    test('Cell 按值相等，可直接进 Set / 做 == 比较', () {
      expect(const Cell(2, 3), const Cell(2, 3));
      expect(const Cell(2, 3).hashCode, const Cell(2, 3).hashCode);
      expect(const Cell(2, 3), isNot(const Cell(3, 2)));
      expect(const Cell(1, 1).shifted(2, 3), const Cell(3, 4));
    });
  });

  group('kPieceCatalog', () {
    test('形状 id 不重复', () {
      final ids = kPieceCatalog.map((p) => p.id).toSet();
      expect(ids.length, kPieceCatalog.length);
    });

    test('每个形状都非空、已归一化到左上角、权重为正', () {
      for (final piece in kPieceCatalog) {
        expect(piece.cells, isNotEmpty, reason: '${piece.id} 是空形状');
        expect(
          piece.cells.any((c) => c.row == 0),
          isTrue,
          reason: '${piece.id} 没有贴到第 0 行',
        );
        expect(
          piece.cells.any((c) => c.col == 0),
          isTrue,
          reason: '${piece.id} 没有贴到第 0 列',
        );
        expect(piece.weight, greaterThan(0), reason: '${piece.id} 权重非正');
      }
    });

    test('形状内不含重复格', () {
      for (final piece in kPieceCatalog) {
        expect(
          piece.cells.toSet().length,
          piece.cells.length,
          reason: '${piece.id} 有重复格',
        );
      }
    });

    test('所有形状都放得进 8×8 棋盘', () {
      for (final piece in kPieceCatalog) {
        expect(piece.width, lessThanOrEqualTo(8), reason: '${piece.id} 太宽');
        expect(piece.height, lessThanOrEqualTo(8), reason: '${piece.id} 太高');
      }
    });

    test('配色索引都在 AppColors.pieceColors 的范围内', () {
      for (final piece in kPieceCatalog) {
        expect(piece.colorIndex, inInclusiveRange(0, 8));
      }
    });
  });
}
