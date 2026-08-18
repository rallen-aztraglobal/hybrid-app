import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:webview_flutter/webview_flutter.dart';

/// B 面全屏 WebView。加载服务端下发的 url，无应用自身 UI 外壳。
///
/// 说明：url 来自网关判定结果，绝不硬编码在包内（见 GateConfig 注释）。
///
/// 系统栏对齐渠道包 / decktallypro：
///   - 顶部垫满状态栏；底部只垫「一半」安全区（home indicator 区）；左右铺满不内缩。
///   - B 面是浅色品牌站：状态栏用深色图标，白底上清晰可读。
///   - 安全区留白用白色，与浅色站一致。
class WebScreen extends StatefulWidget {
  const WebScreen({super.key, required this.url});

  final String url;

  @override
  State<WebScreen> createState() => _WebScreenState();
}

class _WebScreenState extends State<WebScreen> {
  late final WebViewController _controller;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(Colors.white)
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

  void _stopLoading() {
    if (mounted && _loading) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    // 底部安全区取原始值的一半（对齐 decktallypro）。
    final halfBottom = MediaQuery.of(context).padding.bottom / 2;
    // 拦截返回键：优先在 WebView 内后退，栈空时才交回系统。
    return AnnotatedRegion<SystemUiOverlayStyle>(
      // 对齐渠道包 bp（浅色站）的 applyBrandSystemBars：状态栏/底部导航栏都用白底 + 深色图标。
      // 注意不能直接用 SystemUiOverlayStyle.dark —— 它虽给深色状态栏图标，却把 Android 底部
      // 导航栏染成黑色（systemNavigationBarColor: 0xFF000000），导致底部安全区发黑。
      value: const SystemUiOverlayStyle(
        statusBarColor: Colors.transparent,
        statusBarIconBrightness: Brightness.dark, // Android：深色状态栏图标
        statusBarBrightness: Brightness.light, // iOS：浅底 → 深色文字
        systemNavigationBarColor: Colors.white, // Android：底部导航栏纯白
        systemNavigationBarIconBrightness: Brightness.dark,
        systemNavigationBarDividerColor: Colors.white,
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
          backgroundColor: Colors.white,
          // 顶部垫满状态栏；左右铺满、底部不由 SafeArea 全垫（改手动垫一半）。
          body: SafeArea(
            left: false,
            right: false,
            bottom: false,
            child: Padding(
              padding: EdgeInsets.only(bottom: halfBottom),
              child: Stack(
                children: [
                  WebViewWidget(controller: _controller),
                  if (_loading)
                    const Center(child: CircularProgressIndicator()),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
