import 'package:flutter_test/flutter_test.dart';
import 'package:calculator_app/gate/gate_service.dart';

/// 网关的安全不变量都收敛在 GateResult 上：只有「mode=B 且 url 非空」才算 B 面，
/// openMode 只有显式 'external' 才外开。判定链路上任何一步出错都会落到 GateResult.aSide()，
/// 故这里锁住这两条判定，避免后续改动把「不确定」误判成放行。
void main() {
  group('GateResult', () {
    test('A side is never treated as B and defaults to internal open', () {
      const result = GateResult.aSide();

      expect(result.mode, 'A');
      expect(result.url, isNull);
      expect(result.isBSide, isFalse);
      expect(result.isExternal, isFalse);
    });

    test('B side requires a non-empty url', () {
      expect(const GateResult.bSide('https://example.com').isBSide, isTrue);
      expect(const GateResult.bSide('').isBSide, isFalse);
    });

    test('open mode defaults to internal and is external only when set', () {
      const internal = GateResult.bSide('https://example.com');
      const external = GateResult.bSide(
        'https://example.com',
        openMode: 'external',
      );

      expect(internal.isExternal, isFalse);
      expect(external.isExternal, isTrue);
    });
  });
}
