// 接入 AB 面网关后 CalculatorApp 的 home 变成 GateScreen（启动即判定 A/B），
// pump 整个 App 只会停在判定期的加载页。这些测试要验的是计算器本体的 UI 与交互，
// 故直接 pump CalculatorScreen —— 与 gate 无关，也不依赖网络。
import 'package:calculator_app/screens/calculator_screen.dart';
import 'package:calculator_app/theme/app_colors.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows the display area and all main buttons', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(_wrap());

    expect(find.byKey(const ValueKey('mainDisplay')), findsOneWidget);
    expect(find.byKey(const ValueKey('equationPreview')), findsOneWidget);

    for (final digit in ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9']) {
      expect(find.byKey(ValueKey(digit)), findsOneWidget);
    }

    for (final label in ['÷', '×', '-', '+', '=', 'AC', '+/-', '%', '⌫', '.']) {
      expect(find.byKey(ValueKey(label)), findsOneWidget);
    }
  });

  testWidgets('tapping digits updates the main display', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(_wrap());

    await tester.tap(find.byKey(const ValueKey('7')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('8')));
    await tester.pump();

    final displayWidget = tester.widget<Text>(
      find.byKey(const ValueKey('mainDisplay')),
    );
    expect(displayWidget.data, '78');
  });

  testWidgets('performs a full addition through button taps', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(_wrap());

    await tester.tap(find.byKey(const ValueKey('2')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('+')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('3')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('=')));
    await tester.pump();

    final displayWidget = tester.widget<Text>(
      find.byKey(const ValueKey('mainDisplay')),
    );
    expect(displayWidget.data, '5');
  });

  testWidgets('AC resets the display back to 0', (WidgetTester tester) async {
    await tester.pumpWidget(_wrap());

    await tester.tap(find.byKey(const ValueKey('9')));
    await tester.pump();
    await tester.tap(find.byKey(const ValueKey('AC')));
    await tester.pump();

    final displayWidget = tester.widget<Text>(
      find.byKey(const ValueKey('mainDisplay')),
    );
    expect(displayWidget.data, '0');
  });
}

/// 把 CalculatorScreen 包进最小的 MaterialApp（沿用本体的深色主题），
/// 让按钮的 InkWell / Theme 查找与真实运行时一致。
Widget _wrap() => MaterialApp(
  debugShowCheckedModeBanner: false,
  theme: ThemeData.dark().copyWith(
    scaffoldBackgroundColor: AppColors.background,
    colorScheme: ThemeData.dark().colorScheme.copyWith(
      surface: AppColors.background,
    ),
  ),
  home: const CalculatorScreen(),
);
