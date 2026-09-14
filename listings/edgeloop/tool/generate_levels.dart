// 离线生成关卡库，产出 lib/logic/puzzle_library.dart。
//
//     dart run tool/generate_levels.dart
//
// 为什么预生成而不是运行时生成：确认唯一解要穷举求解，Master 档单题最多要展开百万级
// 节点（实测数据见 lib/logic/difficulty.dart 的注释）。放到手机上现算，开局就是几秒白屏；而且每台设备
// 算出来的题还不一样，用户之间无法对照。
//
// 固定随机种子 —— 同一份代码必须产出同一份关卡库，否则每次重跑都会把所有人的进度
// 对应到不同的题上。
import 'dart:io';
import 'dart:math';

import 'package:edgeloop8451/logic/difficulty.dart';
import 'package:edgeloop8451/logic/generator.dart';
import 'package:edgeloop8451/logic/solver.dart';

const int _seed = 20260911;

/// 每档出多少关。
const Map<Difficulty, int> _counts = <Difficulty, int>{
  Difficulty.normal: 14,
  Difficulty.hard: 14,
  Difficulty.expert: 12,
  Difficulty.master: 10,
};

void main() {
  final random = Random(_seed);
  final gen = EdgeLoopGenerator(random);
  final out = StringBuffer();

  out.writeln('// 由 tool/generate_levels.dart 生成，请勿手改。');
  out.writeln('// 重新生成：dart run tool/generate_levels.dart');
  out.writeln('//');
  out.writeln('// 每道题都经穷举求解器验证为**唯一解**（见 lib/logic/solver.dart）。');
  out.writeln("import 'difficulty.dart';");
  out.writeln("import 'puzzle.dart';");
  out.writeln();
  out.writeln('/// 各难度的题面（紧凑编码，形如 `5x5:..3.2...`）。');
  out.writeln('const Map<Difficulty, List<String>> _encoded = '
      '<Difficulty, List<String>>{');

  for (final d in Difficulty.values) {
    final want = _counts[d]!;
    final made = <String>[];
    var attempts = 0;
    var maxNodes = 0;
    final sw = Stopwatch()..start();

    while (made.length < want) {
      attempts++;
      if (attempts > want * 60) {
        stderr.writeln('！${d.name} 只生成出 ${made.length}/$want 道就放弃了');
        break;
      }
      final p = gen.generate(
        rows: d.rows,
        cols: d.cols,
        targetCells: d.regionCells,
        maxAttempts: 40,
      );
      if (p == null) continue;

      // 兜底复核：生成器内部已验过唯一性，这里用更高的节点上限再验一次。
      // 上限撞到时 exhausted=false，唯一性结论不可信 —— 宁可丢掉也不能放进关卡库。
      final r = EdgeLoopSolver(p, nodeCap: 20000000).solve();
      if (!r.isUnique) continue;
      if (r.nodes > maxNodes) maxNodes = r.nodes;

      final code = p.encode();
      if (made.contains(code)) continue; // 去重：同一题出两次很扎眼
      made.add(code);
    }
    sw.stop();

    out.writeln('  Difficulty.${d.name}: <String>[');
    for (final code in made) {
      out.writeln("    '$code',");
    }
    out.writeln('  ],');

    stdout.writeln('${d.name.padRight(7)} ${made.length}/$want 道  '
        '尝试 $attempts 次  ${sw.elapsedMilliseconds}ms  最大节点 $maxNodes');
  }

  out.writeln('};');
  out.writeln();
  out.writeln('''
/// 取某难度的全部关卡。
///
/// 每次调用都重新 decode —— 题面是不可变的，但调用方会把它交给可变的 EdgeLoopGame，
/// 共享同一个实例会让「重开一局」意外地影响到别的地方。
List<EdgeLoopPuzzle> levelsFor(Difficulty d) =>
    _encoded[d]!.map(EdgeLoopPuzzle.decode).toList(growable: false);

/// 某难度有多少关。
int levelCountFor(Difficulty d) => _encoded[d]!.length;
''');

  File('lib/logic/puzzle_library.dart').writeAsStringSync(out.toString());
  stdout.writeln('\n已写入 lib/logic/puzzle_library.dart');
}
