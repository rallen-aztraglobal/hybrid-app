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

/// Normal 档题库（5×5），由易到难。
const List<LevelDef> _normal = <LevelDef>[
  // 5 色 · 求解节点 25
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(0, 4), moves: 'LL'),
    FlowPath('b', start: Cell(0, 1), moves: 'DDDR'),
    FlowPath('c', start: Cell(3, 3), moves: 'UL'),
    FlowPath('d', start: Cell(1, 2), moves: 'RRDDDL'),
    FlowPath('e', start: Cell(4, 2), moves: 'LLUUUU'),
  ]),
  // 5 色 · 求解节点 25
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(1, 1), moves: 'DDDL'),
    FlowPath('b', start: Cell(3, 0), moves: 'UUURR'),
    FlowPath('c', start: Cell(1, 2), moves: 'RURDD'),
    FlowPath('d', start: Cell(2, 3), moves: 'DR'),
    FlowPath('e', start: Cell(4, 4), moves: 'LLUU'),
  ]),
  // 5 色 · 求解节点 26
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(2, 4), moves: 'DDLLL'),
    FlowPath('b', start: Cell(4, 0), moves: 'UU'),
    FlowPath('c', start: Cell(1, 0), moves: 'URDD'),
    FlowPath('d', start: Cell(3, 1), moves: 'RUUUR'),
    FlowPath('e', start: Cell(0, 4), moves: 'DLDD'),
  ]),
  // 5 色 · 求解节点 26
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(3, 3), moves: 'RULU'),
    FlowPath('b', start: Cell(1, 4), moves: 'ULLDL'),
    FlowPath('c', start: Cell(0, 1), moves: 'LDDRR'),
    FlowPath('d', start: Cell(3, 2), moves: 'LLD'),
    FlowPath('e', start: Cell(4, 1), moves: 'RRR'),
  ]),
  // 5 色 · 求解节点 27
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(2, 4), moves: 'DD'),
    FlowPath('b', start: Cell(4, 3), moves: 'LLLUU'),
    FlowPath('c', start: Cell(1, 0), moves: 'URDD'),
    FlowPath('d', start: Cell(3, 1), moves: 'RUUUR'),
    FlowPath('e', start: Cell(0, 4), moves: 'DLDD'),
  ]),
  // 5 色 · 求解节点 28
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(2, 4), moves: 'DDLLLLU'),
    FlowPath('b', start: Cell(3, 1), moves: 'RRU'),
    FlowPath('c', start: Cell(1, 3), moves: 'RULL'),
    FlowPath('d', start: Cell(0, 1), moves: 'LDRR'),
    FlowPath('e', start: Cell(2, 2), moves: 'LL'),
  ]),
  // 5 色 · 求解节点 29
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(1, 3), moves: 'DLLD'),
    FlowPath('b', start: Cell(4, 1), moves: 'LUUU'),
    FlowPath('c', start: Cell(0, 0), moves: 'RDR'),
    FlowPath('d', start: Cell(0, 2), moves: 'RRDDD'),
    FlowPath('e', start: Cell(4, 4), moves: 'LLUR'),
  ]),
  // 5 色 · 求解节点 31
  LevelDef(5, <FlowPath>[
    FlowPath('a', start: Cell(1, 1), moves: 'URDRU'),
    FlowPath('b', start: Cell(0, 4), moves: 'DDLD'),
    FlowPath('c', start: Cell(3, 4), moves: 'DLL'),
    FlowPath('d', start: Cell(4, 1), moves: 'LURRU'),
    FlowPath('e', start: Cell(2, 1), moves: 'LUU'),
  ]),
];

