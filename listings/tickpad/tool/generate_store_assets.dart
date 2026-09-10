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
final ColorRgba8 kBackground = ColorRgba8(0x10, 0x14, 0x18, 0xFF);
final ColorRgba8 kSurface = ColorRgba8(0x1A, 0x20, 0x26, 0xFF);
final ColorRgba8 kBorder = ColorRgba8(0x26, 0x2E, 0x36, 0xFF);
final ColorRgba8 kStroke = ColorRgba8(0xE9, 0xEE, 0xF2, 0xFF);
final ColorRgba8 kMuted = ColorRgba8(0x3A, 0x44, 0x4E, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0x3D, 0xDC, 0x84, 0xFF);

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

/// 特色图：四组正字计数，最后一组只划到一半。
///
/// 「数到一半」比「数完」更说明这是个计数器 —— 静止的完成态看不出动作，
/// 半组的那一笔让人一眼想到「按一下就多一画」。
///
/// 不放文字：Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
  const supersample = 3;
  const cw = w * supersample;
  const ch = h * supersample;

  final image = Image(width: cw, height: ch, numChannels: 4)
    ..clear(kBackground);

  // 底下垫一张卡片，和 App 里的计数卡片同一个观感
  final cardLeft = cw * 0.05;
  final cardTop = ch * 0.16;
  fillRect(
    image,
    x1: cardLeft.round(),
    y1: cardTop.round(),
    x2: (cw - cardLeft).round(),
    y2: (ch - cardTop).round(),
    radius: (ch * 0.07).round(),
    color: kSurface,
  );
  // 卡片描边
  drawRect(
    image,
    x1: cardLeft.round(),
    y1: cardTop.round(),
    x2: (cw - cardLeft).round(),
    y2: (ch - cardTop).round(),
    radius: (ch * 0.07).round(),
    color: kBorder,
    thickness: ch * 0.006,
  );

  final groupW = cw * 0.19;
  final gap = cw * 0.035;
  final totalW = groupW * 4 + gap * 3;
  final startX = (cw - totalW) / 2;
  final top = ch * 0.30;
  final bottom = ch * 0.70;
  final thickness = ch * 0.045;

  /// 一组正字：四竖 + 一斜。[strokes] 决定画几笔（1~5）。
  void group(double x, int strokes) {
    final barGap = groupW / 4.6;
    for (var i = 0; i < 4 && i < strokes; i++) {
      final bx = x + barGap * (i + 0.35);
      _bar(image, bx, top, bx, bottom, kStroke, thickness);
    }
    // 没画到的竖笔用暗色留个位置，让「还没数到」也看得见
    for (var i = strokes; i < 4; i++) {
      final bx = x + barGap * (i + 0.35);
      _bar(image, bx, top, bx, bottom, kMuted, thickness);
    }
    if (strokes >= 5) {
      _bar(image, x, bottom - ch * 0.02, x + groupW * 0.92,
          top + ch * 0.02, kAccent, thickness);
    }
  }

  group(startX, 5);
  group(startX + groupW + gap, 5);
  group(startX + (groupW + gap) * 2, 5);
  // 最后一组只数到 3
  group(startX + (groupW + gap) * 3, 3);

  final out = copyResize(image, width: w, height: h,
      interpolation: Interpolation.average);
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote store/feature-graphic.png (24-bit, no alpha)');
}

/// 一段圆头粗线。端头的圆比线宽的一半略大 —— drawLine 画斜线时实际宽度稍宽，
/// 严格取一半会在端头留下缺口。
void _bar(
  Image image,
  double x1,
  double y1,
  double x2,
  double y2,
  ColorRgba8 color,
  double thickness,
) {
  drawLine(image,
      x1: x1.round(), y1: y1.round(), x2: x2.round(), y2: y2.round(),
      color: color, thickness: thickness);
  final r = (thickness * 0.58).round();
  fillCircle(image, x: x1.round(), y: y1.round(), radius: r, color: color);
  fillCircle(image, x: x2.round(), y: y2.round(), radius: r, color: color);
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
