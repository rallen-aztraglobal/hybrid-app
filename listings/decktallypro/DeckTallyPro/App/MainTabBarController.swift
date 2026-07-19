import UIKit

final class MainTabBarController: UITabBarController {
    private var isReselectingCurrentTab = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        AppTheme.applyTabBarAppearance(tabBar)
        delegate = self
        viewControllers = [
            makeNav(root: HomeViewController(), title: "Home", symbol: "house.fill"),
            makeNav(root: ArenaViewController(), title: "Arena", symbol: "gamecontroller.fill"),
            makeNav(root: ProgressViewController(), title: "Progress", symbol: "chart.bar.fill"),
            makeNav(root: SettingsViewController(), title: "Settings", symbol: "gearshape.fill")
        ]
        presentOnboardingIfNeeded()
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

    private func presentOnboardingIfNeeded() {
        guard !UserSettingsStore.shared.didShowOnboarding else { return }
        let guide = GuideViewController()
        let nav = UINavigationController(rootViewController: guide)
        AppTheme.applyNavigationAppearance(nav.navigationBar)
        nav.modalPresentationStyle = .formSheet
        guide.onDismiss = {
            UserSettingsStore.shared.didShowOnboarding = true
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) { [weak self] in
            self?.present(nav, animated: true)
        }
    }

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
            let topOffset = CGPoint(x: -scrollView.adjustedContentInset.left, y: -scrollView.adjustedContentInset.top)
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
