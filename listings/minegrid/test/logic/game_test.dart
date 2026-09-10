import 'package:flutter_test/flutter_test.dart';
import 'package:minegrid/logic/difficulty.dart';
import 'package:minegrid/logic/game.dart';
import 'package:minegrid/logic/mine_field.dart';

/// 状态机的行为。用 [MineGame.withField] 指定雷区，这样每条断言都指着具体某一格，
/// 挂了能直接看出是哪一步错。
void main() {
  const d = Difficulty.normal; // 8 列 × 10 行 · 10 雷

  /// 把 10 颗雷全塞在最后一行（下标 72..81 里取 10 个），
  /// 上面九行全是安全格，方便断言。
  MineGame gameWithBottomRowMines() {
    final mines = <int>{for (var c = 0; c < 8; c++) 9 * 8 + c, 8 * 8 + 0, 8 * 8 + 1};
    return MineGame.withField(
      d,
      MineField(cols: d.cols, rows: d.rows, mines: mines),
    );
  }

  group('挖', () {
    test('第一下永远安全 —— 雷区是首点之后才生成的', () {
      for (var trial = 0; trial < 20; trial++) {
        final game = MineGame(d);
        expect(game.status, GameStatus.ready);
        expect(game.dig(trial % d.cellCount), isTrue);
        expect(game.status, GameStatus.playing, reason: '第一下就炸了');
        // 首点周围没雷 ⇒ 必然连锁展开一片
        expect(game.revealedCount, greaterThan(1));
      }
    });

    test('空格连锁展开，碰到数字就停', () {
      final game = gameWithBottomRowMines();
      game.dig(0); // 左上角，离雷很远
      expect(game.isRevealed(0), isTrue);
      // 第 7 行（下标 56..63）挨着第 8 行的雷，应当被翻开但不再往下带
      expect(game.isRevealed(56), isTrue);
      // 雷本身不会被翻开
      expect(game.isRevealed(72), isFalse);
    });

    test('插了旗的格子挖不动 —— 防误触', () {
      final game = gameWithBottomRowMines();
      game.toggleFlag(0);
      expect(game.dig(0), isFalse);
      expect(game.isRevealed(0), isFalse);
    });

    test('已翻开的格子再挖没有反应', () {
      final game = gameWithBottomRowMines();
      game.dig(0);
      expect(game.dig(0), isFalse);
    });

    test('踩雷 → 记下是哪一颗，并结束这局', () {
      final game = gameWithBottomRowMines();
      expect(game.dig(72), isTrue);
      expect(game.status, GameStatus.lost);
      expect(game.explodedIndex, 72);
      // 结束后所有操作都失效
      expect(game.dig(0), isFalse);
      expect(game.toggleFlag(0), isFalse);
    });
  });

  group('插旗', () {
    test('插旗与取消，剩余雷数跟着变', () {
      final game = gameWithBottomRowMines();
      expect(game.minesLeft, 10);
      game.toggleFlag(0);
      expect(game.minesLeft, 9);
      game.toggleFlag(0);
      expect(game.minesLeft, 10);
    });

    test('已翻开的格子插不了旗', () {
      final game = gameWithBottomRowMines();
      game.dig(0);
      expect(game.toggleFlag(0), isFalse);
    });

    test('插错旗会让剩余雷数算错 —— 这是扫雷的常规行为，不是 bug', () {
      final game = gameWithBottomRowMines();
      game.toggleFlag(0); // 0 不是雷
      expect(game.minesLeft, 9);
    });
  });

  group('和弦', () {
    // 56 在第 7 行第 0 列，邻居是 48、49、57、64、65，其中 64/65 是雷 ⇒ 值为 2。
    // 直接挖 56（而不是从角上摊开）是有意的：这样它的五个邻居都还盖着，
    // 和弦才有东西可挖。

    test('旗数对得上就一次挖开周围', () {
      final game = gameWithBottomRowMines();
      game.dig(56);
      expect(game.valueAt(56), 2);
      expect(game.revealedCount, 1, reason: '数字格不连锁，只翻开自己');

      game.toggleFlag(64);
      game.toggleFlag(65);
      expect(game.chord(56), isTrue);
      expect(game.isRevealed(48), isTrue);
      expect(game.isRevealed(49), isTrue);
      expect(game.isRevealed(57), isTrue);
      // 48/49 是空格，连锁一路把上面八行全摊开 —— 这盘的安全格到此正好翻完，
      // 所以和弦这一下直接通关了。顺带验到「和弦之后也要判胜」这条。
      expect(game.status, GameStatus.won);
    });

    test('旗数对不上不动手', () {
      final game = gameWithBottomRowMines();
      game.dig(56);
      game.toggleFlag(64); // 只插了一面旗，56 的值是 2
      expect(game.chord(56), isFalse);
      expect(game.isRevealed(48), isFalse);
    });

    test('旗插错了，和弦会当场炸 —— 这是应有的代价', () {
      final game = gameWithBottomRowMines();
      game.dig(56);
      game.toggleFlag(64);
      game.toggleFlag(57); // 57 不是雷；真正的 65 没插
      expect(game.chord(56), isTrue);
      expect(game.status, GameStatus.lost);
      expect(game.explodedIndex, 65);
    });

    test('没翻开的格子不能和弦', () {
      final game = gameWithBottomRowMines();
      game.dig(56);
      expect(game.chord(72), isFalse);
    });

    test('值为 0 的格子不和弦 —— 它周围本来就已经全摊开了', () {
      final game = gameWithBottomRowMines();
      game.dig(0);
      expect(game.valueAt(0), 0);
      expect(game.chord(0), isFalse);
    });
  });

  group('通关', () {
    test('翻完所有安全格就赢，并自动把剩下的旗补上', () {
      final field = MineField(
        cols: d.cols,
        rows: d.rows,
        mines: <int>{for (var c = 0; c < 8; c++) 9 * 8 + c, 8 * 8 + 0, 8 * 8 + 1},
      );
      final game = MineGame.withField(d, field);
      for (var i = 0; i < d.cellCount; i++) {
        if (!field.isMine(i)) game.dig(i);
      }
      expect(game.status, GameStatus.won);
      expect(game.minesLeft, 0, reason: '通关后剩余雷数应当归零');
      for (var i = 0; i < d.cellCount; i++) {
        if (field.isMine(i)) {
          expect(game.isFlagged(i), isTrue, reason: '$i 号雷没被自动插旗');
        }
      }
    });
  });

  group('重来', () {
    test('reset 清空一切，回到开局前', () {
      final game = gameWithBottomRowMines();
      game.dig(0);
      game.toggleFlag(72);
      game.reset();
      expect(game.status, GameStatus.ready);
      expect(game.revealedCount, 0);
      expect(game.flagsPlaced, 0);
      expect(game.explodedIndex, isNull);
      expect(game.minesLeft, d.mines);
    });
  });
}
