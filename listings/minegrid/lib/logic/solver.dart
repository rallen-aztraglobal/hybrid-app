import 'mine_field.dart';

/// 逻辑求解器：模拟一个「只讲道理、从不猜」的玩家，看这盘能不能走完。
///
/// **为什么非要有它。** 扫雷最劝退的一件事是：明明一步没走错，最后剩两格
/// 五五开，蒙错就前功尽弃。那种局失败跟玩家水平无关，纯粹是运气。
/// 这个包的做法是：出题时就把这种局筛掉 —— 只有能纯靠推理走完的布局才发给玩家。
/// 于是「输了」永远是自己算错，而不是运气不好。
///
/// **用的三条规则就是人真正会用的那三条**：
/// 1. 基本规则：某个数字周围的雷已经标满 → 剩下的都安全；剩下的格数正好等于
///    还缺的雷数 → 剩下的都是雷。
/// 2. 子集规则：A 的未知格集合是 B 的子集时，B 多出来的那部分就装着 (B缺 − A缺) 颗雷。
///    这是「1-2-1」「1-2-2-1」这类经典手法的一般形式。
/// 3. 总数规则：已标记的雷数等于总雷数 → 剩下全安全；剩下的未知格数等于剩余雷数 → 全是雷。
///
/// 规则集比「完美求解器」弱（不做前沿全枚举）。这个方向是**安全**的：
/// 我们只会把某些其实可解的布局误判成不可解，代价是生成时多试几次；
/// 绝不会把需要猜的布局放行。反过来做才会出事。
class MineSolver {
  MineSolver._(this._field)
      : _revealed = List<bool>.filled(_field.cellCount, false),
        _flagged = List<bool>.filled(_field.cellCount, false);

  final MineField _field;
  final List<bool> _revealed;
  final List<bool> _flagged;

  /// 从 [firstClick] 开局，这盘能否全程不猜地走完。
  static bool isNoGuess(MineField field, int firstClick) {
    if (field.isMine(firstClick)) return false;
    return MineSolver._(field)._solve(firstClick);
  }

  bool _solve(int firstClick) {
    _reveal(firstClick);
    while (_basicRule() || _subsetRule() || _globalCountRule()) {
      // 每有进展就从最便宜的规则重新来一轮
    }
    for (var i = 0; i < _field.cellCount; i++) {
      if (!_revealed[i] && !_field.isMine(i)) return false;
    }
    return true;
  }

  /// 翻开一格；空格（周围 0 雷）连带翻开邻居，和真实玩法一致。
  void _reveal(int index) {
    if (_revealed[index] || _flagged[index]) return;
    _revealed[index] = true;
    if (_field.adjacentMines(index) != 0) return;
    for (final q in _field.neighboursOf(index)) {
      _reveal(q);
    }
  }

  bool _basicRule() {
    var changed = false;
    for (var i = 0; i < _field.cellCount; i++) {
      if (!_revealed[i]) continue;
      final value = _field.adjacentMines(i);
      var flags = 0;
      final unknown = <int>[];
      for (final q in _field.neighboursOf(i)) {
        if (_flagged[q]) {
          flags++;
        } else if (!_revealed[q]) {
          unknown.add(q);
        }
      }
      if (unknown.isEmpty) continue;

      if (value == flags) {
        for (final q in unknown) {
          _reveal(q);
        }
        changed = true;
      } else if (value - flags == unknown.length) {
        for (final q in unknown) {
          _flagged[q] = true;
        }
        changed = true;
      }
    }
    return changed;
  }

  bool _subsetRule() {
    final constraints = <_Constraint>[];
    for (var i = 0; i < _field.cellCount; i++) {
      if (!_revealed[i]) continue;
      var flags = 0;
      final unknown = <int>{};
      for (final q in _field.neighboursOf(i)) {
        if (_flagged[q]) {
          flags++;
        } else if (!_revealed[q]) {
          unknown.add(q);
        }
      }
      if (unknown.isEmpty) continue;
      constraints.add(_Constraint(unknown, _field.adjacentMines(i) - flags));
    }

    var changed = false;
    for (final a in constraints) {
      for (final b in constraints) {
        if (identical(a, b)) continue;
        if (a.cells.length >= b.cells.length) continue;
        if (!b.cells.containsAll(a.cells)) continue;

        final diff = b.cells.difference(a.cells);
        final rest = b.remaining - a.remaining;
        if (rest == 0) {
          for (final q in diff) {
            _reveal(q);
          }
          changed = true;
        } else if (rest == diff.length) {
          for (final q in diff) {
            _flagged[q] = true;
          }
          changed = true;
        }
      }
      // 本轮已经有推进就先回去跑基本规则，通常更快收敛
      if (changed) break;
    }
    return changed;
  }

  bool _globalCountRule() {
    var flags = 0;
    final unknown = <int>[];
    for (var i = 0; i < _field.cellCount; i++) {
      if (_flagged[i]) {
        flags++;
      } else if (!_revealed[i]) {
        unknown.add(i);
      }
    }
    if (unknown.isEmpty) return false;

    final remaining = _field.mines.length - flags;
    if (remaining == 0) {
      for (final q in unknown) {
        _reveal(q);
      }
      return true;
    }
    if (remaining == unknown.length) {
      for (final q in unknown) {
        _flagged[q] = true;
      }
      return true;
    }
    return false;
  }
}

/// 一条约束：[cells] 这组未知格里正好有 [remaining] 颗雷。
class _Constraint {
  _Constraint(this.cells, this.remaining);

  final Set<int> cells;
  final int remaining;
}
