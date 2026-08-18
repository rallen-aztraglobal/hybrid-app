import 'dart:async';
import 'dart:math';

import 'package:flutter/foundation.dart';

import '../../core/constants/game_constants.dart';
import '../../core/services/haptic_service.dart';
import '../../core/services/settings_service.dart';
import '../../core/services/sound_service.dart';
import '../models/color_piece.dart';
import '../models/game_state.dart';
import '../models/move_record.dart';
import '../models/stack_model.dart';
import 'board_generator.dart';
import 'deadlock_detector.dart';
import 'game_event.dart';
import 'move_validator.dart';
import 'scoring_service.dart';

/// Owns the authoritative [GameState] and all transitions into/out of it.
/// Widgets only read [state] and call the public methods here; none of the
/// rule logic lives in the UI layer.
class GameController extends ChangeNotifier {
  GameController({
    int? seed,
    SettingsService? settingsService,
    HapticService? hapticService,
    SoundService? soundService,
    DateTime Function()? clock,
    Duration? moveAnimationDuration,
    Duration? clearAnimationDuration,
    @visibleForTesting List<StackModel>? initialStacksOverride,
  }) : _seed = seed ?? DateTime.now().millisecondsSinceEpoch,
       _settingsService = settingsService ?? SettingsService(),
       hapticService = hapticService ?? HapticService(),
       soundService = soundService ?? SoundService(),
       _clock = clock ?? DateTime.now,
       _moveAnimationDuration =
           moveAnimationDuration ?? GameConstants.moveAnimationDuration,
       _clearAnimationDuration =
           clearAnimationDuration ?? GameConstants.clearAnimationDuration {
    _state = initialStacksOverride != null
        ? GameState(
            stacks: initialStacksOverride,
            score: 0,
            bestScore: 0,
            stage: 1,
            colorCount: GameConstants.initialColorCount,
          )
        : _buildInitialState(
            stage: 1,
            colorCount: GameConstants.initialColorCount,
            bestScore: 0,
          );
    unawaited(_loadPreferences());
  }

  final int _seed;
  int _seedOffset = 0;

  final SettingsService _settingsService;
  final HapticService hapticService;
  final SoundService soundService;
  final DateTime Function() _clock;
  final Duration _moveAnimationDuration;
  final Duration _clearAnimationDuration;

  late GameState _state;
  GameState get state => _state;

  bool _isAnimating = false;
  bool get isAnimating => _isAnimating;

  bool _isPaused = false;
  bool get isPaused => _isPaused;

  DateTime? _lastClearTime;
  int _comboStreak = 0;

  final StreamController<GameEvent> _eventController =
      StreamController<GameEvent>.broadcast();
  Stream<GameEvent> get events => _eventController.stream;

  bool get canUndo => !_isAnimating && !_isPaused && _state.lastMove != null;

  GameState _buildInitialState({
    required int stage,
    required int colorCount,
    required int bestScore,
  }) {
    final stacks = BoardGenerator.generate(
      seed: _seed + _seedOffset,
      colorCount: colorCount,
    );
    return GameState(
      stacks: stacks,
      score: 0,
      bestScore: bestScore,
      stage: stage,
      colorCount: colorCount,
    );
  }

  Future<void> _loadPreferences() async {
    final best = await _settingsService.getBestScore();
    hapticService.enabled = await _settingsService.getVibrationEnabled();
    soundService.enabled = await _settingsService.getSoundEnabled();
    _state = _state.copyWith(bestScore: best);
    notifyListeners();
  }

  void setPaused(bool paused) {
    if (_isPaused == paused) return;
    _isPaused = paused;
    notifyListeners();
  }

  Future<void> setVibrationEnabled(bool value) async {
    hapticService.enabled = value;
    await _settingsService.setVibrationEnabled(value);
    notifyListeners();
  }

  Future<void> setSoundEnabled(bool value) async {
    soundService.enabled = value;
    await _settingsService.setSoundEnabled(value);
    notifyListeners();
  }

  /// Handles a tap on the stack at [index]: selects it, deselects it,
  /// or attempts a move from the currently selected stack.
  Future<void> selectStack(int index) async {
    if (_isPaused || _isAnimating || _state.isGameOver) return;

    final selected = _state.selectedStackIndex;
    if (selected == index) {
      _state = _state.copyWith(clearSelection: true);
      notifyListeners();
      return;
    }

    if (selected == null) {
      if (_state.stacks[index].isEmpty) return;
      _state = _state.copyWith(selectedStackIndex: index);
      hapticService.selection();
      soundService.playSelect();
      notifyListeners();
      return;
    }

    final from = _state.stacks[selected];
    final to = _state.stacks[index];
    final evaluation = MoveValidator.evaluate(from: from, to: to);
    if (!evaluation.isLegal) {
      hapticService.error();
      soundService.playIllegal();
      _eventController.add(IllegalMoveEvent(index));
      return;
    }

    await _performMove(selected, index, evaluation.moveCount);
  }

