import 'nonogram.dart';

/// 一张可解的像素图 —— 谜题的答案、一个给玩家看的名字，以及配色。
///
/// 数织的乐趣在于解开之后浮现出一幅画。早先的实现是随机铺格子，
/// 解完只得到一片没有意义的杂色 —— 推理过程再严谨也没有回报，
/// 那些分段线索就纯粹成了数字作业。所以谜题必须来自手绘的图库。
///
/// **配色不改变规则。** 线索仍然是单色的标准数织：玩家只判断「涂或不涂」，
/// 不需要判断颜色。颜色是涂开之后揭示出来的画的本色，纯视觉回报 ——
/// 真正的「彩色数织」是另一种更难的变体（线索本身带颜色），本作不做。
class Picture {
  const Picture(this.name, this.rows, {this.palette = const <String, int>{}});

  /// 解开后显示给玩家的名字。
  final String name;

  /// 图案。`.` 表示留空，其余任何字符都表示涂黑 ——
  /// 用不同字符区分颜色，具体色值查 [palette]。
  final List<String> rows;

  /// 字符到颜色的映射（ARGB）。查不到的字符回落到主色，
  /// 所以单色图直接全用 `#` 、不给 palette 就行。
  final Map<String, int> palette;

  Nonogram toNonogram() => Nonogram.fromSolution(<List<bool>>[
        for (final row in rows)
          <bool>[for (final ch in row.split('')) ch != '.'],
      ]);

  /// 某一格涂开后该显示什么颜色。[fallback] 由主题传入。
  int colorAt(int row, int col, int fallback) =>
      palette[rows[row][col]] ?? fallback;

  int get side => rows.length;
}

// 图库共用的一组颜色。刻意取饱和度中等的色 —— 米白底上过艳会刺眼，
// 而且相邻两色要能分辨。
const int _green = 0xFF4CAF50;
const int _darkGreen = 0xFF2E7D32;
const int _brown = 0xFF8D6E63;
const int _red = 0xFFE53935;
const int _pink = 0xFFEC407A;
const int _amber = 0xFFFFB300;
const int _yellow = 0xFFFDD835;
const int _blue = 0xFF1E88E5;
const int _teal = 0xFF00ACC1;
const int _purple = 0xFF8E24AA;
const int _orange = 0xFFF57C00;
const int _slate = 0xFF546E7A;
// 帆用的暖沙色。**不能取真正的米白**（原来是 #FFF3E0）——
// 棋盘底板是纯白、空格是 #EBE7E0，米白夹在两者之间，涂完了跟没涂一样。
// 数织的全部回报就是最后看见那幅图，图看不见等于白做。
const int _cream = 0xFFE9C88C;

/// 5×5 图库。这个尺寸只够表达最简单的轮廓，所以都取辨识度高的形状。
const List<Picture> pictures5 = <Picture>[
  Picture('Heart', <String>[
    '.#.#.',
    '#####',
    '#####',
    '.###.',
    '..#..',
  ], palette: <String, int>{'#': _red}),
  Picture('Plus', <String>[
    '..#..',
    '..#..',
    '#####',
    '..#..',
    '..#..',
  ], palette: <String, int>{'#': _red}),
  Picture('Diamond', <String>[
    '..#..',
    '.###.',
    '#####',
    '.###.',
    '..#..',
  ], palette: <String, int>{'#': _teal}),
  Picture('Arrow', <String>[
    '..#..',
    '.###.',
    '#####',
    '..#..',
    '..#..',
  ], palette: <String, int>{'#': _blue}),
  Picture('House', <String>[
    '..r..',
    '.rrr.',
    'rrrrr',
    'w...w',
    'wwwww',
  ], palette: <String, int>{'r': _red, 'w': _brown}),
  Picture('Cup', <String>[
    'ccccc',
    'c...c',
    'c...c',
    '.ccc.',
    '.....',
  ], palette: <String, int>{'c': _teal}),
  Picture('Tree', <String>[
    '..g..',
    '.ggg.',
    'ggggg',
    '..t..',
    '.ttt.',
  ], palette: <String, int>{'g': _green, 't': _brown}),
  Picture('Boat', <String>[
    '...s.',
    '...s.',
    '..ss.',
    'hhhhh',
    '.hhh.',
  ], palette: <String, int>{'s': _cream, 'h': _brown}),
  Picture('Key', <String>[
    '.yyy.',
    '.y.y.',
    '.yyy.',
    '..y..',
    '..yy.',
  ], palette: <String, int>{'y': _amber}),
  Picture('Flag', <String>[
    'ffff.',
    'ffff.',
    'p....',
    'p....',
    'p....',
  ], palette: <String, int>{'f': _orange, 'p': _slate}),
  // 原来这里是一张左右上下都对称的「Face」，线索无法唯一确定 ——
  // 完全对称的图案是数织里典型的二义来源，被可解性测试拦下了。
  // 换成阶梯这种明显非对称的形状。
  Picture('Stairs', <String>[
    '#....',
    '##...',
    '###..',
    '####.',
    '#####',
  ], palette: <String, int>{'#': _purple}),
  Picture('Hourglass', <String>[
    '#####',
    '.###.',
    '..#..',
    '.###.',
    '#####',
  ], palette: <String, int>{'#': _amber}),
];

