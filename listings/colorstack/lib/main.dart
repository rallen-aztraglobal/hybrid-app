import 'package:flutter/material.dart';

import 'app.dart';
import 'push/push_service.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // 初始化 Firebase（收推送用）；未配置/失败静默降级，不阻断启动。
  await PushService.instance.initFirebase();
  runApp(const ColorStackApp());
}