  Future<void> _performMove(int fromIndex, int toIndex, int moveCount) async {
    _isAnimating = true;
    final snapshotBeforeMove = _state.asUndoSnapshot;
    final colorId = _state.stacks[fromIndex].topColorId!;

    _state = _state.copyWith(clearSelection: true);
    notifyListeners();

    _eventController.add(
      MoveEvent(
        fromIndex: fromIndex,
        toIndex: toIndex,
        count: moveCount,
        colorId: colorId,
      ),
    );
    soundService.playMove();
    await Future.delayed(_moveAnimationDuration);

    var stacks = [..._state.stacks];
    final movingPieces = stacks[fromIndex].pieces.sublist(
      stacks[fromIndex].pieces.length - moveCount,
    );
    stacks[fromIndex] = stacks[fromIndex].removeTop(moveCount);
    stacks[toIndex] = stacks[toIndex].addPieces(movingPieces);

    _state = _state.copyWith(
      stacks: stacks,
      lastMove: MoveRecord(snapshotBeforeMove),
    );
    hapticService.success();
    notifyListeners();

    await _resolveClears(stacks);

    _isAnimating = false;
    _checkGameOver();
    notifyListeners();
  }

  Future<void> _resolveClears(List<StackModel> stacks) async {
    while (true) {
      final clearIndex = stacks.indexWhere(
        (s) => s.topRunLength >= GameConstants.clearThreshold,
      );
      if (clearIndex == -1) break;

      final colorId = stacks[clearIndex].topColorId!;
      final now = _clock();
      if (_lastClearTime != null &&
          now.difference(_lastClearTime!) <= GameConstants.comboWindow) {
        _comboStreak++;
      } else {
        _comboStreak = 1;
      }
      _lastClearTime = now;
      final awarded = ScoringService.scoreForCombo(_comboStreak);

      _eventController.add(
        ClearEvent(
          stackIndex: clearIndex,
          colorId: colorId,
          comboStreak: _comboStreak,
          scoreAwarded: awarded,
        ),
      );
      soundService.playClear();
      await Future.delayed(_clearAnimationDuration);

      stacks = [...stacks];
      stacks[clearIndex] = stacks[clearIndex].removeTop(
        GameConstants.clearThreshold,
      );

      _state = _state.copyWith(
        stacks: stacks,
        score: _state.score + awarded,
        comboCount: _comboStreak,
        maxCombo: max(_comboStreak, _state.maxCombo),
        piecesClearedTotal:
            _state.piecesClearedTotal + GameConstants.clearThreshold,
      );
      notifyListeners();

      _maybeAdvanceStage(stacks);
      stacks = _state.stacks;
    }
  }

  void _maybeAdvanceStage(List<StackModel> currentStacks) {
    final threshold = GameConstants.piecesPerStage * _state.stage;
    if (_state.piecesClearedTotal < threshold) return;

    final nextStage = _state.stage + 1;
    final action = nextStage % 3;
    var stacks = [...currentStacks];
    var colorCount = _state.colorCount;
    final random = Random(_seed + nextStage * 104729);

    switch (action) {
      case 0:
        if (colorCount < GameConstants.maxColorCount) colorCount++;
      case 1:
        final candidates = [
          for (var i = 0; i < stacks.length; i++)
            if (!stacks[i].isFull) i,
        ];
        if (candidates.isNotEmpty) {
          final target = candidates[random.nextInt(candidates.length)];
          final colorId = random.nextInt(colorCount);
          stacks[target] = stacks[target].addPieces([ColorPiece(colorId)]);
        }
      default:
        final emptyIndices = [
          for (var i = 0; i < stacks.length; i++)
            if (stacks[i].isEmpty) i,
        ];
        if (emptyIndices.isNotEmpty) {
          final target = emptyIndices[random.nextInt(emptyIndices.length)];
          final colorId = random.nextInt(colorCount);
          final candidate = [...stacks];
          candidate[target] = candidate[target].addPieces([
            ColorPiece(colorId),
          ]);
          if (!DeadlockDetector.isDeadlocked(candidate)) {
            stacks = candidate;
          }
        }
    }

    if (DeadlockDetector.isDeadlocked(stacks)) {
      stacks = currentStacks;
    }

    _state = _state.copyWith(
      stage: nextStage,
      colorCount: colorCount,
      stacks: stacks,
    );
    _eventController.add(StageUpEvent(nextStage));
  }

  void _checkGameOver() {
    if (!_state.isGameOver && DeadlockDetector.isDeadlocked(_state.stacks)) {
      _state = _state.copyWith(isGameOver: true);
      unawaited(_finalizeGameOver());
    }
  }

  Future<void> _finalizeGameOver() async {
    final isNewBest = await _settingsService.submitScore(_state.score);
    if (isNewBest) {
      _state = _state.copyWith(bestScore: _state.score);
    }
    _eventController.add(GameOverEvent(isNewBest: isNewBest));
    notifyListeners();
  }

  void undo() {
    if (!canUndo) return;
    _state = _state.lastMove!.previousState;
    notifyListeners();
  }

  void restart() {
    _seedOffset++;
    _isAnimating = false;
    _isPaused = false;
    _comboStreak = 0;
    _lastClearTime = null;
    final bestScore = _state.bestScore;
    _state = _buildInitialState(
      stage: 1,
      colorCount: GameConstants.initialColorCount,
      bestScore: bestScore,
    );
    notifyListeners();
  }

  @override
  void dispose() {
    _eventController.close();
    super.dispose();
  }
}
