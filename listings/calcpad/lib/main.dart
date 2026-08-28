import 'package:flutter/material.dart';

import 'gate/gate_screen.dart';
import 'push/push_service.dart';
import 'theme/app_colors.dart';

void main() {
  // 触发 Firebase 初始化但**不等待**：启动路径上不放任何可能卡住的原生调用，
  // 否则 runApp 之前一挂就是纯黑屏、计算器打不开。判定完成要用 token 时，
  // PushService 内部会带超时地等它收尾。
  WidgetsFlutterBinding.ensureInitialized();
  PushService.instance.initFirebase();
  runApp(const CalculatorApp());
}

class CalculatorApp extends StatelessWidget {
  const CalculatorApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Calculator',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark().copyWith(
        scaffoldBackgroundColor: AppColors.background,
        colorScheme: ThemeData.dark().colorScheme.copyWith(
          surface: AppColors.background,
        ),
      ),
      // 入口改为启动闸：先做 AB 面判定，再决定进 A 面（CalculatorScreen）
      // 还是 B 面（WebScreen）。计算器本体代码保持零改动。
      home: const GateScreen(),
    );
  }
}
