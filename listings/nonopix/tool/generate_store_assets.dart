// 生成 Play Console 需要的商店素材，全部写到 store/ 下：
//
//   dart run tool/generate_store_assets.dart <截图1.png> <截图2.png> ...
//
//   store/icon-512.png         512×512，商店图标（由 icon.png 缩放而来）
//   store/feature-graphic.png  1024×500，特色图
//   store/screenshot-N.png     1200×2400，实机截图补边而成
//
// 为什么截图要补边：实机是 1080×2400（9:20），比 Play 允许的最长边比例 2:1 还要瘦，
// 直接传会被拒。左右各补 60px 的背景色 —— 与 App 背景同色，所以补出来的边和画面
// 浑然一体，看不出是补的。**不要**改成裁剪：裁掉的是棋盘两侧，画面会失衡。

import 'dart:io';

import 'package:image/image.dart';

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF2, 0xEF, 0xE9, 0xFF);
final ColorRgba8 kBoardSurface = ColorRgba8(0xFF, 0xFF, 0xFF, 0xFF);
final ColorRgba8 kEmptyCell = ColorRgba8(0xEB, 0xE7, 0xE0, 0xFF);
final ColorRgba8 kFilled = ColorRgba8(0x4F, 0x46, 0xE5, 0xFF);
final ColorRgba8 kFilledTop = ColorRgba8(0x6C, 0x63, 0xFF, 0xFF);

const int kStoreWidth = 1200;

void main(List<String> args) {
  Directory('store').createSync(recursive: true);

  _icon512();
  _featureGraphic();

  if (args.isEmpty) {
    stdout.writeln('没有传截图路径，只生成了图标与特色图。');
    return;
  }
  for (var i = 0; i < args.length; i++) {
    _padScreenshot(args[i], i + 1);
  }
}

void _icon512() {
  final src = decodePng(File('icon.png').readAsBytesSync());
  if (src == null) {
    stderr.writeln('读不到 icon.png，先跑 tool/generate_icons.dart');
    exitCode = 1;
    return;
  }
  final out = copyResize(
    src,
    width: 512,
    height: 512,
    interpolation: Interpolation.average,
  );
  File('store/icon-512.png').writeAsBytesSync(encodePng(out));
  stdout.writeln('  wrote store/icon-512.png');
}

/// 特色图：一条横向的数织棋盘，上面涂出三幅小图。
///
/// 不放文字 —— Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架；
/// 而且 package:image 只有位图字体，排出来的字远不如纯图形干净。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
  // 列数受画布宽度限制：cell 由高度定，cols × cell 再加两侧留白必须 ≤ 画布宽，
  // 否则白卡片会被切在画面外、圆角看不见。21 列就超了，15 列刚好。
  const cols = 15;
  const rows = 7;
  const supersample = 3;

  final image = Image(
    width: w * supersample,
    height: h * supersample,
    numChannels: 4,
  )..clear(kBackground);

  final cell = (h * supersample) / (rows + 2.2);
  final gridW = cols * cell;
  final left = ((w * supersample) - gridW) / 2;
  final top = ((h * supersample) - rows * cell) / 2;
  final radius = (cell * 0.2).round();
  final inset = cell * 0.07;

  // 棋盘底板：白卡片，从暖底色里浮出来，和游戏里一样
  fillRect(
    image,
    x1: (left - cell * 0.5).round(),
    y1: (top - cell * 0.5).round(),
    x2: (left + gridW + cell * 0.5).round(),
    y2: (top + rows * cell + cell * 0.5).round(),
    radius: (cell * 0.5).round(),
    color: kBoardSurface,
  );

  void tile(int r, int c, ColorRgba8 color) {
    fillRect(
      image,
      x1: (left + c * cell + inset).round(),
      y1: (top + r * cell + inset).round(),
      x2: (left + (c + 1) * cell - inset).round(),
      y2: (top + (r + 1) * cell - inset).round(),
      radius: radius,
      color: color,
    );
  }

  for (var r = 0; r < rows; r++) {
    for (var c = 0; c < cols; c++) {
      tile(r, c, kEmptyCell);
    }
  }

  // 两幅涂出来的小图，横向排开。`#` 是涂黑的格子。
  const pictures = <List<String>>[
    // 心
    <String>['.#.#.', '#####', '#####', '.###.', '..#..'],
    // 星
    <String>['..#..', '.###.', '#####', '.###.', '#...#'],
  ];

  for (var p = 0; p < pictures.length; p++) {
    final art = pictures[p];
    final offsetC = 2 + p * 6;
    for (var r = 0; r < art.length; r++) {
      for (var c = 0; c < art[r].length; c++) {
        if (art[r][c] != '#') continue;
        tile(r + 1, offsetC + c, r < 2 ? kFilledTop : kFilled);
      }
    }
  }

  final out = copyResize(
    image,
    width: w,
    height: h,
    interpolation: Interpolation.average,
  );
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  // 画布是按 4 通道建的（超采样时需要），这里落盘前转成 3 通道。
  final flat = out.convert(numChannels: 3);
  File('store/feature-graphic.png').writeAsBytesSync(encodePng(flat));
  stdout.writeln('  wrote store/feature-graphic.png (24-bit, no alpha)');
}

void _padScreenshot(String path, int index) {
  final src = decodePng(File(path).readAsBytesSync());
  if (src == null) {
    stderr.writeln('读不到 $path，跳过');
    exitCode = 1;
    return;
  }
  if (src.width >= kStoreWidth) {
    stderr.writeln('$path 宽 ${src.width} ≥ $kStoreWidth，无需补边，跳过');
    return;
  }

  final out = Image(width: kStoreWidth, height: src.height, numChannels: 4)
    ..clear(kBackground);
  compositeImage(out, src, dstX: (kStoreWidth - src.width) ~/ 2, dstY: 0);

  final name = 'store/screenshot-$index.png';
  // 与特色图同口径去掉 alpha：截图本来就不透明，留着通道只是白白多占体积。
  File(name).writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote $name  (${out.width}×${out.height})');
}
