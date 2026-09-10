/// 清单里的一条。
class ChecklistItem {
  ChecklistItem({required this.id, required this.text, this.done = false});

  /// 从存档还原。每个字段都当成可能是坏的来读 —— 一条读坏了不能带崩整份清单。
  factory ChecklistItem.fromJson(Map<String, dynamic> json) {
    final rawId = json['id'];
    final rawText = json['text'];
    final rawDone = json['done'];
    return ChecklistItem(
      id: rawId is String && rawId.isNotEmpty
          ? rawId
          : 'item-${json.hashCode.abs()}',
      text: rawText is String && rawText.trim().isNotEmpty
          ? rawText
          : 'Untitled',
      done: rawDone is bool && rawDone,
    );
  }

  final String id;
  String text;
  bool done;

  Map<String, dynamic> toJson() =>
      <String, dynamic>{'id': id, 'text': text, 'done': done};

  ChecklistItem copy() => ChecklistItem(id: id, text: text, done: done);
}

/// 一份清单。
///
/// 这个 App 的核心不是「待办」而是「**核对表**」——
/// 待办勾完就删了；核对表要的是勾完 → 清一遍勾 → 下次照原样再走一遍。
/// 所以 [reset] 只清勾选、**保留条目**，它才是这里最要紧的一个动作。
class Checklist {
  Checklist({required this.id, required this.title, List<ChecklistItem>? items})
      : items = items ?? <ChecklistItem>[];

  factory Checklist.fromJson(Map<String, dynamic> json) {
    final rawId = json['id'];
    final rawTitle = json['title'];
    final rawItems = json['items'];
    return Checklist(
      id: rawId is String && rawId.isNotEmpty
          ? rawId
          : 'list-${json.hashCode.abs()}',
      title: rawTitle is String && rawTitle.trim().isNotEmpty
          ? rawTitle
          : 'Checklist',
      items: <ChecklistItem>[
        if (rawItems is List)
          for (final e in rawItems)
            if (e is Map<String, dynamic>) ChecklistItem.fromJson(e),
      ],
    );
  }

  final String id;
  String title;
  List<ChecklistItem> items;

  int get total => items.length;
  int get doneCount => items.where((i) => i.done).length;
  int get remaining => total - doneCount;

  /// 完成度 0~1。空清单算 0 —— 不能除以零，也不该显示成「已完成」。
  double get progress => total == 0 ? 0 : doneCount / total;

  /// 全部勾完。**空清单不算完成** —— 一条都没有的清单说「done」是没意义的。
  bool get isComplete => total > 0 && doneCount == total;

  ChecklistItem? byId(String itemId) {
    for (final i in items) {
      if (i.id == itemId) return i;
    }
    return null;
  }

  /// 清掉所有勾选，条目原样留着 —— 这就是「再走一遍」。
  ///
  /// 返回 false 表示本来就一个勾都没有，什么也没发生。
  bool reset() {
    if (doneCount == 0) return false;
    for (final i in items) {
      i.done = false;
    }
    return true;
  }

  /// 复制一份当模板用。
  ///
  /// 复制出来的**一律是未勾选的**，而且每条都换了新 id ——
  /// 沿用旧 id 会让两份清单在存档里互相串。
  Checklist duplicateAs(String newId, String newTitle, String idSeed) {
    return Checklist(
      id: newId,
      title: newTitle,
      items: <ChecklistItem>[
        for (var i = 0; i < items.length; i++)
          ChecklistItem(id: '$idSeed-$i', text: items[i].text),
      ],
    );
  }

  Checklist copy() => Checklist(
        id: id,
        title: title,
        items: <ChecklistItem>[for (final i in items) i.copy()],
      );

  Map<String, dynamic> toJson() => <String, dynamic>{
        'id': id,
        'title': title,
        'items': <Map<String, dynamic>>[for (final i in items) i.toJson()],
      };
}
