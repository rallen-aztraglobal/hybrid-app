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
//    **不要改成裁剪** —— 裁掉的是日期行两端的星期标记。
//
// 2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布按 4 通道建
//    （超采样需要），落盘前 convert(numChannels: 3) 转成 24 位。
//
// 3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
//    只会打架；而且 package:image 只有位图字体，1024 宽下排出来的字又糊又不对齐。
import 'dart:io';

import 'package:image/image.dart';

const int kPhoneWidth = 1200;
final ColorRgb8 kBackground = ColorRgb8(0xF6, 0xF5, 0xF2);
final ColorRgba8 kBackgroundA = ColorRgba8(0xF6, 0xF5, 0xF2, 0xFF);
final ColorRgba8 kAccent = ColorRgba8(0xB4, 0x53, 0x09, 0xFF);
final ColorRgba8 kMark = ColorRgba8(0x29, 0x30, 0x3A, 0xFF);
final ColorRgba8 kIdle = ColorRgba8(0xDD, 0xD8, 0xCE, 0xFF);

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

/// 特色图：一整月的日子铺成网格，其中连续的一段被数了出来。
///
/// 与图标同一个语汇（起止两天用深色、数进去的用主色、其余留浅灰），
/// 只是铺成一个月的尺度 —— 图标说「这是什么」，特色图说「拿来干什么」。
///
/// 不放文字（理由见文件头）。
Image _featureGraphic() {
  const w = 1024, h = 500;
  const ss = 2;
  final img = Image(width: w * ss, height: h * ss, numChannels: 4)
    ..clear(kBackgroundA);

  // 7 列 × 3 行 = 三周。
  //
  // 列数取 7（一周）是为了读得出「按周排」；行数取 3 而不是一个月的 5，
  // 是被画布比例逼的：1024×500 是 2.05:1，而 7×5 的网格只有 1.4:1，
  // 按高度撑满后宽度只占到画布的 40%，整块图缩在中间显得很小。
  // 7×3 是 2.33:1，与画布接近，按**宽度**定格子大小就能填满。
  const cols = 7, rows = 3;
  final cellH = w * ss * 0.86 / cols;
  final gap = cellH * 0.22;
  final unit = cellH - gap;
  final gridW = cols * cellH - gap;
  final gridH = rows * cellH - gap;
  final left = (w * ss - gridW) / 2;
  final top = (h * ss - gridH) / 2;
  final radius = (unit * 0.26).round();

  // 被数出来的那一段：第 4 格到第 17 格 —— 跨了两个周界，看得出「跨周」。
  const spanStart = 4, spanEnd = 17;

  for (var i = 0; i < cols * rows; i++) {
    final r = i ~/ cols, c = i % cols;
    final x1 = left + c * cellH;
    final y1 = top + r * cellH;
    final Color color;
    if (i == spanStart || i == spanEnd) {
      color = kMark;
    } else if (i > spanStart && i < spanEnd) {
      color = kAccent;
    } else {
      color = kIdle;
    }
    fillRect(
      img,
      x1: x1.round(),
      y1: y1.round(),
      x2: (x1 + unit).round(),
      y2: (y1 + unit).round(),
      radius: radius,
      color: color,
    );
  }

  final resized = copyResize(img,
      width: w, height: h, interpolation: Interpolation.average);
  return resized.convert(numChannels: 3);
}
