import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'gate_config.dart';

/// B 面全屏 WebView。加载服务端下发的 url，无应用自身 UI 外壳。
///
/// 说明：url 来自网关判定结果，绝不硬编码在包内（见 GateConfig 注释）。
///
/// 系统栏与安全区对齐渠道壳 `WebViewActivity`（app/src/main/java/com/hybrid/android/）：
///   - 状态栏与底部导航栏都设为透明，由本页自己把那块区域涂成 `bSideChromeColor`；
///     渠道壳的 applySystemBarColors 也是这么做的（MIUI 全屏 flag 下 statusBarColor 常
///     不生效，靠底层视图上色才可靠）。
///   - WebView 上下各留满系统栏高度，内容不被刘海/手势条压住；左右铺满不内缩。
///   - 图标明暗按填充色的感知亮度决定（与渠道壳 isLightColor 同一口径：>0.6 视为浅色，
///     浅色底配深色图标）。
///
/// 为什么填充色不写死白色：本包挂 `gp`(GameZone)、B 面是深色站，白边在深色游戏切过去时
/// 非常突兀。渠道壳按品牌取色（gp=#1C1D27、ap/bp=白），这里同口径，见 GateConfig。
class WebScreen extends StatefulWidget {
  const WebScreen({super.key, required this.url});

  final String url;

  @override
  State<WebScreen> createState() => _WebScreenState();
}

class _WebScreenState extends State<WebScreen> {
  late final WebViewController _controller;
  bool _loading = true;

  /// B 面系统栏/安全区填充色。
  static const Color _chrome = Color(GateConfig.bSideChromeColor);

  /// 感知亮度 > 0.6 视为浅色（与渠道壳 isLightColor 一致）。
  static bool get _chromeIsLight {
    final l =
        (0.299 * _chrome.r + 0.587 * _chrome.g + 0.114 * _chrome.b) / 255.0;
    return l > 0.6;
  }

  @override
  void initState() {
    super.initState();
    // 真正的 edge-to-edge：Android 15+（targetSdk>=35）会忽略 systemNavigationBarColor，
    // 系统自己画底部那条栏 —— 涂色改不动它，实测仍是浅色。故让内容铺到屏幕底部，
    // 系统只在上面叠一个悬浮手势条（它自己按背景反色，深色页上呈浅色）。
    SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      // 加载期也用填充色，避免 WebView 默认白底在深色站前闪一下白。
      ..setBackgroundColor(_chrome)
      // decktallypro 的 allowsInlineMediaPlayback（视频内联播放、不强制全屏）在 Android 侧
      // 是 WebView 的默认行为，无需对应设置。注意别顺手关掉「播放需用户手势」——
      // 那是另一个开关（iOS 的 mediaTypesRequiringUserActionForPlayback），decktallypro 也没关。
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageFinished: (_) => _stopLoading(),
          // 对齐 decktallypro 的 didFail / didFailProvisionalNavigation：加载失败也必须收掉转圈。
          // 只靠 onPageFinished 的话（colorstack 现状），断网或域名被封时会永远转圈盖在白屏上，
          // 用户既看不到内容也看不出「加载失败」，只能杀进程。
          onWebResourceError: (_) => _stopLoading(),
          onHttpError: (_) => _stopLoading(),
        ),
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  void dispose() {
    // 外开失败会退回 A 面（游戏），把系统 UI 模式还原，别把 edge-to-edge 带进游戏页。
    SystemChrome.setEnabledSystemUIMode(
      SystemUiMode.manual,
      overlays: SystemUiOverlay.values,
    );
    super.dispose();
  }

  void _stopLoading() {
    if (mounted && _loading) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final insets = MediaQuery.of(context).padding;
    final iconBrightness = _chromeIsLight ? Brightness.dark : Brightness.light;
    // 拦截返回键：优先在 WebView 内后退，栈空时才交回系统。
    return AnnotatedRegion<SystemUiOverlayStyle>(
      // 两根系统栏都透明，颜色由下面的 Scaffold 背景透出来 —— 与渠道壳同一做法。
      value: SystemUiOverlayStyle(
        statusBarColor: Colors.transparent,
        statusBarIconBrightness: iconBrightness, // Android
        statusBarBrightness: _chromeIsLight
            ? Brightness.light
            : Brightness.dark, // iOS
        systemNavigationBarColor: _chrome,
        systemNavigationBarIconBrightness: iconBrightness,
        systemNavigationBarDividerColor: _chrome,
      ),
      child: PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, _) async {
          if (didPop) return;
          if (await _controller.canGoBack()) {
            await _controller.goBack();
          }
        },
        child: Scaffold(
          backgroundColor: _chrome,
          // 上下各垫满系统栏高度（渠道壳的 topMargin/bottomMargin = systemInsets），
          // 左右铺满。垫出来的区域由 Scaffold 背景着色，故与站点浑然一体。
          // 顶部垫满状态栏（站点头部不被时钟压住），底部不垫 —— 内容铺到底，
          // 手势条悬浮其上。这就是「全面屏 + 沉浸式」的实际做法。
          body: Padding(
            padding: EdgeInsets.only(top: insets.top),
            child: Stack(
              children: [
                WebViewWidget(controller: _controller),
                if (_loading)
                  Center(
                    child: CircularProgressIndicator(
                      color: _chromeIsLight ? Colors.black54 : Colors.white70,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
