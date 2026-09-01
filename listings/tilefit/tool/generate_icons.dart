// 生成 TileFit 的三张图标源图（仓库根目录下的 icon.png / icon_foreground.png /
// icon_background.png），再由 flutter_launcher_icons 渲染成各档 Android 资源：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是纯几何拼出来的（不依赖任何字体渲染），故各机型、各 DPI 下完全一致。
// 取色直接抄 lib/theme/app_colors.dart，图标与游戏本体是同一套颜色。
//
// 抗锯齿的做法：package:image 的圆角矩形没有抗锯齿，直接画 1024 会露出锯齿边。
// 这里先按 4 倍尺寸画、再用均值插值缩回去，等价于 4×4 超采样，边缘干净。

import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0x0E, 0x11, 0x16, 0xFF);
final ColorRgba8 kTeal = ColorRgba8(0x4E, 0xCD, 0xC4, 0xFF);
final ColorRgba8 kBlue = ColorRgba8(0x5A, 0xA9, 0xE6, 0xFF);
final ColorRgba8 kViolet = ColorRgba8(0x7C, 0x7C, 0xE0, 0xFF);

/// 空槽的填充与描边。取 AppColors.emptyCell 作填充、再描一圈更亮的边：
/// 只描边的话，缩到启动器里的 48dp 时那圈线会糊没，图标看上去只剩三个块、
/// "还差一格"的意思就丢了。填充给了它体积，描边给了它轮廓。
final ColorRgba8 kSlotFill = ColorRgba8(0x1E, 0x24, 0x2E, 0xFF);
final ColorRgba8 kSlotStroke = ColorRgba8(0x4A, 0x54, 0x62, 0xFF);

void main() {
  // 前景层（自适应图标）：启动器会再套一层 16% 的 inset 并按各家形状裁切，
  // 故图案只占画布的 60%，四角留够余量，圆形遮罩下也不会被切到。
  _write('icon_foreground.png', _art(transparent: true, artRatio: 0.60));

  // 背景层（自适应图标）：纯底色铺满，启动器用它填满整个图标形状。
  _write('icon_background.png', _solid(kBackground));

  // legacy 图标：底色 + 图案一张出。老启动器不做 inset，图案可以画大一点。
  _write('icon.png', _art(transparent: false, artRatio: 0.66));

  stdout.writeln('三张图标源图已生成，接着跑：dart run flutter_launcher_icons');
}

Image _solid(Color color) =>
    Image(width: kOutputSize, height: kOutputSize, numChannels: 4)
      ..clear(color);

/// 画 2×2 的四格：三格实心（游戏里的方块），右下一格是空槽的描边。
///
/// 这就是这个游戏的一句话说明 —— 手上的块要拼进空出来的位置。
Image _art({required bool transparent, required double artRatio}) {
  final image = Image(width: kCanvas, height: kCanvas, numChannels: 4)
    ..clear(transparent ? ColorRgba8(0, 0, 0, 0) : kBackground);

  final art = kCanvas * artRatio;
  final gap = art * 0.085;
  final cell = (art - gap) / 2;
  final radius = (cell * 0.24).round();
  final left = (kCanvas - art) / 2;
  final top = (kCanvas - art) / 2;

  double x(int col) => left + col * (cell + gap);
  double y(int row) => top + row * (cell + gap);

  void solid(int row, int col, Color color) {
    fillRect(
      image,
      x1: x(col).round(),
      y1: y(row).round(),
      x2: (x(col) + cell).round(),
      y2: (y(row) + cell).round(),
      radius: radius,
      color: color,
    );
  }

  solid(0, 0, kTeal);
  solid(0, 1, kBlue);
  solid(1, 0, kViolet);

  // 右下的空槽：先填一层棋盘空格色，再描一圈亮边。
  solid(1, 1, kSlotFill);
  // thickness 跟着画布尺寸走，缩放后仍是同一视觉粗细。
  drawRect(
    image,
    x1: x(1).round(),
    y1: y(1).round(),
    x2: (x(1) + cell).round(),
    y2: (y(1) + cell).round(),
    radius: radius,
    color: kSlotStroke,
    thickness: cell * 0.11,
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
