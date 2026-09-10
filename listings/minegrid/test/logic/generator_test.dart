import 'dart:math';

import 'package:flutter_test/flutter_test.dart';
import 'package:minegrid/logic/difficulty.dart';
import 'package:minegrid/logic/generator.dart';
import 'package:minegrid/logic/solver.dart';

/// 出题的验收：**每一盘都必须能不猜地走完**，而且要在玩家察觉不到的时间内出题。
///
/// 这两条缺一不可 —— 只保证「不用猜」但每次开局卡半秒，一样没法玩。
void main() {
  const samplesPerTier = 25;

  group('四个难度档', () {
    for (final difficulty in Difficulty.values) {
      test('${difficulty.label}：抽 $samplesPerTier 盘，盘盘不用猜', () {
        final rng = Random(4771 + difficulty.index);
        var worstAttempts = 0;

        for (var s = 0; s < samplesPerTier; s++) {
          // 首点随机取，覆盖角、边、中间三种情形 —— 安全区大小不同，
          // 开局摊开的面积差很多，命中率也会差。
          final firstClick = rng.nextInt(difficulty.cellCount);
          final generated = MineGenerator.generate(
            difficulty,
            firstClick,
            random: rng,
          );

          expect(
            generated.noGuess,
            isTrue,
            reason: '${difficulty.label} 第 $s 盘试满预算也没抽到不用猜的布局',
          );
          expect(
            generated.field.mines.length,
            difficulty.mines,
            reason: '雷数不对',
          );
          // 出题时验过一次，这里再独立验一次 —— 防的是「返回的盘和验过的盘不是同一个」
          expect(
            MineSolver.isNoGuess(generated.field, firstClick),
            isTrue,
          );

          if (generated.attempts > worstAttempts) {
            worstAttempts = generated.attempts;
          }
        }

        // 打印出来，密度调整时一眼看得到代价
        // ignore: avoid_print
        print('${difficulty.label}: 密度 '
            '${(difficulty.density * 100).toStringAsFixed(1)}% · '
            '最多试了 $worstAttempts 次');
      });
    }
  });

  group('首点保护', () {
    test('首点及其 3×3 邻域一定没有雷', () {
      final rng = Random(99);
      for (final difficulty in Difficulty.values) {
        for (var s = 0; s < 8; s++) {
          final firstClick = rng.nextInt(difficulty.cellCount);
          final field =
              MineGenerator.generate(difficulty, firstClick, random: rng).field;

          expect(field.isMine(firstClick), isFalse);
          for (final q in field.neighboursOf(firstClick)) {
            expect(
              field.isMine(q),
              isFalse,
              reason: '${difficulty.label} 首点 $firstClick 的邻居 $q 是雷',
            );
          }
          // 首点周围没雷 ⇒ 首点必然是 0 ⇒ 一按就摊开一片。
          // 开局只翻出个孤零零的数字的话，接下来就只能瞎点了。
          expect(field.adjacentMines(firstClick), 0);
        }
      }
    });
  });

  group('难度确实是递增的', () {
    test('雷密度逐档上升', () {
      final densities = Difficulty.values.map((d) => d.density).toList();
      for (var i = 1; i < densities.length; i++) {
        expect(densities[i], greaterThan(densities[i - 1]),
            reason: '密度必须递增，实际是 $densities');
      }
    });

    test('盘面逐档变大', () {
      final counts = Difficulty.values.map((d) => d.cellCount).toList();
      for (var i = 1; i < counts.length; i++) {
        expect(counts[i], greaterThan(counts[i - 1]));
      }
    });
  });
}
