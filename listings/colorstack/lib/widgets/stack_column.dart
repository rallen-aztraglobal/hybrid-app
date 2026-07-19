import 'package:flutter/material.dart';

class StackColumn extends StatelessWidget {
  const StackColumn({super.key, required this.blocks});

  final List<Color> blocks;

  @override
  Widget build(BuildContext context) {
    final shownBlocks = blocks.reversed.take(10).toList();
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: Colors.white24),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.end,
        children: shownBlocks
            .map(
              (color) => AnimatedContainer(
                duration: const Duration(milliseconds: 180),
                margin: const EdgeInsets.only(bottom: 4),
                height: 22,
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(7),
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}
