import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/theme/app_theme.dart';

class ScoreHeader extends StatelessWidget {
  final int score;
  final int bestScore;
  final int stage;
  final VoidCallback onPause;

  const ScoreHeader({
    super.key,
    required this.score,
    required this.bestScore,
    required this.stage,
    required this.onPause,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _StatCard(label: AppStrings.score, value: '$score'),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: _StatCard(label: AppStrings.best, value: '$bestScore'),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: _StatCard(label: AppStrings.stage, value: '$stage'),
        ),
        const SizedBox(width: 8),
        Material(
          color: AppTheme.surface,
          borderRadius: BorderRadius.circular(16),
          child: IconButton(
            icon: const Icon(Icons.pause_rounded, color: AppTheme.textPrimary),
            onPressed: onPause,
            tooltip: 'Pause',
          ),
        ),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  final String label;
  final String value;

  const _StatCard({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 10),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            label,
            style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 2),
          Text(
            value,
            style: const TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.bold,
              color: AppTheme.textPrimary,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ),
    );
  }
}
