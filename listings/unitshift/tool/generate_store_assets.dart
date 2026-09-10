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
final ColorRgba8 kBackground = ColorRgba8(0xF4, 0xF5, 0xF7, 0xFF);
final ColorRgba8 kSurface = ColorRgba8(0xFF, 0xFF, 0xFF, 0xFF);
final ColorRgba8 kBorder = ColorRgba8(0xE3, 0xE5, 0xEA, 0xFF);
final ColorRgba8 kMuted = ColorRgba8(0xD8, 0xDC, 0xE3, 0xFF);
final ColorRgba8 kText = ColorRgba8(0xB4, 0xBA, 0xC6, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0x2F, 0x6F, 0xED, 0xFF);
final ColorRgba8 kAccentSoft = ColorRgba8(0xE8, 0xF0, 0xFE, 0xFF);

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

/// 特色图：左边一张「单位列表」卡片（第一行是选中的源单位），右边一对反向箭头。
///
/// 画的就是这个 App 的核心交互 —— 输入一次、整列跟着变。
/// 行里的名字与数字用灰条代替：package:image 只有位图字体，1024 宽下排出来的字
/// 又糊又不对齐，灰条反而更像一张排版整齐的表。
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

  // —— 左边：单位列表卡片 ——
  final cardLeft = cw * 0.06;
  final cardTop = ch * 0.12;
  final cardW = cw * 0.52;
  final cardH = ch * 0.76;
  final rowH = cardH / 5;

  fillRect(
    image,
    x1: cardLeft.round(),
    y1: cardTop.round(),
    x2: (cardLeft + cardW).round(),
    y2: (cardTop + cardH).round(),
    radius: (ch * 0.05).round(),
    color: kSurface,
  );

  for (var i = 0; i < 5; i++) {
    final top = cardTop + i * rowH;
    // 第一行是当前源单位：淡蓝底 + 蓝色数字
    if (i == 0) {
      fillRect(
        image,
        x1: cardLeft.round(),
        y1: top.round(),
        x2: (cardLeft + cardW).round(),
        y2: (top + rowH).round(),
        radius: (ch * 0.05).round(),
        color: kAccentSoft,
      );
    } else {
      // 行间分隔线
      fillRect(
        image,
        x1: (cardLeft + cardW * 0.04).round(),
        y1: top.round(),
        x2: (cardLeft + cardW * 0.96).round(),
        y2: (top + ch * 0.004).round(),
        color: kBorder,
      );
    }

    final barH = rowH * 0.22;
    final barY = top + (rowH - barH) / 2;
    final rowColor = i == 0 ? kAccent : kMuted;

    // 左侧：单位简写（短条）
    fillRect(
      image,
      x1: (cardLeft + cardW * 0.06).round(),
      y1: barY.round(),
      x2: (cardLeft + cardW * 0.06 + cardW * 0.11).round(),
      y2: (barY + barH).round(),
      radius: (barH / 2).round(),
      color: rowColor,
    );
    // 中间：单位全名（中条，灰）
    fillRect(
      image,
      x1: (cardLeft + cardW * 0.23).round(),
      y1: barY.round(),
      x2: (cardLeft + cardW * 0.23 + cardW * 0.24).round(),
      y2: (barY + barH).round(),
      radius: (barH / 2).round(),
      color: kText,
    );
    // 右侧：数值（长度各行不同，像一列右对齐的数字）
    final widths = <double>[0.20, 0.26, 0.15, 0.30, 0.22];
    final valueW = cardW * widths[i];
    fillRect(
      image,
      x1: (cardLeft + cardW * 0.94 - valueW).round(),
      y1: barY.round(),
      x2: (cardLeft + cardW * 0.94).round(),
      y2: (barY + barH).round(),
      radius: (barH / 2).round(),
      color: rowColor,
    );
  }

  // —— 右边：一对反向箭头 ——
  final ax = cw * 0.64;
  final aw = cw * 0.30;
  final shaft = ch * 0.075;
  final head = ch * 0.095;
  _arrow(image, ax, ax + aw, ch * 0.40, shaft, head, kAccent, true);
  _arrow(image, ax + aw, ax, ch * 0.62, shaft, head,
      ColorRgba8(0x7D, 0xA6, 0xF5, 0xFF), false);

  final out = copyResize(image, width: w, height: h,
      interpolation: Interpolation.average);
  // 特色图必须是 24 位、**不带 alpha 通道** —— 带透明通道的 Play 会直接拒收。
  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(out.convert(numChannels: 3)));
  stdout.writeln('  wrote store/feature-graphic.png (24-bit, no alpha)');
}

void _arrow(
  Image image,
  double fromX,
  double toX,
  double centerY,
  double shaft,
  double head,
  ColorRgba8 color,
  bool pointsRight,
) {
  final tipInset = pointsRight ? -head * 0.6 : head * 0.6;
  drawLine(
    image,
    x1: fromX.round(),
    y1: centerY.round(),
    x2: (toX + tipInset).round(),
    y2: centerY.round(),
    color: color,
    thickness: shaft,
  );
  fillCircle(image,
      x: fromX.round(), y: centerY.round(),
      radius: (shaft / 2).round(), color: color);
  final back = pointsRight ? toX - head : toX + head;
  fillPolygon(
    image,
    vertices: <Point>[
      Point(toX, centerY),
      Point(back, centerY - head),
      Point(back, centerY + head),
    ],
    color: color,
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
