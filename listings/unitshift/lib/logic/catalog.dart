import 'unit.dart';

/// 单位表。
///
/// 系数一律写成**精确定义值**，不抄近似值：
/// 1 in = 0.0254 m、1 lb = 0.45359237 kg、1 mile = 1609.344 m 都是国际协定的
/// 精确定义，写全位数不会有误差；而抄一个 2.54 cm ≈ 2.5 的近似值，
/// 在长距离换算上会一路放大。每个不显然的系数后面都注明了出处。
///
/// 每一类的基准单位（factor = 1）写在第一位，`catalog_test.dart` 会逐类检查。
class Catalog {
  Catalog._();

  static const List<UnitCategory> categories = <UnitCategory>[
    _length,
    _mass,
    _temperature,
    _area,
    _volume,
    _speed,
    _time,
    _data,
    _pressure,
    _energy,
  ];

  static UnitCategory? byId(String id) {
    for (final c in categories) {
      if (c.id == id) return c;
    }
    return null;
  }

  // ---------------------------------------------------------------- 长度（米）

  static const UnitCategory _length = UnitCategory(
    id: 'length',
    label: 'Length',
    defaultUnitId: 'length.m',
    units: <Unit>[
      Unit(id: 'length.m', name: 'Meter', symbol: 'm', factor: 1),
      Unit(id: 'length.km', name: 'Kilometer', symbol: 'km', factor: 1000),
      Unit(id: 'length.cm', name: 'Centimeter', symbol: 'cm', factor: 0.01),
      Unit(id: 'length.mm', name: 'Millimeter', symbol: 'mm', factor: 0.001),
      Unit(id: 'length.um', name: 'Micrometer', symbol: 'µm', factor: 1e-6),
      // 1 in = 0.0254 m（1959 国际码磅协定，精确值）
      Unit(id: 'length.in', name: 'Inch', symbol: 'in', factor: 0.0254),
      Unit(id: 'length.ft', name: 'Foot', symbol: 'ft', factor: 0.3048),
      Unit(id: 'length.yd', name: 'Yard', symbol: 'yd', factor: 0.9144),
      Unit(id: 'length.mi', name: 'Mile', symbol: 'mi', factor: 1609.344),
      // 海里：1929 起国际定义为整 1852 m
      Unit(id: 'length.nmi', name: 'Nautical mile', symbol: 'nmi', factor: 1852),
    ],
  );

  // ---------------------------------------------------------------- 质量（千克）

  static const UnitCategory _mass = UnitCategory(
    id: 'mass',
    label: 'Mass',
    defaultUnitId: 'mass.kg',
    units: <Unit>[
      Unit(id: 'mass.kg', name: 'Kilogram', symbol: 'kg', factor: 1),
      Unit(id: 'mass.g', name: 'Gram', symbol: 'g', factor: 0.001),
      Unit(id: 'mass.mg', name: 'Milligram', symbol: 'mg', factor: 1e-6),
      Unit(id: 'mass.t', name: 'Tonne', symbol: 't', factor: 1000),
      // 1 lb = 0.45359237 kg（精确定义）；盎司 = 1/16 磅
      Unit(id: 'mass.lb', name: 'Pound', symbol: 'lb', factor: 0.45359237),
      Unit(id: 'mass.oz', name: 'Ounce', symbol: 'oz', factor: 0.028349523125),
      Unit(id: 'mass.st', name: 'Stone', symbol: 'st', factor: 6.35029318),
      // 短吨 = 2000 磅（美制），与上面的公吨不是一回事
      Unit(id: 'mass.ton_us', name: 'Short ton', symbol: 'ton', factor: 907.18474),
    ],
  );

  // ---------------------------------------------------------------- 温度（摄氏）

  /// 温度是唯一一类**不能用系数**的：它带零点偏移。
  /// 用系数算的话 0°C 会变成 0°F，差了 32 度 —— 这是单位换算 App 最典型的 bug。
  static const UnitCategory _temperature = UnitCategory(
    id: 'temperature',
    label: 'Temperature',
    defaultUnitId: 'temp.c',
    units: <Unit>[
      Unit.custom(
        id: 'temp.c',
        name: 'Celsius',
        symbol: '°C',
        toBase: _identity,
        fromBase: _identity,
      ),
      Unit.custom(
        id: 'temp.f',
        name: 'Fahrenheit',
        symbol: '°F',
        toBase: _fahrenheitToCelsius,
        fromBase: _celsiusToFahrenheit,
      ),
      Unit.custom(
        id: 'temp.k',
        name: 'Kelvin',
        symbol: 'K',
        toBase: _kelvinToCelsius,
        fromBase: _celsiusToKelvin,
      ),
      Unit.custom(
        id: 'temp.r',
        name: 'Rankine',
        symbol: '°R',
        toBase: _rankineToCelsius,
        fromBase: _celsiusToRankine,
      ),
    ],
  );

