import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/book.dart';
import '../logic/checklist.dart';
import '../theme/app_colors.dart';

/// 一份清单的详情：一条条勾。
///
/// 直接操作总览页传下来的同一个 [ChecklistBook]，改完回调让上层写盘 ——
/// 两页各存一份状态再想办法同步，是这类界面最常见的错误来源。
class ListDetailScreen extends StatefulWidget {
  const ListDetailScreen({
    super.key,
    required this.book,
    required this.listId,
    required this.onChanged,
  });

  final ChecklistBook book;
  final String listId;
  final VoidCallback onChanged;

  @override
  State<ListDetailScreen> createState() => _ListDetailScreenState();
}

class _ListDetailScreenState extends State<ListDetailScreen> {
  final TextEditingController _input = TextEditingController();
  final FocusNode _inputFocus = FocusNode();

  Checklist? get _list => widget.book.byId(widget.listId);

  @override
  void dispose() {
    _input.dispose();
    _inputFocus.dispose();
    super.dispose();
  }

  void _changed() {
    setState(() {});
    widget.onChanged();
  }

  void _toggle(ChecklistItem item) {
    if (!widget.book.toggle(widget.listId, item.id)) return;
    final list = _list;
    // 刚好全部勾完时给一记更实的震动 —— 核对表的意义就在最后那一下
    if (list != null && list.isComplete && item.done) {
      HapticFeedback.mediumImpact();
    } else {
      HapticFeedback.selectionClick();
    }
    _changed();
  }

  void _add() {
    final text = _input.text;
    if (text.trim().isEmpty) return;
    widget.book.addItem(
      widget.listId,
      text,
      'i${DateTime.now().microsecondsSinceEpoch}',
    );
    _input.clear();
    HapticFeedback.selectionClick();
    _changed();
    // 焦点留在输入框：连着录一串条目时不用每条都点一次
    _inputFocus.requestFocus();
  }

