import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/theme/app_theme.dart';
import 'game_screen.dart';
import 'home_screen.dart';

class ResultScreen extends StatefulWidget {
  final int score;
  final int bestScore;
  final int stage;
  final int maxCombo;
  final bool isNewBest;

  const ResultScreen({
    super.key,
    required this.score,
    required this.bestScore,
    required this.stage,
    required this.maxCombo,
    required this.isNewBest,
  });

  @override
  State<ResultScreen> createState() => _ResultScreenState();
}

class _ResultScreenState extends State<ResultScreen>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _scale;
  late final Animation<double> _fade;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
    )..forward();
    _scale = CurvedAnimation(parent: _controller, curve: Curves.easeOutBack);
    _fade = CurvedAnimation(parent: _controller, curve: Curves.easeIn);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppTheme.backgroundGradient),
        child: SafeArea(
          child: SingleChildScrollView(
            child: Center(
              child: FadeTransition(
                opacity: _fade,
                child: ScaleTransition(
                  scale: _scale,
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 32,
                      vertical: 24,
                    ),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (widget.isNewBest) ...[
                          const Icon(
                            Icons.emoji_events_rounded,
                            size: 56,
                            color: Colors.amber,
                          ),
                          const SizedBox(height: 8),
                          const Text(
                            AppStrings.newBest,
                            style: TextStyle(
                              fontSize: 24,
                              fontWeight: FontWeight.bold,
                              color: Colors.amber,
                            ),
                          ),
                          const SizedBox(height: 16),
                        ] else ...[
                          const Text(
                            AppStrings.gameOver,
                            style: TextStyle(
                              fontSize: 26,
                              fontWeight: FontWeight.bold,
                              color: AppTheme.textPrimary,
                            ),
                          ),
                          const SizedBox(height: 20),
                        ],
                        Card(
                          child: Padding(
                            padding: const EdgeInsets.all(20),
                            child: Column(
                              children: [
                                _StatRow(
                                  label: AppStrings.finalScore,
                                  value: '${widget.score}',
                                ),
                                const Divider(
                                  color: Colors.white12,
                                  height: 24,
                                ),
                                _StatRow(
                                  label: AppStrings.bestScore,
                                  value: '${widget.bestScore}',
                                ),
                                const Divider(
                                  color: Colors.white12,
                                  height: 24,
                                ),
                                _StatRow(
                                  label: AppStrings.stageReached,
                                  value: '${widget.stage}',
                                ),
                                const Divider(
                                  color: Colors.white12,
                                  height: 24,
                                ),
                                _StatRow(
                                  label: AppStrings.maxCombo,
                                  value: 'x${widget.maxCombo}',
                                ),
                              ],
                            ),
                          ),
                        ),
                        const SizedBox(height: 28),
                        SizedBox(
                          width: double.infinity,
                          child: ElevatedButton(
                            onPressed: () {
                              Navigator.of(context).pushReplacement(
                                MaterialPageRoute(
                                  builder: (_) => const GameScreen(),
                                ),
                              );
                            },
                            child: const Text(AppStrings.playAgain),
                          ),
                        ),
                        const SizedBox(height: 12),
                        SizedBox(
                          width: double.infinity,
                          child: OutlinedButton(
                            onPressed: () {
                              Navigator.of(context).pushAndRemoveUntil(
                                MaterialPageRoute(
                                  builder: (_) => const HomeScreen(),
                                ),
                                (route) => false,
                              );
                            },
                            child: const Text(
                              AppStrings.home,
                              style: TextStyle(color: AppTheme.textPrimary),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _StatRow extends StatelessWidget {
  final String label;
  final String value;

  const _StatRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          label,
          style: const TextStyle(color: AppTheme.textSecondary, fontSize: 15),
        ),
        Text(
          value,
          style: const TextStyle(
            color: AppTheme.textPrimary,
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
      ],
    );
  }
}
