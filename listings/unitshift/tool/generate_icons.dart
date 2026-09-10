// 生成 UnitShift 的三张图标源图，用法见 NonoPix 同名脚本：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**一对反向的箭头**（⇄）—— 换算类工具的通用符号，不认字也看得懂，
// 而且笔画少，48dp 下不会糊。两支箭用不同深浅的蓝，避免看成一个双头箭头。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF4, 0xF5, 0xF7, 0xFF);
final ColorRgba8 kArrowTop = ColorRgba8(0x2F, 0x6F, 0xED, 0xFF);
final ColorRgba8 kArrowBottom = ColorRgba8(0x7D, 0xA6, 0xF5, 0xFF);

void main() {
  _write('icon_foreground.png', _art(transparent: true, artRatio: 0.60));
  _write('icon_background.png', _solid(kBackground));
  _write('icon.png', _art(transparent: false, artRatio: 0.68));
  stdout.writeln('三张图标源图已生成，接着跑：dart run flutter_launcher_icons');
}

Image _solid(Color color) =>
    Image(width: kOutputSize, height: kOutputSize, numChannels: 4)
      ..clear(color);

Image _art({required bool transparent, required double artRatio}) {
  final image = Image(width: kCanvas, height: kCanvas, numChannels: 4)
    ..clear(transparent ? ColorRgba8(0, 0, 0, 0) : kBackground);

  final art = kCanvas * artRatio;
  final left = (kCanvas - art) / 2;
  final top = (kCanvas - art) / 2;
  double x(double t) => left + art * t;
  double y(double t) => top + art * t;

  final shaft = art * 0.15;
  final head = art * 0.19;

  // 上面一支：向右
  _arrow(
    image,
    fromX: x(0.06),
    toX: x(0.80),
    centerY: y(0.30),
    shaft: shaft,
    head: head,
    color: kArrowTop,
    pointsRight: true,
  );

  // 下面一支：向左。深浅拉开，免得看成一根双头箭。
  _arrow(
    image,
    fromX: x(0.94),
    toX: x(0.20),
    centerY: y(0.70),
    shaft: shaft,
    head: head,
    color: kArrowBottom,
    pointsRight: false,
  );

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

/// 一支箭：一段圆头横杆 + 一个三角箭头。
void _arrow(
  Image image, {
  required double fromX,
  required double toX,
  required double centerY,
  required double shaft,
  required double head,
  required Color color,
  required bool pointsRight,
}) {
  // 杆画到箭头根部为止，免得三角形被杆顶出去一块
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
  fillCircle(
    image,
    x: fromX.round(),
    y: centerY.round(),
    radius: (shaft / 2).round(),
    color: color,
  );

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

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  wrote $name');
}
