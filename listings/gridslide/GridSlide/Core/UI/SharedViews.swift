import UIKit

/// 圆角面板。内容块都装在它里面，和背景分层。
///
/// 刻意**不加 `final`**：成绩页的 `RecordCardView` 继承它，只是往里塞内容、
/// 不改外观，这样卡片样式只有一处定义。
class CardView: UIView {
    init(cornerRadius: CGFloat = AppTheme.cornerRadius) {
        super.init(frame: .zero)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = cornerRadius
        layer.cornerCurve = .continuous
    }

    required init?(coder: NSCoder) { nil }
}

/// 「步数 / 用时」这类小统计块：上面一行小标题，下面一行数值。
final class StatTileView: UIView {
    private let titleLabel = UILabel()
    private let valueLabel = UILabel()

    init(title: String, valueColor: UIColor = AppTheme.textPrimary) {
        super.init(frame: .zero)

        titleLabel.font = .systemFont(ofSize: 11, weight: .semibold)
        titleLabel.textColor = AppTheme.textSecondary
        // 小字全大写时字距太挤，撑开一点更像标签而不是正文。
        titleLabel.attributedText = NSAttributedString(
            string: title.uppercased(),
            attributes: [.kern: 0.8]
        )
        titleLabel.textAlignment = .center
        // 三个统计块在 320pt 宽屏上均分后，单格只有 76pt 左右，
        // "BEST MOVES" 这类标题会顶到边。允许它略微缩一点，别截断。
        titleLabel.adjustsFontSizeToFitWidth = true
        titleLabel.minimumScaleFactor = 0.8

        valueLabel.font = .monospacedDigitSystemFont(ofSize: 22, weight: .semibold)
        valueLabel.textColor = valueColor
        valueLabel.textAlignment = .center
        valueLabel.adjustsFontSizeToFitWidth = true
        valueLabel.minimumScaleFactor = 0.6

        let stack = UIStackView(arrangedSubviews: [titleLabel, valueLabel])
        stack.axis = .vertical
        stack.spacing = 3
        addForAutoLayout(stack)
        stack.pinToEdges(of: self)
    }

    required init?(coder: NSCoder) { nil }

    func setValue(_ text: String) {
        valueLabel.text = text
    }

    func setValueColor(_ color: UIColor) {
        valueLabel.textColor = color
    }
}

/// 设置页里的一行「标题 + 右侧值 + 可点」。
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
        // 值可能很长，压缩它而不是把标题挤没。
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

    /// 右侧换成一个开关（震动反馈那一行用）。放上开关后箭头就没意义了，一并去掉。
    func attachSwitch(_ toggle: UISwitch) {
        chevron.isHidden = true
        valueLabel.isHidden = true
        addForAutoLayout(toggle)
        NSLayoutConstraint.activate([
            toggle.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            toggle.centerYAnchor.constraint(equalTo: centerYAnchor)
        ])
    }

    override var isHighlighted: Bool {
        didSet {
            backgroundColor = isHighlighted ? AppTheme.separator : .clear
        }
    }
}

/// 分隔线。厚度取一物理像素，视网膜屏上才是"一根细线"而不是一条灰带。
final class HairlineView: UIView {
    init() {
        super.init(frame: .zero)
        backgroundColor = AppTheme.separator
        translatesAutoresizingMaskIntoConstraints = false
        heightAnchor.constraint(equalToConstant: 1 / UIScreen.main.scale).isActive = true
    }

    required init?(coder: NSCoder) { nil }
}
