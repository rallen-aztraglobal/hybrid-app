import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        let window = UIWindow(frame: .zero)
        window.frame = window.screen.bounds
        // 本 App 用一套固定的深色配色；锁住外观，免得系统浅色模式把系统控件
        // （分段控件、开关、警告框）翻成浅色，与自绘的深色内容打架。
        window.overrideUserInterfaceStyle = .dark
        self.window = window
        // FCM 推送初始化（配置 Firebase + 注册 APNs）；未加 SPM 包或缺配置文件时整段 no-op。
        PushService.shared.start(application)
        // 启动闸：先做 AB 面判定，再决定根控制器（A=MainTabBarController / B=WebView）。
        // 判定失败或非 B 一律进 A 面，游戏本体代码零改动。
        GateCoordinator.start(in: window)
        return true
    }

    // APNs 注册成功：把 device token 交给 Firebase（FCM token 依赖它）。
    func application(_ application: UIApplication,
                     didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        PushService.shared.setAPNsToken(deviceToken)
    }

    // APNs 注册失败：忽略（推送不可用不影响 App）。
    func application(_ application: UIApplication,
                     didFailToRegisterForRemoteNotificationsWithError error: Error) {}
}
