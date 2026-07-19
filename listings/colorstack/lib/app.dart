import 'package:flutter/material.dart';

import 'gate/gate_screen.dart';

class ColorStackApp extends StatelessWidget {
  const ColorStackApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'Color Stack',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF4C5BFF)),
      ),
      // 入口改为启动闸：先做 AB 面判定，再决定进 A 面(HomeScreen)还是 B 面(WebScreen)。
      // HomeScreen 及其下游游戏代码保持零改动。
      home: const GateScreen(),
    );
  }
}
