import 'package:flutter/material.dart';

import '../navigation/fade_route.dart';
import '../theme/app_gradients.dart';
import 'game_screen.dart';
import 'home_screen.dart';

class ResultScreen extends StatelessWidget {
  const ResultScreen({super.key, required this.score, required this.bestScore});

  final int score;
  final int bestScore;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppGradients.result),
        child: SafeArea(
          child: Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(
                    Icons.emoji_events_rounded,
                    size: 88,
                    color: Colors.white,
                  ),
                  const SizedBox(height: 16),
                  const Text(
                    'Round Complete',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 30,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Score: $score',
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 26,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Best: $bestScore',
                    style: const TextStyle(color: Colors.white70, fontSize: 18),
                  ),
                  const SizedBox(height: 34),
                  FilledButton(
                    onPressed: () {
                      Navigator.of(context).pushReplacement(
                        createFadeRoute(const GameScreen()),
                      );
                    },
                    style: FilledButton.styleFrom(
                      minimumSize: const Size(220, 52),
                      backgroundColor: Colors.white,
                      foregroundColor: const Color(0xFF2A4ACC),
                    ),
                    child: const Text(
                      'RESTART',
                      style: TextStyle(fontWeight: FontWeight.w700),
                    ),
                  ),
                  const SizedBox(height: 10),
                  TextButton(
                    onPressed: () {
                      Navigator.of(context).pushAndRemoveUntil(
                        createFadeRoute(const HomeScreen()),
                        (_) => false,
                      );
                    },
                    child: const Text(
                      'Back to Home',
                      style: TextStyle(color: Colors.white),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
