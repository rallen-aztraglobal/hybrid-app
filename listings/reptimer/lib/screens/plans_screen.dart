import 'package:flutter/material.dart';

import '../logic/interval_plan.dart';
import '../storage/plan_store.dart';
import '../theme/app_colors.dart';
import 'run_screen.dart';

/// A 面本体的首屏：方案列表。
class PlansScreen extends StatefulWidget {
  const PlansScreen({super.key});

  @override
  State<PlansScreen> createState() => _PlansScreenState();
}

class _PlansScreenState extends State<PlansScreen> {
  List<IntervalPlan> _plans = <IntervalPlan>[];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final plans = await PlanStore.load();
    if (!mounted) return;
    setState(() {
      _plans = plans;
      _loading = false;
    });
  }

  void _persist() {
    // 不 await：写盘失败也不该打断用户，store 内部已兜底。
    // ignore: discarded_futures
    PlanStore.save(_plans);
  }

  Future<void> _edit({IntervalPlan? existing, int? index}) async {
    final result = await showModalBottomSheet<IntervalPlan>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (_) => _PlanEditor(
        initial: existing ??
            const IntervalPlan(
              name: '',
              prepareSeconds: 10,
              workSeconds: 30,
              restSeconds: 15,
              rounds: 8,
            ),
      ),
    );
    if (result == null) return;
    setState(() {
      if (index == null) {
        _plans = <IntervalPlan>[..._plans, result];
      } else {
        final next = <IntervalPlan>[..._plans];
        next[index] = result;
        _plans = next;
      }
    });
    _persist();
  }

  void _delete(int index) {
    final removed = _plans[index];
    setState(() {
      final next = <IntervalPlan>[..._plans]..removeAt(index);
      _plans = next;
    });
    _persist();
    // 删除给撤销，而不是删之前弹「确定吗」——
    // 确认框每次都要点，撤销只在真删错时才用到。
    ScaffoldMessenger.of(context)
      ..clearSnackBars()
      ..showSnackBar(SnackBar(
        content: Text('Deleted "${removed.name}"'),
        action: SnackBarAction(
          label: 'Undo',
          textColor: AppColors.accent,
          onPressed: () {
            setState(() {
              final next = <IntervalPlan>[..._plans];
              next.insert(index.clamp(0, next.length), removed);
              _plans = next;
            });
            _persist();
          },
        ),
      ));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Padding(
              padding: EdgeInsets.fromLTRB(20, 18, 20, 6),
              child: Text(
                'RepTimer',
                style: TextStyle(
                  color: AppColors.primaryText,
                  fontSize: 26,
                  fontWeight: FontWeight.w700,
                  letterSpacing: -0.3,
                ),
              ),
            ),
            Expanded(
              child: _loading
                  ? const Center(
                      child: CircularProgressIndicator(color: AppColors.accent))
                  : _plans.isEmpty
                      ? const _Empty()
                      : ListView.builder(
                          padding: const EdgeInsets.fromLTRB(16, 8, 16, 96),
                          itemCount: _plans.length,
                          itemBuilder: (_, i) => _PlanCard(
                            plan: _plans[i],
                            onStart: () => Navigator.of(context).push(
                              MaterialPageRoute<void>(
                                builder: (_) => RunScreen(plan: _plans[i]),
                              ),
                            ),
                            onEdit: () => _edit(existing: _plans[i], index: i),
                            onDelete: () => _delete(i),
                          ),
                        ),
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        backgroundColor: AppColors.accent,
        foregroundColor: AppColors.background,
        onPressed: () => _edit(),
        icon: const Icon(Icons.add),
        label: const Text('New', style: TextStyle(fontWeight: FontWeight.w700)),
      ),
    );
  }
}

class _Empty extends StatelessWidget {
  const _Empty();

