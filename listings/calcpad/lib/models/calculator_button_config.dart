import 'package:flutter/widgets.dart';

/// The visual/semantic category of a calculator button, used to decide its
/// color scheme.
enum CalculatorButtonType { number, operatorButton, function }

/// Static description of a single calculator button: its label, category
/// and how many grid columns it should span.
@immutable
class CalculatorButtonConfig {
  const CalculatorButtonConfig({
    required this.label,
    required this.type,
    this.flex = 1,
  });

  final String label;
  final CalculatorButtonType type;
  final int flex;
}
