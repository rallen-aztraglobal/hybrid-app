import 'package:flutter_test/flutter_test.dart';
import 'package:minegrid/logic/mine_field.dart';
import 'package:minegrid/logic/solver.dart';

/// 求解器是出题的把关人：它说「不用猜」，这盘才发给玩家。
/// 所以它自己得先被验：该放行的放行，该拦的拦。
///
/// 用的都是 3×3 / 4×1 这种能在脑子里推完的小盘 —— 一旦哪条断言挂了，
/// 看着注释就能自己走一遍，不用去猜求解器在想什么。
void main() {
  MineField field(int cols, int rows, Set<int> mines) =>
      MineField(cols: cols, rows: rows, mines: mines);

  group('放行：能纯靠推理走完的盘', () {
    test('一颗雷在角上，从对角开局 —— 连锁展开直接翻完', () {
      //  M . .
      //  . . .
      //  . . ●   ← 首点，周围 0 雷，一路摊开
      expect(MineSolver.isNoGuess(field(3, 3, <int>{0}), 8), isTrue);
    });

    test('需要用到子集规则的盘', () {
      //  M . M      首点 (2,1)。展开后 3 和 5 都只剩一个数字，
      //  . . .      基本规则在这里会卡住 —— 谁也定不下 0/1/2 里哪个是雷。
      //  . ● .      子集规则一比：{0,1}⊂{0,1,2} 且缺的雷差 1，2 必然是雷。
      expect(MineSolver.isNoGuess(field(3, 3, <int>{0, 2}), 7), isTrue);
    });

    test('需要用到总数规则的盘', () {
      //  ● . M M    一行四格。最右那格谁都挨不着（没有已翻开的邻居），
      //             只能靠「剩余雷数正好等于剩余未知格数」把它定死。
      expect(MineSolver.isNoGuess(field(4, 1, <int>{2, 3}), 0), isTrue);
    });
  });

  group('拦下：非猜不可的盘', () {
    test('首点只翻出一个数字，剩下八格里藏两颗雷 —— 没有任何推理入口', () {
      //  . M .
      //  . ● .      首点是 (1,1)，值为 2。除它以外全是未知，
      //  . M .      八格里两颗雷，怎么排都说得通。
      expect(MineSolver.isNoGuess(field(3, 3, <int>{1, 7}), 4), isFalse);
    });

    test('首点踩在雷上直接判否', () {
      expect(MineSolver.isNoGuess(field(3, 3, <int>{0}), 0), isFalse);
    });
  });

  group('不能误判', () {
    test('放行的盘里，非雷格必须一格不剩地被推出来', () {
      // 这条是求解器的定义本身：isNoGuess 为真 ⇔ 所有安全格都能被推开。
      // 把它单独写出来，是因为「推开了大部分」在界面上看不出问题，
      // 但玩家会在最后几格发现自己必须猜。
      final f = field(3, 3, <int>{0, 2});
      expect(MineSolver.isNoGuess(f, 7), isTrue);
      // 反向确认盘面本身没写错：9 格里 2 颗雷，7 个安全格
      var safe = 0;
      for (var i = 0; i < f.cellCount; i++) {
        if (!f.isMine(i)) safe++;
      }
      expect(safe, 7);
    });
  });
}
