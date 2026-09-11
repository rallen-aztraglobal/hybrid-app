import 'package:flutter/foundation.dart';

import 'color_piece.dart';

/// An immutable vertical tray holding pieces bottom-to-top. The last
/// element of [pieces] is the visible top piece.
@immutable
class StackModel {
  final int id;
  final List<ColorPiece> pieces;
  final int capacity;

  const StackModel({
    required this.id,
    required this.pieces,
    required this.capacity,
  });

  bool get isEmpty => pieces.isEmpty;

  bool get isFull => pieces.length >= capacity;

  int get remainingCapacity => capacity - pieces.length;

  ColorPiece? get topPiece => pieces.isEmpty ? null : pieces.last;

  int? get topColorId => topPiece?.colorId;

  /// Number of consecutive same-color pieces counting down from the top.
  int get topRunLength {
    if (pieces.isEmpty) return 0;
    final color = pieces.last.colorId;
    var count = 0;
    for (var i = pieces.length - 1; i >= 0; i--) {
      if (pieces[i].colorId == color) {
        count++;
      } else {
        break;
      }
    }
    return count;
  }

  StackModel copyWith({List<ColorPiece>? pieces}) {
    return StackModel(
      id: id,
      pieces: pieces ?? this.pieces,
      capacity: capacity,
    );
  }

  /// Returns a new stack with the top [count] pieces removed.
  StackModel removeTop(int count) {
    assert(count <= pieces.length);
    return copyWith(pieces: pieces.sublist(0, pieces.length - count));
  }

  /// Returns a new stack with [newPieces] appended on top.
  StackModel addPieces(List<ColorPiece> newPieces) {
    return copyWith(pieces: [...pieces, ...newPieces]);
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is StackModel &&
          other.id == id &&
          other.capacity == capacity &&
          listEquals(other.pieces, pieces));

  @override
  int get hashCode => Object.hash(id, capacity, Object.hashAll(pieces));

  @override
  String toString() => 'StackModel(id: $id, pieces: $pieces)';
}
