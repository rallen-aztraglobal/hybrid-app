/// 一个计数器。
class Counter {
  Counter({
    required this.id,
    required this.label,
    this.value = 0,
    this.step = 1,
    this.target = 0,
    this.colorIndex = 0,
  });

  /// 从存档还原。
  ///
  /// **每个字段都当成可能是坏的来读。** 这是用户唯一的数据 ——
  /// 存档里少一个键、类型不对、被别的版本写过，都不能让整份计数丢掉；
  /// 读不出来的字段退回默认值，能救几个救几个。
  ///
  /// `target` / `colorIndex` 是后加的字段，旧存档里根本没有 ——
  /// 这两个键缺失是**正常情况**，不是损坏。
  factory Counter.fromJson(Map<String, dynamic> json) {
    final rawId = json['id'];
    final rawLabel = json['label'];
    final rawValue = json['value'];
    final rawStep = json['step'];
    final rawTarget = json['target'];
    final rawColor = json['color'];
    return Counter(
      id: rawId is String && rawId.isNotEmpty ? rawId : _fallbackId(json),
      label: rawLabel is String && rawLabel.trim().isNotEmpty
          ? rawLabel
          : 'Counter',
      value: rawValue is int && rawValue >= 0 ? rawValue : 0,
      step: rawStep is int && rawStep > 0 ? rawStep : 1,
      target: rawTarget is int && rawTarget > 0 ? rawTarget : 0,
      colorIndex: rawColor is int && rawColor >= 0 ? rawColor : 0,
    );
  }

  /// id 丢了也不能让这条记录消失 —— 用内容凑一个稳定的出来。
  static String _fallbackId(Map<String, dynamic> json) =>
      'restored-${json.hashCode.abs()}';

  final String id;
  String label;

  /// 当前计数。**不允许为负** —— 见 [bump]。
  int value;

  /// 每次加减的步长。
  int step;

  /// 目标值。0 表示没设目标。
  ///
  /// 有目标时界面上会多一条进度和「12 / 20」这样的读数 ——
  /// 数到一半时最想知道的就是「还差几个」，心算这件事正是用计数器要免掉的。
  int target;

  /// 颜色标签在调色板里的下标。
  ///
  /// 快速点数时人是靠颜色扫卡片的，不是靠读名字。多个计数器并排时，
  /// 一眼的颜色差比一行小字管用得多。
  int colorIndex;

  bool get hasTarget => target > 0;

  /// 已达成目标。
  bool get isComplete => hasTarget && value >= target;

  /// 完成度 0~1（没设目标时恒为 0）。
  double get progress {
    if (!hasTarget) return 0;
    final p = value / target;
    return p > 1 ? 1 : p;
  }

  Map<String, dynamic> toJson() => <String, dynamic>{
        'id': id,
        'label': label,
        'value': value,
        'step': step,
        'target': target,
        'color': colorIndex,
      };

  Counter copy() => Counter(
        id: id,
        label: label,
        value: value,
        step: step,
        target: target,
        colorIndex: colorIndex,
      );

  /// 加减 [delta] 个步长，结果夹在 0 以上。
  ///
  /// 为什么不许负数：这类工具数的是「东西的个数」，没有负数的含义。
  /// 而减到 0 以下几乎总是误触（连点减号），如果放任下去，用户要一直点加号
  /// 才能回到 0，中间还看不出自己错在哪。夹住反而是对的。
  ///
  /// **超过目标不封顶** —— 目标只是个参考线，数超了是真事，替用户抹掉就是骗人。
  void bump(int delta) {
    final next = value + delta * step;
    value = next < 0 ? 0 : next;
  }

  void reset() => value = 0;
}
