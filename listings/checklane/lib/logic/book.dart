import 'checklist.dart';

/// 所有清单的集合，带撤销。
///
/// 撤销用**整份状态的快照**，不是「反向操作」：
/// 反向操作要为每种动作单独写逆操作，删除一条的逆操作还得记住它原来的位置，
/// 漏一个就会出现「撤销之后数据变了样」这种最难查的 bug。
/// 这里的数据量很小（几份清单、几十条），一份快照不值几个字节。
class ChecklistBook {
  ChecklistBook(List<Checklist> lists) : _lists = lists;

  static const int maxUndo = 30;

  List<Checklist> _lists;
  final List<List<Checklist>> _undo = <List<Checklist>>[];

  List<Checklist> get lists => List<Checklist>.unmodifiable(_lists);
  int get length => _lists.length;
  bool get canUndo => _undo.isNotEmpty;

  Checklist? byId(String id) {
    for (final l in _lists) {
      if (l.id == id) return l;
    }
    return null;
  }

  void _snapshot() {
    _undo.add(<Checklist>[for (final l in _lists) l.copy()]);
    if (_undo.length > maxUndo) _undo.removeAt(0);
  }

  /// 勾 / 取消勾。
  bool toggle(String listId, String itemId) {
    final item = byId(listId)?.byId(itemId);
    if (item == null) return false;
    _snapshot();
    item.done = !item.done;
    return true;
  }

  bool addItem(String listId, String text, String itemId) {
    final list = byId(listId);
    if (list == null || text.trim().isEmpty) return false;
    _snapshot();
    list.items = <ChecklistItem>[
      ...list.items,
      ChecklistItem(id: itemId, text: text.trim()),
    ];
    return true;
  }

  bool editItem(String listId, String itemId, String text) {
    final item = byId(listId)?.byId(itemId);
    if (item == null || text.trim().isEmpty) return false;
    _snapshot();
    item.text = text.trim();
    return true;
  }

  bool removeItem(String listId, String itemId) {
    final list = byId(listId);
    if (list == null || list.byId(itemId) == null) return false;
    _snapshot();
    list.items = <ChecklistItem>[
      for (final i in list.items)
        if (i.id != itemId) i,
    ];
    return true;
  }

  /// 条目排序。越界的下标一律不动 ——
  /// 拖拽回调的下标语义容易搞错，宁可什么都不做也不要错位。
  bool moveItem(String listId, int from, int to) {
    final list = byId(listId);
    if (list == null) return false;
    if (from < 0 || from >= list.items.length) return false;
    if (to < 0 || to >= list.items.length) return false;
    if (from == to) return false;
    _snapshot();
    final next = <ChecklistItem>[...list.items];
    next.insert(to, next.removeAt(from));
    list.items = next;
    return true;
  }

  /// 清掉某份清单的所有勾选，条目保留 —— 「再走一遍」。
  bool resetList(String listId) {
    final list = byId(listId);
    if (list == null) return false;
    _snapshot();
    if (!list.reset()) {
      _undo.removeLast(); // 本来就没勾，别占撤销栈
      return false;
    }
    return true;
  }

  bool addList(Checklist list) {
    _snapshot();
    _lists = <Checklist>[..._lists, list];
    return true;
  }

  bool renameList(String listId, String title) {
    final list = byId(listId);
    if (list == null || title.trim().isEmpty) return false;
    _snapshot();
    list.title = title.trim();
    return true;
  }

  bool removeList(String listId) {
    if (byId(listId) == null) return false;
    _snapshot();
    _lists = <Checklist>[
      for (final l in _lists)
        if (l.id != listId) l,
    ];
    return true;
  }

  /// 复制一份清单（当模板用）。副本插在原件后面，找得到。
  bool duplicateList(String listId, String newId, String idSeed) {
    final list = byId(listId);
    if (list == null) return false;
    _snapshot();
    final copy = list.duplicateAs(newId, '${list.title} copy', idSeed);
    final index = _lists.indexOf(list);
    final next = <Checklist>[..._lists];
    next.insert(index + 1, copy);
    _lists = next;
    return true;
  }

  bool moveList(int from, int to) {
    if (from < 0 || from >= _lists.length) return false;
    if (to < 0 || to >= _lists.length) return false;
    if (from == to) return false;
    _snapshot();
    final next = <Checklist>[..._lists];
    next.insert(to, next.removeAt(from));
    _lists = next;
    return true;
  }

  bool undo() {
    if (_undo.isEmpty) return false;
    _lists = _undo.removeLast();
    return true;
  }

  List<Map<String, dynamic>> toJson() =>
      <Map<String, dynamic>>[for (final l in _lists) l.toJson()];
}
