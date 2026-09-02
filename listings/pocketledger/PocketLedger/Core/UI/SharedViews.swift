import UIKit

/// 白色圆角卡片。整个 App 的内容块都装在它里面，视觉上和浅灰背景分层。
final class CardView: UIView {
    init(cornerRadius: CGFloat = AppTheme.cornerRadius) {
        super.init(frame: .zero)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = cornerRadius
        layer.cornerCurve = .continuous
    }

    required init?(coder: NSCoder) { nil }
}

/// 圆形图标徽章。账户按类型取图标与颜色，流水按分类取 —— 列表里一眼分辨。
final class IconBadgeView: UIView {
    private let imageView = UIImageView()

    init(diameter: CGFloat = 40) {
        super.init(frame: .zero)
        layer.cornerRadius = diameter / 2
        imageView.contentMode = .scaleAspectFit
        imageView.tintColor = .white
        addForAutoLayout(imageView)
        NSLayoutConstraint.activate([
            widthAnchor.constraint(equalToConstant: diameter),
            heightAnchor.constraint(equalToConstant: diameter),
            imageView.centerXAnchor.constraint(equalTo: centerXAnchor),
            imageView.centerYAnchor.constraint(equalTo: centerYAnchor),
            imageView.widthAnchor.constraint(equalToConstant: diameter * 0.5),
            imageView.heightAnchor.constraint(equalToConstant: diameter * 0.5)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(symbolName: String, color: UIColor) {
        imageView.image = UIImage(systemName: symbolName)
        backgroundColor = color
    }
}

/// 「本月支出 / 本月收入」这类统计小块：上面一行小标题，下面一行金额。
final class StatTileView: UIView {
    private let titleLabel = UILabel()
    private let valueLabel = UILabel()

    init(title: String, valueColor: UIColor) {
        super.init(frame: .zero)
        titleLabel.font = .systemFont(ofSize: 11, weight: .semibold)
        titleLabel.textColor = AppTheme.textSecondary
        // 小字全大写时字距太挤，撑开一点更像标签而不是正文。
        titleLabel.attributedText = NSAttributedString(
            string: title.uppercased(),
            attributes: [.kern: 0.8]
        )

        valueLabel.font = .systemFont(ofSize: 20, weight: .semibold)
        valueLabel.textColor = valueColor
        valueLabel.adjustsFontSizeToFitWidth = true
        valueLabel.minimumScaleFactor = 0.6

        let stack = UIStackView(arrangedSubviews: [titleLabel, valueLabel])
        stack.axis = .vertical
        stack.spacing = 4
        addForAutoLayout(stack)
        stack.pinToEdges(of: self)
    }

    required init?(coder: NSCoder) { nil }

    func setValue(_ text: String) {
        valueLabel.text = text
    }
}

/// 空状态。列表为空时给一句话说明加一个行动按钮，而不是丢给用户一片白。
final class EmptyStateView: UIView {
    private let button = UIButton(type: .system)
    private var action: (() -> Void)?

    init(symbolName: String, title: String, message: String, actionTitle: String? = nil) {
        super.init(frame: .zero)

        let icon = UIImageView(image: UIImage(systemName: symbolName))
        icon.tintColor = AppTheme.textSecondary.withAlphaComponent(0.5)
        icon.contentMode = .scaleAspectFit

        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 18, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.textAlignment = .center

        let messageLabel = UILabel()
        messageLabel.text = message
        messageLabel.font = .systemFont(ofSize: 15)
        messageLabel.textColor = AppTheme.textSecondary
        messageLabel.textAlignment = .center
        messageLabel.numberOfLines = 0

        let stack = UIStackView(arrangedSubviews: [icon, titleLabel, messageLabel])
        stack.axis = .vertical
        stack.alignment = .center
        stack.spacing = 12
        stack.setCustomSpacing(18, after: icon)

        if let actionTitle {
            // 用 iOS 15 的 UIButton.Configuration 而不是 contentEdgeInsets ——
            // 后者从 iOS 15 起已废弃，本包最低就是 15.6，直接用新 API 更干净。
            var config = UIButton.Configuration.filled()
            config.title = actionTitle
            config.baseBackgroundColor = AppTheme.accent
            config.baseForegroundColor = .white
            config.cornerStyle = .capsule
            config.contentInsets = NSDirectionalEdgeInsets(
                top: 12, leading: 26, bottom: 12, trailing: 26
            )
            var titleAttributes = AttributeContainer()
            // 写全 `UIFont.` 而不是靠前导点做隐式成员查找：这里要先经 AttributeContainer 的
            // dynamicMemberLookup 推出 Value == UIFont，再在其上查找 —— 双重推导，
            // 而这是本包唯一一处没有「同款写法已在本仓库验证过能编译」背书的 API，
            // 不值得省这几个字符。（gridslide 的同一处也是这么写的。）
            titleAttributes.font = UIFont.systemFont(ofSize: 16, weight: .semibold)
            config.attributedTitle = AttributedString(actionTitle, attributes: titleAttributes)
            button.configuration = config
            button.addTarget(self, action: #selector(handleTap), for: .touchUpInside)
            stack.addArrangedSubview(button)
            stack.setCustomSpacing(22, after: messageLabel)
        }

        addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 52),
            icon.heightAnchor.constraint(equalToConstant: 52),
            stack.centerXAnchor.constraint(equalTo: centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: leadingAnchor, constant: 32),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor, constant: -32)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func onAction(_ handler: @escaping () -> Void) {
        action = handler
    }

    @objc private func handleTap() {
        action?()
    }
}

/// 表单里的一行「标题 + 右侧值 + 可点」。账户选择、分类选择、日期这类都用它。
final class FormRowView: UIControl {
    private let titleLabel = UILabel()
    private let valueLabel = UILabel()
    private let chevron = UIImageView(image: UIImage(systemName: "chevron.right"))

    init(title: String, showsChevron: Bool = true) {
        super.init(frame: .zero)
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 16)
        titleLabel.textColor = AppTheme.textPrimary

        valueLabel.font = .systemFont(ofSize: 16)
        valueLabel.textColor = AppTheme.textSecondary
        valueLabel.textAlignment = .right
        // 值可能很长（长账户名），压缩它而不是把标题挤没。
        valueLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        valueLabel.lineBreakMode = .byTruncatingTail

        chevron.tintColor = AppTheme.textSecondary.withAlphaComponent(0.6)
        chevron.contentMode = .scaleAspectFit
        chevron.isHidden = !showsChevron

        let stack = UIStackView(arrangedSubviews: [titleLabel, valueLabel, chevron])
        stack.axis = .horizontal
        stack.alignment = .center
        stack.spacing = 10
        stack.isUserInteractionEnabled = false
        addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            heightAnchor.constraint(equalToConstant: 52),
            chevron.widthAnchor.constraint(equalToConstant: 12),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func setValue(_ text: String?) {
        valueLabel.text = text
    }

    override var isHighlighted: Bool {
        didSet {
            backgroundColor = isHighlighted
                ? AppTheme.separator.withAlphaComponent(0.5)
                : .clear
        }
    }
}

/// 分隔线。厚度按屏幕缩放取一物理像素，视网膜屏上才是"一根细线"而不是一条灰带。
final class HairlineView: UIView {
    init() {
        super.init(frame: .zero)
        backgroundColor = AppTheme.separator
        translatesAutoresizingMaskIntoConstraints = false
        heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
    }

    required init?(coder: NSCoder) { nil }
}