  static double _identity(double v) => v;
  static double _fahrenheitToCelsius(double v) => (v - 32) * 5 / 9;
  static double _celsiusToFahrenheit(double v) => v * 9 / 5 + 32;
  static double _kelvinToCelsius(double v) => v - 273.15;
  static double _celsiusToKelvin(double v) => v + 273.15;
  static double _rankineToCelsius(double v) => (v - 491.67) * 5 / 9;
  static double _celsiusToRankine(double v) => v * 9 / 5 + 491.67;

  // ---------------------------------------------------------------- 面积（平方米）

  static const UnitCategory _area = UnitCategory(
    id: 'area',
    label: 'Area',
    defaultUnitId: 'area.m2',
    units: <Unit>[
      Unit(id: 'area.m2', name: 'Square meter', symbol: 'm²', factor: 1),
      Unit(id: 'area.km2', name: 'Square kilometer', symbol: 'km²', factor: 1e6),
      Unit(id: 'area.cm2', name: 'Square centimeter', symbol: 'cm²', factor: 1e-4),
      Unit(id: 'area.ha', name: 'Hectare', symbol: 'ha', factor: 10000),
      Unit(id: 'area.ft2', name: 'Square foot', symbol: 'ft²', factor: 0.09290304),
      Unit(id: 'area.in2', name: 'Square inch', symbol: 'in²', factor: 0.00064516),
      Unit(id: 'area.yd2', name: 'Square yard', symbol: 'yd²', factor: 0.83612736),
      // 1 acre = 4840 平方码
      Unit(id: 'area.acre', name: 'Acre', symbol: 'ac', factor: 4046.8564224),
      Unit(id: 'area.mi2', name: 'Square mile', symbol: 'mi²', factor: 2589988.110336),
    ],
  );

  // ---------------------------------------------------------------- 体积（升）

  static const UnitCategory _volume = UnitCategory(
    id: 'volume',
    label: 'Volume',
    defaultUnitId: 'volume.l',
    units: <Unit>[
      Unit(id: 'volume.l', name: 'Liter', symbol: 'L', factor: 1),
      Unit(id: 'volume.ml', name: 'Milliliter', symbol: 'mL', factor: 0.001),
      Unit(id: 'volume.m3', name: 'Cubic meter', symbol: 'm³', factor: 1000),
      Unit(id: 'volume.cm3', name: 'Cubic centimeter', symbol: 'cm³', factor: 0.001),
      // 美制液量：1 gal = 231 in³ = 3.785411784 L（精确）
      Unit(id: 'volume.gal_us', name: 'Gallon (US)', symbol: 'gal', factor: 3.785411784),
      Unit(id: 'volume.qt_us', name: 'Quart (US)', symbol: 'qt', factor: 0.946352946),
      Unit(id: 'volume.pt_us', name: 'Pint (US)', symbol: 'pt', factor: 0.473176473),
      Unit(id: 'volume.cup_us', name: 'Cup (US)', symbol: 'cup', factor: 0.2365882365),
      Unit(id: 'volume.floz_us', name: 'Fluid ounce (US)', symbol: 'fl oz', factor: 0.0295735295625),
      // 英制加仑是另一个数（4.54609 L），跟美制差 20%，分开列
      Unit(id: 'volume.gal_uk', name: 'Gallon (UK)', symbol: 'gal UK', factor: 4.54609),
      Unit(id: 'volume.ft3', name: 'Cubic foot', symbol: 'ft³', factor: 28.316846592),
    ],
  );

  // ---------------------------------------------------------------- 速度（米每秒）

  static const UnitCategory _speed = UnitCategory(
    id: 'speed',
    label: 'Speed',
    defaultUnitId: 'speed.kmh',
    units: <Unit>[
      Unit(id: 'speed.ms', name: 'Meter per second', symbol: 'm/s', factor: 1),
      Unit(id: 'speed.kmh', name: 'Kilometer per hour', symbol: 'km/h', factor: 1 / 3.6),
      Unit(id: 'speed.mph', name: 'Mile per hour', symbol: 'mph', factor: 0.44704),
      Unit(id: 'speed.fts', name: 'Foot per second', symbol: 'ft/s', factor: 0.3048),
      // 1 节 = 1 海里/小时 = 1852/3600 m/s
      Unit(id: 'speed.knot', name: 'Knot', symbol: 'kn', factor: 1852 / 3600),
    ],
  );

  // ---------------------------------------------------------------- 时间（秒）

