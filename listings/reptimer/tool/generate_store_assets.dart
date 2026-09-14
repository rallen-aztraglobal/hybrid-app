// 生成 Play Console 要传的商店素材：
//
//   dart run tool/generate_store_assets.dart store/raw/shot-1.png store/raw/shot-2.png
//
// 产出 store/ 下：icon-512.png、feature-graphic.png、screenshot-N.png
//
// ## 三条 Play 的硬规矩（改脚本时别破坏）
//
// 1. **手机截图的长边比例不能超过 2:1。** 实机是 1080×2400（9:20），比 2:1 还瘦，
//    直接传会被拒。这里左右各补 60px 的 App 背景色补到 1200 宽（=1200×2400=1:2）。
//    **不要改成裁剪** —— 计时页的大数字本来就占满宽度，裁边会切掉数字。
//
// 2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布按 4 通道建
//    （超采样需要），落盘前 convert(numChannels: 3) 转成 24 位。
//
// 3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
//    只会打架；而且 package:image 只有位图字体，1024 宽下排出来的字又糊又不对齐。
import 'dart:io';

import 'package:image/image.dart';

const int kPhoneWidth = 1200;
final ColorRgb8 kBackground = ColorRgb8(0x11, 0x15, 0x1A);
final ColorRgba8 kBackgroundA = ColorRgba8(0x11, 0x15, 0x1A, 0xFF);
final ColorRgba8 kWork = ColorRgba8(0x2D, 0xD4, 0xBF, 0xFF);
final ColorRgba8 kRest = ColorRgba8(0x60, 0xA5, 0xFA, 0xFF);
final ColorRgba8 kPrepare = ColorRgba8(0xF5, 0x9E, 0x0B, 0xFF);
final ColorRgba8 kIdle = ColorRgba8(0x2E, 0x37, 0x41, 0xFF);

void main(List<String> args) {
  Directory('store').createSync(recursive: true);

  final icon = decodePng(File('icon.png').readAsBytesSync());
  if (icon == null) {
    stderr.writeln('读不到 icon.png，先跑 dart run tool/generate_icons.dart');
    exit(1);
  }
  File('store/icon-512.png').writeAsBytesSync(
    encodePng(copyResize(icon,
        width: 512, height: 512, interpolation: Interpolation.average)),
  );
  stdout.writeln('  store/icon-512.png  512x512');

  File('store/feature-graphic.png')
      .writeAsBytesSync(encodePng(_featureGraphic()));
  stdout.writeln('  store/feature-graphic.png  1024x500（24 位，无 alpha）');

  for (var i = 0; i < args.length; i++) {
    final src = decodePng(File(args[i]).readAsBytesSync());
    if (src == null) {
      stderr.writeln('读不到 ${args[i]}');
      continue;
    }
    final padded = _padToWidth(src, kPhoneWidth);
    final out = 'store/screenshot-${i + 1}.png';
    File(out).writeAsBytesSync(encodePng(padded));
    stdout.writeln('  $out  ${padded.width}x${padded.height}'
        '（原 ${src.width}x${src.height}）');
  }
}

Image _padToWidth(Image src, int width) {
  if (src.width >= width) return src;
  final out = Image(width: width, height: src.height, numChannels: 3)
    ..clear(kBackground);
  compositeImage(out, src, dstX: (width - src.width) ~/ 2, dstY: 0);
  return out;
}

/// 特色图：把一次训练摊平成一条时间轴 —— 准备、然后做/休交替若干组。
///
/// 图标是「一圈分段的环」，特色图把同一件事拉直成一条带子：
/// 图标说「这是什么」，特色图说「一次训练长什么样」。
/// 每段的宽度按真实秒数成比例（10 准备 / 20 做 / 10 休），所以「做比休长」
/// 这个信息是看得出来的，不是示意。
///
/// 不放文字（理由见文件头）。
Image _featureGraphic() {
  const w = 1024, h = 500;
  const ss = 2;
  final img = Image(width: w * ss, height: h * ss, numChannels: 4)
    ..clear(kBackgroundA);

  // 一次 Tabata：准备 10，然后 (做 20 / 休 10) × 6 组，最后一组不留休息。
  final segments = <(Color, int)>[
    (kPrepare, 10),
    for (var i = 0; i < 6; i++) ...<(Color, int)>[
      (kWork, 20),
      if (i < 5) (kRest, 10),
    ],
  ];
  final totalSeconds = segments.fold<int>(0, (s, e) => s + e.$2);

  final barH = h * ss * 0.34;
  final barTop = (h * ss - barH) / 2;
  final marginX = w * ss * 0.06;
  final barW = w * ss - marginX * 2;
  final gap = barH * 0.10;

  var x = marginX;
  for (final (color, seconds) in segments) {
    final segW = barW * seconds / totalSeconds;
    final drawW = segW - gap;
    // 圆角半径要同时受**段高**和**段宽**限制。
    // 第一版只按 barH 取（0.28×高），结果窄段（准备 10 秒、休息 10 秒）的宽度
    // 比这个半径还小，四个角一削就成了月牙，完全不像时间轴上的一段。
    final radius = (<double>[barH * 0.20, drawW * 0.32]
            .reduce((a, b) => a < b ? a : b))
        .round();
    fillRect(
      img,
      x1: x.round(),
      y1: barTop.round(),
      x2: (x + drawW).round(),
      y2: (barTop + barH).round(),
      radius: radius,
      color: color,
    );
    x += segW;
  }

  // 时间轴下方一排小圆点，对应组数 —— 已完成的亮、未完成的暗。
  // 这一排让画面不至于只有一条孤零零的带子，也补上「一共几组」这个信息。
  final dotY = barTop + barH + barH * 0.55;
  final dotR = (barH * 0.09).round();
  const dotCount = 6;
  final dotGap = barW / (dotCount + 1);
  for (var i = 0; i < dotCount; i++) {
    fillCircle(
      img,
      x: (marginX + dotGap * (i + 1)).round(),
      y: dotY.round(),
      radius: dotR,
      color: i < 2 ? kWork : kIdle,
    );
  }

  final resized = copyResize(img,
      width: w, height: h, interpolation: Interpolation.average);
  return resized.convert(numChannels: 3);
}
