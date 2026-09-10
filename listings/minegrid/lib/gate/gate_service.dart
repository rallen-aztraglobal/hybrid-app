import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'gate_config.dart';

/// B 面 url 是否可用：能解析、且是 http/https、且有主机名。
///
/// 对齐 decktallypro `GateService.parse` 里的 `URL(string:)` 校验 —— 那边解析失败即判 A。
/// Flutter 的 `Uri.tryParse` 比 iOS 宽松（'not a url' 也能解析成一个只有 path 的 Uri），
/// 故这里额外要求 scheme 与 host，免得把一个 WebView 加载不了的地址当成有效 B 面。
bool _isUsableUrl(String raw) {
  if (raw.isEmpty) return false;
  final uri = Uri.tryParse(raw);
  if (uri == null) return false;
  return (uri.scheme == 'http' || uri.scheme == 'https') && uri.host.isNotEmpty;
}

/// 网关判定结果。默认构造即「A 面」——任何不确定都应落到这个安全默认值上。
class GateResult {
  const GateResult.aSide() : mode = 'A', url = null, openMode = 'internal';
  const GateResult.bSide(this.url, {this.openMode = 'internal'}) : mode = 'B';

  final String mode;
  final String? url;

  /// B 面打开方式：'internal'=内开（原生 WebView）/ 'external'=外开（外部浏览器）。
  /// 仅 mode=B 时有意义，服务端只在判 B 时下发；缺省/非法一律 internal（默认内开）。
  final String openMode;

  bool get isBSide => mode == 'B' && url != null && _isUsableUrl(url!);

  /// 是否外开（外部浏览器）。仅在 isBSide 时才应参考此值。
  bool get isExternal => openMode == 'external';
}

/// 调用服务端 AB 面网关。判定全在服务端（按请求真实 IP 查国家 + 时区/IP 规则）；
/// 客户端只上报 platform/bundleId/timezone，从不上报 IP，也从不内置 B 面地址。
///
/// 安全不变量：**任何异常（超时、网络错误、非 200、解析失败）一律返回 A 面**。
/// 与后端 fail-closed 一致——判错成 A 只是少放一个用户，判错成 B 可能放进审核员。
class GateService {
  const GateService();

  Future<GateResult> evaluate({required String timezone}) async {
    final body = jsonEncode(<String, String>{
      'platform': GateConfig.platform,
      'bundleId': GateConfig.bundleId,
      'timezone': timezone,
    });

    // 依次尝试各候选基址，任一成功即返回；全部失败落到 A 面。
    for (final base in GateConfig.apiBases) {
      final result = await _post(base, body);
      if (result != null) return result;
    }
    return const GateResult.aSide();
  }

  /// 向单个基址发一次判定请求。返回 null 表示这个基址不可用（外层继续试下一个）。
  Future<GateResult?> _post(String base, String body) async {
    final uri = Uri.tryParse('$base${GateConfig.gatePath}');
    if (uri == null) return null;

    final client = HttpClient()..connectionTimeout = GateConfig.requestTimeout;
    try {
      final req = await client.postUrl(uri).timeout(GateConfig.requestTimeout);
      req.headers.contentType = ContentType.json;
      req.add(utf8.encode(body));

      final resp = await req.close().timeout(GateConfig.requestTimeout);
      if (resp.statusCode != HttpStatus.ok) {
        return const GateResult.aSide();
      }
      final text = await resp
          .transform(utf8.decoder)
          .join()
          .timeout(GateConfig.requestTimeout);
      return _parse(text);
    } on TimeoutException {
      return const GateResult.aSide();
    } on SocketException {
      // 连不上这个基址：返回 null 让外层尝试下一个候选。
      return null;
    } catch (_) {
      return const GateResult.aSide();
    } finally {
      client.close(force: true);
    }
  }

  /// 解析响应体。形状不符一律当 A 面。
  GateResult _parse(String text) {
    try {
      final data = jsonDecode(text);
      if (data is Map && data['mode'] == 'B') {
        final url = data['url'];
        // 非空之外还必须真的能解析成 http(s) 地址，对齐 decktallypro 的
        // `URL(string:)` 校验（解析不出就判 A）。colorstack 只校验非空，一个畸形
        // url 会让内开路径的 Uri.parse 抛 FormatException、B 面白屏；这里前置挡掉，
        // 挡不住的畸形值宁可判 A 也不带着崩。
        if (url is String && _isUsableUrl(url)) {
          // openMode 只认 'external'，其余（含缺省）一律 internal（默认内开）。
          final openMode = data['openMode'] == 'external'
              ? 'external'
              : 'internal';
          return GateResult.bSide(url, openMode: openMode);
        }
      }
    } catch (_) {
      // 落到 A 面。
    }
    return const GateResult.aSide();
  }
}
