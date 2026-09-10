// 生成 TickPad 的三张图标源图，用法见 NonoPix 同名脚本：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**正字计数的第五笔**：四竖 + 一斜。
// 这是全世界通用的手工计数符号，不认字也看得懂，笔画又极简，48dp 下仍然清楚。
// 比画一个「+」强得多 —— 加号太泛，什么 App 都能用。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x10, 0x14, 0x18, 0xFF);
final ColorRgba8 kStroke = ColorRgba8(0xE9, 0xEE, 0xF2, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0x3D, 0xDC, 0x84, 0xFF);

void main() {
  _write('icon_foreground.png', _art(transparent: true, artRatio: 0.58));
  _write('icon_background.png', _solid(kBackground));
  _write('icon.png', _art(transparent: false, artRatio: 0.66));
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

  final thickness = art * 0.115;

  // 四竖。用浅色（不是主色）—— 主色留给第五笔，才有「刚记上这一个」的意思。
  for (var i = 0; i < 4; i++) {
    final cx = x(0.10 + i * 0.235);
    _bar(image, cx, y(0.14), cx, y(0.86), kStroke, thickness);
  }

  // 第五笔：斜着划过前四竖，用主色。
  _bar(image, x(0.02), y(0.80), x(0.90), y(0.20), kAccent, thickness);

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

/// 一段圆头粗线。drawLine 的端头是平的，两端各补一个圆才是圆角笔画。
void _bar(
  Image image,
  double x1,
  double y1,
  double x2,
  double y2,
  Color color,
  double thickness,
) {
  drawLine(
    image,
    x1: x1.round(),
    y1: y1.round(),
    x2: x2.round(),
    y2: y2.round(),
    color: color,
    thickness: thickness,
  );
  // 端头的圆比线宽的一半略大一点点：drawLine 画斜线时的实际宽度比 thickness 稍宽，
  // 严格取一半会在端头留下一个小缺口。
  final r = (thickness * 0.58).round();
  fillCircle(image, x: x1.round(), y: y1.round(), radius: r, color: color);
  fillCircle(image, x: x2.round(), y: y2.round(), radius: r, color: color);
}

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  wrote $name');
}
