import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/board.dart';
import '../logic/counter.dart';
import '../logic/session.dart';
import '../logic/summary.dart';
import '../platform/volume_keys.dart';
import '../storage/counter_store.dart';
import '../storage/session_store.dart';
import '../storage/settings_store.dart';
import '../theme/app_colors.dart';
import '../widgets/counter_card.dart';
import 'history_screen.dart';

/// A 面：计数器本体。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
class CountersScreen extends StatefulWidget {
  const CountersScreen({super.key});

  @override
  State<CountersScreen> createState() => _CountersScreenState();
}

class _CountersScreenState extends State<CountersScreen> {
  CounterBoard _board = CounterBoard(CounterStore.defaults());
  bool _loaded = false;

  /// 音量键作用在哪张卡片上。默认第一张，点过谁就换成谁。
  String? _activeId;

  bool _volumeKeys = false;

  @override
  void initState() {
    super.initState();
    VolumeKeys.listen(_onVolumeKey);
    _load();
  }

  @override
  void dispose() {
    // 离开这一页就把音量键还给系统，别让它在别处仍被拦着
    VolumeKeys.setEnabled(false);
    super.dispose();
  }

  Future<void> _load() async {
    final counters = await CounterStore.load();
    final volumeKeys = await SettingsStore.volumeKeys();
    if (!mounted) return;
    setState(() {
      _board = CounterBoard(counters);
      _activeId = counters.isEmpty ? null : counters.first.id;
      _volumeKeys = volumeKeys;
      _loaded = true;
    });
    await VolumeKeys.setEnabled(volumeKeys);
  }

  /// 每一次改动都立刻写盘。
  ///
  /// 不做防抖、不等退出 —— 计数器经常是「按完就锁屏塞回口袋」，
  /// 进程随时可能被系统回收。晚写一秒就可能少记一个。
  /// 写的是一小串 JSON，频繁写的代价可以忽略。
  Future<void> _persist() => CounterStore.save(_board.counters);

  void _bump(String id, int delta) {
    final changed = _board.bump(id, delta);
    if (!changed) return; // 比如已经是 0 还要减 —— 不震动，免得给出「按上了」的假反馈
    final counter = _board.byId(id);
    // 刚好数到目标时给一记更实的震动 —— 不用盯着屏幕也知道够数了
    if (counter != null && counter.isComplete && counter.value - counter.step < counter.target) {
      HapticFeedback.mediumImpact();
    } else {
      HapticFeedback.selectionClick();
    }
    setState(() => _activeId = id);
    _persist();
  }

  void _onVolumeKey(int delta) {
    if (!_volumeKeys || !mounted) return;
    final id = _activeId ?? (_board.length > 0 ? _board.counters.first.id : null);
    if (id == null) return;
    _bump(id, delta);
  }

  Future<void> _toggleVolumeKeys() async {
    final next = !_volumeKeys;
    setState(() => _volumeKeys = next);
    await SettingsStore.setVolumeKeys(next);
    await VolumeKeys.setEnabled(next);
    if (!mounted) return;
    _toast(next
        ? 'Volume keys count the highlighted card'
        : 'Volume keys back to normal');
  }

  void _undo() {
    if (!_board.undo()) return;
    HapticFeedback.lightImpact();
    setState(() {});
    _persist();
  }

  void _addCounter() {
    HapticFeedback.selectionClick();
    final counter = Counter(
      // 用「当前时刻 + 序号」拼 id：同一毫秒里连加两个也不会撞。
      id: 'c${DateTime.now().microsecondsSinceEpoch}-${_board.length}',
      label: 'Counter ${_board.length + 1}',
      // 新卡片自动换个颜色，省得用户还要手动挑
      colorIndex: _board.length % AppColors.tags.length,
    );
    _board.add(counter);
    setState(() => _activeId = counter.id);
    _persist();
  }

  /// 拖动排序。
  ///
  /// 用的是 `onReorderItem` 而不是老的 `onReorder`：后者给的 newIndex 是
  /// 「移除之前」的下标，往下拖时调用方得自己减一 —— 这一处历来最容易错。
  /// 新回调已经替你调好了，直接用。
  void _reorder(int oldIndex, int newIndex) {
    if (!_board.move(oldIndex, newIndex)) return;
    HapticFeedback.selectionClick();
    setState(() {});
    _persist();
  }

