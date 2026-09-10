/// 单行（或单列）的约束求解。整个谜题的生成与「可纯逻辑推解」判定都建立在这上面。
///
/// 术语：一条线由若干格子组成，每格三态 —— `true` 已确定涂黑、`false` 已确定留空、
/// `null` 尚未确定。线索是一串正整数，表示从左到右各段连续涂黑的长度，段间至少隔一格。
/// 空线的线索约定为 `[0]`（与展示层一致），求解时按「没有任何段」处理。
library;

/// 枚举一条线在给定线索下的所有合法涂法。
///
/// 返回的每个元素是长度为 [length] 的布尔表。段之间强制留至少一格空白，
/// 这是数织的定义，不是启发式。
///
/// 复杂度是组合级的：段数 k、空位 s 时约为 C(s+k, k)。本包最大 10 宽，
/// 最坏情形（线索 `[1,1,1,1,1]`）也只有 C(6,5)=6 种，完全够用。
/// 若将来支持 15 宽以上，这里要改成带记忆化的逐格递推。
List<List<bool>> enumerateLines(List<int> clues, int length) {
  final blocks = clues.where((c) => c > 0).toList();
  final out = <List<bool>>[];

  if (blocks.isEmpty) {
    out.add(List<bool>.filled(length, false));
    return out;
  }

  // 段占用的最小总宽 = 各段长度之和 + 段间强制的 (k-1) 个空格
  final minWidth = blocks.reduce((a, b) => a + b) + blocks.length - 1;
  if (minWidth > length) return out; // 线索本身就放不下，无解

  final line = List<bool>.filled(length, false);

  void place(int blockIndex, int cursor) {
    if (blockIndex == blocks.length) {
      out.add(List<bool>.of(line));
      return;
    }
    final block = blocks[blockIndex];
    // 剩余段（含段间空格）还需要多少宽度
    var tail = 0;
    for (var i = blockIndex + 1; i < blocks.length; i++) {
      tail += blocks[i] + 1;
    }
    final lastStart = length - tail - block;
    for (var start = cursor; start <= lastStart; start++) {
      for (var i = 0; i < block; i++) {
        line[start + i] = true;
      }
      // 段后强制隔一格：下一段最早从 start + block + 1 开始
      place(blockIndex + 1, start + block + 1);
      for (var i = 0; i < block; i++) {
        line[start + i] = false;
      }
    }
  }

  place(0, 0);
  return out;
}

/// 在已知信息 [known] 下推进一条线，返回能被**唯一确定**的格子。
///
/// 做法是取所有与 [known] 相容的涂法的交集：某格在全部候选里都是涂黑 → 确定涂黑；
/// 都是留空 → 确定留空；两者都有 → 仍未确定。这是数织求解的标准手法，
/// 只产出被逻辑迫出的结论，不做猜测。
///
/// 返回 `null` 表示 [known] 与线索矛盾（无任何相容涂法）—— 生成器用它来判定死路。
List<bool?>? refineLine(List<int> clues, List<bool?> known) {
  final candidates = enumerateLines(clues, known.length)
      .where((cand) {
        for (var i = 0; i < known.length; i++) {
          final k = known[i];
          if (k != null && k != cand[i]) return false;
        }
        return true;
      })
      .toList();

  if (candidates.isEmpty) return null;

  final result = List<bool?>.filled(known.length, null);
  for (var i = 0; i < known.length; i++) {
    final first = candidates.first[i];
    var uniform = true;
    for (final cand in candidates) {
      if (cand[i] != first) {
        uniform = false;
        break;
      }
    }
    result[i] = uniform ? first : null;
  }
  return result;
}

/// 由一条线的答案反推它的线索。空线返回 `[0]`。
List<int> cluesOf(List<bool> line) {
  final out = <int>[];
  var run = 0;
  for (final filled in line) {
    if (filled) {
      run++;
    } else if (run > 0) {
      out.add(run);
      run = 0;
    }
  }
  if (run > 0) out.add(run);
  return out.isEmpty ? <int>[0] : out;
}
