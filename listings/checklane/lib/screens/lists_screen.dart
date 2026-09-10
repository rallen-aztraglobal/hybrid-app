import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/book.dart';
import '../logic/checklist.dart';
import '../storage/checklist_store.dart';
import '../theme/app_colors.dart';
import 'list_detail_screen.dart';

/// A 面：清单总览。对 AB 面网关完全无感知 —— 这里没有一处 import 到 `lib/gate/`。
class ListsScreen extends StatefulWidget {
  const ListsScreen({super.key});

  @override
  State<ListsScreen> createState() => _ListsScreenState();
}

class _ListsScreenState extends State<ListsScreen> {
  ChecklistBook _book = ChecklistBook(ChecklistStore.defaults());
  bool _loaded = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final lists = await ChecklistStore.load();
    if (!mounted) return;
    setState(() {
      _book = ChecklistBook(lists);
      _loaded = true;
    });
  }

  /// 每一次改动都立刻写盘。写的是一小串 JSON，频繁写的代价可以忽略，
  /// 而清单是用户唯一的数据，晚写一步就可能丢。
  Future<void> _persist() => ChecklistStore.save(_book.lists);

  Future<void> _open(Checklist list) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => ListDetailScreen(
          book: _book,
          listId: list.id,
          onChanged: () {
            setState(() {});
            _persist();
          },
        ),
      ),
    );
    // 详情页里改过东西，回来要刷新卡片上的进度
    if (mounted) setState(() {});
  }

  Future<void> _newList() async {
    final title = await _askText(title: 'New checklist', hint: 'Name');
    if (title == null || title.trim().isEmpty) return;
    final list = Checklist(
      id: 'l${DateTime.now().microsecondsSinceEpoch}',
      title: title.trim(),
    );
    _book.addList(list);
    setState(() {});
    await _persist();
    if (mounted) await _open(list);
  }

  Future<void> _rename(Checklist list) async {
    final title = await _askText(
      title: 'Rename',
      hint: 'Name',
      initial: list.title,
    );
    if (title == null) return;
    if (!_book.renameList(list.id, title)) return;
    setState(() {});
    await _persist();
  }

  Future<void> _duplicate(Checklist list) async {
    final seed = DateTime.now().microsecondsSinceEpoch;
    _book.duplicateList(list.id, 'l$seed', 'i$seed');
    HapticFeedback.selectionClick();
    setState(() {});
    await _persist();
  }

  Future<void> _delete(Checklist list) async {
    final ok = await _confirm(
      title: 'Delete “${list.title}”?',
      message: 'The list and its items go away. You can undo this.',
      confirmLabel: 'Delete',
    );
    if (ok != true) return;
    if (!_book.removeList(list.id)) return;
    await HapticFeedback.mediumImpact();
    setState(() {});
    await _persist();
  }

  Future<void> _reset(Checklist list) async {
    if (!_book.resetList(list.id)) return;
    await HapticFeedback.mediumImpact();
    setState(() {});
    await _persist();
  }

  void _undo() {
    if (!_book.undo()) return;
    HapticFeedback.lightImpact();
    setState(() {});
    _persist();
  }

  void _reorder(int oldIndex, int newIndex) {
    if (!_book.moveList(oldIndex, newIndex)) return;
    HapticFeedback.selectionClick();
    setState(() {});
    _persist();
  }

  Future<String?> _askText({
    required String title,
    required String hint,
    String? initial,
  }) {
    final controller = TextEditingController(text: initial);
    return showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: Text(title, style: const TextStyle(color: AppColors.primaryText)),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 40,
          style: const TextStyle(color: AppColors.primaryText),
          decoration: InputDecoration(
            hintText: hint,
            counterText: '',
            hintStyle: const TextStyle(color: AppColors.mutedText),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: AppColors.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(10),
              borderSide: const BorderSide(color: AppColors.accent),
            ),
          ),
          onSubmitted: (value) => Navigator.pop(context, value),
        ),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel',
                style: TextStyle(color: AppColors.mutedText)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, controller.text),
            child: const Text('OK', style: TextStyle(color: AppColors.accent)),
          ),
        ],
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
        content:
            Text(message, style: const TextStyle(color: AppColors.mutedText)),
        actions: <Widget>[
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel',
                style: TextStyle(color: AppColors.mutedText)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child:
                Text(confirmLabel, style: const TextStyle(color: AppColors.danger)),
          ),
        ],
      ),
    );
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
                Expanded(child: _body()),
                _actions(),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _header() {
    final done = _book.lists.where((l) => l.isComplete).length;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
      child: Row(
        children: <Widget>[
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                const Text(
                  'CheckLane',
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
                  _book.length == 1
                      ? '1 checklist'
                      : '${_book.length} checklists',
                  style: const TextStyle(
                    color: AppColors.mutedText,
                    fontSize: 12.5,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ),
          if (_book.length > 0)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: AppColors.surface,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: AppColors.border),
              ),
              child: Text(
                '$done / ${_book.length} done',
                style: TextStyle(
                  color: done == _book.length
                      ? AppColors.complete
                      : AppColors.mutedText,
                  fontSize: 12.5,
                  fontWeight: FontWeight.w700,
                  fontFeatures: const <FontFeature>[
                    FontFeature.tabularFigures(),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _body() {
    if (!_loaded) return const SizedBox.shrink();
    if (_book.length == 0) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.symmetric(horizontal: 40),
          child: Text(
            'No checklists yet.\n'
            'Make one for something you do more than once — '
            'packing, opening up, handover.',
            textAlign: TextAlign.center,
            style: TextStyle(
              color: AppColors.mutedText,
              fontSize: 13.5,
              height: 1.5,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      );
    }

    return ReorderableListView(
      padding: const EdgeInsets.fromLTRB(14, 0, 14, 8),
      onReorderItem: _reorder,
      proxyDecorator: (child, index, animation) => AnimatedBuilder(
        animation: animation,
        builder: (context, _) => Transform.scale(
          scale: 1 + 0.04 * Curves.easeOut.transform(animation.value),
          child: Material(color: Colors.transparent, child: child),
        ),
      ),
      children: <Widget>[
        for (final list in _book.lists)
          _ListCard(
            key: ValueKey<String>(list.id),
            list: list,
            onTap: () => _open(list),
            onReset: () => _reset(list),
            onRename: () => _rename(list),
            onDuplicate: () => _duplicate(list),
            onDelete: () => _delete(list),
          ),
      ],
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
              onPressed: _book.canUndo ? _undo : null,
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
                color:
                    _book.canUndo ? AppColors.primaryText : AppColors.mutedText,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: SizedBox(
              height: 48,
              child: FilledButton.icon(
                onPressed: _newList,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  foregroundColor: AppColors.onAccent,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(14),
                  ),
                ),
                icon: const Icon(Icons.add_rounded, size: 20),
                label: const Text(
                  'New checklist',
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

/// 总览里的一张清单卡片。
class _ListCard extends StatelessWidget {
  const _ListCard({
    super.key,
    required this.list,
    required this.onTap,
    required this.onReset,
    required this.onRename,
    required this.onDuplicate,
    required this.onDelete,
  });

  final Checklist list;
  final VoidCallback onTap;
  final VoidCallback onReset;
  final VoidCallback onRename;
  final VoidCallback onDuplicate;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final complete = list.isComplete;
    final accent = complete ? AppColors.complete : AppColors.accent;

    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.fromLTRB(14, 12, 4, 12),
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: complete ? AppColors.complete : AppColors.border,
          ),
        ),
        child: Row(
          children: <Widget>[
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Flexible(
                        child: Text(
                          list.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            color: AppColors.primaryText,
                            fontSize: 16,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      if (complete) ...<Widget>[
                        const SizedBox(width: 6),
                        const Icon(Icons.check_circle_rounded,
                            size: 15, color: AppColors.complete),
                      ],
                    ],
                  ),
                  const SizedBox(height: 6),
                  Row(
                    children: <Widget>[
                      Text(
                        list.total == 0
                            ? 'Empty'
                            : '${list.doneCount} / ${list.total}',
                        style: TextStyle(
                          color: complete ? AppColors.complete : AppColors.mutedText,
                          fontSize: 12.5,
                          fontWeight: FontWeight.w600,
                          fontFeatures: const <FontFeature>[
                            FontFeature.tabularFigures(),
                          ],
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(3),
                          child: TweenAnimationBuilder<double>(
                            tween: Tween<double>(begin: 0, end: list.progress),
                            duration: const Duration(milliseconds: 220),
                            curve: Curves.easeOut,
                            builder: (context, value, _) =>
                                LinearProgressIndicator(
                              value: value,
                              minHeight: 4,
                              backgroundColor: AppColors.surfacePressed,
                              valueColor:
                                  AlwaysStoppedAnimation<Color>(accent),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert_rounded,
                  color: AppColors.mutedText, size: 20),
              color: AppColors.surface,
              onSelected: (value) {
                switch (value) {
                  case 'reset':
                    onReset();
                  case 'rename':
                    onRename();
                  case 'duplicate':
                    onDuplicate();
                  case 'delete':
                    onDelete();
                }
              },
              itemBuilder: (context) => <PopupMenuEntry<String>>[
                _item('reset', Icons.restart_alt_rounded, 'Reset ticks'),
                _item('rename', Icons.edit_outlined, 'Rename'),
                _item('duplicate', Icons.copy_all_outlined, 'Duplicate'),
                const PopupMenuDivider(),
                _item('delete', Icons.delete_outline_rounded, 'Delete',
                    danger: true),
              ],
            ),
          ],
        ),
      ),
    );
  }

  PopupMenuItem<String> _item(
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
}
