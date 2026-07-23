import UIKit
import WebKit

/// B 面全屏 WebView 容器。加载服务端下发的 url，无应用自身 UI 外壳。
/// url 来自网关判定结果，绝不硬编码在包内（见 GateConfig 注释）。
///
/// 系统栏对齐渠道包（Android WebViewActivity）：
///   - 安全区：WebView 垫在状态栏之下、home indicator 之上（对齐 systemInsets.top/bottom），
///     内容不被刘海/圆角/底部横条遮挡；左右不内缩，与 Android 只 inset 上下一致。
///   - 状态栏：B 面加载的是浅色品牌站（ap/bp），用深色图标（.darkContent），白底上文字清晰，
///     对齐 Android「浅底→深色图标」的 applyBrandSystemBars 逻辑。
///   - 安全区留白用白色，与浅色站背景一致，无深色露边。
final class WebContainerViewController: UIViewController {
    private let url: URL
    private var webView: WKWebView!
    private let spinner = UIActivityIndicatorView(style: .large)
    /// WebView 底部约束（相对容器底边）。底部安全区只垫「一半」，故约束常量在
    /// viewSafeAreaInsetsDidChange 里动态设为 -safeAreaInsets.bottom / 2。
    private var webBottomConstraint: NSLayoutConstraint!

    init(url: URL) {
        self.url = url
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) not supported") }

    override func loadView() {
        // 根容器：安全区留白（状态栏/home indicator 区域）用白色，与浅色站一致。
        let container = UIView()
        container.backgroundColor = .white

        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        webView = WKWebView(frame: .zero, configuration: config)
        // 允许左右边缘手势前进/后退，替代无导航栏时的返回操作。
        webView.allowsBackForwardNavigationGestures = true
        webView.navigationDelegate = self
        webView.backgroundColor = .white
        webView.isOpaque = true
        // WebView 已被约束进安全区，禁用 scrollView 的自动安全区 inset，避免重复内缩。
        webView.scrollView.contentInsetAdjustmentBehavior = .never
        webView.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(webView)

        // 顶部垫满状态栏、底部只垫「一半」安全区（常量在 viewSafeAreaInsetsDidChange 更新）、左右铺满。
        webBottomConstraint = webView.bottomAnchor.constraint(equalTo: container.bottomAnchor)
        NSLayoutConstraint.activate([
            webView.topAnchor.constraint(equalTo: container.safeAreaLayoutGuide.topAnchor),
            webBottomConstraint,
            webView.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: container.trailingAnchor)
        ])

        view = container
    }

    // 底部安全区取原始值的一半：把 WebView 底边下压到「距容器底 safeAreaInsets.bottom/2」处。
    override func viewSafeAreaInsetsDidChange() {
        super.viewSafeAreaInsetsDidChange()
        webBottomConstraint.constant = -view.safeAreaInsets.bottom / 2
    }

    override func viewDidLoad() {
        super.viewDidLoad()

        // 白底上灰色转圈才可见（原为白色，白底不可见）。
        spinner.color = .gray
        spinner.translatesAutoresizingMaskIntoConstraints = false
        spinner.hidesWhenStopped = true
        view.addSubview(spinner)
        NSLayoutConstraint.activate([
            spinner.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            spinner.centerYAnchor.constraint(equalTo: view.centerYAnchor)
        ])

        spinner.startAnimating()
        webView.load(URLRequest(url: url))
    }

    // B 面是浅色品牌站：深色状态栏图标，白底上清晰可读（对齐渠道包浅底→深色图标）。
    override var preferredStatusBarStyle: UIStatusBarStyle {
        if #available(iOS 13.0, *) { return .darkContent }
        return .default
    }
}

extension WebContainerViewController: WKNavigationDelegate {
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        spinner.stopAnimating()
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        spinner.stopAnimating()
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        spinner.stopAnimating()
    }
}
