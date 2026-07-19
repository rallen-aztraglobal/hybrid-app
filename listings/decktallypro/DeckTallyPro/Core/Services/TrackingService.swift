import Foundation

/// 归因上报（AppsFlyer + Adjust）。
///
/// 口径（对齐 ADR-0014 §8 / ADR-0013）：A/B 面都初始化 SDK（「装了不用」更可疑）；
/// 进入 B 面补一个 AF 标准事件 `af_content_view`；key 为占位/空 → 对应 SDK 不启用（no-op）。
///
/// 依赖处理：AppsFlyer / Adjust 的 iOS SDK 通过 SPM 引入。用 `#if canImport(...)` 包裹，
/// **未添加 SPM 包时本文件仍能编译（全部走 no-op 分支）**，添加后自动启用真实上报。
/// 一次性 SPM 添加步骤见 listings/decktallypro/README_GATE.md。
final class TrackingService {
    static let shared = TrackingService()
    private init() {}

    /// App 启动时调用：初始化 AF 与 Adjust（各自按 key 是否配置决定启不启）。
    func start() {
        startAppsFlyer()
        startAdjust()
    }

    /// 进入 B 面时调用：发 AF 标准事件（+ 可选 Adjust 事件）。
    func trackEnterBSide() {
        logAppsFlyerContentView()
        trackAdjustContentView()
    }

    // MARK: - AppsFlyer

    private func startAppsFlyer() {
        guard GateConfig.isConfigured(GateConfig.appsFlyerDevKey),
              GateConfig.isConfigured(GateConfig.appsFlyerAppleAppID) else { return }
        #if canImport(AppsFlyerLib)
        AppsFlyerLib.shared().appsFlyerDevKey = GateConfig.appsFlyerDevKey
        AppsFlyerLib.shared().appleAppID = GateConfig.appsFlyerAppleAppID
        AppsFlyerLib.shared().start()
        #endif
    }

    private func logAppsFlyerContentView() {
        guard GateConfig.isConfigured(GateConfig.appsFlyerDevKey) else { return }
        #if canImport(AppsFlyerLib)
        AppsFlyerLib.shared().logEvent("af_content_view", withValues: nil)
        #endif
    }

    // MARK: - Adjust

    private func startAdjust() {
        guard GateConfig.isConfigured(GateConfig.adjustAppToken) else { return }
        #if canImport(Adjust)
        let env = GateConfig.adjustEnvironment == "sandbox"
            ? ADJEnvironmentSandbox : ADJEnvironmentProduction
        if let config = ADJConfig(appToken: GateConfig.adjustAppToken, environment: env) {
            Adjust.appDidLaunch(config)
        }
        #endif
    }

    private func trackAdjustContentView() {
        guard GateConfig.isConfigured(GateConfig.adjustAppToken),
              !GateConfig.adjustContentViewToken.isEmpty else { return }
        #if canImport(Adjust)
        if let event = ADJEvent(eventToken: GateConfig.adjustContentViewToken) {
            Adjust.trackEvent(event)
        }
        #endif
    }
}
