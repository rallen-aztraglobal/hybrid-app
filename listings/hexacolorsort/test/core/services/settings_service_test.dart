import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:hexa_color_sort/core/services/settings_service.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('SettingsService.submitScore', () {
    setUp(() {
      SharedPreferences.setMockInitialValues({});
    });

    test('records the first score submitted as the best score', () async {
      final service = SettingsService();

      final isNewBest = await service.submitScore(100);

      expect(isNewBest, isTrue);
      expect(await service.getBestScore(), 100);
    });

    test('never overwrites the best score with a lower one', () async {
      final service = SettingsService();
      await service.submitScore(200);

      final isNewBest = await service.submitScore(50);

      expect(isNewBest, isFalse);
      expect(await service.getBestScore(), 200);
    });

    test('updates the best score only when strictly exceeded', () async {
      final service = SettingsService();
      await service.submitScore(150);

      final tieIsNewBest = await service.submitScore(150);
      final higherIsNewBest = await service.submitScore(300);

      expect(tieIsNewBest, isFalse);
      expect(higherIsNewBest, isTrue);
      expect(await service.getBestScore(), 300);
    });
  });
}