/// 8×8 图库。多出来的格子够画一点细节，但仍要保持轮廓清晰 ——
/// 8×8 下太碎的图案解出来也认不出是什么。
const List<Picture> pictures8 = <Picture>[
  Picture('Heart', <String>[
    '.##..##.',
    '########',
    '########',
    '########',
    '.######.',
    '..####..',
    '...##...',
    '........',
  ], palette: <String, int>{'#': _pink}),
  Picture('Cat', <String>[
    'o......o',
    'oo....oo',
    'oooooooo',
    'o.oooo.o',
    'oooooooo',
    'o.oooo.o',
    'oo....oo',
    '.oooooo.',
  ], palette: <String, int>{'o': _orange}),
  Picture('Star', <String>[
    '...yy...',
    '...yy...',
    'yyyyyyyy',
    '.yyyyyy.',
    '..yyyy..',
    '.yy..yy.',
    '.y....y.',
    '........',
  ], palette: <String, int>{'y': _yellow}),
  Picture('Mug', <String>[
    '........',
    '.tttttt.',
    '.t....t.',
    '.t....tt',
    '.t....tt',
    '.t....t.',
    '.tttttt.',
    '........',
  ], palette: <String, int>{'t': _teal}),
  Picture('Tree', <String>[
    '...gg...',
    '..gggg..',
    '.gggggg.',
    'gggggggg',
    '..gggg..',
    '.gggggg.',
    '...tt...',
    '...tt...',
  ], palette: <String, int>{'g': _darkGreen, 't': _brown}),
  Picture('Boat', <String>[
    '....s...',
    '....ss..',
    '....sss.',
    '....ssss',
    '........',
    'hhhhhhhh',
    '.hhhhhh.',
    '..hhhh..',
  ], palette: <String, int>{'s': _cream, 'h': _brown}),
  Picture('Note', <String>[
    '....pppp',
    '....p..p',
    '....pppp',
    '....p...',
    '....p...',
    '..ppp...',
    '.pppp...',
    '..pp....',
  ], palette: <String, int>{'p': _purple}),
  Picture('Umbrella', <String>[
    '..rrrr..',
    '.rrrrrr.',
    'rrrrrrrr',
    '...ss...',
    '...ss...',
    '...ss...',
    '..sss...',
    '..ss....',
  ], palette: <String, int>{'r': _red, 's': _slate}),
  Picture('Fish', <String>[
    '........',
    '..bbbb..',
    '.bbbbbbb',
    'bbbbbbbb',
    'bbbbbbbb',
    '.bbbbbbb',
    '..bbbb..',
    '........',
  ], palette: <String, int>{'b': _blue}),
  Picture('Envelope', <String>[
    '........',
    'aaaaaaaa',
    'aa....aa',
    'a.aaaa.a',
    'a.aaaa.a',
    'aa....aa',
    'aaaaaaaa',
    '........',
  ], palette: <String, int>{'a': _amber}),
  Picture('Bulb', <String>[
    '..yyyy..',
    '.yyyyyy.',
    '.yyyyyy.',
    '.yyyyyy.',
    '..yyyy..',
    '...ss...',
    '..ssss..',
    '...ss...',
  ], palette: <String, int>{'y': _yellow, 's': _slate}),
  Picture('Anchor', <String>[
    '...ss...',
    '...ss...',
    '.ssssss.',
    '...ss...',
    's..ss..s',
    's..ss..s',
    'ss....ss',
    '.ssssss.',
  ], palette: <String, int>{'s': _slate}),
];

/// 取某个边长下的全部图片。
List<Picture> picturesFor(int side) => switch (side) {
      5 => pictures5,
      8 => pictures8,
      _ => const <Picture>[],
    };
