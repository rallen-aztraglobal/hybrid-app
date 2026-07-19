import UIKit

enum AppTheme {
    static let background = UIColor(red: 0.06, green: 0.08, blue: 0.14, alpha: 1)
    static let surface = UIColor(red: 0.11, green: 0.14, blue: 0.22, alpha: 1)
    static let surfaceElevated = UIColor(red: 0.16, green: 0.20, blue: 0.30, alpha: 1)
    static let accent = UIColor(red: 0.98, green: 0.55, blue: 0.20, alpha: 1)
    static let accentSecondary = UIColor(red: 0.20, green: 0.78, blue: 0.74, alpha: 1)
    static let textPrimary = UIColor.white
    static let textSecondary = UIColor(white: 0.78, alpha: 1)
    static let danger = UIColor(red: 1.0, green: 0.32, blue: 0.45, alpha: 1)
    static let success = UIColor(red: 0.22, green: 0.86, blue: 0.55, alpha: 1)

    static let cornerRadius: CGFloat = 18
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
        if #available(iOS 15.0, *) {
            tabBar.scrollEdgeAppearance = appearance
        }
        tabBar.tintColor = accent
        tabBar.unselectedItemTintColor = textSecondary
    }

    static func applyNavigationAppearance(_ navigationBar: UINavigationBar) {
        let appearance = UINavigationBarAppearance()
        appearance.configureWithOpaqueBackground()
        appearance.backgroundColor = background
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
        navigationBar.tintColor = accentSecondary
    }

}
