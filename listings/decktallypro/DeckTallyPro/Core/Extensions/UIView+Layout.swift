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
}

extension UIViewController {
    func showBanner(message: String, style: BannerStyle = .info) {
        view.subviews
            .compactMap { $0 as? MessageBannerView }
            .forEach { $0.dismissImmediately() }
        let banner = MessageBannerView(message: message, style: style)
        banner.present(in: view)
    }
}

enum BannerStyle {
    case info, success, failure
}

final class MessageBannerView: UIView {
    private let label = UILabel()

    init(message: String, style: BannerStyle) {
        super.init(frame: .zero)
        backgroundColor = {
            switch style {
            case .info: return AppTheme.surfaceElevated
            case .success: return AppTheme.success.withAlphaComponent(0.92)
            case .failure: return AppTheme.danger.withAlphaComponent(0.92)
            }
        }()
        layer.cornerRadius = 14
        label.text = message
        label.textColor = .white
        label.font = .systemFont(ofSize: 14, weight: .semibold)
        label.numberOfLines = 0
        label.textAlignment = .center
        addSubview(label)
        label.translatesAutoresizingMaskIntoConstraints = false
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
