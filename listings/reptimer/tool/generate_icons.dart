// 生成 RepTimer 的三张图标源图：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**一圈按阶段分段的环** —— 青绿段是「做」、蓝段是「休」，交替四组，
// 段与段之间留缺口。这说的正是这个 App 的全部：做—休交替，若干组。
//
// 为什么不画秒表或沙漏：那两个符号表达的是「计时」，而市面上每个计时器都这么画。
// 「分段的环」才说得出这个包的重点 —— 它计的不是一段时间，是交替的若干段。
//
// 深色底：App 本体是深色主题，图标跟着走，装到桌面上不会显得是另一个 App。
//
// 抗锯齿：package:image 不给圆弧做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';
import 'dart:math' as math;

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 lib/theme/app_colors.dart 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x11, 0x15, 0x1A, 0xFF);
final ColorRgba8 kWork = ColorRgba8(0x2D, 0xD4, 0xBF, 0xFF); // 青绿：做
final ColorRgba8 kRest = ColorRgba8(0x60, 0xA5, 0xFA, 0xFF); // 蓝：休

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

  final cx = kCanvas / 2;
  final cy = kCanvas / 2;
  final outer = kCanvas * artRatio / 2;
  final stroke = outer * 0.30;
  final ringRadius = outer - stroke / 2;

  // 八段：做/休 交替四组。段间要留出**看得见**的缺口 —— 没有缺口就成了一个
  // 双色圆环，看不出「分段」这层意思。
  //
  // 缺口要留多大不能凭感觉：每段两端各有一个半径 = 笔宽/2 的圆头端帽，
  // 沿弧各多伸 (stroke/2)/ringRadius 弧度。这里 stroke = 0.30×outer、
  // ringRadius = 0.85×outer，于是单端约 10°、两端共约 20°。
  // 第一版取 11° 的缺口，反而被端帽吃掉还重叠了 9°，出来就是个没有分段的双色环。
  // 取 30° 后净剩约 10° 的可见缺口。
  const segments = 8;
  const gapDegrees = 30.0;
  final sweep = 360.0 / segments - gapDegrees;

  for (var i = 0; i < segments; i++) {
    final start = i * (360.0 / segments) + gapDegrees / 2;
    _arc(
      image,
      cx: cx,
      cy: cy,
      radius: ringRadius,
      thickness: stroke,
      startDeg: start,
      sweepDeg: sweep,
      color: i.isEven ? kWork : kRest,
    );
  }

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

/// 画一段圆弧。
///
/// package:image 没有画弧的 API，用密集地沿弧铺圆点来实现 ——
/// 步长取「半个笔宽对应的弧度」，保证相邻圆点充分重叠、弧看不出是点串出来的。
void _arc(
  Image image, {
  required double cx,
  required double cy,
  required double radius,
  required double thickness,
  required double startDeg,
  required double sweepDeg,
  required Color color,
}) {
  final r = (thickness / 2).round();
  // 沿弧前进 thickness/8 就补一个点：重叠越多边缘越平滑（取 /4 时能看出轻微扇贝纹）。
  final stepRad = (thickness / 8) / radius;
  final startRad = startDeg * math.pi / 180;
  final sweepRad = sweepDeg * math.pi / 180;

  for (var t = 0.0; t <= sweepRad; t += stepRad) {
    final a = startRad + t;
    fillCircle(
      image,
      x: (cx + radius * math.cos(a)).round(),
      y: (cy + radius * math.sin(a)).round(),
      radius: r,
      color: color,
    );
  }
  // 补上终点，免得因步长取整少画一小截。
  final a = startRad + sweepRad;
  fillCircle(
    image,
    x: (cx + radius * math.cos(a)).round(),
    y: (cy + radius * math.sin(a)).round(),
    radius: r,
    color: color,
  );
}

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  $name  ${image.width}x${image.height}');
}
