import 'package:flutter/services.dart';

/// Thin wrapper around platform haptics, gated by the user's vibration
/// preference so callers never need to check the setting themselves.
class HapticService {
  bool enabled;

  HapticService({this.enabled = true});

  void selection() {
    if (enabled) HapticFeedback.selectionClick();
  }

  void success() {
    if (enabled) HapticFeedback.mediumImpact();
  }

  void error() {
    if (enabled) HapticFeedback.vibrate();
  }
}
