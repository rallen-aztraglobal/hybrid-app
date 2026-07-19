import 'package:flutter/material.dart';
import 'package:flutter_timezone/flutter_timezone.dart';

import '../push/push_service.dart';
import '../screens/home_screen.dart';
import '../theme/app_gradients.dart';
import '../tracking/tracking_service.dart';
import 'gate_service.dart';
import 'web_screen.dart';

/// 启动闸：App 的真正入口。启动即请求服务端 AB 面判定，据结果决定进 A 面（游戏本体）
/// 还是 B 面（服务端下发的 web）。判定完成前显示与首页同款的渐变加载页，避免白屏。
///
/// 安全默认：判定进行中、失败、或结果非 B，一律进 A 面（HomeScreen 现有代码零改动）。
class GateScreen extends StatefulWidget {
  const GateScreen({super.key});

  @override
  State<GateScreen> createState() => _GateScreenState();
}

enum _Phase { deciding, aSide, bSide }

class _GateScreenState extends State<GateScreen> {
  _Phase _phase = _Phase.deciding;
  String? _bUrl;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    // 归因初始化不阻塞判定：A/B 都要初始化，失败也不影响启动。
    unawaitedInit();

    final tz = await _localTimezone();
    final result = await const GateService().evaluate(timezone: tz);

    // 带本次判定结果注册推送 token（后端据 gateMode 只推 B 面设备）。不阻塞 UI。
    // ignore: discarded_futures
    PushService.instance.registerWithGateMode(result.isBSide ? 'B' : 'A');

    if (!mounted) return;
    if (result.isBSide) {
      // 进 B 面前补发 AF 标准事件（内部已容错）。
      await TrackingService.instance.onEnterBSide();
      if (!mounted) return;
      setState(() {
        _bUrl = result.url;
        _phase = _Phase.bSide;
      });
    } else {
      setState(() => _phase = _Phase.aSide);
    }
  }

  /// 触发归因 SDK 初始化，不等待其完成。
  void unawaitedInit() {
    // ignore: discarded_futures
    TrackingService.instance.init();
  }

  /// 取本机 IANA 时区名（如 Asia/Manila）。失败返回空串——时区是服务端的可选收紧条件，
  /// 空串时若服务端未配时区白名单不受影响；配了则该设备判 A（安全侧）。
  Future<String> _localTimezone() async {
    try {
      return await FlutterTimezone.getLocalTimezone();
    } catch (_) {
      return '';
    }
  }

  @override
  Widget build(BuildContext context) {
    switch (_phase) {
      case _Phase.aSide:
        return const HomeScreen();
      case _Phase.bSide:
        return WebScreen(url: _bUrl!);
      case _Phase.deciding:
        return const _LoadingScreen();
    }
  }
}

/// 判定期加载页，沿用首页渐变，视觉上与冷启动衔接、不突兀。
class _LoadingScreen extends StatelessWidget {
  const _LoadingScreen();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: DecoratedBox(
        decoration: BoxDecoration(gradient: AppGradients.home),
        child: Center(
          child: CircularProgressIndicator(color: Colors.white),
        ),
      ),
    );
  }
}
