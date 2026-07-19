import UIKit

final class HeroHeaderView: UIView {
    init(title: String, subtitle: String) {
        super.init(frame: .zero)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = 24

        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 30, weight: .bold)
        titleLabel.textColor = AppTheme.textPrimary

        let subtitleLabel = UILabel()
        subtitleLabel.text = subtitle
        subtitleLabel.font = .systemFont(ofSize: 15, weight: .medium)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.numberOfLines = 0

        let icon = UIImageView(image: UIImage(named: "dtp_brand_logo"))
        icon.contentMode = .scaleAspectFit

        let stack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
        stack.axis = .vertical
        stack.spacing = 8

        addSubview(icon)
        addSubview(stack)
        icon.translatesAutoresizingMaskIntoConstraints = false
        stack.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.topAnchor.constraint(equalTo: topAnchor, constant: 20),
            icon.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -20),
            icon.widthAnchor.constraint(equalToConstant: 56),
            icon.heightAnchor.constraint(equalToConstant: 56),
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 20),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 20),
            stack.trailingAnchor.constraint(equalTo: icon.leadingAnchor, constant: -12),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -20)
        ])
    }

    required init?(coder: NSCoder) { nil }
}

final class MetricTileView: UIView {
    private let valueLabel = UILabel()

    init(title: String, value: String, symbol: String) {
        super.init(frame: .zero)
        backgroundColor = AppTheme.surfaceElevated
        layer.cornerRadius = AppTheme.cornerRadius

        let icon = UIImageView(image: UIImage(systemName: symbol))
        icon.tintColor = AppTheme.accentSecondary

        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.textColor = AppTheme.textSecondary
        titleLabel.font = .systemFont(ofSize: 13, weight: .medium)

        valueLabel.text = value
        valueLabel.textColor = AppTheme.textPrimary
        valueLabel.font = .systemFont(ofSize: 24, weight: .bold)

        let textStack = UIStackView(arrangedSubviews: [titleLabel, valueLabel])
        textStack.axis = .vertical
        textStack.spacing = 4

        let row = UIStackView(arrangedSubviews: [icon, textStack])
        row.spacing = 12
        row.alignment = .center

        addSubview(row)
        row.translatesAutoresizingMaskIntoConstraints = false
        icon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 28),
            icon.heightAnchor.constraint(equalToConstant: 28),
            row.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            row.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            row.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            row.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func update(value: String) {
        valueLabel.text = value
    }
}

final class InfoCardView: UIView {
    private let titleLabel = UILabel()
    private let bodyLabel = UILabel()

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = AppTheme.cornerRadius
        titleLabel.font = .systemFont(ofSize: 17, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        bodyLabel.font = .systemFont(ofSize: 14, weight: .regular)
        bodyLabel.textColor = AppTheme.textSecondary
        bodyLabel.numberOfLines = 0
        let stack = UIStackView(arrangedSubviews: [titleLabel, bodyLabel])
        stack.axis = .vertical
        stack.spacing = 8
        addSubview(stack)
        stack.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) { nil }

    convenience init(title: String, body: String) {
        self.init(frame: .zero)
        configure(title: title, body: body)
    }

    func configure(title: String, body: String) {
        titleLabel.text = title
        bodyLabel.text = body
    }
}

final class GameModeCardView: UIView {
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()
    private let progressLabel = UILabel()
    private let iconView = UIImageView()

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = AppTheme.surfaceElevated
        layer.cornerRadius = 22
        titleLabel.font = .systemFont(ofSize: 22, weight: .bold)
        titleLabel.textColor = .white
        subtitleLabel.font = .systemFont(ofSize: 14, weight: .medium)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.numberOfLines = 0
        progressLabel.font = .systemFont(ofSize: 13, weight: .semibold)
        progressLabel.textColor = AppTheme.accent
        iconView.tintColor = AppTheme.accentSecondary
        iconView.contentMode = .scaleAspectFit
        let textStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel, progressLabel])
        textStack.axis = .vertical
        textStack.spacing = 6
        addSubview(iconView)
        addSubview(textStack)
        iconView.translatesAutoresizingMaskIntoConstraints = false
        textStack.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            iconView.topAnchor.constraint(equalTo: topAnchor, constant: 18),
            iconView.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -18),
            iconView.widthAnchor.constraint(equalToConstant: 34),
            iconView.heightAnchor.constraint(equalToConstant: 34),
            textStack.topAnchor.constraint(equalTo: topAnchor, constant: 18),
            textStack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 18),
            textStack.trailingAnchor.constraint(equalTo: iconView.leadingAnchor, constant: -12),
            textStack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -18)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(title: String, subtitle: String, progress: String, symbolName: String) {
        titleLabel.text = title
        subtitleLabel.text = subtitle
        progressLabel.text = progress
        iconView.image = UIImage(systemName: symbolName)
    }
}

final class PanelTitleView: UIView {
    init(text: String) {
        super.init(frame: .zero)
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 16, weight: .semibold)
        label.textColor = AppTheme.accentSecondary
        addSubview(label)
        label.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            label.topAnchor.constraint(equalTo: topAnchor),
            label.leadingAnchor.constraint(equalTo: leadingAnchor),
            label.trailingAnchor.constraint(equalTo: trailingAnchor),
            label.bottomAnchor.constraint(equalTo: bottomAnchor)
        ])
    }

    required init?(coder: NSCoder) { nil }
}

final class PrimaryButton: UIButton {
    init(title: String) {
        super.init(frame: .zero)
        setTitle(title, for: .normal)
        setTitleColor(.white, for: .normal)
        titleLabel?.font = .systemFont(ofSize: 17, weight: .bold)
        backgroundColor = AppTheme.accent
        layer.cornerRadius = 16
        heightAnchor.constraint(equalToConstant: 52).isActive = true
    }

    required init?(coder: NSCoder) { nil }
}

final class SecondaryButton: UIButton {
    init(title: String) {
        super.init(frame: .zero)
        setTitle(title, for: .normal)
        setTitleColor(AppTheme.accentSecondary, for: .normal)
        titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = 16
        layer.borderWidth = 1
        layer.borderColor = AppTheme.accentSecondary.withAlphaComponent(0.35).cgColor
        heightAnchor.constraint(equalToConstant: 48).isActive = true
    }

    required init?(coder: NSCoder) { nil }
}