  static const UnitCategory _time = UnitCategory(
    id: 'time',
    label: 'Time',
    defaultUnitId: 'time.min',
    units: <Unit>[
      Unit(id: 'time.s', name: 'Second', symbol: 's', factor: 1),
      Unit(id: 'time.ms', name: 'Millisecond', symbol: 'ms', factor: 0.001),
      Unit(id: 'time.min', name: 'Minute', symbol: 'min', factor: 60),
      Unit(id: 'time.h', name: 'Hour', symbol: 'h', factor: 3600),
      Unit(id: 'time.d', name: 'Day', symbol: 'd', factor: 86400),
      Unit(id: 'time.wk', name: 'Week', symbol: 'wk', factor: 604800),
      // 「年」取儒略年 365.25 天 —— 天文与工程通用的那个定义。
      // 不用日历年是因为日历年长度不固定，换算结果会随口径变。
      Unit(id: 'time.yr', name: 'Year (Julian)', symbol: 'yr', factor: 31557600),
    ],
  );

  // ---------------------------------------------------------------- 数据（字节）

  /// 二进制前缀（KiB/MiB）与十进制前缀（KB/MB）分开列。
  /// 把 1 KB 当 1024 B 是历史遗留的混用，这里按标准各归各的，
  /// 谁想要哪个自己挑 —— 硬盘厂商用十进制、操作系统用二进制，两边都得有。
  static const UnitCategory _data = UnitCategory(
    id: 'data',
    label: 'Data',
    defaultUnitId: 'data.mib',
    units: <Unit>[
      Unit(id: 'data.b', name: 'Byte', symbol: 'B', factor: 1),
      Unit(id: 'data.bit', name: 'Bit', symbol: 'bit', factor: 0.125),
      Unit(id: 'data.kib', name: 'Kibibyte', symbol: 'KiB', factor: 1024),
      Unit(id: 'data.mib', name: 'Mebibyte', symbol: 'MiB', factor: 1048576),
      Unit(id: 'data.gib', name: 'Gibibyte', symbol: 'GiB', factor: 1073741824),
      Unit(id: 'data.tib', name: 'Tebibyte', symbol: 'TiB', factor: 1099511627776),
      Unit(id: 'data.kb', name: 'Kilobyte', symbol: 'KB', factor: 1000),
      Unit(id: 'data.mb', name: 'Megabyte', symbol: 'MB', factor: 1000000),
      Unit(id: 'data.gb', name: 'Gigabyte', symbol: 'GB', factor: 1000000000),
      Unit(id: 'data.tb', name: 'Terabyte', symbol: 'TB', factor: 1000000000000),
    ],
  );

  // ---------------------------------------------------------------- 压强（帕）

  static const UnitCategory _pressure = UnitCategory(
    id: 'pressure',
    label: 'Pressure',
    defaultUnitId: 'pressure.bar',
    units: <Unit>[
      Unit(id: 'pressure.pa', name: 'Pascal', symbol: 'Pa', factor: 1),
      Unit(id: 'pressure.kpa', name: 'Kilopascal', symbol: 'kPa', factor: 1000),
      Unit(id: 'pressure.bar', name: 'Bar', symbol: 'bar', factor: 100000),
      // 标准大气压的定义值
      Unit(id: 'pressure.atm', name: 'Atmosphere', symbol: 'atm', factor: 101325),
      Unit(id: 'pressure.psi', name: 'Pound per sq inch', symbol: 'psi', factor: 6894.757293168),
      // 1 mmHg = 1 torr = 101325/760 Pa
      Unit(id: 'pressure.mmhg', name: 'Millimeter of mercury', symbol: 'mmHg', factor: 101325 / 760),
    ],
  );

  // ---------------------------------------------------------------- 能量（焦）

  static const UnitCategory _energy = UnitCategory(
    id: 'energy',
    label: 'Energy',
    defaultUnitId: 'energy.kcal',
    units: <Unit>[
      Unit(id: 'energy.j', name: 'Joule', symbol: 'J', factor: 1),
      Unit(id: 'energy.kj', name: 'Kilojoule', symbol: 'kJ', factor: 1000),
      // 热化学卡：1 cal = 4.184 J（食品标签用的就是这个口径的千卡）
      Unit(id: 'energy.cal', name: 'Calorie', symbol: 'cal', factor: 4.184),
      Unit(id: 'energy.kcal', name: 'Kilocalorie', symbol: 'kcal', factor: 4184),
      Unit(id: 'energy.wh', name: 'Watt hour', symbol: 'Wh', factor: 3600),
      Unit(id: 'energy.kwh', name: 'Kilowatt hour', symbol: 'kWh', factor: 3600000),
      Unit(id: 'energy.btu', name: 'British thermal unit', symbol: 'BTU', factor: 1055.05585262),
    ],
  );
}