  @override
  Widget build(BuildContext context) => const Center(
        child: Padding(
          padding: EdgeInsets.symmetric(horizontal: 40),
          child: Text(
            'No timers yet.\nTap New to add one.',
            textAlign: TextAlign.center,
            style: TextStyle(color: AppColors.mutedText, height: 1.6),
          ),
        ),
      );
}

class _PlanCard extends StatelessWidget {
  const _PlanCard({
    required this.plan,
    required this.onStart,
    required this.onEdit,
    required this.onDelete,
  });

  final IntervalPlan plan;
  final VoidCallback onStart;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        children: [
          // 整张卡片都是「开始」的点击区 —— 这是最常做的动作，
          // 不该缩在一个小按钮里（手上出汗、隔着一段距离，小目标很难点）。
          InkWell(
            onTap: onStart,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(18, 16, 12, 12),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          plan.name,
                          style: const TextStyle(
                            color: AppColors.primaryText,
                            fontSize: 17,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          '${plan.workSeconds}s work · ${plan.restSeconds}s rest · '
                          '${plan.rounds} rounds'
                          '${plan.cycles > 1 ? " × ${plan.cycles}" : ""}',
                          style: const TextStyle(
                              color: AppColors.mutedText, fontSize: 12.5),
                        ),
                      ],
                    ),
                  ),
                  Container(
                    width: 42,
                    height: 42,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: AppColors.accent.withValues(alpha: 0.14),
                      shape: BoxShape.circle,
                    ),
                    child: const Icon(Icons.play_arrow_rounded,
                        color: AppColors.accent, size: 26),
                  ),
                ],
              ),
            ),
          ),
          Divider(height: 1, color: AppColors.border),
          Row(
            children: [
              Expanded(
                child: TextButton.icon(
                  onPressed: onEdit,
                  icon: const Icon(Icons.tune, size: 16),
                  label: const Text('Edit'),
                  style: TextButton.styleFrom(
                      foregroundColor: AppColors.mutedText),
                ),
              ),
              Container(width: 1, height: 22, color: AppColors.border),
              Expanded(
                child: TextButton.icon(
                  onPressed: onDelete,
                  icon: const Icon(Icons.delete_outline, size: 16),
                  label: const Text('Delete'),
                  style: TextButton.styleFrom(
                      foregroundColor: AppColors.mutedText),
                ),
              ),
            ],
          ),
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Text(
              'Total ${formatSeconds(plan.totalSeconds)}',
              style: const TextStyle(color: AppColors.mutedText, fontSize: 11.5),
            ),
          ),
        ],
      ),
    );
  }
}

/// 新建/编辑方案的底部面板。
class _PlanEditor extends StatefulWidget {
  const _PlanEditor({required this.initial});
  final IntervalPlan initial;

  @override
  State<_PlanEditor> createState() => _PlanEditorState();
}

