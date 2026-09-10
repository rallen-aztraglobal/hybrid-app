import 'package:flutter_test/flutter_test.dart';
import 'package:unitshift/logic/catalog.dart';
import 'package:unitshift/logic/unit.dart';

/// 单位表的验收。
///
/// 换算 App 唯一不能出错的地方就是数。一个抄错的系数在界面上完全看不出来 ——
/// 它只是安静地给出一个错的数字，用户拿去下料、算配方、算行程。
/// 所以这里对每一类都钉了**权威定义值**，而不是拿代码自己的结果对自己。
void main() {
  /// a 与 b 是否在相对误差 [tolerance] 内相等。
  ///
  /// 用相对误差而不是绝对误差：这里的量级横跨 1e-6（微米）到 1e12（TB），
  /// 一个固定的绝对阈值对大数太松、对小数太严。
  void expectClose(double actual, double expected, {double tolerance = 1e-9}) {
    final diff = (actual - expected).abs();
    final scale = expected.abs() < 1 ? 1.0 : expected.abs();
    expect(
      diff / scale < tolerance,
      isTrue,
      reason: '期望 $expected，实际 $actual（相对误差 ${diff / scale}）',
    );
  }

  group('表结构', () {
    test('分类 id 不重复', () {
      final ids = Catalog.categories.map((c) => c.id).toList();
      expect(ids.toSet().length, ids.length);
    });

    test('单位 id 全局唯一 —— 存档里记的就是它', () {
      final ids = <String>[];
      for (final c in Catalog.categories) {
        ids.addAll(c.units.map((u) => u.id));
      }
      expect(ids.toSet().length, ids.length, reason: '有重复的单位 id');
    });

    test('同一类里的简写不重复 —— 列表上会挨着显示', () {
      for (final c in Catalog.categories) {
        final symbols = c.units.map((u) => u.symbol).toList();
        expect(symbols.toSet().length, symbols.length,
            reason: '${c.label} 里有重复简写');
      }
    });

    test('每类的第一个单位就是基准单位（系数 1）', () {
      for (final c in Catalog.categories) {
        if (c.id == 'temperature') continue; // 温度不是线性的，另有一组用例
        expect(c.units.first.isBase, isTrue,
            reason: '${c.label} 的第一个单位不是基准单位');
      }
    });

    test('每类的默认单位确实在表里', () {
      for (final c in Catalog.categories) {
        expect(c.byId(c.defaultUnitId), isNotNull,
            reason: '${c.label} 的 defaultUnitId 指向一个不存在的单位');
      }
    });

    test('每类至少 4 个单位 —— 少于这个数不值得单开一类', () {
      for (final c in Catalog.categories) {
        expect(c.units.length, greaterThanOrEqualTo(4), reason: c.label);
      }
    });
  });

  group('往返换算', () {
    test('任一单位 → 基准 → 回来，值不变', () {
      // 这条守的是「toBase / fromBase 是一对真互逆函数」。
      // 温度那种手写的一对函数最容易写反某一步，往返一跑就现形。
      for (final c in Catalog.categories) {
        for (final u in c.units) {
          for (final v in <double>[1, 0, -40, 123.456, 1e-4, 98765]) {
            expectClose(u.fromBase(u.toBase(v)), v, tolerance: 1e-9);
          }
        }
      }
    });

    test('convertAll 的顺序与 units 一致，且含源单位自己', () {
      for (final c in Catalog.categories) {
        final from = c.units.first;
        final values = c.convertAll(7, from);
        expect(values.length, c.units.length);
        expectClose(values.first, 7);
      }
    });
  });

  group('长度', () {
    final length = Catalog.byId('length')!;
    Unit u(String id) => length.byId(id)!;

    test('1 in = 2.54 cm（1959 国际协定的精确值）', () {
      expectClose(u('length.cm').fromBase(u('length.in').toBase(1)), 2.54);
    });

    test('1 ft = 12 in', () {
      expectClose(u('length.in').fromBase(u('length.ft').toBase(1)), 12);
    });

    test('1 mi = 1609.344 m = 5280 ft', () {
      expectClose(u('length.m').fromBase(u('length.mi').toBase(1)), 1609.344);
      expectClose(u('length.ft').fromBase(u('length.mi').toBase(1)), 5280);
    });

    test('1 nmi = 1852 m', () {
      expectClose(u('length.m').fromBase(u('length.nmi').toBase(1)), 1852);
    });
  });

  group('质量', () {
    final mass = Catalog.byId('mass')!;
    Unit u(String id) => mass.byId(id)!;

    test('1 lb = 0.45359237 kg（精确定义）', () {
      expectClose(u('mass.kg').fromBase(u('mass.lb').toBase(1)), 0.45359237);
    });

    test('1 lb = 16 oz', () {
      expectClose(u('mass.oz').fromBase(u('mass.lb').toBase(1)), 16);
    });

    test('1 st = 14 lb', () {
      expectClose(u('mass.lb').fromBase(u('mass.st').toBase(1)), 14);
    });

    test('1 短吨 = 2000 lb，且不等于 1 公吨', () {
      expectClose(u('mass.lb').fromBase(u('mass.ton_us').toBase(1)), 2000);
      final shortTon = u('mass.ton_us').toBase(1);
      final tonne = u('mass.t').toBase(1);
      expect(shortTon == tonne, isFalse, reason: '短吨和公吨被写成一样的了');
    });
  });

  group('温度 —— 唯一一类不能用系数的', () {
    final temp = Catalog.byId('temperature')!;
    Unit u(String id) => temp.byId(id)!;

    test('0°C = 32°F，100°C = 212°F', () {
      expectClose(u('temp.f').fromBase(u('temp.c').toBase(0)), 32);
      expectClose(u('temp.f').fromBase(u('temp.c').toBase(100)), 212);
    });

    test('-40 是两个刻度的交点', () {
      expectClose(u('temp.f').fromBase(u('temp.c').toBase(-40)), -40);
    });

    test('0°C = 273.15 K，绝对零度 = -273.15°C', () {
      expectClose(u('temp.k').fromBase(u('temp.c').toBase(0)), 273.15);
      expectClose(u('temp.c').fromBase(u('temp.k').toBase(0)), -273.15);
    });

    test('0°R = -459.67°F（兰氏零点即绝对零度）', () {
      expectClose(u('temp.f').fromBase(u('temp.r').toBase(0)), -459.67);
    });

    test('如果有人把温度改成按系数算，这条会挂', () {
      // 按系数算的话 0°C 会得出 0°F —— 差 32 度。
      // 这是单位换算 App 最典型、也最容易被忽略的 bug，单独钉一条。
      expect(u('temp.f').fromBase(u('temp.c').toBase(0)) == 0, isFalse);
    });
  });

  group('体积 —— 美制与英制不能混', () {
    final volume = Catalog.byId('volume')!;
    Unit u(String id) => volume.byId(id)!;

    test('1 美制加仑 = 3.785411784 L', () {
      expectClose(u('volume.l').fromBase(u('volume.gal_us').toBase(1)),
          3.785411784);
    });

    test('1 英制加仑 = 4.54609 L，比美制大约 20%', () {
      expectClose(u('volume.l').fromBase(u('volume.gal_uk').toBase(1)), 4.54609);
      final ratio = u('volume.gal_uk').toBase(1) / u('volume.gal_us').toBase(1);
      expect(ratio, greaterThan(1.2));
    });

    test('1 美制加仑 = 4 qt = 8 pt = 128 fl oz', () {
      final gal = u('volume.gal_us').toBase(1);
      expectClose(u('volume.qt_us').fromBase(gal), 4);
      expectClose(u('volume.pt_us').fromBase(gal), 8);
      expectClose(u('volume.floz_us').fromBase(gal), 128);
    });
  });

  group('数据 —— 二进制与十进制前缀分开', () {
    final data = Catalog.byId('data')!;
    Unit u(String id) => data.byId(id)!;

    test('1 KiB = 1024 B，1 KB = 1000 B', () {
      expectClose(u('data.b').fromBase(u('data.kib').toBase(1)), 1024);
      expectClose(u('data.b').fromBase(u('data.kb').toBase(1)), 1000);
    });

    test('1 B = 8 bit', () {
      expectClose(u('data.bit').fromBase(u('data.b').toBase(1)), 8);
    });

    test('1 TiB = 1099511627776 B', () {
      expectClose(u('data.b').fromBase(u('data.tib').toBase(1)), 1099511627776);
    });
  });

  group('速度 · 压强 · 能量', () {
    test('1 km/h = 1/3.6 m/s；1 kn = 1 nmi/h', () {
      final speed = Catalog.byId('speed')!;
      expectClose(
        speed.byId('speed.ms')!.fromBase(speed.byId('speed.kmh')!.toBase(3.6)),
        1,
      );
      expectClose(
        speed.byId('speed.kmh')!.fromBase(speed.byId('speed.knot')!.toBase(1)),
        1.852,
      );
    });

    test('1 atm = 101325 Pa = 760 mmHg', () {
      final p = Catalog.byId('pressure')!;
      final atm = p.byId('pressure.atm')!.toBase(1);
      expectClose(p.byId('pressure.pa')!.fromBase(atm), 101325);
      expectClose(p.byId('pressure.mmhg')!.fromBase(atm), 760);
    });

    test('1 kcal = 4184 J；1 kWh = 3.6 MJ', () {
      final e = Catalog.byId('energy')!;
      expectClose(e.byId('energy.j')!.fromBase(e.byId('energy.kcal')!.toBase(1)),
          4184);
      expectClose(e.byId('energy.j')!.fromBase(e.byId('energy.kwh')!.toBase(1)),
          3.6e6);
    });
  });
}
