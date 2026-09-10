import 'counter.dart';

/// 一整屏计数器的状态机，带撤销。
///
/// **撤销是这个 App 最重要的一个功能。** 计数器是闭着眼快点的东西 ——
/// 多点一下、点错一个卡片、手滑清零，都是家常便饭。没有撤销的话，
/// 用户只能凭记忆手动改回去，而他之所以用计数器，正是因为记不住。
///
/// 实现上存的是**整份状态的快照**而不是「反向操作」：
/// 反向操作要为每种动作单独写一个逆操作，删除卡片的逆操作还得记住它原来的位置，
/// 漏一个就会出现「撤销之后数据变了样」这种最难查的 bug。
/// 快照顶多几十个整数，一份不到 1 KB，存下来毫不费事。
class CounterBoard {
  CounterBoard(List<Counter> counters) : _counters = counters;

  /// 撤销栈的深度。够覆盖「连点了一串然后发现点错卡片」的场景，
  /// 又不至于让内存无限涨。
  static const int maxUndo = 30;

  List<Counter> _counters;
  final List<List<Counter>> _undo = <List<Counter>>[];

  List<Counter> get counters => List<Counter>.unmodifiable(_counters);

  int get length => _counters.length;

  bool get canUndo => _undo.isNotEmpty;

  /// 所有计数器的总和。
  int get total => _counters.fold(0, (sum, c) => sum + c.value);

  Counter? byId(String id) {
    for (final c in _counters) {
      if (c.id == id) return c;
    }
    return null;
  }

  void _snapshot() {
    _undo.add(<Counter>[for (final c in _counters) c.copy()]);
    if (_undo.length > maxUndo) _undo.removeAt(0);
  }

  /// 加减。[delta] 为步数，实际变化量是 `delta × 该计数器的步长`。
  ///
  /// 返回 false 表示什么都没变（比如已经是 0 还要减）——
  /// 调用方据此不记撤销、不震动，免得给出「按上了」的假反馈。
  bool bump(String id, int delta) {
    final counter = byId(id);
    if (counter == null) return false;
    final before = counter.value;
    _snapshot();
    counter.bump(delta);
    if (counter.value == before) {
      _undo.removeLast(); // 没变化就把刚压进去的快照撤掉，别占撤销栈
      return false;
    }
    return true;
  }

  void add(Counter counter) {
    _snapshot();
    _counters = <Counter>[..._counters, counter];
  }

  bool remove(String id) {
    if (byId(id) == null) return false;
    _snapshot();
    _counters = <Counter>[
      for (final c in _counters)
        if (c.id != id) c,
    ];
    return true;
  }

  /// 改名 / 改步长 / 改目标 / 改颜色。传 null 表示这一项不动。
  ///
  /// [target] 传 0 表示**取消目标**（这是有意的：0 不是「不动」，null 才是），
  /// 否则用户设了目标之后就没法再撤回到「不设目标」。
  bool edit(String id, {String? label, int? step, int? target, int? colorIndex}) {
    final counter = byId(id);
    if (counter == null) return false;
    _snapshot();
    if (label != null && label.trim().isNotEmpty) counter.label = label.trim();
    if (step != null && step > 0) counter.step = step;
    if (target != null && target >= 0) counter.target = target;
    if (colorIndex != null && colorIndex >= 0) counter.colorIndex = colorIndex;
    return true;
  }

  /// 拖动排序。越界的下标一律不动 ——
  /// ReorderableListView 的下标语义容易搞错，这里宁可什么都不做也不要错位。
  bool move(int from, int to) {
    if (from < 0 || from >= _counters.length) return false;
    if (to < 0 || to >= _counters.length) return false;
    if (from == to) return false;
    _snapshot();
    final next = <Counter>[..._counters];
    next.insert(to, next.removeAt(from));
    _counters = next;
    return true;
  }

  bool resetOne(String id) {
    final counter = byId(id);
    if (counter == null || counter.value == 0) return false;
    _snapshot();
    counter.reset();
    return true;
  }

  /// 全部清零。计数值全是 0 时什么都不做 —— 免得往撤销栈里塞空操作。
  bool resetAll() {
    if (_counters.every((c) => c.value == 0)) return false;
    _snapshot();
    for (final c in _counters) {
      c.reset();
    }
    return true;
  }

  /// 撤销上一步。返回 false 表示没得撤了。
  bool undo() {
    if (_undo.isEmpty) return false;
    _counters = _undo.removeLast();
    return true;
  }

  List<Map<String, dynamic>> toJson() =>
      <Map<String, dynamic>>[for (final c in _counters) c.toJson()];
}
