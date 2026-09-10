// 生成 CheckLane 的三张图标源图，用法见 NonoPix 同名脚本：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**两行清单：上面一行勾了、下面一行没勾**。
// 只画一个对勾的话跟一堆待办类 App 撞脸；画成「一勾一空」才说得出这个包的重点 ——
// 一条条核对，而且勾完可以清掉重来。
//
// 只画两行不画三四行：行数一多，48dp 下每一行就细成一根发丝，全糊在一起。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x13, 0x12, 0x18, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0x9B, 0x7C, 0xFF, 0xFF);
final ColorRgba8 kOnAccent = ColorRgba8(0x12, 0x10, 0x1A, 0xFF);
final ColorRgba8 kMuted = ColorRgba8(0x8A, 0x85, 0xA0, 0xFF);
final ColorRgba8 kDone = ColorRgba8(0x5C, 0x58, 0x72, 0xFF);

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
  double x(double t) => left + art * t;
  double y(double t) => top + art * t;

  final box = art * 0.30;
  final boxRadius = (box * 0.28).round();
  final barHeight = art * 0.155;

  // —— 第一行：勾上了 ——
  fillRect(
    image,
    x1: x(0.0).round(),
    y1: y(0.10).round(),
    x2: (x(0.0) + box).round(),
    y2: (y(0.10) + box).round(),
    radius: boxRadius,
    color: kAccent,
  );
  _check(
    image,
    cx: x(0.0) + box / 2,
    cy: y(0.10) + box / 2,
    size: box,
    color: kOnAccent,
  );
  // 已勾这一行的横条压暗 —— 和界面里「勾过的字变灰」对应
  fillRect(
    image,
    x1: x(0.42).round(),
    y1: (y(0.10) + (box - barHeight) / 2).round(),
    x2: x(1.0).round(),
    y2: (y(0.10) + (box + barHeight) / 2).round(),
    radius: (barHeight / 2).round(),
    color: kDone,
  );

  // —— 第二行：还没勾 ——
  //
  // 空框用「外圈实心 + 内圈填底色」画，而不是 drawRect 描边：
  // drawRect 带圆角时描出来的线极细，深底上几乎看不见。
  // 内圈填的是 kBackground —— 前景层虽然是透明的，但背景层正好是这个色，
  // 叠起来看不出差别。
  final row2 = y(0.60);
  final ring = box * 0.14;
  fillRect(
    image,
    x1: x(0.0).round(),
    y1: row2.round(),
    x2: (x(0.0) + box).round(),
    y2: (row2 + box).round(),
    radius: boxRadius,
    color: kMuted,
  );
  fillRect(
    image,
    x1: (x(0.0) + ring).round(),
    y1: (row2 + ring).round(),
    x2: (x(0.0) + box - ring).round(),
    y2: (row2 + box - ring).round(),
    radius: (boxRadius * 0.7).round(),
    color: kBackground,
  );
  fillRect(
    image,
    x1: x(0.42).round(),
    y1: (row2 + (box - barHeight) / 2).round(),
    x2: x(1.0).round(),
    y2: (row2 + (box + barHeight) / 2).round(),
    radius: (barHeight / 2).round(),
    color: kMuted,
  );

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

/// 对勾：两段圆头线。
void _check(
  Image image, {
  required double cx,
  required double cy,
  required double size,
  required Color color,
}) {
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

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  wrote $name');
}