/// Hard 档题库（7×7），由易到难。
const List<LevelDef> _hard = <LevelDef>[
  // 7 色 · 求解节点 174
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(4, 4), moves: 'DDLULDLLUR'),
    FlowPath('b', start: Cell(4, 1), moves: 'LUUU'),
    FlowPath('c', start: Cell(0, 0), moves: 'RDDDRD'),
    FlowPath('d', start: Cell(4, 3), moves: 'URUUL'),
    FlowPath('e', start: Cell(2, 3), moves: 'LUURRR'),
    FlowPath('f', start: Cell(0, 6), moves: 'DDDDDDL'),
    FlowPath('g', start: Cell(5, 5), moves: 'UUUU'),
  ]),
  // 7 色 · 求解节点 189
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(5, 3), moves: 'DRRR'),
    FlowPath('b', start: Cell(5, 6), moves: 'LLURUUL'),
    FlowPath('c', start: Cell(3, 4), moves: 'LDLUUR'),
    FlowPath('d', start: Cell(1, 3), moves: 'LLDDDD'),
    FlowPath('e', start: Cell(5, 2), moves: 'DLLUUUU'),
    FlowPath('f', start: Cell(1, 0), moves: 'URRRR'),
    FlowPath('g', start: Cell(1, 4), moves: 'RURDDDD'),
  ]),
  // 6 色 · 求解节点 211
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(5, 3), moves: 'DRRRULLURU'),
    FlowPath('b', start: Cell(2, 5), moves: 'LDLDLUU'),
    FlowPath('c', start: Cell(2, 3), moves: 'ULLDDDDRD'),
    FlowPath('d', start: Cell(6, 1), moves: 'LUUUUU'),
    FlowPath('e', start: Cell(0, 0), moves: 'RRRRDR'),
    FlowPath('f', start: Cell(0, 5), moves: 'RDDDD'),
  ]),
  // 7 色 · 求解节点 228
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(0, 6), moves: 'DDDDDDLULDL'),
    FlowPath('b', start: Cell(5, 3), moves: 'UUULLU'),
    FlowPath('c', start: Cell(1, 2), moves: 'RRDDD'),
    FlowPath('d', start: Cell(4, 5), moves: 'UUUULLLLL'),
    FlowPath('e', start: Cell(1, 0), moves: 'DDD'),
    FlowPath('f', start: Cell(4, 1), moves: 'DLD'),
    FlowPath('g', start: Cell(6, 1), moves: 'RUUUL'),
  ]),
  // 7 色 · 求解节点 245
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(1, 3), moves: 'LLL'),
    FlowPath('b', start: Cell(0, 0), moves: 'RRRRRRD'),
    FlowPath('c', start: Cell(1, 5), moves: 'LDRRDL'),
    FlowPath('d', start: Cell(3, 4), moves: 'LULLLDDDDRRR'),
    FlowPath('e', start: Cell(6, 4), moves: 'RRUUL'),
    FlowPath('f', start: Cell(5, 5), moves: 'LULLU'),
    FlowPath('g', start: Cell(3, 1), moves: 'DDRR'),
  ]),
  // 7 色 · 求解节点 272
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(5, 5), moves: 'ULDLDR'),
    FlowPath('b', start: Cell(6, 5), moves: 'RUUUL'),
    FlowPath('c', start: Cell(3, 4), moves: 'LDLD'),
    FlowPath('d', start: Cell(6, 2), moves: 'LLUUU'),
    FlowPath('e', start: Cell(2, 0), moves: 'UURRRRD'),
    FlowPath('f', start: Cell(1, 5), moves: 'URDDLLLULL'),
    FlowPath('g', start: Cell(2, 1), moves: 'RDLDD'),
  ]),
  // 6 色 · 求解节点 300
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(3, 1), moves: 'DRRRU'),
    FlowPath('b', start: Cell(3, 3), moves: 'LULLDDDDRU'),
    FlowPath('c', start: Cell(5, 2), moves: 'RRRUUUUL'),
    FlowPath('d', start: Cell(2, 4), moves: 'LULLLU'),
    FlowPath('e', start: Cell(0, 1), moves: 'RRRRRDDD'),
    FlowPath('f', start: Cell(4, 6), moves: 'DDLLLL'),
  ]),
  // 7 色 · 求解节点 346
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(1, 1), moves: 'DRRRUR'),
    FlowPath('b', start: Cell(2, 5), moves: 'RUULLL'),
    FlowPath('c', start: Cell(1, 3), moves: 'LULLDD'),
    FlowPath('d', start: Cell(3, 0), moves: 'RRRD'),
    FlowPath('e', start: Cell(4, 4), moves: 'URRDLDRD'),
    FlowPath('f', start: Cell(6, 5), moves: 'LULDLU'),
    FlowPath('g', start: Cell(4, 2), moves: 'LLDDRU'),
  ]),
  // 7 色 · 求解节点 384
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(2, 4), moves: 'LURULLD'),
    FlowPath('b', start: Cell(1, 1), moves: 'ULDDDDDDRU'),
    FlowPath('c', start: Cell(4, 1), moves: 'RRRRD'),
    FlowPath('d', start: Cell(5, 4), moves: 'LLD'),
    FlowPath('e', start: Cell(6, 3), moves: 'RRRU'),
    FlowPath('f', start: Cell(4, 6), moves: 'UUUUL'),
    FlowPath('g', start: Cell(1, 5), moves: 'DDLLLULD'),
  ]),
  // 7 色 · 求解节点 463
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(6, 0), moves: 'URDRUUL'),
    FlowPath('b', start: Cell(3, 1), moves: 'RRDDDRRR'),
    FlowPath('c', start: Cell(5, 6), moves: 'LLUUR'),
    FlowPath('d', start: Cell(4, 5), moves: 'RUUL'),
    FlowPath('e', start: Cell(1, 5), moves: 'RULLL'),
    FlowPath('f', start: Cell(1, 3), moves: 'RDLLLU'),
    FlowPath('g', start: Cell(1, 2), moves: 'ULLDDDD'),
  ]),
  // 6 色 · 求解节点 546
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(3, 1), moves: 'LUUUR'),
    FlowPath('b', start: Cell(0, 2), moves: 'RRRRDDDDD'),
    FlowPath('c', start: Cell(6, 6), moves: 'LUUUUUL'),
    FlowPath('d', start: Cell(2, 4), moves: 'DDDDLLLL'),
    FlowPath('e', start: Cell(5, 0), moves: 'URRUUL'),
    FlowPath('f', start: Cell(1, 1), moves: 'RRDDDDLL'),
  ]),
  // 6 色 · 求解节点 660
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(3, 1), moves: 'LUU'),
    FlowPath('b', start: Cell(0, 0), moves: 'RRRRRRDDD'),
    FlowPath('c', start: Cell(4, 6), moves: 'DDLUUUU'),
    FlowPath('d', start: Cell(1, 5), moves: 'LDDDD'),
    FlowPath('e', start: Cell(6, 4), moves: 'LLLLUURRUU'),
    FlowPath('f', start: Cell(2, 1), moves: 'URRDDDDLL'),
  ]),
  // 6 色 · 求解节点 896
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(4, 4), moves: 'RULLDLLLU'),
    FlowPath('b', start: Cell(3, 1), moves: 'RURRRU'),
    FlowPath('c', start: Cell(1, 4), moves: 'LLLDLU'),
    FlowPath('d', start: Cell(0, 0), moves: 'RRRRRRDDD'),
    FlowPath('e', start: Cell(4, 6), moves: 'DDLULLL'),
    FlowPath('f', start: Cell(5, 1), moves: 'LDRRRR'),
  ]),
  // 7 色 · 求解节点 2761
  LevelDef(7, <FlowPath>[
    FlowPath('a', start: Cell(5, 5), moves: 'RDLLLLL'),
    FlowPath('b', start: Cell(5, 1), moves: 'UURURRDLDL'),
    FlowPath('c', start: Cell(5, 2), moves: 'RRURRUL'),
    FlowPath('d', start: Cell(2, 5), moves: 'RUU'),
    FlowPath('e', start: Cell(0, 5), moves: 'DLUL'),
    FlowPath('f', start: Cell(1, 3), moves: 'LULL'),
    FlowPath('g', start: Cell(1, 0), moves: 'RDLDDDD'),
  ]),
];

