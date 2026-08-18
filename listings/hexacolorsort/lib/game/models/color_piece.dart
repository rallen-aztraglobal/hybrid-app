import 'package:flutter/foundation.dart';

/// An immutable single color block. [colorId] indexes into the active
/// color palette (see `kColorPalette`).
@immutable
class ColorPiece {
  final int colorId;

  const ColorPiece(this.colorId);

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is ColorPiece && other.colorId == colorId);

  @override
  int get hashCode => colorId.hashCode;

  @override
  String toString() => 'ColorPiece($colorId)';
}
