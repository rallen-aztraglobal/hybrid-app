import 'package:flutter/material.dart';

import 'glass_info_card.dart';

class ScoreHeader extends StatelessWidget {
  const ScoreHeader({super.key, required this.score, required this.secondsLeft});

  final int score;
  final int secondsLeft;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: GlassInfoCard(
            title: 'Score',
            value: '$score',
            icon: Icons.star_rounded,
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: GlassInfoCard(
            title: 'Time',
            value: '${secondsLeft}s',
            icon: Icons.timer_rounded,
          ),
        ),
      ],
    );
  }
}
