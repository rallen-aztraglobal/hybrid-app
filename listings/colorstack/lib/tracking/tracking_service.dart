import 'dart:async';

import 'package:appsflyer_sdk/appsflyer_sdk.dart';
import 'package:adjust_sdk/adjust.dart';
import 'package:adjust_sdk/adjust_config.dart';
import 'package:adjust_sdk/adjust_event.dart';

import '../gate/gate_config.dart';

/// 归因上报（AppsFlyer + Adjust）。
///
/// 口径（对齐 ADR-0014 §8 / ADR-0013）：
/// - **A 面、B 面都初始化 SDK**。安装/会话由 SDK 自动归因。「集成了却完全不用」在审核侧反而可疑，
///   故初始化本身放在启动路径、与 A/B 判定无关。
/// - 进入 B 面时补一个 AF **标准事件** `af_content_view`（标准事件名，出现在游戏里不违和）。
/// - key 为占位/空 → 对应 SDK **不初始化、不上报**（no-op），与后端「未绑定即休眠」同构。
/// - 全程吞异常：归因失败绝不能影响 App 启动或 A/B 呈现。
class TrackingService {
  TrackingService._();
  static final TrackingService instance = TrackingService._();

  bool _afReady = false;
  AppsflyerSdk? _af;

  /// 占位/空值判定：TODO_ 前缀或空串都视为「未配置」。
  static bool _configured(String v) =>
      v.isNotEmpty && !v.startsWith('TODO') && !v.startsWith('id' 'TODO');

  /// 启动时调用，初始化 AF 与 Adjust（各自按 key 是否配置决定启不启）。
  Future<void> init() async {
    await _initAppsFlyer();
    _initAdjust();
  }

  Future<void> _initAppsFlyer() async {
    // devKey 与 appId 都需配置才启用：ColorStack iOS 的 App Store ID 未定，appId 占位时
    // iOS 侧 AF 保持 no-op；Android 的 appId=包名恒有效，devKey 填好即启用。
    if (!_configured(GateConfig.appsFlyerDevKey) || !_configured(GateConfig.appsFlyerAppId)) return;
    try {
      final options = AppsFlyerOptions(
        afDevKey: GateConfig.appsFlyerDevKey,
        appId: GateConfig.appsFlyerAppId,
        showDebug: false,
      );
      final af = AppsflyerSdk(options);
      await af.initSdk(
        registerConversionDataCallback: false,
        registerOnAppOpenAttributionCallback: false,
        registerOnDeepLinkingCallback: false,
      );
      _af = af;
      _afReady = true;
    } catch (_) {
      _afReady = false;
    }
  }

  void _initAdjust() {
    if (!_configured(GateConfig.adjustAppToken)) return;
    try {
      final env = GateConfig.adjustEnvironment == 'sandbox'
          ? AdjustEnvironment.sandbox
          : AdjustEnvironment.production;
      Adjust.initSdk(AdjustConfig(GateConfig.adjustAppToken, env));
    } catch (_) {
      // 忽略：Adjust 不可用不影响 App。
    }
  }

  /// 进入 B 面时调用：发 AF 标准事件 af_content_view（+ 可选的 Adjust 事件）。
  Future<void> onEnterBSide() async {
    if (_afReady && _af != null) {
      try {
        await _af!.logEvent('af_content_view', <String, dynamic>{});
      } catch (_) {}
    }
    final token = GateConfig.adjustContentViewToken;
    if (_configured(GateConfig.adjustAppToken) && token.isNotEmpty) {
      try {
        Adjust.trackEvent(AdjustEvent(token));
      } catch (_) {}
    }
  }

  /// 外开进入 B 面时调用：AF + Adjust 各发一个自定义事件 `OpenBLanding`。
  /// 与 onEnterBSide 的 af_content_view 相互独立、可并存；仅在外部浏览器唤起成功后触发。
  /// AF 用事件名字符串（无需 token），Adjust 用编译期烧录的 event token。全程吞异常。
  Future<void> onOpenBLanding() async {
    if (_afReady && _af != null) {
      try {
        await _af!.logEvent('OpenBLanding', <String, dynamic>{});
      } catch (_) {}
    }
    final token = GateConfig.adjustOpenBLandingToken;
    if (_configured(GateConfig.adjustAppToken) && token.isNotEmpty) {
      try {
        Adjust.trackEvent(AdjustEvent(token));
      } catch (_) {}
    }
  }
}
