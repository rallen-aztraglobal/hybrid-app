import 'package:flutter_test/flutter_test.dart';
import 'package:tilefit/gate/gate_service.dart';

/// 网关的安全不变量都收敛在 GateResult 上：只有「mode=B 且 url 非空且能解析成 http(s)」
/// 才算 B 面，openMode 只有显式 'external' 才外开。判定链路上任何一步出错都会落到
/// GateResult.aSide()，故这里锁住这几条判定，避免后续改动把「不确定」误判成放行。
void main() {
  group('GateResult', () {
    test('A 面永远不会被当成 B 面，且默认内开', () {
      const result = GateResult.aSide();

      expect(result.mode, 'A');
      expect(result.url, isNull);
      expect(result.isBSide, isFalse);
      expect(result.isExternal, isFalse);
    });

    test('B 面必须有一个非空 url', () {
      expect(const GateResult.bSide('https://example.com').isBSide, isTrue);
      expect(const GateResult.bSide('').isBSide, isFalse);
    });

    test('url 必须是能解析的 http(s) 地址，畸形值一律判 A', () {
      expect(const GateResult.bSide('not a url').isBSide, isFalse);
      expect(const GateResult.bSide('ftp://example.com').isBSide, isFalse);
      expect(
        const GateResult.bSide('https://').isBSide,
        isFalse,
        reason: '无主机名',
      );
      expect(const GateResult.bSide('http://example.com').isBSide, isTrue);
    });

    test('打开方式默认内开，只有显式 external 才外开', () {
      const internal = GateResult.bSide('https://example.com');
      const unknown = GateResult.bSide(
        'https://example.com',
        openMode: 'whatever',
      );
      const external = GateResult.bSide(
        'https://example.com',
        openMode: 'external',
      );

      expect(internal.isExternal, isFalse);
      expect(unknown.isExternal, isFalse);
      expect(external.isExternal, isTrue);
    });
  });
}
