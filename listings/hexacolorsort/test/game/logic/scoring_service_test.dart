import 'package:flutter_test/flutter_test.dart';
import 'package:hexa_color_sort/game/logic/scoring_service.dart';

void main() {
  group('ScoringService.scoreForCombo', () {
    test('awards 100 points for the first combo hit', () {
      expect(ScoringService.scoreForCombo(1), 100);
    });

    test('awards 200 points for the second consecutive combo hit', () {
      expect(ScoringService.scoreForCombo(2), 200);
    });

    test('awards 300 points for the third consecutive combo hit', () {
      expect(ScoringService.scoreForCombo(3), 300);
    });

    test('keeps scaling linearly for later combo hits', () {
      expect(ScoringService.scoreForCombo(5), 500);
    });
  });
}
