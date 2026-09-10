import 'dart:math';

import 'nonogram.dart';
import 'picture_library.dart';

/// 谜题尺寸档位。只做两档 —— 5×5 上手、8×8 进阶。
///
/// **为什么上限是 8 而不是 10。** 竖屏手机的棋盘是正方形、被宽度卡死：
/// 10 列时格子只剩约 30dp，而线索最多能到 5 段、要挤进一条窄槽，
/// 结果是格子小、数字小、排得密，看一眼就累，更别说点准。
/// 8 列的格子大 25%、线索最多 4 段、槽也窄了，棋盘反而更大。
enum PuzzleSize {
  small(5, '5 × 5'),
  large(8, '8 × 8');

  const PuzzleSize(this.side, this.label);

  final int side;
  final String label;
}

/// 发一道题：从图库里随机取一张画。
///
/// **为什么不是随机生成。** 早先是按密度随机铺格子再验收「纯逻辑可解」，
/// 题目本身没问题，但解开之后只是一片没有意义的杂色 —— 数织的回报在于
/// 浮现出一幅画，没有画就只剩数字作业。所以答案改为手绘图库，
/// 每一张都由 `picture_library_test.dart` 保证可解。
///
/// [avoid] 用于「下一题」：避免紧接着又发到同一张，连着两次一样很出戏。
/// 图库只有一张时忽略该参数。
Picture dealPicture(PuzzleSize size, {Random? random, Picture? avoid}) {
  final pool = picturesFor(size.side);
  if (pool.isEmpty) {
    throw StateError('图库里没有 ${size.label} 的图案');
  }
  if (pool.length == 1) return pool.first;

  final rng = random ?? Random();
  Picture picked;
  do {
    picked = pool[rng.nextInt(pool.length)];
  } while (avoid != null && identical(picked, avoid));
  return picked;
}

/// 发一道题并直接构造成谜题。界面通常用这个。
Nonogram dealPuzzle(PuzzleSize size, {Random? random}) =>
    dealPicture(size, random: random).toNonogram();
