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

  /// 本次启动的 AB 面判定结果。token 刷新时要用它补报，故存下来。
  String? _pendingGateMode;

  /// token 刷新订阅。只订阅一次，避免重复注册。
  StreamSubscription<String>? _refreshSub;

  /// Firebase 初始化的 Future。启动路径只「触发」它，真正需要 token 时才带超时地等。
  Future<void>? _initFuture;

  /// 在 main() 里尽早调用：触发 Firebase 初始化并**立即返回**。
  ///
  /// 故意不是 async —— 调用方 await 不到东西，也就没法把它挂在 runApp 之前。
  /// Firebase.initializeApp() 在缺 google-services.json 时会失败，正常是快速抛异常，
  /// 但它是过原生通道的调用，卡住的可能性不为零；一旦卡在 runApp 之前就是纯黑屏、
  /// 游戏完全打不开。推送再重要也不值得拿启动去赌，故改为后台跑。
  void initFirebase() {
    _initFuture ??= _initFirebase();
  }

  Future<void> _initFirebase() async {
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
    _pendingGateMode = gateMode;
    // 带超时地等初始化收尾（main 里只触发未等待）。超时/失败都直接放弃推送。
    try {
      await _initFuture?.timeout(GateConfig.requestTimeout);
    } catch (_) {
      return;
    }
    if (!_firebaseReady) return;
    try {
      final messaging = FirebaseMessaging.instance;
      // iOS 需用户授权才发；Android 13+ 也需 POST_NOTIFICATIONS。拿不到授权仍尝试取 token。
      await messaging.requestPermission();

      // 对齐 decktallypro 的 MessagingDelegate.didReceiveRegistrationToken：token 首次就绪
      // 或轮换时补报一次。首启时 getToken() 可能赶在 token 生成之前返回 null，只报一次就会
      // 让这台设备在后端没有记录 —— 上架包推送只推 last_gate_mode='B' 的设备，漏登记等于收不到推送。
      _refreshSub ??= messaging.onTokenRefresh.listen((token) {
        final mode = _pendingGateMode;
        if (mode == null || token.isEmpty) return;
        // ignore: discarded_futures
        _report(token: token, gateMode: mode);
      }, onError: (_) {});

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
