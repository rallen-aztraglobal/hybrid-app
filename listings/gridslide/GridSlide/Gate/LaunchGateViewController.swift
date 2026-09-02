import UIKit

/// 判定期的加载页（App 冷启动后的第一屏）。
///
/// 底色与 LaunchScreen.storyboard、以及后续的 A 面完全一致（AppTheme.background），
/// 于是「系统启动图 → 判定加载页 → 游戏主界面」三段之间没有任何背景跳变。
final class LaunchGateViewController: UIViewController {
    private let spinner = UIActivityIndicatorView(style: .large)

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        spinner.color = AppTheme.accent
        spinner.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(spinner)
        NSLayoutConstraint.activate([
            spinner.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            spinner.centerYAnchor.constraint(equalTo: view.centerYAnchor)
        ])
        spinner.startAnimating()
    }

    // 深色底：状态栏用浅色文字。
    override var preferredStatusBarStyle: UIStatusBarStyle { .lightContent }
}
