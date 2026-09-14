import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'gate/gate_screen.dart';
import 'push/push_service.dart';
import 'theme/app_colors.dart';
import 'tracking/tracking_service.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  SystemChrome.setSystemUIOverlayStyle(const SystemUiOverlayStyle(
    statusBarColor: Colors.transparent,
    statusBarIconBrightness: Brightness.dark,
    systemNavigationBarColor: AppColors.background,
    systemNavigationBarIconBrightness: Brightness.dark,
  ));

  // 启动路径上不放任何可能卡住的原生调用 —— runApp 之前一挂就是纯黑屏、App 完全打不开。
  // 两个都故意不 await：需要结果时各自内部带超时地等。
  // ignore: discarded_futures
  PushService.instance.initFirebase();
  // ignore: discarded_futures
  TrackingService.instance.init();

  runApp(const EdgeLoopApp());
}

class EdgeLoopApp extends StatelessWidget {
  const EdgeLoopApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'EdgeLoop',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        scaffoldBackgroundColor: AppColors.background,
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.accent,
          brightness: Brightness.light,
        ),
      ),
      home: const GateScreen(),
    );
  }
}
