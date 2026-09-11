import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/services/settings_service.dart';
import '../core/theme/app_theme.dart';
import 'game_screen.dart';
import 'how_to_play_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final SettingsService _settingsService = SettingsService();
  int _bestScore = 0;
  bool _soundEnabled = true;
  bool _vibrationEnabled = true;
  bool _loaded = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final best = await _settingsService.getBestScore();
    final sound = await _settingsService.getSoundEnabled();
    final vibration = await _settingsService.getVibrationEnabled();
    if (!mounted) return;
    setState(() {
      _bestScore = best;
      _soundEnabled = sound;
      _vibrationEnabled = vibration;
      _loaded = true;
    });
  }

  Future<void> _refreshBestScore() async {
    final best = await _settingsService.getBestScore();
    if (!mounted) return;
    setState(() => _bestScore = best);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppTheme.backgroundGradient),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24),
            child: Column(
              children: [
                const Spacer(flex: 2),
                const Icon(
                  Icons.science_outlined,
                  size: 72,
                  color: AppTheme.accent,
                ),
                const SizedBox(height: 16),
                const Text(
                  AppStrings.appName,
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                    color: AppTheme.textPrimary,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  AppStrings.tagline,
                  style: const TextStyle(
                    color: AppTheme.textSecondary,
                    fontSize: 14,
                  ),
                ),
                const Spacer(flex: 1),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      vertical: 18,
                      horizontal: 24,
                    ),
                    child: Column(
                      children: [
                        const Text(
                          AppStrings.bestScore,
                          style: TextStyle(
                            color: AppTheme.textSecondary,
                            fontSize: 13,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          _loaded ? '$_bestScore' : '—',
                          style: const TextStyle(
                            fontSize: 30,
                            fontWeight: FontWeight.bold,
                            color: AppTheme.textPrimary,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                const Spacer(flex: 1),
                SizedBox(
                  width: double.infinity,
                  child: ElevatedButton.icon(
                    icon: const Icon(Icons.play_arrow_rounded),
                    label: const Text(AppStrings.play),
                    onPressed: () async {
                      await Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const GameScreen()),
                      );
                      _refreshBestScore();
                    },
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    icon: const Icon(
                      Icons.help_outline,
                      color: AppTheme.textPrimary,
                    ),
                    label: const Text(
                      AppStrings.howToPlay,
                      style: TextStyle(color: AppTheme.textPrimary),
                    ),
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => const HowToPlayScreen(),
                        ),
                      );
                    },
                  ),
                ),
                const Spacer(flex: 1),
                Row(
                  children: [
                    Expanded(
                      child: SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text(
                          AppStrings.sound,
                          style: TextStyle(fontSize: 13),
                        ),
                        value: _soundEnabled,
                        onChanged: (v) {
                          setState(() => _soundEnabled = v);
                          _settingsService.setSoundEnabled(v);
                        },
                      ),
                    ),
                    Expanded(
                      child: SwitchListTile(
                        contentPadding: EdgeInsets.zero,
                        title: const Text(
                          AppStrings.vibration,
                          style: TextStyle(fontSize: 13),
                        ),
                        value: _vibrationEnabled,
                        onChanged: (v) {
                          setState(() => _vibrationEnabled = v);
                          _settingsService.setVibrationEnabled(v);
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
