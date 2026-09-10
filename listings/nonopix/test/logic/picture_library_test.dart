import 'package:flutter_test/flutter_test.dart';
import 'package:nonopix/logic/generator.dart';
import 'package:nonopix/logic/picture_library.dart';

void main() {
  group('图库结构', () {
    test('每个尺寸都有图，且数量够用', () {
      for (final size in PuzzleSize.values) {
        final pics = picturesFor(size.side);
        expect(pics, isNotEmpty, reason: '${size.label} 没有任何图');
        expect(
          pics.length,
          greaterThanOrEqualTo(8),
          reason: '${size.label} 只有 ${pics.length} 张，太少会很快重复',
        );
      }
    });

    test('图案是正方形且尺寸对得上', () {
      for (final size in PuzzleSize.values) {
        for (final p in picturesFor(size.side)) {
          expect(p.rows.length, size.side, reason: '${p.name} 行数不对');
          for (final row in p.rows) {
            expect(row.length, size.side, reason: '${p.name} 有一行长度不对');
          }
        }
      }
    });

    test('名字不重复', () {
      for (final size in PuzzleSize.values) {
        final names = picturesFor(size.side).map((p) => p.name).toList();
        expect(names.toSet().length, names.length, reason: '${size.label} 有重名');
      }
    });

    test('没有全空或全满的图', () {
      for (final size in PuzzleSize.values) {
        for (final p in picturesFor(size.side)) {
          final filled = p.toNonogram().filledCount;
          expect(filled, greaterThan(0), reason: '${p.name} 是空的');
          expect(filled, lessThan(size.side * size.side),
              reason: '${p.name} 全涂满了');
        }
      }
    });
  });

  group('可解性 —— 这是图库的验收标准', () {
    test('每一张图都能纯逻辑解出，不需要猜', () {
      final failed = <String>[];
      for (final size in PuzzleSize.values) {
        for (final p in picturesFor(size.side)) {
          if (!p.toNonogram().isLineSolvable) {
            failed.add('${size.label} · ${p.name}');
          }
        }
      }
      expect(
        failed,
        isEmpty,
        reason: '以下图片需要猜才能解，必须调整图案：\n  ${failed.join("\n  ")}',
      );
    });
  });
}
