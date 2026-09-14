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
//    **不要改成裁剪** —— 裁掉的正是棋盘的最外圈。
//
// 2. **特色图不能带 alpha 通道。** 带透明的 Play 直接拒收。画布按 4 通道建
//    （超采样需要），落盘前 convert(numChannels: 3) 转成 24 位。
//
// 3. **特色图里不放文字。** Play 会在上面叠自己的应用名与安装按钮，图里再写一遍标题
//    只会打架；而且 package:image 只有位图字体，1024 宽下排出来的字又糊又不对齐。
import 'dart:io';

import 'package:image/image.dart';

const int kPhoneWidth = 1200; // 补边后的目标宽度
final ColorRgb8 kBackground = ColorRgb8(0xF4, 0xF1, 0xEA);
final ColorRgba8 kBackgroundA = ColorRgba8(0xF4, 0xF1, 0xEA, 0xFF);
final ColorRgba8 kSurface = ColorRgba8(0xFF, 0xFF, 0xFF, 0xFF);
final ColorRgba8 kLine = ColorRgba8(0x2B, 0x3A, 0x67, 0xFF);
final ColorRgba8 kDot = ColorRgba8(0xBF, 0xB8, 0xAA, 0xFF);

void main(List<String> args) {
  Directory('store').createSync(recursive: true);

  // 商店图标：直接用 512 的 icon.png。
  final icon = decodePng(File('icon.png').readAsBytesSync());
  if (icon == null) {
    stderr.writeln('读不到 icon.png，先跑 dart run tool/generate_icons.dart');
    exit(1);
  }
  File('store/icon-512.png').writeAsBytesSync(
    encodePng(copyResize(icon, width: 512, height: 512,
        interpolation: Interpolation.average)),
  );
  stdout.writeln('  store/icon-512.png  512x512');

  File('store/feature-graphic.png').writeAsBytesSync(
    encodePng(_featureGraphic()),
  );
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

/// 左右补边到目标宽度，补的是 App 背景色 —— 看不出是补的。
Image _padToWidth(Image src, int width) {
  if (src.width >= width) return src;
  final out = Image(width: width, height: src.height, numChannels: 3)
    ..clear(kBackground);
  compositeImage(out, src, dstX: (width - src.width) ~/ 2, dstY: 0);
  return out;
}

/// 特色图：横向铺三个小盘面，各自画着一条已解出的回路。
///
/// 不放文字（理由见文件头）。用三个而不是一个：一个盘面在 1024×500 里要么太小、
/// 要么撑得很空；三个横排既填满画面，也顺带说明「有很多关」。
Image _featureGraphic() {
  const w = 1024, h = 500;
  const ss = 2; // 超采样倍数
  final img = Image(width: w * ss, height: h * ss, numChannels: 4)
    ..clear(kBackgroundA);

  // 三个盘面，尺寸略有差异，中间那个最大 —— 全等大会显得像占位图。
  // [中心x比例, 中心y比例, 边长比例(按高), 格数]
  //
  // 尺寸和间距是算过的，不是凑的：白底板比格点阵每侧外扩 0.36 格，
  // 第一版取 0.62/0.78/0.62 + 中心 0.17/0.50/0.83，算下来左右两块的底板
  // 既盖住中间那块、又被画布左右边缘切掉一角。
  // 现在这组：中间底板占 329..695，两侧分别 6..302 与 722..1018 ——
  // 彼此留 27px 间隙，距画布边缘 6px。
  const boards = <List<num>>[
    [0.15, 0.50, 0.50, 4],
    [0.50, 0.50, 0.64, 5],
    [0.85, 0.50, 0.50, 4],
  ];
  // 每个盘面上画的回路（顶点为格点坐标，闭合）。
  const paths = <List<List<int>>>[
    [
      [0, 1], [1, 1], [1, 0], [3, 0], [3, 2], [2, 2], [2, 3], [0, 3],
    ],
    [
      [0, 1], [1, 1], [1, 0], [4, 0], [4, 2], [3, 2], [3, 4], [1, 4],
      [1, 3], [0, 3],
    ],
    [
      [1, 0], [4, 0], [4, 2], [3, 2], [3, 3], [1, 3], [1, 2], [0, 2],
      [0, 1], [1, 1],
    ],
  ];

  for (var b = 0; b < boards.length; b++) {
    final cx = w * ss * boards[b][0];
    final cy = h * ss * boards[b][1];
    final side = h * ss * boards[b][2];
    final grid = boards[b][3].toInt();
    final cell = side / grid;
    final left = cx - side / 2;
    final top = cy - side / 2;

    // 白底板
    final padPx = cell * 0.36;
    fillRect(
      img,
      x1: (left - padPx).round(),
      y1: (top - padPx).round(),
      x2: (left + side + padPx).round(),
      y2: (top + side + padPx).round(),
      radius: (cell * 0.24).round(),
      color: kSurface,
    );

    // 格点
    for (var r = 0; r <= grid; r++) {
      for (var c = 0; c <= grid; c++) {
        fillCircle(
          img,
          x: (left + c * cell).round(),
          y: (top + r * cell).round(),
          radius: (cell * 0.055).round(),
          color: kDot,
        );
      }
    }

    // 回路
    final stroke = cell * 0.11;
    final path = paths[b];
    for (var i = 0; i < path.length; i++) {
      final p1 = path[i], p2 = path[(i + 1) % path.length];
      drawLine(
        img,
        x1: (left + p1[0] * cell).round(),
        y1: (top + p1[1] * cell).round(),
        x2: (left + p2[0] * cell).round(),
        y2: (top + p2[1] * cell).round(),
        color: kLine,
        thickness: stroke,
      );
    }
    // 拐角补圆：drawLine 是方头，直角处会缺一块。
    for (final v in path) {
      fillCircle(
        img,
        x: (left + v[0] * cell).round(),
        y: (top + v[1] * cell).round(),
        radius: (stroke / 2).round(),
        color: kLine,
      );
    }
  }

  final resized = copyResize(img,
      width: w, height: h, interpolation: Interpolation.average);
  // Play 拒收带 alpha 的特色图。
  return resized.convert(numChannels: 3);
}
