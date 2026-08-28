import 'package:flutter/material.dart';

import '../models/calculator_button_config.dart';
import '../theme/app_colors.dart';

/// A single calculator button. Colors depend on [config.type]; [isActive]
/// highlights an operator button that is currently pending (e.g. after
/// pressing "+", before the second operand is entered).
class CalculatorButton extends StatelessWidget {
  const CalculatorButton({
    super.key,
    required this.config,
    required this.onPressed,
    this.isActive = false,
  });

  final CalculatorButtonConfig config;
  final VoidCallback onPressed;
  final bool isActive;

  @override
  Widget build(BuildContext context) {
    final Color backgroundColor;
    final Color textColor;

    switch (config.type) {
      case CalculatorButtonType.number:
        backgroundColor = AppColors.numberButton;
        textColor = AppColors.numberButtonText;
        break;
      case CalculatorButtonType.operatorButton:
        backgroundColor = isActive
            ? AppColors.operatorButtonActive
            : AppColors.operatorButton;
        textColor = isActive
            ? AppColors.operatorButtonActiveText
            : AppColors.operatorButtonText;
        break;
      case CalculatorButtonType.function:
        backgroundColor = AppColors.functionButton;
        textColor = AppColors.functionButtonText;
        break;
    }

    // A large, self-clamping border radius makes square buttons look like
    // circles and wide buttons (like "0") look like pills, without relying
    // on a fixed aspect ratio that could overflow on narrow/short screens.
    final shape = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(999),
    );

    return Expanded(
      flex: config.flex,
      child: Padding(
        padding: const EdgeInsets.all(6),
        child: Material(
          color: backgroundColor,
          shape: shape,
          child: InkWell(
            onTap: onPressed,
            splashColor: AppColors.splash,
            customBorder: shape,
            child: Center(
              child: Text(
                config.label,
                style: TextStyle(
                  color: textColor,
                  fontSize: 28,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
