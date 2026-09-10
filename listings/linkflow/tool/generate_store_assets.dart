// 生成 Play Console 需要的商店素材，全部写到 store/ 下：
//
//   dart run tool/generate_store_assets.dart <截图1.png> <截图2.png> ...
//
//   store/icon-512.png         512×512，商店图标（由 icon.png 缩放而来）
//   store/feature-graphic.png  1024×500，特色图
//   store/screenshot-N.png     1200×2400，实机截图补边而成
//
// 为什么截图要补边：实机是 1080×2400（9:20），比 Play 允许的最长边比例 2:1 还要瘦，
// 直接传会被拒。左右各补 60px 的背景色，与 App 背景同色，看不出是补的。
// **不要**改成裁剪：裁掉的是棋盘两侧，画面会失衡。

import 'dart:io';

import 'package:image/image.dart';

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF2, 0xEF, 0xE9, 0xFF);
final ColorRgba8 kBoardSurface = ColorRgba8(0xFF, 0xFF, 0xFF, 0xFF);
final ColorRgba8 kEmptyCell = ColorRgba8(0xED, 0xE9, 0xE2, 0xFF);

/// AppColors.pathColors 的前几支。
final List<ColorRgba8> kPaths = <ColorRgba8>[
  ColorRgba8(0xE5, 0x39, 0x35, 0xFF), // 红
  ColorRgba8(0x1E, 0x88, 0xE5, 0xFF), // 蓝
  ColorRgba8(0x2E, 0x7D, 0x32, 0xFF), // 深绿
  ColorRgba8(0xFB, 0x8C, 0x00, 0xFF), // 橙
];

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
  File('store/icon-512.png').writeAsBytesSync(encodePng(
    copyResize(src, width: 512, height: 512,
        interpolation: Interpolation.average),
  ));
  stdout.writeln('  wrote store/icon-512.png');
}

/// 特色图：一块横向的棋盘，上面画着四条已经连通的彩色通路。
///
/// 通路都画成圆角粗管、端点是更粗的圆点，和游戏里完全一样 ——
/// 商店图和实际画面对得上，用户点进来不会有落差。
///
/// 不放文字：Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
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

  fillRect(
    image,
    x1: (left - cell * 0.5).round(),
    y1: (top - cell * 0.5).round(),
    x2: (left + gridW + cell * 0.5).round(),
    y2: (top + rows * cell + cell * 0.5).round(),
    radius: (cell * 0.5).round(),
    color: kBoardSurface,
  );

  final inset = cell * 0.06;
  for (var r = 0; r < rows; r++) {
    for (var c = 0; c < cols; c++) {
      fillRect(
        image,
        x1: (left + c * cell + inset).round(),
        y1: (top + r * cell + inset).round(),
        x2: (left + (c + 1) * cell - inset).round(),
        y2: (top + (r + 1) * cell - inset).round(),
        radius: (cell * 0.18).round(),
        color: kEmptyCell,
      );
    }
  }

  double cx(num c) => left + (c + 0.5) * cell;
  double cy(num r) => top + (r + 0.5) * cell;

  final stroke = cell * 0.42;
  final dot = cell * 0.30;

  /// 一条通路：按格子坐标依次连过去，两端画端点圆。
  void path(List<List<int>> cells, ColorRgba8 color) {
    for (var i = 1; i < cells.length; i++) {
      drawLine(
        image,
        x1: cx(cells[i - 1][1]).round(),
        y1: cy(cells[i - 1][0]).round(),
        x2: cx(cells[i][1]).round(),
        y2: cy(cells[i][0]).round(),
        color: color,
        thickness: stroke,
      );
      // 拐角补圆，等价于 strokeJoin.round
      fillCircle(
        image,
        x: cx(cells[i][1]).round(),
        y: cy(cells[i][0]).round(),
        radius: (stroke / 2).round(),
        color: color,
      );
    }
    for (final end in <List<int>>[cells.first, cells.last]) {
      fillCircle(
        image,
        x: cx(end[1]).round(),
        y: cy(end[0]).round(),
        radius: dot.round(),
        color: color,
      );
    }
  }

  // 四条互不相交的通路，把整块盘面铺得比较满
  path(<List<int>>[
    [0, 0], [0, 4], [3, 4], [3, 0],
  ], kPaths[0]);
  path(<List<int>>[
    [0, 6], [0, 10], [2, 10], [2, 6],
  ], kPaths[1]);
  path(<List<int>>[
    [5, 1], [5, 8],
  ], kPaths[2]);
  path(<List<int>>[
    [1, 12], [5, 12], [5, 14],
  ], kPaths[3]);

  final out = copyResize(image, width: w, height: h,
      interpolation: Interpolation.average);
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
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
  File(name).writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote $name  (${out.width}×${out.height})');
}
