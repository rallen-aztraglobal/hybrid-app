import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'gate_config.dart';

/// 网关判定结果。默认构造即「A 面」——任何不确定都应落到这个安全默认值上。
class GateResult {
  const GateResult.aSide()
      : mode = 'A',
        url = null;
  const GateResult.bSide(this.url) : mode = 'B';

  final String mode;
  final String? url;

  bool get isBSide => mode == 'B' && url != null && url!.isNotEmpty;
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
      final req = await client
          .postUrl(uri)
          .timeout(GateConfig.requestTimeout);
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
        if (url is String && url.isNotEmpty) {
          return GateResult.bSide(url);
        }
      }
    } catch (_) {
      // 落到 A 面。
    }
    return const GateResult.aSide();
  }
}
