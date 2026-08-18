import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'app.dart';
import 'push/push_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);
  // 触发 Firebase 初始化但**不等待**：启动路径上不放任何可能卡住的原生调用，
  // 否则 runApp 之前一挂就是纯黑屏、游戏打不开。判定完成要用 token 时，
  // PushService 内部会带超时地等它收尾。
  PushService.instance.initFirebase();
  runApp(const HexaColorSortApp());
}