class _PlanEditorState extends State<_PlanEditor> {
  late final TextEditingController _name =
      TextEditingController(text: widget.initial.name);
  late IntervalPlan _draft = widget.initial;

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final runnable = _draft.isRunnable;
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Container(
        decoration: const BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.vertical(top: Radius.circular(22)),
        ),
        padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Center(
                child: Container(
                  width: 36,
                  height: 4,
                  margin: const EdgeInsets.only(bottom: 16),
                  decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              TextField(
                controller: _name,
                style: const TextStyle(
                    color: AppColors.primaryText,
                    fontSize: 18,
                    fontWeight: FontWeight.w700),
                decoration: InputDecoration(
                  hintText: 'Timer name',
                  hintStyle: const TextStyle(color: AppColors.mutedText),
                  filled: true,
                  fillColor: AppColors.surfaceHigh,
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(12),
                    borderSide: BorderSide.none,
                  ),
                ),
                textCapitalization: TextCapitalization.sentences,
              ),
              const SizedBox(height: 14),
              _Field(
                label: 'Prepare',
                value: _draft.prepareSeconds,
                suffix: 's',
                step: 5,
                onChanged: (v) =>
                    setState(() => _draft = _draft.copyWith(prepareSeconds: v)),
              ),
              _Field(
                label: 'Work',
                value: _draft.workSeconds,
                suffix: 's',
                step: 5,
                min: 1,
                color: AppColors.work,
                onChanged: (v) =>
                    setState(() => _draft = _draft.copyWith(workSeconds: v)),
              ),
              _Field(
                label: 'Rest',
                value: _draft.restSeconds,
                suffix: 's',
                step: 5,
                color: AppColors.rest,
                onChanged: (v) =>
                    setState(() => _draft = _draft.copyWith(restSeconds: v)),
              ),
              _Field(
                label: 'Rounds',
                value: _draft.rounds,
                step: 1,
                min: 1,
                onChanged: (v) =>
                    setState(() => _draft = _draft.copyWith(rounds: v)),
              ),
              _Field(
                label: 'Cycles',
                value: _draft.cycles,
                step: 1,
                min: 1,
                onChanged: (v) =>
                    setState(() => _draft = _draft.copyWith(cycles: v)),
              ),
              if (_draft.cycles > 1)
                _Field(
                  label: 'Cycle rest',
                  value: _draft.cycleRestSeconds,
                  suffix: 's',
                  step: 15,
                  color: AppColors.cycleRest,
                  onChanged: (v) => setState(
                      () => _draft = _draft.copyWith(cycleRestSeconds: v)),
                ),
              const SizedBox(height: 10),
              Text(
                'Total ${formatSeconds(_draft.totalSeconds)} · '
                '${_draft.totalRounds} rounds',
                textAlign: TextAlign.center,
                style: const TextStyle(
                    color: AppColors.mutedText, fontSize: 12.5),
              ),
              const SizedBox(height: 14),
              FilledButton(
                onPressed: runnable
                    ? () {
                        final name = _name.text.trim();
                        Navigator.of(context).pop(_draft.copyWith(
                          // 名字留空就给个默认值，而不是不让保存 ——
                          // 拦着不让存比自动兜底烦人得多。
                          name: name.isEmpty ? 'Timer' : name,
                        ));
                      }
                    : null,
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  foregroundColor: AppColors.background,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
                child: const Text('Save',
                    style: TextStyle(fontWeight: FontWeight.w700)),
              ),
              if (!runnable)
                const Padding(
                  padding: EdgeInsets.only(top: 8),
                  child: Text(
                    'Work must be at least 1 second and there must be at least 1 round.',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: AppColors.mutedText, fontSize: 11.5),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Field extends StatelessWidget {
  const _Field({
    required this.label,
    required this.value,
    required this.step,
    required this.onChanged,
    this.suffix = '',
    this.min = 0,
    this.color,
  });

  final String label;
  final int value;
  final int step;
  final String suffix;
  final int min;
  final Color? color;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    Widget btn(IconData icon, int delta) => GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: () {
            final next = value + delta;
            onChanged(next < min ? min : next);
          },
          child: Container(
            width: 40,
            height: 36,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: AppColors.surfaceHigh,
              borderRadius: BorderRadius.circular(9),
            ),
            child: Icon(icon, size: 17, color: color ?? AppColors.accent),
          ),
        );

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Container(
            width: 8,
            height: 8,
            margin: const EdgeInsets.only(right: 10),
            decoration: BoxDecoration(
              color: color ?? Colors.transparent,
              shape: BoxShape.circle,
            ),
          ),
          Expanded(
            child: Text(label,
                style: const TextStyle(
                    color: AppColors.primaryText,
                    fontWeight: FontWeight.w600,
                    fontSize: 14)),
          ),
          btn(Icons.remove, -step),
          Container(
            width: 62,
            alignment: Alignment.center,
            child: Text(
              '$value$suffix',
              style: const TextStyle(
                color: AppColors.primaryText,
                fontSize: 16,
                fontWeight: FontWeight.w700,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),
          ),
          btn(Icons.add, step),
        ],
      ),
    );
  }
}
