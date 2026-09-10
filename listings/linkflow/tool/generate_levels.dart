// 题库生成器。**开发期工具，不进 App**（不在 lib/ 下，不会被打进包）。
//
//   cd listings/linkflow && dart run tool/generate_levels.dart
//
// 直接覆写 lib/logic/puzzle_library.dart。固定随机种子 ⇒ 同一份代码跑出同一套题，
// 改了参数才会变，方便 review diff。
//
// ## 怎么造题
//
// 1. **先造一条走遍全盘的蛇**（哈密顿路径），再把它随机剪成 k 段。
//    这样每一段天然是一条合法通路，段与段不重叠，合起来正好铺满 —— 不可能造出坏题。
//    早先手写谜题时反复出现「盖不满 / 盖重了」的错，这个办法从根上消掉了。
//
// 2. 蛇的随机化用 **backbite**：从蛇尾出发随机选一个邻格 q，把 q 之后的那一截翻转。
//    翻完仍是一条哈密顿路径。跑几千步就足够乱。比「回溯搜一条随机哈密顿路径」
//    简单得多，也不会在大盘上卡住。
//
// 3. 剪完用 [FlowSolver] 数解，**只留唯一解的题**。多解题玩起来会让人觉得
//    「我明明连上了怎么不算」或者「随便乱连也能过」，是这类游戏最伤口碑的地方。
//
// 4. 难度用求解器的搜索节点数当代理量：需要试错越多的题越难。
//    每档在自己的候选池里按节点数排序，取一段分位区间，档内也是由易到难。
import 'dart:io';
import 'dart:math';

import 'package:linkflow/logic/board.dart';
import 'package:linkflow/logic/puzzle.dart';
import 'package:linkflow/logic/solver.dart';

/// 一个难度档的生成参数。
class TierSpec {
  const TierSpec({
    required this.id,
    required this.label,
    required this.side,
    required this.colorChoices,
    required this.count,
    required this.poolSize,
    required this.loPercentile,
    required this.hiPercentile,
  });

  final String id;
  final String label;
  final int side;

  /// 这一档允许的通路条数（随机取一个）。
  final List<int> colorChoices;

  /// 最终出厂几道题。
  final int count;

  /// 候选池要攒多大，再从里面按难度分位挑。
  final int poolSize;

  /// 从候选池（按难度升序）的哪一段里挑。
  final double loPercentile;
  final double hiPercentile;
}

/// 难度爬升靠两个量一起推：盘面变大 + 通路变多。
///
/// 通路数不能只当「更花」看 —— 它同时决定了题好不好造：
/// 通路越少，每条越长、走法越自由，随机剪出来的题基本都是多解的，
/// 唯一解的命中率会掉到千分之几。7×7 用 5 条时试 4 万次才攒到 20 道，
/// 加到 7 条就正常了。所以每档的通路数是按「盘子大小」等比往上带的。
///
/// 上限 12 —— [AppColors.pathColors] 只有 12 支颜色，再多同一题里会有两条同色线。
const List<TierSpec> tiers = <TierSpec>[
  // 正常档只留一小撮 —— 上手用，主要内容在后面三档。
  TierSpec(
    id: 'normal',
    label: 'Normal',
    side: 5,
    colorChoices: <int>[4, 5],
    count: 8,
    poolSize: 200,
    loPercentile: 0.10,
    hiPercentile: 0.55,
  ),
  TierSpec(
    id: 'hard',
    label: 'Hard',
    side: 7,
    colorChoices: <int>[6, 7],
    count: 14,
    poolSize: 200,
    loPercentile: 0.40,
    hiPercentile: 1.00,
  ),
  TierSpec(
    id: 'expert',
    label: 'Expert',
    side: 8,
    colorChoices: <int>[9, 10],
    count: 14,
    poolSize: 140,
    loPercentile: 0.45,
    hiPercentile: 1.00,
  ),
  TierSpec(
    id: 'master',
    label: 'Master',
    side: 9,
    // 9×9 是唯一解最难碰的一档：36 万刀只攒到 19 道，出厂 10 关（别的档 14 关）。
    //
    // 试过收成「只用 12 条」——想着线多盘面紧、命中率该更高。结果反着来：
    // 24 万刀只攒到 7 道，而且最难的一道才 697 节点（11~12 条混着时是 6644）。
    // 线一多，单条就短，短线的两个端点更容易挨在一起、被前置过滤器毙掉，
    // 剩下能过的又都是被夹死的送分题。所以维持 11~12 混着。
    colorChoices: <int>[11, 12],
    count: 14,
    poolSize: 120,
    loPercentile: 0.50,
    hiPercentile: 1.00,
  ),
];

const String keys = 'abcdefghijkl';

/// 一条随机蛇上试几种剪法。
///
/// 造蛇比数解贵得多（backbite 要跑 n×30 步，每步 O(n)），而同一条蛇换个剪法
/// 就是一道全新的题。一条蛇多剪几刀，吞吐能翻几十倍。
const int cutsPerSnake = 48;

