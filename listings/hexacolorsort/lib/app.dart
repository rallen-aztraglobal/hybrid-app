import 'package:flutter/material.dart';

import 'core/constants/app_strings.dart';
import 'core/theme/app_theme.dart';
import 'gate/gate_screen.dart';

class HexaColorSortApp extends StatelessWidget {
  const HexaColorSortApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: AppStrings.appName,
      debugShowCheckedModeBanner: false,
      theme: AppTheme.themeData,
      // 入口改为启动闸：先做 AB 面判定，再决定进 A 面（SplashScreen → HomeScreen）
      // 还是 B 面（WebScreen）。SplashScreen 及其下游游戏代码保持零改动。
      home: const GateScreen(),
    );
  }
}
