import 'package:flutter/material.dart';

import '../logic/calculator_logic.dart';
import '../models/calculator_button_config.dart';
import '../theme/app_colors.dart';
import '../widgets/calculator_button.dart';

/// The main (and only) screen of the calculator app: a display area on top
/// and a grid of buttons below.
class CalculatorScreen extends StatefulWidget {
  const CalculatorScreen({super.key});

  @override
  State<CalculatorScreen> createState() => _CalculatorScreenState();
}

class _CalculatorScreenState extends State<CalculatorScreen> {
  final CalculatorLogic _logic = CalculatorLogic();

  static const List<List<CalculatorButtonConfig>> _buttonRows = [
    [
      CalculatorButtonConfig(label: 'AC', type: CalculatorButtonType.function),
      CalculatorButtonConfig(label: '+/-', type: CalculatorButtonType.function),
      CalculatorButtonConfig(label: '%', type: CalculatorButtonType.function),
      CalculatorButtonConfig(
        label: '÷',
        type: CalculatorButtonType.operatorButton,
      ),
    ],
    [
      CalculatorButtonConfig(label: '7', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '8', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '9', type: CalculatorButtonType.number),
      CalculatorButtonConfig(
        label: '×',
        type: CalculatorButtonType.operatorButton,
      ),
    ],
    [
      CalculatorButtonConfig(label: '4', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '5', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '6', type: CalculatorButtonType.number),
      CalculatorButtonConfig(
        label: '-',
        type: CalculatorButtonType.operatorButton,
      ),
    ],
    [
      CalculatorButtonConfig(label: '1', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '2', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '3', type: CalculatorButtonType.number),
      CalculatorButtonConfig(
        label: '+',
        type: CalculatorButtonType.operatorButton,
      ),
    ],
    [
      CalculatorButtonConfig(label: '⌫', type: CalculatorButtonType.function),
      CalculatorButtonConfig(label: '0', type: CalculatorButtonType.number),
      CalculatorButtonConfig(label: '.', type: CalculatorButtonType.number),
      CalculatorButtonConfig(
        label: '=',
        type: CalculatorButtonType.operatorButton,
      ),
    ],
  ];

  void _onButtonPressed(String label) {
    setState(() {
      switch (label) {
        case 'AC':
          _logic.clearAll();
          break;
        case '+/-':
          _logic.toggleSign();
          break;
        case '%':
          _logic.inputPercent();
          break;
        case '⌫':
          _logic.delete();
          break;
        case '.':
          _logic.inputDot();
          break;
        case '+':
        case '-':
        case '×':
        case '÷':
          _logic.setOperator(label);
          break;
        case '=':
          _logic.calculateEquals();
          break;
        default:
          _logic.inputDigit(label);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: Column(
            children: [
              Expanded(
                flex: 3,
                child: _DisplayArea(
                  equationPreview: _logic.equationPreview,
                  display: _logic.display,
                ),
              ),
              Expanded(
                flex: 5,
                child: Column(
                  children: [
                    for (final row in _buttonRows)
                      Expanded(
                        child: Row(
                          children: [
                            for (final config in row)
                              CalculatorButton(
                                key: ValueKey(config.label),
                                config: config,
                                isActive:
                                    config.type ==
                                        CalculatorButtonType.operatorButton &&
                                    config.label == _logic.pendingOperator,
                                onPressed: () => _onButtonPressed(config.label),
                              ),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DisplayArea extends StatelessWidget {
  const _DisplayArea({required this.equationPreview, required this.display});

  final String equationPreview;
  final String display;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.end,
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          Text(
            equationPreview,
            key: const ValueKey('equationPreview'),
            style: const TextStyle(
              color: AppColors.equationPreviewText,
              fontSize: 22,
            ),
            maxLines: 1,
          ),
          const SizedBox(height: 8),
          FittedBox(
            fit: BoxFit.scaleDown,
            alignment: Alignment.centerRight,
            child: Text(
              display,
              key: const ValueKey('mainDisplay'),
              style: const TextStyle(
                color: AppColors.displayText,
                fontSize: 64,
                fontWeight: FontWeight.w300,
              ),
              maxLines: 1,
            ),
          ),
        ],
      ),
    );
  }
}
