// 生成 DaySpan 的三张图标源图：
//
//   dart run tool/generate_icons.dart
//   dart run flutter_launcher_icons
//
// 图案是**两个日期之间的一段跨距** —— 左右两个方块代表两个日期，
// 中间一条带端点的横杠代表「相隔多少」。这正是这个 App 做的唯一一件事。
//
// 为什么不画日历图标：日历是这一类 App 的通用符号，Play 上一搜一大把，
// 而且它表达的是「看日期」，不是「算日期之间的距离」。两个方块加一条跨距
// 才说得出这个包的重点。
//
// 抗锯齿：package:image 不给圆角矩形做抗锯齿，故先按 4 倍尺寸画再均值缩回。
import 'dart:io';

import 'package:image/image.dart';

const int kOutputSize = 1024;
const int kSupersample = 4;
const int kCanvas = kOutputSize * kSupersample;

// 与 lib/theme/app_colors.dart 一一对应。
final ColorRgba8 kBackground = ColorRgba8(0xF6, 0xF5, 0xF2, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0xB4, 0x53, 0x09, 0xFF); // 暖橙：跨距
final ColorRgba8 kMark = ColorRgba8(0x29, 0x30, 0x3A, 0xFF); // 深灰：起止那两天
final ColorRgba8 kIdle = ColorRgba8(0xDD, 0xD8, 0xCE, 0xFF); // 浅灰：没数到的天

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
  // 3×3 的「日子格」，其中连续的一段被数进来：
  // 起止两天用深色，中间被数进去的天用主色，没数到的天留浅灰。
  //
  // 前两版都被推翻过，记下来免得再走一遍：
  //   v1「两个大方块夹一根横杠」→ 活像一只哑铃，而这批里真正的健身 App 是 RepTimer；
  //   v2「一排五个竖条」→ 像条形码/均衡器，而且横向一条带子在方形图标里上下留白太多。
  // 排成方阵之后既填满画面，也真的读得出「在日子里数出一段」。
  const n = 3;
  const gapRatio = 0.20;
  final unit = art / (n + (n - 1) * gapRatio);
  final gap = unit * gapRatio;
  final radius = (unit * 0.24).round();

  // 按阅读顺序，第 0..5 格是被数进来的那一段。
  const spanStart = 0;
  const spanEnd = 5;

  for (var i = 0; i < n * n; i++) {
    final r = i ~/ n, c = i % n;
    final x1 = left + c * (unit + gap);
    final y1 = top + r * (unit + gap);
    final Color color;
    if (i == spanStart || i == spanEnd) {
      color = kMark; // 起止那两天
    } else if (i > spanStart && i < spanEnd) {
      color = kAccent; // 数进去的天
    } else {
      color = kIdle; // 没数到的天
    }
    fillRect(
      image,
      x1: x1.round(),
      y1: y1.round(),
      x2: (x1 + unit).round(),
      y2: (y1 + unit).round(),
      radius: radius,
      color: color,
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
