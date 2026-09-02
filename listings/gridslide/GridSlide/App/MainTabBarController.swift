import UIKit

/// A 面根控制器：三个标签页 —— 玩 / 成绩 / 设置。
final class MainTabBarController: UITabBarController {
    private var isReselectingCurrentTab = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        AppTheme.applyTabBarAppearance(tabBar)
        delegate = self
        viewControllers = [
            makeNav(root: PlayViewController(), title: "Play", symbol: "square.grid.3x3.fill"),
            makeNav(root: RecordsViewController(), title: "Records", symbol: "rosette"),
            makeNav(root: SettingsViewController(), title: "Settings", symbol: "gearshape.fill")
        ]
    }

    private func makeNav(root: UIViewController, title: String, symbol: String) -> UINavigationController {
        root.title = title
        let nav = UINavigationController(rootViewController: root)
        nav.tabBarItem = UITabBarItem(
            title: title,
            image: UIImage(systemName: symbol),
            selectedImage: UIImage(systemName: symbol)
        )
        AppTheme.applyNavigationAppearance(nav.navigationBar)
        nav.navigationBar.prefersLargeTitles = true
        return nav
    }

    /// 再次点已选中的标签：先返回栈顶，否则把当前页滚回顶部（iOS 的通行手势）。
    private func handleTabReselection(_ viewController: UIViewController) {
        if let nav = viewController as? UINavigationController {
            if nav.viewControllers.count > 1 {
                nav.popToRootViewController(animated: true)
                return
            }
            scrollFirstScrollViewToTop(in: nav.topViewController?.view)
        } else {
            scrollFirstScrollViewToTop(in: viewController.view)
        }
    }

    private func scrollFirstScrollViewToTop(in view: UIView?) {
        guard let view else { return }
        if let scrollView = view as? UIScrollView {
            let topOffset = CGPoint(
                x: -scrollView.adjustedContentInset.left,
                y: -scrollView.adjustedContentInset.top
            )
            scrollView.setContentOffset(topOffset, animated: true)
            return
        }
        for subview in view.subviews {
            scrollFirstScrollViewToTop(in: subview)
        }
    }
}

extension MainTabBarController: UITabBarControllerDelegate {
    func tabBarController(_ tabBarController: UITabBarController, shouldSelect viewController: UIViewController) -> Bool {
        isReselectingCurrentTab = viewController === selectedViewController
        return true
    }

    func tabBarController(_ tabBarController: UITabBarController, didSelect viewController: UIViewController) {
        if isReselectingCurrentTab {
            handleTabReselection(viewController)
        }
        isReselectingCurrentTab = false
    }
}
