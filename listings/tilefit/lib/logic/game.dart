import 'dart:math';

import '../models/piece.dart';
import 'board.dart';

/// 一局游戏的完整状态：棋盘 + 待放区 + 分数 + 结束判定。
///
/// 纯 Dart，无 Flutter 依赖，随机源可注入种子 —— 整局都能在测试里逐步复现。
class Game {
  Game({int? seed, this.traySize = 3, int boardSize = 8})
    : board = Board(size: boardSize),
      _random = seed == null ? Random() : Random(seed) {
    _refillTray();
  }

  /// 每次发几个方块。
  final int traySize;

  final Board board;
  final Random _random;

  /// 待放区。已用掉的槽位置 null，三个都为 null 时补发一批。
  final List<Piece?> _tray = <Piece?>[];
  List<Piece?> get tray => List<Piece?>.unmodifiable(_tray);

  int _score = 0;
  int get score => _score;

  bool _isOver = false;

  /// 待放区里所有剩下的方块都无处可放。
  bool get isOver => _isOver;

  /// 消除一条线得 10 分；同时消除 n 条按 10·n² 计（2 条 40 分、3 条 90 分），
  /// 让「攒一手同时消多条」明显优于「逐条消」—— 这是这类玩法的乐趣所在。
  static int lineBonus(int lines) => lines <= 0 ? 0 : 10 * lines * lines;

  /// 从待放区第 [index] 个槽取方块，左上角落在 ([row], [col])。
  ///
  /// 放不下（槽是空的、越界、压到已填格）返回 null 且不改动任何状态；
  /// 放得下则落子、结算分数、必要时补发，并在补发后重算结束判定。
  PlacementResult? place(int index, int row, int col) {
    if (_isOver) return null;
    if (index < 0 || index >= _tray.length) return null;
    final piece = _tray[index];
    if (piece == null) return null;
    if (!board.canPlace(piece, row, col)) return null;

    final result = board.place(piece, row, col);
    _tray[index] = null;
    _score += result.cellsPlaced + lineBonus(result.linesCleared);

    if (_tray.every((p) => p == null)) _refillTray();
    _isOver = !_hasAnyMove();

    return result;
  }

  /// 待放区第 [index] 个方块能否落在 ([row], [col])。UI 画落点预览用。
  bool canPlace(int index, int row, int col) {
    if (_isOver) return false;
    if (index < 0 || index >= _tray.length) return false;
    final piece = _tray[index];
    return piece != null && board.canPlace(piece, row, col);
  }

  /// 待放区第 [index] 个方块在棋盘上是否还有落点。UI 据此把放不下的方块画灰。
  bool isPlaceable(int index) {
    if (index < 0 || index >= _tray.length) return false;
    final piece = _tray[index];
    return piece != null && board.hasAnyPlacement(piece);
  }

  void restart() {
    board.clear();
    _score = 0;
    _isOver = false;
    _tray.clear();
    _refillTray();
  }

  bool _hasAnyMove() {
    for (final piece in _tray) {
      if (piece != null && board.hasAnyPlacement(piece)) return true;
    }
    return false;
  }

  void _refillTray() {
    _tray
      ..clear()
      ..addAll(List<Piece?>.generate(traySize, (_) => _randomPiece()));
  }

  /// 按 [Piece.weight] 加权取一个形状。
  Piece _randomPiece() {
    var total = 0;
    for (final piece in kPieceCatalog) {
      total += piece.weight;
    }
    var roll = _random.nextInt(total);
    for (final piece in kPieceCatalog) {
      roll -= piece.weight;
      if (roll < 0) return piece;
    }
    // 权重全为正时走不到这里；兜底避免返回 null。
    return kPieceCatalog.last;
  }
}
