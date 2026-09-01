import Foundation

/// 用户设置。只有寥寥几项，放 UserDefaults 正合适（账本本体在 LedgerStore 的文件里）。
final class UserSettingsStore {
    static let shared = UserSettingsStore()
    private init() {}

    private let currencyKey = "pkl.currencyCode"

    /// 可选币种。以目标市场常见的几个打头，其余按字母序。
    /// 不做成「全世界 150 种」的长列表：滚半天找不到自己那个，反而更难用。
    static let supportedCurrencies: [String] = [
        "PHP", "USD", "EUR", "GBP", "AUD", "CAD", "CNY", "HKD",
        "IDR", "INR", "JPY", "KRW", "MYR", "SGD", "THB", "TWD", "VND"
    ]

    /// 当前币种。默认取设备地区的币种；设备币种不在支持列表里时回落到 USD。
    var currencyCode: String {
        get {
            if let stored = UserDefaults.standard.string(forKey: currencyKey),
               Self.supportedCurrencies.contains(stored) {
                return stored
            }
            return Self.deviceCurrencyCode
        }
        set {
            guard Self.supportedCurrencies.contains(newValue) else { return }
            UserDefaults.standard.set(newValue, forKey: currencyKey)
        }
    }

    private static var deviceCurrencyCode: String {
        let raw: String?
        if #available(iOS 16.0, *) {
            raw = Locale.current.currency?.identifier
        } else {
            // iOS 16 起 currencyCode 被 currency 取代，但本包最低支持 15.6，两条都要留。
            raw = Locale.current.currencyCode
        }
        // 统一转大写再比：`Locale.Currency.identifier` 在部分场景返回小写（如 "php"），
        // 直接比会永远失配、静默回落到 USD —— 用户看到的是「币种默认值莫名其妙不对」。
        if let code = raw?.uppercased(), supportedCurrencies.contains(code) { return code }
        return "USD"
    }

    /// 商店与政策链接。**上线前必须换成本包自己的**，见 STORE_ASSETS.md：
    /// 与其余上架包共用同一个政策页，等于在 App Store 侧把它们公开关联起来。
    static let privacyPolicyURLString = "TODO_PRIVACY_POLICY_URL"
    static let supportEmail = "TODO_SUPPORT_EMAIL"

    static var privacyPolicyURL: URL? {
        guard privacyPolicyURLString.hasPrefix("http") else { return nil }
        return URL(string: privacyPolicyURLString)
    }

    static var appVersionDisplay: String {
        let info = Bundle.main.infoDictionary
        let version = info?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = info?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }
}
