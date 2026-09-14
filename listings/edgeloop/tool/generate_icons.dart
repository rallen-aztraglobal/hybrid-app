// 生成 EdgeLoop 的三张图标源图：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**一条闭合的回路走在格点之间** —— 这就是这个游戏的一句话说明。
// 特意画成有拐折的不规则环，而不是一个矩形：矩形看着像边框，不像「玩家画出来的路径」。
//
// 只画回路和格点，不画提示数字：48dp 下数字会糊成一个小点，纯属噪声。
//
// 抗锯齿：package:image 不给线段和圆角做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 lib/theme/app_colors.dart 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF4, 0xF1, 0xEA, 0xFF);
final ColorRgba8 kLine = ColorRgba8(0x2B, 0x3A, 0x67, 0xFF); // 墨蓝：回路
final ColorRgba8 kDot = ColorRgba8(0xBF, 0xB8, 0xAA, 0xFF); // 灰：格点

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

  // 4x4 的格点阵（5x5 个点），回路走在点与点之间。
  const grid = 4;
  double px(int c) => left + art * c / grid;
  double py(int r) => top + art * r / grid;

  final dotRadius = art * 0.030;
  final stroke = art * 0.085;

  // 先铺格点（在回路下面）。
  for (var r = 0; r <= grid; r++) {
    for (var c = 0; c <= grid; c++) {
      fillCircle(
        image,
        x: px(c).round(),
        y: py(r).round(),
        radius: dotRadius.round(),
        color: kDot,
      );
    }
  }

  // 回路：一个**不对称**的闭环，顺序首尾相接，每个顶点是 (列, 行)。
  //
  // 不对称是有意的。第一版用的是上下左右都对称的顶点序列，画出来正好是一个加号 ——
  // 看着像医疗十字，完全不像「玩家画出来的一条路径」。回路必须歪一点、
  // 凹口的位置必须各不相同，才读得出是路径而不是符号。
  const path = <List<int>>[
    [0, 1], [1, 1], [1, 0], [4, 0], [4, 2], [3, 2],
    [3, 4], [1, 4], [1, 3], [0, 3],
  ];
  for (var i = 0; i < path.length; i++) {
    final a = path[i];
    final b = path[(i + 1) % path.length];
    drawLine(
      image,
      x1: px(a[0]).round(),
      y1: py(a[1]).round(),
      x2: px(b[0]).round(),
      y2: py(b[1]).round(),
      color: kLine,
      thickness: stroke,
    );
  }
  // 拐角补圆：drawLine 是方头，直角处会缺一块。
  // 这与棋盘上 StrokeCap.round 的观感一致。
  for (final v in path) {
    fillCircle(
      image,
      x: px(v[0]).round(),
      y: py(v[1]).round(),
      radius: (stroke / 2).round(),
      color: kLine,
    );
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
  stdout.writeln('  $name  ${image.width}x${image.height}');
}
