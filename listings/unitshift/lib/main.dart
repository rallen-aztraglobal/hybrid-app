import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'gate/gate_screen.dart';
import 'push/push_service.dart';
import 'theme/app_colors.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  // 竖屏锁定：数字键盘固定在底部、单位列表纵向铺开，横屏下这套布局站不住。
  // ignore: discarded_futures
  SystemChrome.setPreferredOrientations(<DeviceOrientation>[
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);

  // 触发 Firebase 初始化但**不等待**：启动路径上不放任何可能卡住的原生调用，
  // 否则 runApp 之前一挂就是纯黑屏、App 完全打不开。判定完成要用 token 时，
  // PushService 内部会带超时地等它收尾。
  PushService.instance.initFirebase();

  runApp(const UnitShiftApp());
}

class UnitShiftApp extends StatelessWidget {
  const UnitShiftApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'UnitShift',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.light().copyWith(
        scaffoldBackgroundColor: AppColors.background,
        colorScheme: ThemeData.light().colorScheme.copyWith(
          surface: AppColors.background,
          primary: AppColors.accent,
        ),
      ),
      // 入口是启动闸：先做 AB 面判定，再决定进 A 面（ConverterScreen）还是 B 面（WebScreen）。
      // 工具本体（lib/logic · lib/screens · lib/widgets）对网关无感知。
      home: const GateScreen(),
    );
  }
}
