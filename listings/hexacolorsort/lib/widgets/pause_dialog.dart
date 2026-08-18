import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/theme/app_theme.dart';
import '../game/logic/game_controller.dart';

class PauseDialog extends StatefulWidget {
  final GameController controller;
  final VoidCallback onResume;
  final VoidCallback onRestart;
  final VoidCallback onHome;

  const PauseDialog({
    super.key,
    required this.controller,
    required this.onResume,
    required this.onRestart,
    required this.onHome,
  });

  @override
  State<PauseDialog> createState() => _PauseDialogState();
}

class _PauseDialogState extends State<PauseDialog> {
  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    return AlertDialog(
      title: const Text(AppStrings.paused, textAlign: TextAlign.center),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          SwitchListTile(
            title: const Text(AppStrings.sound),
            value: controller.soundService.enabled,
            onChanged: (v) => setState(() => controller.setSoundEnabled(v)),
          ),
          SwitchListTile(
            title: const Text(AppStrings.vibration),
            value: controller.hapticService.enabled,
            onChanged: (v) => setState(() => controller.setVibrationEnabled(v)),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: widget.onHome,
          child: const Text(AppStrings.home),
        ),
        TextButton(
          onPressed: widget.onRestart,
          child: const Text(AppStrings.restart),
        ),
        ElevatedButton(
          onPressed: widget.onResume,
          child: const Text(AppStrings.resume),
        ),
      ],
      backgroundColor: AppTheme.surface,
    );
  }
}
