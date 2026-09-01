// 生成 Play Console 需要的商店素材，全部写到 store/ 下：
//
//   dart run tool/generate_store_assets.dart <截图1.png> <截图2.png> ...
//
//   store/icon-512.png         512×512，商店图标（由 icon.png 缩放而来）
//   store/feature-graphic.png  1024×500，特色图
//   store/screenshot-N.png     1200×2400，实机截图补边而成
//
// 为什么截图要补边：实机是 1080×2400（9:20），比 Play 允许的最长边比例 2:1 还要瘦，
// 直接传会被拒。左右各补 60px 的 #0E1116 —— 与 App 背景同色，所以补出来的边和画面
// 浑然一体，看不出是补的。**不要**改成裁剪：裁掉的是棋盘两侧，画面会失衡。

import 'dart:io';

import 'package:image/image.dart';

final ColorRgba8 kBackground = ColorRgba8(0x0E, 0x11, 0x16, 0xFF);
final ColorRgba8 kEmptyCell = ColorRgba8(0x1E, 0x24, 0x2E, 0xFF);

/// 与 AppColors.pieceColors 一致。
final List<ColorRgba8> kPieceColors = <ColorRgba8>[
  ColorRgba8(0x4E, 0xCD, 0xC4, 0xFF),
  ColorRgba8(0x5A, 0xA9, 0xE6, 0xFF),
  ColorRgba8(0x7C, 0x7C, 0xE0, 0xFF),
  ColorRgba8(0xB9, 0x7F, 0xE0, 0xFF),
  ColorRgba8(0xE8, 0x7F, 0xA8, 0xFF),
  ColorRgba8(0xF0, 0x88, 0x5A, 0xFF),
  ColorRgba8(0xF0, 0xC1, 0x5A, 0xFF),
  ColorRgba8(0x8F, 0xCF, 0x6E, 0xFF),
  ColorRgba8(0x5F, 0xD0, 0xA8, 0xFF),
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
  final out = copyResize(
    src,
    width: 512,
    height: 512,
    interpolation: Interpolation.average,
  );
  File('store/icon-512.png').writeAsBytesSync(encodePng(out));
  stdout.writeln('  wrote store/icon-512.png');
}

/// 特色图：一条 12×4 的网格带，上面摆几个游戏里的形状。
///
/// 不放文字 —— Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架；
/// 而且 package:image 只有位图字体，排出来的字远不如纯图形干净。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
  const cols = 12;
  const rows = 4;

  const supersample = 3;
  final image = Image(
    width: w * supersample,
    height: h * supersample,
    numChannels: 4,
  )..clear(kBackground);

  final cell = (h * supersample) / (rows + 1.6);
  final gridW = cols * cell;
  final left = ((w * supersample) - gridW) / 2;
  final top = ((h * supersample) - rows * cell) / 2;
  final radius = (cell * 0.22).round();
  final inset = cell * 0.06;

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

  // 先铺满空格，再把形状盖上去 —— 和游戏里棋盘的观感一致。
  for (var r = 0; r < rows; r++) {
    for (var c = 0; c < cols; c++) {
      tile(r, c, kEmptyCell);
    }
  }

  // 几个真实存在于形状表里的块，横向排开。
  const shapes = <List<List<int>>>[
    // 直角三格
    <List<int>>[
      [1, 1],
      [2, 1],
      [2, 2],
    ],
    // 2×2
    <List<int>>[
      [1, 4],
      [1, 5],
      [2, 4],
      [2, 5],
    ],
    // 三连
    <List<int>>[
      [1, 7],
      [1, 8],
      [1, 9],
    ],
    // 二连（竖）
    <List<int>>[
      [2, 8],
      [3, 8],
    ],
    // 单格
    <List<int>>[
      [0, 10],
    ],
  ];
  const colorIndex = <int>[7, 5, 1, 2, 0];

  for (var s = 0; s < shapes.length; s++) {
    for (final cellPos in shapes[s]) {
      tile(cellPos[0], cellPos[1], kPieceColors[colorIndex[s]]);
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
