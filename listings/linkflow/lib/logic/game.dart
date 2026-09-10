import 'board.dart';
import 'puzzle.dart';

/// 一局连线的状态机。纯数据，不依赖 Flutter，便于直接测。
///
/// 玩家的操作只有一种：从某个端点按下，沿正交方向拖过去，松手。
/// 拖过的格子依次记进这条通路；碰到别的通路就把对方从被撞的那一格起截断 ——
/// 这是这类游戏的通行手感，比「禁止穿过」友好得多：玩家不必先清理旧线。
class FlowGame {
  FlowGame(this.puzzle) : _endpoints = puzzle.endpoints;

  final FlowPuzzle puzzle;

  /// 端点表。[FlowPuzzle.endpoints] 是个每次都重建 Map 的 getter，
  /// 而 [isConnected] 在拖动过程中每帧都要查，所以开局缓存一次。
  final Map<String, List<Cell>> _endpoints;

  /// 每条通路当前画到哪。按拖动顺序存，第一格一定是某个端点。
  final Map<String, List<Cell>> _paths = <String, List<Cell>>{};

  /// 正在拖的那条通路。松手后置空。
  String? _dragging;

  String? get draggingKey => _dragging;

  List<Cell> pathOf(String key) => List<Cell>.unmodifiable(_paths[key] ?? const <Cell>[]);

  /// 这一格当前被哪条通路占着；没有则返回 null。
  String? occupantAt(Cell cell) {
    for (final entry in _paths.entries) {
      if (entry.value.contains(cell)) return entry.key;
    }
    return null;
  }

  /// 某条通路是否已经连通 —— 两端都在路径里，且路径首尾正是这两端。
  bool isConnected(String key) {
    final path = _paths[key];
    if (path == null || path.length < 2) return false;
    final ends = _endpoints[key]!;
    return (path.first == ends.first && path.last == ends.last) ||
        (path.first == ends.last && path.last == ends.first);
  }

  int get connectedCount => puzzle.colorKeys.where(isConnected).length;

  /// 已被任意通路覆盖的格子数。满盘是通关的另一个必要条件。
  int get filledCount {
    var n = 0;
    for (final path in _paths.values) {
      n += path.length;
    }
    return n;
  }

  /// 通关：每条通路都连通，且整盘填满。
  ///
  /// 两个条件缺一不可 —— 只连通不填满是这类游戏的经典「假通关」，
  /// 玩家会觉得自己赢了却没过关，所以两条都要判。
  bool get isSolved =>
      connectedCount == puzzle.colorKeys.length &&
      filledCount == puzzle.side * puzzle.side;

  /// 在 [cell] 按下。只有按在端点或已有通路上才会起手。
  ///
  /// 按在端点上 → 从该端点重新起画（丢弃这条通路原有的路径）。
  /// 按在自己通路的中段 → 从该处截断，接着往下画，相当于「回退重画」。
  bool beginDrag(Cell cell) {
    if (!puzzle.contains(cell)) return false;

    final owner = occupantAt(cell);
    if (owner != null) {
      // 踩在已有通路上：从这一格起截断，继续画
      final path = _paths[owner]!;
      final index = path.indexOf(cell);
      _paths[owner] = path.sublist(0, index + 1);
      _dragging = owner;
      return true;
    }

    final endKey = puzzle.endpointKeyAt(cell);
    if (endKey != null) {
      final key = endKey;
      _paths[key] = <Cell>[cell];
      _dragging = key;
      return true;
    }

    return false;
  }

  /// 拖到 [cell]。返回 true 表示路径有变化（调用方据此重绘）。
  bool dragTo(Cell cell) {
    final key = _dragging;
    if (key == null || !puzzle.contains(cell)) return false;

    final path = _paths[key]!;
    final tail = path.last;
    if (cell == tail) return false;

    // 往回拖 = 撤销最后一步。这是拖动式连线必备的手感，
    // 没有的话画错一格就得整条重来。
    if (path.length >= 2 && cell == path[path.length - 2]) {
      path.removeLast();
      return true;
    }

    if (!cell.isAdjacentTo(tail)) return false; // 只能一格一格走，不能跳

    // 已经连通之后就不再延伸 —— 否则玩家会不小心把线画过头
    if (isConnected(key)) return false;

    // 不能穿过别人的端点。**这一条必须在截断之前判**：
    // 早先写反了顺序，先把对方的线截掉、再判断发现不能走而 return，
    // 结果是玩家没走成，对方的线却被截了 —— 一个只在特定位置复现的怪 bug。
    final endKey = puzzle.endpointKeyAt(cell);
    if (endKey != null && endKey != key) return false;

    final owner = occupantAt(cell);
    if (owner == key) return false; // 撞到自己，忽略（回退走上面那条分支）

    if (owner != null) {
      // 撞到别的通路：把对方从被撞的那一格起截掉
      final other = _paths[owner]!;
      final index = other.indexOf(cell);
      _paths[owner] = other.sublist(0, index);
      if (_paths[owner]!.isEmpty) _paths.remove(owner);
    }

    path.add(cell);
    return true;
  }

  void endDrag() => _dragging = null;

  /// 清掉某条通路。玩家点一下已连通的线可以重画。
  void clearPath(String key) => _paths.remove(key);

  /// 全部清空，重来一局（谜题不变）。
  void reset() {
    _paths.clear();
    _dragging = null;
  }
}
