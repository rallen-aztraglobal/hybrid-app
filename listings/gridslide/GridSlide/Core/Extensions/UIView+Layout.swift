import UIKit

extension UIView {
    func pinToEdges(of container: UIView, insets: UIEdgeInsets = .zero) {
        translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            topAnchor.constraint(equalTo: container.topAnchor, constant: insets.top),
            leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: insets.left),
            trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -insets.right),
            bottomAnchor.constraint(equalTo: container.bottomAnchor, constant: -insets.bottom)
        ])
    }

    /// 加子视图并关掉 autoresizing 转换 —— 纯代码布局时这两步永远成对出现。
    func addForAutoLayout(_ subview: UIView) {
        subview.translatesAutoresizingMaskIntoConstraints = false
        addSubview(subview)
    }
}

extension UIViewController {
    func showBanner(message: String, style: BannerStyle = .info) {
        view.subviews
            .compactMap { $0 as? MessageBannerView }
            .forEach { $0.dismissImmediately() }
        let banner = MessageBannerView(message: message, style: style)
        banner.present(in: view)
    }

    /// 统一的确认弹窗。删除这类不可撤销的操作一律走它，破坏性按钮标红。
    func confirmDestructive(
        title: String,
        message: String,
        confirmTitle: String,
        onConfirm: @escaping () -> Void
    ) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        alert.addAction(UIAlertAction(title: confirmTitle, style: .destructive) { _ in onConfirm() })
        present(alert, animated: true)
    }
}

enum BannerStyle {
    case info, success, failure
}

final class MessageBannerView: UIView {
    private let label = UILabel()

    init(message: String, style: BannerStyle) {
        super.init(frame: .zero)
        // 深色主题下的提示条：info 用抬高一档的面色 + 描边（不能像浅色主题那样
        // 拿近白色当底 —— 白底配白字直接看不见）。
        switch style {
        case .info:
            backgroundColor = AppTheme.surface
            layer.borderWidth = 1
            layer.borderColor = AppTheme.separator.cgColor
        case .success:
            backgroundColor = AppTheme.success.withAlphaComponent(0.95)
        case .failure:
            backgroundColor = AppTheme.danger.withAlphaComponent(0.95)
        }
        layer.cornerRadius = 14
        label.text = message
        label.textColor = style == .info ? AppTheme.textPrimary : .white
        label.font = .systemFont(ofSize: 14, weight: .semibold)
        label.numberOfLines = 0
        label.textAlignment = .center
        addForAutoLayout(label)
        NSLayoutConstraint.activate([
            label.topAnchor.constraint(equalTo: topAnchor, constant: 14),
            label.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            label.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            label.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -14)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func present(in container: UIView) {
        translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(self)
        NSLayoutConstraint.activate([
            leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 24),
            trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -24),
            bottomAnchor.constraint(equalTo: container.safeAreaLayoutGuide.bottomAnchor, constant: -16)
        ])
        alpha = 0
        transform = CGAffineTransform(translationX: 0, y: 20)
        UIView.animate(withDuration: 0.25) {
            self.alpha = 1
            self.transform = .identity
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.2) { [weak self] in
            UIView.animate(withDuration: 0.2, animations: {
                self?.alpha = 0
            }, completion: { _ in
                self?.removeFromSuperview()
            })
        }
    }

    func dismissImmediately() {
        layer.removeAllAnimations()
        removeFromSuperview()
    }
}
