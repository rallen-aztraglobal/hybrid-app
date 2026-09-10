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
final ColorRgba8 kBackground = ColorRgba8(0x13, 0x12, 0x18, 0xFF);
final ColorRgba8 kSurface = ColorRgba8(0x1D, 0x1B, 0x26, 0xFF);
final ColorRgba8 kBorder = ColorRgba8(0x2B, 0x28, 0x38, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0x9B, 0x7C, 0xFF, 0xFF);
final ColorRgba8 kOnAccent = ColorRgba8(0x12, 0x10, 0x1A, 0xFF);
final ColorRgba8 kMuted = ColorRgba8(0x8A, 0x85, 0xA0, 0xFF);
final ColorRgba8 kDone = ColorRgba8(0x5C, 0x58, 0x72, 0xFF);

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

/// 特色图：一份走到一半的核对表 —— 上面三条勾了（压暗划掉），下面两条还空着。
///
/// 「走到一半」是有意的：全勾完看不出这是个能反复用的表，
/// 一半勾一半空才说得出「一条条核对」这件事。
///
/// 不放文字：Play 会在特色图上叠自己的应用名与按钮，图里再写一遍标题只会打架；
/// 而且 package:image 只有位图字体，排出来的字远不如纯图形干净。
void _featureGraphic() {
  const w = 1024;
  const h = 500;
  const supersample = 3;
  const cw = w * supersample;
  const ch = h * supersample;
  const rows = 5;
  const doneRows = 3;

  final image = Image(width: cw, height: ch, numChannels: 4)
    ..clear(kBackground);

  final cardLeft = cw * 0.08;
  final cardW = cw * 0.84;
  final cardTop = ch * 0.08;
  final cardH = ch * 0.84;

  fillRect(
    image,
    x1: cardLeft.round(),
    y1: cardTop.round(),
    x2: (cardLeft + cardW).round(),
    y2: (cardTop + cardH).round(),
    radius: (ch * 0.06).round(),
    color: kSurface,
  );
  drawRect(
    image,
    x1: cardLeft.round(),
    y1: cardTop.round(),
    x2: (cardLeft + cardW).round(),
    y2: (cardTop + cardH).round(),
    radius: (ch * 0.06).round(),
    color: kBorder,
    thickness: ch * 0.006,
  );

  final rowH = cardH / rows;
  final box = rowH * 0.46;
  final boxRadius = (box * 0.28).round();
  final barH = rowH * 0.17;

  // 每行的横条长度不一样，看着才像一份真的清单而不是色卡
  const barWidths = <double>[0.62, 0.48, 0.70, 0.55, 0.40];

  for (var i = 0; i < rows; i++) {
    final done = i < doneRows;
    final rowTop = cardTop + i * rowH;
    final boxTop = rowTop + (rowH - box) / 2;
    final boxLeft = cardLeft + cardW * 0.06;

    if (done) {
      fillRect(
        image,
        x1: boxLeft.round(),
        y1: boxTop.round(),
        x2: (boxLeft + box).round(),
        y2: (boxTop + box).round(),
        radius: boxRadius,
        color: kAccent,
      );
      _check(image, boxLeft + box / 2, boxTop + box / 2, box, kOnAccent);
    } else {
      // 空框用「外圈实心 + 内圈填底色」画：drawRect 带圆角时描出来的线极细，
      // 深底上几乎看不见。
      final ring = box * 0.13;
      fillRect(
        image,
        x1: boxLeft.round(),
        y1: boxTop.round(),
        x2: (boxLeft + box).round(),
        y2: (boxTop + box).round(),
        radius: boxRadius,
        color: kMuted,
      );
      fillRect(
        image,
        x1: (boxLeft + ring).round(),
        y1: (boxTop + ring).round(),
        x2: (boxLeft + box - ring).round(),
        y2: (boxTop + box - ring).round(),
        radius: (boxRadius * 0.7).round(),
        color: kSurface,
      );
    }

    final barLeft = boxLeft + box + cardW * 0.05;
    final barTop = rowTop + (rowH - barH) / 2;
    final barW = cardW * barWidths[i];
    fillRect(
      image,
      x1: barLeft.round(),
      y1: barTop.round(),
      x2: (barLeft + barW).round(),
      y2: (barTop + barH).round(),
      radius: (barH / 2).round(),
      color: done ? kDone : kMuted,
    );

    // 勾掉的那几行画一道删除线，和界面里的删除线对应
    if (done) {
      fillRect(
        image,
        x1: barLeft.round(),
        y1: (barTop + barH * 0.42).round(),
        x2: (barLeft + barW).round(),
        y2: (barTop + barH * 0.58).round(),
        color: kSurface,
      );
    }
  }

  final out = copyResize(image, width: w, height: h,
      interpolation: Interpolation.average);
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote store/feature-graphic.png (24-bit, no alpha)');
}

void _check(Image image, double cx, double cy, double size, ColorRgba8 color) {
  final thickness = size * 0.155;
  final r = (thickness / 2).round();
  final ax = cx - size * 0.24;
  final ay = cy + size * 0.01;
  final bx = cx - size * 0.06;
  final by = cy + size * 0.19;
  final dx = cx + size * 0.26;
  final dy = cy - size * 0.20;

  drawLine(image,
      x1: ax.round(), y1: ay.round(), x2: bx.round(), y2: by.round(),
      color: color, thickness: thickness);
  drawLine(image,
      x1: bx.round(), y1: by.round(), x2: dx.round(), y2: dy.round(),
      color: color, thickness: thickness);
  fillCircle(image, x: ax.round(), y: ay.round(), radius: r, color: color);
  fillCircle(image, x: bx.round(), y: by.round(), radius: r, color: color);
  fillCircle(image, x: dx.round(), y: dy.round(), radius: r, color: color);
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
