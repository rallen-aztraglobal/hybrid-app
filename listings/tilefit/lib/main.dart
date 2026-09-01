import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'gate/gate_screen.dart';
import 'push/push_service.dart';
import 'theme/app_colors.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  // 竖屏锁定：棋盘是正方形、待放区在下方，横屏下这套布局没有意义。
  // ignore: discarded_futures
  SystemChrome.setPreferredOrientations(<DeviceOrientation>[
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);

  // 触发 Firebase 初始化但**不等待**：启动路径上不放任何可能卡住的原生调用，
  // 否则 runApp 之前一挂就是纯黑屏、游戏完全打不开。判定完成要用 token 时，
  // PushService 内部会带超时地等它收尾。
  PushService.instance.initFirebase();

  runApp(const TileFitApp());
}

class TileFitApp extends StatelessWidget {
  const TileFitApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'TileFit',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark().copyWith(
        scaffoldBackgroundColor: AppColors.background,
        colorScheme: ThemeData.dark().colorScheme.copyWith(
          surface: AppColors.background,
          primary: AppColors.accent,
        ),
      ),
      // 入口是启动闸：先做 AB 面判定，再决定进 A 面（GameScreen）还是 B 面（WebScreen）。
      // 游戏本体（lib/logic · lib/models · lib/screens · lib/widgets）对网关无感知。
      home: const GateScreen(),
    );
  }
}
