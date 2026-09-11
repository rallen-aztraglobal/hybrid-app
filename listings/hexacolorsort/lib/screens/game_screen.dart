import 'dart:async';

import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/constants/game_constants.dart';
import '../core/theme/app_theme.dart';
import '../game/logic/game_controller.dart';
import '../game/logic/game_event.dart';
import '../widgets/combo_overlay.dart';
import '../widgets/game_board.dart';
import '../widgets/pause_dialog.dart';
import '../widgets/score_header.dart';
import 'result_screen.dart';

class GameScreen extends StatefulWidget {
  const GameScreen({super.key});

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> with WidgetsBindingObserver {
  late final GameController _controller;
  StreamSubscription<GameEvent>? _subscription;
  int _comboDisplay = 0;
  Timer? _comboTimer;
  bool _navigatingToResult = false;
  bool _dimmed = false;
  bool _isNewBest = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _controller = GameController();
    _subscription = _controller.events.listen(_handleEvent);
  }

  void _handleEvent(GameEvent event) {
    if (event is ClearEvent) {
      setState(() => _comboDisplay = event.comboStreak);
      _comboTimer?.cancel();
      _comboTimer = Timer(GameConstants.comboWindow, () {
        if (mounted) setState(() => _comboDisplay = 0);
      });
    } else if (event is GameOverEvent) {
      _isNewBest = event.isNewBest;
      _goToResult();
    }
  }

  Future<void> _goToResult() async {
    if (_navigatingToResult) return;
    _navigatingToResult = true;
    setState(() => _dimmed = true);
    await Future.delayed(const Duration(milliseconds: 400));
    if (!mounted) return;
    final state = _controller.state;
    await Navigator.of(context).pushReplacement(
      MaterialPageRoute(
        builder: (_) => ResultScreen(
          score: state.score,
          bestScore: state.bestScore,
          stage: state.stage,
          maxCombo: state.maxCombo,
          isNewBest: _isNewBest,
        ),
      ),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) {
      _controller.setPaused(true);
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _subscription?.cancel();
    _comboTimer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _openPause() {
    _controller.setPaused(true);
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => PauseDialog(
        controller: _controller,
        onResume: () {
          Navigator.of(context).pop();
          _controller.setPaused(false);
        },
        onRestart: () {
          Navigator.of(context).pop();
          _controller.restart();
          setState(() => _comboDisplay = 0);
        },
        onHome: () {
          Navigator.of(context).pop();
          Navigator.of(context).pop();
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: true,
      child: Scaffold(
        body: Container(
          decoration: const BoxDecoration(
            gradient: AppTheme.backgroundGradient,
          ),
          child: SafeArea(
            child: Stack(
              children: [
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 8,
                  ),
                  child: Column(
                    children: [
                      ListenableBuilder(
                        listenable: _controller,
                        builder: (context, _) {
                          final state = _controller.state;
                          return ScoreHeader(
                            score: state.score,
                            bestScore: state.bestScore,
                            stage: state.stage,
                            onPause: _openPause,
                          );
                        },
                      ),
                      const SizedBox(height: 8),
                      Expanded(child: GameBoard(controller: _controller)),
                      const SizedBox(height: 8),
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          ComboBadge(combo: _comboDisplay),
                          Row(
                            children: [
                              ListenableBuilder(
                                listenable: _controller,
                                builder: (context, _) {
                                  return IconButton.filledTonal(
                                    onPressed: _controller.canUndo
                                        ? _controller.undo
                                        : null,
                                    icon: const Icon(Icons.undo_rounded),
                                    tooltip: AppStrings.undo,
                                  );
                                },
                              ),
                              const SizedBox(width: 8),
                              IconButton.filledTonal(
                                onPressed: () {
                                  _controller.restart();
                                  setState(() => _comboDisplay = 0);
                                },
                                icon: const Icon(Icons.refresh_rounded),
                                tooltip: AppStrings.restart,
                              ),
                            ],
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                IgnorePointer(
                  child: AnimatedOpacity(
                    opacity: _dimmed ? 0.7 : 0,
                    duration: const Duration(milliseconds: 400),
                    child: Container(color: Colors.black),
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
