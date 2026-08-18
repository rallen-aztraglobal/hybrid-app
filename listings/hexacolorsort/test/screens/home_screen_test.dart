import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:hexa_color_sort/core/constants/app_strings.dart';
import 'package:hexa_color_sort/core/theme/app_theme.dart';
import 'package:hexa_color_sort/screens/home_screen.dart';

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets(
    'home screen shows the title and Play navigates to the game screen',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(theme: AppTheme.themeData, home: const HomeScreen()),
      );
      await tester.pumpAndSettle();

      expect(find.text(AppStrings.appName), findsOneWidget);
      expect(find.text(AppStrings.play), findsOneWidget);

      await tester.tap(find.text(AppStrings.play));
      await tester.pumpAndSettle();

      expect(find.text(AppStrings.score), findsOneWidget);
    },
  );
}
