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
  // 初始化 Firebase（收推送用）；未配置/失败静默降级，不阻断启动。
  await PushService.instance.initFirebase();
  runApp(const HexaColorSortApp());
}