  Future<void> _edit(ChecklistItem item) async {
    final controller = TextEditingController(text: item.text);
    final text = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: const Text('Edit item',
            style: TextStyle(color: AppColors.primaryText)),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 80,
          style: const TextStyle(color: AppColors.primaryText),
          decoration: InputDecoration(
            counterText: '',
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
            onPressed: () {
              Navigator.pop(context);
              widget.book.removeItem(widget.listId, item.id);
              _changed();
            },
            child:
                const Text('Delete', style: TextStyle(color: AppColors.danger)),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, controller.text),
            child: const Text('Save', style: TextStyle(color: AppColors.accent)),
          ),
        ],
      ),
    );
    if (text == null) return;
    if (!widget.book.editItem(widget.listId, item.id, text)) return;
    _changed();
  }

  void _reset() {
    if (!widget.book.resetList(widget.listId)) return;
    HapticFeedback.mediumImpact();
    _changed();
  }

  void _undo() {
    if (!widget.book.undo()) return;
    HapticFeedback.lightImpact();
    _changed();
  }

  void _reorder(int oldIndex, int newIndex) {
    if (!widget.book.moveItem(widget.listId, oldIndex, newIndex)) return;
    HapticFeedback.selectionClick();
    _changed();
  }

  @override
  Widget build(BuildContext context) {
    final list = _list;
    // 这份清单在别处被删掉了（撤销回到删除之后的状态）—— 直接退回总览，
    // 而不是留在一个指向空气的页面上
    if (list == null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) Navigator.of(context).maybePop();
      });
      return const Scaffold(backgroundColor: AppColors.background);
    }

    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        surfaceTintColor: Colors.transparent,
        foregroundColor: AppColors.primaryText,
        title: Text(
          list.title,
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
        actions: <Widget>[
          IconButton(
            onPressed: widget.book.canUndo ? _undo : null,
            icon: const Icon(Icons.undo_rounded, size: 20),
            color: AppColors.primaryText,
            disabledColor: AppColors.mutedText,
            tooltip: 'Undo',
          ),
          IconButton(
            onPressed: list.doneCount > 0 ? _reset : null,
            icon: const Icon(Icons.restart_alt_rounded, size: 21),
            color: AppColors.primaryText,
            disabledColor: AppColors.mutedText,
            tooltip: 'Reset ticks',
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: <Widget>[
            _progress(list),
            Expanded(child: _items(list)),
            _composer(),
          ],
        ),
      ),
    );
  }

  Widget _progress(Checklist list) {
    final accent =
        list.isComplete ? AppColors.complete : AppColors.accent;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      child: Row(
        children: <Widget>[
          Text(
            list.total == 0
                ? 'Nothing on this list yet'
                : list.isComplete
                    ? 'All done'
                    : '${list.remaining} left',
            style: TextStyle(
              color: list.isComplete ? AppColors.complete : AppColors.mutedText,
              fontSize: 12.5,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: ClipRRect(
              borderRadius: BorderRadius.circular(3),
              child: TweenAnimationBuilder<double>(
                tween: Tween<double>(begin: 0, end: list.progress),
                duration: const Duration(milliseconds: 220),
                curve: Curves.easeOut,
                builder: (context, value, _) => LinearProgressIndicator(
                  value: value,
                  minHeight: 5,
                  backgroundColor: AppColors.surface,
                  valueColor: AlwaysStoppedAnimation<Color>(accent),
                ),
              ),
            ),
          ),
          const SizedBox(width: 12),
          Text(
            '${list.doneCount} / ${list.total}',
            style: TextStyle(
              color: accent,
              fontSize: 12.5,
              fontWeight: FontWeight.w700,
              fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
  }

  Widget _items(Checklist list) {
    if (list.items.isEmpty) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.symmetric(horizontal: 40),
          child: Text(
            'Add the things you check every time.\n'
            'Tick them off, then hit reset to run the list again.',
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
          scale: 1 + 0.03 * Curves.easeOut.transform(animation.value),
          child: Material(color: Colors.transparent, child: child),
        ),
      ),
      children: <Widget>[
        for (final item in list.items)
          _ItemRow(
            key: ValueKey<String>(item.id),
            item: item,
            onTap: () => _toggle(item),
            onLongPress: () => _edit(item),
          ),
      ],
    );
  }

  /// 底部录入条。固定在键盘上方，录完一条焦点不走 ——
  /// 建清单时是连着敲一串的，每条都要重新点一次输入框会很烦。
  Widget _composer() {
    return Padding(
      padding: EdgeInsets.fromLTRB(
        14,
        4,
        14,
        14 + MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Row(
        children: <Widget>[
          Expanded(
            child: TextField(
              controller: _input,
              focusNode: _inputFocus,
              maxLength: 80,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _add(),
              style: const TextStyle(
                color: AppColors.primaryText,
                fontSize: 15,
                fontWeight: FontWeight.w500,
              ),
              decoration: InputDecoration(
                hintText: 'Add an item',
                counterText: '',
                isDense: true,
                contentPadding:
                    const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
                hintStyle: const TextStyle(color: AppColors.mutedText),
                filled: true,
                fillColor: AppColors.surface,
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(14),
                  borderSide: const BorderSide(color: AppColors.border),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(14),
                  borderSide: const BorderSide(color: AppColors.accent),
                ),
              ),
            ),
          ),
          const SizedBox(width: 10),
          SizedBox(
            height: 48,
            width: 52,
            child: FilledButton(
              onPressed: _add,
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.accent,
                foregroundColor: AppColors.onAccent,
                padding: EdgeInsets.zero,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(14),
                ),
              ),
              child: const Icon(Icons.add_rounded, size: 22),
            ),
          ),
        ],
      ),
    );
  }
}

/// 一条待勾项。整行都是热区 —— 只让小方框可点的话，走路时根本点不准。
class _ItemRow extends StatelessWidget {
  const _ItemRow({
    super.key,
    required this.item,
    required this.onTap,
    required this.onLongPress,
  });

  final ChecklistItem item;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      onLongPress: onLongPress,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: AppColors.border),
        ),
        child: Row(
          children: <Widget>[
            AnimatedContainer(
              duration: const Duration(milliseconds: 140),
              width: 24,
              height: 24,
              decoration: BoxDecoration(
                color: item.done ? AppColors.accent : Colors.transparent,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: item.done ? AppColors.accent : AppColors.mutedText,
                  width: 1.8,
                ),
              ),
              child: item.done
                  ? const Icon(Icons.check_rounded,
                      size: 16, color: AppColors.onAccent)
                  : null,
            ),
            const SizedBox(width: 13),
            Expanded(
              child: AnimatedDefaultTextStyle(
                duration: const Duration(milliseconds: 140),
                style: TextStyle(
                  color: item.done ? AppColors.doneText : AppColors.primaryText,
                  fontSize: 15,
                  fontWeight: FontWeight.w500,
                  height: 1.3,
                  decoration:
                      item.done ? TextDecoration.lineThrough : TextDecoration.none,
                  decorationColor: AppColors.doneText,
                ),
                child: Text(item.text),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
