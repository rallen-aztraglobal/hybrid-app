import 'package:flutter/services.dart';

/// 音量键计数。对应原生侧 `MainActivity` 里的按键拦截。
///
/// 只做两件事：把开关传下去，把按下的事件传上来。
/// 所有调用都包了 try/catch —— 这是个锦上添花的功能，
/// 平台通道出任何问题都不该影响用手点数这条主路径。
class VolumeKeys {
  VolumeKeys._();

  static const MethodChannel _channel = MethodChannel('tickpad/volume_keys');

  static void Function(int delta)? _onCount;

  /// 注册回调。[delta] 为 +1（音量加）或 -1（音量减）。
  static void listen(void Function(int delta) onCount) {
    _onCount = onCount;
    _channel.setMethodCallHandler((call) async {
      if (call.method != 'count') return null;
      final direction = call.arguments;
      if (direction == 'up') {
        _onCount?.call(1);
      } else if (direction == 'down') {
        _onCount?.call(-1);
      }
      return null;
    });
  }

  /// 开 / 关。关掉之后音量键恢复成音量键。
  static Future<void> setEnabled(bool enabled) async {
    try {
      await _channel.invokeMethod<void>('setEnabled', enabled);
    } catch (_) {
      // 平台不支持（比如将来补 iOS 时）就静默降级成「只能用手点」
    }
  }
}
