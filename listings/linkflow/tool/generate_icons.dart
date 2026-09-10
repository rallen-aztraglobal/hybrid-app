// 生成 LinkFlow 的三张图标源图，用法见 NonoPix 同名脚本：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**两个端点被一条折线连起来**，外加一对还没连上的端点 ——
// 这就是这个游戏的一句话说明。折线画成圆角粗管，和棋盘上真正的线是同一个观感。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应（pathColors 的前两支）。
final ColorRgba8 kBackground = ColorRgba8(0xF2, 0xEF, 0xE9, 0xFF);
final ColorRgba8 kPathA = ColorRgba8(0x1E, 0x88, 0xE5, 0xFF); // 蓝：已连通
final ColorRgba8 kPathB = ColorRgba8(0xE5, 0x39, 0x35, 0xFF); // 红：还没连

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

  final stroke = art * 0.17;
  final dot = art * 0.115;

  // 蓝色通路：左下 → 上 → 右上，一个直角弯。
  // 画成两段直线 + 拐角处补一个圆，就是圆角折线；这和棋盘上 strokeJoin.round 的效果一致。
  _pipe(image, x(0.12), y(0.80), x(0.12), y(0.30), kPathA, stroke);
  _pipe(image, x(0.12), y(0.30), x(0.72), y(0.30), kPathA, stroke);
  fillCircle(
    image,
    x: x(0.12).round(),
    y: y(0.30).round(),
    radius: (stroke / 2).round(),
    color: kPathA,
  );
  // 两个端点画得比线粗，和游戏里的端点圆点一致
  fillCircle(
    image, x: x(0.12).round(), y: y(0.80).round(),
    radius: dot.round(), color: kPathA,
  );
  fillCircle(
    image, x: x(0.72).round(), y: y(0.30).round(),
    radius: dot.round(), color: kPathA,
  );

  // 红色的一对端点：还没连上。有了它才看得出「这是道题」而不只是个箭头。
  fillCircle(
    image, x: x(0.55).round(), y: y(0.82).round(),
    radius: dot.round(), color: kPathB,
  );
  fillCircle(
    image, x: x(0.94).round(), y: y(0.82).round(),
    radius: dot.round(), color: kPathB,
  );

  return copyResize(
    image,
    width: kOutputSize,
    height: kOutputSize,
    interpolation: Interpolation.average,
  );
}

/// 一段圆头粗线。drawLine 的端头是平的，两端各补一个圆才是圆角管。
void _pipe(
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
}

void _write(String name, Image image) {
  File(name).writeAsBytesSync(encodePng(image));
  stdout.writeln('  wrote $name');
}
