import 'package:edgeloop8451/logic/hit_test.dart';
import 'package:edgeloop8451/logic/puzzle.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final puzzle = EdgeLoopPuzzle(
    rows: 5,
    cols: 5,
    clues: List<int?>.filled(25, null),
  );
  final geo = BoardGeometry.fit(400, 400, 5, 5);

  group('棋盘几何', () {
    test('格子是正方的，棋盘居中', () {
      final g = BoardGeometry.fit(600, 400, 5, 5);
      // 高度更紧，格子边长由高度决定
      expect(g.cell, closeTo(400 / (5 + BoardGeometry.pad * 2), 0.001));
      // 居中：左右留白相等
      final rightGap = 600 - (g.originX + 5 * g.cell);
      expect(g.originX, closeTo(rightGap, 0.001));
    });

    test('非正方棋盘也按短边定格子大小', () {
      final g = BoardGeometry.fit(400, 400, 8, 5);
      expect(g.cell, closeTo(400 / (8 + BoardGeometry.pad * 2), 0.001), reason: '行多，由高度决定');
    });
  });

  group('命中判定 —— 棋盘内没有死区', () {
    // 这是这个游戏在手机上最要紧的一条：边只有两三像素宽，
    // 判定一旦留下够不着的区域，玩家点下去没反应，会以为 App 坏了。
    //
    // 用密集采样穷举整个棋盘区域，每一点都必须命中某条边。
    test('棋盘范围内每一点都能命中一条边', () {
      var misses = 0;
      const steps = 120;
      final w = 5 * geo.cell;
      final h = 5 * geo.cell;
      for (var i = 0; i <= steps; i++) {
        for (var j = 0; j <= steps; j++) {
          final x = geo.originX + w * i / steps;
          final y = geo.originY + h * j / steps;
          if (edgeAt(puzzle, geo, x, y) == null) misses++;
        }
      }
      expect(misses, 0, reason: '棋盘内有 $misses 个采样点点不中任何边');
    });

    test('命中的边下标始终在合法范围内', () {
      const steps = 80;
      final w = 5 * geo.cell;
      final h = 5 * geo.cell;
      for (var i = 0; i <= steps; i++) {
        for (var j = 0; j <= steps; j++) {
          final e = edgeAt(
            puzzle,
            geo,
            geo.originX + w * i / steps,
            geo.originY + h * j / steps,
          );
          if (e == null) continue;
          expect(e, inInclusiveRange(0, puzzle.edgeCount - 1));
        }
      }
    });
  });

  group('命中判定 —— 点在该点的边上', () {
    test('正落在横边中点，命中那条横边', () {
      final (x, _) = geo.dot(2, 2);
      final (x2, _) = geo.dot(2, 3);
      final (_, y) = geo.dot(2, 2);
      final e = edgeAt(puzzle, geo, (x + x2) / 2, y);
      expect(e, puzzle.hIndex(2, 2));
    });

    test('正落在竖边中点，命中那条竖边', () {
      final (x, y) = geo.dot(2, 2);
      final (_, y2) = geo.dot(3, 2);
      final e = edgeAt(puzzle, geo, x, (y + y2) / 2);
      expect(e, puzzle.vIndex(2, 2));
    });

    test('格子正中心也必须命中（早先的中点距离法在这里是死区）', () {
      final (x, y) = geo.dot(2, 2);
      final e = edgeAt(
        puzzle,
        geo,
        x + geo.cell / 2,
        y + geo.cell / 2,
      );
      expect(e, isNotNull, reason: '格子正中心点不中 = 那一带是死区');
    });

    test('略偏向上边，命中上边而不是左边', () {
      final (x, y) = geo.dot(1, 1);
      // 横向在格子中间、纵向紧贴上边
      final e = edgeAt(puzzle, geo, x + geo.cell * 0.5, y + geo.cell * 0.05);
      expect(e, puzzle.hIndex(1, 1));
    });

    test('略偏向左边，命中左边而不是上边', () {
      final (x, y) = geo.dot(1, 1);
      final e = edgeAt(puzzle, geo, x + geo.cell * 0.05, y + geo.cell * 0.5);
      expect(e, puzzle.vIndex(1, 1));
    });
  });

  group('命中判定 —— 棋盘外不响应', () {
    test('远离棋盘的留白不命中', () {
      expect(edgeAt(puzzle, geo, geo.originX - geo.cell * 2, geo.originY), isNull);
      expect(
        edgeAt(puzzle, geo, geo.originX, geo.originY + 5 * geo.cell + geo.cell * 2),
        isNull,
      );
    });

    test('紧贴棋盘外沿仍可命中最外圈 —— 否则最外圈特别难点', () {
      final (x, y) = geo.dot(0, 0);
      final e = edgeAt(puzzle, geo, x + geo.cell * 0.5, y - geo.cell * 0.2);
      expect(e, puzzle.hIndex(0, 0));
    });
  });
}
