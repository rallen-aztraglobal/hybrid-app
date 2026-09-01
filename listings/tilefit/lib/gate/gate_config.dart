import 'dart:io' show Platform;

/// AB 面网关的编译期配置。
///
/// 设计原则（对齐 docs/admin/09-listing.md 与 ADR-0014）：
/// - 这里**不含任何 B 面地址**。B 面 URL 由服务端在判定为 B 时下发，客户端从不内置，
///   审核方静态扫描包也扫不到线上域名。
/// - 这里能出现的只有「网关 API 的基址」——它只是一个返回 A/B 的接口，泄露它不等于泄露 B 面；
///   即便该 API 被封，客户端拿不到判定也只会**回退 A 面**（fail-closed），不影响审核安全。
/// - AF / Adjust 的 key 按 ADR-0013 的口径编译期烧录（非机密、随包分发、极少变）。留空/占位即该 SDK 不启用。
class GateConfig {
  GateConfig._();

  /// 本包在服务端 listing_app 里的 (platform, bundleId) 键。
  /// 本上架包**仅发 Android**（Google Play），故 Console 里只需建 android 一条；
  /// 保留 platform 分支是为了将来补 iOS 端时此文件无需改结构。
  static const String bundleId = 'com.emberlane.tilefit8264';

  /// 网关 API 基址候选，按顺序尝试，任一返回即用。全部失败 → A 面。
  /// 与 colorstack / hexacolorsort / calcpad 用同一个渠道中台基址。可再加候选抗封。
  static const List<String> apiBases = <String>[
    'https://api.fortunegems-jackpot.online',
  ];

  /// 网关判定接口路径（与后端 routes 一致）。
  static const String gatePath = '/api/app/listing/gate';

  /// 设备 push token 注册接口路径（带 AB 面判定结果，用于「只推 B 面设备」）。
  static const String registerTokenPath = '/api/app/listing/register-token';

  /// 单次网关请求超时。宁短勿长：启动路径上不该为一个可失败即回退 A 面的判定长时间卡白屏。
  static const Duration requestTimeout = Duration(seconds: 6);

  /// 平台标识，随请求发给服务端。
  static String get platform => Platform.isIOS ? 'ios' : 'android';

  // —— AppsFlyer ——
  /// 账号级 devKey，与 colorstack / decktallypro / hexacolorsort / calcpad 同一个 AF 账号，
  /// 故直接复用（AF 的 devKey 按账号发，不按 App 发）。
  static String get appsFlyerDevKey => 'fXoKsKQwxPCRdhD8CD8q6F';

  /// AF appId：Android 为包名（恒有效）；iOS 为 App Store 数字 id（形如 id6740000000）。
  /// 本包仅发 Android，iOS 侧留占位 —— AF 会因此在 iOS 保持 no-op（见 tracking_service 的双重校验）。
  static String get appsFlyerAppId =>
      Platform.isIOS ? 'idTODO_AF_IOS_APPID' : bundleId;

  // —— Adjust（留空/占位 = 不启用）——
  /// 本包在 Adjust 后台的应用识别码（App Token）。Adjust 按 App 建，故**不能**复用
  /// colorstack(bytg13h7yubk) / decktallypro(sn947o53ym80) / hexacolorsort(2yhxl7paa3ls)
  /// 的 token —— 复用会把本包的安装与会话归到别的 App 上。
  /// **本包尚未在 Adjust 建条目**：占位期间 Adjust 全链路 no-op（不初始化、不上报），
  /// 不影响网关与游戏本体。建条目时 reporting currency 选 PHP（与现有 App 一致，建后不可改），
  /// 拿到 12 位 App Token 后填入。
  static String get adjustAppToken => 'TODO_ADJUST_APP_TOKEN';

  /// 生产环境用 'production'，联调用 'sandbox'。
  static const String adjustEnvironment = 'production';

  /// 「进入 B 面」对应的 Adjust 事件 token（可留空，只发 AF 标准事件也可）。
  /// 与 colorstack / calcpad 一致，本包不发这个事件。
  static String get adjustContentViewToken => '';

  /// B 面系统栏/安全区的填充色。对齐渠道壳 WebViewActivity.applyBrandSystemBars：
  /// 该处按 BuildConfig.BRAND 取色，`gp`(GameZone) 用 #1C1D27、`ap`/`bp` 等浅色站用白色。
  /// 本包计划挂 `gp`（与 hexacolorsort / calcpad 一致）、B 面是深色站，故取 #1C1D27。
  /// 游戏本体也是深色（AppColors.background = #0E1116），切换观感连续。
  /// **若本包改挂 ap/bp（浅色站），这里要同步改成白色。**
  static const int bSideChromeColor = 0xFF1C1D27;

  /// 「外开进入 B 面」对应的 Adjust 事件 token（Adjust 后台 event `OpenBLanding`，非 unique
  /// event，故每次外开成功都计一次）。仅在 openMode=external 唤起外部浏览器成功后触发。
  static String get adjustOpenBLandingToken => '';
}
