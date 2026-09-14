import 'package:edgeloop8451/logic/difficulty.dart';
import 'package:edgeloop8451/logic/game.dart';
import 'package:edgeloop8451/logic/puzzle_library.dart';
import 'package:edgeloop8451/logic/solver.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('关卡库', () {
    test('四档都有关卡，数量与难度递减一致', () {
      for (final d in Difficulty.values) {
        expect(levelCountFor(d), greaterThan(0), reason: '${d.name} 不能是空档');
      }
      expect(levelCountFor(Difficulty.normal),
          greaterThanOrEqualTo(levelCountFor(Difficulty.master)),
          reason: '越难的档题越少是有意的 —— 生成代价高，玩到的人也少');
    });

    test('每一关的尺寸与该档声明的一致', () {
      for (final d in Difficulty.values) {
        for (final p in levelsFor(d)) {
          expect(p.rows, d.rows, reason: '${d.name} 的题尺寸不符');
          expect(p.cols, d.cols);
        }
      }
    });

    test('每一关都恰好一个解', () {
      // 这是「每关有且仅有一个解」这句商店文案的**唯一**依据。
      // 删掉这项测试之前，必须先把那句话从 PLAY_STORE_LISTING.md 里删掉。
      //
      // 跑满四档要几十秒（Master 单题百万级节点），但这是唯一能保证卖点为真的办法 ——
      // 抽查几关不够：不唯一的题恰好落在没抽到的那几关里，是最可能发生的情形。
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          final r = EdgeLoopSolver(levels[i], nodeCap: 20000000).solve();
          expect(r.exhausted, isTrue,
              reason: '${d.name} 第 ${i + 1} 关搜索被截断，唯一性无从谈起');
          expect(r.solutions.length, 1,
              reason: '${d.name} 第 ${i + 1} 关有 ${r.solutions.length} 个解');
        }
      }
    }, timeout: const Timeout(Duration(minutes: 5)));

    test('每一关的解，游戏都判为已解出', () {
      for (final d in Difficulty.values) {
        for (final p in levelsFor(d)) {
          final solution = EdgeLoopSolver(p, nodeCap: 20000000)
              .solve()
              .solutions
              .single;
          final g = EdgeLoopGame(p);
          for (var e = 0; e < solution.length; e++) {
            if (solution[e] == 1) g.marks[e] = EdgeMark.line;
          }
          expect(g.isSolved, isTrue,
              reason: '${d.name} 有一关的正解在游戏里过不了关');
        }
      }
    }, timeout: const Timeout(Duration(minutes: 5)));

    test('没有重复的题面', () {
      for (final d in Difficulty.values) {
        final codes = levelsFor(d).map((p) => p.encode()).toList();
        expect(codes.toSet().length, codes.length,
            reason: '${d.name} 里有重复的题');
      }
      // 跨档也不该重复（尺寸不同，理论上不会，但钉住）
      final all = Difficulty.values
          .expand((d) => levelsFor(d).map((p) => p.encode()))
          .toList();
      expect(all.toSet().length, all.length);
    });

    test('提示数取值合法：只在 0..3', () {
      for (final d in Difficulty.values) {
        for (final p in levelsFor(d)) {
          for (final c in p.clues) {
            if (c != null) expect(c, inInclusiveRange(0, 3));
          }
        }
      }
    });
  });
}
