// 生成 MineGrid 的三张图标源图，用法见 NonoPix 同名脚本：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**格子上插着一面旗**。
// 为什么不画雷：雷代表踩爆了，是失败；旗代表推理出来了，是这个包想给的印象
// （每一盘都能不猜地走完）。而且旗的轮廓比雷更简单，48dp 下仍认得出。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x10, 0x13, 0x19, 0xFF);
final ColorRgba8 kCovered = ColorRgba8(0x2C, 0x32, 0x40, 0xFF);
final ColorRgba8 kCoveredTop = ColorRgba8(0x37, 0x3E, 0x4F, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0xFF, 0x9A, 0x3C, 0xFF);

void main() {
  _write('icon_foreground.png', _art(transparent: true, artRatio: 0.62));
  _write('icon_background.png', _solid(kBackground));
  _write('icon.png', _art(transparent: false, artRatio: 0.70));
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
  final radius = (art * 0.2).round();

  // 一块「盖着的格子」：底色 + 顶部高光，和棋盘上的未翻开格是同一个画法
  fillRect(
    image,
    x1: left.round(),
    y1: top.round(),
    x2: (left + art).round(),
    y2: (top + art).round(),
    radius: radius,
    color: kCovered,
  );
  fillRect(
    image,
    x1: left.round(),
    y1: top.round(),
    x2: (left + art).round(),
    y2: (top + art * 0.42).round(),
    radius: radius,
    color: kCoveredTop,
  );

  double x(double t) => left + art * t;
  double y(double t) => top + art * t;

  // 旗杆
  final pole = art * 0.075;
  drawLine(
    image,
    x1: x(0.40).round(),
    y1: y(0.20).round(),
    x2: x(0.40).round(),
    y2: y(0.80).round(),
    color: kAccent,
    thickness: pole,
  );
  fillCircle(
    image, x: x(0.40).round(), y: y(0.80).round(),
    radius: (pole / 2).round(), color: kAccent,
  );

  // 旗面：一个三角形，从杆顶向右下收
  fillPolygon(
    image,
    vertices: <Point>[
      Point(x(0.40), y(0.18)),
      Point(x(0.80), y(0.36)),
      Point(x(0.40), y(0.54)),
    ],
    color: kAccent,
  );

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  wrote $name');
}
