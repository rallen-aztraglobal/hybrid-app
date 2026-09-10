
import 'difficulty.dart';
import 'generator.dart';
import 'mine_field.dart';

enum GameStatus { ready, playing, won, lost }

/// 点击的含义。手机上没有右键，得给一个显式的模式开关；
/// 长按仍然是另一种动作的快捷方式（挖的时候长按=插旗，反之亦然）。
enum InputMode { dig, flag }

/// 一局扫雷的状态机。纯数据，不依赖 Flutter，便于直接测。
class MineGame {
  MineGame(this.difficulty)
      : _field = null,
        _status = GameStatus.ready,
        _revealed = List<bool>.filled(difficulty.cellCount, false),
        _flagged = List<bool>.filled(difficulty.cellCount, false);

  /// 直接拿一盘现成的雷区开局。**只给测试用。**
  ///
  /// 正式玩法必须走 [MineGame.new] —— 雷区要等首点之后再生成，
  /// 否则「第一下必然安全」根本无从保证。这个构造器绕过了那一步，
  /// 换来的是测试可以断言具体某一格的行为，而不是对着随机盘面碰运气。
  MineGame.withField(this.difficulty, MineField field)
      : assert(
          field.cols == difficulty.cols && field.rows == difficulty.rows,
          '雷区尺寸和难度档对不上',
        ),
        _field = field,
        _status = GameStatus.playing,
        _revealed = List<bool>.filled(difficulty.cellCount, false),
        _flagged = List<bool>.filled(difficulty.cellCount, false);

  final Difficulty difficulty;

  /// 雷区**在第一次挖之后才生成**，这样首点永远安全（见 [MineGenerator.generate]）。
  MineField? _field;

  final List<bool> _revealed;
  final List<bool> _flagged;

  GameStatus _status;

  /// 踩到的那颗雷。败局里要把它和其他雷区分开画。
  int? _explodedIndex;

  /// 这一盘是不是「保证不用猜」的。生成预算用尽时会是 false。
  bool _noGuess = true;

  GameStatus get status => _status;
  int? get explodedIndex => _explodedIndex;
  bool get noGuess => _noGuess;
  int get cellCount => difficulty.cellCount;

  bool isRevealed(int index) => _revealed[index];
  bool isFlagged(int index) => _flagged[index];

  /// 已翻开格子周围的雷数；没翻开返回 null。
  int? valueAt(int index) =>
      _revealed[index] ? _field?.adjacentMines(index) : null;

  /// 是不是雷。开局前（还没布雷）一律 false —— 只在败局展示时才有意义。
  bool isMineAt(int index) => _field?.isMine(index) ?? false;

  int get flagsPlaced => _flagged.where((f) => f).length;

  /// 剩余雷数 = 总雷数 − 已插旗数。插错旗会让它算错，这是扫雷的常规行为。
  int get minesLeft => difficulty.mines - flagsPlaced;

  int get revealedCount => _revealed.where((r) => r).length;

  /// 进度：已翻开的安全格 / 全部安全格。
  double get progress => revealedCount / (cellCount - difficulty.mines);

  /// 挖开一格。返回 true 表示界面需要重绘。
  bool dig(int index) {
    if (_status == GameStatus.won || _status == GameStatus.lost) return false;
    if (_flagged[index]) return false; // 插了旗的格子点不动，防误触
    if (_revealed[index]) return false;

    if (_status == GameStatus.ready) {
      final generated = MineGenerator.generate(difficulty, index);
      _field = generated.field;
      _noGuess = generated.noGuess;
      _status = GameStatus.playing;
    }

    if (_field!.isMine(index)) {
      _explodedIndex = index;
      _status = GameStatus.lost;
      return true;
    }

    _revealCascade(index);
    _checkWin();
    return true;
  }

  /// 插旗 / 取消插旗。
  bool toggleFlag(int index) {
    if (_status == GameStatus.won || _status == GameStatus.lost) return false;
    if (_revealed[index]) return false;
    _flagged[index] = !_flagged[index];
    return true;
  }

  /// 和弦：点一个已翻开的数字，若它周围的旗数正好等于该数字，
  /// 就把周围没插旗的格子一次挖开。
  ///
  /// 这是扫雷手感的一大半 —— 没有它，中后期每一格都要单独点，累得没法玩。
  /// 旗插错了会当场炸，这是应有的代价：和弦是玩家在为自己的判断背书。
  bool chord(int index) {
    if (_status != GameStatus.playing) return false;
    if (!_revealed[index]) return false;
    final field = _field;
    if (field == null) return false;

    final value = field.adjacentMines(index);
    if (value <= 0) return false;

    final neighbours = field.neighboursOf(index);
    var flags = 0;
    final targets = <int>[];
    for (final q in neighbours) {
      if (_flagged[q]) {
        flags++;
      } else if (!_revealed[q]) {
        targets.add(q);
      }
    }
    if (flags != value || targets.isEmpty) return false;

    for (final q in targets) {
      if (field.isMine(q)) {
        _explodedIndex = q;
        _status = GameStatus.lost;
        return true;
      }
    }
    for (final q in targets) {
      _revealCascade(q);
    }
    _checkWin();
    return true;
  }

  void reset() {
    _field = null;
    _status = GameStatus.ready;
    _explodedIndex = null;
    _noGuess = true;
    for (var i = 0; i < cellCount; i++) {
      _revealed[i] = false;
      _flagged[i] = false;
    }
  }

  /// 翻开一格；空格（周围 0 雷）连带把邻居也翻开。
  ///
  /// 写成显式栈而不是递归：最大档 11×16，一次全空展开的递归深度会到上百层，
  /// 在 release 下没问题但没必要冒险。
  void _revealCascade(int start) {
    final field = _field!;
    final stack = <int>[start];
    while (stack.isNotEmpty) {
      final i = stack.removeLast();
      if (_revealed[i] || _flagged[i]) continue;
      _revealed[i] = true;
      if (field.adjacentMines(i) != 0) continue;
      stack.addAll(field.neighboursOf(i));
    }
  }

  void _checkWin() {
    if (revealedCount != cellCount - difficulty.mines) return;
    _status = GameStatus.won;
    // 剩下的必然全是雷，帮玩家把旗补上 —— 通关画面上一片没插完的旗很难看，
    // 而且会让人以为还没打完。
    for (var i = 0; i < cellCount; i++) {
      if (!_revealed[i]) _flagged[i] = true;
    }
  }
}
