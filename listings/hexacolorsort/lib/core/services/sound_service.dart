/// Reserved interface for future sound effects. No audio assets ship with
/// this version, so every call is a no-op; wiring a real audio player
/// later only requires filling in these methods.
class SoundService {
  bool enabled;

  SoundService({this.enabled = false});

  void playSelect() {}

  void playMove() {}

  void playClear() {}

  void playCombo() {}

  void playIllegal() {}
}
