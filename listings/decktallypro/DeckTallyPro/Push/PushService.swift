import Foundation
import UIKit
import UserNotifications
#if canImport(FirebaseCore)
import FirebaseCore
#endif
#if canImport(FirebaseMessaging)
import FirebaseMessaging
#endif

/// FCM 推送接入：配置 Firebase、注册 APNs、取 FCM token 并随「最近一次 AB 面判定结果」上报后端。
///
/// 后端「上架包推送」强制只发 last_gate_mode='B' 的设备，故注册必须带上本次判定 mode。
///
/// 依赖处理：Firebase iOS SDK 通过 SPM 引入，用 `#if canImport(...)` 包裹——
/// **未添加 SPM 包时本文件仍能编译（推送整段 no-op）**，添加后自动启用。步骤见 README_GATE.md。
///
/// 全程容错：推送不可用绝不影响 App 启动或 A/B 呈现。
final class PushService: NSObject {
    static let shared = PushService()
    private override init() {}

    /// 本次启动的 AB 面判定结果，取到 token 时带上上报。
    private var pendingGateMode: String?

    /// 在 didFinishLaunching 里调用：配置 Firebase、设委托、请求授权并注册 APNs。
    func start(_ application: UIApplication) {
        #if canImport(FirebaseCore)
        FirebaseApp.configure()
        #endif
        #if canImport(FirebaseMessaging)
        Messaging.messaging().delegate = self
        #endif
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { _, _ in }
        application.registerForRemoteNotifications()
    }

    /// AppDelegate 收到 APNs token 时转交 Firebase（FCM token 依赖它）。
    func setAPNsToken(_ deviceToken: Data) {
        #if canImport(FirebaseMessaging)
        Messaging.messaging().apnsToken = deviceToken
        #endif
    }

    /// 拿到 AB 面判定结果后调用：记录 mode，并尝试取 FCM token 上报。
    /// token 可能此刻尚未就绪（APNs 未回调）——MessagingDelegate 回调里会再补一次。
    func registerWithGateMode(_ mode: String) {
        pendingGateMode = mode
        #if canImport(FirebaseMessaging)
        Messaging.messaging().token { [weak self] token, _ in
            guard let self, let token else { return }
            self.report(token: token, gateMode: mode)
        }
        #endif
    }

    /// POST /api/app/listing/register-token，逐个候选基址尝试。
    private func report(token: String, gateMode: String) {
        let payload: [String: String] = [
            "platform": GateConfig.platform,
            "bundleId": GateConfig.bundleId,
            "deviceToken": token,
            "gateMode": gateMode,
            "model": UIDevice.current.systemVersion
        ]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else { return }
        postToFirstAvailable(body: body, bases: GateConfig.apiBases)
    }

    private func postToFirstAvailable(body: Data, bases: [String]) {
        guard let base = bases.first, let url = URL(string: base + GateConfig.registerTokenPath) else { return }
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = body
        req.timeoutInterval = GateConfig.requestTimeout
        URLSession.shared.dataTask(with: req) { _, response, _ in
            if let http = response as? HTTPURLResponse, http.statusCode == 200 { return }
            // 该基址不通：尝试下一个候选。
            self.postToFirstAvailable(body: body, bases: Array(bases.dropFirst()))
        }.resume()
    }
}

#if canImport(FirebaseMessaging)
extension PushService: MessagingDelegate {
    /// FCM token 刷新回调（首次就绪 / 轮换时触发）。带上最近判定 mode 补一次上报。
    func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
        guard let fcmToken, let mode = pendingGateMode else { return }
        report(token: fcmToken, gateMode: mode)
    }
}
#endif
