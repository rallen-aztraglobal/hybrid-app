import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/logic/game.dart';
import 'package:tilefit/models/piece.dart';

final Piece _dot = kPieceCatalog.firstWhere((p) => p.id == 'dot');

/// 找一个「第一个槽是扁平形状（高度 1）」的种子。
///
/// 计分测试要把某一行填到只差方块本身那几格，方块必须能横躺在这一行里；
/// 竖着的形状会跨多行、凑不出「刚好补满一行」的局面。发牌是随机的，故这里扫种子
/// 而不是写死一个 —— 将来调了形状权重，这段仍能自己找到可用的开局。
Game _gameWithFlatFirstPiece() {
  for (var seed = 0; seed < 500; seed++) {
    final game = Game(seed: seed);
    if (game.tray[0]!.height == 1) return game;
  }
  throw StateError('前 500 个种子里没有扁平的首个方块');
}

/// 找一个合法落点并落下去。返回是否落成功（false = 无路可走）。
bool _playFirstLegalMove(Game game) {
  for (var i = 0; i < game.tray.length; i++) {
    final piece = game.tray[i];
    if (piece == null) continue;
    for (var r = 0; r <= game.board.size - piece.height; r++) {
      for (var c = 0; c <= game.board.size - piece.width; c++) {
        if (game.canPlace(i, r, c)) {
          game.place(i, r, c);
          return true;
        }
      }
    }
  }
  return false;
}

void main() {
  group('Game 起手', () {
    test('开局发满一手、棋盘全空、零分、未结束', () {
      final game = Game(seed: 1);

      expect(game.tray.length, 3);
      expect(game.tray.every((p) => p != null), isTrue);
      expect(game.board.filledCount, 0);
      expect(game.score, 0);
      expect(game.isOver, isFalse);
    });

    test('同一个种子发出同一手牌', () {
      final a = Game(seed: 42);
      final b = Game(seed: 42);

      expect(
        a.tray.map((p) => p!.id).toList(),
        b.tray.map((p) => p!.id).toList(),
      );
    });
  });

  group('落子', () {
    test('落子得分等于填入的格数', () {
      final game = Game(seed: 3);
      final piece = game.tray[0]!;

      final result = game.place(0, 0, 0);

      expect(result, isNotNull);
      expect(result!.cellsPlaced, piece.size);
      expect(game.score, piece.size);
      expect(game.tray[0], isNull, reason: '用掉的槽应清空');
    });

    test('非法落子返回 null 且不改动任何状态', () {
      final game = Game(seed: 3);
      game.place(0, 0, 0);
      final scoreAfterFirst = game.score;
      final filledAfterFirst = game.board.filledCount;

      expect(game.place(0, 0, 0), isNull, reason: '空槽');
      expect(game.place(9, 0, 0), isNull, reason: '槽下标越界');
      expect(game.place(1, -1, 0), isNull, reason: '落点越界');

      expect(game.score, scoreAfterFirst);
      expect(game.board.filledCount, filledAfterFirst);
    });

    test('三个槽用完才补发新的一手', () {
      final game = Game(seed: 11);

      var placed = 0;
      while (placed < 2 && _playFirstLegalMove(game)) {
        placed++;
      }

      expect(placed, 2);
      expect(
        game.tray.where((p) => p == null).length,
        2,
        reason: '还剩一个没用，不该补发',
      );

      _playFirstLegalMove(game);
      expect(game.tray.every((p) => p != null), isTrue, reason: '三个都用完后应补满');
    });
  });

  group('消除计分', () {
    test('lineBonus 随同时消除的线数平方增长', () {
      expect(Game.lineBonus(0), 0);
      expect(Game.lineBonus(-1), 0);
      expect(Game.lineBonus(1), 10);
      expect(Game.lineBonus(2), 40);
      expect(Game.lineBonus(3), 90);
    });

    test('消除一行 = 落子格数 + 10 分', () {
      final game = _gameWithFlatFirstPiece();
      final piece = game.tray[0]!;

      // 把第 0 行里 piece 落点之外的格全部填掉，只留左端 piece.width 格。
      // 单列只填一格，不可能凑满任何一列，故这一步不会提前触发消除。
      for (var c = piece.width; c < game.board.size; c++) {
        game.board.place(_dot, 0, c);
      }

      final result = game.place(0, 0, 0);

      expect(result, isNotNull);
      expect(result!.linesCleared, 1);
      expect(game.score, piece.size + Game.lineBonus(1));
      expect(game.board.filledCount, 0);
    });
  });

  group('结束判定', () {
    test('一直贪心地放，最终会结束，且结束时确实无处可放', () {
      final game = Game(seed: 2024);

      var guard = 0;
      while (!game.isOver && guard < 2000) {
        guard++;
        if (!_playFirstLegalMove(game)) break;
      }

      expect(game.isOver, isTrue, reason: '贪心首位填放最终必然卡死');
      expect(guard, lessThan(2000), reason: '不该跑到保险丝上限');
      expect(game.score, greaterThan(0));
      for (var i = 0; i < game.tray.length; i++) {
        expect(game.isPlaceable(i), isFalse, reason: '结束时第 $i 个槽仍有落点');
      }
    });

    test('结束后再落子无效', () {
      final game = Game(seed: 2024);
      while (!game.isOver && _playFirstLegalMove(game)) {}

      expect(game.place(0, 0, 0), isNull);
      expect(game.canPlace(0, 0, 0), isFalse);
    });
  });

  test('restart 把棋盘、分数、结束标记全部还原', () {
    final game = Game(seed: 2024);
    while (!game.isOver && _playFirstLegalMove(game)) {}
    expect(game.isOver, isTrue);

    game.restart();

    expect(game.isOver, isFalse);
    expect(game.score, 0);
    expect(game.board.filledCount, 0);
    expect(game.tray.every((p) => p != null), isTrue);
  });

  test('tray 暴露出去的是副本，外部改不动内部状态', () {
    final game = Game(seed: 8);
    expect(() => game.tray[0] = null, throwsUnsupportedError);
  });
}
