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

import 'dart:io';

import 'package:image/image.dart';

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x10, 0x13, 0x19, 0xFF);
final ColorRgba8 kBoardSurface = ColorRgba8(0x1A, 0x1E, 0x27, 0xFF);
final ColorRgba8 kCovered = ColorRgba8(0x2C, 0x32, 0x40, 0xFF);
final ColorRgba8 kCoveredTop = ColorRgba8(0x37, 0x3E, 0x4F, 0xFF);
final ColorRgba8 kRevealed = ColorRgba8(0x14, 0x18, 0x21, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0xFF, 0x9A, 0x3C, 0xFF);

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

/// 特色图：一片横向的雷区，大部分还盖着，挖开一块，插了两面旗。
///
/// 不画数字：package:image 只有位图字体，1024 宽下排出来的数字又糊又不对齐，
/// 还不如让盖着/挖开/插旗这三种状态自己说话 —— 玩过扫雷的一眼就懂。
///
/// 不放文字：Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
  const cols = 17;
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
  final radius = (cell * 0.2).round();

  /// 挖开的一片：左下角那块连通区
  bool isRevealed(int r, int c) => c < 6 && r > 2;

  /// 插旗的三格。散开放：只插一两面的话，整张图看着就只是一片灰格子，
  /// 认不出是扫雷。
  bool isFlag(int r, int c) =>
      (r == 1 && c == 7) || (r == 4 && c == 12) || (r == 6 && c == 9);

  for (var r = 0; r < rows; r++) {
    for (var c = 0; c < cols; c++) {
      final x1 = (left + c * cell + inset).round();
      final y1 = (top + r * cell + inset).round();
      final x2 = (left + (c + 1) * cell - inset).round();
      final y2 = (top + (r + 1) * cell - inset).round();

      if (isRevealed(r, c)) {
        fillRect(image,
            x1: x1, y1: y1, x2: x2, y2: y2, radius: radius, color: kRevealed);
        continue;
      }

      // 盖着的格子：底色 + 顶部高光，和棋盘上一样
      fillRect(image,
          x1: x1, y1: y1, x2: x2, y2: y2, radius: radius, color: kCovered);
      fillRect(
        image,
        x1: x1,
        y1: y1,
        x2: x2,
        y2: (y1 + (y2 - y1) * 0.42).round(),
        radius: radius,
        color: kCoveredTop,
      );

      if (isFlag(r, c)) _flag(image, x1.toDouble(), y1.toDouble(), cell);
    }
  }

  final out = copyResize(image, width: w, height: h,
      interpolation: Interpolation.average);
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote store/feature-graphic.png (24-bit, no alpha)');
}

/// 一面旗，画在以 (x, y) 为左上角、边长 cell 的格子里。
///
/// 比 App 里的旗画得更满一些：特色图在商店列表里会被缩到很小，
/// 按界面比例画的话旗子就只剩一个点了。
void _flag(Image image, double x, double y, double cell) {
  final pole = cell * 0.11;
  final px = x + cell * 0.33;
  drawLine(
    image,
    x1: px.round(),
    y1: (y + cell * 0.14).round(),
    x2: px.round(),
    y2: (y + cell * 0.86).round(),
    color: kAccent,
    thickness: pole,
  );
  fillCircle(
    image,
    x: px.round(),
    y: (y + cell * 0.86).round(),
    radius: (pole / 2).round(),
    color: kAccent,
  );
  fillPolygon(
    image,
    vertices: <Point>[
      Point(px, y + cell * 0.11),
      Point(x + cell * 0.88, y + cell * 0.33),
      Point(px, y + cell * 0.55),
    ],
    color: kAccent,
  );
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
