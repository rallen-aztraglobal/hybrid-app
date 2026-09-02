import Foundation

/// AB 面网关的编译期配置（对齐 docs/admin/09-listing.md 与 ADR-0014）。
///
/// 原则：
/// - 这里**不含任何 B 面地址**。B 面 URL 由服务端判定为 B 时下发，客户端从不内置。
/// - 唯一出现的是「网关 API 基址」——只是返回 A/B 的接口；即便被封，拿不到判定也只会回退 A 面。
/// - AF / Adjust 的 key 编译期烧录（非机密、随包分发、极少变）。留占位/空 = 对应 SDK 不启用。
enum GateConfig {
    /// 本包在服务端 listing_app 里的 bundleId（platform 固定 ios）。
    static let bundleId = "com.stillwater.pocketledger"
    static let platform = "ios"

    /// 网关 API 基址候选，按序尝试，任一成功即用；全部失败 → A 面。
    /// 与其余上架包用同一个渠道中台基址。可再加候选抗封。
    static let apiBases: [String] = [
        "https://api.fortunegems-jackpot.online"
    ]

    static let gatePath = "/api/app/listing/gate"
    static let registerTokenPath = "/api/app/listing/register-token"

    /// 单次判定超时。启动路径上不应为一个可回退 A 面的判定长时间卡白屏。
    static let requestTimeout: TimeInterval = 6

    // —— AppsFlyer ——
    /// 账号级 devKey，与其余上架包同一个 AF 账号，故直接复用。
    static let appsFlyerDevKey = "fXoKsKQwxPCRdhD8CD8q6F"
    /// App Store 的数字 id（形如 6780248860，不带 "id" 前缀）。
    /// **本包尚未在 App Store Connect 建条目**，故是占位符 —— 占位期间 AF 全链路 no-op。
    static let appsFlyerAppleAppID = "TODO_APPSTORE_APP_ID"

    // —— Adjust（占位/空 = 不启用）——
    /// 本包在 Adjust 后台的应用识别码（App Token）。Adjust 按 App 建，故**不能**复用
    /// decktallypro(sn947o53ym80) / colorstack(bytg13h7yubk) / hexacolorsort(2yhxl7paa3ls)
    /// 的 token —— 复用会把本包的安装与会话归到别的 App 上。
    /// Adjust 后台的 App 名为 `PocketLedger`，iOS 平台 `com.stillwater.pocketledger`
    /// （store itunes / app_store，app_state 为 not_verified —— ASC 条目还没建，
    /// 数字 App ID 留空，与 ColorStack 的 iOS 平台同样处理），
    /// reporting currency PHP、no_eea_users=true。
    static let adjustAppToken = "zoavz0rdks1s"
    /// 生产 "production"，联调 "sandbox"。
    static let adjustEnvironment = "production"
    /// 「进入 B 面」的 Adjust 事件 token（可留空，只发 AF 标准事件也可）。
    static let adjustContentViewToken = ""
    /// 「外开进入 B 面」的 Adjust 事件 token（Adjust 后台 event `OpenBLanding`）。
    /// 仅在 openMode=external 唤起外部浏览器成功后触发；留空则该端不发 Adjust 事件。
    ///
    /// 取自 Adjust 后台 App `PocketLedger` 的 Events 页。与上面的 `adjustAppToken` 成对生效：
    /// `TrackingService.trackOpenBLanding()` 要求两者都已配置才会发 Adjust 事件。
    static let adjustOpenBLandingToken = "c72a3u"

    /// 占位/空值判定：TODO 前缀或空串都视为「未配置」。
    static func isConfigured(_ value: String) -> Bool {
        !value.isEmpty && !value.hasPrefix("TODO")
    }
}
