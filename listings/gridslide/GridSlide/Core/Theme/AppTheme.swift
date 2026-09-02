import UIKit

/// 全局配色与外观。
///
/// 本包用**深色**主题：滑块拼图是要盯着看的东西，深底衬浅色块最省眼，
/// 也和同为 iOS 上架包的 pocketledger（浅色记账）明显区分开。
enum AppTheme {
    static let background = UIColor(red: 0.063, green: 0.078, blue: 0.094, alpha: 1) // #101418
    static let surface = UIColor(red: 0.106, green: 0.129, blue: 0.169, alpha: 1)    // #1B212B
    static let separator = UIColor(white: 1, alpha: 0.08)

    static let textPrimary = UIColor(red: 0.925, green: 0.941, blue: 0.961, alpha: 1) // #ECF0F5
    static let textSecondary = UIColor(red: 0.545, green: 0.588, blue: 0.647, alpha: 1) // #8B96A5

    static let accent = UIColor(red: 0.298, green: 0.761, blue: 0.627, alpha: 1)     // #4CC2A0
    static let danger = UIColor(red: 0.898, green: 0.361, blue: 0.361, alpha: 1)
    static let success = accent

    /// 方块本体：浅色实体块 + 深色数字，像真的塑料滑块。
    static let tileFill = UIColor(red: 0.906, green: 0.925, blue: 0.949, alpha: 1)   // #E7ECF2
    static let tileText = UIColor(red: 0.071, green: 0.086, blue: 0.106, alpha: 1)   // #12161B

    /// 已归位的方块换成强调色底 + 深色字 —— 玩家一眼看得出「这块已经对了」，
    /// 这是这类拼图最有用的一条即时反馈。
    static let tilePlacedFill = accent
    static let tilePlacedText = UIColor(red: 0.043, green: 0.145, blue: 0.118, alpha: 1)

    /// 棋盘底板上空格的凹槽色。
    static let slotFill = UIColor(white: 1, alpha: 0.05)

    static let cornerRadius: CGFloat = 16
    static let horizontalPadding: CGFloat = 20

    static func applyTabBarAppearance(_ tabBar: UITabBar) {
        let appearance = UITabBarAppearance()
        appearance.configureWithOpaqueBackground()
        appearance.backgroundColor = surface
        appearance.stackedLayoutAppearance.normal.iconColor = textSecondary
        appearance.stackedLayoutAppearance.normal.titleTextAttributes = [
            .foregroundColor: textSecondary,
            .font: UIFont.systemFont(ofSize: 11, weight: .medium)
        ]
        appearance.stackedLayoutAppearance.selected.iconColor = accent
        appearance.stackedLayoutAppearance.selected.titleTextAttributes = [
            .foregroundColor: accent,
            .font: UIFont.systemFont(ofSize: 11, weight: .semibold)
        ]
        tabBar.standardAppearance = appearance
        // scrollEdgeAppearance 是 iOS 15 的 API；本包最低就是 15.6，无需可用性判断。
        tabBar.scrollEdgeAppearance = appearance
        tabBar.tintColor = accent
        tabBar.unselectedItemTintColor = textSecondary
    }

    static func applyNavigationAppearance(_ navigationBar: UINavigationBar) {
        let appearance = UINavigationBarAppearance()
        appearance.configureWithOpaqueBackground()
        appearance.backgroundColor = background
        appearance.shadowColor = .clear
        appearance.titleTextAttributes = [
            .foregroundColor: textPrimary,
            .font: UIFont.systemFont(ofSize: 17, weight: .semibold)
        ]
        appearance.largeTitleTextAttributes = [
            .foregroundColor: textPrimary,
            .font: UIFont.systemFont(ofSize: 34, weight: .bold)
        ]
        navigationBar.standardAppearance = appearance
        navigationBar.scrollEdgeAppearance = appearance
        navigationBar.compactAppearance = appearance
        navigationBar.tintColor = accent
    }
}
