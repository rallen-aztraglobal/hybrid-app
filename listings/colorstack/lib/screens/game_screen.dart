import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';

import '../models/game_stats.dart';
import '../navigation/fade_route.dart';
import '../theme/app_gradients.dart';
import '../widgets/score_header.dart';
import '../widgets/stack_column.dart';
import 'result_screen.dart';

class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen>
    with SingleTickerProviderStateMixin {
  static const int maxMoves = 30;
  static const int columns = 5;
  static const int roundSeconds = 60;

  final List<List<Color>> _stacks = List.generate(columns, (_) => <Color>[]);
  // TODO: Add a colorblind/accessibility mode that pairs colors with
  // shapes, patterns, or Semantics labels so gameplay does not rely on
  // color hue alone.
  final List<Color> _palette = const [
    Color(0xFFF94144),
    Color(0xFFF8961E),
    Color(0xFFF9C74F),
    Color(0xFF43AA8B),
    Color(0xFF5775FF),
    Color(0xFFB74CFF),
  ];

  final Random _random = Random();
  late Color _activeColor;
  late final AnimationController _activeCardController;
  Timer? _timer;

  int _score = 0;
  int _moves = 0;
  int _secondsLeft = roundSeconds;
  bool _isFinished = false;

  @override
  void initState() {
    super.initState();
    _activeColor = _nextColor();
    _activeCardController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 450),
      lowerBound: 0.92,
      upperBound: 1.0,
    )..repeat(reverse: true);

    _timer = Timer.periodic(const Duration(seconds: 1), _handleTick);
  }

  @override
  void dispose() {
    _timer?.cancel();
    _activeCardController.dispose();
    super.dispose();
  }

  Color _nextColor() => _palette[_random.nextInt(_palette.length)];

  void _handleTick(Timer timer) {
    if (!mounted || _isFinished) {
      timer.cancel();
      return;
    }

    if (_secondsLeft <= 0) {
      _finishGame();
      return;
    }

    setState(() {
      _secondsLeft--;
    });

    if (_secondsLeft == 0) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _finishGame());
    }
  }

  void _placeColor(int index) {
    if (_isFinished) {
      return;
    }

    setState(() {
      final stack = _stacks[index];
      final bool matchesTop = stack.isNotEmpty && stack.last == _activeColor;

      stack.add(_activeColor);
      _moves++;
      _score += matchesTop ? 3 : 1;

      if (_moves < maxMoves) {
        _activeColor = _nextColor();
      }
    });

    if (_moves >= maxMoves) {
      _finishGame();
    }
  }

  void _finishGame() {
    if (_isFinished || !mounted) {
      return;
    }

    _isFinished = true;
    _timer?.cancel();

    final bestScore = GameStats.updateBestScore(_score);

    Navigator.of(context).pushReplacement(
      createFadeRoute(ResultScreen(score: _score, bestScore: bestScore)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final double progress = _moves / maxMoves;

    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppGradients.game),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              children: [
                ScoreHeader(score: _score, secondsLeft: _secondsLeft),
                const SizedBox(height: 12),
                ClipRRect(
                  borderRadius: BorderRadius.circular(99),
                  child: LinearProgressIndicator(
                    value: progress,
                    minHeight: 10,
                    color: Colors.lightBlueAccent,
                    backgroundColor: Colors.white24,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Moves: $_moves / $maxMoves',
                  style: const TextStyle(color: Colors.white70),
                ),
                const SizedBox(height: 12),
                ScaleTransition(
                  scale: _activeCardController,
                  child: Container(
                    width: 160,
                    height: 64,
                    decoration: BoxDecoration(
                      color: _activeColor,
                      borderRadius: BorderRadius.circular(16),
                      boxShadow: const [
                        BoxShadow(
                          color: Colors.black26,
                          blurRadius: 14,
                          offset: Offset(0, 8),
                        ),
                      ],
                    ),
                    alignment: Alignment.center,
                    child: const Text(
                      'Tap a Stack',
                      style: TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 18),
                Expanded(
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: List.generate(columns, (index) {
                      return Expanded(
                        child: GestureDetector(
                          key: Key('stack-$index'),
                          onTap: () => _placeColor(index),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 4),
                            child: StackColumn(blocks: _stacks[index]),
                          ),
                        ),
                      );
                    }),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
