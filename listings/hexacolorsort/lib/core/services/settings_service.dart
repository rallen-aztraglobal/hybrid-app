import 'package:shared_preferences/shared_preferences.dart';

/// Wraps local persistence for best score and user preferences. Isolated
/// behind this service so game logic never touches shared_preferences
/// directly and can be tested without a platform channel.
class SettingsService {
  static const String _bestScoreKey = 'best_score';
  static const String _soundEnabledKey = 'sound_enabled';
  static const String _vibrationEnabledKey = 'vibration_enabled';

  Future<int> getBestScore() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_bestScoreKey) ?? 0;
  }

  /// Persists [score] as the new best only if it exceeds the stored value.
  /// Returns true if a new best was recorded.
  Future<bool> submitScore(int score) async {
    final prefs = await SharedPreferences.getInstance();
    final current = prefs.getInt(_bestScoreKey) ?? 0;
    if (score > current) {
      await prefs.setInt(_bestScoreKey, score);
      return true;
    }
    return false;
  }

  Future<bool> getSoundEnabled() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getBool(_soundEnabledKey) ?? true;
  }

  Future<void> setSoundEnabled(bool value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_soundEnabledKey, value);
  }

  Future<bool> getVibrationEnabled() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getBool(_vibrationEnabledKey) ?? true;
  }

  Future<void> setVibrationEnabled(bool value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_vibrationEnabledKey, value);
  }
}
