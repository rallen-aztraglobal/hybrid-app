import UIKit

/// 全局配色与外观。
///
/// 本包用**浅色**主题（其余上架包都是深色）——理由有两条：
/// 一是记账类 App 的通行观感就是浅色；二是 B 面加载的是浅色品牌站，
/// `WebContainerViewController` 的安全区留白也是白色，A→B 切换时不会闪一下深色。
enum AppTheme {
    static let background = UIColor(red: 0.961, green: 0.965, blue: 0.973, alpha: 1) // #F5F6F8
    static let surface = UIColor.white
    static let separator = UIColor(red: 0.898, green: 0.906, blue: 0.922, alpha: 1)  // #E5E7EB

    static let textPrimary = UIColor(red: 0.071, green: 0.078, blue: 0.102, alpha: 1) // #12141A
    static let textSecondary = UIColor(red: 0.420, green: 0.447, blue: 0.502, alpha: 1) // #6B7280

    static let accent = UIColor(red: 0.184, green: 0.420, blue: 1.0, alpha: 1)        // #2F6BFF
    static let expense = UIColor(red: 0.898, green: 0.282, blue: 0.302, alpha: 1)     // #E5484D
    static let income = UIColor(red: 0.086, green: 0.639, blue: 0.290, alpha: 1)      // #16A34A

    static let cornerRadius: CGFloat = 16
    static let horizontalPadding: CGFloat = 20

    /// 账户/分类的可选配色。新建账户时让用户挑一个，列表里一眼区分。
    static let palette: [UIColor] = [
        UIColor(red: 0.184, green: 0.420, blue: 1.0, alpha: 1),   // 蓝
        UIColor(red: 0.486, green: 0.361, blue: 1.0, alpha: 1),   // 紫
        UIColor(red: 0.055, green: 0.647, blue: 0.643, alpha: 1), // 青
        UIColor(red: 0.086, green: 0.639, blue: 0.290, alpha: 1), // 绿
        UIColor(red: 0.918, green: 0.616, blue: 0.114, alpha: 1), // 琥珀
        UIColor(red: 0.937, green: 0.416, blue: 0.259, alpha: 1), // 橙
        UIColor(red: 0.898, green: 0.282, blue: 0.302, alpha: 1), // 红
        UIColor(red: 0.902, green: 0.318, blue: 0.612, alpha: 1)  // 粉
    ]

    /// 取配色，越界回落到强调色而不是崩 —— 存档里的旧索引不该让 App 挂掉。
    static func paletteColor(_ index: Int) -> UIColor {
        guard index >= 0 && index < palette.count else { return accent }
        return palette[index]
    }

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
        // scrollEdgeAppearance 是 iOS 15 的 API；本包最低就是 15.6，无需再做可用性判断。
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