void main() {
  final rng = Random(20260909);
  final buffer = StringBuffer();

  buffer.writeln(_header);

  for (final tier in tiers) {
    stderr.writeln('=== ${tier.label} (${tier.side}×${tier.side}) 攒候选池…');
    final pool = _buildPool(tier, rng);
    pool.sort((a, b) => a.nodes.compareTo(b.nodes));
    stderr.writeln(
      '    池子 ${pool.length} 道，节点数 ${pool.first.nodes} … ${pool.last.nodes}',
    );

    final picked = _pick(pool, tier);
    stderr.writeln(
      '    出厂 ${picked.length} 道，节点数 ${picked.first.nodes} … ${picked.last.nodes}',
    );

    buffer.writeln(_emitTier(tier, picked));
  }

  buffer.write(_footer);

  File('lib/logic/puzzle_library.dart').writeAsStringSync(buffer.toString());
  stderr.writeln('\n已写入 lib/logic/puzzle_library.dart');
}

/// 一道候选题。
class Candidate {
  Candidate(this.side, this.paths, this.nodes, this.colors);

  final int side;
  final List<FlowPath> paths;
  final int nodes;
  final int colors;
}

List<Candidate> _buildPool(TierSpec tier, Random rng) {
  final pool = <Candidate>[];
  final seen = <String>{};
  var attempts = 0;
  final maxAttempts = tier.poolSize * 3000;

  while (pool.length < tier.poolSize && attempts < maxAttempts) {
    final snake = _randomHamiltonian(tier.side, rng);
    for (var c = 0; c < cutsPerSnake && pool.length < tier.poolSize; c++) {
      attempts++;
      final candidate = _tryCut(tier, snake, rng);
      if (candidate == null) continue;
      final sig = candidate.paths
          .map((p) => '${p.start.row},${p.start.col}:${p.moves}')
          .join('|');
      if (!seen.add(sig)) continue;
      pool.add(candidate);
      if (pool.length % 20 == 0) {
        stderr.writeln('    …${pool.length}/${tier.poolSize}（试了 $attempts 刀）');
      }
    }
  }

  if (pool.length < tier.poolSize) {
    stderr.writeln('    ！只攒到 ${pool.length} 道就用完了 $maxAttempts 次尝试');
  }
  return pool;
}

/// 在一条现成的蛇上剪一刀，看能不能剪出一道唯一解的题。
Candidate? _tryCut(TierSpec tier, List<int> snake, Random rng) {
  final side = tier.side;
  final n = side * side;
  final colors = tier.colorChoices[rng.nextInt(tier.colorChoices.length)];

  const minLen = 3; // 两格一条太送分，端点挨着直接连上就完了
  final maxLen = (n / colors * 2.2).round().clamp(minLen + 1, n);
  final lens = _cutLengths(n, colors, minLen, maxLen, rng);
  if (lens == null) return null;

  final paths = <FlowPath>[];
  var offset = 0;
  for (var i = 0; i < colors; i++) {
    final seg = snake.sublist(offset, offset + lens[i]);
    offset += lens[i];
    // 两个端点挨在一起的通路一票否决。这种线可以直接缩成两格，
    // 空出来的格子让别的线随便分 —— 几乎必然是多解。
    // 放在数解之前挡掉，因为它便宜得多，而这正是多解的主要来源。
    final a = seg.first;
    final b = seg.last;
    final dist = (a ~/ side - b ~/ side).abs() + (a % side - b % side).abs();
    if (dist <= 1) return null;
    paths.add(_toPath(keys[i], seg, side));
  }

  final puzzle = FlowPuzzle(side, paths);
  if (puzzle.validate().isNotEmpty) return null; // 理论上不会发生，兜底

  final result = FlowSolver(puzzle).run();
  if (!result.isUnique) return null;

  return Candidate(side, paths, result.nodes, colors);
}

/// 按难度分位从候选池里挑 [TierSpec.count] 道，档内由易到难。
List<Candidate> _pick(List<Candidate> pool, TierSpec tier) {
  final lo = (pool.length * tier.loPercentile).floor();
  final hi = (pool.length * tier.hiPercentile).ceil().clamp(lo + 1, pool.length);
  final slice = pool.sublist(lo, hi);
  if (slice.length <= tier.count) return slice;

  final out = <Candidate>[];
  for (var i = 0; i < tier.count; i++) {
    // 在区间里均匀取点，让档内难度是一条平滑的坡，而不是挤在一头
    final idx = (i * (slice.length - 1) / (tier.count - 1)).round();
    out.add(slice[idx]);
  }
  return out;
}

// ---------------------------------------------------------------- 哈密顿路径

