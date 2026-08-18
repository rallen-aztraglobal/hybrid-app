import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

import '../gate/gate_config.dart';

/// FCM 推送接入：初始化 Firebase、取 token、随「最近一次 AB 面判定结果」上报后端。
///
/// 后端「上架包推送」强制只发 last_gate_mode='B' 的设备，故注册时必须带上本次判定的 mode，
/// 让服务端准确记录该设备属 A 还是 B 面（见 docs/admin/09-listing.md §6）。
///
/// 全程容错：推送不可用绝不能影响 App 启动或 A/B 呈现——所有失败吞掉即可。
class PushService {
  PushService._();
  static final PushService instance = PushService._();

  bool _firebaseReady = false;

  /// 在 main() 里尽早调用：初始化 Firebase。失败（未配 google-services 等）静默降级。
  Future<void> initFirebase() async {
    try {
      await Firebase.initializeApp();
      _firebaseReady = true;
    } catch (_) {
      _firebaseReady = false;
    }
  }

  /// 拿到 AB 面判定结果后调用：取 FCM token 并带 gateMode 上报后端。
  /// gateMode 传本次判定的 'A' / 'B'。
  Future<void> registerWithGateMode(String gateMode) async {
    if (!_firebaseReady) return;
    try {
      final messaging = FirebaseMessaging.instance;
      // iOS 需用户授权才发；Android 13+ 也需 POST_NOTIFICATIONS。拿不到授权仍尝试取 token。
      await messaging.requestPermission();
      final token = await messaging.getToken();
      if (token == null || token.isEmpty) return;
      await _report(token: token, gateMode: gateMode);
    } catch (_) {
      // 忽略：推送注册失败不影响 App。
    }
  }

  /// POST /api/app/listing/register-token，逐个候选基址尝试。
  Future<void> _report({
    required String token,
    required String gateMode,
  }) async {
    final body = jsonEncode(<String, String>{
      'platform': GateConfig.platform,
      'bundleId': GateConfig.bundleId,
      'deviceToken': token,
      'gateMode': gateMode,
      'model': Platform.operatingSystemVersion,
    });
    for (final base in GateConfig.apiBases) {
      final uri = Uri.tryParse('$base${GateConfig.registerTokenPath}');
      if (uri == null) continue;
      final client = HttpClient()
        ..connectionTimeout = GateConfig.requestTimeout;
      try {
        final req = await client
            .postUrl(uri)
            .timeout(GateConfig.requestTimeout);
        req.headers.contentType = ContentType.json;
        req.add(utf8.encode(body));
        final resp = await req.close().timeout(GateConfig.requestTimeout);
        await resp.drain<void>();
        if (resp.statusCode == HttpStatus.ok) return; // 成功即止
      } on Object {
        // 试下一个候选基址。
      } finally {
        client.close(force: true);
      }
    }
  }
}
