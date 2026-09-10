import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../logic/session.dart';
import '../logic/summary.dart';
import '../storage/session_store.dart';
import '../theme/app_colors.dart';

/// 会话历史。存下来的每一轮计数。
class HistoryScreen extends StatefulWidget {
  const HistoryScreen({super.key});

  @override
  State<HistoryScreen> createState() => _HistoryScreenState();
}

class _HistoryScreenState extends State<HistoryScreen> {
  List<Session> _sessions = <Session>[];
  bool _loaded = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final sessions = await SessionStore.load();
    if (!mounted) return;
    setState(() {
      _sessions = sessions;
      _loaded = true;
    });
  }

  Future<void> _delete(int index) async {
    setState(() => _sessions = <Session>[
          for (var i = 0; i < _sessions.length; i++)
            if (i != index) _sessions[i],
        ]);
    await SessionStore.save(_sessions);
  }

  Future<void> _copy(Session session) async {
    await Clipboard.setData(ClipboardData(text: Summary.forSession(session)));
    await HapticFeedback.mediumImpact();
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..clearSnackBars()
      ..showSnackBar(
        const SnackBar(
          content: Text('Copied to clipboard'),
          duration: Duration(milliseconds: 1200),
          behavior: SnackBarBehavior.floating,
        ),
      );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        backgroundColor: AppColors.background,
        surfaceTintColor: Colors.transparent,
        foregroundColor: AppColors.primaryText,
        title: const Text(
          'History',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
        ),
      ),
      body: !_loaded
          ? const SizedBox.shrink()
          : _sessions.isEmpty
              ? _empty()
              : ListView.builder(
                  padding: const EdgeInsets.fromLTRB(14, 4, 14, 20),
                  itemCount: _sessions.length,
                  itemBuilder: (context, i) => _card(_sessions[i], i),
                ),
    );
  }

  Widget _empty() {
    return const Center(
      child: Padding(
        padding: EdgeInsets.symmetric(horizontal: 40),
        child: Text(
          'Saved counts show up here.\n'
          'Use “Save session” before you reset, so the round is not lost.',
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

  Widget _card(Session session, int index) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  Summary.formatDateTime(session.savedAt),
                  style: const TextStyle(
                    color: AppColors.primaryText,
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                  ),
                ),
              ),
              IconButton(
                onPressed: () => _copy(session),
                icon: const Icon(Icons.copy_rounded, size: 18),
                color: AppColors.mutedText,
                tooltip: 'Copy',
              ),
              IconButton(
                onPressed: () => _delete(index),
                icon: const Icon(Icons.delete_outline_rounded, size: 19),
                color: AppColors.mutedText,
                tooltip: 'Delete',
              ),
            ],
          ),
          const SizedBox(height: 4),
          for (final entry in session.entries)
            Padding(
              padding: const EdgeInsets.only(right: 6, bottom: 3),
              child: Row(
                children: <Widget>[
                  Expanded(
                    child: Text(
                      entry.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: AppColors.mutedText,
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
                  Text(
                    '${entry.value}',
                    style: const TextStyle(
                      color: AppColors.primaryText,
                      fontSize: 13.5,
                      fontWeight: FontWeight.w600,
                      fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                    ),
                  ),
                ],
              ),
            ),
          const Divider(height: 14, color: AppColors.border),
          Padding(
            padding: const EdgeInsets.only(right: 6),
            child: Row(
              children: <Widget>[
                const Expanded(
                  child: Text(
                    'Total',
                    style: TextStyle(
                      color: AppColors.mutedText,
                      fontSize: 13,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                Text(
                  '${session.total}',
                  style: const TextStyle(
                    color: AppColors.accent,
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