List<int> _randomHamiltonian(int side, Random rng) {
  final n = side * side;

  // 先来一条蛇形铺满的平凡路径当起点
  final path = <int>[];
  for (var r = 0; r < side; r++) {
    if (r.isEven) {
      for (var c = 0; c < side; c++) {
        path.add(r * side + c);
      }
    } else {
      for (var c = side - 1; c >= 0; c--) {
        path.add(r * side + c);
      }
    }
  }

  final pos = List<int>.filled(n, 0);
  for (var i = 0; i < n; i++) {
    pos[path[i]] = i;
  }

  // backbite：从尾巴随机咬一口。翻转后仍是哈密顿路径。
  // 一半的步数先把整条翻过来，好让头尾两端都被随机化。
  final steps = n * 30;
  for (var s = 0; s < steps; s++) {
    if (rng.nextInt(2) == 0) {
      _reverse(path, 0, n - 1);
      for (var i = 0; i < n; i++) {
        pos[path[i]] = i;
      }
    }
    final tail = path[n - 1];
    final nbrs = _neighbours(tail, side);
    final q = nbrs[rng.nextInt(nbrs.length)];
    final i = pos[q];
    if (i >= n - 2) continue; // q 就是倒数第二格，翻了等于没动
    _reverse(path, i + 1, n - 1);
    for (var k = i + 1; k < n; k++) {
      pos[path[k]] = k;
    }
  }

  return path;
}

void _reverse(List<int> list, int a, int b) {
  while (a < b) {
    final t = list[a];
    list[a] = list[b];
    list[b] = t;
    a++;
    b--;
  }
}

List<int> _neighbours(int idx, int side) {
  final r = idx ~/ side;
  final c = idx % side;
  return <int>[
    if (r > 0) idx - side,
    if (r < side - 1) idx + side,
    if (c > 0) idx - 1,
    if (c < side - 1) idx + 1,
  ];
}

/// 把 [n] 格随机切成 [k] 段，每段长度在 [minLen, maxLen] 之间。
List<int>? _cutLengths(int n, int k, int minLen, int maxLen, Random rng) {
  if (k * minLen > n || k * maxLen < n) return null;
  final lens = List<int>.filled(k, minLen);
  var extra = n - k * minLen;
  var guard = 0;
  while (extra > 0) {
    if (++guard > 100000) return null;
    final i = rng.nextInt(k);
    if (lens[i] >= maxLen) continue;
    lens[i]++;
    extra--;
  }
  return lens;
}

FlowPath _toPath(String key, List<int> seg, int side) {
  final buf = StringBuffer();
  for (var i = 1; i < seg.length; i++) {
    final a = seg[i - 1];
    final b = seg[i];
    final dr = b ~/ side - a ~/ side;
    final dc = b % side - a % side;
    buf.write(
      dr == 1
          ? 'D'
          : dr == -1
              ? 'U'
              : dc == 1
                  ? 'R'
                  : 'L',
    );
  }
  return FlowPath(
    key,
    start: Cell(seg.first ~/ side, seg.first % side),
    moves: buf.toString(),
  );
}

// ------------------------------------------------------------------ 代码生成

String _emitTier(TierSpec tier, List<Candidate> picked) {
  final b = StringBuffer();
  b.writeln('/// ${tier.label} 档题库（${tier.side}×${tier.side}），由易到难。');
  b.writeln('const List<LevelDef> _${tier.id} = <LevelDef>[');
  for (final c in picked) {
    b.writeln('  // ${c.colors} 色 · 求解节点 ${c.nodes}');
    b.writeln('  LevelDef(${c.side}, <FlowPath>[');
    for (final p in c.paths) {
      b.writeln(
        "    FlowPath('${p.key}', start: Cell(${p.start.row}, ${p.start.col}), "
        "moves: '${p.moves}'),",
      );
    }
    b.writeln('  ]),');
  }
  b.writeln('];');
  return b.toString();
}

const String _header = '''
// 本文件由 tool/generate_levels.dart 生成，不要手改。
// 重新生成：cd listings/linkflow && dart run tool/generate_levels.dart
//
// 每道题都经过求解器验证：唯一解、铺满全盘。注释里的「求解节点」是求解器
// 走完搜索空间访问的节点数，档内按它由易到难排。

import 'board.dart';
import 'puzzle.dart';

/// 难度档。盘面越大、通路越多，试错空间越大。
enum Difficulty {
  normal('Normal'),
  hard('Hard'),
  expert('Expert'),
  master('Master');

  const Difficulty(this.label);

  final String label;
}

/// 一道题的定义：尺寸 + 若干条通路。
class LevelDef {
  const LevelDef(this.side, this.paths);

  final int side;
  final List<FlowPath> paths;

  FlowPuzzle build() => FlowPuzzle(side, paths);
}

// 走法串：U 上 / D 下 / L 左 / R 右。
''';

const String _footer = '''
List<LevelDef> levelsFor(Difficulty difficulty) => switch (difficulty) {
      Difficulty.normal => _normal,
      Difficulty.hard => _hard,
      Difficulty.expert => _expert,
      Difficulty.master => _master,
    };
''';
