import 'package:flutter/material.dart';

class AppGradients {
  const AppGradients._();

  static const home = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [Color(0xFF1A4DFF), Color(0xFF9A35FF)],
  );

  static const game = LinearGradient(
    begin: Alignment.topCenter,
    end: Alignment.bottomCenter,
    colors: [Color(0xFF111B3A), Color(0xFF283FA7)],
  );

  static const result = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [Color(0xFF0E6FFF), Color(0xFF00C2FF)],
  );
}
