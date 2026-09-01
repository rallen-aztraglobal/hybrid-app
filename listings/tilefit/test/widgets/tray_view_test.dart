import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/models/piece.dart';
import 'package:tilefit/widgets/tile.dart';
import 'package:tilefit/widgets/tray_view.dart';

Piece _piece(String id) => kPieceCatalog.firstWhere((p) => p.id == id);

Widget _host(List<Piece?> pieces, {VoidCallback? onDragEnd}) => MaterialApp(
  home: Scaffold(
    body: Center(
      child: SizedBox(
        width: 360,
        height: 140,
        child: TrayView(
          pieces: pieces,
          isPlaceable: (_) => true,
          thumbCellSize: 20,
          dragCellSize: 40,
          onDragEnd: onDragEnd ?? () {},
        ),
      ),
    ),
  ),
);

/// 按住某点并拖出去，返回拖动是否真的启动了。
///
/// 判据是"跟手的浮层有没有出现"：拖动一旦启动，[Draggable] 会把 feedback 挂到 Overlay 上，
/// 于是树里的 [PieceView] 会比拖之前多一个。
Future<bool> _dragStartsFrom(WidgetTester tester, Offset point) async {
  final before = find.byType(PieceView).evaluate().length;
  final gesture = await tester.startGesture(point);
  await tester.pump(const Duration(milliseconds: 20));
  await gesture.moveTo(point + const Offset(0, -60));
  await tester.pump(const Duration(milliseconds: 20));
  final started = find.byType(PieceView).evaluate().length > before;
  await gesture.up();
  await tester.pumpAndSettle();
  return started;
}

void main() {
  // 外接矩形的宽或高是偶数时，矩形正中心正好落在两格之间的那道缝上
  // （方格只画到格边长的 88%，缝里没有任何可命中的东西）。
  // 这几个形状按下正中心都必须能拖起来 —— 那正是手指最自然的落点。
  for (final id in <String>['h2', 'v2', 'sq2', 'el3a', 'v4']) {
    testWidgets('按在 $id 外接矩形的正中心（两格之间的缝）也能拖起来', (WidgetTester tester) async {
      await tester.pumpWidget(_host(<Piece?>[_piece(id), null, null]));

      final seam = tester.getCenter(find.byType(PieceView).first);
      expect(await _dragStartsFrom(tester, seam), isTrue);
    });
  }

  testWidgets('按在 L 形空出来的那个角上也能拖起来', (WidgetTester tester) async {
    // el3a = ['#.', '##']，右上角那格是空的，底下没有方格可命中。
    await tester.pumpWidget(_host(<Piece?>[_piece('el3a'), null, null]));

    final box = tester.getRect(find.byType(PieceView).first);
    // 右上角那一格的中心。
    final emptyCorner = Offset(
      box.left + box.width * 0.75,
      box.top + box.height * 0.25,
    );
    expect(await _dragStartsFrom(tester, emptyCorner), isTrue);
  });

  testWidgets('拖动结束会回调 onDragEnd', (WidgetTester tester) async {
    var ended = false;
    await tester.pumpWidget(
      _host(<Piece?>[_piece('sq2'), null, null], onDragEnd: () => ended = true),
    );

    await _dragStartsFrom(
      tester,
      tester.getCenter(find.byType(PieceView).first),
    );
    expect(ended, isTrue);
  });

  testWidgets('已用掉的槽不出现可拖的方块', (WidgetTester tester) async {
    await tester.pumpWidget(_host(<Piece?>[null, null, null]));

    expect(find.byType(Draggable<int>), findsNothing);
    expect(find.byType(PieceView), findsNothing);
  });
}
