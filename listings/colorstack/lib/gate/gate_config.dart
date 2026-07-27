import 'dart:io' show Platform;

/// AB 面网关的编译期配置。
///
/// 设计原则（对齐 docs/admin/09-listing.md 与 ADR-0014）：
/// - 这里**不含任何 B 面地址**。B 面 URL 由服务端在判定为 B 时下发，客户端从不内置，
///   审核方静态扫描包也扫不到线上域名。
/// - 这里能出现的只有「网关 API 的基址」——它只是一个返回 A/B 的接口，泄露它不等于泄露 B 面；
///   即便该 API 被封，客户端拿不到判定也只会**回退 A 面**（fail-closed），不影响审核安全。
/// - AF / Adjust 的 key 按 ADR-0013 的口径编译期烧录（非机密、随包分发、极少变）。留空即该 SDK 不启用。
class GateConfig {
  GateConfig._();

  /// 本包在服务端 listing_app 里的 (platform, bundleId) 键。两端同包名，靠 platform 区分。
  static const String bundleId = 'com.vividnest.colorstack5821';

  /// 网关 API 基址候选，按顺序尝试，任一返回即用。全部失败 → A 面。
  /// 与现有 Android APK 用同一个渠道中台基址（APK bootstrap.json 的 configUrl 即
  /// https://api.fortunegems-jackpot.online/api/app/config，故基址一致）。可再加候选抗封。
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
  static String get appsFlyerDevKey => 'fXoKsKQwxPCRdhD8CD8q6F';

  /// AF appId：iOS 为 App Store 数字 id（形如 id6740000000），Android 为包名。
  /// ColorStack 的 iOS App Store ID 暂未确认（Apple 接口查不到，疑未公开上架），
  /// 故 iOS 侧仍留占位——AF 会因此在 iOS 保持 no-op（见 tracking_service 的双重校验），
  /// 拿到数字 id 后填成 'id<数字>' 即可启用 iOS 归因。Android 用包名，已可用。
  static String get appsFlyerAppId =>
      Platform.isIOS ? 'idTODO_AF_IOS_APPID' : bundleId;

  // —— Adjust（留空 = 不启用）——
  // ColorStack 的 Adjust 应用识别码（App Token）。运营给的是单个 token，两端共用；
  // 若后续为 iOS/Android 分建了不同的 Adjust app，在此按 Platform.isIOS 分开即可。
  static String get adjustAppToken => 'bytg13h7yubk';

  /// 生产环境用 'production'，联调用 'sandbox'。
  static const String adjustEnvironment = 'production';

  /// 「进入 B 面」对应的 Adjust 事件 token（可留空，只发 AF 标准事件也可）。
  static String get adjustContentViewToken => '';

  /// 「外开进入 B 面」对应的 Adjust 事件 token（Adjust 后台 event `OpenBLanding`）。
  /// 仅在 openMode=external 唤起外部浏览器成功后触发；留空则该端不发 Adjust 事件。
  static String get adjustOpenBLandingToken => 'y2qjxp';
}
