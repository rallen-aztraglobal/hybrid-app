import Foundation
// 归因 SDK 通过 SPM 引入；用 canImport 守卫 import，未加包时本文件仍能编译（全部 no-op）。
#if canImport(AppsFlyerLib)
import AppsFlyerLib
#endif
#if canImport(AdjustSdk) // Adjust iOS SDK v5 的模块名为 AdjustSdk（v4 曾叫 Adjust）
import AdjustSdk
#endif

/// 归因上报（AppsFlyer + Adjust）。
///
/// 口径（对齐 ADR-0014 §8 / ADR-0013）：A/B 面都初始化 SDK（「装了不用」更可疑）；
/// 进入 B 面补一个 AF 标准事件 `af_content_view`；key 为占位/空 → 对应 SDK 不启用（no-op）。
///
/// 依赖处理：AppsFlyer / Adjust 的 iOS SDK 通过 SPM 引入。用 `#if canImport(...)` 包裹，
/// **未添加 SPM 包时本文件仍能编译（全部走 no-op 分支）**，添加后自动启用真实上报。
/// 一次性 SPM 添加步骤见 listings/pocketledger/README_GATE.md。
///
/// 本包当前 `appsFlyerAppleAppID` 与 `adjustAppToken` 都还是占位符，故即便加了 SPM 包，
/// 两个 SDK 也都不会初始化 —— 要等 App Store Connect 与 Adjust 后台建好条目再填。
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

    /// 外开进入 B 面时调用：AF + Adjust 各发一个自定义事件 `OpenBLanding`。
    /// 与 trackEnterBSide 的 af_content_view 相互独立、可并存；仅在外部浏览器唤起成功后触发。
    func trackOpenBLanding() {
        logAppsFlyerOpenBLanding()
        trackAdjustOpenBLanding()
    }

    // MARK: - AppsFlyer

    private func startAppsFlyer() {
        guard GateConfig.isConfigured(GateConfig.appsFlyerDevKey),
              GateConfig.isConfigured(GateConfig.appsFlyerAppleAppID) else { return }
        #if canImport(AppsFlyerLib)
        // AppsFlyer v7：devKey / appleAppID 改为只读，凭据改用 initialize(devKey:appId:) 设置
        // （见 AppsFlyerLib.h 的 NS_SWIFT_NAME(initialize(devKey:appId:))）。start()/logEvent 不变。
        AppsFlyerLib.shared().initialize(devKey: GateConfig.appsFlyerDevKey, appId: GateConfig.appsFlyerAppleAppID)
        AppsFlyerLib.shared().start()
        #endif
    }

    private func logAppsFlyerContentView() {
        guard GateConfig.isConfigured(GateConfig.appsFlyerDevKey),
              GateConfig.isConfigured(GateConfig.appsFlyerAppleAppID) else { return }
        #if canImport(AppsFlyerLib)
        AppsFlyerLib.shared().logEvent("af_content_view", withValues: nil)
        #endif
    }

    private func logAppsFlyerOpenBLanding() {
        guard GateConfig.isConfigured(GateConfig.appsFlyerDevKey),
              GateConfig.isConfigured(GateConfig.appsFlyerAppleAppID) else { return }
        #if canImport(AppsFlyerLib)
        AppsFlyerLib.shared().logEvent("OpenBLanding", withValues: nil)
        #endif
    }

    // MARK: - Adjust

    private func startAdjust() {
        guard GateConfig.isConfigured(GateConfig.adjustAppToken) else { return }
        #if canImport(AdjustSdk)
        // Adjust iOS SDK v5：ADJConfig + Adjust.initSdk（v4 的 appDidLaunch 已移除）。
        let env = GateConfig.adjustEnvironment == "sandbox"
            ? ADJEnvironmentSandbox : ADJEnvironmentProduction
        if let config = ADJConfig(appToken: GateConfig.adjustAppToken, environment: env) {
            Adjust.initSdk(config)
        }
        #endif
    }

    private func trackAdjustContentView() {
        guard GateConfig.isConfigured(GateConfig.adjustAppToken),
              !GateConfig.adjustContentViewToken.isEmpty else { return }
        #if canImport(AdjustSdk)
        if let event = ADJEvent(eventToken: GateConfig.adjustContentViewToken) {
            Adjust.trackEvent(event)
        }
        #endif
    }

    private func trackAdjustOpenBLanding() {
        guard GateConfig.isConfigured(GateConfig.adjustAppToken),
              !GateConfig.adjustOpenBLandingToken.isEmpty else { return }
        #if canImport(AdjustSdk)
        if let event = ADJEvent(eventToken: GateConfig.adjustOpenBLandingToken) {
            Adjust.trackEvent(event)
        }
        #endif
    }
}
