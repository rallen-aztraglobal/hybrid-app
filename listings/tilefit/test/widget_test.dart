import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/models/piece.dart';
import 'package:tilefit/screens/game_screen.dart';
import 'package:tilefit/widgets/board_view.dart';
import 'package:tilefit/widgets/tile.dart';
import 'package:tilefit/widgets/tray_view.dart';

/// 直接挂 [GameScreen]，**不挂 TileFitApp**。
///
/// App 的 home 是启动闸（GateScreen），一 pump 就会发真实的网关判定请求；
/// 测试环境里那既慢又不确定，而且测的根本不是游戏。这里绕开启动闸，
/// 测的仍是玩家看到的同一棵 widget 树。
Widget _wrap(Widget child) => MaterialApp(home: child);

/// 把待放区第一个方块拖到棋盘左上角那一格，返回它的格数（= 本次应得的分）。
///
/// 落点**不能**随手取棋盘中心：发牌是随机的，方块尺寸每次不同，按中心算出来的
/// 左上角落点可能越界，测试就成了碰运气。这里按 App 的同一套几何反推出
/// 「让方块左上角正好落在第 (0,0) 格」的手指位置 —— 空棋盘上任何形状都放得下，
/// 与发到什么牌无关。
Future<int> _dragFirstPieceToOrigin(WidgetTester tester) async {
  final draggable = find.byType(Draggable<int>).first;
  final thumb = tester.widget<PieceView>(
    find.descendant(of: draggable, matching: find.byType(PieceView)).first,
  );
  final Piece piece = thumb.piece;
  // 缩略图按格边长的一半画（GameScreen._thumbRatio），故格边长是它的两倍。
  final cell = thumb.cellSize * 2;

  // 手指位置 = 浮层左上角 + dragAnchorStrategy 的偏移；
  // 要让浮层左上角落在棋盘第 (0,0) 格，手指就得落在这里。
  final boardTopLeft = tester.getTopLeft(find.byType(BoardView));
  final target =
      boardTopLeft +
      Offset(cell * 0.16, cell * 0.16) + // BoardView 底板的内边距
      Offset(
        piece.width * cell / 2,
        piece.height * cell / 2 + kDragLiftCells * cell,
      );

  // 分几步移动：Draggable 要越过拖动阈值才启动，DragTarget 的 onMove
  // 也要真的收到中间帧才会更新落点预览。
  final from = tester.getCenter(draggable);
  final gesture = await tester.startGesture(from);
  await tester.pump(const Duration(milliseconds: 20));
  await gesture.moveTo(Offset(from.dx, from.dy - 40));
  await tester.pump(const Duration(milliseconds: 20));
  await gesture.moveTo(target);
  await tester.pump(const Duration(milliseconds: 20));
  await gesture.up();
  await tester.pumpAndSettle();

  return piece.size;
}

int _score(WidgetTester tester) => int.parse(
  tester.widget<Text>(find.byKey(const ValueKey<String>('scoreText'))).data!,
);

void main() {
  testWidgets('开局渲染出计分条、棋盘与待放区', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));

    expect(find.byType(BoardView), findsOneWidget);
    expect(find.byType(TrayView), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('scoreText')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('bestText')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('restartButton')), findsOneWidget);
  });

  testWidgets('开局分数为 0，且没有结束浮层', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));

    expect(_score(tester), 0);
    expect(find.byKey(const ValueKey<String>('gameOverOverlay')), findsNothing);
  });

  testWidgets('待放区开局有三个可拖的方块', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));

    expect(find.byType(Draggable<int>), findsNWidgets(3));
  });

  testWidgets('把方块拖到棋盘左上角会落子，得分等于它的格数', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));

    final cells = await _dragFirstPieceToOrigin(tester);

    // 最宽的形状也只有 5 格，落在左上角凑不满任何一行一列，故不会有消除奖励。
    // 分数恰好等于格数，说明整条链路（Draggable → DragTarget → 坐标换算 → 落子）
    // 把方块放到了预期的那一格，而不是"碰巧放下了"。
    expect(_score(tester), cells);
  });

  testWidgets('拖动结束后棋盘上不留落点预览', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));
    await _dragFirstPieceToOrigin(tester);

    final board = tester.widget<BoardView>(find.byType(BoardView));
    expect(board.ghost, isNull);
  });

  testWidgets('重开按钮把分数清回 0', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap(const GameScreen()));
    await _dragFirstPieceToOrigin(tester);
    expect(_score(tester), greaterThan(0));

    await tester.tap(find.byKey(const ValueKey<String>('restartButton')));
    await tester.pumpAndSettle();

    expect(_score(tester), 0);
  });
}
