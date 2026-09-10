// 生成 NonoPix 的三张图标源图（仓库根目录下的 icon.png / icon_foreground.png /
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
//
// 图案选的是**用格子拼出来的一颗心**：数织的全部意义就是「一格格涂出一幅图」，
// 画一幅现成的像素图比画一个空棋盘更说明问题。不画网格线 —— 48dp 下线会糊成一片，
// 留出格缝反而更干净。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 AppColors 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF2, 0xEF, 0xE9, 0xFF);
final ColorRgba8 kFilled = ColorRgba8(0x4F, 0x46, 0xE5, 0xFF);
final ColorRgba8 kFilledTop = ColorRgba8(0x6C, 0x63, 0xFF, 0xFF);

/// 5×5 的像素心。`#` 是涂黑的格子。
const List<String> kPicture = <String>[
  '.#.#.',
  '#####',
  '#####',
  '.###.',
  '..#..',
];

void main() {
  // 前景层（自适应图标）：启动器会再套一层 16% 的 inset 并按各家形状裁切，
  // 故图案只占画布的 60%，四角留够余量，圆形遮罩下也不会被切到。
  _write('icon_foreground.png', _art(transparent: true, artRatio: 0.60));

  // 背景层（自适应图标）：纯底色铺满，启动器用它填满整个图标形状。
  _write('icon_background.png', _solid(kBackground));

  // legacy 图标：底色 + 图案一张出。老启动器不做 inset，图案可以画大一点。
  _write('icon.png', _art(transparent: false, artRatio: 0.68));

  stdout.writeln('三张图标源图已生成，接着跑：dart run flutter_launcher_icons');
}

Image _solid(Color color) =>
    Image(width: kOutputSize, height: kOutputSize, numChannels: 4)
      ..clear(color);

Image _art({required bool transparent, required double artRatio}) {
  final image = Image(width: kCanvas, height: kCanvas, numChannels: 4)
    ..clear(transparent ? ColorRgba8(0, 0, 0, 0) : kBackground);

  final side = kPicture.length;
  final art = kCanvas * artRatio;
  final gap = art * 0.045;
  final cell = (art - gap * (side - 1)) / side;
  final radius = (cell * 0.22).round();
  final left = (kCanvas - art) / 2;
  final top = (kCanvas - art) / 2;

  for (var r = 0; r < side; r++) {
    for (var c = 0; c < side; c++) {
      if (kPicture[r][c] != '#') continue;
      // 上半部分用亮一档的紫：整块图案有一点纵向渐变，不至于是一坨死色
      final color = r < 2 ? kFilledTop : kFilled;
      final x = left + c * (cell + gap);
      final y = top + r * (cell + gap);
      fillRect(
        image,
        x1: x.round(),
        y1: y.round(),
        x2: (x + cell).round(),
        y2: (y + cell).round(),
        radius: radius,
        color: color,
      );
    }
  }

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
