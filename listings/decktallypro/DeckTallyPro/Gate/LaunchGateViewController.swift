import UIKit

/// 判定期的加载页（App 冷启动后的第一屏）。深色背景 + 转圈，视觉上与后续 A/B 面衔接。
final class LaunchGateViewController: UIViewController {
    private let spinner = UIActivityIndicatorView(style: .large)

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        spinner.color = .white
        spinner.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(spinner)
        NSLayoutConstraint.activate([
            spinner.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            spinner.centerYAnchor.constraint(equalTo: view.centerYAnchor)
        ])
        spinner.startAnimating()
    }

    override var preferredStatusBarStyle: UIStatusBarStyle { .lightContent }
}