/// Expert 档题库（8×8），由易到难。
const List<LevelDef> _expert = <LevelDef>[
  // 10 色 · 求解节点 209
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(2, 7), moves: 'UULLLLL'),
    FlowPath('b', start: Cell(1, 2), moves: 'RRRRD'),
    FlowPath('c', start: Cell(2, 5), moves: 'LLL'),
    FlowPath('d', start: Cell(3, 2), moves: 'RRRRRD'),
    FlowPath('e', start: Cell(5, 7), moves: 'DDLU'),
    FlowPath('f', start: Cell(5, 6), moves: 'ULL'),
    FlowPath('g', start: Cell(4, 3), moves: 'LLUUU'),
    FlowPath('h', start: Cell(0, 1), moves: 'LDDDDDR'),
    FlowPath('i', start: Cell(6, 1), moves: 'LDRRRUR'),
    FlowPath('j', start: Cell(7, 4), moves: 'RUULLLD'),
  ]),
  // 9 色 · 求解节点 228
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(7, 6), moves: 'RULLDLU'),
    FlowPath('b', start: Cell(6, 3), moves: 'DLLLU'),
    FlowPath('c', start: Cell(6, 1), moves: 'ULUR'),
    FlowPath('d', start: Cell(3, 1), moves: 'LUUU'),
    FlowPath('e', start: Cell(0, 1), moves: 'RRRRRRDDDL'),
    FlowPath('f', start: Cell(2, 6), moves: 'ULLD'),
    FlowPath('g', start: Cell(2, 5), moves: 'DLLUULL'),
    FlowPath('h', start: Cell(2, 1), moves: 'RDDRRR'),
    FlowPath('i', start: Cell(4, 6), moves: 'RDLLLLLD'),
  ]),
  // 9 色 · 求解节点 244
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(5, 5), moves: 'LURUURU'),
    FlowPath('b', start: Cell(1, 5), moves: 'LLLLD'),
    FlowPath('c', start: Cell(3, 1), moves: 'RUR'),
    FlowPath('d', start: Cell(2, 4), moves: 'DLDLLLU'),
    FlowPath('e', start: Cell(2, 0), moves: 'UURRRRR'),
    FlowPath('f', start: Cell(0, 6), moves: 'RDDDD'),
    FlowPath('g', start: Cell(5, 7), moves: 'DDLLLLLU'),
    FlowPath('h', start: Cell(6, 1), moves: 'DLUU'),
    FlowPath('i', start: Cell(5, 1), moves: 'RRDRRRUUU'),
  ]),
  // 9 色 · 求解节点 263
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(5, 6), moves: 'DDLULDL'),
    FlowPath('b', start: Cell(6, 3), moves: 'ULDDLLU'),
    FlowPath('c', start: Cell(6, 1), moves: 'ULURRUU'),
    FlowPath('d', start: Cell(1, 2), moves: 'LDDL'),
    FlowPath('e', start: Cell(2, 0), moves: 'UURRRDD'),
    FlowPath('f', start: Cell(3, 3), moves: 'DRD'),
    FlowPath('g', start: Cell(5, 5), moves: 'UULUUUR'),
    FlowPath('h', start: Cell(1, 5), moves: 'DRUURDDD'),
    FlowPath('i', start: Cell(3, 6), moves: 'DRDDD'),
  ]),
  // 10 色 · 求解节点 285
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(1, 4), moves: 'RRDDDDD'),
    FlowPath('b', start: Cell(7, 6), moves: 'RUUUUU'),
    FlowPath('c', start: Cell(1, 7), moves: 'ULLLLLLL'),
    FlowPath('d', start: Cell(1, 0), moves: 'RRRDRRD'),
    FlowPath('e', start: Cell(3, 4), moves: 'LDRR'),
    FlowPath('f', start: Cell(5, 5), moves: 'DDLL'),
    FlowPath('g', start: Cell(7, 2), moves: 'LLUUR'),
    FlowPath('h', start: Cell(6, 1), moves: 'RRRU'),
    FlowPath('i', start: Cell(5, 3), moves: 'LULLU'),
    FlowPath('j', start: Cell(3, 1), moves: 'RULL'),
  ]),
  // 10 色 · 求解节点 298
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(7, 3), moves: 'LULDLUU'),
    FlowPath('b', start: Cell(5, 1), moves: 'ULUUR'),
    FlowPath('c', start: Cell(3, 1), moves: 'RUULLU'),
    FlowPath('d', start: Cell(0, 1), moves: 'RRRD'),
    FlowPath('e', start: Cell(1, 3), moves: 'DRRUU'),
    FlowPath('f', start: Cell(0, 6), moves: 'RDLDDLL'),
    FlowPath('g', start: Cell(3, 3), moves: 'DLD'),
    FlowPath('h', start: Cell(5, 3), moves: 'DRDRUULUR'),
    FlowPath('i', start: Cell(4, 6), moves: 'DDDRU'),
    FlowPath('j', start: Cell(5, 7), moves: 'UUU'),
  ]),
  // 10 色 · 求解节点 325
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(1, 0), moves: 'URDD'),
    FlowPath('b', start: Cell(2, 0), moves: 'DRD'),
    FlowPath('c', start: Cell(4, 0), moves: 'DDDRRRRR'),
    FlowPath('d', start: Cell(7, 6), moves: 'RULLLLL'),
    FlowPath('e', start: Cell(6, 1), moves: 'URRULUU'),
    FlowPath('f', start: Cell(1, 2), moves: 'URDDD'),
    FlowPath('g', start: Cell(3, 4), moves: 'UUUR'),
    FlowPath('h', start: Cell(0, 6), moves: 'RDD'),
    FlowPath('i', start: Cell(2, 6), moves: 'ULDDDL'),
    FlowPath('j', start: Cell(5, 4), moves: 'RRRUULD'),
  ]),
  // 10 色 · 求解节点 379
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(1, 0), moves: 'URRRD'),
    FlowPath('b', start: Cell(1, 2), moves: 'LDLDDDR'),
    FlowPath('c', start: Cell(6, 1), moves: 'LDRRRRRR'),
    FlowPath('d', start: Cell(7, 7), moves: 'UUUU'),
    FlowPath('e', start: Cell(3, 6), moves: 'DDDLL'),
    FlowPath('f', start: Cell(6, 3), moves: 'LURUL'),
    FlowPath('g', start: Cell(4, 1), moves: 'URURDRD'),
    FlowPath('h', start: Cell(5, 4), moves: 'RUUU'),
    FlowPath('i', start: Cell(1, 5), moves: 'RDRU'),
    FlowPath('j', start: Cell(0, 7), moves: 'LLLDD'),
  ]),
  // 10 色 · 求解节点 452
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(2, 1), moves: 'URRR'),
    FlowPath('b', start: Cell(2, 4), moves: 'LLD'),
    FlowPath('c', start: Cell(3, 1), moves: 'LUUUR'),
    FlowPath('d', start: Cell(0, 2), moves: 'RRRDDR'),
    FlowPath('e', start: Cell(1, 6), moves: 'URDDDL'),
    FlowPath('f', start: Cell(4, 6), moves: 'RDLLD'),
    FlowPath('g', start: Cell(6, 6), moves: 'RDLLLL'),
    FlowPath('h', start: Cell(7, 2), moves: 'LLURULURRD'),
    FlowPath('i', start: Cell(6, 2), moves: 'RRUU'),
    FlowPath('j', start: Cell(4, 5), moves: 'ULLDD'),
  ]),
  // 10 色 · 求解节点 591
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(1, 1), moves: 'RRRRRD'),
    FlowPath('b', start: Cell(2, 7), moves: 'UULLL'),
    FlowPath('c', start: Cell(0, 3), moves: 'LLLDDD'),
    FlowPath('d', start: Cell(4, 0), moves: 'DDDRU'),
    FlowPath('e', start: Cell(6, 2), moves: 'DRR'),
    FlowPath('f', start: Cell(7, 5), moves: 'RRUL'),
    FlowPath('g', start: Cell(6, 5), moves: 'LLUUUR'),
    FlowPath('h', start: Cell(4, 4), moves: 'DRRRUUL'),
    FlowPath('i', start: Cell(4, 6), moves: 'LUULLLLD'),
    FlowPath('j', start: Cell(3, 2), moves: 'DDLU'),
  ]),
  // 10 色 · 求解节点 726
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(5, 7), moves: 'DDLUULD'),
    FlowPath('b', start: Cell(7, 5), moves: 'LLLLLU'),
    FlowPath('c', start: Cell(6, 1), moves: 'RRRUURUUU'),
    FlowPath('d', start: Cell(1, 6), moves: 'DDD'),
    FlowPath('e', start: Cell(4, 7), moves: 'UUUULLL'),
    FlowPath('f', start: Cell(1, 4), moves: 'DDLLD'),
    FlowPath('g', start: Cell(4, 1), moves: 'UUR'),
    FlowPath('h', start: Cell(2, 3), moves: 'UUL'),
    FlowPath('i', start: Cell(1, 2), moves: 'LULD'),
    FlowPath('j', start: Cell(2, 0), moves: 'DDDRRRU'),
  ]),
  // 10 色 · 求解节点 952
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(7, 7), moves: 'LLLLLU'),
    FlowPath('b', start: Cell(6, 3), moves: 'RRRRU'),
    FlowPath('c', start: Cell(5, 6), moves: 'LLLU'),
    FlowPath('d', start: Cell(4, 4), moves: 'RURDRUU'),
    FlowPath('e', start: Cell(1, 7), moves: 'ULDD'),
    FlowPath('f', start: Cell(2, 5), moves: 'UULLD'),
    FlowPath('g', start: Cell(1, 2), moves: 'ULLDRDL'),
    FlowPath('h', start: Cell(3, 0), moves: 'RDLDDD'),
    FlowPath('i', start: Cell(7, 1), moves: 'UURU'),
    FlowPath('j', start: Cell(3, 2), moves: 'URDRUU'),
  ]),
  // 9 色 · 求解节点 1138
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(0, 5), moves: 'LDLUL'),
    FlowPath('b', start: Cell(1, 2), moves: 'DLUULDDDR'),
    FlowPath('c', start: Cell(4, 1), moves: 'LDDDRU'),
    FlowPath('d', start: Cell(5, 1), moves: 'RUURU'),
    FlowPath('e', start: Cell(2, 4), moves: 'DRDRU'),
    FlowPath('f', start: Cell(2, 6), moves: 'LURURDDDD'),
    FlowPath('g', start: Cell(5, 7), moves: 'LDRDLLU'),
    FlowPath('h', start: Cell(5, 5), moves: 'LULDD'),
    FlowPath('i', start: Cell(6, 4), moves: 'DLLU'),
  ]),
  // 10 色 · 求解节点 5909
  LevelDef(8, <FlowPath>[
    FlowPath('a', start: Cell(7, 0), moves: 'UU'),
    FlowPath('b', start: Cell(4, 0), moves: 'UUU'),
    FlowPath('c', start: Cell(0, 0), moves: 'RDR'),
    FlowPath('d', start: Cell(0, 2), moves: 'RDDLLD'),
    FlowPath('e', start: Cell(4, 1), moves: 'DDDR'),
    FlowPath('f', start: Cell(6, 2), moves: 'URULURRDR'),
    FlowPath('g', start: Cell(3, 5), moves: 'RUULDLUUR'),
    FlowPath('h', start: Cell(0, 6), moves: 'RDDDDLD'),
    FlowPath('i', start: Cell(5, 7), moves: 'DDLULD'),
    FlowPath('j', start: Cell(7, 4), moves: 'LURUR'),
  ]),
];

