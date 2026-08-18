import 'package:flutter/foundation.dart';

import 'game_state.dart';

/// A single undo checkpoint: the full immutable game state captured right
/// before a move (and any clears it triggers) was applied. Restoring it is
/// therefore correct even when the move caused a clear/combo.
@immutable
class MoveRecord {
  final GameState previousState;

  const MoveRecord(this.previousState);
}
