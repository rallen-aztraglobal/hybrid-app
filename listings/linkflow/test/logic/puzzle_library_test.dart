import 'package:flutter_test/flutter_test.dart';
import 'package:linkflow/logic/game.dart';
import 'package:linkflow/logic/puzzle_library.dart';
import 'package:linkflow/logic/solver.dart';
import 'package:linkflow/theme/app_colors.dart';

/// 题库是生成出来的（`tool/generate_levels.dart`），这组测试是它的**验收**。
///
/// 生成器改参数、换种子、或者有人手改了一行，都由这里兜住 ——
/// 坏题在界面上表现得极隐晦（玩家只会觉得「这条怎么连都连不完」），
/// 靠人肉试玩根本发现不了。
void main() {
  // 求解器要跑遍全部关卡，跑一次存下来给多个用例共用。
  final results = <Difficulty, List<SolveResult>>{};

  setUpAll(() {
    for (final d in Difficulty.values) {
      results[d] = <SolveResult>[
        for (final level in levelsFor(d)) FlowSolver(level.build()).run(),
      ];
    }
  });

  group('题库结构', () {
    test('每档都有题，且数量够用', () {
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        expect(levels, isNotEmpty, reason: '${d.label} 档没有任何题');
        expect(levels.length, greaterThanOrEqualTo(8),
            reason: '${d.label} 只有 ${levels.length} 道，太少会很快重复');
      }
    });

    test('同一档里所有题的盘面尺寸一致', () {
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        final side = levels.first.side;
        for (var i = 0; i < levels.length; i++) {
          expect(levels[i].side, side,
              reason: '${d.label} 第 ${i + 1} 关的 side 和同档其他题不一样');
        }
      }
    });

    test('盘面逐档变大', () {
      final sides = <int>[
        for (final d in Difficulty.values) levelsFor(d).first.side,
      ];
      for (var i = 1; i < sides.length; i++) {
        expect(sides[i], greaterThan(sides[i - 1]),
            reason: '难度档的盘面必须是递增的，实际是 $sides');
      }
    });

    test('通路数不超过配色表长度', () {
      // 超了就会循环取色，同一题里出现两条同色的线 —— 直接没法玩。
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          expect(levels[i].paths.length,
              lessThanOrEqualTo(AppColors.pathColors.length),
              reason: '${d.label} 第 ${i + 1} 关有 ${levels[i].paths.length} 条线，'
                  '配色只有 ${AppColors.pathColors.length} 支');
        }
      }
    });

    test('同一题里通路的键不重复', () {
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          final keys = levels[i].paths.map((p) => p.key).toList();
          expect(keys.toSet().length, keys.length,
              reason: '${d.label} 第 ${i + 1} 关有重复的通路键');
        }
      }
    });
  });

  group('可解性 —— 题库的验收标准', () {
    test('每道题的答案都不重不漏地铺满整盘', () {
      final failed = <String>[];
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          final problems = levels[i].build().validate();
          if (problems.isNotEmpty) {
            failed.add('${d.label} 第 ${i + 1} 关:\n      ${problems.join("\n      ")}');
          }
        }
      }
      expect(failed, isEmpty, reason: '以下题目的答案有问题：\n  ${failed.join("\n  ")}');
    });

    test('每条通路的两个端点互不相同', () {
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          final puzzle = levels[i].build();
          for (final entry in puzzle.endpoints.entries) {
            expect(entry.value.first == entry.value.last, isFalse,
                reason: '${d.label} 第 ${i + 1} 关的通路 ${entry.key} 首尾同格');
          }
        }
      }
    });

    test('每道题都只有一个解', () {
      // 多解题是这类游戏最伤口碑的地方：玩家会发现「随便乱连也能过」，
      // 或者反过来，照着自己算出来的另一种解连上却不被认。
      final failed = <String>[];
      for (final d in Difficulty.values) {
        for (var i = 0; i < results[d]!.length; i++) {
          final r = results[d]![i];
          if (!r.isUnique) {
            failed.add('${d.label} 第 ${i + 1} 关：'
                '解数 ${r.solutions}${r.exhausted ? "" : "（没搜完）"}');
          }
        }
      }
      expect(failed, isEmpty, reason: '以下题目不是唯一解：\n  ${failed.join("\n  ")}');
    });

    test('按答案画一遍就能通关', () {
      // 交叉验证：求解器说有解是一回事，游戏状态机认不认是另一回事。
      // 这条把两边的「通关」口径钉在一起。
      for (final d in Difficulty.values) {
        final levels = levelsFor(d);
        for (var i = 0; i < levels.length; i++) {
          final puzzle = levels[i].build();
          final game = FlowGame(puzzle);
          for (final key in puzzle.colorKeys) {
            final cells = puzzle.paths[key]!;
            game.beginDrag(cells.first);
            for (var j = 1; j < cells.length; j++) {
              game.dragTo(cells[j]);
            }
            game.endDrag();
          }
          expect(game.isSolved, isTrue,
              reason: '${d.label} 第 ${i + 1} 关照答案画完却没判通关');
        }
      }
    });
  });

  group('难度排布', () {
    test('档内由易到难', () {
      // 生成器按求解节点数升序出厂。顺着关卡往下打，难度应该是一条平滑的坡，
      // 不能一会儿难一会儿易。
      for (final d in Difficulty.values) {
        final nodes = results[d]!.map((r) => r.nodes).toList();
        for (var i = 1; i < nodes.length; i++) {
          expect(nodes[i], greaterThanOrEqualTo(nodes[i - 1]),
              reason: '${d.label} 第 ${i + 1} 关（${nodes[i]} 节点）比'
                  '第 $i 关（${nodes[i - 1]} 节点）还容易');
        }
      }
    });

    test('难档的最高关确实比正常档更难', () {
      final normalMax = results[Difficulty.normal]!
          .map((r) => r.nodes)
          .reduce((a, b) => a > b ? a : b);
      for (final d in <Difficulty>[
        Difficulty.hard,
        Difficulty.expert,
        Difficulty.master,
      ]) {
        final max = results[d]!.map((r) => r.nodes).reduce((a, b) => a > b ? a : b);
        expect(max, greaterThan(normalMax),
            reason: '${d.label} 档最难的一关只要 $max 节点，'
                '还不如 Normal 档的 $normalMax');
      }
    });
  });
}