  Future<void> _saveSession() async {
    final sessions = await SessionStore.load();
    final next = <Session>[
      Session.snapshot(_board.counters, DateTime.now()),
      ...sessions,
    ];
    await SessionStore.save(next);
    await HapticFeedback.mediumImpact();
    if (!mounted) return;
    _toast('Session saved');
  }

  Future<void> _copySummary() async {
    await Clipboard.setData(
      ClipboardData(text: Summary.forCounters(_board.counters, DateTime.now())),
    );
    await HapticFeedback.mediumImpact();
    if (!mounted) return;
    _toast('Copied to clipboard');
  }

  Future<void> _openHistory() async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(builder: (_) => const HistoryScreen()),
    );
  }

  Future<void> _resetAll() async {
    final ok = await _confirm(
      title: 'Reset all counters?',
      message: 'Every count goes back to zero. You can undo this.',
      confirmLabel: 'Reset all',
    );
    if (ok != true) return;
    if (!_board.resetAll()) return;
    await HapticFeedback.mediumImpact();
    setState(() {});
    await _persist();
  }

  void _toast(String message) {
    ScaffoldMessenger.of(context)
      ..clearSnackBars()
      ..showSnackBar(
        SnackBar(
          content: Text(message),
          duration: const Duration(milliseconds: 1400),
          behavior: SnackBarBehavior.floating,
        ),
      );
  }

  Future<bool?> _confirm({
    required String title,
    required String message,
    required String confirmLabel,
  }) {
    return showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: Text(title, style: const TextStyle(color: AppColors.primaryText)),
        content: Text(message, style: const TextStyle(color: AppColors.mutedText)),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel', style: TextStyle(color: AppColors.mutedText)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(confirmLabel, style: const TextStyle(color: AppColors.danger)),
          ),
        ],
      ),
    );
  }

  Future<void> _edit(Counter counter) async {
    HapticFeedback.lightImpact();
    final result = await showModalBottomSheet<_EditResult>(
      context: context,
      backgroundColor: Colors.transparent,
      isScrollControlled: true,
      builder: (context) => _EditSheet(counter: counter),
    );
    if (result == null) return;

    switch (result.action) {
      case _EditAction.save:
        _board.edit(
          counter.id,
          label: result.label,
          step: result.step,
          target: result.target,
          colorIndex: result.colorIndex,
        );
      case _EditAction.reset:
        _board.resetOne(counter.id);
      case _EditAction.delete:
        _board.remove(counter.id);
        if (_activeId == counter.id) {
          _activeId = _board.length > 0 ? _board.counters.first.id : null;
        }
    }
    setState(() {});
    await _persist();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: AnnotatedRegion<SystemUiOverlayStyle>(
        // 深色底必须配浅色状态栏图标，否则系统图标会黑底黑字看不见。
        value: SystemUiOverlayStyle.light.copyWith(
          statusBarColor: Colors.transparent,
        ),
        child: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: <Color>[AppColors.backgroundTint, AppColors.background],
              stops: <double>[0.0, 0.5],
            ),
          ),
          child: SafeArea(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                _header(),
                Expanded(child: _list()),
                _hint(),
                _actions(),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _header() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 6, 6, 10),
      child: Row(
        children: <Widget>[
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                const Text(
                  'TickPad',
                  style: TextStyle(
                    color: AppColors.primaryText,
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                    letterSpacing: -0.3,
                    height: 1.1,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  _board.length == 1 ? '1 counter' : '${_board.length} counters',
                  style: const TextStyle(
                    color: AppColors.mutedText,
                    fontSize: 12.5,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ),
          // 总和。多个计数器一起用时（比如按品类盘点），总数往往才是最终要的那个。
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: AppColors.border),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                const Text(
                  'Total',
                  style: TextStyle(
                    color: AppColors.mutedText,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  '${_board.total}',
                  style: const TextStyle(
                    color: AppColors.accent,
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                    fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                  ),
                ),
              ],
            ),
          ),
          _menu(),
        ],
      ),
    );
  }

  /// 次要功能收进溢出菜单。
  ///
  /// 底部只留「撤销」和「新建」两个 —— 那是点得最频繁的两个动作，
  /// 塞满一排按钮反而会让人在赶时间时按错。
  Widget _menu() {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.more_vert_rounded, color: AppColors.mutedText),
      color: AppColors.surface,
      onSelected: (value) {
        switch (value) {
          case 'save':
            _saveSession();
          case 'history':
            _openHistory();
          case 'copy':
            _copySummary();
          case 'volume':
            _toggleVolumeKeys();
          case 'reset':
            _resetAll();
        }
      },
      itemBuilder: (context) => <PopupMenuEntry<String>>[
        _menuItem('save', Icons.bookmark_add_outlined, 'Save session'),
        _menuItem('history', Icons.history_rounded, 'History'),
        _menuItem('copy', Icons.copy_rounded, 'Copy summary'),
        const PopupMenuDivider(),
        _menuItem(
          'volume',
          _volumeKeys
              ? Icons.check_box_rounded
              : Icons.check_box_outline_blank_rounded,
          'Volume keys',
        ),
        const PopupMenuDivider(),
        _menuItem('reset', Icons.restart_alt_rounded, 'Reset all',
            danger: true),
      ],
    );
  }

  PopupMenuItem<String> _menuItem(
    String value,
    IconData icon,
    String label, {
    bool danger = false,
  }) {
    final color = danger ? AppColors.danger : AppColors.primaryText;
    return PopupMenuItem<String>(
      value: value,
      height: 44,
      child: Row(
        children: <Widget>[
          Icon(icon, size: 18, color: color),
          const SizedBox(width: 12),
          Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 14,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  Widget _list() {
    if (!_loaded) return const SizedBox.shrink();
    return ReorderableListView(
      padding: const EdgeInsets.fromLTRB(14, 0, 14, 4),
      onReorderItem: _reorder,
      // 默认的拖拽提示会给卡片加一层 Material 阴影，在深色下是一块灰斑。
      // 换成轻微放大，深浅主题下都干净。
      proxyDecorator: (child, index, animation) => AnimatedBuilder(
        animation: animation,
        builder: (context, _) => Transform.scale(
          scale: 1 + 0.04 * Curves.easeOut.transform(animation.value),
          child: Material(color: Colors.transparent, child: child),
        ),
      ),
      children: <Widget>[
        for (final counter in _board.counters)
          CounterCard(
            key: ValueKey<String>(counter.id),
            counter: counter,
            active: _volumeKeys && counter.id == _activeId,
            onIncrement: () => _bump(counter.id, 1),
            onDecrement: () => _bump(counter.id, -1),
            onEdit: () => _edit(counter),
          ),
      ],
    );
  }

  Widget _hint() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 2, 14, 6),
      child: Text(
        _volumeKeys
            ? 'Volume keys count the highlighted card'
            : 'Tap to count · hold to edit · drag to reorder',
        textAlign: TextAlign.center,
        style: TextStyle(
          color: _volumeKeys ? AppColors.accent : AppColors.mutedText,
          fontSize: 11.5,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }

  Widget _actions() {
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 0, 14, 14),
      child: Row(
        children: <Widget>[
          SizedBox(
            height: 48,
            width: 62,
            child: TextButton(
              // 没得撤时置灰，而不是按下去毫无反应
              onPressed: _board.canUndo ? _undo : null,
              style: TextButton.styleFrom(
                backgroundColor: AppColors.surface,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(14),
                  side: const BorderSide(color: AppColors.border),
                ),
                padding: EdgeInsets.zero,
              ),
              child: Icon(
                Icons.undo_rounded,
                size: 20,
                color: _board.canUndo
                    ? AppColors.primaryText
                    : AppColors.mutedText,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: SizedBox(
              height: 48,
              child: FilledButton.icon(
                onPressed: _addCounter,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  foregroundColor: AppColors.onTag,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(14),
                  ),
                ),
                icon: const Icon(Icons.add_rounded, size: 20),
                label: const Text(
                  'New counter',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

enum _EditAction { save, reset, delete }

class _EditResult {
  const _EditResult(
    this.action, {
    this.label,
    this.step,
    this.target,
    this.colorIndex,
  });

  final _EditAction action;
  final String? label;
  final int? step;
  final int? target;
  final int? colorIndex;
}

/// 编辑面板：改名、颜色、步长、目标、清零、删除。
class _EditSheet extends StatefulWidget {
  const _EditSheet({required this.counter});

  final Counter counter;

  @override
  State<_EditSheet> createState() => _EditSheetState();
}

class _EditSheetState extends State<_EditSheet> {
  late final TextEditingController _name =
      TextEditingController(text: widget.counter.label);
  late final TextEditingController _target = TextEditingController(
    text: widget.counter.hasTarget ? '${widget.counter.target}' : '',
  );
  late int _step = widget.counter.step;
  late int _color = widget.counter.colorIndex;

  /// 常见步长。装箱按 6/12 个一提、点票按 5 个一叠 —— 这几档覆盖绝大多数场合。
  static const List<int> _steps = <int>[1, 2, 5, 10, 12];

  @override
  void dispose() {
    _name.dispose();
    _target.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      // 键盘弹起时把面板顶上去，否则输入框会被挡住
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Container(
        padding: const EdgeInsets.fromLTRB(18, 14, 18, 18),
        decoration: const BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              Center(
                child: Container(
                  width: 38,
                  height: 4,
                  decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              const SizedBox(height: 16),
              _field(_name, 'Name', maxLength: 24),
              const SizedBox(height: 14),
              _label('Color'),
              const SizedBox(height: 8),
              Row(
                children: <Widget>[
                  for (var i = 0; i < AppColors.tags.length; i++) ...<Widget>[
                    Expanded(
                      child: GestureDetector(
                        behavior: HitTestBehavior.opaque,
                        onTap: () => setState(() => _color = i),
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 140),
                          height: 40,
                          decoration: BoxDecoration(
                            color: AppColors.tag(i),
                            borderRadius: BorderRadius.circular(11),
                            border: Border.all(
                              color: i == _color
                                  ? AppColors.primaryText
                                  : Colors.transparent,
                              width: 2,
                            ),
                          ),
                          child: i == _color
                              ? const Icon(Icons.check_rounded,
                                  size: 18, color: AppColors.onTag)
                              : null,
                        ),
                      ),
                    ),
                    if (i != AppColors.tags.length - 1)
                      const SizedBox(width: 8),
                  ],
                ],
              ),
              const SizedBox(height: 14),
              _label('Step'),
              const SizedBox(height: 8),
              Row(
                children: <Widget>[
                  for (final s in _steps) ...<Widget>[
                    Expanded(
                      child: GestureDetector(
                        behavior: HitTestBehavior.opaque,
                        onTap: () => setState(() => _step = s),
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 140),
                          height: 42,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: s == _step
                                ? AppColors.tag(_color)
                                : AppColors.surfacePressed,
                            borderRadius: BorderRadius.circular(11),
                          ),
                          child: Text(
                            '+$s',
                            style: TextStyle(
                              color: s == _step
                                  ? AppColors.onTag
                                  : AppColors.primaryText,
                              fontSize: 13.5,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                      ),
                    ),
                    if (s != _steps.last) const SizedBox(width: 8),
                  ],
                ],
              ),
              const SizedBox(height: 14),
              _field(
                _target,
                'Target (leave empty for none)',
                keyboardType: TextInputType.number,
                maxLength: 7,
              ),
              const SizedBox(height: 18),
              SizedBox(
                height: 48,
                child: FilledButton(
                  onPressed: () => Navigator.pop(
                    context,
                    _EditResult(
                      _EditAction.save,
                      label: _name.text,
                      step: _step,
                      // 空着或填了非法值 → 0，即「取消目标」
                      target: int.tryParse(_target.text.trim()) ?? 0,
                      colorIndex: _color,
                    ),
                  ),
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.tag(_color),
                    foregroundColor: AppColors.onTag,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(14),
                    ),
                  ),
                  child: const Text(
                    'Save',
                    style: TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Row(
                children: <Widget>[
                  Expanded(
                    child: TextButton(
                      onPressed: () => Navigator.pop(
                        context,
                        const _EditResult(_EditAction.reset),
                      ),
                      child: const Text(
                        'Reset to 0',
                        style: TextStyle(
                          color: AppColors.mutedText,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ),
                  Expanded(
                    child: TextButton(
                      onPressed: () => Navigator.pop(
                        context,
                        const _EditResult(_EditAction.delete),
                      ),
                      child: const Text(
                        'Delete',
                        style: TextStyle(
                          color: AppColors.danger,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _label(String text) => Text(
        text,
        style: const TextStyle(
          color: AppColors.mutedText,
          fontSize: 12.5,
          fontWeight: FontWeight.w600,
        ),
      );

  Widget _field(
    TextEditingController controller,
    String label, {
    TextInputType? keyboardType,
    int? maxLength,
  }) {
    return TextField(
      controller: controller,
      keyboardType: keyboardType,
      maxLength: maxLength,
      style: const TextStyle(
        color: AppColors.primaryText,
        fontSize: 16,
        fontWeight: FontWeight.w600,
      ),
      decoration: InputDecoration(
        labelText: label,
        counterText: '',
        labelStyle: const TextStyle(color: AppColors.mutedText, fontSize: 14),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: const BorderSide(color: AppColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: AppColors.tag(_color)),
        ),
      ),
    );
  }
}
