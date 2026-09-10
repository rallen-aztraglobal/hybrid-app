import 'counter.dart';

/// 存下来的一次计数结果。
///
/// 为什么要有它：盘点、点票、观鸟这些事都是**一轮一轮**做的。
/// 数完一轮清零重来之后，上一轮的数字就没了 —— 而那才是用户真正要留的东西。
/// 有了会话，「清零」才敢放心按。
class Session {
  const Session({
    required this.savedAt,
    required this.entries,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    final rawAt = json['at'];
    final rawEntries = json['entries'];
    return Session(
      // 时间戳坏了就当纪元 0 —— 排序会把它排到最后，但记录本身还在
      savedAt: DateTime.fromMillisecondsSinceEpoch(rawAt is int ? rawAt : 0),
      entries: <SessionEntry>[
        if (rawEntries is List)
          for (final e in rawEntries)
            if (e is Map<String, dynamic>) SessionEntry.fromJson(e),
      ],
    );
  }

  /// 从当前这一屏计数器拍一张。
  factory Session.snapshot(List<Counter> counters, DateTime at) => Session(
        savedAt: at,
        entries: <SessionEntry>[
          for (final c in counters) SessionEntry(label: c.label, value: c.value),
        ],
      );

  final DateTime savedAt;
  final List<SessionEntry> entries;

  int get total => entries.fold(0, (sum, e) => sum + e.value);

  Map<String, dynamic> toJson() => <String, dynamic>{
        'at': savedAt.millisecondsSinceEpoch,
        'entries': <Map<String, dynamic>>[
          for (final e in entries) e.toJson(),
        ],
      };
}

/// 会话里的一条：某个计数器当时叫什么、数到几。
///
/// 存的是**名字的副本**而不是计数器 id：计数器后来被改名或删掉了，
/// 历史记录也该保持它当时的样子。历史是账，不该跟着现状变。
class SessionEntry {
  const SessionEntry({required this.label, required this.value});

  factory SessionEntry.fromJson(Map<String, dynamic> json) {
    final rawLabel = json['label'];
    final rawValue = json['value'];
    return SessionEntry(
      label: rawLabel is String && rawLabel.isNotEmpty ? rawLabel : 'Counter',
      value: rawValue is int && rawValue >= 0 ? rawValue : 0,
    );
  }

  final String label;
  final int value;

  Map<String, dynamic> toJson() =>
      <String, dynamic>{'label': label, 'value': value};
}