/// Master 档题库（9×9），由易到难。
const List<LevelDef> _master = <LevelDef>[
  // 12 色 · 求解节点 1115
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(5, 7), moves: 'RULU'),
    FlowPath('b', start: Cell(3, 8), moves: 'ULURULL'),
    FlowPath('c', start: Cell(1, 6), moves: 'LULLL'),
    FlowPath('d', start: Cell(0, 1), moves: 'LDRRRRDRRD'),
    FlowPath('e', start: Cell(3, 5), moves: 'LDRR'),
    FlowPath('f', start: Cell(5, 6), moves: 'DRRDDL'),
    FlowPath('g', start: Cell(7, 7), moves: 'LDLL'),
    FlowPath('h', start: Cell(8, 3), moves: 'LLLUUR'),
    FlowPath('i', start: Cell(5, 1), moves: 'LUUURRRDL'),
    FlowPath('j', start: Cell(3, 1), moves: 'DR'),
    FlowPath('k', start: Cell(4, 3), moves: 'DLDRR'),
    FlowPath('l', start: Cell(5, 4), moves: 'RDDLLLL'),
  ]),
  // 12 色 · 求解节点 1245
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(5, 1), moves: 'RRUL'),
    FlowPath('b', start: Cell(4, 1), moves: 'URRRD'),
    FlowPath('c', start: Cell(5, 4), moves: 'RDDRRR'),
    FlowPath('d', start: Cell(8, 8), moves: 'LLLL'),
    FlowPath('e', start: Cell(7, 4), moves: 'ULDDLU'),
    FlowPath('f', start: Cell(6, 2), moves: 'LDDLU'),
    FlowPath('g', start: Cell(6, 0), moves: 'UUUURU'),
    FlowPath('h', start: Cell(1, 0), moves: 'URRRRRRRRDL'),
    FlowPath('i', start: Cell(2, 7), moves: 'RDDDDLLU'),
    FlowPath('j', start: Cell(5, 7), moves: 'UULDLUUR'),
    FlowPath('k', start: Cell(1, 6), moves: 'LLDL'),
    FlowPath('l', start: Cell(2, 2), moves: 'UR'),
  ]),
  // 12 色 · 求解节点 1486
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(6, 0), moves: 'URRUUU'),
    FlowPath('b', start: Cell(2, 3), moves: 'DDDDLLDLDRRU'),
    FlowPath('c', start: Cell(7, 3), moves: 'DRUURRDL'),
    FlowPath('d', start: Cell(8, 5), moves: 'RRRUL'),
    FlowPath('e', start: Cell(6, 7), moves: 'RUUUULL'),
    FlowPath('f', start: Cell(3, 6), moves: 'RDDLULD'),
    FlowPath('g', start: Cell(5, 4), moves: 'UURUL'),
    FlowPath('h', start: Cell(1, 4), moves: 'LLLDD'),
    FlowPath('i', start: Cell(4, 1), moves: 'LUUU'),
    FlowPath('j', start: Cell(0, 0), moves: 'RRRRR'),
    FlowPath('k', start: Cell(1, 5), moves: 'RUR'),
    FlowPath('l', start: Cell(0, 8), moves: 'DL'),
  ]),
  // 12 色 · 求解节点 1559
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(1, 7), moves: 'DLUL'),
    FlowPath('b', start: Cell(0, 5), moves: 'RRR'),
    FlowPath('c', start: Cell(1, 8), moves: 'DDDDDDD'),
    FlowPath('d', start: Cell(8, 7), moves: 'LLLLLLL'),
    FlowPath('e', start: Cell(7, 0), moves: 'UUUUUUU'),
    FlowPath('f', start: Cell(0, 1), moves: 'RRRDLDD'),
    FlowPath('g', start: Cell(3, 4), moves: 'URDR'),
    FlowPath('h', start: Cell(3, 7), moves: 'DLDDL'),
    FlowPath('i', start: Cell(5, 5), moves: 'ULLLUUU'),
    FlowPath('j', start: Cell(1, 1), moves: 'DDDDD'),
    FlowPath('k', start: Cell(7, 1), moves: 'RUURRD'),
    FlowPath('l', start: Cell(6, 3), moves: 'DRRRRUU'),
  ]),
  // 11 色 · 求解节点 1561
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(3, 1), moves: 'LURULURR'),
    FlowPath('b', start: Cell(1, 2), moves: 'RURDDL'),
    FlowPath('c', start: Cell(2, 2), moves: 'DRRDR'),
    FlowPath('d', start: Cell(5, 5), moves: 'DDL'),
    FlowPath('e', start: Cell(6, 4), moves: 'ULULLL'),
    FlowPath('f', start: Cell(5, 0), moves: 'RRDRDLLU'),
    FlowPath('g', start: Cell(6, 0), moves: 'DDRRRRR'),
    FlowPath('h', start: Cell(8, 6), moves: 'RRUUUUUUU'),
    FlowPath('i', start: Cell(0, 8), moves: 'LDLULDDR'),
    FlowPath('j', start: Cell(2, 7), moves: 'DDDDD'),
    FlowPath('k', start: Cell(7, 6), moves: 'UUUUL'),
  ]),
  // 12 色 · 求解节点 1617
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(1, 3), moves: 'URDDRD'),
    FlowPath('b', start: Cell(4, 5), moves: 'DLLLLUU'),
    FlowPath('c', start: Cell(3, 2), moves: 'DRRULULLU'),
    FlowPath('d', start: Cell(1, 2), moves: 'ULLD'),
    FlowPath('e', start: Cell(2, 0), moves: 'DDDDR'),
    FlowPath('f', start: Cell(7, 1), moves: 'LDRRRRR'),
    FlowPath('g', start: Cell(8, 6), moves: 'RRUUU'),
    FlowPath('h', start: Cell(4, 8), moves: 'UUUULD'),
    FlowPath('i', start: Cell(2, 7), moves: 'DD'),
    FlowPath('j', start: Cell(5, 7), moves: 'DDLLL'),
    FlowPath('k', start: Cell(7, 3), moves: 'LURRRRU'),
    FlowPath('l', start: Cell(4, 6), moves: 'UUUULD'),
  ]),
  // 11 色 · 求解节点 2246
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(7, 7), moves: 'RDLLLLLLUU'),
    FlowPath('b', start: Cell(6, 3), moves: 'DRRRUUUU'),
    FlowPath('c', start: Cell(2, 6), moves: 'URDDDDDR'),
    FlowPath('d', start: Cell(5, 8), moves: 'UUUUUL'),
    FlowPath('e', start: Cell(0, 6), moves: 'LDDDDDDL'),
    FlowPath('f', start: Cell(5, 4), moves: 'LLLDDD'),
    FlowPath('g', start: Cell(8, 0), moves: 'UUU'),
    FlowPath('h', start: Cell(4, 0), moves: 'RRRRU'),
    FlowPath('i', start: Cell(2, 4), moves: 'LDLL'),
    FlowPath('j', start: Cell(3, 0), moves: 'URRURRU'),
    FlowPath('k', start: Cell(0, 3), moves: 'LLLDR'),
  ]),
  // 12 色 · 求解节点 2993
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(8, 4), moves: 'LLLLU'),
    FlowPath('b', start: Cell(7, 1), moves: 'RRRR'),
    FlowPath('c', start: Cell(8, 5), moves: 'RRRUUU'),
    FlowPath('d', start: Cell(4, 8), moves: 'UUUUL'),
    FlowPath('e', start: Cell(0, 6), moves: 'LLLLLD'),
    FlowPath('f', start: Cell(1, 2), moves: 'RRRRRDDDL'),
    FlowPath('g', start: Cell(3, 6), moves: 'ULLLD'),
    FlowPath('h', start: Cell(3, 4), moves: 'RDDRRDDL'),
    FlowPath('i', start: Cell(6, 6), moves: 'LLLL'),
    FlowPath('j', start: Cell(6, 1), moves: 'LUUU'),
    FlowPath('k', start: Cell(3, 1), moves: 'DDRR'),
    FlowPath('l', start: Cell(5, 4), moves: 'ULLUULLUU'),
  ]),
  // 12 色 · 求解节点 5842
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(1, 7), moves: 'DDDDD'),
    FlowPath('b', start: Cell(7, 7), moves: 'LLLD'),
    FlowPath('c', start: Cell(8, 5), moves: 'RRRU'),
    FlowPath('d', start: Cell(6, 8), moves: 'UUUUUUL'),
    FlowPath('e', start: Cell(0, 6), moves: 'LLLLDDDDR'),
    FlowPath('f', start: Cell(3, 3), moves: 'RDDLLL'),
    FlowPath('g', start: Cell(4, 1), moves: 'UUU'),
    FlowPath('h', start: Cell(0, 1), moves: 'LDDDDDDR'),
    FlowPath('i', start: Cell(6, 2), moves: 'DLLD'),
    FlowPath('j', start: Cell(8, 1), moves: 'RRUU'),
    FlowPath('k', start: Cell(6, 4), moves: 'RUUUULL'),
    FlowPath('l', start: Cell(1, 3), moves: 'RRRDDDDD'),
  ]),
  // 11 色 · 求解节点 6644
  LevelDef(9, <FlowPath>[
    FlowPath('a', start: Cell(7, 5), moves: 'LLLL'),
    FlowPath('b', start: Cell(7, 0), moves: 'DRRRR'),
    FlowPath('c', start: Cell(8, 5), moves: 'RURDRUU'),
    FlowPath('d', start: Cell(6, 7), moves: 'URUU'),
    FlowPath('e', start: Cell(2, 8), moves: 'LURULLD'),
    FlowPath('f', start: Cell(1, 5), moves: 'ULDDD'),
    FlowPath('g', start: Cell(3, 3), moves: 'UUULLL'),
    FlowPath('h', start: Cell(1, 0), moves: 'RRDDDRR'),
    FlowPath('i', start: Cell(5, 4), moves: 'RUUURDR'),
    FlowPath('j', start: Cell(4, 7), moves: 'LDDLLLULLUU'),
    FlowPath('k', start: Cell(2, 1), moves: 'LDDDDRR'),
  ]),
];

List<LevelDef> levelsFor(Difficulty difficulty) => switch (difficulty) {
      Difficulty.normal => _normal,
      Difficulty.hard => _hard,
      Difficulty.expert => _expert,
      Difficulty.master => _master,
    };
