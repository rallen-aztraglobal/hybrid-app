import Foundation

/// 用户设置。只有两项，放 UserDefaults。
///
/// 与 RecordStore 同理：`init` 不做任何会发通知的事。
final class SettingsStore {
    static let shared = SettingsStore()
    private init() {}

    private let sizeKey = "gsl.boardSize"
    private let hapticsKey = "gsl.hapticsEnabled"

    /// 可选的棋盘尺寸。
    ///
    /// 不做 2×2（只有两种局面，不成其为游戏），也不做 6×6 以上
    /// （方块小到点不准，且随机打乱后的复原步数会长到多数人不会玩完）。
    static let boardSizes = [3, 4, 5]

    /// 当前棋盘尺寸，默认 4×4（这个玩法最广为人知的形态就是十五数码）。
    var boardSize: Int {
        get {
            let stored = UserDefaults.standard.integer(forKey: sizeKey)
            return Self.boardSizes.contains(stored) ? stored : 4
        }
        set {
            guard Self.boardSizes.contains(newValue) else { return }
            UserDefaults.standard.set(newValue, forKey: sizeKey)
        }
    }

    /// 震动反馈。默认开。
    ///
    /// 用 `object(forKey:)` 判断有没有存过，而不是直接 `bool(forKey:)` ——
    /// 后者在「从没设置过」时返回 false，会让默认值变成「关」。
    var hapticsEnabled: Bool {
        get {
            guard UserDefaults.standard.object(forKey: hapticsKey) != nil else { return true }
            return UserDefaults.standard.bool(forKey: hapticsKey)
        }
        set { UserDefaults.standard.set(newValue, forKey: hapticsKey) }
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
